# D0 需求澄清 — 桌面体验切片（desktop-shell 增量）

版本：v1.0 | 日期：2026-10-29 | 模块：desktop-shell | 输入：用户四项反馈

## 反馈 → 决策

| # | 反馈 | 根因 | 决策 |
|---|---|---|---|
| 1 | 无应用图标 | 从未设计 | /visualize 设计 Git 分支 × 身份令牌图形；Go 生成器按同一几何输出 16–256px ICO；运行时经 WM_SETICON 注入窗口（标题栏/任务栏/Alt-Tab）。Explorer 文件图标需 .syso 资源编译，另行处理 |
| 2 | UI 结构问题 + 无 dark mode | 顶栏固定显示 PROJECT CONTEXT；侧边栏"工作区"分组混装全局页与项目；全站仅浅色样式，系统深色下被强制反色导致不可读 | 顶栏按视图切换 eyebrow+标题；侧边栏分"全局能力/已登记项目"两组；CSS 全量变量化 + prefers-color-scheme 深色调色板 + AntD darkAlgorithm 跟随系统 |
| 3 | 点 Git 身份卡顿数秒 | webview Bind 回调在 UI 线程同步执行，git 子进程 spawn 期间整窗冻结（WV-05 违规） | 桥接改异步：Bind 立即 ACK，操作转 goroutine，响应经 Dispatch/Eval 回填 `__devToolboxResolve`；Git 面板首屏合并为单个 `git.bootstrap` 操作并加加载骨架 |
| 4 | 突兀的 Project Context | 顶栏与视图无关联 | 并入 #2：仅项目视图显示 PROJECT CONTEXT 语境；全局页有自己的标题 |

## 异步桥接契约（新增）

- `__devToolboxInvoke` 返回 `{accepted:true, requestId}`（立即）；处理结果以 `__devToolboxResolve(requestId, {requestId, ok, data, error})` 推送，sequence 无需去重（每请求一次）。
- 前端 bridge.ts 维护 requestId→Promise 映射；演示桥路径不变。
- 安装事件推送 `__devToolboxEvent` 保持不变。

## 暗色调色板

CSS 变量双套：浅色沿用原色值；深色 --bg-page #0d1928 / --bg-panel #13253a / --border #253c55 / --text #e6edf5 / 强调薄荷绿提亮至 #6fb3a9/#7fd6c2。侧边栏保持深海军（两套主题一致），selected #1c3954。`color-scheme: light dark` + index.html meta 同步。

## 暂不纳入

窗口记忆/托盘（后续切片）、多窗口、主题手动切换开关（当前系统跟随）。

## 更新记录

| 版本 | 日期 | 说明 |
|---|---|---|
| v1.0 | 2026-10-29 | 四项桌面体验反馈的根因与决策 |
