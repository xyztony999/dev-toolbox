# PRD — Git 身份密钥生成

版本：v1.0 | 日期：2026-11-27 | 模块：git-gpg-keygen | 状态：Gate 1 待检查

## 1. 目标

本机开发者在 DevToolbox 内为**已有** Git 身份档案生成或选用 OpenPGP / SSH 签名密钥，自动写回 `signingkey` 与 `gpg.format`，并按确认决定是否打开 `gpgSign`。口令与私钥永不落盘到产品存储。解决「档案有签名字段却无法在产品内安全生成材料」的缺口。

## 2. 功能类型与范围

- **功能类型**：工具类（无独立聚合状态机；写回既有 IdentityProfile 聚合根字段）。
- **做什么**：探测工具 → 选项枚举 → 预览 → 生成/选用写回档案 → 导出公钥；CLI + 桌面。
- **不做什么**：x509 生成；撤销/删除/改口令；密钥服务器；代管私钥；改系统信任库；隐式新建档案。

## 3. 角色

| 角色 | 职责 |
|------|------|
| 本机开发者 | 选定档案、类型、算法/有效期/口令；确认预览；选用已有密钥；导出公钥 |

## 4. 领域对象

### 4.1 本域信息

| 项目 | 内容 |
|------|------|
| 本域名称 | Git 身份密钥生成（挂 git 域，增强） |
| 领域分层 | 支撑域 |
| 聚合根 | 无新增；写回关联域聚合根 **IdentityProfile**（git-identity） |

### 4.2 领域对象清单

| 对象名 | 分类 | 与上级关系 | 核心业务字段 | 关键枚举字段 | 备注 |
|--------|------|-----------|------------|------------|------|
| IdentityProfile | 关联领域对象 | 外部聚合根 | id、name、email、signingKey、gpgSign、gpgFormat | GitSigningFormat | 本模块只更新签名三字段 |
| KeygenRequest | 过程数据 P5 | — | profileId、keyType、algorithm、expiry、passphrase?、enableGpgSign、comment?、outputPath? | GitKeyType、GitKeyAlgorithm、GitKeyExpiry | 口令仅请求瞬态 |
| KeygenPlan | 过程数据 P2/P5 | — | commandSummary[]、profilePatch、toolPath、warnings[] | — | dry-run 产物；摘要不含口令 |
| KeygenResult | 过程数据 P2 | — | fingerprintOrPath、publicKeyPreview?、profile | — | 成功后含更新后档案 |
| LocalKeySummary | 过程数据 P2 | — | keyType、id、fingerprint、uidOrComment、publicKeyPath?、createdAt? | GitKeyType | 列表项；无私钥 |
| PublicKeyExport | 过程数据 P2 | — | armorOrText、writtenPath? | — | 导出结果 |

### 4.3 关联领域对象

| 关联领域对象 | 所属业务域 | 关联字段 | 需获取信息 | 触发关系 | 映射模式 |
|------------|--------|---------|----------|---------|---------|
| IdentityProfile | git-identity | profileId | name、email（作 OpenPGP UID）、既有签名字段 | 生成/选用成功后写回 | ACL（本模块不改档案其它字段语义） |

### 4.4 过程数据分类

| 数据项 | 类别 | 处理方式 |
|--------|------|---------|
| 算法/有效期下拉 | P2 | 由 options 接口返回 |
| 口令输入框状态 | P3 | 仅前端瞬态；提交后清空 |
| 命令预览行 | P2 | 来自 preview |
| 导出目标路径 | P1 | 用户显式选择 |

## 5. 功能需求

