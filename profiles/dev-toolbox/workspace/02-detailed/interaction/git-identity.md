# PageSpec — Git 身份管理

版本：v1.0 | 日期：2026-10-28 | 模块：git-identity-migration | 状态：Gate 1 待检查

## 入口与页面

侧边栏新增"Git 身份"入口，含四个面板：身份档案、应用与绑定、仓库审计、迁移与凭据。`window.devToolbox` 桥接操作以 `git.` 前缀白名单注册。

## 身份档案面板

- 列表：id、name/email、签名/格式标签、★ 全局生效、◆ 本地覆盖、绑定目录数；空状态引导"收编当前全局身份"。
- 编辑表单：name/email 必填校验；signingkey、gpgSign 开关、gpg.format 下拉（可留空=应用时 unset）；insteadOf 与账号选择器为可增删行；自定义 kv 键值对。
- 保存/删除均弹确认；删除提示"不影响已写入的 git 配置"。

## 应用与绑定面板

| Action | op | 状态与反馈 |
|---|---|---|
| 选择档案与作用域 | `git.applyProfile`(dryRun=true) | 显示将写入/将清除的每键清单与每仓预检结果；含将失败仓库时禁用执行并高亮原因 |
| 执行应用 | `git.applyProfile`(confirm) | 确认弹窗列明作用域与"全量覆盖会清除档案外受管键"；逐仓结果 success/failed/skipped + 明细 |
| 查看生效身份 | `git.effectiveIdentity` | 目录选择 + 生效 id 与每键来源（本地/目录绑定/全局）；manual/none 状态可见 |
| 绑定目录 | `git.bindProfile` | 目录可不存在；绑定后显示片段路径与钩子状态 |
| 解绑 | `git.unbindProfile` | 手改漂移时弹警告并要求确认，不静默丢弃 |
| 清除身份 | `git.clearIdentity` | 明确本地/全局作用域与将被清除的键 |

## 仓库审计面板

- 扫描：根目录+深度 → `git.scanRepos`；结果表列仓库、身份状态（✓/✗/⚠/◆）、镜像 ⊕、问题；状态用文字+图标不只靠颜色。
- 收编：勾选 ⚠ 项 → `git.importScanFindings`（预览确认）；`git.saveScanRoot` 保存默认目录。
- 审计：选仓库与期望档案 → `git.auditRepo`；作者分布表 + 污染标注 + mailmap 建议（标注"仅建议"）；strict 失败以醒目错误呈现，无自动修复按钮。

## Git 体检

面板内嵌 `git.gitDoctor` 结果：检查项、级别、明细；绑定健康包含钩子缺失/片段缺失/孤儿/失效目录/嵌套/漂移；只读提示，不提供一键修复。

## 镜像与凭据

- 镜像：`git.mirrorList/mirrorSet/mirrorUnset`；显示当前 pushUrls；set/unset 需确认。
- 凭据：`git.credentialList` 列 host/用户名（无 token 列）；login 说明由 helper 弹窗交互；set/remove 需确认；任何界面元素不显示、不粘贴回显 token。

## 迁移向导

`git.legacyPreview` → 显示来源路径、档案列表、冲突标注、备份目录 → 确认后 `git.legacyImport`；完成后显示 created/linked/conflicts；重复导入显示幂等结果。旧 gitid 侧不做任何写入。

## 状态与键盘

覆盖：无档案、git 未安装/不可用、非仓库路径、片段漂移、strict 污染、离线（扫描不涉网）、凭据助手缺失。表单与列表可键盘操作；状态有可读文本。

## API 对齐

字段、错误码、confirm 语义与 `api-git-identity.md` 一致；写操作响应含逐项变更明细，前端不自行推断覆盖范围。

## 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-10-28 | initial | 四面板交互与迁移向导 |
