//go:build windows

package session

import (
	"os"
	"path/filepath"
	"syscall"
)

func getEnv(key string) string { return os.Getenv(key) }

// launchFor 经 conhost.exe 在新控制台窗口运行目标终端：conhost 创建
// 控制台并作为目标进程的父进程，环境由 ProcAttr.Env 直接继承——
// 没有 `cmd /C start` 的参数二次解析与 env 转发问题（实测该链路
// 派生的 powershell 不会收到参数）。返回 (conhost 全路径, argv)。
func launchFor(terminal TerminalInfo, cwd string) (string, []string) {
	conhost := filepath.Join(getEnv("SystemRoot"), "System32", "conhost.exe")
	var target []string
	switch terminal.ID {
	case "powershell":
		target = []string{"powershell", "-NoLogo", "-NoExit"}
	case "cmd":
		target = []string{"cmd.exe", "/K"}
	case "bash":
		target = []string{terminal.ExePath, "--login", "-i"}
	default:
		target = []string{terminal.ID}
	}
	return conhost, append([]string{"conhost.exe"}, target...)
}

// sessionSysProcAttr：conhost 自己管理窗口，无需特殊标志。
func sessionSysProcAttr() *syscall.SysProcAttr { return nil }
