# Go 应用服务契约 — 会话启动器

版本：v1.0 | 日期：2026-11-03 | 模块：session-launcher | 状态：Gate 1 待检查

> CLI/WebView 调用同一 Go 应用服务；open 为显式用户动作。

## DTO

- `SessionToolState`: tool、status(`injected|skipped`)、version?、binDir?、reason?（skipped 原因：未选择/路径不可用）。
- `SessionPreview`: cwd、tools[]、javaHome?、pathPrepends[]（有序）、gitIdentity{resolved,profileId?}、terminals[]（TerminalInfo{id,label,available}）。
- `SessionOpenResult`: terminalId、pid?、startedAt、error?。

## 操作

| op | 请求 | 响应 | 副作用 |
|---|---|---|---|
| `session.preview` | path? | SessionPreview | 只读 |
| `session.open` | path、terminalId、confirm | SessionOpenResult | 创建终端子进程（新控制台窗口）；无持久写 |

CLI 对应：`dev-toolbox session preview [path]`、`session open [path] [--terminal id]`（open 默认预览结果中第一个可用终端）。

## 枚举与错误码

TerminalId：`powershell`、`cmd`、`bash`（Windows 上为 Git Bash；Linux 为 $SHELL 回退 bash）。

`SESSION_TERMINAL_NOT_FOUND`（指定 id 不可用）、`SESSION_TERMINAL_LAUNCH_FAILED`、`SESSION_CWD_INVALID`、`SESSION_STATE_UNRESOLVED`（非错误：对应 tools[].status=skipped）。

## 环境组装规范

- PATH：`pathPrepends`（bin 目录，按 tools 顺序去重）+ 父 PATH。
- JAVA_HOME：选中 java 安装根目录；未选择则不设置（继承）。
- 其余变量不注入（GOROOT/PYTHONHOME 等后续按需增补，届时同步更新本契约）。

## 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-11-03 | initial | 上下文求值与终端启动契约 |
