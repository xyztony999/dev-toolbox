# PageSpec — 项目上下文

版本：v1.0 | 日期：2026-10-27 | 用户类型：本机开发者 | 模块：project-context

> 页面通过白名单 WebView 桥接调用 Go 服务。无 HTTP 后端。所有字段与 `api-project-context.md` 中 DTO 对齐。

## 1. 页面与路由

| 路由 | 页面 | 数据源 |
|---|---|---|
| `/projects` | 项目列表 | `project.list` |
| `/projects/register` | 登记预览 | `project.scanPreview` |
| `/projects/:projectId` | 项目详情 | `project.get` |

## 2. Action / 桥接操作

| Action | 操作 | 请求字段 | 响应/反馈 |
|---|---|---|---|
| scanPreview | `project.scanPreview` | rootPath | 候选树、requirements、issues、状态 |
| register | `project.register` | candidateId、displayName、rootPath、expectedEvidenceDigest | 登记成功/重复登记提示 |
| list | `project.list` | cursor?、limit? | ProjectSummary[] |
| detail | `project.get` | projectId | 仓库/模块/要求/证据详情 |
| rescan | `project.rescan` | projectId、expectedRevision | 进度、差异、扫描状态 |
| unregister | `project.unregister` | projectId、expectedRevision、confirm | 移除结果 |
| cancel | 桥接取消当前 requestId | requestId | cancelled 结果与部分扫描摘要 |

## 3. 项目列表页

- 列：项目名称、登记目录、仓库数、模块数、最近扫描、扫描状态。
- 空状态：说明登记目录后可查看项目结构；主操作“添加项目”。
- 加载/错误：可重试；存储损坏时显示恢复/导出诊断入口，不自动重置数据。
- 用户动作：添加、打开详情、刷新列表、移除（确认弹窗说明只移除登记）。

## 4. 登记预览页

- 输入：目录选择器；允许手动路径输入。
- 扫描中：进度条显示已检查文件/候选数量，提供取消；重复点击禁用。
- 预览树：仓库、模块、构建系统、运行时声明；每项展示来源文件与置信状态。
- 冲突：同一模块多来源约束并列显示，不提供“自动选一个”。
- 无版本声明：展示“未发现版本要求（unknown）”，不得显示 latest。
- 权限/越界/不支持：按 ScanIssue 显示相对路径和可恢复说明。
- 确认登记：用户编辑展示名称后提交；若证据 digest 变化，提示重新预览，不静默登记。

## 5. 项目详情页

- 顶部摘要：名称、项目根、最后扫描时间、整体状态。
- 仓库区：路径、普通仓库/worktree/嵌套类型及识别来源。
- 模块区：相对路径、构建系统、声明列表。
- 版本声明区：运行时、原始约束、可选规范化结果、来源、置信度与状态。
- 刷新：调用 rescan；期间显示进度、取消；完成后列出新增/变化/移除证据。
- 移除：二次确认只删除 DevToolbox 登记数据。

## 6. 状态、键盘与安全

- 状态需覆盖 empty/loading/ready/partial/cancelled/failed/conflict/permission-denied。
- 列表/树可键盘导航，焦点顺序稳定；小窗口下详情改为分区折叠。
- 不显示或广播文件正文、token、Git 凭据；界面中的路径展示本机项目路径，仅限本机窗口。
- Mock 与真实桥接采用相同字段、错误码、requestId 和取消语义。

## 7. 一致性检查

Page Action 对应 API 操作；列表字段来自 ProjectSummary；详情字段来自 RepositorySummary、ModuleSummary、RuntimeRequirement、Evidence、ScanSnapshot；所有异步结果用 requestId 关联。

## 8. 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-10-27 | 初始项目上下文页面规格 |
