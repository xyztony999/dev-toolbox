# 合并方向与实施边界

2026-10-27。性质：项目初始化的方向记录；不是已通过 Gate 1 的完整 BA/AA/PRD/API/PageSpec。

## 用户已确定

- 在 D:/Project 新建 dev-toolbox，使用 DevToolbox 产品名。
- 引擎注册新业务 profile。
- 桌面采用 Go 核心 + WebView + Vue 3 + Ant Design Vue。
- 依据现有项目技术栈决定合并方向。

## 本轮形成的工程方向

一个产品、一个新仓库、多个清晰模块。Go 应用服务是 CLI、GUI、托盘的共同入口。以 DevSwitch 现有 Go 能力为迁移基础，把 gitid 的 Node 核心行为移植为 Go Git 模块，并以原有沙箱行为规格回归。最终产品不依赖被管理的 Node 才能管理 Git 身份，避免切换 Node 影响管理工具自身。

概念依赖为：Vue 页面 → 白名单桥接 → Go 应用服务 → 项目发现 / 运行时 / Git 身份 / 迁移 / 诊断 → 操作系统适配。CLI 直接调用同一应用服务，前端不能执行任意 shell，也不复制核心状态。

计划目录：cmd/（命令入口）、internal/project、internal/runtime、internal/gitidentity、internal/diagnostics、internal/migration、internal/platform、internal/gui、frontend/。当前尚未创建这些业务实现。

## 项目上下文

- Project 是用户登记的工作集合；Repository 是 Git 工作树；Module 是拥有独立构建清单的子目录。三者不强制一一对应。
- 每个模块记录 RuntimeRequirement，保留来源文件、版本约束原文、解析置信度、冲突与未知状态。
- LocalBinding 记录本机安装路径与身份选择；共享清单只记录可移植版本要求。Git 身份绑定默认本机保存，不能把私人账号/token提交到团队项目。
- EffectiveContext 根据当前目录、项目规则、会话覆盖和全局默认动态求值，Git 身份的最终结果仍交给 git --show-origin 等原生命令解析。
- Git includeIf 只承担 Git 身份；它不会改变 Node/Python/JDK。运行时切换使用独立的会话/子进程环境机制。

## 范围优先级

### 第一批：保留现有能力并建立项目关联

迁移 DevSwitch 七类工具的扫描/导入/切换/已有下载能力，并保留它的**用户级状态**：七种工具分别选择的全局默认运行时，以及运行时清单中的版本、安装路径、来源、label/vendor。用户执行迁移前先预览和备份；导入幂等，保留原安装目录与原 `state.json`，新旧应用不同时写旧状态。缺失路径的选择也保留为待处理来源记录，不猜测或自动替换。

项目级与会话级选择作为覆盖层；切换项目或启动子进程不改写用户全局默认，只有显式“设为全局默认”操作能更新它。DevSwitch 的项目自动切换只影响当前 shell 会话，不得在迁移后变成全局设置。其用户 shell hook/PATH 生效物归后续兼容迁移设计，迁移状态导入本身不改 shell profile、PATH 或注册表。

迁移 Git 身份、签名、目录绑定、凭据选择、仓库审计和远程镜像语义。下载覆盖按现有 provider 明确列出，不能将目前 latest-only 的 Python/Go 下载能力描述为任意历史版本可安装。

新增多模块静态发现与只读联合体检。支持 Node/Python/Java 的显式版本约束；Python 必须能表达严格 3.11，而不是只选 latest。Maven/Gradle 优先展示项目 Wrapper，识别源码目标、构建 JVM、toolchain 之间的区别。端点协议和版本解析规则留给后续设计。

### 第二批：可靠应用与兼容迁移

从项目视图打开指定上下文的终端/子进程；多个项目同时运行时互不切换全局默认。PowerShell/cmd/Git Bash/Linux shell 分别设计并验证。

GUI 不能改变已有父 shell 的环境；shim 能力和 JAVA_HOME 等会话环境刷新分别展示，不承诺“所有已开程序立即生效”。项目一键应用先计划、后执行，结果逐步记录；跨 Git/配置/安装过程不宣称原子事务，部分失败有可重试与尽力回退的明确边界。

旧 gitid、devswitch 的数据目录默认只读探测，用户执行迁移时预览、备份、版本标记、幂等导入。新项目采用自己的数据目录（具体 schema 后续设计），不让两个应用同时写旧文件。旧命令先保持独立可用；新安装器里的兼容命令仅在实现完整语义后提供，不能抢占现有 PATH 命令。

### 第三批：真实需要的扩展

Rust/rustup/Tauri 与 ARM GCC/Make 检测和已有工具链导入；保留 provider 扩展点。Docker/数据库暂只考虑诊断线索，不纳入首版自动安装、启动、卸载。

## 迁移质量基线

- 原有沙箱测试是行为证据；逐能力搬迁并对比，不直接重用原项目“Gate PASS”作为新项目结论。
- Git 的全量覆盖、配置来源、目录绑定、per-repo 镜像与凭据通道必须保持。
- 运行时选择、下载 provider、shell hook、环境清理必须分别验证；卸载仅清理本产品拥有且未被用户改写的资产。
- 静态 CLI 与 GUI 分开构建；Windows WebView2、Linux GTK/WebKitGTK/托盘依赖列入真实矩阵，GUI 产物不能声称完全无系统依赖。
- 各版本依赖以实施时的实际清单和验证结果锁定；本轮不引入依赖或执行原项目构建/安装。

## 后续工作

先执行新 profile 下的 D0–D4 与 Gate 1，再进入迁移实现。本轮仅把初始化接入标记完成，所有业务模块设计/实现与 Gate 1/2 均保持待办。
