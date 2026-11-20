//go:build windows

package session

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestOpenNewConsoleLaunchOnWindows 锁定 Windows 启动契约：
// 目标终端可执行文件直启 + CREATE_NEW_CONSOLE（GUI 子系统可开窗），
// 父侧 stdio 由 startSession 兜底为 DevNull（os.StartProcess 要求显式句柄）。
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
		if len(gotAttr.Files) != 3 {
			t.Fatalf("%s must carry 3 std handles (set by startSession): %d", tc.terminal, len(gotAttr.Files))
		}
	}
}
