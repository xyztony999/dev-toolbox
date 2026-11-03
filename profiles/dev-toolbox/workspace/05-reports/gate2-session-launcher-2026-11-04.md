# Gate 2 验证报告 — session-launcher

检查日期：2026-11-04 | 范围：internal/session 服务（上下文求值/环境组装/终端目录/启动器）、CLI、桥接、SessionManager 面板 | 设计：gate1-session-launcher-2026-11-03.md

## 检查结果

| 检查项 | 结果 | 依据 |
|---|---|---|
| GR-01 激活链规则扫描 | PASS | 零持久写（SL-05 测试断言启动前后 revision 不变）；注入仅子进程树；启动终端是显式用户动作 |
| GR-02 编译通过 | PASS | `go build ./...` + `GOOS=linux go build ./...` + `-tags gui` 三重通过 |
| GR-03 新接口完整链路测试 | PASS | 4 测试：上下文求值（injected/skipped×3 原因/JAVA_HOME/去重）、BuildEnv（PATH 前置/保留父 JAVA_HOME）、Open（argv/env/cwd 捕获+全局状态断言）、**env 继承实证**（注入变量经启动链→新窗口 powershell→写文件验证，UTF-16LE 载体） |
| GR-04 测试编译通过 | PASS | `go test ./internal/... -count=1` 全绿 |
| GR-05 前后端契约对齐 | PASS | session.preview/open 与 D3 字段一致；面板完全由 preview 数据驱动 |
| GR-06 对齐记录 | PASS | 本报告 |
| 真机 CLI | PASS | `session preview`（真机全局选择求值正确）；`session open --terminal powershell` 启动 conhost+powershell 窗口（进程存活验证） |
| 真机 GUI | PASS | 项目详情「在此打开终端」抽屉：预览/选择/启动全链路截图验证（toast "终端已启动 powershell pid 47404"） |

## 过程发现并修复的缺陷

1. **WebView2 NavigateToString 隐性 1.5MB 上限**：bundle 从 1,566,363 涨到 1,574,138 后页面静默留 about:blank（无任何 JS 错误）。修复：超限回退写临时文件 + file:// 导航（保持单文件内嵌分发）。已沉淀前端错误通道（window.onerror/unhandledrejection → frontend-error.log）与心跳标记供后续诊断。
2. **终端启动链**：cmd /C start 的参数转发不可靠（实测派生 powershell 收不到参数）；DevNull stdio 使交互 shell 立即 EOF。修复：Windows 经 conhost.exe 新控制台直启；ProcAttr.Files 必须显式给句柄。
3. `-EncodedCommand` 需 UTF-16LE（测试实证链路用）。

## 已知限制（如实）

- 返回的 pid 为启动器进程（conhost）id，不是窗口内 shell 的 pid。
- Linux GUI 内启动无 tty 的 $SHELL 属已知限制（CLI 场景正常）；Linux 未真机验证。
- Git Bash 窗口未在真机点验（探测逻辑就位）。

## 判定

**PASS** — 自动化（4 测试含 env 继承实证）+ Windows 真机 CLI/GUI 全部通过。
