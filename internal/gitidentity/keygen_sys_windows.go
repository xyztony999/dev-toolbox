//go:build windows

package gitidentity

import "syscall"

// toolSysAttr hides the console window without CREATE_NO_WINDOW.
// CREATE_NO_WINDOW breaks gpg-agent IPC on Windows (agent cannot start /
// connect when the parent was spawned with that flag).
func toolSysAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true}
}
