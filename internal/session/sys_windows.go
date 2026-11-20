//go:build windows

package session

import (
	"os"
	"path/filepath"
	"syscall"
)

func getEnv(key string) string { return os.Getenv(key) }

// createNewConsole 令 Windows GUI 子系统进程（桌面客户端）也能开出
// 独立控制台窗口；父进程侧不对外暴露终端窗口。
const createNewConsole = 0x00000010

// launchFor 直接以目标终端的可执行文件启动新控制台窗口。
// 历史 conhost.exe 封装链路在 GUI 子系统实测派不出目标进程（用户
// 反馈"打不开终端"），从既有控制台运行时又会劫持当前控制台；
// CreateProcess 的 CREATE_NEW_CONSOLE 语义直白，环境由
// ProcAttr.Env 一次传递，stdin EOF 不影响 -NoExit//K 会话存活。
// 返回 (exe 全路径, argv)。
func launchFor(terminal TerminalInfo, cwd string) (string, []string) {
	switch terminal.ID {
	case "powershell":
		return terminal.ExePath, []string{"powershell", "-NoLogo", "-NoExit"}
	case "cmd":
		return terminal.ExePath, []string{"cmd.exe", "/K"}
	case "bash":
		return terminal.ExePath, []string{filepath.Base(terminal.ExePath), "--login", "-i"}
	default:
		return terminal.ExePath, []string{terminal.ID}
	}
}

// sessionSysProcAttr：新控制台标志；无需 HideWindow（窗口即产品意图）。
func sessionSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{CreationFlags: createNewConsole}
}
