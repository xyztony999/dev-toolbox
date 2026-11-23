//go:build !windows

package toolconfig

import "syscall"

func spawnAttrs() *syscall.SysProcAttr { return nil }
