# 应用架构 — 运行时下载与安装

版本：v1.0 | 日期：2026-10-28 | 模块：runtime-download-install

## 1. 服务边界

| 服务 | 职责 |
|---|---|
| RuntimeProviderCatalog | 给出固定 provider、受支持平台/架构、版本粒度和镜像 |
| RuntimeInstallPlanner | 查询 metadata、解析精确版本、计算目标并生成摘要 |
| RuntimeDownloader | HTTPS allowlist、重定向/DNS 策略、流式下载、限速/进度/取消、校验和 |
| SafeArchiveInstaller | staging 解包、路径/链接/资源限制、验证安装结构、原子归位 |
| RuntimeCatalogStore | 登记完成安装或记录 Android partial 状态；不改全局选择 |

## 2. 调用关系

CLI/WebView → 共用 Go 安装服务 → 固定 provider adapter → 网络与安全 staging adapter → DevToolbox 本机数据目录。前端只能提供工具、选择条件、镜像枚举和计划确认，不传 URL、文件名或目标路径。

## 3. 数据与副作用

安装清单扩展 `source=downloaded`、provider、已解析版本、平台/架构、校验摘要和目标路径。目标位于 Windows `%LOCALAPPDATA%/DevToolbox/runtimes` 或 Linux `$XDG_DATA_HOME/dev-toolbox/runtimes`（默认 `~/.local/share/dev-toolbox/runtimes`）。权限按用户私有目录设置。网络临时文件与 staging 限定于应用数据目录，可取消清理。

## 4. 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-10-28 | 初始应用架构 |
