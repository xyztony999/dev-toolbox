package gitidentity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// BaseDir owns every managed artifact: the profile store lives next to it,
// snippets under snippets/, the include chain in includes.gitconfig and
// legacy backups under backups/.
func BaseDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "DevToolbox", "git-identity"), nil
}

func snippetPath(id string) (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "snippets", id+".gitconfig"), nil
}

func includesPath() (string, error) {
	base, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "includes.gitconfig"), nil
}

func toPosixDir(path string) string {
	normalized := filepath.ToSlash(strings.TrimSpace(path))
	if !strings.HasSuffix(normalized, "/") {
		normalized += "/"
	}
	return normalized
}

func expandUserPath(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "~" {
		home, _ := os.UserHomeDir()
		return home
	}
	if strings.HasPrefix(trimmed, "~/") || strings.HasPrefix(trimmed, "~\\") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, trimmed[2:])
	}
	return trimmed
}

// realPath resolves symlinks/junctions for existing paths (gitid ADR: the
// includeIf condition matches the real .git path), else returns the
// cleaned absolute path so directories may be bound before they exist.
func realPath(input string) string {
	expanded := expandUserPath(input)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return expanded
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return resolved
	}
	return abs
}

func normalizeGitdir(dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return "", errors.New("GIT_DIR_INVALID: bind needs a directory")
	}
	return toPosixDir(realPath(dir)), nil
}

func sameFsPath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	na := strings.TrimSuffix(toPosixDir(realPath(a)), "/")
	nb := strings.TrimSuffix(toPosixDir(realPath(b)), "/")
	if runtime.GOOS == "windows" {
		return strings.EqualFold(na, nb)
	}
	return na == nb
}

func identityGitdirs(profile *Profile) []string {
	out := []string{}
	for _, dir := range profile.Gitdirs {
		if strings.TrimSpace(dir) != "" {
			out = append(out, dir)
		}
	}
	return out
}

func findGitdirOwners(store *Store, normalized string) []string {
	owners := []string{}
	ids := make([]string, 0, len(store.Profiles))
	for id := range store.Profiles {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for _, dir := range identityGitdirs(store.Profiles[id]) {
			if dir == normalized || sameFsPath(dir, normalized) {
				owners = append(owners, id)
				break
			}
		}
	}
	return owners
}

func includeIfGitdirPatterns(dir string) []string {
	seen := map[string]bool{}
	patterns := []string{}
	push := func(raw string) {
		posix := toPosixDir(raw)
		keys := []string{posix}
		if len(posix) >= 2 && posix[1] == ':' {
			keys = append(keys, string(posix[0]+'a'-'A')+posix[1:])
		}
		for _, key := range keys {
			fold := key
			if runtime.GOOS == "windows" {
				fold = strings.ToLower(key)
			}
			if seen[fold] {
				continue
			}
			seen[fold] = true
			patterns = append(patterns, key)
		}
	}
	push(dir)
	if real := realPath(dir); real != dir {
		push(real)
	}
	return patterns
}

// writeSnippetFile materializes the managed include snippet for one profile
// through `git config --file`, the same tool that reads it back, so byte
// comparison against a regenerated expectation is meaningful.
func writeSnippetFile(ctx context.Context, path string, profile *Profile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := path + ".tmp"
	tmpFile, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_ = tmpFile.Close()
	call := gitAt(ctx, "")
	set := func(key, value string) { call.must("config", "--file", tmp, key, value) }
	set("user.name", profile.Name)
	set("user.email", profile.Email)
	if profile.SigningKey != "" {
		set("user.signingkey", profile.SigningKey)
	}
	if profile.GpgSign {
		set("commit.gpgsign", "true")
	}
	if profile.GpgFormat != "" {
		set("gpg.format", profile.GpgFormat)
	}
	for _, key := range customKeysSorted(profile) {
		set(key, profile.CustomKv[key])
	}
	for _, pair := range profile.InsteadOf {
		set(insteadOfConfigKey(pair), pair.Original)
	}
	for _, account := range profile.Accounts {
		set(accountConfigKey(account), account.Username)
	}
	if err := os.Rename(tmp, path); err != nil {
		data, copyErr := os.ReadFile(tmp)
		if copyErr == nil {
			copyErr = os.WriteFile(path, data, 0o600)
		}
		_ = os.Remove(tmp)
		if copyErr != nil {
			return copyErr
		}
	}
	return nil
}

