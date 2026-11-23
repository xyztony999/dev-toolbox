//go:build windows

// guiopen：以 GUI 子系统（-H windowsgui，无控制台）验证产品 session.Open
// 启动链——桌面客户端的等价条件。结果写入临时文件（GUI 进程无 stdout）。
package main

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"
	"unsafe"

	"dev-toolbox/internal/session"
)

var modkernel32 = syscall.NewLazyDLL("kernel32.dll")

func main() {
	out := os.TempDir() + "\\guiopen-result.txt"
	started := time.Now()
	result, err := session.Open(context.Background(), `D:\Project`, "powershell")
	if err != nil {
		_ = os.WriteFile(out, []byte("OPEN-FAIL: "+err.Error()), 0o600)
		return
	}
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if !alive(result.Pid) {
			_ = os.WriteFile(out, []byte(fmt.Sprintf("FLASH-EXIT pid=%d code=%d elapsed=%s",
				result.Pid, exitCodeOf(result.Pid), time.Since(started).Round(time.Millisecond))), 0o600)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	_ = os.WriteFile(out, []byte(fmt.Sprintf("ALIVE-AFTER-8S pid=%d", result.Pid)), 0o600)
}

func openProc(pid int) uintptr {
	open := modkernel32.NewProc("OpenProcess")
	h, _, _ := open.Call(0x1000, 0, uintptr(pid))
	return h
}

func alive(pid int) bool {
	h := openProc(pid)
	if h == 0 {
		return false
	}
	defer modkernel32.NewProc("CloseHandle").Call(h)
	var code uint32
	r, _, _ := modkernel32.NewProc("GetExitCodeProcess").Call(h, uintptr(unsafe.Pointer(&code)))
	return r != 0 && code == 259
}

func exitCodeOf(pid int) uint32 {
	h := openProc(pid)
	if h == 0 {
		return 0
	}
	defer modkernel32.NewProc("CloseHandle").Call(h)
	var code uint32
	r, _, _ := modkernel32.NewProc("GetExitCodeProcess").Call(h, uintptr(unsafe.Pointer(&code)))
	if r == 0 {
		return 0
	}
	return code
}
