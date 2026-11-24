package clientsettings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// isolate 把设置文件重定向到临时目录，避免触碰真实用户配置。
func isolate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	t.Setenv("DEVTOOLBOX_SETTINGS", path)
	return path
}

func TestNewDefaults(t *testing.T) {
	isolate(t)
	got, err := Get()
	if err != nil {
		t.Fatal(err)
	}
	if got.Schema != storeVersion || got.Revision != 1 {
		t.Fatalf("unexpected schema/revision: %+v", got)
	}
	if got.Theme != ThemeSystem {
		t.Fatalf("expected system theme, got %s", got.Theme)
	}
	if len(got.SourceProjects) != 0 || len(got.Snapshots) != 0 {
		t.Fatalf("expected empty lists: %+v", got)
	}
}

func TestSetThemeRejectsBadAccent(t *testing.T) {
	isolate(t)
	if _, err := SetTheme(ThemeCustom, "red", 1); err == nil {
		t.Fatal("expected accent rejection")
	}
	if _, err := SetTheme("neon", "", 1); err == nil {
		t.Fatal("expected theme rejection")
	}
}

func TestSetThemeCustomPersistsAccent(t *testing.T) {
	path := isolate(t)
	got, err := SetTheme(ThemeCustom, "#A34B2A", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Theme != ThemeCustom || got.CustomAccent != "#a34b2a" {
		t.Fatalf("unexpected theme payload: %+v", got)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var stored Settings
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.CustomAccent != "#a34b2a" {
		t.Fatalf("accent not persisted: %+v", stored)
	}
	// 切回系统主题应清除自定义色
	got, err = SetTheme(ThemeSystem, "", got.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if got.CustomAccent != "" {
		t.Fatalf("custom accent should be cleared, got %q", got.CustomAccent)
	}
}

func TestSetThemeRevisionConflict(t *testing.T) {
	isolate(t)
	if _, err := SetTheme(ThemeDark, "", 99); err != ErrConflict {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestSourceProjectRoundTrip(t *testing.T) {
	path := isolate(t)
	root := t.TempDir()
	projectRoot := filepath.Join(root, "repo")
	if err := os.MkdirAll(projectRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := AddSourceProject(projectRoot, "演示仓库", "", BindingLocal, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.SourceProjects) != 1 || got.SourceProjects[0].Binding != BindingLocal {
		t.Fatalf("unexpected registration: %+v", got.SourceProjects)
	}
	if _, err := AddSourceProject(projectRoot, "重复", "", BindingShared, got.Revision); err == nil {
		t.Fatal("duplicate path should be rejected")
	}
	back, removed, err := RemoveSourceProject(projectRoot, got.Revision)
	if err != nil || !removed {
		t.Fatalf("remove failed: removed=%v err=%v", removed, err)
	}
	if len(back.SourceProjects) != 0 {
		t.Fatalf("expected empty list, got %+v", back.SourceProjects)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("settings file missing: %v", err)
	}
}

func TestSharedRejectsPrivateHomePaths(t *testing.T) {
	isolate(t)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	privatePath := filepath.Join(home, "secret-project")
	if _, err := AddSourceProject(privatePath, "", "", BindingShared, 1); err == nil {
		t.Fatal("shared entries must not point inside the user home directory")
	}
}

func TestSourceProjectRejectsURLs(t *testing.T) {
	isolate(t)
	if _, err := AddSourceProject("https://github.com/xyztony999/dev-toolbox", "", "", BindingShared, 1); err == nil {
		t.Fatal("URL paths must be rejected")
	}
	if _, err := AddSourceProject("", "", "", BindingLocal, 1); err == nil {
		t.Fatal("empty paths must be rejected")
	}
}

func TestSnapshotExportImportIdempotent(t *testing.T) {
	path := isolate(t)
	sharedRoot := t.TempDir()
	settings, err := AddSourceProject(sharedRoot, "共享", "", BindingShared, 1)
	if err != nil {
		t.Fatal(err)
	}
	snap, err := ExportSnapshot("demo", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.SourceProjects) != 1 {
		t.Fatalf("expected only shared entries, got %+v", snap.SourceProjects)
	}
	data, err := SnapshotBytes(snap)
	if err != nil {
		t.Fatal(err)
	}
	// 先切走主题，再导入
	if _, err := SetTheme(ThemeLight, "", settings.Revision); err != nil {
		t.Fatal(err)
	}
	imported, record, err := ImportSnapshot(data, 0)
	if err != nil {
		t.Fatal(err)
	}
	if imported.Revision <= settings.Revision {
		t.Fatalf("revision must advance: %d -> %d", settings.Revision, imported.Revision)
	}
	if record.Applied == 0 {
		t.Fatalf("expected applied count, got %+v", record)
	}
	// 同一份快照重复导入：幂等，修订号不变且不新增记录
	again, record2, err := ImportSnapshot(data, imported.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if again.Revision != imported.Revision {
		t.Fatalf("revision must not advance on repeat import: %d -> %d", imported.Revision, again.Revision)
	}
	if len(again.Snapshots) != 1 || record2.SourceDigest != snap.Digest {
		t.Fatalf("import must be idempotent: %+v", record2)
	}
	// 备份副本存在
	if _, err := os.Stat(record.BackupPath); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	_ = path
}

func TestSnapshotRejectsTamperedDigest(t *testing.T) {
	isolate(t)
	snap, err := ExportSnapshot("demo", true)
	if err != nil {
		t.Fatal(err)
	}
	data, err := SnapshotBytes(snap)
	if err != nil {
		t.Fatal(err)
	}
	// 篡改参与摘要的字段（主题）后必须被拒绝；标签不参与摘要
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	payload["theme"] = ThemeDark
	tamperedBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	tampered := string(tamperedBytes)
	if tampered == string(data) {
		t.Fatal("tamper did not apply")
	}
	if _, err := ParseSnapshot([]byte(tampered)); err == nil {
		t.Fatal("tampered snapshot must be rejected")
	}
}

func TestSnapshotRejectsForeignSchema(t *testing.T) {
	isolate(t)
	if _, err := ParseSnapshot([]byte(`{"schema":99,"theme":"dark"}`)); err == nil {
		t.Fatal("foreign schema must be rejected")
	}
}

func TestCorruptSettings(t *testing.T) {
	path := isolate(t)
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Get(); err != ErrCorrupt {
		t.Fatalf("expected ErrCorrupt, got %v", err)
	}
}
