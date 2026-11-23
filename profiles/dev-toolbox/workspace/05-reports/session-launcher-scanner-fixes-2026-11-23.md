# 交付报告 — 终端闪退 / androidsdk 误报 / java 未声明 三问题修复

日期：2026-11-23 | 输入：用户报障（codex 修复轮次后的残留问题）+ 四张截图 | 模块：session-launcher、project-context

## 问题与根因（全部实测定位，零猜测）

### 1. 终端"能呼叫但立马闪退"（session-launcher）

二分实验（tools/diagterm，已删）实证三因子：

| 启动方式 | 结果 |
|---|---|
| stdio 指向 NUL 设备（codex 版实现：os.StartProcess + DevNull×3） | 100% 立即退出——交互 shell 读 stdin 得 EOF，`-NoExit`/`/K` 挡不住 |
| 继承父进程真实句柄 | 附着父控制台，不开新窗 |
| Win32 CreateProcess + 不传 std 句柄 + CREATE_NEW_CONSOLE | 存活，但**随机闪退**（见下） |

进一步采样发现隐层缺陷：不传 stdio 时，子进程的 stdio 依赖 conhost attach 与 .NET 运行时的启动赛跑——输掉即 PowerShell 完全启动后（3.3~4.1s，exit code 0）读 stdin 得 EOF 正常退出，实测概率 1/4~1/2；挂起启动 + Resume 前宽限治不了该竞态。

**修复（10/10 连测存活）**：`sys_windows.go` 重写启动链——Win32 CreateProcessW + CREATE_SUSPENDED + 显式 `STARTF_USESTDHANDLES`：stdin = 命名管道读端（写端父持有一份 + DuplicateHandle 给子进程自持一份，双侧持有期间**永不 EOF**，任一方退出终端仍独立存活）；stdout/stderr = 可继承 NUL；bInheritHandles=TRUE；ResumeThread 前给 conhost 300ms 宽限（保险层）。键盘交互不受影响（交互 shell 走 conhost 输入缓冲，不经管道）。

验证：`tools/guiopen`（-H windowsgui，桌面客户端等价条件，保留作启动链回归工具）产品路径 10/10 ALIVE-8S；CLI 路径 powershell 11s+ / cmd / Git Bash 全部存活；`TestStartChainInheritsEnv`（env 注入经新链路）与新增 `TestTerminalSessionStaysAlive`（闪退回归锁）全绿；客户端真机点击全链路（抽屉→确认弹窗→启动）PowerShell 存活 22.9s+。

### 2. androidsdk 仍显示 conflict（project-context）

后端判定已由 codex 修对（本次真机复扫 SEG2105/app：compileSdk 31 + minSdk 21 均 resolved）。界面仍显示冲突的两个叠加原因：

- **旧扫描快照**：项目页展示登记时的扫描结果（11/12），不会自动重扫——修复前的判定。
- **partial 阻塞快照落地**：重扫时 `nim-proxy` 下新建的 `.venv-workers`（pyodide venv）触达 maxDepth → 整次扫描降级 partial → 按产品语义保留上次完整快照 → 用户重扫了也看不到修复。

**修复**：① 扫描器 v3——忽略目录在精确名单外增加 venv 族前缀规则（`.venv*`/`venv*`，覆盖 `.venv-workers` 等自定义名）；② 快照带 `scanToolVersion` 落盘，前端比对桥接的当前版本显示"扫描器已更新，建议重新扫描"横幅；partial 时不刷新版本号（数据与提示一致）。

真机 D:\Project 全树重扫：**complete、0 issues、冲突 6→0、已声明 19→37、模块 56→62**；客户端内重扫同样结果。

### 3. java"未声明"（project-context）

`findGradleJavaVersions` 只认 Gradle toolchain（`JavaLanguageVersion.of`），不识 Android 工程最常见的 `sourceCompatibility/targetCompatibility JavaVersion.VERSION_11` 与 kotlin `jvmTarget`。SEG2105F21Project_Group_9 即此。

**修复**：识别三类声明（toolchain / source、targetCompatibility / jvmTarget 含字面量与调用形式）；`VERSION_1_8` 与字面量 `8` 同义归一（Gradle JavaVersion.toVersion 语义）；冲突分组从"androidsdk 特例"泛化为"多属性运行时按 runtime+locator 拆组"——同一属性多个互斥值才冲突，source 11 + target 17 合法共存。真机：Group_9 app 模块 java 11 resolved。

## 状态与验证汇总

- Go 全套件绿（含 Windows 真实启动测试）；GOOS=linux vet/build 通过（Linux 测试由 CI runner 执行）。
- 前端 npm build 通过；桌面客户端（-tags gui）重建并重启，UI 横幅/重扫/终端启动真机截图验证。
- INDEX 已回写（project-context changelog_2026_11_23、session-launcher gate2 增量）。

## 遗留

- 「未声明」占位行的语义未变：根模块（AGP 项目根 build.gradle 确实无 Java 声明）仍如实显示未声明——判定的改进只覆盖有声明但此前不识别的形态。
- CI 的 Windows runner 可跑 `TestTerminalSessionStaysAlive`（长测试，short 模式跳过）；Linux 平台会话启动仍受"GUI 内无终端可附"限制（既有已知项）。
- 引擎仓（ai-project-engine）上一轮修复仍未 push，与本次无关，维持用户决策。
