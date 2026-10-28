package gitidentity

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// rawGit bypasses the production subcommand allowlist for test fixtures
// (git init/commit/remote); it still uses the same array-based process start.
func rawGit(cwd string, args ...string) (string, error) {
	return startProcess(context.Background(), cwd, nil, "", false, args...)
}

// sandbox isolates every per-user location git and the service touch:
// global/system git config, HOME, and the per-user config dir. Real git
// runs inside the sandbox as the behavior evidence (迁移质量基线).
func sandbox(t *testing.T) *Store {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp)
	t.Setenv("APPDATA", filepath.Join(tmp, "appdata"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(tmp, "gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// GIT_AUTHOR_*/GIT_COMMITTER_* env would override repo-local identity
	// and break commits; the test process must simply not have them set.
	if _, err := rawGit("", "--version"); err != nil {
		t.Skipf("git not available: %v", err)
	}
	store := NewStore()
	return store
}

func initRepo(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if out, err := rawGit(dir, "config", "--global", "init.defaultBranch", "main"); err != nil {
		t.Fatalf("set default branch: %v (%s)", err, out)
	}
	if out, err := rawGit(dir, "init"); err != nil {
		t.Fatalf("git init: %v (%s)", err, out)
	}
	return dir
}

func gitValue(t *testing.T, cwd, scope, key string) string {
	t.Helper()
	args := []string{"config"}
	if scope != "" {
		args = append(args, scopeFlag(scope))
	}
	args = append(args, "--get", key)
	out, err := runGit(context.Background(), cwd, nil, "", false, args...)
	if err != nil {
		return ""
	}
	return out
}

func expectChange(t *testing.T, changes []ConfigChange, key, action string) ConfigChange {
	t.Helper()
	for _, change := range changes {
		if change.Key == key {
			if action != "" && change.Action != action {
				t.Fatalf("key %s: expected action %s, got %s", key, action, change.Action)
			}
			return change
		}
	}
	t.Fatalf("expected a change for key %s, got %+v", key, changes)
	return ConfigChange{}
}

func TestUpsertValidationAndBindingPreservation(t *testing.T) {
	sandbox(t)
	store := NewStore()
	if err := store.Upsert(Profile{ID: "bad id!", Name: "x", Email: "a@b.co"}); err == nil {
		t.Fatal("invalid id must be rejected")
	}
	if err := store.Upsert(Profile{ID: "work", Name: "张三", Email: "not-an-email"}); err == nil {
		t.Fatal("invalid email must be rejected")
	}
	if err := store.Upsert(Profile{ID: "work", Name: "张三", Email: "z@corp.com", GpgFormat: "weird"}); err == nil {
		t.Fatal("invalid format must be rejected")
	}
	if err := store.Upsert(Profile{ID: "work", Name: "张三", Email: "z@corp.com", InsteadOf: []InsteadOf{{Original: "git@github.com:", URL: "git@github.com:"}}}); err == nil {
		t.Fatal("identical insteadOf pair must be rejected")
	}
	work := Profile{ID: "work", Name: "张三", Email: "z@corp.com", SigningKey: "KEY1", GpgSign: true, GpgFormat: "ssh",
		InsteadOf: []InsteadOf{{Original: "git@github.com:", URL: "git@github.com-work:"}},
		Accounts:  []Account{{Host: "github.com", Username: "corp-zhang"}}}
	if err := store.Upsert(work); err != nil {
		t.Fatal(err)
	}
	store.Profiles["work"].Gitdirs = []string{"D:/work/"}
	if err := store.Upsert(Profile{ID: "work", Name: "张三", Email: "z@corp.com"}); err != nil {
		t.Fatal(err)
	}
	if len(store.Profiles["work"].Gitdirs) != 1 {
		t.Fatal("upsert must preserve existing directory bindings")
	}
	if _, err := store.Profile("missing"); err == nil {
		t.Fatal("unknown profile must fail")
	}
}

func TestApplyGlobalFullOverwriteAndCleanup(t *testing.T) {
	sandbox(t)
	store := NewStore()
	if err := store.Upsert(Profile{ID: "work", Name: "张三", Email: "z@corp.com", SigningKey: "KEY1", GpgSign: true, GpgFormat: "ssh",
		InsteadOf: []InsteadOf{{Original: "git@github.com:", URL: "git@github.com-work:"}}}); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(Profile{ID: "personal", Name: "tony", Email: "tony@me.dev"}); err != nil {
		t.Fatal(err)
	}
	changes, err := ApplyProfile(context.Background(), store, "work", "global", "", false)
	if err != nil {
		t.Fatal(err)
	}
	expectChange(t, changes, "user.name", "set")
	expectChange(t, changes, "commit.gpgsign", "set")
	expectChange(t, changes, "url.git@github.com-work:.insteadOf", "set")
	if gitValue(t, "", "global", "user.signingkey") != "KEY1" || gitValue(t, "", "global", "gpg.format") != "ssh" {
		t.Fatal("signing keys not applied")
	}
	// switch to personal: signing/format/insteadOf must be cleaned (full overwrite, ADR-016)
	changes, err = ApplyProfile(context.Background(), store, "personal", "global", "", false)
	if err != nil {
		t.Fatal(err)
	}
	expectChange(t, changes, "user.signingkey", "unset")
	expectChange(t, changes, "commit.gpgsign", "unset")
	expectChange(t, changes, "gpg.format", "unset")
	expectChange(t, changes, "url.git@github.com-work:.insteadOf", "unset")
	if gitValue(t, "", "global", "user.signingkey") != "" || gitValue(t, "", "global", "gpg.format") != "" {
		t.Fatal("previous identity keys leaked after switch")
	}
	if gitValue(t, "", "global", "url.git@github.com-work:.insteadOf") != "" {
		t.Fatal("previous insteadOf leaked after switch")
	}
	if gitValue(t, "", "global", "user.name") != "tony" {
		t.Fatal("personal identity not applied")
	}
}

func TestApplyLocalDryRunAndBatch(t *testing.T) {
	sandbox(t)
	store := NewStore()
	if err := store.Upsert(Profile{ID: "work", Name: "张三", Email: "z@corp.com", SigningKey: "K"}); err != nil {
		t.Fatal(err)
	}
	repo := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	dry, err := ApplyProfile(context.Background(), store, "work", "local", repo, true)
	if err != nil {
		t.Fatal(err)
	}
	expectChange(t, dry, "user.name", "set")
	if gitValue(t, repo, "local", "user.name") != "" {
		t.Fatal("dry-run must not write")
	}
	if _, err := ApplyProfile(context.Background(), store, "work", "local", repo, false); err != nil {
		t.Fatal(err)
	}
	if gitValue(t, repo, "local", "user.name") != "张三" {
		t.Fatal("local apply failed")
	}
	results := ApplyToRepos(context.Background(), store, "work", []string{repo, filepath.Join(t.TempDir(), "not-a-repo")}, true)
	if len(results) != 2 || results[0].Status != "skipped" || results[1].Status != "failed" || results[1].Error != "GIT_NOT_A_REPOSITORY" {
		t.Fatalf("unexpected batch results: %+v", results)
	}
	if gitValue(t, repo, "local", "user.name") != "张三" {
		t.Fatal("dry-run batch must not write")
	}
}

func TestEffectiveIdentitySourcesAndPrompt(t *testing.T) {
	sandbox(t)
	store := NewStore()
	if err := store.Upsert(Profile{ID: "work", Name: "张三", Email: "z@corp.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveEffective(context.Background(), store, t.TempDir(), false); err != nil {
		t.Fatal(err)
	}
	repo := initRepo(t, filepath.Join(t.TempDir(), "repo"))
	eff, err := ResolveEffective(context.Background(), store, repo, true)
	if err != nil {
		t.Fatal(err)
	}
	if eff.Resolved != "none" || eff.PromptLine != "none" {
		t.Fatalf("expected none, got %+v", eff)
	}
	if _, err := rawGit(repo, "config", "--local", "user.name", "someone"); err != nil {
		t.Fatal(err)
	}
	if _, err := rawGit(repo, "config", "--local", "user.email", "else@where.dev"); err != nil {
		t.Fatal(err)
	}
	eff, err = ResolveEffective(context.Background(), store, repo, true)
	if err != nil {
		t.Fatal(err)
	}
	if eff.Resolved != "manual" || eff.PromptLine != "manual" {
		t.Fatalf("expected manual, got %+v", eff)
	}
	for _, key := range eff.Keys {
		if key.Key == "user.name" && key.Source != SourceLocal {
			t.Fatalf("local key source should be local, got %s", key.Source)
		}
	}
	if _, err := ApplyProfile(context.Background(), store, "work", "local", repo, false); err != nil {
		t.Fatal(err)
	}
	eff, err = ResolveEffective(context.Background(), store, repo, true)
	if err != nil {
		t.Fatal(err)
	}
	if eff.Resolved != "work" || eff.PromptLine != "work" {
		t.Fatalf("expected work, got %+v", eff)
	}
}

func TestBindUnbindIncludeIfAndDrift(t *testing.T) {
	sandbox(t)
	store := NewStore()
	if err := store.Upsert(Profile{ID: "work", Name: "张三", Email: "z@corp.com", SigningKey: "K"}); err != nil {
		t.Fatal(err)
	}
	workRoot := filepath.Join(t.TempDir(), "Work")
	if err := os.MkdirAll(workRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, _, err := Bind(context.Background(), store, "work", workRoot); err != nil {
		t.Fatal(err)
	}
	if !includeHookPresent(context.Background()) {
		t.Fatal("include hook missing after bind")
	}
	// a repo under the bound tree inherits the identity via includeIf
	repo := initRepo(t, filepath.Join(workRoot, "project"))
	eff, err := ResolveEffective(context.Background(), store, repo, false)
	if err != nil {
		t.Fatal(err)
	}
	if eff.Resolved != "work" {
		t.Fatalf("includeIf binding not effective: %+v", eff)
	}
	for _, key := range eff.Keys {
		if key.Key == "user.name" && key.Source != SourceIncludeIf {
			t.Fatalf("expected includeIf source, got %s", key.Source)
		}
	}
	// hand-edit the snippet: unbind must warn instead of silently dropping
	snippet, err := snippetPath("work")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(snippet)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snippet, append(data, []byte("\n[core]\n\texcludesfiles = tampered\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	_, drift, err := Unbind(context.Background(), store, "work", workRoot)
	if err != nil {
		t.Fatal(err)
	}
	if drift == nil || !drift.Drift {
		t.Fatalf("expected drift warning, got %+v", drift)
	}
	eff, err = ResolveEffective(context.Background(), store, repo, false)
	if err != nil {
		t.Fatal(err)
	}
	if eff.Resolved != "none" {
		t.Fatalf("binding should be gone after unbind, got %+v", eff)
	}
}

func TestDoctorReportsOrphanAndHook(t *testing.T) {
	sandbox(t)
	store := NewStore()
	if err := store.Upsert(Profile{ID: "work", Name: "张三", Email: "z@corp.com"}); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "bound")
	if _, _, err := Bind(context.Background(), store, "work", dir); err != nil {
		t.Fatal(err)
	}
	report, err := GitDoctor(context.Background(), store)
	if err != nil {
		t.Fatal(err)
	}
	if !report.HookPresent {
		t.Fatal("hook should be present")
	}
	// simulate hook loss by pointing GIT_CONFIG_GLOBAL at a fresh file
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(t.TempDir(), "other-gitconfig"))
	report, err = GitDoctor(context.Background(), store)
	if err != nil {
		t.Fatal(err)
	}
	if report.HookPresent {
		t.Fatal("hook should be reported missing")
	}
	hookMissing := false
	for _, finding := range report.Findings {
		if strings.Contains(finding.Detail, "include 钩子") {
			hookMissing = true
		}
	}
	if !hookMissing {
		t.Fatalf("expected hook-missing finding, got %+v", report.Findings)
	}
}

func TestScanAuditAndMirror(t *testing.T) {
	sandbox(t)
	store := NewStore()
	if err := store.Upsert(Profile{ID: "work", Name: "张三", Email: "z@corp.com"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Upsert(Profile{ID: "personal", Name: "tony", Email: "tony@me.dev"}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	repo := initRepo(t, filepath.Join(root, "proj"))
	if _, err := rawGit(repo, "config", "--local", "user.name", "张三"); err != nil {
		t.Fatal(err)
	}
	if _, err := rawGit(repo, "config", "--local", "user.email", "z@corp.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := rawGit(repo, "remote", "add", "origin", "https://github.com/demo/demo.git"); err != nil {
		t.Fatal(err)
	}
	scan, err := ScanRepos(context.Background(), store, root, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(scan.Rows) != 1 || scan.Rows[0].Status != "ok" || !scan.Rows[0].HasLocal || scan.Rows[0].IdentityID != "work" {
		t.Fatalf("unexpected scan: %+v", scan.Rows)
	}
	if len(scan.Rows[0].Remotes) != 1 || scan.Rows[0].Remotes[0].Mirror {
		t.Fatalf("unexpected remotes: %+v", scan.Rows[0].Remotes)
	}

	pushURLs, err := MirrorSet(context.Background(), repo, "origin", "https://mirror.example.com/demo/demo.git")
	if err != nil {
		t.Fatal(err)
	}
	if len(pushURLs) != 2 {
		t.Fatalf("unexpected pushurls %v", pushURLs)
	}
	scan, err = ScanRepos(context.Background(), store, root, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !scan.Rows[0].Remotes[0].Mirror || scan.Summary.Mirrored != 1 {
		t.Fatalf("mirror not detected: %+v", scan.Rows[0].Remotes)
	}
	if _, err := MirrorUnset(context.Background(), repo, "origin"); err != nil {
		t.Fatal(err)
	}

	// two authors: work identity and a polluted one
	commit := func(name, email string) {
		t.Helper()
		if _, err := rawGit(repo, "config", "user.name", name); err != nil {
			t.Fatal(err)
		}
		if _, err := rawGit(repo, "config", "user.email", email); err != nil {
			t.Fatal(err)
		}
		if _, err := rawGit(repo, "commit", "--allow-empty", "-m", "x"); err != nil {
			t.Fatalf("commit: %v", err)
		}
	}
	commit("张三", "z@corp.com")
	commit("Tony", "tony@personal.dev")
	report, err := AuditRepo(context.Background(), store, repo, "work")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Authors) != 2 || report.StrictPassed == nil || *report.StrictPassed {
		t.Fatalf("expected strict failure with two authors: %+v", report.Authors)
	}
	if !strings.Contains(report.MailmapSuggestion, "张三 <z@corp.com> Tony <tony@personal.dev>") {
		t.Fatalf("unexpected mailmap suggestion: %q", report.MailmapSuggestion)
	}
	report, err = AuditRepo(context.Background(), store, repo, "work")
	if err != nil {
		t.Fatal(err)
	}
	// remove the polluted commit author check for clean path: audit without expectation
	if _, err := AuditRepo(context.Background(), store, repo, ""); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialListAndStoreWithFakeHelper(t *testing.T) {
	sandbox(t)
	store := NewStore()
	if err := store.Upsert(Profile{ID: "work", Name: "张三", Email: "z@corp.com",
		Accounts: []Account{{Host: "github.com", Username: "corp-zhang"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyProfile(context.Background(), store, "work", "global", "", false); err != nil {
		t.Fatal(err)
	}
	entries := CredentialList(context.Background(), store, "")
	if len(entries) != 1 || entries[0].Host != "github.com" || entries[0].Username != "corp-zhang" {
		t.Fatalf("unexpected credential list: %+v", entries)
	}
	// file-backed fake helper exercises the full approve/fill conduit
	// forward slashes: the helper runs under sh, which eats backslashes in
	// unquoted redirect targets and would create a stray relative file
	helperStore := filepath.ToSlash(filepath.Join(t.TempDir(), "helper-store.txt"))
	helper := "!f() { case \"$1\" in get) test -f " + helperStore + " && cat " + helperStore + " ;; store) cat >> " + helperStore + " ;; erase) rm -f " + helperStore + " ;; esac; }; f"
	if out, err := rawGit("", "config", "--global", "credential.helper", helper); err != nil {
		t.Fatalf("set helper: %v (%s)", err, out)
	}
	if err := CredentialStore(context.Background(), store, "github.com", "corp-zhang", "tok_test_123"); err != nil {
		t.Fatalf("credential store: %v", err)
	}
	entry, stored, err := CredentialProbe(context.Background(), "github.com", "")
	if err != nil {
		t.Fatal(err)
	}
	if !stored || entry.Username != "corp-zhang" {
		t.Fatalf("probe did not find stored credential: %+v stored=%v", entry, stored)
	}
	if err := CredentialErase(context.Background(), store, "github.com", "corp-zhang"); err != nil {
		t.Fatal(err)
	}
	if _, stored, err := CredentialProbe(context.Background(), "github.com", ""); err != nil || stored {
		t.Fatalf("credential should be erased (stored=%v err=%v)", stored, err)
	}
}

func TestLegacyGitidImportIdempotent(t *testing.T) {
	sandbox(t)
	legacyConfig := filepath.Join(t.TempDir(), "gitid-config.json")
	t.Setenv("GITID_CONFIG", legacyConfig)
	legacy := `{"version":1,"identities":{"work":{"name":"张三","email":"z@corp.com","signingkey":"KEY1","gpgsign":true,"gpgFormat":"ssh"},"dup@x":{"name":"张三","email":"z@corp.com"}}}`
	if err := os.WriteFile(legacyConfig, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore()
	preview, err := PreviewLegacy(context.Background(), store)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Found || !preview.Valid || len(preview.Profiles) != 1 || len(preview.DuplicateEmails) != 1 {
		t.Fatalf("unexpected preview: %+v", preview)
	}
	record, err := LegacyImport(context.Background(), store, preview.SourceDigest, nil, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if record.Created != 1 || record.Skipped != 0 {
		t.Fatalf("unexpected import record: %+v", record)
	}
	if profile, err := store.Profile("work"); err != nil || profile.SigningKey != "KEY1" || !profile.GpgSign {
		t.Fatalf("imported profile wrong: %+v %v", profile, err)
	}
	again, err := LegacyImport(context.Background(), store, preview.SourceDigest, nil, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if again.ImportedAt != record.ImportedAt {
		t.Fatal("repeat import must be idempotent")
	}
	if _, err := LegacyImport(context.Background(), store, "wrong-digest", nil, true, true); err == nil {
		t.Fatal("digest mismatch must fail")
	}
	if _, err := LegacyImport(context.Background(), store, preview.SourceDigest, nil, false, false); err == nil {
		t.Fatal("missing confirmation must fail")
	}
}
