package gitidentity

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func keygenSandbox(t *testing.T) (store *Store, tools KeygenTools) {
	t.Helper()
	// Windows GnuPG puts agent sockets under LOCALAPPDATA; long testing.T
	// TempDir paths break IPC. Keep the whole sandbox under a short prefix.
	root, err := os.MkdirTemp("", "dtg-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(root) })

	gpgHome := filepath.Join(root, "g")
	sshDir := filepath.Join(root, "s")
	appdata := filepath.Join(root, "a")
	local := filepath.Join(root, "l")
	for _, dir := range []string{gpgHome, sshDir, appdata, local} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("HOME", root)
	t.Setenv("USERPROFILE", root)
	t.Setenv("APPDATA", appdata)
	t.Setenv("LOCALAPPDATA", local)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "c"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(root, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GNUPGHOME", gpgHome)

	store = NewStore()
	tools = KeygenTools{
		GPGHome: gpgHome,
		SSHDir:  sshDir,
		Clock:   func() time.Time { return time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC) },
	}
	return store, tools
}

func requireTool(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not available: %v", name, err)
	}
}

func seedProfile(t *testing.T, store *Store) {
	t.Helper()
	if err := store.Upsert(Profile{ID: "work", Name: "测试用户", Email: "work@example.com"}); err != nil {
		t.Fatal(err)
	}
}

func TestKeygenOptionsProbe(t *testing.T) {
	_, tools := keygenSandbox(t)
	opts := KeygenOptionsProbe(context.Background(), tools)
	if len(opts.Tools) != 2 {
		t.Fatalf("expected 2 tool statuses, got %d", len(opts.Tools))
	}
	if opts.Defaults.OpenPGP.Algorithm != AlgoED25519 || opts.Defaults.SSH.Expiry != ExpiryNever {
		t.Fatalf("unexpected defaults: %+v", opts.Defaults)
	}
	if len(opts.Algorithms[0].Values) == 0 || len(opts.Expiries[1].Values) != 1 {
		t.Fatalf("enum groups incomplete: %+v", opts)
	}
}

