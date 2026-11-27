package clientsettings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

// isolateProjects 重定向登记项目与共享要求载荷，保证快照测试不读真实 projects.json。
func isolateProjects(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "projects.json")
	t.Setenv("DEVTOOLBOX_PROJECT_STORE", path)
	t.Setenv("DEVTOOLBOX_SHARED_REQUIREMENTS", filepath.Join(dir, "shared-requirements.json"))
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
	if len(got.Snapshots) != 0 {
		t.Fatalf("expected empty snapshots: %+v", got)
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

func TestExportSnapshotHasNoLocalArtifacts(t *testing.T) {
	isolate(t)
	isolateProjects(t)
	snap, err := ExportSnapshot("demo")
	if err != nil {
		t.Fatal(err)
	}
	data, err := SnapshotBytes(snap)
	if err != nil {
		t.Fatal(err)
	}
	payload := string(data)
	for forbidden := range map[string]struct{}{
		"repositories": {},
		"rootPath":     {},
		"remoteUrl":    {},
		"userEmail":    {},
	} {
		if strings.Contains(payload, forbidden) {
			t.Fatalf("snapshot must not leak %q: %s", forbidden, payload)
		}
	}
	if len(snap.Requirements) != 0 {
		t.Fatalf("expected no requirements for a machine with no shared projects, got %+v", snap.Requirements)
	}
}

func TestSnapshotRejectsForeignSchema(t *testing.T) {
	isolate(t)
	if _, err := ParseSnapshot([]byte(`{"schema":99,"theme":"dark"}`)); err == nil {
		t.Fatal("foreign schema must be rejected")
	}
}

func TestSnapshotRejectsTamperedDigest(t *testing.T) {
	isolate(t)
	snap, err := ExportSnapshot("demo")
	if err != nil {
		t.Fatal(err)
	}
	data, err := SnapshotBytes(snap)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	payload["theme"] = ThemeDark
	tampered, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseSnapshot(tampered); err == nil {
		t.Fatal("tampered snapshot must be rejected")
	}
}

func TestSnapshotImportIdempotent(t *testing.T) {
	isolate(t)
	isolateProjects(t)
	snap, err := ExportSnapshot("demo")
	if err != nil {
		t.Fatal(err)
	}
	data, err := SnapshotBytes(snap)
	if err != nil {
		t.Fatal(err)
	}
	imported, record, err := ImportSnapshot(data, 0)
	if err != nil {
		t.Fatal(err)
	}
	if record.Applied == 0 {
		t.Fatalf("expected applied count, got %+v", record)
	}
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
	if _, err := os.Stat(record.BackupPath); err != nil {
		t.Fatalf("backup missing: %v", err)
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
