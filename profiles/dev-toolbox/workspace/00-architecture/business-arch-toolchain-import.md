# 业务架构 — uv/rustup 外部工具链检测与登记

版本：v1.0 | 日期：2026-11-03 | 模块：toolchain-import

## 1. 业务域与活动

归属支撑业务域（运行时管理与项目上下文的只读扩展）。用户把 rustup 管理的 Rust 工具链与 uv 自管的 CPython 纳入 DevToolbox 清单，并让 Rust 项目被项目上下文识别。

| L3 关键业务 | L4 业务活动 | 输入 | 输出 | 边界 |
|---|---|---|---|---|
| 外部工具链发现 | 扫描 rustup 工具链 | RUSTUP_HOME/默认目录 | 工具链候选（版本/通道/默认标记） | 只读；不执行 rustup/rustc |
| 外部工具链发现 | 扫描 uv 解释器 | UV_PYTHON_INSTALL_DIR/默认目录 | python 候选（source=uv） | 只读；不执行 uv/python |
| 外部工具链登记 | 确认登记候选 | 用户勾选的候选 | RuntimeCatalog 条目 | 显式确认；不改来源目录 |
| 项目 Rust 识别 | 识别 cargo/rust 清单 | Cargo.toml、rust-toolchain(.toml) | 构建模块与 rust 版本要求 | 只读解析；不运行 cargo |
| 全局默认 | 设置 rust 默认 | 登记的工具链 | GlobalRuntimeSelection(rust) | 与七工具同语义 |

## 2. 业务规则

- 发现与登记不改变 rustup/uv 的任何状态；两工具对同一安装的"所有权"完整保留。
- 版本证据不足时呈现通道名与 unknown，不推断具体版本。
- rust 项目版本要求保留来源文件与约束原文（rust-version、channel），置信度沿用现有三级。
- 旧用户状态文件不含 rust 键时读取自动补齐，无需迁移。

## 3. 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-11-03 | 初始业务架构 |
