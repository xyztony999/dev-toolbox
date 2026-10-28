package gitidentity

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// CredentialProxy is a conduit, never a store (ADR-017): tokens flow from
// the caller through the git credential protocol into the helper and are
// dropped. They are never written to disk, logs, errors or any response.

type CredentialEntry struct {
	Host     string `json:"host"`
	Username string `json:"username,omitempty"`
}

func parseCredBasis(hostInput string) (protocol, host, basis string, err error) {
	value := strings.TrimSpace(hostInput)
	if value == "" || strings.ContainsAny(value, " \t") {
		return "", "", "", fmt.Errorf("GIT_CREDENTIAL_REJECTED: invalid credential host %q", hostInput)
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	idx := strings.Index(value, "://")
	protocol = strings.ToLower(value[:idx])
	rest := value[idx+3:]
	if i := strings.IndexAny(rest, "/?#"); i >= 0 {
		rest = rest[:i]
	}
	if protocol != "https" && protocol != "http" {
		return "", "", "", fmt.Errorf("GIT_CREDENTIAL_REJECTED: only http(s) hosts are supported: %s", hostInput)
	}
	if rest == "" {
		return "", "", "", fmt.Errorf("GIT_CREDENTIAL_REJECTED: invalid credential host %q", hostInput)
	}
	return protocol, rest, protocol + "://" + rest, nil
}

func credentialPayload(fields map[string]string) string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(fields)+1)
	for _, key := range keys {
		lines = append(lines, key+"="+fields[key])
	}
	return strings.Join(lines, "\n") + "\n"
}

func helperPresent(ctx context.Context) bool {
	_, ok := gitAt(ctx, "").cfgGetEffective("credential.helper")
	return ok
}

// CredentialProbe asks the helper non-interactively; the password line is
// dropped immediately and only the username surfaces.
func CredentialProbe(ctx context.Context, hostInput, username string) (CredentialEntry, bool, error) {
	protocol, host, _, err := parseCredBasis(hostInput)
	if err != nil {
		return CredentialEntry{}, false, err
	}
	query := map[string]string{"protocol": protocol, "host": host}
	if username != "" {
		query["username"] = username
	}
	env := []string{"GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS=:", "GCM_INTERACTIVE=never"}
	out, ok := gitAt(ctx, "").withEnv(env...).withStdin(credentialPayload(query)).allowFail("credential", "fill")
	if !ok {
		return CredentialEntry{Host: host, Username: username}, false, nil
	}
	found := ""
	for _, line := range strings.Split(out, "\n") {
		if value, foundPrefix := strings.CutPrefix(strings.TrimSpace(line), "username="); foundPrefix {
			found = value
		}
		// password= lines are deliberately discarded
	}
	if found == "" {
		return CredentialEntry{Host: host, Username: username}, false, nil
	}
	return CredentialEntry{Host: host, Username: found}, true, nil
}

func hostOf(hostInput string) string {
	_, host, _, err := parseCredBasis(hostInput)
	if err != nil {
		return ""
	}
	return host
}

// CredentialStore feeds one credential to `git credential approve`. The
// token lives only in this call frame and is dropped on return; only the
// non-secret host+username hint is recorded.
func CredentialStore(ctx context.Context, store *Store, hostInput, username, token string) error {
	protocol, _, _, err := parseCredBasis(hostInput)
	if err != nil {
		return err
	}
	username = strings.TrimSpace(username)
	if username == "" || strings.ContainsAny(username, " \t") {
		return errors.New("GIT_CREDENTIAL_REJECTED: username must be non-blank without spaces")
	}
	if token == "" || strings.ContainsAny(token, "\r\n") {
		return errors.New("GIT_CREDENTIAL_REJECTED: token must be non-blank without newlines")
	}
	if !helperPresent(ctx) {
		return errors.New("GIT_CREDENTIAL_HELPER_UNAVAILABLE: no credential.helper is configured")
	}
	payload := credentialPayload(map[string]string{
		"protocol": protocol, "host": hostOf(hostInput), "username": username, "password": token,
	})
	if _, err := runGit(ctx, "", nil, payload, false, "credential", "approve"); err != nil {
		return errors.New("GIT_CREDENTIAL_REJECTED: helper refused the credential")
	}
	noteCredentialHost(store, hostOf(hostInput), username)
	return nil
}

