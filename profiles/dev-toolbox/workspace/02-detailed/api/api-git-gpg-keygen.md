# Go 应用服务契约 — Git 身份密钥生成

版本：v1.0 | 日期：2026-11-27 | 模块：git-gpg-keygen | 状态：Gate 1 待检查

> CLI/WebView 调用同一 Go 应用服务；不存在 HTTP server。写操作含 confirm；本机工具调用使用参数数组 + context + 明确 env。路径风格对齐既有 `git.*`。功能类型：**工具类**。

## 1. 模块设计概述

### 1.1 业务背景及目标

为已有 IdentityProfile 生成或选用 OpenPGP/SSH 签名密钥并写回签名字段，补齐档案侧「有字段无能生成」缺口。

### 1.2 边界说明

| 做 | 不做 |
|----|------|
| options / preview / create / list / select / export | x509 生成、撤销/删除/改口令、密钥服务器、代管私钥 |
| 写回 signingKey、gpgFormat、gpgSign | 隐式新建档案；改 name/email 等无关字段 |
| 导出公钥 | 导出或回传私钥正文 |

### 1.3 设计决策

| 决策 | 理由 |
|------|------|
| 操作挂 `git.` 前缀，不另起域前缀 | 与 git-identity / git-remote 同一应用服务与白名单 |
| create/select 强制 profileId | D0：禁止悬空密钥不落档案 |
| enableGpgSign 默认 true | D0：默认建议打开提交签名 |
| SSH expiry 仅 never | SSH 文件无原生有效期 |
| 部分成功显式警告 | 本机密钥已产生但档案写失败时不可静默 |

### 1.4 领域对象总览

- 关联聚合根：IdentityProfile（既有）
- 过程对象：KeygenOptions、KeygenPlan、KeygenResult、LocalKeySummary、PublicKeyExport

### 1.5 上下文映射

| 对方 | 模式 | 说明 |
|------|------|------|
| git-identity ProfileStore | 同进程调用 | 读校验 + 签名字段更新 |
| 本机 gpg / ssh-keygen | ACL 适配 | 版本与参数差异在适配层消化 |

## 2. 跨聚合根设计

无（单关联聚合根字段更新）。

## 3. 应用/基础设施层设计

### 3.1 接口规范

- 操作名：`git.<resource><Action>` 驼峰，与既有表一致。
- 写操作：请求含 `confirm: true`；缺省 → `CONFIRMATION_REQUIRED`。
- 响应：业务结果 JSON；错误为稳定 code + message（面向用户、直白）。
- 权限：本机单用户；无多角色矩阵。

### 3.2 依赖与约束

- 依赖本机 `gpg`（openpgp）或 `ssh-keygen`（ssh）；探测失败不降级为空成功。
- 测试必须可注入 `GNUPGHOME`、SSH 主目录与可执行路径。

### 3.3 公共规范 — 枚举

写入 `common-enums.md`（本模块新增）：

| enum | code | label |
|------|------|-------|
| GitKeyType | openpgp | OpenPGP |
| GitKeyType | ssh | SSH 签名 |
| GitKeyAlgorithm | ed25519 | Ed25519 |
| GitKeyAlgorithm | rsa4096 | RSA 4096 |
| GitKeyAlgorithm | rsa3072 | RSA 3072 |
| GitKeyAlgorithm | nistp256 | NIST P-256 |
| GitKeyAlgorithm | ecdsa256 | ECDSA P-256 |
| GitKeyExpiry | never | 永不过期 |
| GitKeyExpiry | 1y | 1 年 |
| GitKeyExpiry | 2y | 2 年 |
| GitKeyExpiry | 5y | 5 年 |

既有 `GitSigningFormat`（openpgp/ssh/x509）继续用于档案字段；本模块生成仅写 openpgp|ssh。

**算法 × 类型合法集**（服务端强制）：

| keyType | 允许 algorithm |
|---------|----------------|
| openpgp | ed25519, rsa4096, rsa3072, nistp256 |
| ssh | ed25519, rsa4096, ecdsa256 |

