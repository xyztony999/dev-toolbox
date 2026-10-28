# D0 需求澄清 — 运行时下载与安装

版本：v1.0 | 日期：2026-10-28 | 模块：runtime-download-install

## 匹配与范围

- 与现有 RuntimeCatalog 和 user-runtime-state 设计完全匹配：安装结果成为用户本机清单项；全局选择仍需单独明确设置。
- DevSwitch Go 来源在 `internal/downloader` 为七类工具提供下载/解压入口。本模块迁移这些 provider 能力，不照搬其删除既有目标目录、安装后自动切换、无取消、某些校验缺失等副作用。
- 用户：本机开发者；Windows/Linux；x64/arm64 仅支持 provider 明确提供的组合。

## Provider 能力基线

| 工具 | 来源能力 | 可选版本粒度 | 来源平台限制 |
|---|---|---|---|
| Node | 官方/npmmirror/清华镜像；选择大版本后解析当前可用补丁 | latest 或 major | Windows/Linux x64、arm64（以清单验证为准） |
| Java | Temurin/Adoptium | major 对应当前最新补丁 | Windows/Linux x64、arm64（以清单验证为准） |
| Maven | Apache 官方/清华镜像 | major 对应当前稳定版 | Windows/Linux；架构独立 |
| Gradle | 官方/腾讯镜像 | major 对应当前稳定版 | Windows/Linux；架构独立 |
| Android SDK | platform-tools 与 cmdline-tools 最新版 | latest | Windows/Linux x64；arm64 需 provider 核实 |
| Go | golang.google.cn 稳定版清单 | latest 或 major 对应当前补丁 | 来源实现仅 x64 |
| Python | python-build-standalone 镜像 | latest 或来源明确支持的版本前缀 | 来源实现仅 x64 |

此表只界定迁移来源能力。具体平台未被 provider 清单验证时返回 unsupported；不得回退到错误架构或假称任意历史精确版本可安装。

## 已确认规则

1. 安装前先查询并显示 provider、解析后的实际版本、平台/架构、镜像、目标目录、校验状态和磁盘空间估算；预览不落盘。
2. 下载/安装只由用户明确确认触发；安装不自动切换 GlobalRuntimeSelection，不改 PATH、注册表、shell profile、shim 或当前终端。
3. 只写 DevToolbox 自己的每用户运行时目录；同版本目标已存在时复用或报冲突，不删除/覆盖未知目录。
4. provider URL 由服务端固定模板生成，HTTPS host allowlist；拒绝内网/环回地址与不安全重定向。要求可验证的 provider SHA-256/SHA-512；没有校验值时不安装。
5. 压缩包在同一目标文件系统 staging 解压；拒绝路径穿越、绝对路径、链接成员、超限文件数/体积；验证完成后原子归位并写清单。
6. 下载支持进度与取消；取消清理 staging，已存在安装和用户选择不变。安装完成只登记来源=`downloaded`。
7. Android SDK 是多个 archive 的组合安装；预览列清 platform-tools/cmdline-tools 两个包，任一阶段失败时保留明确 partial 状态和可重试计划，不宣称原子事务。

## 暂不纳入

卸载、shim/PATH、shell hooks、终端会话覆盖、项目约束自动安装、Git 身份以及自动更新。以上需分别设计，不在下载确认中隐式发生。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-10-28 | 按 DevSwitch provider 行为和产品安全边界确定安装切片 |
