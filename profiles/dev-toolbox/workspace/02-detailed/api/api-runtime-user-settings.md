# Go 应用服务契约草案 — 用户级运行时状态迁移

版本：v1.0 | 日期：2026-10-27 | 归属：user-runtime-state-migration | 状态：本切片 Gate 1 PASS

> 本机工具操作契约；CLI/GUI 调用共用 Go 服务，不使用 HTTP/SQL。此草案仅定义用户状态迁移与全局选择边界，不是七工具迁移模块完整 API。

## 状态模型

- `UserRuntimeState`: schema、revision、globalSelections、runtimeCatalog、legacyImports。
- `GlobalRuntimeSelection`: runtime → installationId?、source（user/legacy）、status（active/unresolved/empty）。
- `RuntimeInstallation`: id、runtime、version、major、localHome、binaryPath?、source、label?、vendor?、availability。
- `LegacyImportRecord`: sourceKind、sourceDigest、sourceSchema、backupPath、importedAt、result。
- `localHome`、`binaryPath`、`backupPath` 仅本机本地存储/API 返回；不写共享清单、日志或通用事件。

## 操作草案

### `legacy.inspectDevSwitch`

- 请求：无；按支持的平台用户配置路径探测已知 DevSwitch 状态文件。
- 响应：候选来源（路径、可读性、schema、文件摘要）；不返回凭据或任意文件正文。
- 作用域/副作用：只读；不自动加载状态进入当前运行时决策。

### `legacy.previewDevSwitchImport`

- 请求：`sourceId`。
- 响应：工具级 global selection 映射、runtime 条目映射、缺失/重复/未支持记录、备份目标与预期冲突；不包含文件正文。
- 幂等性：无写操作。

### `legacy.importDevSwitch`

- 请求：`sourceId`、`expectedSourceDigest`、`confirmBackup=true`、`confirmImport=true`。
- 响应：导入记录、创建/复用/未解析计数、备份摘要。
- 副作用：先在新产品用户数据区备份源文件，再将映射写入新产品用户状态；任何失败不得写回旧文件。digest 不匹配需重新预览。
- 幂等：已存在相同 sourceDigest 的记录时返回既有结果，不覆盖导入后的用户更改。

### `runtime.getUserState` / `runtime.setGlobalDefault`

- `getUserState` 返回用户级全局选择与安装清单。
- `setGlobalDefault` 请求 `runtime`、`installationId`、`expectedRevision`、`confirm=true`；必须验证实例属于该 runtime 且可用，成功后只更新用户状态，不改 Project/Session 绑定。

## scope 约束

运行时求值顺序与完整 `EffectiveContext` 留待 runtime-migration D3 定稿。已确认不变量：会话级/项目级选择不会调用 `setGlobalDefault`；操作终端、shell hook 或系统环境属于独立的明确动作。

## 本切片范围外

七种工具 catalog 生命周期、手动扫描合并、下载 provider、版本匹配、切换生效物，以及与 Git 身份/会话环境的完整接口对齐属于 `runtime-migration` 后续设计，不由本切片 Gate 1 覆盖。
