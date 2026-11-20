package session

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DetectTerminals 探测本机可用终端（只查文件存在性，不执行任何程序）。
func DetectTerminals() []TerminalInfo {
	terminals := []TerminalInfo{}
	if runtime.GOOS == "windows" {
		if systemRoot := os.Getenv("SystemRoot"); systemRoot != "" {
			powershell := filepath.Join(systemRoot, "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
			if fileExists(powershell) {
				terminals = append(terminals, TerminalInfo{ID: "powershell", Label: "PowerShell", ExePath: powershell})
			}
			cmd := filepath.Join(systemRoot, "System32", "cmd.exe")
			if fileExists(cmd) {
				terminals = append(terminals, TerminalInfo{ID: "cmd", Label: "命令提示符", ExePath: cmd})
			}
		}
		if bash := findGitBash(); bash != "" {
			terminals = append(terminals, TerminalInfo{ID: "bash", Label: "Git Bash", ExePath: bash})
		}
		return terminals
	}
	shell := strings.TrimSpace(os.Getenv("SHELL"))
	if shell == "" || !fileExists(shell) {
		shell = "/bin/bash"
	}
	if fileExists(shell) {
		terminals = append(terminals, TerminalInfo{ID: "bash", Label: filepath.Base(shell), ExePath: shell})
	}
	return terminals
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// wslBash 是 Windows 自带的 WSL bash（bash.exe），不是 Git Bash：
// 它既不接受 --login -i 的 msys 语义，也不是用户预期的终端。
var wslBash = filepath.Join(os.Getenv("SystemRoot"), "System32", "bash.exe")

// findGitBash 先查 Git for Windows 常见安装位置，再查 PATH 但排除
// WSL 的 System32\bash.exe；找不到返回空。
func findGitBash() string {
	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Git", "bin", "bash.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Git", "bin", "bash.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "Git", "bin", "bash.exe"),
	}
	for _, candidate := range candidates {
		if fileExists(candidate) {
			return candidate
		}
	}
	if path := searchPathFor("bash.exe"); path != "" && !strings.EqualFold(path, wslBash) {
		return path
	}
	return ""
}

func searchPathFor(name string) string {
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		if fileExists(candidate) {
			return candidate
		}
	}
	return ""
}

// startSession 是进程启动原语；测试注入替换。Windows 下 ProcAttr.Files
// 为 nil 会得到 invalid argument（gitidentity 的经验：必须显式给句柄），
// 终端有自己的新控制台，父侧给 DevNull 即可。
var startSession = func(exePath string, argv []string, attr *os.ProcAttr) (int, error) {
	if attr.Files == nil {
		if files := nullStdHandles(); len(files) == 3 {
			attr.Files = files
		}
	}
	proc, err := os.StartProcess(exePath, argv, attr)
	if err != nil {
		return 0, err
	}
	go func() { _, _ = proc.Wait() }()
	return proc.Pid, nil
}

// nullStdHandles 提供三份 DevNull 句柄：os.StartProcess 在 Windows 上
// 要求显式 stdio（Files 为 nil 直接 EINVAL）；终端窗口自带新控制台，
// 父侧不需要交互流，用完由调用方 GC/句柄表回收。
func nullStdHandles() []*os.File {
	files := make([]*os.File, 3)
	for i := range files {
		handle, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
		if err != nil {
			return nil
		}
		files[i] = handle
	}
	return files
}

// Open 以注入环境在 cwd 启动终端。环境在服务端重新求值，不信任调用方。
// Windows 以目标终端可执行文件 + CREATE_NEW_CONSOLE 开出新窗口
// （环境经 ProcAttr.Env 一次传递）；Linux 直接启动交互 shell
// 继承调用方 stdio。
func Open(ctx context.Context, cwd, terminalID string) (*OpenResult, error) {
	preview, err := ResolveContext(ctx, cwd)
	if err != nil {
		return nil, err
	}
	if terminalID == "" && len(preview.Terminals) > 0 {
		terminalID = preview.Terminals[0].ID
	}
	var chosen *TerminalInfo
	for i := range preview.Terminals {
		if preview.Terminals[i].ID == terminalID {
			chosen = &preview.Terminals[i]
			break
		}
	}
	if chosen == nil {
		return nil, fmt.Errorf("SESSION_TERMINAL_NOT_FOUND: %s", terminalID)
	}
	exePath, argv := launchFor(*chosen, preview.Cwd)
	attr := &os.ProcAttr{
		Dir:   preview.Cwd,
		Env:   BuildEnv(os.Environ(), preview.PathPrepends, preview.JavaHome),
		Files: nullStdHandles(),
		Sys:   sessionSysProcAttr(),
	}
	pid, err := startSession(exePath, argv, attr)
	if err != nil {
		return nil, errors.New("SESSION_TERMINAL_LAUNCH_FAILED: " + err.Error())
	}
	return &OpenResult{TerminalID: terminalID, Pid: pid}, nil
}
