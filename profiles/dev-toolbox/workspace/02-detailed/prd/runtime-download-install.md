# PRD — 运行时下载与安装

版本：v1.0 | 日期：2026-10-28 | 模块：runtime-download-install | 状态：Gate 1 待检查

## 1. 目标

本机用户可从 DevToolbox 支持的 provider 预览并显式安装 Node、Java、Maven、Gradle、Android SDK、Go、Python。完成安装写入 DevToolbox 用户级 RuntimeCatalog，但不改变全局默认或系统环境。

## 2. Provider 与选择能力

| 工具 | 选择器 | provider 解析结果 | 来源镜像枚举 |
|---|---|---|---|
| Node | latest 或 major | major 下最新稳定补丁 | official / npmmirror / tuna |
| Java | major | 对应 major 最新 Temurin JDK | official / tuna |
| Maven | major | 对应 major 最新稳定版 | official / tuna |
| Gradle | major | 对应 major 最新稳定版 | official / tencent |
| Android SDK | latest | platform-tools + cmdline-tools 当前版 | official / tencent |
| Go | latest 或 major | 对应条件最新稳定版 | official-cn |
| Python | latest 或 provider 支持的版本前缀 | 对应 Python build-standalone 版本 | npmmirror |

只向用户展示来源实际可解析的粒度，不承诺任意精确历史补丁。目标平台为 Windows/Linux；架构由 provider metadata 明确判定，来源当前不支持 Go/Python arm64 时必须显示 unsupported。

## 3. 功能需求

| ID | 需求 | 验收标准 |
|---|---|---|
| INS-01 | 版本查询 | 显示 provider 返回的实际版本、provider、镜像和平台状态 |
| INS-02 | 安装预览 | 显示目标路径、包数、大小（可得时）、校验类型与目录冲突；无写入 |
| INS-03 | 用户确认 | 安装只接受尚未过期且摘要匹配的计划和显式确认 |
| INS-04 | 安全下载 | URL 由固定 provider 生成；HTTPS/host allowlist/重定向校验；拒绝私网解析与响应超限 |
| INS-05 | 完整性 | 安装前必须核验可信 SHA-256/SHA-512；无校验值或不匹配即失败 |
| INS-06 | 安全解包 | staging 限制路径穿越、绝对路径、软/硬链接、文件数与总解压大小；只允许预期布局 |
| INS-07 | 冲突保护 | 已有相同版本且内容一致则复用；目标不属于本产品或内容不同则停止，不覆盖 |
| INS-08 | 取消与进度 | 前端可取消；取消清理临时/未归位目录并报告最终状态 |
| INS-09 | 完成登记 | 只有安装结构验证和归位成功后加入 RuntimeCatalog；不设置全局默认 |
| INS-10 | 多包安装 | Android SDK 显示每阶段结果；部分失败保留可恢复状态和明确重试动作 |
| INS-11 | 隔离 | 仅写 DevToolbox 用户数据目录；不写 PATH/注册表/shell profile，不运行安装后脚本 |

## 4. 状态机

`idle → resolving → preview → confirmed → downloading → verifying → extracting → registering → complete`。查询失败进入 `failed`；用户未确认或计划过期不产生写入；下载/校验/解压失败进入 `failed` 并清理 staging；用户取消进入 `cancelled`；Android 单包失败进入 `partial` 并显示成功/失败包。

## 5. 安全与可靠性

下载任务有 context、总超时、响应大小上限和进度；绝不接受 UI 的任意 URL。计划摘要绑定 provider/version/os/arch/mirror/destination/checksum 与创建时间。最终目录写入使用同文件系统 staging + 原子 rename；崩溃残留由后续只读清理提示处理，不自动递归删除未知路径。

## 6. 更新记录

| 版本 | 日期 | 变更类型 | 说明 |
|---|---|---|---|
| v1.0 | 2026-10-28 | initial | 固定 provider 下的预览、确认与安装 |
