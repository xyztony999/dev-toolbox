//go:build windows

package toolconfig

import "syscall"

// createNoWindow：npm/pip 等为控制台程序；GUI 父进程派生时不抑制控制台
// 分配会在无窗口站/锁屏场景挂起或闪窗。后台读写一律隐藏窗口。
const createNoWindow = 0x08000000

func spawnAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNoWindow}
}
