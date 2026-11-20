# Gate 2 — 桌面体验增量修复（project-context / session-launcher）

日期：2026-11-18 | 模式：增量（既有模块行为修正） | 结论：PASS

## 背景

用户反馈三条：登记目录（D:\Project）下嵌套多项目时仓库/模块长列表难以管理；"在此打开终端"无法使用且面板意图不清；部分项目出现"未声明"，询问能否在客户端解决 conflict。

## 修复内容

| # | 项 | 落点 |
|---|---|---|
| 1 | 终端启动链：conhost 封装（GUI 子系统派不出目标进程 / 控制台下劫持当前控制台）→ 目标终端 exe + CREATE_NEW_CONSOLE 直启 + 三份 DevNull stdio；Open 自带 Files，startSession 仅兜底 | internal/session/sys_windows.go、terminal.go |
| 2 | Git Bash 探测排除 WSL `System32\bash.exe`，优先 Git for Windows 安装位置 | internal/session/terminal.go |
| 3 | 项目详情分区导航：概览 / Git 仓库 / 构建模块 / 扫描提示四页签 + 关键字过滤 + "仅看待处理" | frontend/src/App.vue |
| 4 | 打开终端面板重设计：意图说明、工具逐行（版本/跳过原因/binDir）、JAVA_HOME 与 PATH 摘要、Git 身份行、空终端禁用态 | frontend/src/SessionManager.vue |
| 5 | 运行时声明状态中文标签 + 悬浮说明 + 证据明细；`evidence` 进入前端契约 | App.vue、types.ts |
| 6 | 冲突语义修正：默认按 runtime 分组，androidsdk 按 locator 拆分（compileSdk/minSdk 不再误报） | internal/project/scan.go |
| 7 | 登记预览 issues 重复渲染修复；演示桥补 session.preview/open | App.vue、dev-mock.ts |

## 验证

| 检查 | 命令 | 结果 |
|---|---|---|
| 静态构建 | `go build ./...` | PASS |
| 单元测试 | `go test ./...` | PASS（session/project 新增启动契约与冲突语义用例） |
| go vet | `go vet ./...` | PASS |
| GUI 编译 | `go build -tags gui` | PASS（CGO mingw-w64 16.2.0） |
| 前端类型 | `npm run typecheck` | PASS |
| 前端构建 | `npm run build:desktop`（同步内嵌） | PASS（1,583,851 B > 1.5MB，宿主走 file:// 回退路径） |
| 真机终端冒烟 | `session preview/open D:\Project` | PASS（返回 pid，新控制台窗口启动） |

## 内嵌 UI 体积：vendor 拆分的结论（2026-11-20 修订）

原报告建议"后续可做 vendor 拆分缩减体积"，实测后**不建议**，理由：

1. 构成：内嵌 HTML 1,578,252 B，其中 ant-design-vue + Vue 依赖约 1,503,874 B（95%），本项目自身代码约 74,378 B（5%）。
2. 真正的问题是 antd 全量引入（`main.ts` 的 `import Antd from 'ant-design-vue'`），不是"缺拆分"。
3. vendor 拆分不能恢复 SetHtml 路径：拆出 `vendor.js` 后 index.html 会降到约 7.4 万字符而低于 1.5 MB 阈值，但 `SetHtml` 的文档源是 about:blank，相对路径的 `<script src>` 无法解析——仍必须走 file:// 导航。收益为零，只增加一次文件写入和失败面。
4. 若目标是压回 1.5 MB 以内以恢复 SetHtml：需要的是 antd 按需引入（`ant-design-vue/es/*` 逐个注册，约 20 个组件，配合既有 `components.d.ts` 白名单）。仅注册本项目用到的 17 个标签 + message/Modal/theme 三个命令式 API。复杂度中等，收益确定，但属独立改动，应在完整 D0-D4 后单独排期，不混入本增量。

当前 file:// 回退路径已验证可用（客户端实机启动、前端心跳上报正常、无前端错误），非阻塞。

## 待办与限制

- GUI 内点击"打开终端"的真机点验留给用户（窗口级交互，自动化不覆盖）。
- Git for Windows 未安装的机器上终端候选只剩 PowerShell/cmd；已由面板禁用态与说明覆盖。
- antd 按需引入以压回 1.5 MB（恢复 SetHtml 直载）作为独立后续项，见上文结论。

## 对用户第三个问题的答复

"未声明"= 构建清单存在但无版本声明，按产品规范 7 只读呈现、不擅自 latest。"conflict" 现在是证据齐全的并列展示 + 人工决定：列出来源文件、属性名、原始值与解析器，改哪个由你决定；产品不做自动挑选（也不写回你的构建文件）。
