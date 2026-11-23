# PageSpec — Git 身份管理

版本：v1.1 | 日期：2026-11-23 | 模块：git-identity-migration | 状态：Gate 1 待检查

## 入口与页面

侧边栏"Git 身份"入口，含四个页签：身份档案、生效范围、仓库与体检、迁移与凭据。`window.devToolbox` 桥接操作以 `git.` 前缀白名单注册。v1.1 重设计「身份档案」编辑交互与「生效范围」（原"应用与绑定"）信息架构，解决编辑表单拥挤错乱、作用域三层关系不直观两类困惑。

## 身份档案面板

- 列表：首字母标识 + id + 姓名/邮箱 + 签名/重写/账号摘要 + 「生效范围」行（全局默认 / 各绑定目录树）；全局默认以强调色标记；空状态引导"收编当前全局身份/新建档案"。
- 编辑/新建改为右侧抽屉（与 Remote、终端面板同一交互语言）：单列垂直分组表单（档案 / 提交签名 / URL 重写 / 账号选择器），label 用人话、git 键名作等宽注解；name/email 必填、id 与邮箱校验与服务端 `Profile.Validate` 同源；signingkey、gpgSign 开关、gpg.format 下拉（留空=应用时 unset）；insteadOf 与账号选择器为可增删行（成对校验）；customKv/gitdirs 不在表单编辑、保存时原样带回。
- 校验失败滚到第一个出错字段；保存失败回显在抽屉内（页面顶部错误条会被抽屉遮住）；有未保存修改时关闭弹"放弃修改"确认（z-index 高于抽屉）。
- 保存/删除均弹确认；删除带绑定档案时提示目录规则与片段一并移除、已写入 git 的其它配置不变。

## 生效范围面板

以"范围从大到小"的层级呈现三层作用域，缩进深度对应范围大小/优先级：全局默认（根）→ 目录规则（一级缩进+竖线）→ 单个仓库（二级缩进）。每个写操作嵌在它影响的那一层内。

| Action | op | 状态与反馈 |
|---|---|---|
| 更换全局默认 | `git.applyProfile`(scope=global, dryRun=true) | 行内展开选档案→逐键「当前值→应用后」变更表；无变更提示"已一致"；桥接 v1.1 补全局 dry-run 分支（此前全局预览恒空） |
| 确认更换 | `git.applyProfile`(scope=global, confirm) | 确认弹窗列明"全量覆盖会清除档案外受管键" |
| 目录规则列表 | 档案 `gitdirs` + `git.gitDoctor`(bindings) | 每行目录→档案；健康标记（目录不存在/配置被手改/片段缺失）带 tooltip；只读体检失败不阻塞列表 |
| 绑定目录 | `git.bindProfile` | 目录+档案+浏览（原生 `app.pickFolder`）；已绑定他档时人话报错"先解绑" |
| 解绑 | `git.unbindProfile` | 确认弹窗；手改漂移时强调回收，不静默丢弃 |
| 查某个仓库 | `git.effectiveIdentity` | 路径+浏览→生效 id/手动/无 + 每键来源标签（仓库设置/目录规则/全局默认/未设置） |
| 写进仓库设置 | `git.applyProfile`(scope=repos, dryRun/confirm) | 多仓库 tags 输入+浏览；逐仓预检表；含 failed 时禁用执行并高亮原因；应用后逐仓 success/failed 明细 |

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
| v1.1 | 2026-11-23 | redesign | 编辑改抽屉分组表单；"应用与绑定"重设计为"生效范围"三层规则表；桥接补全局 dry-run；真机截图自查 |
