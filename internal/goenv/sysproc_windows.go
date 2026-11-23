//go:build windows

package goenv

import "syscall"

// createNoWindow：go env 是控制台程序；从 GUI 子系统父进程派生时若不抑制
// 控制台分配，会在无窗口站/锁屏场景挂起或闪出黑窗。后台内省一律隐藏窗口。
const createNoWindow = 0x08000000

func spawnAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
