# PRD — 项目上下文发现

版本：v1.0 | 日期：2026-10-27 | 模块：project-context

## 1. 目标与成功标准

开发者能在 DevToolbox 中登记本机工作集合，并可靠查看其仓库、构建模块和运行时版本声明。成功标准：每个已展示的声明都有来源；无法确定的信息显式为 unknown/冲突；扫描不会执行项目代码或修改被扫描目录。

## 2. 用户与场景

唯一角色为本机开发者。核心场景：登记项目、理解多仓/多模块结构、发现版本要求、重扫检查变化、移除登记。

## 3. 范围与排除

包含静态发现：Git 工作树、Node、Python、Java、Go 及 Maven/Gradle/Android SDK 相关声明。支持识别已有 Wrapper 文件并展示其存在，不执行 Wrapper。

排除运行时安装/下载/切换、任意脚本执行、Git 身份写入、旧应用数据导入、自动修复、容器/数据库生命周期管理。

## 4. 领域模型

| 对象 | 关键字段 | 关系 |
|---|---|---|
| Project | id、displayName、rootPath（本机）、createdAt、lastScanAt | 聚合根，包含 0..N Repository、Module |
| Repository | id、canonicalPath、worktreeKind、discoverySource | 可被多个 Project 引用；路径本机化 |
| Module | id、relativePath、buildSystems、discoveryStatus | 隶属发现快照；可有多个 RuntimeRequirement |
| RuntimeRequirement | runtime、rawConstraint、normalizedConstraint?、sourcePath、confidence、status | 属于 Module；保留多来源声明 |
| ScanSnapshot | id、startedAt、completedAt、status、summary | 属于 Project；部分失败可保留已有快照并标明状态 |
| Evidence | sourcePath、locator、rawValue、parserId、observedAt | 随 Requirement 或发现节点保存 |

不保存本机安装选择、Git 凭据、token 或共享团队身份。跨项目路径身份策略与稳定 ID 格式在实现阶段锁定，但必须可迁移且不得将私有绝对路径写入共享项目清单。

## 5. 功能需求

1. 登记目录前先扫描并预览候选仓库/模块；用户明确确认后才保存登记。
2. 支持普通 Git 仓库、worktree 与嵌套仓库识别，不能只凭 `.git` 目录判断。
3. 模块发现依赖已知构建清单/配置；一目录可含多个构建系统，不能强制唯一 runtime。
4. 版本声明需保留原文与来源；支持 package.json engines、`.nvmrc`、Python 项目配置、pom.xml、Gradle toolchain/Wrapper、Go module 声明等已确认来源；无声明返回 unknown。
5. 严格约束、模糊约束、多个来源冲突分别呈现，不将 `^3.11` 等约束简化为精确版本。
6. 扫描期间支持取消；取消/权限错误/格式错误产生部分结果和明确状态，不覆盖为完整成功。
7. 重扫可比较证据变化；移除登记不删除任何项目文件，也不撤销 Git 或系统设置。
8. UI 提供来源路径、扫描时间和未知/冲突说明。

## 6. 业务规则与验收

- 只读：不写扫描根；不运行 scripts、hooks、Wrapper 或项目命令；不读取 `.env`、密钥文件、依赖/构建目录。
- 路径：规范化后限制在授权根内；链接越界不跟随；worktree 元数据按安全规则解析。
- 证据：解析结果可追溯到文件与声明位置；解析器不认识的语法保留原文并标 unknown。
- 多根：Project 与 Repository/Module 分别显示，嵌套关系可表达。
- 隔离：项目登记数据存于 DevToolbox 自有数据目录；共享清单不保存绝对路径。
- 登记/移除幂等；重复确认同一根目录不创建重复项目。

## 7. 非功能要求

扫描可取消；大目录需要忽略常见依赖/产物目录并提供扫描摘要。单个文件读取失败不能导致全项目失败。错误不包含凭据内容。测试使用独立 HOME/USERPROFILE/APPDATA/XDG_CONFIG_HOME 与 fixture，不访问真实用户 PATH、注册表、shell profile 或 Git 配置。

## 8. 领域模型分类

- 聚合根：Project、ScanSnapshot。
- 实体：Repository、Module。
- 值对象：RuntimeRequirement、Evidence、ScanIssue。
- 外部事实：文件内容、Git 元数据；只读适配器读取。
- 领域事件：本模块暂不对外发布事件；操作结果由应用服务直接返回。

## 9. 更新记录

| 版本 | 日期 | 类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-10-27 | initial | 项目上下文发现 PRD |
