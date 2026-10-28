package gitidentity

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"
)

// ConfigSource is where an effective value came from, mirroring gitid:
// repo local override, our includeIf snippet chain, global fallback, or
// undeclared.
type ConfigSource string

const (
	SourceLocal     ConfigSource = "local"
	SourceIncludeIf ConfigSource = "includeIf"
	SourceGlobal    ConfigSource = "global"
	SourceNone      ConfigSource = "none"
)

type KeyOrigin struct {
	Key    string       `json:"key"`
	Value  string       `json:"value,omitempty"`
	Source ConfigSource `json:"source"`
}

type EffectiveIdentity struct {
	Path       string      `json:"path"`
	RepoRoot   string      `json:"repoRoot,omitempty"`
	Resolved   string      `json:"resolved"` // profile id | manual | none
	ProfileID  string      `json:"profileId,omitempty"`
	Keys       []KeyOrigin `json:"keys"`
	InsteadOf  []InsteadOf `json:"insteadOf,omitempty"`
	Accounts   []Account   `json:"accounts,omitempty"`
	PromptLine string      `json:"promptLine,omitempty"`
}

func resolveOriginFile(origin, cwd string) string {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return ""
	}
	origin = filepath.FromSlash(origin)
	if filepath.IsAbs(origin) {
		return filepath.Clean(origin)
	}
	if cwd == "" {
		return filepath.Clean(origin)
	}
	return filepath.Join(cwd, origin)
}

func classifyConfigSource(ctx context.Context, origin, cwd string) ConfigSource {
	if origin == "" {
		return SourceNone
	}
	resolved := resolveOriginFile(strings.TrimPrefix(origin, "file:"), cwd)
	if root := gitAt(ctx, cwd).repoRoot(); root != "" {
		candidates := []string{filepath.Join(root, ".git", "config")}
		if absGitDir, ok := gitAt(ctx, root).allowFail("rev-parse", "--absolute-git-dir"); ok && absGitDir != "" {
			candidates = append(candidates, filepath.Join(strings.TrimSpace(absGitDir), "config"))
		}
		for _, candidate := range candidates {
			if sameFsPath(resolved, candidate) {
				return SourceLocal
			}
		}
	}
	if global := globalGitconfigPath(); global != "" && sameFsPath(resolved, global) {
		return SourceGlobal
	}
	return SourceIncludeIf
}

// cfgGetOrigin reads one key's effective value and classifies the origin
// file, following gitid's `git config --show-origin` parsing (including the
// Windows `//C:/...` spelling).
func cfgGetOrigin(ctx context.Context, key, cwd string) (string, ConfigSource) {
	out, ok := gitAt(ctx, cwd).allowFail("config", "--show-origin", "--get", key)
	if !ok || out == "" {
		return "", SourceNone
	}
	line := strings.ReplaceAll(out, "\r", "")
	tab := strings.Index(line, "\t")
	rawOrigin, value := "", line
	if tab >= 0 {
		rawOrigin, value = line[:tab], line[tab+1:]
	}
	origin := strings.TrimPrefix(rawOrigin, "file:")
	if strings.HasPrefix(origin, "//") && len(origin) > 2 && origin[2] != '/' {
		// Windows git prints file://C:/path as file://C:/path — strip the extra slash pair
		origin = origin[1:]
	}
	return strings.TrimSpace(value), classifyConfigSource(ctx, origin, cwd)
}

func listUrlRewrites(ctx context.Context, cwd string) []InsteadOf {
	out, ok := gitAt(ctx, cwd).allowFail("config", "--get-regexp", `^url\..+\.insteadof$`)
	if !ok || out == "" {
		return nil
	}
	rewrites := []InsteadOf{}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(parts[0])
		const marker = ".insteadof"
		if !strings.HasSuffix(key, marker) {
			continue
		}
		url := strings.TrimSuffix(strings.TrimPrefix(key, "url."), marker)
		if url == "" {
			continue
		}
		rewrites = append(rewrites, InsteadOf{Original: strings.TrimSpace(parts[1]), URL: url})
	}
	return rewrites
}

