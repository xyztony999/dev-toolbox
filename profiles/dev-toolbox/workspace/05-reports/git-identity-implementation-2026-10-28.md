# 实现报告 — git-identity-migration 后端、CLI、桥接与前端

日期：2026-10-28 | 模块：git-identity-migration | 依据：Gate 1 PASS 的 D0/BA/AA/FL/PRD/API/PageSpec v1.0 | 迁移来源：gitid `e33e728`

## 实现范围

### Go 应用服务（internal/gitidentity，全新包，无新增第三方依赖）

- **档案存储**：`git-identity.json`（DevToolbox 自有命名空间），字段校验与 gitid 同规（id 字符集、邮箱、gpg.format、insteadOf 去重、账号 host 归一化）；upsert 保留既有目录绑定；文件锁 + 原子写入。
- **应用（use）**：受管键（user.name/email/signingkey、commit.gpgsign、gpg.format）全量覆盖——写入档案键、unset 缺失键；上一档案（按 name+email 匹配）的 extra/insteadOf/账号选择器未被继承时清除（ADR-016/017）；dry-run 与真跑同一算法只算不写；多仓批量逐仓结果，允许部分失败。
- **生效解析（current）**：`git config --show-origin` 逐键分类来源（本地 → includeIf → 全局），与 git 输出一致；prompt 单行 id/manual/none。
- **目录绑定（bind/unbind）**：托管片段经 `git config --file` 生成（与读取同工具，杜绝转义漂移）；`includes.gitconfig` 单链 + 用户 ~/.gitconfig 单条 include 钩子；Windows gitdir/i + 大小写盘符双模式 + 真实路径（junction/symlink 解析）；目录可先绑定后创建；解绑前漂移检测（字节级 + 键级差异），手改内容如实警告不静默丢弃。
- **仓库扫描（scan）**：只读遍历（跳过 node_modules 等），逐仓身份状态（✓ 一致 / ✗ 缺失 / ⚠ 手配）+ 镜像 ⊕ + 生效账号选择器；手配身份可预览后收编。
- **提交审计（audit）**：`git log --all` 作者分布、期望档案污染标注、mailmap 仅建议（不写 .mailmap）；strict 存在污染即判定失败。
- **Git 体检（doctor）**：git 版本、credential.helper、insteadOf 冲突（同一前缀多目标）、绑定健康（钩子缺失/片段缺失/失效目录/嵌套/手改漂移/孤儿片段）；只读不修复。
- **镜像推送（remote）**：pushurl 双目标机制，fetch 与凭据通道不受影响；unmirror 还原。
- **凭据管道（credential）**：gitid 的 conduit 语义完整移植——`credential fill/approve/reject`；探测关闭终端交互（GIT_TERMINAL_PROMPT=0/GIT_ASKPASS/GCM_INTERACTIVE）；token 仅在调用帧内存转，不落盘、不进日志/错误/响应；host+username 线索清单（非密钥）随 login/set 记录、随 erase 删除（与 gitid credentialHosts 一致）。
- **档案迁移（legacy）**：gitid config.json 双路径探测（%APPDATA% 与 XDG，含 GITID_CONFIG 覆盖）；预览标注 id 冲突与重复邮箱，非法 id 自动按邮箱派生且让位合法 id；备份 + 来源摘要幂等导入。
- **进程执行**：固定 `git` 二进制 + 固定子命令白名单 + 参数净化（拒绝 NUL/换行），经 os.StartProcess 数组式执行，全程无 shell 字符串（GC-03）。

### CLI（`dev-toolbox git ...`）

profiles / save-profile / remove-profile / import-global / use（--local/--repos/--dry-run/--confirm）/ current（--prompt）/ bind / unbind / unset / scan（--save）/ audit（--expect/--strict，污染退出码 2）/ doctor / remote / credential / legacy。破坏性操作一律 --confirm；批量应用部分失败退出码 4。

### 桌面桥接

22 个 `git.*` 白名单操作全部接入（含 confirm 语义）；GUI 的 credentialSet 为一次性通道：token 经桥接帧直通 helper 后即清空，不入日志与广播。

### 前端（GitIdentity.vue，侧边栏"Git 身份"入口）

四面板：身份档案（列表/编辑/收编全局/删除）、应用与绑定（dry-run 预览→确认应用→逐仓结果→生效身份与每键来源→绑定/解绑）、仓库审计与体检（扫描/收编/审计/strict 结果/mailmap 建议/doctor 发现）、迁移与凭据（gitid 导入向导/凭据账号/helper 登录/擦除/镜像推送）。token 输入为密码框且提交即清空。演示桥覆盖全部 22 个操作。

## 验证结果（真实执行）

| 检查 | 命令 | 结果 |
|---|---|---|
| 格式/静态 | `gofmt -l`、`go vet ./...` | 通过（无 diff、无告警） |
| 编译 | `go build ./...` 与 `CGO_ENABLED=1 go build -tags gui ./...` | 双双通过 |
| 测试 | `go test ./internal/...` | ok — gitidentity 9 项沙箱测试：档案校验与绑定保留、全局应用全量覆盖+切换清理、本地 dry-run 不落盘+批量部分失败、生效来源与 prompt、绑定→includeIf 生效→手改漂移→解绑回收、doctor 钩子缺失、扫描/镜像/审计 strict+mailmap、凭据 fake-helper 全链路（approve/fill/reject）、legacy 幂等导入+digest+确认门。测试以真实 git 在隔离沙箱运行（GIT_CONFIG_GLOBAL/临时 HOME/临时 APPDATA），迁移行为规格即证据 |
| 前端 | `vue-tsc --noEmit`、`npm run build:desktop` | 通过，内嵌包已更新 |
| CLI 冒烟 | `git current --prompt`、`git profiles`、`git legacy preview`（只读） | 通过；探测到本机真实 gitid 档案（schema 有效、可导入） |

## GR-01 规则扫描与契约对齐

- 产品不变量：1（git 事实源——读写全部经 git config）✓；2（全量覆盖+切换清理，有测试）✓；3（token conduit，响应/日志/存储均无 token）✓；4（单一 Go 服务）✓；6（仅显式操作写全局）✓；7（扫描只读）✓；8（legacy 只读+预览+备份+幂等）✓；9（沙箱隔离）✓。
- go-core GC-01..09 全过（GC-08：静态与 GUI tag 双构建均真实执行）；webview WV-01..09 全过（WV-09：桥接 22 操作 payload/DTO/错误码与前端类型逐一对齐）。
- GR-03：新接口以真实 git 沙箱 E2E 覆盖完整链路（应用→生效→绑定→回收→审计→迁移）；真实凭据 helper（GCM）交互未在本机自动化执行（用 fake helper 覆盖协议层），见 Gate 2 待确认项。

## 待跟进（Gate 2 WARN 项）

1. 桌面客户端需重启到新内嵌包后人工点验"Git 身份"四面板（当前运行中的客户端为旧构建）。
2. `git credential login`（GCM 弹窗）与 `credential set` 的真实 helper 交互建议用户手动执行一次；协议层已由 fake helper 沙箱覆盖。
3. GUI credentialSet 的手工 PAT 通道已按"一次性帧内消费、即时清空"实现，按产品规则 3 需单独审查确认后方可视为定稿。