// noteCredentialHost / dropCredentialHost maintain the non-secret hint list
// (port of gitid noteCredentialHost/dropCredentialHost); failures to record
// hints never affect the credential operation itself.
func noteCredentialHost(store *Store, host, username string) {
	if store == nil || host == "" || username == "" {
		return
	}
	kept := store.CredentialHosts[:0]
	for _, entry := range store.CredentialHosts {
		if !strings.EqualFold(entry.Host, host) || entry.Username != username {
			kept = append(kept, entry)
		}
	}
	store.CredentialHosts = append(kept, CredentialEntry{Host: host, Username: username})
	if len(store.CredentialHosts) > 50 {
		store.CredentialHosts = store.CredentialHosts[len(store.CredentialHosts)-50:]
	}
	_ = store.Save()
}

func dropCredentialHost(store *Store, host, username string) {
	if store == nil {
		return
	}
	kept := store.CredentialHosts[:0]
	for _, entry := range store.CredentialHosts {
		if !strings.EqualFold(entry.Host, host) || (username != "" && entry.Username != username) {
			kept = append(kept, entry)
		}
	}
	store.CredentialHosts = kept
	_ = store.Save()
}

// CredentialErase asks the helper to forget one credential (git credential
// reject, as in gitid) and drops the hint record.
func CredentialErase(ctx context.Context, store *Store, hostInput, username string) error {
	protocol, host, _, err := parseCredBasis(hostInput)
	if err != nil {
		return err
	}
	payload := credentialPayload(map[string]string{"protocol": protocol, "host": host})
	if username != "" {
		payload = credentialPayload(map[string]string{"protocol": protocol, "host": host, "username": username})
	}
	if _, err := runGit(ctx, "", nil, payload, false, "credential", "reject"); err != nil {
		return errors.New("GIT_CREDENTIAL_REJECTED: helper refused the erase request")
	}
	dropCredentialHost(store, host, username)
	return nil
}

// CredentialLogin drives the helper's own interactive flow (GCM window or
// terminal prompt). The fill output — including the secret — stays in this
// frame and is immediately replayed into approve; only the username hint
// is recorded.
func CredentialLogin(ctx context.Context, store *Store, hostInput, username string) (string, error) {
	protocol, host, _, err := parseCredBasis(hostInput)
	if err != nil {
		return "", err
	}
	if !helperPresent(ctx) {
		return "", errors.New("GIT_CREDENTIAL_HELPER_UNAVAILABLE: no credential.helper is configured")
	}
	query := map[string]string{"protocol": protocol, "host": host}
	if username != "" {
		query["username"] = strings.TrimSpace(username)
	}
	fillOut, err := runGit(ctx, "", nil, credentialPayload(query), true, "credential", "fill")
	if err != nil {
		return "", errors.New("GIT_CREDENTIAL_REJECTED: helper login did not complete")
	}
	gotUser, gotPassword := "", ""
	for _, line := range strings.Split(fillOut, "\n") {
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "username="); ok {
			gotUser = value
		}
		if value, ok := strings.CutPrefix(strings.TrimSpace(line), "password="); ok {
			gotPassword = value
		}
	}
	if gotUser == "" || gotPassword == "" {
		return "", errors.New("GIT_CREDENTIAL_REJECTED: login did not return a complete credential")
	}
	if _, err := runGit(ctx, "", nil, credentialPayload(map[string]string{
		"protocol": protocol, "host": host, "username": gotUser, "password": gotPassword,
	}), false, "credential", "approve"); err != nil {
		return "", errors.New("GIT_CREDENTIAL_REJECTED: helper refused the credential")
	}
	noteCredentialHost(store, host, gotUser)
	return gotUser, nil
}

// CredentialList merges configured account selectors with recorded login
// hints. No token is or can be included.
func CredentialList(ctx context.Context, store *Store, hostFilter string) []CredentialEntry {
	entries := []CredentialEntry{}
	seen := map[string]bool{}
	add := func(entry CredentialEntry) {
		if hostFilter != "" && !strings.EqualFold(entry.Host, strings.TrimPrefix(strings.TrimPrefix(hostFilter, "https://"), "http://")) {
			return
		}
		key := strings.ToLower(entry.Host) + "/" + entry.Username
		if seen[key] {
			return
		}
		seen[key] = true
		entries = append(entries, entry)
	}
	for _, account := range listAccountSelectors(ctx, "") {
		add(CredentialEntry{Host: account.Host, Username: account.Username})
	}
	for _, hint := range store.CredentialHosts {
		add(hint)
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Host != entries[j].Host {
			return entries[i].Host < entries[j].Host
		}
		return entries[i].Username < entries[j].Username
	})
	return entries
}
