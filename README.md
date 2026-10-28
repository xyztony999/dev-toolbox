# DevToolbox

多技术栈项目的开发环境与 Git 身份管理工具箱。以项目为入口，把"这个项目用哪套运行时、哪个 Git 身份"变成一眼可见、显式确认的一键操作。

- Go 共用业务核心：CLI 与桌面客户端调用同一应用服务。
- Go WebView 桌面宿主（Windows WebView2 / Linux WebKitGTK），不用 Electron，也不以 Node 承载最终 Git 核心。
- Vue 3 + TypeScript + Ant Design Vue 前端，Vite 构建后由 Go 内嵌。
- 由 ai-project-engine 工程流水线驱动：设计 → Gate 1 → 实现 → Gate 2，设计与验证报告随仓库维护（`profiles/dev-toolbox/workspace/`）。

## 当前能力

### 项目上下文

- 登记/扫描/重扫本机项目：识别 Git 工作树（含 worktree、嵌套仓库）与构建模块，提取版本要求（package.json engines、.nvmrc、pyproject、pom、Gradle toolchain/Wrapper），保留来源与置信度；只读，不运行项目脚本。

### 用户级运行时

- 七类工具（Node/Java/Maven/Gradle/Android SDK/Go/Python）的静态发现、检查与登记。
- 从固定 provider 下载并安全安装：预览计划（版本/镜像/校验/目标/体积）→ 显式确认 → 校验和验证 → staging 安全解包 → 原子归位；支持取消与 Android 分包部分重试。
- DevSwitch 用户级状态迁移（预览/备份/幂等导入）；全局默认只能显式设置，项目与会话选择不隐式改写全局。

### Git 身份

- 多身份档案（name/email/signingkey/gpg.format/自定义键/URL 重写/账号选择器），应用为全量覆盖：档案缺失的受管键被清除，不残留上一身份。
- dry-run 与真跑同一算法；多仓库批量逐仓出结果；includeIf 目录绑定（Windows gitdir/i 真实路径匹配，解绑前手改漂移警告）。
- 仓库扫描（✓/✗/⚠/⊕）、提交审计与 strict 门禁（mailmap 仅建议，不写 `.mailmap`）、只读 Git 体检、按仓库镜像推送、凭据账号经 git credential 协议直通 helper（token 不落盘不回显）。
- 旧 gitid 档案（config.json）预览/备份/幂等导入。

### 桌面客户端

- Go WebView2 宿主 + 内嵌单文件前端；桥接只开放白名单操作，异步执行不冻结窗口。
- 深浅主题跟随系统；应用图标与品牌同源。

## 构建

要求：Go ≥ 1.24、Node（仅前端构建）、Windows 下另需 MinGW-w64（CGO）与 WebView2 Runtime。

```powershell
# 静态 CLI（无 CGO 依赖）
go build -o dist/dev-toolbox.exe ./cmd/dev-toolbox

# 桌面客户端
npm --prefix frontend run build:desktop   # 生成 internal/desktop/ui/index.html
$env:CGO_ENABLED = '1'                    # 需 MinGW-w64 的 gcc
go build -tags gui -ldflags="-H windowsgui" -o dist/dev-toolbox-gui.exe ./cmd/dev-toolbox

# 前端开发模式（演示数据，不触磁盘）
npm --prefix frontend install
npm --prefix frontend run dev
```

CLI 入口示例：

```text
dev-toolbox project scan <path>
dev-toolbox runtime discover [目录] / install <tool> <selector> --confirm
dev-toolbox runtime legacy devswitch inspect|preview|import
dev-toolbox git profiles / use <id> --confirm / current / bind <id> <dir> --confirm
dev-toolbox git scan / audit --expect <id> --strict / doctor
dev-toolbox git legacy preview|import
```

## 安全边界

- 扫描与体检只读；不运行项目脚本、Wrapper、hook 或候选程序。
- 安装只写 DevToolbox 自有用户目录；不改 PATH、注册表、shell profile，不自动切换默认。
- Git 配置以 git 自身为事实源；只有显式确认的操作经 `git config` 写入。
- 凭据 token 只经 git credential 协议进系统 helper，不落档案、不进日志、不回显。
- 旧配置迁移先预览、备份、按来源摘要幂等导入；不抢占旧命令，不与旧工具同写旧文件。

## 仓库结构

- `cmd/` `internal/`：CLI、桌面宿主与业务核心（project / runtimecatalog / runtimeinstall / userstate / gitidentity / desktop）。
- `frontend/`：Vue 3 + Ant Design Vue 前端源码与构建脚本。
- `profiles/`：ai-project-engine 工程资产——业务知识、D0–D4 设计、Gate 报告、模块状态索引。
- `docs/`：合并方向与 D:/Project 技术栈盘点。
- `tools/`：静态盘点脚本与应用图标生成器。

## 迁移来源与致谢

本项目的 Git 身份语义来自 [gitid](https://github.com/xyztony999/gitid)，运行时管理与下载 provider 行为来自 [DevSwitch](https://github.com/xyztony999/devswitch)（均为同作者的先行项目，MIT）。迁移以原沙箱行为规格做对比回归，原项目保持独立可用。详见 [NOTICE](./NOTICE)。

第三方依赖：[webview_go](https://github.com/webview/webview_go)（MIT）、[ulikunitz/xz](https://github.com/ulikunitz/xz)（BSD-3-Clause）、Vue/Ant Design Vue/Vite（各自许可）。

## 许可

[MIT](./LICENSE) © 2026 Xinyi Zhang
