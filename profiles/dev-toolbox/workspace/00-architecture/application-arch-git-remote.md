# 应用架构 — 仓库 Git Remote 管理

版本：v1.0 | 日期：2026-11-03 | 模块：git-remote-management

## 1. 服务边界

| 服务 | 职责 |
|---|---|
| RemoteManager（扩展 internal/gitidentity/mirror.go → remote.go） | remoteList（复用 getRemotes+账号选择器）；remoteAdd/Remove/Rename/SetUrl 经 `git remote` 子命令与 `git config --local`；镜像 Set/Unset 沿用现状 |

## 2. 调用关系

CLI/WebView → 既有 gitidentity 应用服务（白名单子命令增补 `remote`）。项目详情仓库卡片与 Git 身份页镜像区块共用同一服务；前端不拼任何 git 命令。

## 3. 数据与副作用

只写目标仓库的本地配置（.git/config）；不写全局配置、不动凭据、无网络请求。名字/URL 校验在服务层先行（正则 + 协议白名单），git 参数数组二次防御（拒换行）。

## 4. 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-11-03 | 初始应用架构 |
