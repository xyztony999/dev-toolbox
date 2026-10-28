package gitidentity

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

var identityKeys = []string{"user.name", "user.email", "user.signingkey", "commit.gpgsign", "gpg.format"}

func insteadOfConfigKey(pair InsteadOf) string {
	return fmt.Sprintf("url.%s.insteadOf", pair.URL)
}

func accountConfigKey(account Account) string {
	return fmt.Sprintf("credential.https://%s.username", account.Host)
}

// profileConfigKeys lists the non-identity config keys a profile may own:
// custom kv, insteadOf rewrites and credential account selectors.
func profileConfigKeys(profile *Profile) []string {
	keys := []string{}
	for key := range profile.CustomKv {
		keys = append(keys, key)
	}
	for _, pair := range profile.InsteadOf {
		keys = append(keys, insteadOfConfigKey(pair))
	}
	for _, account := range profile.Accounts {
		keys = append(keys, accountConfigKey(account))
	}
	return keys
}

// ConfigChange is one planned or executed config key transition; nil From/To
// means the key was or will be unset.
type ConfigChange struct {
	Action string  `json:"action"`
	Key    string  `json:"key"`
	Scope  string  `json:"scope"`
	Target string  `json:"target,omitempty"`
	From   *string `json:"from,omitempty"`
	To     *string `json:"to,omitempty"`
}

func strPtr(value string) *string { return &value }

// ApplyProfile ports gitid applyIdentity: full overwrite semantics — the
// profile's keys are written, every managed key the profile lacks is unset
// (including gpg.format), and keys owned by the previously applied profile
// (matched by name+email in the target scope) are cleaned up so URL
// rewrites never silently redirect pushes (ADR-016/017). dryRun computes
// the identical change list without writing.
func ApplyProfile(ctx context.Context, store *Store, id, scope, cwd string, dryRun bool) (changes []ConfigChange, err error) {
	profile, err := store.Profile(id)
	if err != nil {
		return nil, err
	}
	if scope == "local" && cwd == "" {
		return nil, fmt.Errorf("GIT_REPO_NOT_FOUND: local apply needs a repository path")
	}
	// config writes go through call.must which panics on git failure; keep
	// the panic inside this frame and surface it as a stable error.
	defer func() {
		if r := recover(); r != nil {
			if gitErr, ok := r.(gitError); ok {
				changes, err = nil, fmt.Errorf("GIT_CONFIG_WRITE_FAILED: %s", gitErr.msg)
				return
			}
			panic(r)
		}
	}()
	call := gitAt(ctx, cwd)
	watch := map[string]bool{}
	for _, key := range identityKeys {
		watch[key] = true
	}
	for _, key := range profileConfigKeys(profile) {
		watch[key] = true
	}

	// Keys owned by the previously applied profile but not by this one.
	cleanupKeys := []string{}
	prevName, _ := call.cfgGet(scope, "user.name")
	prevEmail, _ := call.cfgGet(scope, "user.email")
	if prevID := store.MatchIdentity(prevName, prevEmail); prevID != "" && prevID != id {
		if prev, err := store.Profile(prevID); err == nil {
			own := map[string]bool{}
			for _, key := range profileConfigKeys(profile) {
				own[key] = true
			}
			for _, key := range profileConfigKeys(prev) {
				if !own[key] && !watch[key] {
					cleanupKeys = append(cleanupKeys, key)
					watch[key] = true
				}
			}
		}
	}
	sort.Strings(cleanupKeys)

	before := map[string]*string{}
	keys := []string{}
	for key := range watch {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if value, ok := call.cfgGet(scope, key); ok {
			before[key] = strPtr(value)
		} else {
			before[key] = nil
		}
	}

	if !dryRun {
		for _, key := range cleanupKeys {
			call.cfgUnset(scope, key)
		}
		call.cfgSet(scope, "user.name", profile.Name)
		call.cfgSet(scope, "user.email", profile.Email)
		if profile.SigningKey != "" {
			call.cfgSet(scope, "user.signingkey", profile.SigningKey)
		} else {
			call.cfgUnset(scope, "user.signingkey")
		}
		if profile.GpgSign {
			call.cfgSet(scope, "commit.gpgsign", "true")
		} else {
			call.cfgUnset(scope, "commit.gpgsign")
		}
		if profile.GpgFormat != "" {
			call.cfgSet(scope, "gpg.format", profile.GpgFormat)
		} else {
			call.cfgUnset(scope, "gpg.format")
		}
		// Fixed key order keeps custom keys deterministic across rewrites.
		customKeys := make([]string, 0, len(profile.CustomKv))
		for key := range profile.CustomKv {
			customKeys = append(customKeys, key)
		}
		sort.Strings(customKeys)
		for _, key := range customKeys {
			call.cfgSet(scope, key, profile.CustomKv[key])
		}
		for _, pair := range profile.InsteadOf {
			call.cfgSet(scope, insteadOfConfigKey(pair), pair.Original)
		}
		for _, account := range profile.Accounts {
			call.cfgSet(scope, accountConfigKey(account), account.Username)
		}
	}

	after := map[string]*string{}
	setAfter := func(key string, value *string) { after[key] = value }
	setAfter("user.name", strPtr(profile.Name))
	setAfter("user.email", strPtr(profile.Email))
	if profile.SigningKey != "" {
		setAfter("user.signingkey", strPtr(profile.SigningKey))
	}
	if profile.GpgSign {
		setAfter("commit.gpgsign", strPtr("true"))
	}
	if profile.GpgFormat != "" {
		setAfter("gpg.format", strPtr(profile.GpgFormat))
	}
	for _, key := range customKeysSorted(profile) {
		setAfter(key, strPtr(profile.CustomKv[key]))
	}
	for _, pair := range profile.InsteadOf {
		setAfter(insteadOfConfigKey(pair), strPtr(pair.Original))
	}
	for _, account := range profile.Accounts {
		setAfter(accountConfigKey(account), strPtr(account.Username))
	}
	for _, key := range cleanupKeys {
		setAfter(key, nil)
	}

	changes = []ConfigChange{}
	for _, key := range keys {
		from, to := before[key], after[key]
		if !sameStrPtr(from, to) {
			changes = append(changes, ConfigChange{Action: changeAction(from, to), Key: key, Scope: scope, From: from, To: to})
		}
	}
	return changes, nil
}

