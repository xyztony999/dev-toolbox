# D0 需求澄清 — 仓库 Git Remote 管理

版本：v1.0 | 日期：2026-11-03 | 模块：git-remote-management | 类型：git-identity-migration 的增强切片

## 来源与现状差距

gitid 原有 `remote [list | mirror <url> | unmirror] [--remote <name>]`。已迁移部分：镜像推送（pushurl 双目标）的 CLI/桥接/页面。缺口：

1. 只能管镜像 pushurl，不能增删 remote、不能改 fetch 地址、不能改名。
2. 管理入口藏在「Git 身份 → 迁移与凭据」标签的手输路径区块，与"每个仓库"的语境脱节；项目详情页仓库卡片已只读显示 remote（name/url），应就地管理。
3. 无单个 remote 的完整视图（fetch 地址 + push 地址列表 + 镜像状态 + 生效账号选择器）。

## 能力基线

| 能力 | 语义 | 边界 |
|---|---|---|
| remote 列表 | 每个 remote：fetch url、pushurl 列表、是否镜像（pushurl≥2）、生效账号选择器（http(s)） | 只读 |
| 新增 remote | `git remote add <name> <url>` | 显式确认 |
| 删除 remote | `git remote remove <name>` | 显式确认；删除前展示其 url |
| 重命名 remote | `git remote rename <old> <new>` | 显式确认；迁移全部 remote.* 键 |
| 修改 fetch 地址 | set-url（remote.<name>.url） | 显式确认 |
| 镜像推送 | 既有 mirrorSet/mirrorUnset 原样保留 | 不变 |

## 已确认规则

1. 全部写操作经 `git remote` 子命令（参数数组）或 `git config --local`；git 子命令白名单增补 `remote`。
2. remote 名严格校验 `^[A-Za-z0-9._-]+$`（名字会拼进 `remote.<name>.*` 配置键，防配置段注入）；url 仅接受 http(s)://、ssh（user@host:path、ssh://）、git:// 形态且不含换行（gitArgs 已拒）。
3. 网络操作不在本模块：不做 ls-remote 连通性测试、fetch、push；操作即写本地配置，结果由 git 自身保证。
4. 作用域始终单仓库：请求必须带 repo 路径（非 git 仓库报 GIT_NOT_A_REPOSITORY）；不提供跨仓库批量 remote 操作。
5. 主入口移到项目详情页仓库卡片（「管理 Remote」展开面板）；Git 身份页保留现有镜像区块（同一后端），手输路径场景仍可用。
6. 删除/重命名 remote 不触碰凭据（凭据归 credential helper 与账号选择器语义）。

## 暂不纳入

`git fetch/push/pull`、远端分支浏览、remote 顺序调整、`--tags`/`--mirror=mirror` 裸镜像仓库模式、跨仓库批量、SSH known_hosts 管理。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-11-03 | 补齐 gitid remote 管理的完整能力与就地入口 |