// rebuildIncludes rewrites the single managed includes file from all
// bindings; the user's real global gitconfig keeps exactly one include hook
// pointing here.
func rebuildIncludes(ctx context.Context, store *Store) error {
	target, err := includesPath()
	if err != nil {
		return err
	}
	type binding struct{ id, dir string }
	bindings := []binding{}
	ids := make([]string, 0, len(store.Profiles))
	for id := range store.Profiles {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		for _, dir := range identityGitdirs(store.Profiles[id]) {
			bindings = append(bindings, binding{id, dir})
		}
	}
	sort.SliceStable(bindings, func(i, j int) bool {
		if len(bindings[i].dir) != len(bindings[j].dir) {
			return len(bindings[i].dir) < len(bindings[j].dir)
		}
		return bindings[i].dir < bindings[j].dir
	})
	condition := "gitdir"
	if runtime.GOOS == "windows" {
		condition = "gitdir/i"
	}
	lines := []string{}
	for _, entry := range bindings {
		snippet, err := snippetPath(entry.id)
		if err != nil {
			return err
		}
		snippetPosix := filepath.ToSlash(snippet)
		for _, pattern := range windowsDriveVariants(includeIfGitdirPatterns(entry.dir)) {
			lines = append(lines, fmt.Sprintf("[includeIf \"%s:%s\"]", condition, pattern), "\tpath = "+snippetPosix, "")
		}
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
		return err
	}
	return os.WriteFile(target, []byte(strings.Join(lines, "\n")), 0o600)
}

func includeHookPresent(ctx context.Context) bool {
	target, err := includesPath()
	if err != nil {
		return false
	}
	value, ok := gitAt(ctx, "").allowFail("config", "--global", "--get-all", "include.path")
	if !ok {
		return false
	}
	for _, line := range strings.Split(value, "\n") {
		if sameFsPath(strings.TrimSpace(line), target) {
			return true
		}
	}
	return false
}

func ensureIncludeHook(ctx context.Context) error {
	target, err := includesPath()
	if err != nil {
		return err
	}
	if includeHookPresent(ctx) {
		return nil
	}
	gitAt(ctx, "").must("config", "--global", "--add", "include.path", target)
	return nil
}

// SnippetDrift reports whether a managed snippet on disk diverges from what
// the profile would generate. Missing is reported as drift==false plus
// missing==true; comparison problems degrade honestly.
type SnippetDrift struct {
	ID      string   `json:"id"`
	Drift   bool     `json:"drift"`
	Missing bool     `json:"missing"`
	Unknown bool     `json:"unknown"`
	Keys    []string `json:"keys,omitempty"`
}

func checkSnippetDrift(ctx context.Context, id string, profile *Profile) SnippetDrift {
	file, err := snippetPath(id)
	if err != nil {
		return SnippetDrift{ID: id, Unknown: true}
	}
	disk, err := os.ReadFile(file)
	if err != nil {
		return SnippetDrift{ID: id, Missing: true}
	}
	expectedFile := file + ".expected"
	tmpFile, err := os.OpenFile(expectedFile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return SnippetDrift{ID: id, Unknown: true}
	}
	_ = tmpFile.Close()
	defer os.Remove(expectedFile)
	if err := writeSnippetFile(ctx, expectedFile, profile); err != nil {
		return SnippetDrift{ID: id, Unknown: true}
	}
	expected, err := os.ReadFile(expectedFile)
	if err != nil {
		return SnippetDrift{ID: id, Unknown: true}
	}
	if string(disk) == string(expected) {
		return SnippetDrift{ID: id}
	}
	list := func(data []byte) []string {
		path := file + ".list"
		if err := os.WriteFile(path, data, 0o600); err != nil {
			return nil
		}
		defer os.Remove(path)
		out, _ := gitAt(ctx, "").allowFail("config", "--list", "--file", path)
		lines := strings.Split(out, "\n")
		sort.Strings(lines)
		return lines
	}
	a, b := list(disk), list(expected)
	keys := []string{}
	for i := 0; i < len(a) || i < len(b); i++ {
		var line string
		if i < len(a) && (i >= len(b) || a[i] != b[i]) {
			line = a[i]
		} else if i < len(b) {
			line = b[i]
		}
		if line == "" {
			continue
		}
		key := strings.SplitN(line, "=", 2)[0]
		duplicate := false
		for _, existing := range keys {
			if existing == key {
				duplicate = true
			}
		}
		if !duplicate {
			keys = append(keys, key)
		}
	}
	return SnippetDrift{ID: id, Drift: true, Keys: keys}
}

