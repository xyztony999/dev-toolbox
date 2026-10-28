package runtimecatalog

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"dev-toolbox/internal/userstate"
)

func isolateUserDirs(t *testing.T, root string) {
	t.Helper()
	for _, key := range []string{"HOME", "USERPROFILE", "APPDATA", "XDG_CONFIG_HOME"} {
		t.Setenv(key, root)
	}
}

func TestInspectAndRegisterDoesNotExecuteCandidateOrChangeGlobalDefault(t *testing.T) {
	root := t.TempDir()
	isolateUserDirs(t, root)
	if runtime.GOOS == "windows" {
		t.Setenv("PATH", root)
	}
	home := filepath.Join(root, "node-v22.12.0")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(home, "node.exe")
	if runtime.GOOS != "windows" {
		binary = filepath.Join(home, "node")
	}
	marker := filepath.Join(root, "executed")
	content := []byte("static fixture; this file must never be executed")
	if err := os.WriteFile(binary, content, 0o600); err != nil {
		t.Fatal(err)
	}
	candidate, err := Inspect("node", home, "user")
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Version != "22.12.0" || candidate.Confidence != "exact" {
		t.Fatalf("unexpected static version: %#v", candidate)
	}
	state, err := userstate.Get()
	if err != nil {
		t.Fatal(err)
	}
	updated, counts, err := Register(context.Background(), []CandidateRef{{CandidateID: candidate.CandidateID, Digest: candidate.Digest, Tool: candidate.Tool, Home: candidate.Home, Source: candidate.Source}}, state.Revision, true)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Created != 1 || len(updated.Installations) != 1 {
		t.Fatalf("unexpected registration result: %#v %#v", counts, updated.Installations)
	}
	if updated.GlobalSelections["node"].Status != "empty" {
		t.Fatalf("registration changed global default: %#v", updated.GlobalSelections["node"])
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("candidate was executed: %v", err)
	}
	got, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Fatal("candidate file changed during inspection")
	}
}

func TestRegisterRejectsStaleCandidate(t *testing.T) {
	root := t.TempDir()
	isolateUserDirs(t, root)
	home := filepath.Join(root, "runtime")
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	binary := "node"
	if runtime.GOOS == "windows" {
		binary = "node.exe"
	}
	if err := os.WriteFile(filepath.Join(home, binary), []byte("candidate"), 0o600); err != nil {
		t.Fatal(err)
	}
	candidate, err := Inspect("node", home, "user")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(home, binary)); err != nil {
		t.Fatal(err)
	}
	_, _, err = Register(context.Background(), []CandidateRef{{CandidateID: candidate.CandidateID, Digest: candidate.Digest, Tool: candidate.Tool, Home: candidate.Home, Source: candidate.Source}}, 0, true)
	if err == nil || err.Error() != "RUNTIME_CANDIDATE_STALE" {
		t.Fatalf("expected stale candidate error, got %v", err)
	}
}

func TestDiscoverRejectsUnknownTool(t *testing.T) {
	if _, err := Discover(context.Background(), []string{"unknown"}, nil); err == nil {
		t.Fatal("expected unsupported tool error")
	}
}
