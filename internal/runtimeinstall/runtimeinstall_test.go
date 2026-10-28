package runtimeinstall

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"dev-toolbox/internal/userstate"
)

// sandboxEnv redirects every per-user directory the install service touches
// into throwaway temp dirs; tests never write real user state.
func sandboxEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
}

// stubFetcher points the fixed provider hosts at a local httptest server.
func stubFetcher(t *testing.T, handler http.Handler) *fetcher {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &fetcher{
		client: srv.Client(),
		rewrite: func(raw string) string {
			parsed, err := url.Parse(raw)
			if err != nil {
				return raw
			}
			return srv.URL + parsed.Path
		},
	}
}

func useStubFetcher(t *testing.T, handler http.Handler) {
	t.Helper()
	stub := stubFetcher(t, handler)
	previous := fetcherFor
	fetcherFor = func() *fetcher { return stub }
	t.Cleanup(func() { fetcherFor = previous })
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// fakeProvider is a deterministic provider used for manager lifecycle tests.
type fakeProvider struct {
	block chan struct{}
}

func (fakeProvider) option() InstallOption {
	return InstallOption{
		Tool: "fake", ProviderID: "fake",
		Mirrors:            []MirrorOption{{ID: "official", Label: "fake"}},
		VersionSelectors:   []string{"latest"},
		SupportedPlatforms: []string{runtime.GOOS},
		SupportedArchs:     []string{runtime.GOARCH},
	}
}

func (fakeProvider) preview(ctx context.Context, f *fetcher, selector, mirror string) (*InstallPlan, error) {
	if selector != "latest" {
		return nil, errors.New("RUNTIME_VERSION_UNAVAILABLE: fake only offers latest")
	}
	root, err := RuntimeRoot()
	if err != nil {
		return nil, err
	}
	srcs := []pkgSource{{Name: "fake", FileName: "fake-1.0.zip", URL: "https://nodejs.org/fake/fake-1.0.zip", Algo: "sha256", Sum: sha256Hex(fakeArchiveBytes())}}
	return assemblePlan(ctx, f, "fake", "fake", mirror, selector, "1.0", runtime.GOOS, runtime.GOARCH, filepath.Join(root, "fake-1.0"), srcs)
}

func (fakeProvider) sources(plan *InstallPlan) ([]pkgSource, error) {
	return planSources(plan)
}

func (fp fakeProvider) install(ctx context.Context, f *fetcher, plan *InstallPlan, run *installRun) (userstate.Installation, error) {
	if fp.block != nil {
		select {
		case <-ctx.Done():
			return userstate.Installation{}, errors.New("RUNTIME_INSTALL_CANCELLED")
		case <-fp.block:
		}
	}
	run.report(InstallProgress{Phase: StateExtracting, Package: "fake"})
	binary := "fake.exe"
	if runtime.GOOS != "windows" {
		binary = "fake"
	}
	if err := os.MkdirAll(filepath.Join(plan.TargetHome, "bin"), 0o700); err != nil {
		return userstate.Installation{}, err
	}
	if err := os.WriteFile(filepath.Join(plan.TargetHome, "bin", binary), []byte("#!/bin/sh\n"), 0o700); err != nil {
		return userstate.Installation{}, err
	}
	run.mark("fake", "done", nil)
	return userstate.Installation{
		Runtime: "fake", Version: "1.0", LocalHome: plan.TargetHome,
		BinaryPath: filepath.Join(plan.TargetHome, "bin", binary),
		Source:     "downloaded", Label: "Fake 1.0", Available: true,
	}, nil
}

func (fakeProvider) checkTarget(plan *InstallPlan) error {
	return defaultCheckTarget(plan)
}

var fakeArchiveCache []byte

func fakeArchiveBytes() []byte {
	if fakeArchiveCache != nil {
		return fakeArchiveCache
	}
	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)
	entry, err := writer.Create("fake-1.0/bin/fake.txt")
	if err != nil {
		panic(err)
	}
	if _, err := entry.Write([]byte("placeholder")); err != nil {
		panic(err)
	}
	if err := writer.Close(); err != nil {
		panic(err)
	}
	fakeArchiveCache = buf.Bytes()
	return fakeArchiveCache
}

func fakeHandler(t *testing.T) http.Handler {
	t.Helper()
	archive := fakeArchiveBytes()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fake/fake-1.0.zip":
			if r.Method == http.MethodHead {
				w.Header().Set("Content-Length", fmt.Sprint(len(archive)))
				return
			}
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	})
}

