# API / 应用服务契约 — 项目上下文发现

版本：v1.0 | 日期：2026-10-27 | 变更类型：initial | 模块：project-context

> 本模块是本机工具，不采用 HTTP/CRUD/数据库表契约。CLI 与 WebView 白名单桥接均调用同一 Go 应用服务；下表定义稳定操作契约，具体 Go 类型名可在实现中调整但字段语义必须保持。

## 1. 模块概述

### 1.1 目标与边界

提供项目登记预览、只读扫描、项目/仓库/模块查询、重扫和移除登记。输入事实来自本地文件系统与 Git 元数据；不安装工具、不执行项目代码、不读取 Git 用户身份配置、不改系统环境。

### 1.2 决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| 调用形态 | Go 应用服务操作，不建立 HTTP 服务 | CLI/GUI 共用业务核心 |
| 登记路径 | `os.UserConfigDir()/DevToolbox/projects.json`，原子替换 | 用户级私有数据，与共享项目文件分离 |
| 登记目录存储 | 绝对路径仅写入本机配置 | 不进入团队共享要求或日志 |
| 扫描 | 显式允许清单 + 忽略目录策略，只读 | 避免运行不可信项目代码 |
| 仓库识别 | Git 元数据只读解析，支持 `.git` 文件/worktree | 不假设 `.git` 必为目录，也不运行 hooks |
| 冲突 | 保留多个 Evidence 并标记 conflict | 不擅自选择单一约束 |
| 部分错误 | 返回部分结果 + issues | 单个不可读文件不丢弃整次扫描 |

### 1.3 领域对象

对象与 D2 保持一致：Project、Repository、Module、RuntimeRequirement、Evidence、ScanSnapshot、ScanIssue。无需 SQL 表；持久化格式为版本化 JSON 文件，升级采用读旧版→校验→临时文件→原子替换。文件写失败不覆盖最后一次有效数据。

## 2. 操作契约

所有请求带 `requestId`（调用关联 ID）；路径操作需显式 `scope`。响应带同一 `requestId`。公共错误不得包含清单敏感内容或凭据。

### 2.1 `project.scanPreview`

- 请求：`rootPath: string`。
- 响应：`candidate: ProjectCandidate`，含规范化根路径、仓库候选、模块候选、要求证据、issues、scanStatus。
- 作用域：只读指定目录；路径规范化后所有读取必须限制在 root 下。
- 幂等：相同文件快照产生相同候选语义。
- 取消：接收 `context.Context`；取消返回部分候选与 `cancelled` 状态。
- 副作用：无持久化副作用。

### 2.2 `project.register`

- 请求：`candidateId`、`displayName`、`rootPath`、`expectedEvidenceDigest`。
- 响应：`ProjectSummary`、`created: bool`。
- 作用域：本机登记库。
- 幂等：规范化根路径唯一；重复请求返回既有项目，`created=false`。
- 并发/失败：文件锁或进程内互斥 + 原子替换；摘要不匹配返回 `PROJECT_SCAN_STALE`，不登记过期结果。
- 副作用：仅写 DevToolbox 自有配置目录。

### 2.3 `project.list` / `project.get`

- `list` 返回 `ProjectSummary[]`；分页留待登记量级需要时增加。
- `get` 请求 `projectId`；响应为 Project、Repository、Module、Requirement、Evidence 摘要和最新 ScanSnapshot。
- 只读；不存在返回 `PROJECT_NOT_FOUND`。

### 2.4 `project.rescan`

- 请求：`projectId`、`expectedRevision`。
- 响应：`ScanSnapshot`，包括状态、开始/结束时间、发现差异摘要、issues 与部分结果标记。
- 作用域：先验证根路径仍可访问；只读用户项目；持久化仅更新自有登记库中的发现快照。
- 幂等：同一项目同时只允许一个有效扫描；重复请求返回 `PROJECT_SCAN_IN_PROGRESS`。
- 取消/部分失败：取消与权限/解析错误作为 issue 保留；成功证据增量写入一个完整原子快照，失败不丢失上次快照。

### 2.5 `project.unregister`

- 请求：`projectId`、`expectedRevision`、`confirm=true`。
- 响应：`removed: bool`。
- 作用域：仅本机登记数据。
- 幂等：重复移除返回 `removed=false`。
- 明确不删除：项目目录、Git 配置、运行时、本机绑定或旧应用数据。

### 2.6 `project.scanProgress` 事件

仅扫描操作产生事件：`requestId`、`projectId?`、`phase`、`visitedCount`、`candidateCount`、`cancelable`。事件不包含文件正文、凭据、身份信息或绝对路径；GUI 可用 requestId 关联，CLI 可渲染进度。

## 3. DTO 字段

| 类型 | 字段 |
|---|---|
| ProjectSummary | id、displayName、rootPath（仅本机界面/API）、repositoryCount、moduleCount、lastScanAt、scanStatus、revision |
| RepositorySummary | id、path、kind（normal/worktree/nested/unknown）、source |
| ModuleSummary | id、relativePath、buildSystems[]、requirements[]、status |
| RuntimeRequirement | runtime、rawConstraint、normalizedConstraint?、sourcePath（相对 project root）、confidence、status（resolved/unknown/conflict/unsupported） |
| Evidence | sourcePath、locator、rawValue、parserId、observedAt |
| ScanIssue | code、severity、relativePath?、messageKey、recoverable |
| ScanSnapshot | id、status（complete/partial/cancelled/failed）、startedAt、completedAt?、revision、changes、issues[] |

CLI 可将本机绝对路径输出给本机用户；GUI 通用事件和日志只显示相对路径/展示名。凭据不属于任何 DTO。

## 4. 错误目录

`PROJECT_INVALID_PATH`、`PROJECT_PATH_OUTSIDE_ROOT`、`PROJECT_PERMISSION_DENIED`、`PROJECT_NOT_FOUND`、`PROJECT_ALREADY_EXISTS`、`PROJECT_SCAN_STALE`、`PROJECT_SCAN_IN_PROGRESS`、`PROJECT_SCAN_CANCELLED`、`PROJECT_STORE_UNAVAILABLE`、`PROJECT_STORE_CORRUPT`、`PROJECT_REVISION_CONFLICT`、`PROJECT_INVALID_REQUEST`。

错误响应包含 `code`、本地化 `messageKey`、`requestId`、可选 `field`；不回显文件正文或系统敏感值。

## 5. CLI 映射

CLI 命令调用相同应用服务：`dev-toolbox project scan <path>`（预览）、`project add <path>`（预览后确认登记）、`project list`、`project show <id>`、`project rescan <id>`、`project remove <id>`（二次确认）。退出码：0 成功；2 参数错误；3 用户取消；4 扫描部分失败；5 存储/内部错误。命令实现阶段需复核既有命令冲突，安装器不得覆盖来源工具命令。

## 6. 跨聚合边界

Project 包含登记身份与最新扫描快照引用；Repository/Module/Requirement 是扫描产物，不以独立写接口修改。扫描先在内存构造快照，通过校验后原子保存。跨项目重复仓库可引用同一 canonical repository 事实，但各项目保留自己的相对路径关系。

## 7. SQL / 事件清单

- SQL/DDL：不适用，使用本机版本化 JSON 存储。
- 领域事件/MQ：不适用；进度事件是进程内 UI 通知，不对外广播。

## 8. 更新记录

| 版本 | 日期 | 类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-10-27 | initial | 定义项目上下文 Go 应用服务契约 |
