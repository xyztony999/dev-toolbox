# 功能清单 — uv/rustup 外部工具链检测与登记

版本：v1.0 | 日期：2026-11-03 | 用户类型：本机开发者

## 发现与登记

- 「发现本机运行时」扫描自动包含 rustup 工具链与 uv 解释器；候选行显示来源标签（rustup/uv）、版本或通道名、rustup 默认工具链标记。
- 勾选候选 → 确认登记（既有链路）；rust 条目进入用户级清单，可与 PATH 发现的安装并存。
- 「全局默认」出现 rust 行，可显式设默认（语义同七工具）。

## 项目上下文

- 含 Cargo.toml 的目录被识别为构建模块（构建系统 cargo）；rust-toolchain(.toml) 识别为 rust 通道钉子。
- 模块要求列表展示 rust 版本要求：rust-version（Cargo.toml）或 channel（rust-toolchain），含来源文件与置信度。

## 本模块不提供

rustup/uv 的下载安装与升级、工具链组件管理、uv 二进制版本管理、项目 venv 管理、Node 多管理器（volta/mise 等）检测。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-11-03 | 初始功能清单 |
