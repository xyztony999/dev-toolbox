package runtimecatalog

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func writeRustToolchain(t *testing.T, root, name string) string {
	t.Helper()
	home := filepath.Join(root, "toolchains", name)
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatal(err)
	}
	rustc := "rustc"
	if runtime.GOOS == "windows" {
		rustc = "rustc.exe"
	}
	if err := os.WriteFile(filepath.Join(binDir, rustc), []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	return home
}

func TestDiscoverRustupToolchains(t *testing.T) {
	root := t.TempDir()
	isolateUserDirs(t, root)
	t.Setenv("RUSTUP_HOME", filepath.Join(root, "rustup"))
	t.Setenv("UV_PYTHON_INSTALL_DIR", filepath.Join(root, "uv-python"))
	if runtime.GOOS == "windows" {
		t.Setenv("PATH", root)
	}

	exact := writeRustToolchain(t, filepath.Join(root, "rustup"), "1.82.0-x86_64-pc-windows-msvc")
	channel := writeRustToolchain(t, filepath.Join(root, "rustup"), "stable-x86_64-pc-windows-msvc")
	settings := "version = \"12\"\ndefault_toolchain = \"stable-x86_64-pc-windows-msvc\"\nprofile = \"default\"\n\n[overrides]\n"
	if err := os.WriteFile(filepath.Join(root, "rustup", "settings.toml"), []byte(settings), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := Discover(context.Background(), []string{"rust"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 2 {
		t.Fatalf("expected 2 rust candidates, got %+v", result.Candidates)
	}
	byHome := map[string]Candidate{}
	for _, item := range result.Candidates {
		byHome[item.Home] = item
	}
	gotExact, ok := byHome[exact]
	if !ok {
		t.Fatalf("semver toolchain missing: %+v", result.Candidates)
	}
	if gotExact.Version != "1.82.0" || gotExact.Confidence != "exact" || gotExact.Source != "rustup" || gotExact.Vendor != "rust-lang" {
		t.Fatalf("unexpected semver candidate: %+v", gotExact)
	}
	gotChannel, ok := byHome[channel]
	if !ok {
		t.Fatalf("channel toolchain missing: %+v", result.Candidates)
	}
	if gotChannel.Version != "unknown" || gotChannel.Confidence != "unknown" {
		t.Fatalf("channel must stay unknown honestly: %+v", gotChannel)
	}
	hasChannelEvidence, hasDefault := false, false
	for _, evidence := range gotChannel.Evidence {
		if evidence == "rustup channel:stable" {
			hasChannelEvidence = true
		}
		if evidence == "rustup default" {
			hasDefault = true
		}
	}
	if !hasChannelEvidence || !hasDefault {
		t.Fatalf("channel evidence/default marker missing: %+v", gotChannel.Evidence)
	}
}

func TestDiscoverUvPythons(t *testing.T) {
	root := t.TempDir()
	isolateUserDirs(t, root)
	t.Setenv("RUSTUP_HOME", filepath.Join(root, "rustup"))
	t.Setenv("UV_PYTHON_INSTALL_DIR", filepath.Join(root, "uv-python"))
	if runtime.GOOS == "windows" {
		t.Setenv("PATH", root)
	}

	home := filepath.Join(root, "uv-python", "cpython-3.13.5-windows-x86_64-none")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	python := "python3"
	if runtime.GOOS == "windows" {
		python = "python.exe"
	}
	if err := os.WriteFile(filepath.Join(home, python), []byte("stub"), 0o755); err != nil {
		t.Fatal(err)
	}
	// 非解释器目录必须被忽略
	if err := os.MkdirAll(filepath.Join(root, "uv-python", "not-a-python"), 0o700); err != nil {
		t.Fatal(err)
	}

	result, err := Discover(context.Background(), []string{"python"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("expected exactly the uv candidate, got %+v", result.Candidates)
	}
	got := result.Candidates[0]
	if got.Source != "uv" || got.Version != "3.13.5" || got.Confidence != "exact" || got.Vendor != "python-build-standalone" {
		t.Fatalf("unexpected uv candidate: %+v", got)
	}
}

func TestRustRegisterAndOldStateHealsRustKey(t *testing.T) {
	root := t.TempDir()
	isolateUserDirs(t, root)
	t.Setenv("RUSTUP_HOME", filepath.Join(root, "rustup"))
	t.Setenv("UV_PYTHON_INSTALL_DIR", filepath.Join(root, "uv-python"))
	if runtime.GOOS == "windows" {
		t.Setenv("PATH", root)
	}
	toolchain := writeRustToolchain(t, filepath.Join(root, "rustup"), "1.82.0-x86_64-pc-windows-msvc")

	// 旧状态文件：仅七工具键，无 rust
	configDir := filepath.Join(root, "DevToolbox")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatal(err)
	}
	oldState := map[string]any{
		"schema": 1, "revision": 3,
		"globalSelections": map[string]any{
			"node": map[string]any{"status": "empty"},
		},
		"installations": []any{},
		"legacyImports": []any{},
	}
	data, err := json.Marshal(oldState)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "user-runtime-state.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	candidate, err := Inspect("rust", toolchain, "rustup")
	if err != nil {
		t.Fatal(err)
	}
	state, _, err := Register(context.Background(), []CandidateRef{{CandidateID: candidate.CandidateID, Digest: candidate.Digest, Tool: "rust", Home: candidate.Home, Source: "rustup"}}, 3, true)
	if err != nil {
		t.Fatal(err)
	}
	rustSelection, healed := state.GlobalSelections["rust"]
	if !healed || rustSelection.Status != "empty" {
		t.Fatalf("old state must heal rust key to empty: %+v", state.GlobalSelections)
	}
	found := false
	for _, item := range state.Installations {
		if item.Runtime == "rust" {
			found = true
		}
	}
	if !found {
		t.Fatalf("rust installation not registered: %+v", state.Installations)
	}
}
