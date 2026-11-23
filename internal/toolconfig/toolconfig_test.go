package toolconfig

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolateSettings(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.xml")
	t.Setenv("DEVTOOLBOX_MAVEN_SETTINGS", path)
	return path
}

func TestMavenMirrorUpsertPreservesUserMirrors(t *testing.T) {
	path := isolateSettings(t)
	user := "<?xml version=\"1.0\"?>\n<settings>\n  <mirrors>\n    <mirror>\n      <id>mine</id>\n      <mirrorOf>central</mirrorOf>\n      <url>https://mine.example/</url>\n    </mirror>\n  </mirrors>\n</settings>\n"
	if err := os.WriteFile(path, []byte(user), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeMavenMirror("https://maven.aliyun.com/repository/public", false); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, _ := os.ReadFile(path)
	got := string(data)
	if !strings.Contains(got, "<id>dev-toolbox</id>") || !strings.Contains(got, "maven.aliyun.com") {
		t.Fatalf("managed mirror missing: %s", got)
	}
	if !strings.Contains(got, "<id>mine</id>") {
		t.Fatalf("user mirror must be preserved: %s", got)
	}
	if _, err := os.Stat(path + ".devtoolbox.bak"); err != nil {
		t.Fatalf("backup expected: %v", err)
	}
	// 幂等替换：再写一次只应有一个托管块
	if err := writeMavenMirror("https://repo.huaweicloud.com/repository/maven/", false); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if n := strings.Count(string(data), "<id>dev-toolbox</id>"); n != 1 {
		t.Fatalf("expected exactly one managed mirror, got %d", n)
	}
	rep, err := readMaven()
	if err != nil {
		t.Fatal(err)
	}
	if rep.Items[0].Value != "https://repo.huaweicloud.com/repository/maven/" {
		t.Fatalf("read value wrong: %+v", rep.Items[0])
	}
}

func TestMavenMirrorRemove(t *testing.T) {
	path := isolateSettings(t)
	if err := writeMavenMirror("https://maven.aliyun.com/repository/public", false); err != nil {
		t.Fatal(err)
	}
	if err := writeMavenMirror("", true); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Contains(string(data), "<id>dev-toolbox</id>") {
		t.Fatalf("managed mirror should be removed: %s", data)
	}
	rep, _ := readMaven()
	if rep.Items[0].Value != "" {
		t.Fatalf("read after remove should be empty: %+v", rep.Items[0])
	}
}

func TestSetMavenDryRunDoesNotWrite(t *testing.T) {
	path := isolateSettings(t)
	res, err := Set(context.Background(), "maven", "mirror.url", "https://x.example/", false, true, false)
	if err != nil {
		t.Fatalf("dryRun: %v", err)
	}
	if res.Change.To != "https://x.example/" || res.Change.Action != "set" {
		t.Fatalf("change wrong: %+v", res.Change)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("dryRun must not create settings.xml")
	}
}

func TestSetMavenRequiresConfirm(t *testing.T) {
	isolateSettings(t)
	_, err := Set(context.Background(), "maven", "mirror.url", "https://x.example/", false, false, false)
	if err == nil || !strings.Contains(err.Error(), "CONFIRMATION_REQUIRED") {
		t.Fatalf("expected CONFIRMATION_REQUIRED, got %v", err)
	}
}

func TestReadOnlyToolsNotEditable(t *testing.T) {
	for _, tc := range []struct{ tool, key string }{
		{"java", "JAVA_HOME"}, {"androidsdk", "ANDROID_HOME"}, {"gradle", "GRADLE_USER_HOME"}, {"rust", "cargo.mirror"},
	} {
		rep, err := Read(context.Background(), tc.tool)
		if err != nil {
			t.Fatalf("%s read: %v", tc.tool, err)
		}
		if len(rep.Items) == 0 {
			t.Fatalf("%s expected read-only item", tc.tool)
		}
		if rep.Items[0].Editable {
			t.Fatalf("%s/%s must be read-only", tc.tool, tc.key)
		}
		if _, err := Set(context.Background(), tc.tool, tc.key, "x", false, true, false); err == nil || !strings.Contains(err.Error(), "NOT_EDITABLE") {
			t.Fatalf("%s set must be refused, got %v", tc.tool, err)
		}
	}
}

func TestGolangDelegatesToGoenv(t *testing.T) {
	if _, err := lookPathSafe(); err != nil {
		t.Skip("go not available")
	}
	dir := t.TempDir()
	t.Setenv("GOENV", filepath.Join(dir, "go-env"))
	rep, err := Read(context.Background(), "golang")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range rep.Items {
		if item.Key == "GOPROXY" && item.Editable {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected editable GOPROXY via goenv delegate: %+v", rep.Items)
	}
}

func lookPathSafe() (string, error) {
	p := lookPath("go")
	if p == "" {
		return "", os.ErrNotExist
	}
	return p, nil
}
