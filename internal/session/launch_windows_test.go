//go:build windows

package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

var (
	testKernel32       = syscall.NewLazyDLL("kernel32.dll")
	testOpenProcess    = testKernel32.NewProc("OpenProcess")
	testGetExitCode    = testKernel32.NewProc("GetExitCodeProcess")
	testTerminateProc  = testKernel32.NewProc("TerminateProcess")
	testCloseHandleK32 = testKernel32.NewProc("CloseHandle")
)

const (
	processQueryLimited = 0x1000
	processTerminate    = 0x0001
	stillActive         = 259
)

func windowsProcessAlive(pid int) bool {
	h, _, err := testOpenProcess.Call(processQueryLimited, 0, uintptr(pid))
	if h == 0 {
		return false
	}
	defer testCloseHandleK32.Call(h)
	var code uint32
	r, _, _ := testGetExitCode.Call(h, uintptr(unsafe.Pointer(&code)))
	_ = err
	return r != 0 && code == stillActive
}

func windowsTerminate(pid int) {
	h, _, _ := testOpenProcess.Call(processTerminate, 0, uintptr(pid))
	if h != 0 {
		testTerminateProc.Call(h, 1)
		testCloseHandleK32.Call(h)
	}
}

// TestOpenNewConsoleLaunchOnWindows 锁定 Windows 启动契约：
// 目标终端可执行文件直启 + CREATE_NEW_CONSOLE（GUI 子系统可开窗），
// 不携带任何 std 句柄（Files 必须为 nil）——NUL 设备 stdio 会让交互
// shell 读到 EOF 立即退出（"呼叫后闪退"回归）。
func TestOpenNewConsoleLaunchOnWindows(t *testing.T) {
	root := sessionSandbox(t)
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}

	var gotExe string
	var gotArgv []string
	var gotAttr *os.ProcAttr
	original := startSession
	startSession = func(exePath string, argv []string, attr *os.ProcAttr) (int, error) {
		gotExe, gotArgv, gotAttr = exePath, argv, attr
		return 4242, nil
	}
	t.Cleanup(func() { startSession = original })

	powershellPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	cmdPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	cases := []struct {
		terminal string
		wantExe  string
		wantArgv []string
	}{
		{"powershell", powershellPath, []string{"powershell", "-NoLogo", "-NoExit"}},
		{"cmd", cmdPath, []string{"cmd.exe", "/K"}},
		{"bash", "", []string{"bash.exe", "--login", "-i"}},
	}
	// 无 Git for Windows 时 bash 不在终端清单里（WSL 的 System32\bash.exe
	// 已被 findGitBash 排除），用例按实际探测结果裁剪。
	detected := map[string]bool{}
	for _, terminal := range DetectTerminals() {
		detected[terminal.ID] = true
	}
	for _, tc := range cases {
		if !detected[tc.terminal] {
			continue
		}
		_, err := Open(context.Background(), root, tc.terminal)
		if err != nil {
			t.Fatalf("%s: %v", tc.terminal, err)
		}
		if tc.wantExe != "" && !strings.EqualFold(gotExe, tc.wantExe) {
			t.Fatalf("%s exe wrong: %q want %q", tc.terminal, gotExe, tc.wantExe)
		}
		if tc.wantExe == "" && !strings.HasSuffix(strings.ToLower(gotExe), "bash.exe") {
			t.Fatalf("%s exe wrong: %q", tc.terminal, gotExe)
		}
		if strings.Join(gotArgv, " ") != strings.Join(tc.wantArgv, " ") {
			t.Fatalf("%s argv wrong: %+v want %+v", tc.terminal, gotArgv, tc.wantArgv)
		}
		if gotAttr.Sys == nil || gotAttr.Sys.CreationFlags&createNewConsole == 0 {
			t.Fatalf("%s must carry CREATE_NEW_CONSOLE: %+v", tc.terminal, gotAttr.Sys)
		}
		if len(gotAttr.Files) != 0 {
			t.Fatalf("%s must not inherit std handles (new console owns its stdio): %+v", tc.terminal, gotAttr.Files)
		}
	}
}

// TestTerminalSessionStaysAlive 锁定"呼叫后闪退"回归：经产品启动链
// 启动的交互 PowerShell（-NoExit）必须持续存活——stdio 指向 NUL 设备
// 时它会在数百毫秒内读 EOF 退出（2026-11-23 用户报告的根因）。
func TestTerminalSessionStaysAlive(t *testing.T) {
	if testing.Short() {
		t.Skip("long: pops a transient console window")
	}
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if _, err := os.Stat(ps); err != nil {
		t.Skip("powershell not available")
	}
	attr := &os.ProcAttr{Dir: os.TempDir(), Env: os.Environ(), Sys: sessionSysProcAttr()}
	pid, err := startSession(ps, []string{"powershell", "-NoLogo", "-NoExit"}, attr)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	// 启动缓冲 1.5s 后观察 3s：闪退发生在数百毫秒内，存活 4.5s 即视为通过。
	time.Sleep(1500 * time.Millisecond)
	if !windowsProcessAlive(pid) {
		t.Fatalf("terminal exited within 1.5s — NUL-stdio flash-exit regression (pid %d)", pid)
	}
	time.Sleep(3 * time.Second)
	if !windowsProcessAlive(pid) {
		t.Fatalf("terminal did not stay alive for 4.5s — flash-exit regression (pid %d)", pid)
	}
	windowsTerminate(pid)
}