**有效期 × 类型**：openpgp 允许上表全部 expiry；ssh 仅 `never`。

### 3.4 错误码

| code | 何时 |
|------|------|
| `GIT_PROFILE_NOT_FOUND` | profileId 不存在 |
| `GIT_PROFILE_INVALID` | 档案缺 name/email（OpenPGP UID）或签名字段校验失败 |
| `GIT_KEYGEN_TOOL_MISSING` | 对应工具未安装或不在 PATH |
| `GIT_KEYGEN_TOOL_FAILED` | 本机工具非 0 退出（detail 含截断后的 stderr，无私钥/口令） |
| `GIT_KEYGEN_PARAMS_INVALID` | 类型/算法/有效期/路径不合法 |
| `GIT_KEYGEN_PASSPHRASE_REJECTED` | 工具拒绝口令策略（若可识别） |
| `GIT_KEY_NOT_FOUND` | list/select/export 指定指纹或路径不存在 |
| `GIT_KEY_EXPORT_PATH_INVALID` | 导出路径非法或父目录不存在 |
| `GIT_KEY_EXPORT_EXISTS` | 目标文件已存在且未 `overwrite=true` |
| `GIT_KEYGEN_PROFILE_WRITE_FAILED` | 密钥已生成但档案写入失败（见部分成功） |
| `CONFIRMATION_REQUIRED` | 写操作未 confirm |

## 4. DTO

### KeygenToolStatus

| 字段 | 类型 | 说明 |
|------|------|------|
| keyType | GitKeyType | |
| available | bool | |
| executablePath | string? | 探测到的路径 |
| version | string? | 尽力解析；未知可空 |

### KeygenOptions

| 字段 | 类型 | 说明 |
|------|------|------|
| tools | KeygenToolStatus[] | openpgp + ssh 两项 |
| algorithms | { keyType, values: GitKeyAlgorithm[] }[] | |
| expiries | { keyType, values: GitKeyExpiry[], default: GitKeyExpiry }[] | |
| defaults | { openpgp: {algorithm, expiry}, ssh: {algorithm, expiry} } | |

### KeygenCreateRequest

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| profileId | string | 是 | 已有档案 |
| keyType | GitKeyType | 是 | |
| algorithm | GitKeyAlgorithm | 否 | 缺省用 defaults |
| expiry | GitKeyExpiry | 否 | 缺省用 defaults；ssh 强制 never |
| passphrase | string | 否 | 仅请求；永不回显 |
| enableGpgSign | bool | 否 | 缺省 true |
| comment | string | 否 | SSH 注释；默认 `devtoolbox:{profileId}` |
| outputPath | string | 否 | 仅 ssh；私钥路径（无后缀）；见 PRD §7.4 |
| dryRun | bool | preview 用 | create 忽略 |
| confirm | bool | create 是 | |

### KeygenPlan（preview 响应）

| 字段 | 类型 | 说明 |
|------|------|------|
| profileId | string | |
| keyType | GitKeyType | |
| toolPath | string | |
| commandSummary | string[] | 人类可读步骤/参数骨架；口令位置为 `<passphrase>` |
| profilePatch | { signingKey: string\|"(pending)", gpgFormat, gpgSign } | pending 表示生成后才知 fingerprint |
| warnings | string[] | 如将覆盖已有 signingKey |
| wouldFail | string? | 预检失败原因 code |

### KeygenResult

| 字段 | 类型 | 说明 |
|------|------|------|
| profile | GitProfile | 更新后档案（字段同 api-git-identity） |
| keyType | GitKeyType | |
| signingKey | string | 实际写入值 |
| fingerprint | string? | openpgp 指纹；ssh 可为公钥指纹摘要 |
| publicKeyPath | string? | ssh 公钥路径 |
| publicKeyPreview | string? | 截断公钥头若干行（可选） |
| localKeyCreated | bool | 本机是否已产生新密钥 |
| profileUpdated | bool | 档案是否已更新 |
| warning | string? | 部分成功时说明 |

### LocalKeySummary

