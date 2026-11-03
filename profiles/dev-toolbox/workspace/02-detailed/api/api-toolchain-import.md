# Go 应用服务契约 — uv/rustup 外部工具链检测与登记

版本：v1.0 | 日期：2026-11-03 | 模块：toolchain-import | 状态：Gate 1 待检查

## 契约原则：零新增操作

本模块不新增任何桥接/CLI 操作。全部能力经既有白名单操作交付，契约变化仅为**枚举与字段取值扩展**：

| 既有操作 | 变化 |
|---|---|
| `runtime.discover` | 默认与显式 `tools=["rust"]` 均触发 RustupScanner/UvPythonScanner；返回候选含 tool=rust（source=rustup）与 tool=python（source=uv） |
| `runtime.inspectPath` | tool=rust 支持：校验 bin/rustc（Windows 为 rustc.exe）与 bin/cargo 存在，版本按目录名解析 |
| `runtime.register` / `runtime.setGlobalDefault` | tool 取值增 `rust`；登记/设默认语义不变 |
| `runtime.getUserState` / `git.*` 等 | 不变 |
| `project.scanPreview` / `project.rescan` | 模块 buildSystems 增 `cargo`/`rust`；requirements.runtime 增 `rust`（rawConstraint=rust-version 原文或 channel 名） |

CLI 对应：无新命令；`runtime discover`/`inspect`/`register`/`set-default` 接受 `rust`。

## 枚举扩展

| enum | code | label |
|---|---|---|
| RuntimeTool | rust | Rust 工具链 |
| RuntimeCandidateSource | rustup | rustup 管理 |
| RuntimeCandidateSource | uv | uv 自管解释器 |
| ProjectBuildSystem | cargo | Cargo |
| ProjectBuildSystem | rust | rust-toolchain 钉子 |

Candidate.Vendor：rustup 候选 `rust-lang`；uv 候选 `astral-sh/python-build-standalone`（按目录名事实，实现期核实）。rustup 默认工具链以 Evidence 追加 `default toolchain` 标记。

## 错误码

无新增。既有 `RUNTIME_INVALID_TOOL` 等沿用；目录不可读经 Result.Issues 表达。

## 兼容性

userstate toolNames 增 `rust` 后：新状态直接含键；旧状态由 readState 的 NewState 合并自动补齐（现状已实现，无迁移）。前端硬编码工具清单处统一加入 rust（运行时设置两个标签页、演示 mock）。

## 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-11-03 | initial | 零新增操作的枚举级契约扩展 |
