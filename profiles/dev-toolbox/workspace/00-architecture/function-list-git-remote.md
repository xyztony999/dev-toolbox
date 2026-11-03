# 功能清单 — 仓库 Git Remote 管理

版本：v1.0 | 日期：2026-11-03 | 用户类型：本机开发者

## 项目详情页（主入口）

- 仓库卡片增加「管理 Remote」：展开面板列出每个 remote（fetch 地址、push 地址、镜像标记、生效账号）；行内操作：改地址、重命名、删除；面板级：新增 remote。
- 镜像推送开关整合进面板（沿用既有语义）。

## Git 身份页（次入口）

- 「迁移与凭据」现有镜像区块保留；可打开同一管理面板。

## CLI

- `dev-toolbox git remote list|add|remove|rename|set-url|mirror|unmirror`（均需仓库路径；写操作 --confirm）。

## 本模块不提供

fetch/push/pull、远端分支浏览、裸镜像仓库模式、跨仓库批量、SSH known_hosts 管理。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-11-03 | 初始功能清单 |
