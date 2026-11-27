# Gate 2 验证报告 — git-gpg-keygen

检查日期：2026-11-27 | 范围：OpenPGP/SSH 密钥生成六操作（服务/CLI/桥接/档案内嵌面板） | Gate1：gate1-git-gpg-keygen-2026-11-27.md（WARN/W1）

**Profile**：dev-toolbox（stacks: go-core, go-webview-vue3-antd）  
**确认策略**：**W1** — WARN 项仅建议、不阻塞交付继续  
**扫描路径**：`internal/gitidentity/keygen*.go`、`cmd/dev-toolbox/git.go`、`internal/desktop/desktop.go`、`frontend/src/{KeygenPanel,ProfileEditor,types,dev-mock}.*`、`internal/desktop/ui/index.html`

## 检查结果

| 检查项 | 结果 | 依据 |
|---|---|---|
| GR-01 激活链规则扫描 | PASS（附 WARN） | 见下方 GC / WV / product-rules；无 CRITICAL 违规 |
| GR-02 编译通过 | PASS | `go build ./...` 通过；`GOOS=linux CGO_ENABLED=0` CLI 交叉编译通过 |
| GR-03 新接口完整链路测试 | PASS | 9 项 Keygen 测试：options/preview 校验、confirm、SSH create→select→export、OpenPGP 真机 gpg、自定义路径、缺工具、enableGpgSign 默认、部分成功 |
| GR-04 测试编译通过 | PASS | `go test ./internal/gitidentity/ -count=1` ok（~35s） |
| GR-05 前后端契约对齐 | PASS（附 WARN） | 六 op 路径/字段/枚举/错误码对齐；部分成功 Data 在桥接前端丢弃见 WARN-2 |
| GR-06 对齐记录 | PASS | 本报告 |
| WV-08 前端 typecheck | PASS | `npm --prefix frontend run typecheck`（vue-tsc）通过 |
| WV-08 GUI 宿主编译 | **WARN** | `-tags gui` 因既有 `clientsettings` API 漂移失败（见 WARN-1）；与本模块 keygen 改动无关 |
| 包覆盖率（gitidentity） | **WARN** | `go test -cover` → **69.2%** statements（低于模板 Service 80%）；Keygen 专用用例已覆盖主路径 |

## 六操作契约对齐（AL / GR-05）

| op | API | Go 服务 | CLI `git keygen` | 桌面白名单 | 前端 invoke / types / mock |
|---|---|---|---|---|---|
| `git.keygenOptions` | ✓ | `KeygenOptionsProbe` | `options` | ✓ | ✓ |
| `git.keygenPreview` | ✓ | `KeygenPreview` | `preview` | ✓ | ✓ |
| `git.keygenCreate` | ✓ | `KeygenCreate` + confirm | `create --confirm` / `--passphrase-stdin` | ✓（含部分成功 fault） | ✓ |
| `git.keyList` | ✓ | `KeyList` | `list --type` | ✓ | ✓ |
| `git.keySelect` | ✓ | `KeySelect` + confirm | `select --confirm` | ✓ | ✓ |
| `git.keyExport` | ✓ | `KeyExport` | `export [--out --confirm --overwrite]` | ✓ | ✓ |

字段抽查：`profileId/keyType/algorithm/expiry/passphrase/enableGpgSign/comment/outputPath/confirm`、`KeygenPlan.commandSummary/profilePatch/wouldFail`、`KeygenResult.localKeyCreated/profileUpdated`、`LocalKeySummary.id/fingerprint/uidOrComment`、`PublicKeyExport.text/writtenPath` — Go JSON tag 与 `frontend/src/types.ts` 一致。

枚举 code：`openpgp|ssh`；`ed25519|rsa4096|rsa3072|nistp256|ecdsa256`；`never|1y|2y|5y` — 与 common-enums / API 合法集一致（服务端强制）。

错误码（前端 `friendlyKeygenError` 可识别）：`GIT_PROFILE_*`、`GIT_KEYGEN_*`、`GIT_KEY_*`、`CONFIRMATION_REQUIRED` — 与 API §3.4 对齐。

## GR-01 规则扫描明细

### product-rules

