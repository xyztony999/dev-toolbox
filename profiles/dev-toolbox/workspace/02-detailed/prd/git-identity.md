# PRD — Git 身份管理

版本：v1.0 | 日期：2026-10-28 | 模块：git-identity-migration | 状态：Gate 1 待检查

## 1. 目标

本机开发者在 DevToolbox 内维护多套 Git 身份档案，把身份全量覆盖地应用到全局/仓库/目录树，随时查看到效身份与来源，并审计历史提交是否被污染。迁移自 gitid 的行为语义，最终产品不依赖被管理的 Node。

## 2. 领域对象

| 对象 | 关键字段 | 说明 |
|---|---|---|
| IdentityProfile（聚合根） | id、name、email、signingkey、gpgsign、gpg.format、customKv、insteadOf[]、accounts[]、gitdirs[] | token 不入档案；重复保存为覆盖更新 |
| Binding | profileId、dir、真实路径归一 | 托管片段 + include 钩子 |
| EffectiveIdentity | id 或 manual/none、perKeySource | local → includeIf → global |
| ScanFinding | repoPath、identityStatus、mirrorStatus | ✓/✗/⚠/◆/⊕ 语义 |
| AuditReport | authorDistribution[]、polluted[]、mailmapSuggestions[] | 只读，不写 .mailmap |
| ImportRecord | sourceDigest、backupPath、created/linked/conflicts | 幂等键=来源摘要 |

## 3. 功能需求

| ID | 需求 | 验收标准 |
|---|---|---|
| GIT-01 | 档案管理 | 档案字段完整保存与列出；删除档案不改动已写入的 git 配置并如实提示 |
| GIT-02 | 全量覆盖应用 | 应用后受管键与档案一致；档案缺失键被 unset（含 gpg.format、insteadOf、账号选择器） |
| GIT-03 | 作用域 | 默认全局；可指定仓库本地或多仓库；每仓独立成功/失败结果 |
| GIT-04 | dry-run | 与真跑同一算法输出将写入/将清除清单，不落盘；批量含将失败路径时给出可预检的失败汇总 |
| GIT-05 | 生效解析 | 任意目录返回生效 id/manual/none 与每键来源；与 `git config` 实际值一致 |
| GIT-06 | 目录绑定 | 绑定后目录树下仓库（含新 clone）经 includeIf 生效；Windows 按真实路径匹配；目录不存在也可绑定 |
| GIT-07 | 解绑回收 | 回收片段；片段被手改时警告并保留内容；孤儿片段/失效目录在体检中可见 |
| GIT-08 | 仓库扫描 | 逐仓身份与镜像状态；支持深度、保存默认目录；不执行项目命令 |
| GIT-09 | 扫描收编 | ⚠ 手配身份可预览并收编为档案；确认前不写任何档案 |
| GIT-10 | 提交审计 | 作者分布与污染标注；mailmap 仅建议；strict 存在污染时判定失败 |
| GIT-11 | Git 体检 | git 版本、凭据助手、insteadOf 冲突、绑定健康（钩子/片段缺失/孤儿/失效目录/嵌套/漂移）只读可见 |
| GIT-12 | 镜像推送 | 镜像 set 后 push 同时推原始与镜像；unmirror 清理；fetch/凭据不受影响 |
| GIT-13 | 凭据选择 | 账号用户名写入 credential.<host>.username；login/set/remove 走 credential 协议；token 不落盘不回显 |
| GIT-14 | 档案迁移 | gitid config.json 预览（冲突/重复标注）、备份、按来源摘要幂等导入；旧文件只读 |
| GIT-15 | 凭据隔离 | 任何响应、日志、广播事件不含 token；服务进程内存外不保存 |

## 4. 状态与流转

档案与绑定无状态机；应用操作输出计划（dry-run）→ 确认 → 逐仓执行结果（success/failed/skipped + 每键明细）。导入流程：探测 → 预览（含摘要）→ 备份+导入 → 记录；重复导入同摘要幂等返回原记录。

## 5. 安全与可靠性

所有 git 与外部进程调用使用参数数组、context 超时与明确 cwd；配置写入原子替换；托管片段写入前检测漂移；`--strict` 审计失败是明确判定，不做任何自动修复；扫描默认跳过依赖/缓存目录。

## 6. 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-10-28 | initial | gitid 语义移植的档案、应用、绑定、审计、体检与迁移需求 |
