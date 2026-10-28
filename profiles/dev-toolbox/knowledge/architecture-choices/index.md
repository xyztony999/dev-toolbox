# 技术选型与依据

用户已确定 Go + WebView + Vue 3 + Ant Design Vue。TypeScript、Vite、Go embed 为配套工程方向。

- Go 核心与无 GUI CLI：参考 DevSwitch go.mod 的 Go 1.24 基线；本轮不创建 go.mod，最终工具链版本于实现时锁定。
- WebView 宿主：优先评估复用 DevSwitch 的 webview_go/systray 集成；Windows WebView2、Linux GTK/WebKitGTK 的 CGO 和本机依赖需真实验证，未切换到 Wails/Tauri/Electron。
- Vue/AntD：gitid 与 deck-loop 已有 Vue 3.5、ant-design-vue 4.2.6 工程经验；这些是来源声明，非最新版本或兼容性保证。实际 package.json/lockfile 实施时确定。
- Go 项目核心不依赖正在被管理的 Node/Python/Java。Node 只用于前端开发/构建和本轮静态盘点脚本。
- 初始平台为 Windows/Linux，保留既有 x64/arm64 需求；不把 macOS 纳入已验证矩阵。
- 用户数据需要独立命名空间与 schema；不复用旧项目的写入路径。具体路径和迁移协议留待设计。
- DevSwitch 的用户级 `state.json` 是运行时迁移来源：`current` 保存七种工具的全局选择，`runtimes` 保存已发现/导入/下载实例元数据。新产品须显式预览、备份后导入到自己的本机状态；旧文件与运行时目录保持原位，迁移不自动重写旧状态、shell profile 或 PATH。
- 项目/会话选择是对全局默认的有作用域覆盖；只有显式“设为全局默认”才更新 GlobalRuntimeSelection。DevSwitch 项目自动切换的会话级语义不得被折叠成全局选择。

实现边界：核心 UI 无关、桌面桥接白名单、可取消耗时操作、凭据不广播、解析与文件系统副作用分离。项目扫描默认不运行任何项目命令。
