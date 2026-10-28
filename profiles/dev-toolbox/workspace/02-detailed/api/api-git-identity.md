# Go 应用服务契约 — Git 身份管理

版本：v1.0 | 日期：2026-10-28 | 模块：git-identity-migration | 状态：Gate 1 待检查

> CLI/WebView 调用同一 Go 应用服务；不存在 HTTP server。写操作含 confirm；git 调用使用参数数组 + context + cwd。

## DTO

- `GitProfile`: id、name、email、signingKey、gpgSign、gpgFormat(`openpgp|ssh|x509|`)、customKv{}、insteadOf[]{url,match}、accounts[]{host,username}、gitdirs[]、marks{global,localRepos[],bindings[]}。
- `ConfigChange`: action(`set|unset`)、key、from?、to?、scope(`local|global|includeIf`)、target(仓库或 global)。
- `ApplyPlan`: profileId、dryRun、changes[]、perTarget[]、wouldFail[]。`ApplyResult`: perTarget[]{target,status(`success|failed|skipped`),changes[],error?}、summary。
- `EffectiveIdentity`: path、resolved(`id|manual|none`)、profileId?、keys[]{key,value,source(`local|includeIf|global|none`)}、insteadOf[]、accounts[]。
- `BindingInfo`: profileId、dir、realPath、snippetPath、hookPresent、drift、orphan、deadDir。
- `ScanFinding`: repoPath、identity{resolvedId,matchedProfileId,status(`ok|missing|manual|localOverride`)}、mirror{remote,pushUrls[],mirrored}、issue?。
- `AuditReport`: repo、expectedProfileId、authors[]{name,email,count,polluted}、mailmapSuggestions[]、strictPassed。
- `DoctorFinding`: check(`gitVersion|credentialHelper|insteadOfConflict|bindingHealth`)、level(`info|warn|error`)、detail。
- `LegacyGitidPreview`: found、sourcePath、sourceDigest、profiles[]、conflicts[]、backupDirectory。

## 操作

| op | 请求 | 响应 | 副作用 |
|---|---|---|---|
| `git.listProfiles` | — | GitProfile[] | 只读 |
| `git.saveProfile` | profile、confirm | GitProfile | 档案写入；重复 id 覆盖更新 |
| `git.removeProfile` | id、confirm | removed | 只删档案，不改已写入 git 配置 |
| `git.importGlobalIdentity` | asId?、confirm | GitProfile | 读当前全局身份生成档案 |
| `git.applyProfile` | id、scope(`global|repo`)、repos[]、dryRun、confirm | ApplyPlan 或 ApplyResult | dryRun 无副作用；执行按受管键全量覆盖+清除 |
| `git.effectiveIdentity` | path、prompt? | EffectiveIdentity | 只读；git 自身解析 |
| `git.bindProfile` | id、dir、confirm | BindingInfo | 写托管片段 + include 钩子；目录可不存在 |
| `git.unbindProfile` | id、dir?、confirm | BindingInfo、driftWarning? | 回收片段；手改漂移时警告 |
| `git.clearIdentity` | scope、path?、confirm | ConfigChange[] | 清身份键（默认 local） |
| `git.scanRepos` | root?、depth? | ScanFinding[]、savedRoot | 只读；root 显式传入或用已存默认 |
| `git.saveScanRoot` | root、depth、confirm | settings | 记录默认扫描目录 |
| `git.importScanFindings` | findingsDigest、ids[]、confirm | GitProfile[] | 预览确认后收编 ⚠ 为档案 |
| `git.auditRepo` | repo、expectedProfileId?、strict | AuditReport | 只读 git log；不写 .mailmap |
| `git.gitDoctor` | — | DoctorFinding[] | 只读体检 |
| `git.mirrorList` | repo? | mirror 状态 | 只读 |
| `git.mirrorSet` | repo、mirrorUrl、remote?、confirm | pushUrls | 写本地 pushurl |
| `git.mirrorUnset` | repo、remote?、confirm | pushUrls | 移除产品添加的镜像 pushurl |
| `git.credentialList` | host? | hosts[]{host,username} | 只读；无 token |
| `git.credentialLogin` | host | status | helper 自身交互（如 GCM 弹窗）；服务不捕获输出密文 |
| `git.credentialSet` | host、username、confirm | status | 经 credential 协议；token 走隐藏输入/管道，不进日志 |
| `git.credentialRemove` | host、username?、confirm | status | erase 协议 |
| `git.legacyPreview` | — | LegacyGitidPreview | 只读探测 gitid config.json |
| `git.legacyImport` | expectedDigest、profileIds[]、confirmBackup、confirmImport | ImportRecord | 备份+幂等导入 |

CLI 对应：`dev-toolbox git <profiles|use|current|bind|unbind|unset|scan|audit|doctor|remote|credential|legacy> ...`；`use --dry-run` 输出计划；批量失败退出码区分部分失败；`audit --strict` 污染退出码 2。

## 错误码与并发

`GIT_PROFILE_NOT_FOUND`, `GIT_PROFILE_INVALID`, `GIT_PROFILE_ID_CONFLICT`, `GIT_REPO_NOT_FOUND`, `GIT_NOT_A_REPOSITORY`, `GIT_CONFIG_WRITE_FAILED`, `GIT_SNIPPET_DRIFT`, `GIT_DIRECTORY_BOUND`, `GIT_HOOK_MISSING`, `GIT_AUDIT_POLLUTED`, `GIT_MIRROR_ALREADY_SET`, `GIT_CREDENTIAL_HELPER_UNAVAILABLE`, `GIT_CREDENTIAL_REJECTED`, `GIT_SCAN_ROOT_INVALID`, `GIT_LEGACY_SOURCE_CHANGED`, `GIT_LEGACY_UNKNOWN_SCHEMA`, `GIT_LEGACY_CONFIRMATION_REQUIRED`, `CONFIRMATION_REQUIRED`。

写操作在用户状态锁内执行并原子落盘；apply 对同一仓库串行，多仓库逐仓独立；dry-run 不获取写锁。导入在锁内比较来源摘要，重复导入返回既有记录。

## 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-10-28 | initial | 档案、应用、绑定、审计、体检、镜像、凭据与迁移契约 |
