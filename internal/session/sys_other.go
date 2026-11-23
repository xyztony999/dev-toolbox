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

// startProcessSession 非 Windows 平台直接 os.StartProcess：Files 为 nil
// 时子进程继承调用方 stdio（CLI 场景在当前终端内落地交互会话；
// GUI 内启动属已知限制，见实现报告）。
func startProcessSession(exePath string, argv []string, attr *os.ProcAttr) (int, error) {
	proc, err := os.StartProcess(exePath, argv, attr)
	if err != nil {
		return 0, err
	}
	go func() { _, _ = proc.Wait() }()
	return proc.Pid, nil
}

// linuxLaunch 直接启动交互 shell，继承调用方 stdio（适用于 CLI 场景；
// GUI 内启动属已知限制，见实现报告）。
func launchFor(terminal TerminalInfo, cwd string) (string, []string) {
	argv := []string{filepath.Base(terminal.ExePath), "-i"}
	return terminal.ExePath, argv
}