func listAccountSelectors(ctx context.Context, cwd string) []Account {
	out, ok := gitAt(ctx, cwd).allowFail("config", "--get-regexp", `^credential\.https://[^ ]+\.username$`)
	if !ok || out == "" {
		return nil
	}
	accounts := []Account{}
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(parts[0])
		const prefix, suffix = "credential.https://", ".username"
		if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
			continue
		}
		host := strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
		if host == "" {
			continue
		}
		accounts = append(accounts, Account{Host: host, Username: strings.TrimSpace(parts[1])})
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].Host < accounts[j].Host })
	return accounts
}

// EffectiveIdentity reports the identity in force at path with per-key
// sources; git itself is the source of truth (product invariant 1).
func ResolveEffective(ctx context.Context, store *Store, path string, prompt bool) (*EffectiveIdentity, error) {
	call := gitAt(ctx, path)
	result := &EffectiveIdentity{Path: path, Keys: []KeyOrigin{}}
	result.RepoRoot = call.repoRoot()

	name, nameSource := cfgGetOrigin(ctx, "user.name", path)
	email, emailSource := cfgGetOrigin(ctx, "user.email", path)
	result.Keys = append(result.Keys, KeyOrigin{Key: "user.name", Value: name, Source: nameSource})
	result.Keys = append(result.Keys, KeyOrigin{Key: "user.email", Value: email, Source: emailSource})

	for _, key := range []string{"user.signingkey", "commit.gpgsign", "gpg.format"} {
		value, source := cfgGetOrigin(ctx, key, path)
		result.Keys = append(result.Keys, KeyOrigin{Key: key, Value: value, Source: source})
	}
	result.InsteadOf = listUrlRewrites(ctx, path)
	result.Accounts = listAccountSelectors(ctx, path)

	switch {
	case name == "" || email == "":
		result.Resolved = "none"
	case store.MatchIdentity(name, email) != "":
		result.ProfileID = store.MatchIdentity(name, email)
		result.Resolved = result.ProfileID
	default:
		result.Resolved = "manual"
	}
	if prompt {
		switch {
		case result.Resolved == "none":
			result.PromptLine = "none"
		case result.Resolved == "manual":
			result.PromptLine = "manual"
		default:
			result.PromptLine = result.ProfileID
		}
	}
	return result, nil
}

// ImportGlobalIdentity turns the current global git identity into a profile.
func ImportGlobalIdentity(ctx context.Context, store *Store, asID string) (*Profile, error) {
	call := gitAt(ctx, "")
	name, _ := call.cfgGet("global", "user.name")
	email, _ := call.cfgGet("global", "user.email")
	if name == "" || email == "" {
		return nil, errors.New("GIT_PROFILE_INVALID: global git identity (user.name/user.email) is not configured")
	}
	if asID == "" {
		local := strings.ToLower(strings.SplitN(email, "@", 2)[0])
		local = strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
				return r
			}
			return '-'
		}, local)
		local = strings.Trim(local, "-")
		if local == "" || !idPattern.MatchString(local) {
			local = "imported"
		}
		asID = local
		for store.Profiles[asID] != nil {
			asID += "-2"
			if len(asID) > 64 {
				return nil, errors.New("GIT_PROFILE_INVALID: cannot derive a free profile id")
			}
		}
	}
	profile := Profile{ID: asID, Name: name, Email: email}
	if key, ok := call.cfgGet("global", "user.signingkey"); ok {
		profile.SigningKey = key
	}
	if sign, ok := call.cfgGet("global", "commit.gpgsign"); ok && sign == "true" {
		profile.GpgSign = true
	}
	if format, ok := call.cfgGet("global", "gpg.format"); ok && gpgFormats[format] {
		profile.GpgFormat = format
	}
	if err := store.Upsert(profile); err != nil {
		return nil, err
	}
	if err := store.Save(); err != nil {
		return nil, err
	}
	stored, err := store.Profile(asID)
	if err != nil {
		return nil, err
	}
	return stored, nil
}
