# 实现报告 — runtime-download-install 后端与前端

日期：2026-10-28 | 模块：runtime-download-install | 依据：Gate 1 PASS 的 D0/BA/AA/FL/PRD/API/PageSpec v1.0

## 实现范围

### Go 应用服务（internal/runtimeinstall，重写此前的未完成 node.go）

- 任务管理器 `Manager`：`Options / Preview / Start / Cancel / Status / Events / SetSink`。Start 在注册锁内复核 planDigest、expiresAt 与 user state revision；同 plan 重复提交返回现有任务；完成时经 `userstate.RegisterDownloadedInstallation` 幂等登记（source=downloaded），失败不登记。
- 计划 DTO 按契约：InstallOption / InstallPlan（planId+planDigest）/ InstallTask（state、progress、packageResults、installationId、error）。provider 下载 URL 保存在计划非导出字段，不跨 CLI/WebView 桥接返回。
- 事件：进程内单调 sequence 事件日志 + 宿主推送 sink（WebView 经 Dispatch/Eval 转发 `window.__devToolboxEvent`）；`runtime.install.events` 支持按 sequence 增量补齐。
- 七个 provider（行为迁移自 DevSwitch `internal/downloader`，commit `5e3fc4a`）：
  - node（官方/npmmirror/tuna，latest/major，SHASUMS256 必需）
  - java（Adoptium Temurin，latest/major，镜像 tuna；manifest sha256 必需）
  - maven / gradle（latest/major，dlcdn/官方 + tuna/腾讯镜像，sidecar .sha256/.sha512，内容非十六进制不采信）
  - androidsdk（platform-tools + cmdline-tools 组合安装；包名与校验取自 repository2-3.xml 清单；第一阶段完成、第二阶段失败 → partial 状态，重试复用已完成包）
  - golang（golang.google.cn 清单内建 sha256；x64）
  - python（python-build-standalone via npmmirror，install_only；.sha256 sidecar 必需；x64）
- 与 DevSwitch 的行为差异（按 D0 规则刻意收窄）：无校验值不安装（原版可跳过校验）；不删除/覆盖既有目标（原版 RemoveAll 后归位）；支持协作取消与 staging 清理；安装后不自动切换默认；staging 固定在产品数据目录内（同盘原子归位）。
- 安全边界：HTTPS host allowlist + DNS 解析后拒绝非公网地址 + 重定向校验；解包拒绝路径穿越/绝对路径/链接成员/超限文件数与体积；sha256/sha512 校验后才能解包；目标冲突立即失败；start 前按估算体积检查磁盘空间。

### CLI（cmd/dev-toolbox）

`runtime install options [tool]`、`preview <tool> <selector> [--mirror id]`、`start <plan-id> <digest> --confirm`、`status|cancel <task-id>`，以及组合命令 `runtime install <tool> <selector> [--mirror id] --confirm`（CLI 进程内预览→启动→跟随进度，Ctrl+C 触发协作取消）。计划在内存中按会话有效，不接任意 URL/路径。

### 桌面桥接（internal/desktop）

新增白名单操作：`runtime.install.options/preview/start/status/cancel/events`；宿主 sink 将安装事件推送到页面。preview 为同步操作，元数据查询期间 WebView UI 可能短暂阻塞，已列入待跟进。

### 前端（frontend/src）

- `RuntimeInstall.vue`：“获取运行时”面板 — 工具/镜像/版本通道选择（不可用平台禁用并显示原因）→ 预览确认卡（解析版本、provider、OS/架构、分包、校验算法、目标目录、体积、目标冲突）→ 确认弹窗（明确“不切换默认、不改系统环境”）→ 任务进度（阶段、下载进度、Android 分包结果、错误 code + 可操作建议、取消、重新预览重试）→ 完成后刷新用户清单。
- types.ts 安装 DTO、bridge.ts 增加宿主事件回调声明、dev-mock.ts 演示桥支持六个 install 操作；RuntimeSettings.vue 挂载面板。

## 验证结果（真实执行）