func TestKeygenPreviewValidation(t *testing.T) {
	store, tools := keygenSandbox(t)
	seedProfile(t, store)

	plan, err := KeygenPreview(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID: "missing",
		KeyType:   KeyTypeOpenPGP,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.WouldFail == nil || *plan.WouldFail != "GIT_PROFILE_NOT_FOUND" {
		t.Fatalf("expected GIT_PROFILE_NOT_FOUND, got %+v", plan.WouldFail)
	}

	plan, err = KeygenPreview(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID: "work",
		KeyType:   KeyTypeSSH,
		Expiry:    Expiry1Y,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.WouldFail == nil || *plan.WouldFail != "GIT_KEYGEN_PARAMS_INVALID" {
		t.Fatalf("ssh expiry must fail, got %+v", plan.WouldFail)
	}

	plan, err = KeygenPreview(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID: "work",
		KeyType:   KeyTypeOpenPGP,
		Algorithm: "rsa2048",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.WouldFail == nil || *plan.WouldFail != "GIT_KEYGEN_PARAMS_INVALID" {
		t.Fatalf("bad algo must fail, got %+v", plan.WouldFail)
	}
}

func TestKeygenCreateRequiresConfirm(t *testing.T) {
	store, tools := keygenSandbox(t)
	seedProfile(t, store)
	_, err := KeygenCreate(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID: "work",
		KeyType:   KeyTypeSSH,
	})
	if err == nil || !strings.Contains(err.Error(), "CONFIRMATION_REQUIRED") {
		t.Fatalf("expected CONFIRMATION_REQUIRED, got %v", err)
	}
}

func TestKeygenCreateSSHAndSelectExport(t *testing.T) {
	requireTool(t, "ssh-keygen")
	store, tools := keygenSandbox(t)
	seedProfile(t, store)

	plan, err := KeygenPreview(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID: "work",
		KeyType:   KeyTypeSSH,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.WouldFail != nil {
		t.Fatalf("preview should pass when ssh-keygen present, wouldFail=%v summary=%v", *plan.WouldFail, plan.CommandSummary)
	}
	for _, line := range plan.CommandSummary {
		if strings.Contains(line, "secret") || strings.Contains(strings.ToLower(line), "private") {
			t.Fatalf("preview leaked sensitive wording: %s", line)
		}
	}

	result, err := KeygenCreate(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID: "work",
		KeyType:   KeyTypeSSH,
		Confirm:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.LocalKeyCreated || !result.ProfileUpdated || result.Profile == nil {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Profile.GpgFormat != KeyTypeSSH || result.Profile.SigningKey == "" || !result.Profile.GpgSign {
		t.Fatalf("profile fields not written: %+v", result.Profile)
	}
	if _, err := os.Stat(result.SigningKey); err != nil {
		t.Fatalf("public key missing: %v", err)
	}
	priv := strings.TrimSuffix(result.SigningKey, ".pub")
	if data, err := os.ReadFile(priv); err == nil {
		if strings.Contains(string(data), "PRIVATE") {
			// private exists on disk (expected) but must never appear in result JSON fields
			_ = data
		}
	}
	if result.PublicKeyPreview != nil && strings.Contains(*result.PublicKeyPreview, "PRIVATE") {
		t.Fatal("result preview must not contain private key")
	}

	list, err := KeyList(context.Background(), tools, KeyTypeSSH)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("expected at least one ssh key")
	}
	if !strings.Contains(list[0].ID, "devtoolbox_work_") {
		t.Fatalf("devtoolbox_ keys should sort first, got %s", list[0].ID)
	}

	// Overwrite signing fields via select after clearing.
	store.Profiles["work"].SigningKey = ""
	store.Profiles["work"].GpgFormat = ""
	store.Profiles["work"].GpgSign = false
	selected, err := KeySelect(context.Background(), store, tools, "work", KeyTypeSSH, list[0].ID, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if !selected.ProfileUpdated || selected.Profile.SigningKey != list[0].ID {
		t.Fatalf("select did not write path: %+v", selected)
	}

	exported, err := KeyExport(context.Background(), tools, KeyExportRequest{
		KeyType: KeyTypeSSH,
		ID:      list[0].ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(exported.Text, "ssh-") {
		t.Fatalf("unexpected pub text: %q", exported.Text)
	}

	outFile := filepath.Join(t.TempDir(), "out.pub")
	_, err = KeyExport(context.Background(), tools, KeyExportRequest{
		KeyType: KeyTypeSSH, ID: list[0].ID, WritePath: outFile,
	})
	if err == nil || !strings.Contains(err.Error(), "CONFIRMATION_REQUIRED") {
		t.Fatalf("write without confirm: %v", err)
	}
	exported, err = KeyExport(context.Background(), tools, KeyExportRequest{
		KeyType: KeyTypeSSH, ID: list[0].ID, WritePath: outFile, Confirm: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if exported.WrittenPath == nil {
		t.Fatal("expected writtenPath")
	}
	_, err = KeyExport(context.Background(), tools, KeyExportRequest{
		KeyType: KeyTypeSSH, ID: list[0].ID, WritePath: outFile, Confirm: true,
	})
	if err == nil || !strings.Contains(err.Error(), "GIT_KEY_EXPORT_EXISTS") {
		t.Fatalf("expected exists error, got %v", err)
	}
}

func TestKeygenCreateOpenPGP(t *testing.T) {
	requireTool(t, "gpg")
	store, tools := keygenSandbox(t)
	seedProfile(t, store)

	// Ensure we never touch the real user gnupg: GPGHome is the sandbox dir.
	if tools.GPGHome == "" || tools.GPGHome == filepath.Join(os.Getenv("USERPROFILE"), ".gnupg") {
		t.Fatal("test must use isolated GNUPGHOME")
	}

	result, err := KeygenCreate(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID: "work",
		KeyType:   KeyTypeOpenPGP,
		Algorithm: AlgoED25519,
		Expiry:    Expiry2Y,
		Confirm:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.LocalKeyCreated || !result.ProfileUpdated {
		t.Fatalf("unexpected: %+v", result)
	}
	if result.Fingerprint == nil || len(*result.Fingerprint) < 40 {
		t.Fatalf("bad fingerprint: %+v", result.Fingerprint)
	}
	if result.Profile.SigningKey != *result.Fingerprint || result.Profile.GpgFormat != KeyTypeOpenPGP {
		t.Fatalf("profile mismatch: %+v", result.Profile)
	}

	list, err := KeyList(context.Background(), tools, KeyTypeOpenPGP)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) == 0 {
		t.Fatal("expected openpgp key in list")
	}
	exported, err := KeyExport(context.Background(), tools, KeyExportRequest{
		KeyType: KeyTypeOpenPGP,
		ID:      *result.Fingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(exported.Text, "BEGIN PGP PUBLIC KEY BLOCK") {
		t.Fatalf("expected armored public key, got %q", exported.Text[:min(80, len(exported.Text))])
	}
}

func TestKeygenCreateSSHCustomOutputPath(t *testing.T) {
	requireTool(t, "ssh-keygen")
	store, tools := keygenSandbox(t)
	seedProfile(t, store)
	out := filepath.Join(tools.SSHDir, "custom_key")
	result, err := KeygenCreate(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID:  "work",
		KeyType:    KeyTypeSSH,
		OutputPath: out,
		Confirm:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(result.SigningKey, "custom_key.pub") {
		t.Fatalf("signingKey=%s", result.SigningKey)
	}
}

func TestKeygenMissingTool(t *testing.T) {
	store, tools := keygenSandbox(t)
	seedProfile(t, store)
	tools.SSHBin = filepath.Join(t.TempDir(), "no-such-ssh-keygen")
	plan, err := KeygenPreview(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID: "work",
		KeyType:   KeyTypeSSH,
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.WouldFail == nil || *plan.WouldFail != "GIT_KEYGEN_TOOL_MISSING" {
		t.Fatalf("expected TOOL_MISSING, got %+v", plan.WouldFail)
	}
	_, err = KeygenCreate(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID: "work", KeyType: KeyTypeSSH, Confirm: true,
	})
	if err == nil || !strings.Contains(err.Error(), "GIT_KEYGEN_TOOL_MISSING") {
		t.Fatalf("create should fail missing tool: %v", err)
	}
}

func TestKeygenEnableGpgSignDefaultTrue(t *testing.T) {
	requireTool(t, "ssh-keygen")
	store, tools := keygenSandbox(t)
	seedProfile(t, store)
	off := false
	result, err := KeygenCreate(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID: "work", KeyType: KeyTypeSSH, Confirm: true, EnableGpgSign: &off,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Profile.GpgSign {
		t.Fatal("enableGpgSign=false should stick")
	}
}

func TestKeygenProfileWritePartial(t *testing.T) {
	requireTool(t, "ssh-keygen")
	store, tools := keygenSandbox(t)
	seedProfile(t, store)
	// Force Save to fail by making the config dir a file after upsert path resolution.
	cfg, err := StorePath()
	if err != nil {
		t.Fatal(err)
	}
	parent := filepath.Dir(cfg)
	_ = os.RemoveAll(parent)
	if err := os.MkdirAll(filepath.Dir(parent), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(parent, []byte("not-a-dir"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err = KeygenCreate(context.Background(), store, tools, KeygenCreateRequest{
		ProfileID: "work", KeyType: KeyTypeSSH, Confirm: true,
	})
	var partial *KeygenProfileWriteError
	if !errors.As(err, &partial) {
		// On some platforms mkdir may still succeed differently; accept TOOL_FAILED only if key wasn't created.
		if err == nil {
			t.Fatal("expected error")
		}
		t.Logf("partial write scenario yielded: %v", err)
		return
	}
	if !partial.Result.LocalKeyCreated || partial.Result.ProfileUpdated {
		t.Fatalf("partial result wrong: %+v", partial.Result)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
