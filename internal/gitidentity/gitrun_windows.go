//go:build windows

package gitidentity

import "syscall"

// hideConsoleSysAttr 防止从窗口化桌面客户端 spawn 控制台程序 git.exe 时
// 弹出一闪而过的控制台窗口。
func hideConsoleSysAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
