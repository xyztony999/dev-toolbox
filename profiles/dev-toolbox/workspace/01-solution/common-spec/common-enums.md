# 公共枚举候选（draft）

诊断级别候选：info、warn、error。要求解析结果需要区分已知、未知、冲突、证据不完整。

执行结果需要表达成功、失败、跳过、取消及尽力回退结果，具体状态机和 code/label 在 D3 确定。

运行时发现枚举（D3 runtime-migration v1.0）：

| enum | code | label |
|---|---|---|
| RuntimeDiscoveryStatus | complete | 扫描完成 |
| RuntimeDiscoveryStatus | partial | 部分完成 |
| RuntimeDiscoveryStatus | cancelled | 已取消 |
| RuntimeVersionConfidence | exact | 精确识别 |
| RuntimeVersionConfidence | unknown | 未知 |
| RuntimeAvailability | available | 可用 |
| RuntimeAvailability | unavailable | 不可用 |
| RuntimeRegistrationOutcome | created | 已登记 |
| RuntimeRegistrationOutcome | linked | 已存在 |
| RuntimeRegistrationOutcome | skipped | 已跳过 |

运行时安装任务枚举（D3 runtime-download-install v1.0）：

| enum | code | label |
|---|---|---|
| RuntimeInstallState | resolving | 解析版本 |
| RuntimeInstallState | preview | 等待确认 |
| RuntimeInstallState | downloading | 下载中 |
| RuntimeInstallState | verifying | 校验中 |
| RuntimeInstallState | extracting | 解包中 |
| RuntimeInstallState | registering | 登记中 |
| RuntimeInstallState | complete | 已完成 |
| RuntimeInstallState | partial | 部分完成 |
| RuntimeInstallState | cancelled | 已取消 |
| RuntimeInstallState | failed | 失败 |

Git 身份枚举（D3 git-identity-migration v1.0）：

| enum | code | label |
|---|---|---|
| GitConfigScope | global | 全局配置 |
| GitConfigScope | local | 仓库本地配置 |
| GitConfigScope | includeIf | 目录绑定片段 |
| GitConfigSource | local | 仓库本地覆盖 |
| GitConfigSource | includeIf | 目录绑定生效 |
| GitConfigSource | global | 全局回退 |
| GitConfigSource | none | 未声明 |
| GitSigningFormat | openpgp | OpenPGP |
| GitSigningFormat | ssh | SSH |
| GitSigningFormat | x509 | X.509 |
| GitApplyTargetStatus | success | 成功 |
| GitApplyTargetStatus | failed | 失败 |
| GitApplyTargetStatus | skipped | 跳过 |
| GitScanIdentityStatus | ok | 与已知档案一致 |
| GitScanIdentityStatus | missing | 缺失身份键 |
| GitScanIdentityStatus | manual | 手配身份 |
| GitScanIdentityStatus | localOverride | 仓库本地覆盖 |
| GitDoctorLevel | info | 提示 |
| GitDoctorLevel | warn | 警告 |
| GitDoctorLevel | error | 错误 |
