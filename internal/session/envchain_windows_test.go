//go:build windows

package session

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestStartChainInheritsEnv 实证产品启动链（os.StartProcess →
// conhost → 新窗口 powershell）的环境继承：新窗口进程执行
// EncodedCommand 写出注入的变量值。脚本用 base64 载体，无命令拼接。
func TestStartChainInheritsEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("long: pops a transient console window")
	}
	outDir := strings.ReplaceAll(t.TempDir(), "\\", "/")
	outFile := outDir + "/env.txt"
	script := `Set-Content -Path '` + outFile + `' -Value $env:DIAG_JAVA_HOME; exit`
	// -EncodedCommand 要求 UTF-16LE 的 base64（ASCII 脚本也必须转码）
	utf16 := make([]byte, len(script)*2)
	for i, ch := range []byte(script) {
		utf16[i*2] = ch
	}
	encoded := base64.StdEncoding.EncodeToString(utf16)

	env := BuildEnv(os.Environ(), nil, "")
	env = append(env, "DIAG_JAVA_HOME=D:/diag-jdk-21")
	attr := &os.ProcAttr{Dir: t.TempDir(), Env: env, Sys: sessionSysProcAttr()}

	info := TerminalInfo{ID: "diag", Label: "diag"}
	info.ExePath = filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	// 直接以 powershell 为目标验证继承（产品链路中它由 conhost 启动，
	// 继承语义相同：均为 ProcAttr.Env 一次传递）
	argv := []string{"powershell", "-NoLogo", "-NoProfile", "-EncodedCommand", encoded}
	if _, err := startSession(info.ExePath, argv, attr); err != nil {
		t.Fatalf("start chain: %v", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if data, err := os.ReadFile(outFile); err == nil {
			if got := strings.TrimSpace(string(data)); got != `D:/diag-jdk-21` {
				t.Fatalf("injected var not inherited: %q", got)
			}
			return
		}
		time.Sleep(300 * time.Millisecond)
	}
	t.Fatal("timeout: session window did not write env file")
}
