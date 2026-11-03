package session

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"dev-toolbox/internal/userstate"
)

func sessionSandbox(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, key := range []string{"HOME", "USERPROFILE", "APPDATA", "XDG_CONFIG_HOME"} {
		t.Setenv(key, root)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(root, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	return root
}

func writeUserState(t *testing.T, root string, selections map[string]userstate.Selection, installations ...userstate.Installation) {
	t.Helper()
	state := userstate.NewState()
	state.GlobalSelections = selections
	state.Installations = installations
	state.Revision = 1
	data, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "DevToolbox", "user-runtime-state.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestResolveContextInjectionAndSkips(t *testing.T) {
	root := sessionSandbox(t)
	nodeHome := filepath.Join(root, "node-v22")
	jdkHome := filepath.Join(root, "jdk-21")
	deadHome := filepath.Join(root, "gone")
	for _, dir := range []string{nodeHome, filepath.Join(jdkHome, "bin")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	writeUserState(t, root,
		map[string]userstate.Selection{
			"node":   {InstallationID: "node-1", Status: "active", Source: "user"},
			"java":   {InstallationID: "jdk-1", Status: "active", Source: "user"},
			"golang": {InstallationID: "go-1", Status: "active", Source: "user"},
			"gradle": {Status: "empty"},
		},
		userstate.Installation{ID: "node-1", Runtime: "node", Version: "22.23.3", LocalHome: nodeHome, Available: true},
		userstate.Installation{ID: "jdk-1", Runtime: "java", Version: "21.0.6", LocalHome: jdkHome, Available: true},
		userstate.Installation{ID: "go-1", Runtime: "golang", Version: "1.27", LocalHome: deadHome, Available: false},
	)

	preview, err := ResolveContext(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	byTool := map[string]ToolState{}
	for _, item := range preview.Tools {
		byTool[item.Tool] = item
	}
	if byTool["node"].Status != "injected" || byTool["node"].BinDir != nodeHome {
		t.Fatalf("node injection wrong: %+v", byTool["node"])
	}
	if byTool["java"].Status != "injected" || byTool["java"].BinDir != filepath.Join(jdkHome, "bin") {
		t.Fatalf("java injection wrong: %+v", byTool["java"])
	}
	if preview.JavaHome != jdkHome {
		t.Fatalf("JAVA_HOME wrong: %q", preview.JavaHome)
	}
	if byTool["gradle"].Status != "skipped" || byTool["gradle"].Reason != "未选择" {
		t.Fatalf("gradle skip wrong: %+v", byTool["gradle"])
	}
	if byTool["golang"].Status != "skipped" || !strings.Contains(byTool["golang"].Reason, "路径不可用") {
		t.Fatalf("golang skip wrong: %+v", byTool["golang"])
	}
	if len(preview.PathPrepends) != 2 {
		t.Fatalf("path prepends wrong: %+v", preview.PathPrepends)
	}
	// Git 身份预览存在（无配置时应为 none，不报错）
	if preview.GitIdentity.Resolved == "" {
		t.Fatal("git identity preview missing")
	}
	// cwd 校验
	if _, err := ResolveContext(context.Background(), filepath.Join(root, "no-such")); err == nil || !strings.HasPrefix(err.Error(), "SESSION_CWD_INVALID") {
		t.Fatalf("expected SESSION_CWD_INVALID, got %v", err)
	}
}

func TestBuildEnvPrependsPathAndJavaHome(t *testing.T) {
	parent := []string{"PATH=/usr/bin", "HOME=/home/u", "JAVA_HOME=/old"}
	env := BuildEnv(parent, []string{"/a", "/b"}, "/new/jdk")
	var path, java string
	for _, entry := range env {
		if strings.HasPrefix(entry, "PATH=") {
			path = entry
		}
		if strings.HasPrefix(entry, "JAVA_HOME=") {
			java = entry
		}
	}
	wantPath := "PATH=" + "/a" + string(os.PathListSeparator) + "/b" + string(os.PathListSeparator) + "/usr/bin"
	if path != wantPath {
		t.Fatalf("PATH wrong: %q want %q", path, wantPath)
	}
	if java != "JAVA_HOME=/new/jdk" {
		t.Fatalf("JAVA_HOME wrong: %q", java)
	}
	// 无 JAVA_HOME 注入时保留父值
	env = BuildEnv(parent, nil, "")
	for _, entry := range env {
		if strings.HasPrefix(entry, "JAVA_HOME=") && entry != "JAVA_HOME=/old" {
			t.Fatalf("parent JAVA_HOME must be kept: %q", entry)
		}
	}
}

func TestOpenBuildsArgvEnvAndCwd(t *testing.T) {
	root := sessionSandbox(t)
	nodeHome := filepath.Join(root, "node")
	if err := os.MkdirAll(nodeHome, 0o700); err != nil {
		t.Fatal(err)
	}
	writeUserState(t, root,
		map[string]userstate.Selection{"node": {InstallationID: "n1", Status: "active", Source: "user"}},
		userstate.Installation{ID: "n1", Runtime: "node", Version: "22", LocalHome: nodeHome, Available: true},
	)

	var gotExe string
	var gotArgv []string
	var gotAttr *os.ProcAttr
	original := startSession
	startSession = func(exePath string, argv []string, attr *os.ProcAttr) (int, error) {
		gotExe, gotArgv, gotAttr = exePath, argv, attr
		return 777, nil
	}
	t.Cleanup(func() { startSession = original })

	result, err := Open(context.Background(), root, "cmd")
	if err != nil {
		t.Fatal(err)
	}
	if result.Pid != 777 || result.TerminalID != "cmd" {
		t.Fatalf("result wrong: %+v", result)
	}
	// Windows 经 conhost 新控制台启动；Linux 直接启动。exe 与 argv
	// 语义跨平台断言：命令载体正确、终端保持打开、cwd/env 注入到位。
	joined := strings.Join(gotArgv, " ")
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(strings.ToLower(gotExe), "conhost.exe") {
			t.Fatalf("exe wrong: %q", gotExe)
		}
		for _, expect := range []string{"cmd.exe", "/K"} {
			if !strings.Contains(joined, expect) {
				t.Fatalf("argv missing %q: %+v", expect, gotArgv)
			}
		}
	} else {
		if !strings.HasSuffix(strings.ToLower(gotExe), "cmd") && !strings.HasSuffix(strings.ToLower(gotExe), "bash") {
			t.Fatalf("exe wrong: %q", gotExe)
		}
		if !strings.Contains(joined, "-i") {
			t.Fatalf("interactive flag missing: %+v", gotArgv)
		}
	}
	if gotAttr.Dir != root {
		t.Fatalf("cwd wrong: %q", gotAttr.Dir)
	}
	pathValue := ""
	for _, entry := range gotAttr.Env {
		if strings.HasPrefix(entry, "PATH=") {
			pathValue = entry
		}
	}
	if !strings.HasPrefix(pathValue, "PATH="+nodeHome) {
		t.Fatalf("PATH must start with injected bin: %q", pathValue)
	}
	// 全局状态在启动后必须未被修改（SL-05）
	stateAfter, err := userstate.Get()
	if err != nil || stateAfter.Revision != 1 {
		t.Fatalf("global state must be untouched: %+v %v", stateAfter, err)
	}
	// 未知终端
	if _, err := Open(context.Background(), root, "nope"); err == nil || !strings.HasPrefix(err.Error(), "SESSION_TERMINAL_NOT_FOUND") {
		t.Fatalf("expected terminal not found, got %v", err)
	}
}
