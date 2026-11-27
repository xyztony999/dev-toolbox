//go:build !windows

package gitidentity

import "syscall"

func toolSysAttr() *syscall.SysProcAttr { return nil }
