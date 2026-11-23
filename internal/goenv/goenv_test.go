package goenv

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// isolate 把 go 用户配置与用户状态目录都指向临时区，绝不写真实环境。
func isolate(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("GOENV", filepath.Join(dir, "go-env"))
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	t.Setenv("APPDATA", filepath.Join(dir, "appdata"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(dir, "xdgdata"))
}

func haveGo(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not available")
	}
}

func TestReadReportsEditableAndPresets(t *testing.T) {
	isolate(t)
	haveGo(t)
	rep, err := Read(context.Background())
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if rep.GoVersion == "" || rep.GoBinary == "" {
		t.Fatalf("expected go version/binary, got %+v", rep)
	}
	found := false
	for _, k := range rep.Editable {
		if k == "GOPROXY" {
			found = true
		}
	}
	if !found || len(rep.Presets) == 0 {
		t.Fatalf("editable/presets missing: %+v", rep)
	}
}

func TestSetDryRunDoesNotWrite(t *testing.T) {
	isolate(t)
	haveGo(t)
	ctx := context.Background()
	before, err := Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Set(ctx, "GOPROXY", "https://goproxy.cn,direct", false, true, false)
	if err != nil {
		t.Fatalf("dryRun: %v", err)
	}
	if res.Change.To != "https://goproxy.cn,direct" || res.Change.Action != "set" {
		t.Fatalf("change wrong: %+v", res.Change)
	}
	after, _ := Read(ctx)
	if after.Values["GOPROXY"] != before.Values["GOPROXY"] {
		t.Fatalf("dryRun must not write: %q -> %q", before.Values["GOPROXY"], after.Values["GOPROXY"])
	}
}

func TestSetWriteAndUnsetRoundTrip(t *testing.T) {
	isolate(t)
	haveGo(t)
	ctx := context.Background()
	if _, err := Set(ctx, "GOPROXY", "direct", false, false, true); err != nil {
		t.Fatalf("set: %v", err)
	}
	rep, _ := Read(ctx)
	if rep.Values["GOPROXY"] != "direct" {
		t.Fatalf("expected direct, got %q", rep.Values["GOPROXY"])
	}
	if _, err := Set(ctx, "GOPROXY", "", true, false, true); err != nil {
		t.Fatalf("unset: %v", err)
	}
	rep, _ = Read(ctx)
	if rep.Values["GOPROXY"] == "direct" {
		t.Fatalf("unset should restore default, got %q", rep.Values["GOPROXY"])
	}
}

func TestSetRejectsNonEditableKey(t *testing.T) {
	isolate(t)
	haveGo(t)
	_, err := Set(context.Background(), "GOROOT", "x", false, true, false)
	if err == nil || !strings.Contains(err.Error(), "GO_ENV_KEY_NOT_EDITABLE") {
		t.Fatalf("expected GO_ENV_KEY_NOT_EDITABLE, got %v", err)
	}
}

func TestSetRequiresConfirm(t *testing.T) {
	isolate(t)
	haveGo(t)
	_, err := Set(context.Background(), "GOPROXY", "direct", false, false, false)
	if err == nil || !strings.Contains(err.Error(), "CONFIRMATION_REQUIRED") {
		t.Fatalf("expected CONFIRMATION_REQUIRED, got %v", err)
	}
}
