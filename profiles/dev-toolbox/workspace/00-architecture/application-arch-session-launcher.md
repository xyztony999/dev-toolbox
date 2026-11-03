# 应用架构 — 会话启动器

版本：v1.0 | 日期：2026-11-03 | 模块：session-launcher

## 1. 服务边界

| 服务 | 职责 |
|---|---|
| ContextResolver（新 internal/session） | 按 GlobalRuntimeSelection 求值：tool→installation 可用性、bin 目录、JAVA_HOME；复用 userstate 与 gitidentity 的只读解析 |
| TerminalCatalog | 探测本机可用终端（powershell/cmd/bash/Git Bash；Linux $SHELL）及可执行路径 |
| SessionLauncher | 组装 env（os.Environ + 注入项）并以参数数组启动终端进程于所选 cwd |

## 2. 调用关系

CLI/WebView → session 服务 → userstate（读全局选择）/ gitidentity（读生效身份，零写）。启动进程经既有安全启动原语（参数数组、显式 env、明确 cwd）；Windows 终端用 CREATE_NEW_CONSOLE 而非隐藏标志（要的就是可见窗口）。

## 3. 数据与副作用

- 读：用户级运行时清单、git 配置（经 git 原生命令）、终端存在性探测。
- 写：无持久写；副作用仅为创建终端子进程（env 只存在于该进程树）。
- 注入项：PATH = 选中运行时 bin 目录列表 + 父 PATH（去重保留前置顺序）；JAVA_HOME = 选中 JDK 根；其余变量首版不注入。

## 4. 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-11-03 | 初始应用架构 |
