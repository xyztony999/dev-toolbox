# 应用架构 — uv/rustup 外部工具链检测与登记

版本：v1.0 | 日期：2026-11-03 | 模块：toolchain-import

## 1. 服务边界

不新增写服务；对既有服务做只读扩展：

| 服务 | 扩展 |
|---|---|
| RuntimeCatalog.Discover | 新增 RustupScanner、UvPythonScanner 两个静态探测器：目录枚举 + 文件读取，产出 Candidate（tool=rust / tool=python,source=uv）；Inspect 支持 rust 布局校验（bin/rustc、bin/cargo 存在性） |
| userstate | toolNames 增补 `rust`；NewState 合并逻辑天然向前兼容旧状态 |
| ProjectDiscovery | isManifest 增补 Cargo.toml、rust-toolchain、rust-toolchain.toml；构建系统映射 cargo/rust；要求提取 rust-version 与 toolchain channel |

## 2. 调用关系

不变：CLI/WebView → runtime.discover/inspectPath/register、project.scanPreview 等既有白名单操作。两个探测器在 runtimecatalog 包内实现，无新桥接面。

## 3. 数据与副作用

- 读：RUSTUP_HOME/UV_PYTHON_INSTALL_DIR（或文档默认目录）下的目录与 settings.toml；项目内 Cargo.toml、rust-toolchain(.toml)。
- 写：仅既有用户级清单登记（显式确认后）；不写 ~/.rustup、uv 数据目录、项目文件。
- 环境变量覆盖优先于默认路径；路径不存在时安静返回空候选。

## 4. 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-11-03 | 初始应用架构 |
