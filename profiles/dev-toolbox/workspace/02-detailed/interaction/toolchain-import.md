# PageSpec — uv/rustup 外部工具链检测与登记

版本：v1.0 | 日期：2026-11-03 | 模块：toolchain-import | 状态：Gate 1 待检查

## 入口与页面

无新页面。改动集中在「用户级运行时设置」与项目详情：

## 用户级运行时设置

- 「发现与登记」：工具选择器加入 `rust`；扫描结果新增 rustup/uv 候选行——显示版本或通道（stable/nightly）、来源标签（rustup/uv）、rustup 默认工具链标记（候选行 evidence 含"default toolchain"时显示 `默认` tag）；登记链路复用（勾选→确认弹窗→登记所选）。
- 「全局默认」：七工具循环列表扩展为含 rust 的八行；未登记时显示"未选择"，与现有一致。
- 来源标签沿用 availability/confidence 的既有展示；uv 候选与 PATH 发现的 python 候选同时出现时按 Home 去重、来源各自可见。

## 项目详情

- 构建模块卡片：buildSystems 出现 `cargo`/`rust` tag。
- 要求列表：`rust: >=1.75`（Cargo.toml rust-version）或 `rust: stable`（rust-toolchain 通道），来源文件与置信度同现有样式。

## 状态

覆盖：无 rustup/uv 安装（扫描自然为空，无错误）、目录存在但布局不可解析（候选 confidence=unknown 仍可登记）、通道工具链（版本 unknown 但通道可读）。不出现"建议安装/升级"类引导——所有权在 rustup/uv。

## API 对齐

零新增操作，字段/枚举与 api-toolchain-import.md 一致；演示 mock 增加 rust 候选与 cargo 模块样例。

## 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-11-03 | initial | 既有页面的枚举级扩展 |
