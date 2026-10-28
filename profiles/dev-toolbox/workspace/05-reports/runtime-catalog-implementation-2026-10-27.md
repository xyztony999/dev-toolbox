# runtime-migration 本机发现与登记实现记录（2026-10-27）

## 实现

- `internal/runtimecatalog` 提供七工具静态发现、单目录检查、候选摘要复核和确认登记。
- 扫描来源为当前 PATH、限定常见目录及用户指定目录；用户目录只读根及直接子目录（最多 200 项），不递归全盘。
- 识别仅读取预期可执行文件和少量版本元数据，不启动候选二进制。静态版本证据不足显示 unknown。
- CLI 增加 `runtime discover [directory]`、`runtime inspect <tool> <directory>`、`runtime register <tool> <directory> --confirm`。
- WebView 白名单增加 `runtime.discover`、`runtime.inspectPath`、`runtime.register`；用户设置页可预览、选择及确认登记。
- 登记只写 DevToolbox 用户状态，路径+工具去重；不自动改变既有全局默认，也不改 PATH、注册表、shell profile 或来源目录。

## 验证

- `go test ./...`：通过；测试使用临时 HOME/USERPROFILE/APPDATA/XDG_CONFIG_HOME 与隔离 Git 配置。
- `go build ./...`、`go vet ./...`：通过。
- `npm --prefix frontend run typecheck`、`npm --prefix frontend run build:desktop`：通过。
- Windows GUI tag 构建：通过；`dist/dev-toolbox.exe`。
- 未做真实多版本 Windows/Linux 扫描矩阵或 GUI 启动抽检；Gate 2 仍待完成。

## 后续

下载 provider、安装/卸载、shim/PATH、shell hooks、项目/会话覆盖需各自 D0-D4 与门控；本次没有迁移这些副作用。
