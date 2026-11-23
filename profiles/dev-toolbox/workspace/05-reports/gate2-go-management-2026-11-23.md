# Gate 2 — go-management 交付门控

日期：2026-11-23 | 结论：**PASS**

## 自动化
- `go test ./...` 全绿；新增 internal/goenv（5 用例：读/预览不写/写+unset 往返/非可编辑键拒绝/confirm 强制，均隔离 GOENV+HOME+APPDATA+XDG）与 internal/runtimeinstall manage（4 用例：自有目录删除/外部目录保护/dryRun 不改/清除全局默认）。
- `go vet -tags gui ./internal/desktop` 通过；前端 `vue-tsc` 通过、`build:desktop` 成功。

## 真机（Windows）
- GUI 客户端「用户级运行时设置 → Go 管理」加载真实数据：go1.27.3 二进制/GOENV 路径、GOROOT/GOMODCACHE/GOOS/GOARCH 只读项、GOPROXY(https://goproxy.cn,direct)/GOPATH/GOBIN 可编辑行+代理预设、4 个 Go 安装（含全局默认标记、已下载/已登记来源标签、设为默认/注销按钮）。
- 发现并修复：GUI（无控制台）父进程派生 `go env` 控制台子进程在锁屏/无窗口站下挂起 → 子进程加 CREATE_NO_WINDOW（sysproc_windows.go），修复后正常返回。

## 未覆盖（如实）
- 锁屏期间像素截图不可用，视觉自查以无障碍树内容核对代替；解锁后建议人工过目「Go 管理」页签排版。
- Linux 的 goenv spawnAttrs 返回 nil（无 CREATE_NO_WINDOW 概念），未真机验证（CI Linux runner 覆盖单测）。