func TestManagerFullFlowRegistersDownloadedInstallation(t *testing.T) {
	sandboxEnv(t)
	useStubFetcher(t, fakeHandler(t))
	providers["fake"] = fakeProvider{}
	t.Cleanup(func() { delete(providers, "fake") })

	m := NewManager()
	plan, err := m.Preview(context.Background(), "fake", "latest", "", 0)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if plan.PlanDigest == "" || plan.PlanID == "" || len(plan.Packages) != 1 {
		t.Fatalf("incomplete plan: %+v", plan)
	}
	if plan.Packages[0].Checksum != sha256Hex(fakeArchiveBytes()) {
		t.Fatalf("checksum mismatch in plan")
	}
	if _, err := m.Start(plan.PlanID, plan.PlanDigest, 0, false); err == nil || taskError(err).Code != "RUNTIME_CONFIRMATION_REQUIRED" {
		t.Fatalf("expected confirmation required, got %v", err)
	}
	if _, err := m.Start(plan.PlanID, plan.PlanDigest, 999, true); err == nil || taskError(err).Code != "RUNTIME_REVISION_CONFLICT" {
		t.Fatalf("expected revision conflict, got %v", err)
	}
	if _, err := m.Start(plan.PlanID, "deadbeef", 0, true); err == nil || taskError(err).Code != "RUNTIME_PLAN_STALE" {
		t.Fatalf("expected stale plan, got %v", err)
	}
	task, err := m.Start(plan.PlanID, plan.PlanDigest, 0, true)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	// Resubmitting the same active plan returns the existing task.
	if again, err := m.Start(plan.PlanID, plan.PlanDigest, 0, true); err != nil || again.TaskID != task.TaskID {
		t.Fatalf("resubmit should return existing task, got %v %v", again, err)
	}
	final, err := m.WaitTerminal(task.TaskID, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if final.State != StateComplete || final.InstallationID == "" {
		t.Fatalf("expected complete task, got %+v", final)
	}
	state, err := userstate.Get()
	if err != nil {
		t.Fatalf("userstate: %v", err)
	}
	found := false
	for _, item := range state.Installations {
		if item.Runtime == "fake" {
			found = true
			if item.Source != "downloaded" || !item.Available {
				t.Fatalf("unexpected installation record %+v", item)
			}
		}
	}
	if !found {
		t.Fatal("downloaded installation was not registered")
	}
	// The target exists now; replaying the plan must be a conflict, never an overwrite.
	if _, err := m.Start(plan.PlanID, plan.PlanDigest, 0, true); err == nil || taskError(err).Code != "RUNTIME_TARGET_CONFLICT" {
		t.Fatalf("expected target conflict on replay, got %v", err)
	}
}

func TestManagerExpiryAndUnknownTool(t *testing.T) {
	sandboxEnv(t)
	useStubFetcher(t, fakeHandler(t))
	providers["fake"] = fakeProvider{}
	t.Cleanup(func() { delete(providers, "fake") })

	m := NewManager()
	if _, err := m.Preview(context.Background(), "nope", "latest", "", 0); err == nil || taskError(err).Code != "RUNTIME_PROVIDER_UNSUPPORTED" {
		t.Fatalf("expected unsupported provider, got %v", err)
	}
	plan, err := m.Preview(context.Background(), "fake", "latest", "", 0)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m.mu.Lock()
	expired := m.plans[plan.PlanID]
	expired.ExpiresAt = time.Now().Add(-time.Minute)
	m.plans[plan.PlanID] = expired
	m.mu.Unlock()
	if _, err := m.Start(plan.PlanID, plan.PlanDigest, 0, true); err == nil || taskError(err).Code != "RUNTIME_PLAN_EXPIRED" {
		t.Fatalf("expected expired plan, got %v", err)
	}
}

func TestManagerCancelReachesTerminalState(t *testing.T) {
	sandboxEnv(t)
	useStubFetcher(t, fakeHandler(t))
	block := make(chan struct{})
	providers["fake"] = fakeProvider{block: block}
	t.Cleanup(func() { delete(providers, "fake"); close(block) })

	m := NewManager()
	plan, err := m.Preview(context.Background(), "fake", "latest", "", 0)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	task, err := m.Start(plan.PlanID, plan.PlanDigest, 0, true)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if _, err := m.Cancel(task.TaskID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	final, err := m.WaitTerminal(task.TaskID, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if final.State != StateCancelled {
		t.Fatalf("expected cancelled, got %s", final.State)
	}
	if _, err := os.Lstat(plan.TargetHome); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled install must not publish a target: %v", err)
	}
}

func TestManagerPartialStateWhenOnePackageFails(t *testing.T) {
	sandboxEnv(t)
	useStubFetcher(t, http.NotFoundHandler())
	providers["fakepartial"] = partialFakeProvider{}
	t.Cleanup(func() { delete(providers, "fakepartial") })

	m := NewManager()
	plan, err := m.Preview(context.Background(), "fakepartial", "latest", "", 0)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	task, err := m.Start(plan.PlanID, plan.PlanDigest, 0, true)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	final, err := m.WaitTerminal(task.TaskID, 5*time.Millisecond)
	if err != nil {
		t.Fatalf("wait: %v", err)
	}
	if final.State != StatePartial {
		t.Fatalf("expected partial, got %s (%+v)", final.State, final)
	}
	if _, err := os.Lstat(plan.TargetHome); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("failed install must not publish a target: %v", err)
	}
}

type partialFakeProvider struct{}

func (partialFakeProvider) option() InstallOption {
	return InstallOption{
		Tool: "fakepartial", ProviderID: "fakepartial",
		VersionSelectors:   []string{"latest"},
		SupportedPlatforms: []string{runtime.GOOS},
		SupportedArchs:     []string{runtime.GOARCH},
	}
}

func (partialFakeProvider) preview(ctx context.Context, f *fetcher, selector, mirror string) (*InstallPlan, error) {
	root, err := RuntimeRoot()
	if err != nil {
		return nil, err
	}
	srcs := []pkgSource{
		{Name: "a", FileName: "a.zip", URL: "https://nodejs.org/a.zip", Algo: "sha256", Sum: sha256Hex([]byte("a"))},
		{Name: "b", FileName: "b.zip", URL: "https://nodejs.org/b.zip", Algo: "sha256", Sum: sha256Hex([]byte("b"))},
	}
	return assemblePlan(ctx, f, "fakepartial", "fakepartial", mirror, selector, "1", runtime.GOOS, runtime.GOARCH, filepath.Join(root, "fakepartial-1"), srcs)
}

func (partialFakeProvider) sources(plan *InstallPlan) ([]pkgSource, error) {
	return planSources(plan)
}

func (partialFakeProvider) install(ctx context.Context, f *fetcher, plan *InstallPlan, run *installRun) (userstate.Installation, error) {
	run.mark("a", "done", nil)
	err := errors.New("RUNTIME_DOWNLOAD_FAILED: second package missing")
	run.mark("b", "failed", taskError(err))
	return userstate.Installation{}, err
}

func (partialFakeProvider) checkTarget(plan *InstallPlan) error {
	return defaultCheckTarget(plan)
}

func TestExtractRejectsTraversalAndSymlinks(t *testing.T) {
	sandboxEnv(t)
	root := t.TempDir()

	buf := new(bytes.Buffer)
	writer := zip.NewWriter(buf)
	entry, err := writer.Create("../evil.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("nope"))
	_ = writer.Close()
	zipPath := filepath.Join(root, "traversal.zip")
	if err := os.WriteFile(zipPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := extractSingleRoot(zipPath, filepath.Join(root, "out1")); err == nil {
		t.Fatal("traversal zip must be rejected")
	}

	buf = new(bytes.Buffer)
	writer = zip.NewWriter(buf)
	symlinkHeader := &zip.FileHeader{Name: "fake-1.0/link", Method: zip.Store, Modified: time.Now()}
	symlinkHeader.SetMode(os.ModeSymlink | 0o777)
	entry, err = writer.CreateHeader(symlinkHeader)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("/etc/passwd"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(root, "symlink.zip")
	if err := os.WriteFile(symlinkPath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := extractSingleRoot(symlinkPath, filepath.Join(root, "out2")); err == nil {
		t.Fatal("symlink zip must be rejected")
	}
}

func TestExtractSingleRootTarGz(t *testing.T) {
	sandboxEnv(t)
	root := t.TempDir()
	buf := new(bytes.Buffer)
	gz := gzip.NewWriter(buf)
	archive := tar.NewWriter(gz)
	writeFile := func(name, content string) {
		t.Helper()
		if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.WriteHeader(&tar.Header{Name: "tool-1", Mode: 0o700, Typeflag: tar.TypeDir}); err != nil {
		t.Fatal(err)
	}
	writeFile("tool-1/bin/tool", "#!/bin/sh")
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	archivePath := filepath.Join(root, "tool.tar.gz")
	if err := os.WriteFile(archivePath, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	rootDir, err := extractSingleRoot(archivePath, filepath.Join(root, "out"))
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	if filepath.Base(rootDir) != "tool-1" {
		t.Fatalf("unexpected root %q", rootDir)
	}
}

func TestNodePreviewResolvesVersionAndChecksum(t *testing.T) {
	sandboxEnv(t)
	sums := strings.Join([]string{
		strings.Repeat("a", 64) + "  node-v22.14.0-linux-x64.tar.xz",
		strings.Repeat("b", 64) + "  node-v22.14.0-win-x64.zip",
		strings.Repeat("c", 64) + "  node-v22.14.0-linux-arm64.tar.xz",
		strings.Repeat("d", 64) + "  node-v22.14.0-win-arm64.zip",
	}, "\n")
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dist/index.json":
			_, _ = w.Write([]byte(`[{"version":"v22.14.0"},{"version":"v22.13.1"},{"version":"v20.18.0"}]`))
		case "/dist/v22.14.0/SHASUMS256.txt":
			_, _ = w.Write([]byte(sums))
		default:
			http.NotFound(w, r)
		}
	})
	p := nodeProvider{}
	plan, err := p.preview(context.Background(), stubFetcher(t, handler), "22", "official")
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if plan.ResolvedVersion != "22.14.0" {
		t.Fatalf("unexpected version %q", plan.ResolvedVersion)
	}
	if _, _, _, err := nodePlatform(); err != nil {
		t.Skipf("node provider unsupported on this host: %v", err)
	}
	checksums := map[string]string{
		"node-v22.14.0-linux-x64.tar.xz":   strings.Repeat("a", 64),
		"node-v22.14.0-win-x64.zip":        strings.Repeat("b", 64),
		"node-v22.14.0-linux-arm64.tar.xz": strings.Repeat("c", 64),
		"node-v22.14.0-win-arm64.zip":      strings.Repeat("d", 64),
	}
	got := plan.Packages[0].Checksum
	want := checksums[plan.Packages[0].FileName]
	if got != want || want == "" {
		t.Fatalf("checksum for %s: got %q want %q", plan.Packages[0].FileName, got, want)
	}

	noChecksum := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/dist/index.json" {
			_, _ = w.Write([]byte(`[{"version":"v22.14.0"}]`))
			return
		}
		http.NotFound(w, r)
	})
	if _, err := p.preview(context.Background(), stubFetcher(t, noChecksum), "22", "official"); err == nil || taskError(err).Code != "RUNTIME_CHECKSUM_MISSING" {
		t.Fatalf("expected checksum missing, got %v", err)
	}
	if _, err := p.preview(context.Background(), stubFetcher(t, handler), "abc", "official"); err == nil || taskError(err).Code != "RUNTIME_VERSION_UNAVAILABLE" {
		t.Fatalf("expected invalid selector, got %v", err)
	}
}

func TestAndroidManifestChecksumGate(t *testing.T) {
	manifest := []byte(`<remotePackage path="platform-tools"><archives>` +
		`<archive><size>10</size><checksum>ABCD</checksum><url>other.zip</url></archive>` +
		`<archive><size>12</size><checksum>` + sha256Hex([]byte("pt")) + `</checksum><url>platform-tools-latest-windows.zip</url></archive>` +
		`</archives></remotePackage>`)
	entries, err := manifestArchives(manifest, "platform-tools")
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	algo, size, err := manifestChecksum(entries, "platform-tools-latest-windows.zip")
	if err != nil || algo != "sha256" || size != 12 {
		t.Fatalf("unexpected result %q %d %v", algo, size, err)
	}
	if _, _, err := manifestChecksum(entries, "missing.zip"); err == nil {
		t.Fatal("expected missing archive error")
	}
	sha1Manifest := []byte(`<remotePackage path="platform-tools"><archives><archive><checksum>0011223344556677889900112233445566778899</checksum><url>platform-tools-latest-windows.zip</url></archive></archives></remotePackage>`)
	entries, err = manifestArchives(sha1Manifest, "platform-tools")
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if _, _, err := manifestChecksum(entries, "platform-tools-latest-windows.zip"); err == nil || taskError(err).Code != "RUNTIME_CHECKSUM_MISSING" {
		t.Fatal("sha1 digest must not be accepted")
	}
}

func TestPlanDigestCoversTargetFields(t *testing.T) {
	plan := InstallPlan{Tool: "node", Mirror: "official", RequestedSelector: "22", ResolvedVersion: "22.1.0", OS: "win", Arch: "x64", TargetHome: "a"}
	other := plan
	other.TargetHome = "b"
	if computePlanDigest(plan) == computePlanDigest(other) {
		t.Fatal("digest must cover the target home")
	}
	if computePlanDigest(plan) != computePlanDigest(plan) {
		t.Fatal("digest must be stable")
	}
}
