# 桌面客户端实现记录（2026-10-27）

## 交付

- 新增 `internal/desktop` Go WebView 宿主，GUI 构建通过 `-tags gui` 启用；CLI 无参数且 GUI 构建时打开窗口，CLI 构建不引入 WebView 原生依赖。
- Vue 单文件前端通过 `npm --prefix frontend run build:desktop` 输出到 Go `embed` 资源目录。
- 原生桥接只开放项目列表、扫描预览、登记、读取、重扫、确认后移除，及用户级状态读取、DevSwitch 检查/预览/确认导入、确认设置全局默认。
- 桌面客户端调用 `internal/project` 与 `internal/userstate`，不通过命令行子进程调用服务；原项目仍然只读。

## 验证

- `go build ./...`：通过。
- `go vet ./...`：通过。
- `npm --prefix frontend run typecheck`：通过。
- `npm --prefix frontend run build:desktop`：通过，生成约 1.52 MB 内嵌 HTML。
- GUI 二进制构建通过：将 `C:\msys64\ucrt64\bin` 临时加入 PATH，设置 `CGO_ENABLED=1`、`CC=gcc`、`CXX=g++` 后执行 `go build -tags gui -o dist/dev-toolbox.exe ./cmd/dev-toolbox`；产物约 10.5 MB。
- 未运行测试；Gate 2 仍待完成。

## 操作

```powershell
npm --prefix frontend run build:desktop
$env:Path = 'C:\msys64\ucrt64\bin;' + $env:Path
$env:CGO_ENABLED = '1'
$env:CC = 'gcc'
$env:CXX = 'g++'
go build -tags gui -o dist/dev-toolbox.exe ./cmd/dev-toolbox
```

GUI 构建需要 Windows C/C++ 编译器与 WebView2 Runtime。普通 `go build ./...` 继续构建 CLI。
