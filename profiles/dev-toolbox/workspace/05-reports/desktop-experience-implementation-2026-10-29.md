# 实现报告 — 桌面体验切片（异步桥接 / dark mode / 导航 / 图标）

日期：2026-10-29 | 模块：desktop-shell | 依据：d0-desktop-shell-experience v1.0

## 实现

- **异步桥接**（internal/desktop/desktop.go + frontend/src/bridge.ts）：Bind 回调仅解析并 ACK；每个请求在独立 goroutine 处理，响应经 `w.Dispatch` + `__devToolboxResolve` 回填 Promise。UI 线程不再执行任何业务操作——git 子进程、provider 网络查询均不再冻结窗口。前端 pending 映射按 requestId 关联，错误语义不变。
- **git.bootstrap**：新增桥接操作，一次返回 profiles/marks/credentials/settings，Git 面板首屏从 2-3 次串行调用减为 1 次，并配 a-skeleton 加载态。
- **导航与顶栏**（App.vue）：侧边栏"全局能力"（项目上下文/用户级运行时设置/Git 身份）+ "已登记项目"两组；顶栏 eyebrow/标题按视图切换（PROJECT CONTEXT 仅项目视图），操作按钮仅项目视图出现。
- **dark mode**：style.css 全量 CSS 变量化（双调色板 + `color-scheme: light dark`），三个业务组件 scoped 样式同步变量化；AntD 经 a-config-provider 按 `prefers-color-scheme` 切换 darkAlgorithm/defaultAlgorithm，系统切换实时跟随。
- **图标**：tools/icongen（Go，纯标准库）按设计 SVG 同一几何生成 16–256px PNG 条目的 ICO（2× 超采样抗锯齿）；internal/desktop/icon.go 解析 ICO 并经 CreateIconFromResourceEx + WM_SETICON 注入宿主窗口。设计稿经 /visualize 产出供迭代。

## 验证（真实执行）

- `vue-tsc --noEmit` 通过；`npm run build:desktop` 内嵌包更新。
- `go build ./...`、`CGO_ENABLED=1 go build -tags gui` 通过；`go vet ./...` 无告警；`go test ./internal/...` 全 ok（gitidentity/runtimecatalog/runtimeinstall）。
- icongen 像素探针测试：节点薄荷绿 (128,224,192)、渐变海军蓝、令牌白色、角部 alpha=0 全部正确（修复了降采样 uint8 溢出与 ICO 头字节序两处缺陷）。
- 客户端以新构建重启，WebView2 渲染正常，启动日志无错误。

## 已知边界

1. Explorer 中 exe 文件图标需 .syso 资源（go-winres/rsrc 生成）——待网络可用时引入构建工具，当前窗口/任务栏/Alt-Tab 图标已生效。
2. 深浅主题为系统跟随，无手动切换开关（按 D0 决策）。
3. 异步桥接后各操作天然并发；userstate/gitidentity 写路径已有文件锁，project 存储沿用原子替换（最后写入获胜），如出现并发写冲突再收窄。

## 缺陷修复记录

- 2026-10-29：异步桥接首版把 Promise resolve 成了 `response.data`（仅数据），`invoke()` 的关联标识校验随后必然失败，所有成功调用抛"桥接响应关联标识无效"。修复为 resolve 完整响应对象（含 requestId）后再校验。该缺陷属桥接传输层，操作契约与 Go 侧无涉；typecheck + 内嵌构建 + 客户端重启已完成。
- 2026-10-29（第二轮布局反馈）：①"发现本机运行时"五个控件挤行 + 状态条叠压——统一改为显式 flex 工具行（.toolbar：grow 输入 + 固定宽度选择器 + 按钮），获取运行时与 Git 仓库审计两处同步；②顶栏 118px 过高——压缩为紧凑单行；③"已登记项目"空态与空菜单叠加出怪异间距与左缘残影——菜单改条件渲染，空态跟在分组标题下；④登记项目输入框与按钮堆叠——path-entry 改 flex 行内布局。
- 2026-10-29（第三轮，frontend-design 技能重布局）：用户指出应使用其新添加的 /frontend-design 技能而非徒手改样式。按技能方法重做：先出设计方案并对照"AI 默认味"清单自检，确认此前实现踩中多项默认套路，逐一修正——①删除全部 ALL-CAPS eyebrow（PROJECT CONTEXT/RUNTIME SETTINGS/GIT IDENTITY）；②去卡片化：同款圆角卡片+阴影改为文档流分区（透明背景+发丝分隔线），仅状态块保留表面；③主色 token 从 AntD 默认蓝换为品牌薄荷绿（随深浅主题 #3d8f7c/#58b8a4）；④顶栏横幅带取消，页头（标题+主操作+路径副行）融入内容列，登记页标题不再重复出现；⑤文案按技能写作规则过一遍：侧边栏空态改为行动指引、"发现状态：complete"裸枚举改人话、中点连接符改顿号、页头标题与侧边栏菜单名保持一致；⑥侧边栏品牌标替换为与图标同源的 Git 分支 SVG。typecheck/内嵌构建/客户端重启完成。
