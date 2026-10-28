# D0 需求澄清 — DevSwitch 用户级运行时设置迁移

日期：2026-10-27。归属模块：`runtime-migration`。状态：D0 补充输入；完整七工具 D1-D4 与 Gate 1 尚未完成。

## 目标与用户

DevToolbox 除项目级运行时要求外，必须保留用户在 DevSwitch 中维护的运行时用户状态。开发者迁移到新工具后，仍能看到已登记的运行时实例和每种工具的全局默认选择；项目或会话级临时选择不会悄悄替换这些默认值。

## 来源证据

只依据 DevSwitch 源码模型与存储契约，不读取真实用户目录：`internal/models/models.go` 定义 `State{schema,current,runtimes}`；`internal/store/store.go` 将其读写到 `state.json`；`internal/paths/paths.go` 定义用户配置目录。迁移基线见 `migration-baseline-2026-10-27.md`，来源 HEAD 为 `5e3fc4ada6725f0ae552994c3262b53ccade5730`，工作区干净。

## 用户级状态范围

- `current`：node、java、maven、gradle、androidsdk、golang、python 各自的全局当前实例路径；空值表示未选择。
- `runtimes`：已发现/导入/本工具下载实例，含 tool、version、major、home、binary、source，以及可选 label/vendor。
- 已下载运行时目录本身作为用户数据保留原位；状态导入只关联路径和元数据，不复制、移动或删除二进制。
- 手动导入来源与 label/vendor 在重扫后需保留；实例暂时不可访问时记录 unavailable，不能因扫描失败而删除用户选择。
- `DEVSWITCH_MIRROR` 与 CLI `--mirror` 是下载选择，不属于 `state.json` 持久用户设置，不伪造导入值。
- 用户 shell hooks、PATH shim、环境变量属于派生生效物；本次用户状态导入不安装、移除或改写它们，后续兼容阶段单独处理。

## 作用域规则

1. 用户级默认值是全局回退，不代表项目已应用某运行时。
2. 项目绑定/共享约束和会话覆盖各自独立；项目/会话求值不得改写用户级 `current`。
3. 只有明确的用户操作 `setGlobalDefault` 能改用户级当前选择；需要确认的安装/下载是单独动作。
4. Git 身份配置继续由 Git 自身解析，与运行时全局选择分开。

## 迁移流程与安全

读取旧 `state.json` 前只展示来源路径与可读取状态；用户明确发起后预览工具、选择、运行时条目及缺失路径。执行前在新数据目录创建带时间/来源 schema 标记的备份；导入以实例身份和源文件摘要幂等，不对旧状态写回，不在旧新应用间双写。遇到未知 schema、无效条目或路径冲突时保留原始条目并呈现问题，不自动清理或猜测。

默认不读取或导入凭据。迁移报告不得包含真实 token、用户名凭据或文件正文。来源数据中的绝对路径属于本机私有状态，不进入共享项目文件、日志或广播事件。

## 待 D1-D4 完成的决策

- 新产品 UserRuntimeState 的版本化 schema、稳定实例 ID 与同一路径重复条目规则。
- 缺失/重复/冲突全局选择的恢复 UI 与优先级。
- 多个 DevSwitch 安装/状态目录的发现与用户选择。
- 状态快照回滚/重试边界，以及迁移成功标记存储位置。
- 七种工具的扫描器、provider、切换副作用及旧 shell 生效物的逐项行为映射。