func customKeysSorted(profile *Profile) []string {
	keys := make([]string, 0, len(profile.CustomKv))
	for key := range profile.CustomKv {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sameStrPtr(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func changeAction(from, to *string) string {
	switch {
	case from == nil && to != nil:
		return "set"
	case from != nil && to == nil:
		return "unset"
	default:
		return "update"
	}
}

// ApplyTargetResult reports one repository in a batch apply; batches allow
// partial failure and never fake an atomic success.
type ApplyTargetResult struct {
	Target  string         `json:"target"`
	Status  string         `json:"status"` // success | failed | skipped
	Changes []ConfigChange `json:"changes,omitempty"`
	Error   string         `json:"error,omitempty"`
}

func ApplyToRepos(ctx context.Context, store *Store, id string, repos []string, dryRun bool) []ApplyTargetResult {
	results := []ApplyTargetResult{}
	for _, repo := range repos {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			continue
		}
		root := gitAt(ctx, repo).repoRoot()
		if root == "" {
			results = append(results, ApplyTargetResult{Target: repo, Status: "failed", Error: "GIT_NOT_A_REPOSITORY"})
			continue
		}
		changes, err := ApplyProfile(ctx, store, id, "local", root, dryRun)
		if err != nil {
			results = append(results, ApplyTargetResult{Target: root, Status: "failed", Error: err.Error()})
			continue
		}
		status := "success"
		if dryRun {
			status = "skipped"
		}
		results = append(results, ApplyTargetResult{Target: root, Status: status, Changes: changes})
	}
	return results
}

// UnsetIdentity clears the managed identity keys (and the keys of whatever
// profile currently matches) from one scope.
func UnsetIdentity(ctx context.Context, store *Store, scope, cwd string) []ConfigChange {
	call := gitAt(ctx, cwd)
	watch := map[string]bool{}
	for _, key := range identityKeys {
		watch[key] = true
	}
	name, _ := call.cfgGet(scope, "user.name")
	email, _ := call.cfgGet(scope, "user.email")
	if prevID := store.MatchIdentity(name, email); prevID != "" {
		if prev, err := store.Profile(prevID); err == nil {
			for _, key := range profileConfigKeys(prev) {
				watch[key] = true
			}
		}
	}
	changes := []ConfigChange{}
	sorted := make([]string, 0, len(watch))
	for key := range watch {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)
	for _, key := range sorted {
		if value, ok := call.cfgGet(scope, key); ok {
			call.cfgUnset(scope, key)
			changes = append(changes, ConfigChange{Action: "unset", Key: key, Scope: scope, From: strPtr(value), To: nil})
		}
	}
	return changes
}
