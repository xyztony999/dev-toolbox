package runtimeinstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dev-toolbox/internal/userstate"
)

func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("APPDATA", filepath.Join(dir, "appdata"))
	t.Setenv("LOCALAPPDATA", filepath.Join(dir, "localappdata"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "xdgdata"))
	return dir
}

func register(t *testing.T, dir, source string) (userstate.State, userstate.Installation) {
	t.Helper()
	home := filepath.Join(dir, "go-"+source)
	if err := os.MkdirAll(filepath.Join(home, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	state, _, err := userstate.RegisterInstallations([]userstate.Installation{
		{Runtime: "golang", Version: "1.2.3", LocalHome: home, Source: source},
	}, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Installations) == 0 {
		t.Fatal("no installation registered")
	}
	return state, state.Installations[0]
}

func TestRemoveDownloadedDeletesOwnDir(t *testing.T) {
	dir := isolate(t)
	root, err := RuntimeRoot()
	if err != nil {
		t.Fatal(err)
	}
	// 把 downloaded 安装放进产品自有根目录内
	home := filepath.Join(root, "go-own")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	state, _, err := userstate.RegisterInstallations([]userstate.Installation{
		{Runtime: "golang", Version: "1.2.3", LocalHome: home, Source: "downloaded"},
	}, 0, true)
	if err != nil {
		t.Fatal(err)
	}
	inst := state.Installations[0]
	res, err := RemoveInstallation(inst.ID, state.Revision, true, false, true)
	if err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !res.DirDeleted || !res.Removed {
		t.Fatalf("expected dir deleted + removed: %+v", res)
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("own dir should be gone")
	}
	_ = dir
}

func TestRemoveExternalNeverDeletesDir(t *testing.T) {
	dir := isolate(t)
	state, inst := register(t, dir, "user") // 外部目录，不在 RuntimeRoot 内
	_, err := RemoveInstallation(inst.ID, state.Revision, true, false, true)
	if err == nil || !strings.Contains(err.Error(), "RUNTIME_REMOVE_NOT_OWNED") {
		t.Fatalf("expected RUNTIME_REMOVE_NOT_OWNED, got %v", err)
	}
	// deleteFiles=false 只注销，目录保留
	res, err := RemoveInstallation(inst.ID, state.Revision, false, false, true)
	if err != nil {
		t.Fatalf("unregister: %v", err)
	}
	if res.DirDeleted || !res.Removed {
		t.Fatalf("expected record removed, dir kept: %+v", res)
	}
	if _, err := os.Stat(inst.LocalHome); err != nil {
		t.Fatalf("external dir must remain: %v", err)
	}
}

func TestRemoveDryRunNoMutation(t *testing.T) {
	dir := isolate(t)
	state, inst := register(t, dir, "user")
	res, err := RemoveInstallation(inst.ID, state.Revision, false, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.Removed || res.Preview.OwnedByProduct {
		t.Fatalf("dryRun must not mutate and external not owned: %+v", res)
	}
	after, _ := userstate.Get()
	if len(after.Installations) != len(state.Installations) {
		t.Fatal("dryRun removed record")
	}
}

func TestRemoveClearsGlobalDefault(t *testing.T) {
	dir := isolate(t)
	state, inst := register(t, dir, "user")
	if _, err := userstate.SetGlobalDefault("golang", inst.ID, state.Revision, true); err != nil {
		t.Fatal(err)
	}
	cur, _ := userstate.Get()
	res, err := RemoveInstallation(inst.ID, cur.Revision, false, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if !res.WasDefault {
		t.Fatal("expected wasDefault")
	}
	if _, ok := res.State.GlobalSelections["golang"]; ok {
		t.Fatal("default selection should be cleared")
	}
}
