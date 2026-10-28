# D0 需求澄清 — Git 身份核心迁移

版本：v1.0 | 日期：2026-10-28 | 模块：git-identity-migration

## 迁移来源与范围

- 来源：gitid `e33e728`（Node ≥18 CLI + Electron 桌面，共用 `~/.config/gitid/config.json` 身份档案）。工作区仅 `.claude/.profile-mounts.json` 本地修改，源码 clean。
- 本模块把 gitid 的 Git 身份语义移植为 DevToolbox 的 Go 应用服务（`internal/gitidentity`），以原沙箱行为规格做对比回归；不沿用 Node 运行时与 Electron（ADR-002/003）。
- 与现有设计匹配：IdentityProfile、LocalBinding、EffectiveContext（business-concepts）；GitIdentity 服务边界（service-pool）；产品不变量 1–3；ADR-016/017/018 语义随迁移保留。

## 来源能力基线

| 能力 | 来源命令 | 迁移语义 |
|---|---|---|
| 身份档案 | add/list/remove/import | id + name/email/signingkey/gpgsign/gpg.format + 自定义 kv + insteadOf 重写 + host=用户名账号选择器；token 不入档案 |
| 应用身份 | use/local [--local] [--repos] [--dry-run] | 全量覆盖：写入受管键并 unset 档案缺失键（含 gpg.format）；dry-run 与真跑同算法只算不写 |
| 生效身份 | current [--format prompt] | 每键来源解析：仓库本地 → includeIf 目录绑定 → 全局；git 自身是事实源 |
| 目录绑定 | bind/unbind --dir | 托管片段 + ~/.gitconfig 单条 include 钩子；Windows 用 gitdir/i 且按真实路径匹配；目录可不存在；unbind 回收片段并检测手改漂移 |
| 清除身份 | unset [--local\|--global] | 清身份键与派生键，本地默认 |
| 仓库扫描 | scan [--depth] [--save] [--format] | 只读审计各仓库身份（✓/✗/⚠/◆）与镜像（⊕）；--save 存默认扫描目录 |
| 扫描收编 | import --scan | 把 ⚠ 手配身份收编为档案，显式预览+确认 |
| 提交审计 | audit [--expect] [--strict] | 作者分布、污染标注、mailmap 建议（不写 .mailmap）；--strict 有污染退出码 2 |
| 环境体检 | doctor | git 版本、凭据助手、insteadOf 冲突、绑定健康（钩子/片段缺失/孤儿/失效目录/嵌套/漂移），只读不修复 |
| 远程镜像 | remote list/mirror/unmirror | 仓库级多 pushurl；fetch 与凭据通道不受影响 |
| 凭据管道 | credential list/login/set/remove | token 经 git credential 协议直达 helper；不留存、不回显、不进日志 |
| 档案迁移 | （新能力） | gitid config.json 预览/备份/幂等导入，沿用 legacy 迁移模式 |

## 已确认规则

1. Git 实际配置由 git 自身解析；产品只在显式操作时经 `git config` 写入，读回显与生效判断以 git 输出为准（产品不变量 1）。
2. 应用身份是全量覆盖：档案缺的受管键按托管规则清除，不残留前一身份；重写与账号选择器随 use 一并应用，切换身份自动清理（不变量 2、ADR-016/017）。
3. 凭据只经 git credential 协议直通 helper；服务不留存、不回显、不广播 token（不变量 3）。GUI 侧推荐 helper 自身交互（如 GCM 弹窗）。
4. 目录绑定是显式用户动作；写入用户真实 ~/.gitconfig 的 include 钩子与 DevToolbox 托管片段（产品自有目录），回收前检测漂移并如实警告。
5. 扫描与体检只读；doctor 不自动修复；audit 不写 .mailmap；strict 失败是明确退出状态而非修复动作。
6. 数据目录归 DevToolbox 自有命名空间；gitid 旧 config.json 只读探测，导入前预览+备份、按来源摘要幂等。
7. 跨仓库批量操作逐步返回每仓结果，允许部分失败并如实汇总（不伪装原子事务）。

## 暂不纳入

后台钩子/守护进程、自动修复、shell 提示符集成（归会话/启动器模块）、联合体检聚合（归 diagnostics-and-compatibility 模块，本模块提供 Git 体检数据源）、`.mailmap` 写入、token 托管。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-10-28 | 按 gitid e33e728 行为和产品安全边界确定迁移切片 |