| 字段 | 类型 | 说明 |
|------|------|------|
| keyType | GitKeyType | |
| id | string | 稳定选用 id（openpgp=fingerprint；ssh=公钥绝对路径） |
| fingerprint | string | 展示用 |
| uidOrComment | string | OpenPGP UID 或 SSH comment |
| publicKeyPath | string? | ssh |
| createdAt | string? | 若可解析 |

### KeyExportRequest

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyType | GitKeyType | 是 | |
| id | string | 是 | list 返回的 id |
| writePath | string | 否 | 显式文件路径；空则只返回文本 |
| overwrite | bool | 否 | 默认 false |
| confirm | bool | 写文件时是 | |

### PublicKeyExport

| 字段 | 类型 | 说明 |
|------|------|------|
| text | string | 公钥正文（armored 或 OpenSSH 公钥行） |
| writtenPath | string? | 若写入文件 |

## 5. 操作

| op | 请求 | 响应 | 副作用 |
|---|---|---|---|
| `git.keygenOptions` | — | KeygenOptions | 只读探测工具 |
| `git.keygenPreview` | KeygenCreateRequest（可无 confirm；dryRun 语义） | KeygenPlan | 无生成、无档案写 |
| `git.keygenCreate` | KeygenCreateRequest + confirm | KeygenResult | 调本机生成 + 写档案签名字段 |
| `git.keyList` | keyType | LocalKeySummary[] | 只读 |
| `git.keySelect` | profileId、keyType、id、enableGpgSign?、confirm | KeygenResult | 只写档案；不生成 |
| `git.keyExport` | KeyExportRequest | PublicKeyExport | 无或写用户指定文件 |

### 5.1 行为细则

**git.keygenCreate**

1. 校验 profile 存在；openpgp 时 name/email 非空。
2. 校验算法/有效期合法集与工具 available。
3. 调用本机工具；成功得到 fingerprint/路径。
4. 组装 patch：`signingKey`、`gpgFormat`（openpgp|ssh）、`gpgSign=enableGpgSign`（默认 true）；经既有档案更新路径写入（保留其它字段）。
5. 若步骤 3 成功且 4 失败：返回错误 `GIT_KEYGEN_PROFILE_WRITE_FAILED`，body 仍尽量带上 `localKeyCreated=true`、fingerprint/path，便于用户 `git.keySelect` 补救。

**git.keyList**

- openpgp：`gpg --list-secret-keys` 解析（沙箱 GNUPGHOME 可测）。
- ssh：扫描 `{UserSSHDir}` 下 `*.pub`（及默认 `devtoolbox_*` 前缀优先排序）；不递归超深；忽略无法读的文件。

**git.keySelect**

- 校验 id 仍可解析为存在密钥；写档案字段同 create 的 patch 规则。

**git.keyExport**

- openpgp：armor 公钥；ssh：读 `.pub` 文本。
- `writePath` 非空时需 confirm；已存在且 overwrite≠true → `GIT_KEY_EXPORT_EXISTS`。

### 5.2 CLI 对应

```
dev-toolbox git keygen options
dev-toolbox git keygen preview --profile <id> --type openpgp|ssh [--algo ...] [--expiry ...]
dev-toolbox git keygen create  --profile <id> --type ... [--passphrase-stdin] --confirm
dev-toolbox git keygen list    --type openpgp|ssh
dev-toolbox git keygen select  --profile <id> --type ... --id <fingerprint-or-path> --confirm
dev-toolbox git keygen export  --type ... --id ... [--out path --confirm --overwrite]
```

口令：CLI 支持 `--passphrase-stdin`（从 stdin 读一行）；禁止 argv 长选项传口令明文进进程列表（文档约束；若提供 `--passphrase` 仅限测试构建）。

### 5.3 桥接白名单

桌面 `window.devToolbox` 增补上述 6 个 `git.*` 操作；事件不推送口令字段。

## 6. 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-11-27 | initial | 六操作契约；枚举 GitKeyType/Algorithm/Expiry；错误码与部分成功语义 |
