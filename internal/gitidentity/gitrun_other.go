//go:build !windows

package gitidentity

import "syscall"

// hideConsoleSysAttr 非 Windows 平台无需隐藏控制台，返回 nil。
func hideConsoleSysAttr() *syscall.SysProcAttr { return nil }
