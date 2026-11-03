# PRD — uv/rustup 外部工具链检测与登记

版本：v1.0 | 日期：2026-11-03 | 模块：toolchain-import | 状态：Gate 1 待检查

## 1. 目标

rustup 管理的 Rust 工具链与 uv 自管 CPython 能被 DevToolbox 发现并显式登记进用户级清单；Rust 项目被项目上下文识别并提取 rust 版本要求。全程只读，不触碰 rustup/uv 的所有权。

## 2. 领域对象（扩展）

| 对象 | 扩展点 |
|---|---|
| RuntimeInstallation | tool 增 `rust`；source 增 `rustup`、`uv` |
| Candidate | rust 候选：版本=目录名 semver，否则通道（stable/nightly/…）+version=unknown；uv 候选：tool=python、source=uv、版本取目录名 cpython-3.x.y |
| RuntimeRequirement | runtime 增 `rust`；来源 Cargo.toml rust-version（confidence=exact/medium）与 rust-toolchain channel（exact） |
| Module | buildSystems 增 `cargo`、`rust` |

## 3. 功能需求

| ID | 需求 | 验收标准 |
|---|---|---|
| TH-01 | rustup 检测 | 枚举 RUSTUP_HOME（默认 ~/.rustup）toolchains/* 为 rust 候选；目录名含 semver 则版本 exact，channel 名则 unknown+通道；默认工具链（settings.toml）候选带标记 |
| TH-02 | uv 检测 | 枚举 UV_PYTHON_INSTALL_DIR（默认按 uv 文档规则）下 cpython-* 目录为 python 候选（source=uv），版本解析自目录名 |
| TH-03 | 不执行被检程序 | 检测全程零进程执行；版本/布局仅来自文件读取 |
| TH-04 | 所有权不变 | 登记与设默认不写 ~/.rustup、uv 目录；无任何 update/install 触发 |
| TH-05 | rust 登记 | rust 候选可经既有确认链登记；installationID 按 Home 去重；旧状态文件读取自动补齐 rust 选择键 |
| TH-06 | rust 全局默认 | 「全局默认」含 rust 行，设默认语义与七工具一致 |
| TH-07 | Cargo 项目识别 | Cargo.toml 目录成为构建模块（cargo）；rust-version 提取为 rust 要求，保留原文与来源 |
| TH-08 | 通道钉子识别 | rust-toolchain/rust-toolchain.toml 解析 [toolchain].channel，生成 rust 要求（channel 原文，confidence=exact），目录若无 Cargo.toml 也作为模块（buildSystems=rust） |
| TH-09 | 来源可见 | 候选与清单条目显示来源（rustup/uv/path/downloaded…）；前端工具清单出现 rust |
| TH-10 | 布局验证回填 | 实现期在真实 Windows/Linux 验证 uv/rustup 目录布局与 settings.toml 格式，结论回填 D0 表；验证前不得声称覆盖全部布局变体 |

## 4. 状态与流转

无新状态机：候选→确认登记→清单条目→可选设默认，复用既有链路。

## 5. 安全与可靠性

路径解析限制在上述根目录内（不跟随符号链接逃逸）；目录不存在/不可读时安静跳过并记 Issue；解析失败候选不入结果。TOML 解析用受限解析（仅取所需键），坏文件不中断扫描。

## 6. 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-11-03 | initial | uv/rustup 只读检测导入与 Rust 项目识别 |
