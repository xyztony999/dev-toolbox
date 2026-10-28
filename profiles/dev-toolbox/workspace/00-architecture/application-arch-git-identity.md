# 应用架构 — Git 身份管理

版本：v1.0 | 日期：2026-10-28 | 模块：git-identity-migration

## 1. 服务边界

| 服务 | 职责 |
|---|---|
| IdentityProfileStore | 档案 CRUD 与清单；DevToolbox 自有数据目录；不存 token |
| IdentityApplier | 受管键全量覆盖/清除、dry-run 计划、多仓批量与逐仓结果 |
| EffectiveResolver | 按目录解析生效身份与每键来源（local → includeIf → global），调用 git 读取 |
| DirectoryBinder | 托管片段物化、include 钩子维护、gitdir/i 真实路径匹配、漂移检测与回收 |
| RepoScanner | 逐仓只读身份/镜像审计，json/markdown/csv 呈现数据 |
| CommitAuditor | git log 作者分布、污染对照、mailmap 建议、strict 判定 |
| GitDoctor | git 版本/凭据助手/insteadOf 冲突/绑定健康只读检查 |
| MirrorManager | 仓库级 pushurl 镜像配置 |
| CredentialProxy | git credential 协议直通 helper，不落盘不回显 |
| LegacyGitidImport | gitid config.json 只读探测、预览、备份、幂等导入 |

## 2. 调用关系

CLI/WebView → 共用 Go 服务（IdentityApplier/DirectoryBinder 写 git 配置与托管片段；EffectiveResolver/RepoScanner/CommitAuditor/GitDoctor 只读调 git 命令）。外部进程一律参数数组 + context + 明确 cwd（GC-03）。前端只提交档案字段与作用域选择，不传 git 配置键值之外的任意命令。

## 3. 数据与副作用

- DevToolbox 用户配置目录：`git-identity.json`（档案与绑定 gitdirs）、`git-identity/snippets/<id>.gitconfig`（托管片段）、备份目录。
- 用户真实 `~/.gitconfig`：仅 include 钩子一行（绑定存在时）；片段路径指向 DevToolbox 目录。
- git 仓库本地配置：仅显式 apply/unset/mirror 操作经 `git config --local` 写入。
- 副作用全部原子写入（临时文件+rename）；批量操作逐仓记录计划/结果；失败不回滚已成功仓库，但汇总明确列出。

## 4. 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-10-28 | 初始应用架构 |
