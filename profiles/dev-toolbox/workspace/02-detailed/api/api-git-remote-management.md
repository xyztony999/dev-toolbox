# Go 应用服务契约 — 仓库 Git Remote 管理

版本：v1.0 | 日期：2026-11-03 | 模块：git-remote-management | 状态：Gate 1 待检查

> CLI/WebView 调用同一 Go 应用服务；写操作含 confirm；全部经参数数组执行。

## DTO

- `RemoteInfo`（沿用）：name、url(fetch)、pushUrls[]、mirror、account?。

## 操作

| op | 请求 | 响应 | 副作用 |
|---|---|---|---|
| `git.remoteList` | repo | RemoteInfo[] | 只读 |
| `git.remoteAdd` | repo、name、url、confirm | RemoteInfo[] | `git remote add` |
| `git.remoteRemove` | repo、name、confirm | RemoteInfo[] | `git remote remove` |
| `git.remoteRename` | repo、from、to、confirm | RemoteInfo[] | `git remote rename` |
| `git.remoteSetUrl` | repo、name、url、confirm | RemoteInfo[] | `git remote set-url` |
| `git.mirrorSet` / `git.mirrorUnset` / `git.mirrorList` | 既有契约不变 | | |

写操作均返回操作后的完整 remote 列表（前端免二次查询）。

CLI 对应：`dev-toolbox git remote list <repo>`、`add/remove/rename/set-url <repo> <args> --confirm`、`mirror|unmirror`（既有）。

## 校验与错误码

- 名字：`^[A-Za-z0-9._-]+$` 且不含 `..`；url：http(s)://、ssh://、git:// 或 `user@host:path` scp 形态；均拒换行/空白。
- 新增：`GIT_REMOTE_EXISTS`、`GIT_REMOTE_NAME_INVALID`、`GIT_REMOTE_URL_INVALID`、`GIT_REPO_NOT_FOUND`、`GIT_NOT_A_REPOSITORY`、`GIT_REMOTE_NOT_FOUND`。
- git 子命令白名单增补 `remote`。

## 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-11-03 | initial | remote 全生命周期操作契约 |