// Bind attaches a directory tree to a profile: snippet + include chain are
// written immediately; the directory itself may not exist yet.
func Bind(ctx context.Context, store *Store, id, dir string) (*Profile, SnippetDrift, error) {
	profile, err := store.Profile(id)
	if err != nil {
		return nil, SnippetDrift{}, err
	}
	normalized, err := normalizeGitdir(dir)
	if err != nil {
		return nil, SnippetDrift{}, err
	}
	for _, owner := range findGitdirOwners(store, normalized) {
		if owner != id {
			return nil, SnippetDrift{}, fmt.Errorf("GIT_DIRECTORY_BOUND: directory already bound to %s", owner)
		}
	}
	bound := false
	for _, existing := range profile.Gitdirs {
		if existing == normalized || sameFsPath(existing, normalized) {
			bound = true
		}
	}
	if !bound {
		profile.Gitdirs = append(profile.Gitdirs, normalized)
	}
	snippet, err := snippetPath(id)
	if err != nil {
		return nil, SnippetDrift{}, err
	}
	if err := writeSnippetFile(ctx, snippet, profile); err != nil {
		return nil, SnippetDrift{}, errors.New("GIT_SNIPPET_WRITE_FAILED: " + err.Error())
	}
	if err := rebuildIncludes(ctx, store); err != nil {
		return nil, SnippetDrift{}, errors.New("GIT_SNIPPET_WRITE_FAILED: " + err.Error())
	}
	if err := ensureIncludeHook(ctx); err != nil {
		return nil, SnippetDrift{}, errors.New("GIT_HOOK_WRITE_FAILED: " + err.Error())
	}
	if err := store.Save(); err != nil {
		return nil, SnippetDrift{}, err
	}
	return profile, SnippetDrift{ID: id}, nil
}

// Unbind detaches one directory (or all directories of the profile) and
// recycles the snippet when it becomes unused; hand-edited snippets are
// reported, never silently dropped.
func Unbind(ctx context.Context, store *Store, id, dir string) (*Profile, *SnippetDrift, error) {
	var profile *Profile
	var err error
	normalized := ""
	if strings.TrimSpace(dir) != "" {
		if normalized, err = normalizeGitdir(dir); err != nil {
			return nil, nil, err
		}
	}
	if id == "" {
		owners := findGitdirOwners(store, normalized)
		if len(owners) == 0 {
			return nil, nil, errors.New("GIT_PROFILE_NOT_FOUND: no binding for this directory")
		}
		if len(owners) > 1 {
			return nil, nil, fmt.Errorf("GIT_DIRECTORY_BOUND: directory bound to multiple profiles: %s", strings.Join(owners, ", "))
		}
		id = owners[0]
	}
	profile, err = store.Profile(id)
	if err != nil {
		return nil, nil, err
	}
	before := identityGitdirs(profile)
	next := []string{}
	if normalized != "" {
		for _, existing := range before {
			if existing != normalized && !sameFsPath(existing, normalized) {
				next = append(next, existing)
			}
		}
		if len(next) == len(before) {
			return nil, nil, errors.New("GIT_PROFILE_NOT_FOUND: no such directory binding")
		}
	}
	var drift *SnippetDrift
	if len(next) == 0 {
		report := checkSnippetDrift(ctx, id, profile)
		if !report.Missing {
			drift = &report
		}
		profile.Gitdirs = nil
	} else {
		profile.Gitdirs = next
	}
	if err := rebuildIncludes(ctx, store); err != nil {
		return nil, nil, errors.New("GIT_SNIPPET_WRITE_FAILED: " + err.Error())
	}
	if len(identityGitdirs(profile)) == 0 {
		snippet, err := snippetPath(id)
		if err == nil {
			_ = os.Remove(snippet)
		}
	} else {
		snippet, err := snippetPath(id)
		if err == nil {
			if err := writeSnippetFile(ctx, snippet, profile); err != nil {
				return nil, nil, errors.New("GIT_SNIPPET_WRITE_FAILED: " + err.Error())
			}
		}
	}
	if err := store.Save(); err != nil {
		return nil, nil, err
	}
	return profile, drift, nil
}

// RemoveProfile drops a profile after unbinding all of its directories.
func RemoveProfile(ctx context.Context, store *Store, id string) error {
	if _, err := store.Profile(id); err != nil {
		return err
	}
	if _, _, err := Unbind(ctx, store, id, ""); err != nil {
		return err
	}
	delete(store.Profiles, id)
	return store.Save()
}