| # | 结果 | 说明 |
|---|---|---|
| 3 凭据不落盘/不回显 | PASS | create 后 `clearPassphrase`；预览口令占位 `<passphrase>`/「（已设置，不展示）」；CLI `--passphrase-stdin`；GPG 经 stdin fd |
| 4 单一 Go 服务 | PASS | CLI/GUI 均调 `gitidentity.Keygen*` |
| 9 测试隔离 | PASS | 沙箱 HOME/USERPROFILE/APPDATA/LOCALAPPDATA/XDG/GNUPGHOME/GIT_CONFIG_*；可注入 GPGHome/SSHDir/Bin |

### go-core（GC）

| # | 结果 | 说明 |
|---|---|---|
| GC-01 | PASS | 领域在 `gitidentity`，无 WebView 依赖 |
| GC-03 | PASS | `runTool` 参数数组 + `os.StartProcess`，无 shell 拼接 |
| GC-04/05 | PASS | 可注入 `KeygenTools` / `toolRunner`；沙箱测试 |
| GC-06 | PASS | context 120s 超时可取消；稳定错误 code |
| GC-08 | PASS / WARN | `gofmt` 干净；`go build`/`go test` 通过；GUI tag 见 WARN-1 |

### go-webview-vue3-antd（WV）

| # | 结果 | 说明 |
|---|---|---|
| WV-01/04 | PASS | Vue3+AntD；前端不拼 gpg/ssh 命令 |
| WV-03 | PASS | 仅注册六白名单 op |
| WV-06 | PASS | loading/防重复、口令清空、二次确认 |
| WV-09 | PASS（附 WARN） | 字段对齐；部分成功 Data 消费缺口见 WARN-2 |
| ER-04 文案 | PASS | 「已更新档案的签名字段」；无 emoji/感叹号堆砌（吸收 Gate1 W1-1） |

## WARN 建议项（W1，不阻塞交付）

1. **GUI 宿主 `-tags gui` 当前不可编**  
   `go vet/build -tags gui ./internal/desktop` 报错：`clientsettings.AddSourceProject` / `ProjectBinding` / `RemoveSourceProject` 未定义，且 `ExportSnapshot` 签名为单参数、调用方传两参数。属桌面壳与 clientsettings 既有漂移，**非本模块 keygen 引入**。交付本模块 CLI/服务不受阻；桌面联调需先修复 settings API 再 `-tags gui` 编译点验。

2. **部分成功结果在桥接前端丢失**  
   宿主对 `KeygenProfileWriteError` 返回 `ok:false` + `data: KeygenResult`；`bridge.ts` 的 `invoke` / `__devToolboxResolve` 在失败时只抛 `Error(message)`，丢弃 `data`。因此 PageSpec「警告条 + 指纹/路径 + 写入档案」在真实宿主路径上无法填充 `result`（CLI 仍会 `output(partial.Result)`）。建议后续：对该错误码携带 data 解析，或改为 `ok:true` + `profileUpdated:false` + warning。

3. **覆盖率**  
   `internal/gitidentity` 整包 69.2% &lt; 模板 80%；本模块九项专用测试已覆盖主成功/失败/部分成功路径，接受为建议项。

4. **桌面面板人工点验未执行**  
   受 WARN-1 阻塞；内嵌包 `internal/desktop/ui/index.html` 已含 Keygen 文案（先前 `build:desktop`）。GUI 可编后建议在档案编辑「提交签名」点验生成/选用/导出。

## Gate1 W1 吸收情况

| Gate1 建议 | 实现态 |
|---|---|
| 文案改为「已更新档案的签名字段」 | 已落地（KeygenPanel success） |
| 命名以 API camelCase 为准 | 已对齐 |
| SSH comment 无编辑控件 | 仍可接受（默认 `devtoolbox:{profileId}`） |
| 桥接命名映射表 | 六 op 已挂白名单与 CLI |

## 总结

| 项 | 值 |
|----|-----|
| **判定** | **WARN**（确认策略 **W1**） |
| CRITICAL | 0 |
| WARN | 4（GUI 编译漂移、部分成功 Data 丢弃、覆盖率、人工 GUI 点验） |
| **可交付** | **是**（W1：建议项列出后允许继续；CLI/服务/前端 typecheck 与六 op 契约主路径通过） |

**结论**：代码→交付门控可通过（带建议）。优先修复桌面 `clientsettings` 漂移以恢复 GUI 编译，并补齐部分成功桥接 Data 消费后再做档案面板真机点验。
