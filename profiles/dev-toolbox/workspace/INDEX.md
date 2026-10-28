---
pipeline:
  - {id: runtime-catalog-discovery, title: 本机运行时发现与登记设计, path: 00-architecture/d0-runtime-catalog-discovery.md, status: complete, depends_on: [user-runtime-state-migration]}
  - {id: runtime-download-install, title: Provider 下载与安全安装设计, path: 00-architecture/d0-runtime-download-install.md, status: complete, depends_on: [runtime-catalog-discovery]}
modules:
  - name: onboarding
    title: 项目初始化、技术栈盘点与 Profile 接入
    domain: platform
    stages:
      requirement: {status: complete, note: 用户约束与静态证据已登记}
      validation: {status: complete, note: 三个 profile check 均 PASS；激活 7 项；doctor 13 PASS 和插件模式预期 D-14 WARN；详见接入报告}
    gates:
      gate1: {status: pending, note: 本轮未执行设计门控}
      gate2: {status: pending, note: 本轮无产品实现}
  - name: project-context
    title: 项目/仓库/模块发现与上下文
    domain: project
    stages:
      requirement: {status: complete, note: D0-D2 已完成只读项目/仓库/模块发现设计}
      api: {status: complete, note: D3 Go 应用服务契约 v1.0}
      interaction: {status: complete, note: D4 项目上下文页规格 v1.0}
      backend: {status: partial, note: Go 只读发现、JSON CLI 与 WebView 白名单桥接已实现；Gate 2 待完成}
      frontend: {status: partial, note: Vue/Ant Design Vue 页面接入 Go WebView 桌面宿主；需构建前端内嵌包后使用 -tags gui 构建客户端}
      validation: {status: todo}
    gates:
      gate1: {status: pass, note: 2026-10-27 D0-D4 project-context 设计通过；详见 gate1-project-context-2026-10-27.md}
      gate2: {status: pending}
  - name: runtime-migration
    title: 七工具能力迁移与版本约束
    domain: runtime
    stages:
      requirement: {status: complete, note: D0-D2 完成本机只读发现与显式登记设计；下载/生效物另行设计}
      api: {status: complete, note: D3 runtime-catalog API v1.0 定义发现/检查/登记契约}
      interaction: {status: complete, note: D4 runtime-catalog PageSpec v1.0 完成并与 API 对齐}
      backend: {status: partial, note: 七工具静态发现/检查/显式登记与 CLI、WebView 桥接已实现；下载及环境生效后续设计}
      frontend: {status: partial, note: 用户设置页加入候选扫描、指定目录预览和确认登记；Gate 2 待完成}
      validation: {status: todo}
    gates:
      gate1: {status: pass, note: 2026-10-27 runtime-catalog-discovery D0-D4 PASS；详见 gate1-runtime-migration-2026-10-27.md}
      gate2: {status: pending}
  - name: runtime-download-install
    title: 七工具 Provider 下载与安全安装
    domain: runtime
    stages:
      requirement: {status: complete, note: D0-D2 固定 provider 版本能力、预览确认、安全安装边界}
      api: {status: complete, note: D3 options/preview/start/status/cancel/event 契约 v1.0}
      interaction: {status: complete, note: D4 预览确认及异步任务 PageSpec v1.0}
      backend: {status: complete, note: 七 provider + 任务管理器 + CLI + 桥接已实现；go build/vet/test 通过；GUI tag 编译待 CGO 环境}
      frontend: {status: complete, note: 用户设置页新增获取运行时面板（预览/确认/进度/取消/部分成功）；typecheck 与生产构建通过，已同步内嵌包}
      validation: {status: complete, note: Gate 2 执行完成，判定 WARN（GUI 编译与真实网络冒烟未执行）}
    gates:
      gate1: {status: pass, note: 2026-10-28 runtime-download-install D0-D4 PASS；详见 gate1-runtime-download-install-2026-10-28.md}
      gate2: {status: warn, note: 2026-10-28 沙箱链路与契约对齐 PASS；GUI 编译已于同日解决（scoop mingw-w64 16.2.0，-tags gui 构建并成功启动 WebView 客户端）；真实 provider 下载冒烟待用户在客户端执行；详见 gate2-runtime-download-install-2026-10-28.md}
  - name: user-runtime-state-migration
    title: DevSwitch 用户级运行时选择与清单迁移
    domain: runtime
    stages:
      requirement: {status: complete, note: D0-D2 用户级 current/runtimes 迁移范围已定义}
      api: {status: complete, note: D3 预览/备份/幂等导入契约 v1.0}
      interaction: {status: complete, note: D4 全局选择与 DevSwitch 导入页 v1.0}
      backend: {status: partial, note: 用户状态读取/预览/备份/幂等导入已由 CLI 与 Go WebView 白名单桥接复用；Gate 2 待办}
      frontend: {status: partial, note: 全局默认和 DevSwitch 预览导入页已接入桌面原生桥及演示桥；Gate 2 待办}
      validation: {status: todo}
    gates:
      gate1: {status: pass, note: 2026-10-27 用户级状态迁移切片 Gate 1 PASS；完整七工具模块仍独立待设计}
      gate2: {status: pending}
  - name: git-identity-migration
    title: Git 身份核心 Go 移植与语义对比
    domain: git
    stages:
      requirement: {status: complete, note: D0 完成 gitid e33e728 能力基线、匹配与迁移切片}
      api: {status: complete, note: D3 git.* 22 操作契约 v1.0（含档案/绑定/审计/体检/镜像/凭据/legacy 导入）}
      interaction: {status: complete, note: D4 四面板 PageSpec v1.0，与 API 对齐}
      backend: {status: complete, note: internal/gitidentity 全能力 + CLI git 子命令 + 桥接 22 操作；静态与 GUI tag 双构建通过；9 项真实 git 沙箱测试通过}
      frontend: {status: complete, note: Git 身份四面板（档案/应用绑定/审计体检/迁移凭据）+ 演示桥；typecheck 与内嵌构建通过}
      validation: {status: complete, note: Gate 2 执行完成，判定 WARN（真实凭据 helper 交互与桌面面板人工点验待用户）}
    gates:
      gate1: {status: pass, note: 2026-10-28 git-identity-migration D0-D4 PASS；详见 gate1-git-identity-migration-2026-10-28.md}
      gate2: {status: warn, note: 2026-10-28 自动化链路全过（GR-01..06/WV/GC + 沙箱行为对比）；真实凭据 helper 交互与重启后人工点验待用户；详见 gate2-git-identity-migration-2026-10-28.md}
  - name: desktop-shell
    title: Go WebView 与 Vue/AntD 桌面宿主
    domain: desktop
    stages:
      requirement: {status: partial, note: 已确定 Go WebView 桌面窗口与同一应用服务桥接；2026-10-29 体验切片 D0 完成（异步桥接/dark mode/导航/图标），完整 D0-D4 待收尾}
      api: {status: partial, note: window.devToolbox 异步桥接契约确立（Invoke ACK + __devToolboxResolve 回填 + 事件推送）；独立 D3 契约待设计}
      interaction: {status: partial, note: 侧边栏全局能力/已登记项目双分组 + 按视图顶栏 + 系统跟随深浅主题；桌面壳其余交互待补}
      backend: {status: partial, note: gui build-tag WebView 宿主 + 22+ 白名单操作 + WM_SETICON 图标注入；CGO 工具链已就绪（mingw-w64 16.2.0 via scoop）}
      frontend: {status: partial, note: Vite 单文件内嵌输出 + dark mode 变量体系；详见 desktop-experience-implementation-2026-10-29.md}
      validation: {status: todo}
    gates:
      gate1: {status: pending}
      gate2: {status: pending}
  - name: diagnostics-and-compatibility
    title: 联合诊断与旧配置兼容迁移
    domain: platform
    stages:
      requirement: {status: todo}
      api: {status: todo}
      interaction: {status: todo}
      backend: {status: todo}
      frontend: {status: todo}
      validation: {status: todo}
    gates:
      gate1: {status: pending}
      gate2: {status: pending}
---
