# 应用架构 — DevToolbox（全局）

版本：v2.0 | 日期：2026-11-11 | 状态：与 BA v2.0 联动重划

## 1. 应用域分层（DDD）

| 分层 | 应用域 | 聚合根/核心对象 | 包映射 | 归类理由 |
|---|---|---|---|---|
| **核心业务域** | 项目上下文 | Project（聚合 Repository/Module/RuntimeRequirement） | internal/project | 产品主链入口；有独立事实源（扫描证据）与状态（revision） |
| **核心业务域** | 环境应用与会话 | EffectiveContext / Session | internal/session | 产品差异化主线：把资产"应用"为可工作会话；深建模对象（求值、注入、终端适配） |
| 支撑业务域 | 运行时管理 | RuntimeInstallation / GlobalRuntimeSelection | internal/runtimecatalog, runtimeinstall, userstate | 为会话供给运行时资产；含发现/安装/默认/导入四组业务规则 |
| 支撑业务域 | Git 身份与仓库协作 | IdentityProfile（聚合 Binding/Credential 通道）/ RemoteInfo | internal/gitidentity | 为会话供给身份与协作配置；全量覆盖、绑定、审计等强业务规则 |
| 基础数据域 | ——（无） | —— | —— | 本产品无字典/组织类共享主数据；用户级清单归各支撑域自有（如实空缺，不虚构） |
| 平台支撑域 | 桌面宿主与桥接 | —— | internal/desktop | 白名单桥接/异步分发/WebView 加载/UI 注入，零业务语义 |
| 平台支撑域 | 遗留迁移工具集 | ImportRecord | 各域内的 legacy 入口（userstate/gitidentity） | 工具性能力；随来源增长，不承载核心模型 |

## 2. 关键架构决策

1. **模块 ≠ 领域边界**：INDEX 中的模块是**交付批次**（迁移切片），不是架构边界。例如 toolchain-import 的代码落在 runtimecatalog/external.go（运行时管理域），git-remote-management 落在 gitidentity/remote.go（Git 域）。新功能按领域归属包结构，不新建平行模块。
2. **两个核心域的关系**：项目上下文产出"要求"（Requirement），会话域消费"资产"（运行时安装+身份档案）产出"生效环境"。两域通过 userstate/gitidentity 的只读查询衔接，互不写对方状态——这是 ADR-006（共享要求与本机绑定分离）的应用层落点。
3. **会话域是唯一"应用"出口**：所有资产最终经 EffectiveContext 求值进入终端会话；未来的一键应用（ApplyPlan）也应落在此域，而不是散到各支撑域。
4. **userstate 的双重角色**：对运行时域是资产清单（installations+selections），对会话域是只读输入；其 legacy 导入职责属平台迁移工具集视角，代码位置不变。

## 3. 调用关系（运行时视角）

```
CLI ─────────────────┐
                     ├→ Go 应用服务（同一套）
WebView(白名单桥接) ─┤    ├─ 核心域：project（扫描/登记）  session（求值/启动）
                     │    ├─ 支撑域：runtimecatalog/install/userstate（只读供给+显式写）
                     │    │          gitidentity（档案/remote/凭据/体检，只读供给+显式写）
                     │    └─ 平台域：desktop（桥接/宿主） legacy 入口（迁移工具）
                     └─ 前端不执行 shell、不复制核心状态（WV-04）
```

## 4. 质量属性（全局）

- 所有外部进程（git/终端/provider 下载）参数数组执行、context 超时、显式 cwd/env；平台差异经 build tags 适配器隔离（GC-02/03）。
- 用户状态写入原子替换 + 文件锁；并发写经 revision/摘要冲突检测。
- 桥接操作白名单注册；超 1.5MB 内嵌包回退临时文件导航（WebView2 实测约束）。
- 前端错误通道 + 心跳标记常驻（frontend-error.log）。
- Windows/Linux 双平台 CI 门禁；GUI 构建与静态 CLI 分别验证（GC-08）。

## 5. 与模块文档的关系

各模块的 application-arch-*.md 保持为交付切片的详细视图；本文档为全局权威分层。冲突时以本文档为准并回改模块文档。

## 6. 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-10-27 | initial | 仅项目上下文域初稿 |
| v2.0 | 2026-11-11 | redivision | DDD 四层全局归类；确立"模块≠领域边界"与"会话为唯一应用出口"决策 |
