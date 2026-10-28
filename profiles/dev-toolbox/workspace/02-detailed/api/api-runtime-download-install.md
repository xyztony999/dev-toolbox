# Go 应用服务契约 — 运行时下载与安装

版本：v1.0 | 日期：2026-10-28 | 模块：runtime-download-install | 状态：Gate 1 待检查

> CLI/WebView 调用同一 Go 应用服务；不存在 HTTP server。操作含 requestId，事件另含 taskId 和 sequence。

## DTO

- `InstallOption`: tool、providerId、mirror options、versionSelector(`latest|major|supported-prefix`)、supportedPlatforms、supportedArch。
- `InstallPlan`: planId、planDigest、tool、providerId、mirror、requestedSelector、resolvedVersion、os、arch、packages[]、targetHome、estimatedBytes?、checksumAlgorithm/value、expiresAt。
- `InstallTask`: taskId、planId、state、progress{phase,receivedBytes,totalBytes?}、packageResults[]、installationId?、error{code,message}。
- 计划不含用户凭据；URL 只由 provider adapter 内部持有，不回传到桥接。

## 操作

| op | 请求 | 响应 | 副作用 |
|---|---|---|---|
| `runtime.install.options` | `tool` | provider/版本粒度/镜像/当前平台支持 | 只读静态能力；OS/架构从服务主机环境确定 |
| `runtime.install.preview` | `tool`, `selector`, `mirror`, `expectedRevision` | InstallPlan | 允许网络查询 metadata；不得下载 archive 或写安装目录 |
| `runtime.install.start` | `planId`, `planDigest`, `expectedRevision`, `confirm=true` | `taskId`, 初始状态 | 启动后台下载/校验/解包；目标仅产品目录 |
| `runtime.install.cancel` | `taskId` | 当前最终/运行状态 | 协作取消；归位提交开始后返回 busy/completed |
| `runtime.install.status` | `taskId` | InstallTask | 只读快照 |
| `runtime.install.event` | host push | taskId、sequence、InstallTask | 仅进度/状态，不含 URL、凭据或 archive 内容 |

CLI 对应：`runtime install options <tool>`、`runtime install preview <tool> <selector> [--mirror id]`、`runtime install <plan-id> <digest> --confirm`、`runtime install status|cancel <task-id>`；安装命令不能接任意 URL/路径。

## 错误码与并发

`RUNTIME_PROVIDER_UNSUPPORTED`, `RUNTIME_VERSION_UNAVAILABLE`, `RUNTIME_PLAN_EXPIRED`, `RUNTIME_PLAN_STALE`, `RUNTIME_REVISION_CONFLICT`, `RUNTIME_TARGET_CONFLICT`, `RUNTIME_CHECKSUM_MISSING`, `RUNTIME_CHECKSUM_MISMATCH`, `RUNTIME_ARCHIVE_UNSAFE`, `RUNTIME_DOWNLOAD_FAILED`, `RUNTIME_INSTALL_CANCELLED`, `RUNTIME_CONFIRMATION_REQUIRED`, `RUNTIME_DISK_SPACE_INSUFFICIENT`。

planDigest 对所有解析结果和目标字段做摘要。start 在任务注册锁内再次检查 planDigest、expiresAt 与 user state revision，重复提交同 plan 返回现有 task。目标路径已存在时只允许本产品 manifest 证明同一内容可复用；不得删除替换。register 成功后写 user state revision 并发出 terminal event；失败不注册运行时。

## 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-10-28 | initial | provider 查询、计划、后台任务和取消契约 |
