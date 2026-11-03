//go:build !windows

package session

import (
	"os"
	"path/filepath"
	"syscall"
)

func getEnv(key string) string { return os.Getenv(key) }

// sessionSysProcAttr：Linux 无新控制台概念。
func sessionSysProcAttr() *syscall.SysProcAttr { return nil }

// linuxLaunch 直接启动交互 shell，继承调用方 stdio（适用于 CLI 场景；
// GUI 内启动属已知限制，见实现报告）。
func launchFor(terminal TerminalInfo, cwd string) (string, []string) {
	argv := []string{filepath.Base(terminal.ExePath), "-i"}
	return terminal.ExePath, argv
}
