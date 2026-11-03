# D0 需求澄清 — uv/rustup 外部工具链检测与登记

版本：v1.0 | 日期：2026-11-03 | 模块：toolchain-import

## 背景与动机

用户确认的方向讨论结论：uv + rustup 检测导入是"现代项目必遇、传统工具不覆盖"的空白，纯只读实现，成本低（优先级第 1；会话启动器、.NET provider 后续另立项）。

现状空白：

- runtimecatalog 七工具表（node/java/maven/gradle/androidsdk/golang/python）无 rust；uv 自管的 CPython 与 rustup 管理的工具链对发现扫描不可见。
- 项目扫描 `isManifest` 不识别 `Cargo.toml`/`rust-toolchain.toml`——Rust 项目（盘点样本 deck-loop 为 Tauri 2）当前无构建模块与版本要求。
- merge-direction 将 Rust/Tauri 列为第三批"检测/导入"；本模块按用户决定提前实现其中只读检测部分。

## 能力基线

| 能力 | 来源事实 | 迁移语义 |
|---|---|---|
| rustup 工具链检测 | `$RUSTUP_HOME`（默认 `~/.rustup`）下 `toolchains/<name>` 目录 | 静态读目录与文件；不执行 rustup/rustc |
| 工具链版本/通道解析 | 目录名 `1.82.0-x86_64-pc-windows-msvc` 含语义化版本；`stable-*`/`nightly-*` 为通道名 | 优先目录名 semver；通道名解析为 channel、版本标 unknown |
| rustup 默认工具链 | `~/.rustup/settings.toml` 的 `default_toolchain` | 只读解析 TOML，用于候选标注 |
| uv 自管解释器检测 | `$UV_PYTHON_INSTALL_DIR`（默认按 uv 数据目录规则）下 `cpython-3.x.y+…-<target>` 目录 | 登记为 tool=python、source=uv 的候选 |
| 项目 Rust 识别 | `Cargo.toml`（构建清单）、`rust-toolchain`/`rust-toolchain.toml`（通道钉子） | 新增构建系统 cargo/rust；rust-version 要求提取 |
| 项目 uv 线索 | pyproject.toml 已被扫描；uv 专属标记不单独建模 | 不扩展（venv 与包管理器所有权归项目） |

## 已确认规则

1. 全程只读文件系统：不执行 rustup、uv、rustc、python 等任何被检测程序；检测不运行候选（沿用产品纪律"不运行候选程序"）。
2. 不夺取所有权：永不修改 `~/.rustup` 与 uv 数据目录；不触发 `rustup update`/`uv python install`；下载安装类能力不属于本模块（rustup 安装归 rustup）。
3. 新工具 id `rust` 加入工具池；userstate 的 `toolNames` 增补后，旧状态文件经既有 NewState 合并逻辑自动补齐 rust 键（向前兼容，无迁移脚本）。
4. uv 解释器登记为 `python`（source=uv），不新增 uv 工具 id；uv 二进制本身的版本管理不在本模块。
5. 目录解析尊重环境变量覆盖（RUSTUP_HOME、UV_PYTHON_INSTALL_DIR）；默认路径按 uv/rustup 官方文档规则，具体布局在实现时以真实机器验证并回填本表（B 阶段验证项）。
6. 版本置信度沿用既有框架：目录名 semver=exact；channel 名=unknown+通道标签；解析失败如实呈现，不猜 latest。
7. 登记仍是显式用户动作（复用 runtime.register 确认链）；登记后 rust 可设全局默认，语义与七工具一致。
8. rustup 与 uv 候选与 PATH 发现的同工具候选并存时按 Home 去重（既有 installationID 机制），来源各自保留。

## 暂不纳入

- volta/nvm-windows/fnm/mise 等 Node 多管理器检测；Bun/Deno provider；.NET SDK provider（后续独立立项）。
- uv 二进制自身登记为受管工具；uv venv/项目虚拟环境（与系统解释器分离，归项目上下文）。
- rustup 代理下载安装、工具链组件（target/toolchain）增删；`rustup override` 读取。
- 会话启动器、一键应用（第二批，另行设计）。

## 待实现期验证项（已验证，2026-11-03 真机 Windows）

- ✅ rustup：`~/.rustup/toolchains/<name>`（`stable-x86_64-pc-windows-msvc`）；`settings.toml` 顶层 `default_toolchain = "<name>"`（其后紧跟 `[overrides]` 表头，受限解析按"首个表头前"取顶层键成立）；`bin/` 含 rustc.exe/cargo.exe。
- ✅ uv：Windows python 目录为 `%APPDATA%\uv\python\cpython-3.13.5-windows-x86_64-none\`（python.exe 位于根）；`UV_PYTHON_INSTALL_DIR` 覆盖支持。
- ⚠️ Linux 侧默认路径（`~/.rustup`、`~/.local/share/uv/python`）按文档规则实现，未在真机 Linux 验证（CI 编译通过；路径错误时表现为安静无候选，无副作用）。
- ✅ deck-loop（真实 Tauri 项目）：`client/src-tauri` 被识别为 cargo 模块并生成 rust 要求（无 rust-version 时如实 unknown）。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-11-03 | 按方向讨论结论确定 uv/rustup 只读检测导入切片 |