| ID | 需求 | 验收标准 |
|---|---|---|
| KG-01 | 工具探测 | options 返回 gpg/ssh-keygen 是否可用及路径；不可用时 create/list 返回稳定错误码与人话说明 |
| KG-02 | 类型分支 | 同一入口 `keyType=openpgp|ssh`；openpgp→gpg，ssh→ssh-keygen + 档案 `gpgFormat=ssh` |
| KG-03 | 算法与有效期 | 用户可选；各类型有默认值与可选集合（见 §7）；非法组合预览即拒 |
| KG-04 | 档案绑定 | create/select 必须带已存在 profileId；成功后档案含正确 signingKey 与 gpgFormat |
| KG-05 | gpgSign 默认 | 请求可带 enableGpgSign；桌面默认勾选「打开提交签名」；未传时服务端默认 `true` |
| KG-06 | 口令安全 | 允许空口令；有口令时仅进程内短时使用；响应/日志/事件无口令与私钥正文 |
| KG-07 | 预览 | preview 与 create 同一校验；preview 不调生成、不写档案；摘要可含将执行工具名与参数骨架（口令位置以占位符表示） |
| KG-08 | 生成 OpenPGP | 用档案 name+email 作为 UID；返回指纹；signingKey=指纹（或实现选定的完整 fingerprint 形式，与 git 可用值一致） |
| KG-09 | 生成 SSH | 默认输出路径策略见 §7；signingKey=公钥文件绝对路径；私钥仅落用户本机路径 |
| KG-10 | 列表 | 按类型列出本机可用密钥摘要；OpenPGP 来自 gpg 列表；SSH 扫描约定目录（§7）并标注是否像签名钥 |
| KG-11 | 选用 | 将选中摘要写回档案字段，不复制/移动私钥 |
| KG-12 | 导出公钥 | 返回 armored/文本；或写入用户显式路径；拒绝静默覆盖未确认文件（见 API） |
| KG-13 | 双入口 | CLI 与桌面共用服务；桌面挂在档案编辑「提交签名」分组 |
| KG-14 | 平台 | Windows + Linux；路径与工具探测跨平台 |
| KG-15 | 测试隔离 | 单测强制临时 GNUPGHOME / 临时 SSH 目录；不改真实用户密钥环 |

## 6. 状态与流转

无业务状态机。操作流：

```
options（只读）
  → 用户填参
  → preview（只读计划）
  → create|select（confirm）→ 更新档案
  → export（可选，只读/写用户指定文件）
```

## 7. 工程默认（D0 未细定时定稿）

> 以下为合理工程默认，实现与 D3/D4 对齐；变更须走设计修订。

### 7.1 算法枚举默认

| keyType | 默认 algorithm | 可选 |
|---------|----------------|------|
| openpgp | `ed25519` | `ed25519`、`rsa4096`、`rsa3072`、`nistp256` |
| ssh | `ed25519` | `ed25519`、`rsa4096`、`ecdsa256` |

### 7.2 有效期

| keyType | 默认 | 可选 | 说明 |
|---------|------|------|------|
| openpgp | `2y` | `never`、`1y`、`2y`、`5y` | 映射 gpg `--default-expire` / batch expire |
| ssh | `never` | 仅 `never` | SSH 密钥文件无原生有效期；UI 展示「不适用（无到期）」且参数固定 never |

### 7.3 OpenPGP UID

固定使用目标档案的 `name` + `email`；档案缺 name/email 时预览失败（`GIT_PROFILE_INVALID` 语义族）。

### 7.4 SSH 输出路径

默认：`{UserSSHDir}/devtoolbox_{profileId}_{yyyyMMddHHmmss}`（无后缀私钥，`.pub` 为公钥）。`UserSSHDir` 默认识别 `~/.ssh`（测试可注入）。允许请求覆盖 `outputPath`（目录须已存在或可创建的父目录策略：仅创建最后一级文件，不 mkdir -p 多级——若父目录不存在则失败）。

### 7.5 signingKey 形态

| keyType | signingKey 写入值 |
|---------|-------------------|
| openpgp | 40 位或 gpg 返回的完整 fingerprint 十六进制（与 `gpg -K` 一致、git 可识别） |
| ssh | 公钥文件绝对路径（git `user.signingkey` 对 ssh 格式的惯用值） |

### 7.6 gpg.format 写入

| keyType | gpgFormat |
|---------|-----------|
| openpgp | `openpgp` |
| ssh | `ssh` |

x509 本轮不生成；手填仍走既有档案表单。

## 8. 安全与可靠性

- 外部进程：参数数组、context 超时、明确 cwd/env；禁止 shell 拼接。
- 口令：不进 argv 明文（优先 stdin / gpg batch 约定）；内存用后清零尽力而为。
- 并发：同一 profileId 的 create/select 在用户状态锁内串行；preview 不持写锁。
- 失败：工具缺失、生成失败、档案不存在均返回稳定错误码；部分成功（已生成密钥但档案写失败）必须在响应中标明，并给出「密钥已在本机产生、档案未更新」警告，便于用户手工填 fingerprint/路径或重试 select。

## 9. PRD 完备性自检

| 层 | 项 | 结果 |
|----|----|------|
| G1–G5 | 目标/范围/角色/对象/关系 | 通过 |
| T9 | 工具类 I/O | 通过（§5–§7） |
| N1–N3 | 非功能/异常/规则 | 已写 §7–§8 |

## 10. 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-11-27 | initial | D0 定稿落地；算法/有效期/路径工程默认写明 |
