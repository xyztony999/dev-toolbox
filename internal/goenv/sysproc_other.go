//go:build !windows

package goenv

import "syscall"

func spawnAttrs() *syscall.SysProcAttr { return nil }
