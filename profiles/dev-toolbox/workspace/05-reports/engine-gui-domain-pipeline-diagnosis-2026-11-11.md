# 排查报告 — 引擎 GUI 读不到领域视图 / 流水线停 /design

日期：2026-11-11 | 输入：用户报告（ai-project-engine 客户端） | 方法：GUI 同款代码对 dev-toolbox 直接复现（零猜测）

## 根因（三个独立因子）

### 1. 领域视图空：文档标题不满足 GUI 解析约定（v1.0/v2.0 均不满足，非重划引入）

GUI `domains.js` 降级态解析规则：**business-arch.md 中标题含"业务架构"的 H2 之下，H3+ 才入领域树**（H2 本身不入树）。

- 我们的 BA：H2 是"1. 产品业务域 / 2. 业务阶段 / 3. 关键业务与活动"——无一个含"业务架构"→ `inSection` 永为 false → archTree=[]（v1.0 就如此，领域视图从未显示过）。
- FL 树取全文 H3+：v2.0 重划后功能块是 H2 下的 bullet（v1.0 有 H3）→ funcTree=[]（v2.0 重划引入此回归）。
- level='none'（无 domain-model/data-model 富态源 + 两树皆空）。

### 2. 流水线停 /design：引擎 GUI 只看 `modules[0]`（引擎缺陷）

`pipeline.js` 的 `parsePipeline` 取 `data.modules[0]` 作为项目状态。我们 INDEX 的第一个模块是 **onboarding**（引导切片，stages 只有 requirement/validation，无 api/interaction/backend/frontend）→ 设计段永不 complete、后续段全 pending → current 永停 design。单模块项目无此问题；多模块（迁移切片型 INDEX）被第一个模块锁死。

### 3. boards 空为测试脚本假象（排除）

GUI server 每请求都 `docs.bindCore` + `domains.bindCore`，真实运行 boards 正常（复现确认 boards=1）。曾疑似根因，排除。

## 修复

### 引擎仓（D:/Project/ai-project-engine @4699dd6，仅提交我的两个文件，用户未提交的 core.js/doctor.test.js 未触碰）

- `engine/gui/lib/pipeline.js`：**多模块聚合**——只统计声明了该 stage/gate 的模块（缺 key 不拖累，引导切片天然不参与未声明段）；`n/a|not_applicable|skip` 视为不参与；`warn` 归一 in_progress；全 complete → complete。单模块项目行为不变。
- 新增 `engine/gui/test/pipeline.test.js`（6 用例）；domains/docs/funcview/doc-center（13）与 gui e2e（4）全绿。
- **插件缓存副本已同步**（0.5.1 payload）——用户当前 GUI 立即生效；注意插件重装会覆盖，需随引擎下一版发布。

### 项目仓（dev-toolbox）

- BA v2.0→"3. 业务架构——关键业务与活动"（命中 GUI 关键字）→ archTree=5 阶段。
- FL v2.1：功能块提升为 H3 → funcTree=10 功能块。
- INDEX：onboarding gates pending→**not_applicable**（引导记录本就无门控语义，同时不再拖累聚合视图）。

## 修复后状态（端到端复现验证）

- 领域视图：level=**fallback**，archTree=5 阶段、funcTree=10 功能块、boards=1。
- 流水线：current=design（**真实状态**——desktop-shell 的 D0-D4 未收尾使其 requirement/api/interaction=partial），而非旧的假卡死；gate 聚合正常。desktop-shell 设计收尾后 current 自然前进。

## 遗留说明

- 引擎仓提交未 push（含用户自己未提交的工作，推送时机由用户定）。
- GUI 领域视图每次快照实时读文件，无需重启引擎客户端即可看到项目侧文档修复。