| 检查 | 命令 | 结果 |
|---|---|---|
| 格式 | `gofmt -w internal cmd` | 通过（无 diff 残留） |
| 编译 | `go build ./...` | 通过（go.mod 锁定 `github.com/ulikunitz/xz v0.5.12`，取自本机模块缓存；proxy.golang.org 不可达） |
| 静态检查 | `go vet ./...` | 通过 |
| 测试 | `go test ./internal/...` | ok（runtimeinstall 9 项：全流程登记/重复提交/确认与 revision 冲突/计划过期与 stale/取消不落盘/部分失败不落盘/zip 穿越与 symlink 拒绝/tar.gz 单根/node 版本与校验解析/Android 清单校验门/digest 稳定性；userstate/catalog 既有测试通过） |
| 前端类型 | `vue-tsc --noEmit` | 通过 |
| 前端构建 | `npm run build:desktop` | 通过，内嵌包已更新 internal/desktop/ui/index.html |
| GUI 编译 | `go build -tags gui ./...` | **未通过本机验证**：CGO_ENABLED=0 且无 gcc，webview_go 全部文件被构建约束排除（desktop-shell 既有待确认项，非本次引入） |
| 真实网络冒烟 | provider 实际下载 | **未执行**（本回合无网络访问真实 provider；沙箱用 httptest stub 覆盖） |

测试隔离：t.Setenv 隔离 APPDATA/LOCALAPPDATA/XDG_CONFIG_HOME/XDG_DATA_HOME，网络经 fetcher 注入指向本地 httptest，未触碰真实用户状态、PATH 或注册表。

## GR-01 规则扫描（引擎→栈→业务）

- 引擎 GR-01~06：GR-02 编译通过；GR-03 新接口以管理器全流程沙箱测试覆盖完整链路（preview→start→进度→登记→重放冲突），真实 provider E2E 未运行已如实记录；GR-04 测试编译随 `go test` 通过；GR-05/06 见下节对齐表，已记录。
- go-core GC-01~09：服务与 GUI/CLI 共用（GC-01）；测试不触真实环境（GC-05）；下载/安装可取消、错误带稳定分类（GC-06）；接口改动附完整失败路径回归（GC-07）；gofmt/vet/test/build 均执行，GUI tag 编译不可用已如实报告、未冒充（GC-08）；依赖版本锁定于 go.mod/go.sum（GC-09）。
- webview WV-01~09：宿主/组件库未替换（WV-01）；桥接仅白名单操作、稳定关联标识（WV-03）；前端无第二真值（WV-04）；后台 UI 更新经宿主 Dispatch 推送 + 轮询（WV-05）；长操作有进度/取消/重复点击保护/失败详情/重试，事件与错误不含 URL 与凭据（WV-06）；空态/冲突/部分失败/离线错误可见（WV-07）；类型检查与生产构建执行并同步内嵌资源（WV-08）；WV-09 对齐见下。
- 产品不变量 1-10：本模块只涉 4/6/7/9 — 单一 Go 服务来源 ✓；安装不触碰全局默认、不宣称改写父终端 ✓；不执行任何项目命令、provider 之外无任意 URL ✓；测试环境隔离 ✓。

## GR-05/06 前后端契约对齐（逐项）

| 桥接操作 | 前端 payload | Go 解析 | 一致 |
|---|---|---|---|
| runtime.install.options | tool | `{Tool string}` | ✓ |
| runtime.install.preview | tool/selector/mirror/expectedRevision | 同名四字段 | ✓ |
| runtime.install.start | planId/planDigest/expectedRevision/confirm | 同名四字段 | ✓ |
| runtime.install.status / cancel | taskId | `{TaskID string}`（json taskId） | ✓ |
| runtime.install.events | sinceSequence | `{SinceSequence uint64}` | ✓ |

字段对齐：InstallOption/InstallPlan/InstallTask/Event 的 JSON 标签与 types.ts 逐字段一致；state 枚举（confirmed/downloading/verifying/extracting/registering/complete/failed/cancelled/partial）与包结果（pending/done/reused/failed）一致；错误码集合与 API 契约一致，实现另增 `RUNTIME_TASK_NOT_FOUND`、`RUNTIME_INSTALL_FAILED`（仅服务内部终止态），前端建议表覆盖契约码。

## 待跟进（不阻塞 WARN 确认，但需用户知悉）

1. GUI tag 编译需本机启用 CGO（安装 MinGW-w64 并 CGO_ENABLED=1）后验证；当前不能声称桌面端可运行。
2. 真实 provider 下载冒烟（node 官方源或镜像）需网络环境，建议用户手动执行 `dev-toolbox runtime install node latest --confirm` 验证一次。
3. preview 为同步桥接调用，元数据慢时 WebView 界面短暂无响应；后续可改造为异步请求模式。
4. CLI 计划仅存于进程内：`start <plan-id> <digest>` 需与 preview 同会话；跨进程持久化计划待后续设计。
5. arm64 的 Java/Node 之外的 provider 证据仍缺，均按 unsupported 返回，未放宽。
