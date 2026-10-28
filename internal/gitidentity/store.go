// Package gitidentity ports the gitid identity semantics to the shared Go
// application service (migration source: gitid e33e728). Git itself stays
// the source of truth: the service only writes through `git config` on
// explicit user actions and reads back through git.
package gitidentity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	storeVersion    = 1
	defaultScanDept = 6
	maxScanDepth    = 12
)

var (
	idPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	gpgFormats   = map[string]bool{"openpgp": true, "ssh": true, "x509": true}
)

type InsteadOf struct {
	Original string `json:"original"`
	URL      string `json:"url"`
}

type Account struct {
	Host     string `json:"host"`
	Username string `json:"username"`
}

// Profile is the IdentityProfile aggregate. Tokens never live here.
type Profile struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Email      string            `json:"email"`
	SigningKey string            `json:"signingKey,omitempty"`
	GpgSign    bool              `json:"gpgSign,omitempty"`
	GpgFormat  string            `json:"gpgFormat,omitempty"`
	CustomKv   map[string]string `json:"customKv,omitempty"`
	InsteadOf  []InsteadOf       `json:"insteadOf,omitempty"`
	Accounts   []Account         `json:"accounts,omitempty"`
	Gitdirs    []string          `json:"gitdirs,omitempty"`
}

type Settings struct {
	ScanRoot  string `json:"scanRoot"`
	ScanDepth int    `json:"scanDepth"`
}

type ImportRecord struct {
	SourceKind   string    `json:"sourceKind"`
	SourceDigest string    `json:"sourceDigest"`
	BackupPath   string    `json:"backupPath"`
	ImportedAt   time.Time `json:"importedAt"`
	Created      int       `json:"created"`
	Skipped      int       `json:"skipped"`
	Conflicts    int       `json:"conflicts"`
}

type Store struct {
	Version       int                 `json:"version"`
	Profiles      map[string]*Profile `json:"profiles"`
	Settings      Settings            `json:"settings"`
	LegacyImports []ImportRecord      `json:"legacyImports"`
	// CredentialHosts are non-secret login hints (host+username only) so
	// the credential list can enumerate what the protocol cannot; it never
	// holds tokens (port of gitid credentialHosts).
	CredentialHosts []CredentialEntry `json:"credentialHosts,omitempty"`
}

func StorePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "DevToolbox", "git-identity.json"), nil
}

// legacyGitidStorePath lists the gitid config.json candidates in the same
// order gitid itself resolves them; the legacy store is only ever read.
func legacyGitidStorePath() []string {
	var candidates []string
	if env := os.Getenv("GITID_CONFIG"); env != "" {
		candidates = append(candidates, env)
	}
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			candidates = append(candidates, filepath.Join(appdata, "gitid", "config.json"))
		}
	}
	xdg := os.Getenv("XDG_CONFIG_HOME")
	if xdg == "" {
		if home, err := os.UserHomeDir(); err == nil {
			xdg = filepath.Join(home, ".config")
			candidates = append(candidates, filepath.Join(xdg, "gitid", "config.json"))
		}
	} else {
		candidates = append(candidates, filepath.Join(xdg, "gitid", "config.json"))
	}
	return candidates
}

func NewStore() *Store {
	return &Store{Version: storeVersion, Profiles: map[string]*Profile{}, Settings: Settings{ScanDepth: defaultScanDept}, LegacyImports: []ImportRecord{}}
}

func Load() (*Store, error) {
	path, err := StorePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return NewStore(), nil
	}
	if err != nil {
		return nil, err
	}
	var stored Store
	if err := json.Unmarshal(data, &stored); err != nil || stored.Profiles == nil {
		return nil, errors.New("GIT_IDENTITY_STATE_CORRUPT")
	}
	if stored.Settings.ScanDepth < 1 {
		stored.Settings.ScanDepth = defaultScanDept
	}
	if stored.LegacyImports == nil {
		stored.LegacyImports = []ImportRecord{}
	}
	return &stored, nil
}

func (s *Store) Save() error {
	path, err := StorePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".git-identity-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func (s *Store) lock() (func(), error) {
	path, err := StorePath()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	lockPath := path + ".lock"
	deadline := time.Now().Add(10 * time.Second)
	for {
		f, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
			_ = f.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, errors.New("GIT_IDENTITY_STATE_BUSY")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (s *Store) Profile(id string) (*Profile, error) {
	p, ok := s.Profiles[id]
	if !ok {
		return nil, fmt.Errorf("GIT_PROFILE_NOT_FOUND: identity %q does not exist", id)
	}
	return p, nil
}

// Validate runs the profile field checks without touching the store.
func (p Profile) Validate() error {
	if !idPattern.MatchString(p.ID) {
		return fmt.Errorf("GIT_PROFILE_INVALID: id must match [A-Za-z0-9][A-Za-z0-9._-]*, got %q", p.ID)
	}
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("GIT_PROFILE_INVALID: name must not be empty")
	}
	if !emailPattern.MatchString(p.Email) {
		return fmt.Errorf("GIT_PROFILE_INVALID: email %q is not valid", p.Email)
	}
	if p.GpgFormat != "" && !gpgFormats[p.GpgFormat] {
		return errors.New("GIT_PROFILE_INVALID: gpg format must be openpgp, ssh or x509")
	}
	for _, pair := range p.InsteadOf {
		if strings.TrimSpace(pair.Original) == "" || strings.TrimSpace(pair.URL) == "" ||
			strings.ContainsAny(pair.Original, " \t") || strings.ContainsAny(pair.URL, " \t") ||
			strings.TrimSpace(pair.Original) == strings.TrimSpace(pair.URL) {
			return errors.New("GIT_PROFILE_INVALID: insteadOf entries must be distinct non-blank original/url pairs")
		}
	}
	for _, account := range p.Accounts {
		if strings.TrimSpace(account.Host) == "" || strings.TrimSpace(account.Username) == "" ||
			strings.ContainsAny(account.Host, " \t") || strings.ContainsAny(account.Username, " \t") {
			return errors.New("GIT_PROFILE_INVALID: accounts must be non-blank host/username pairs")
		}
	}
	return nil
}

// Upsert validates and stores a profile; an existing id is overwritten but
// its directory bindings are preserved (same as gitid upsertIdentity).
func (s *Store) Upsert(profile Profile) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	seenRewrite := map[string]bool{}
	rewrites := []InsteadOf{}
	for _, pair := range profile.InsteadOf {
		original, url := strings.TrimSpace(pair.Original), strings.TrimSpace(pair.URL)
		if original == "" || url == "" || strings.ContainsAny(original, " \t") || strings.ContainsAny(url, " \t") {
			return errors.New("GIT_PROFILE_INVALID: insteadOf entries must be non-blank original/url pairs")
		}
		if original == url {
			return errors.New("GIT_PROFILE_INVALID: insteadOf original and replacement must differ")
		}
		key := original + "=>" + url
		if seenRewrite[key] {
			continue
		}
		seenRewrite[key] = true
		rewrites = append(rewrites, InsteadOf{Original: original, URL: url})
	}
	profile.InsteadOf = rewrites
	seenHost := map[string]bool{}
	accounts := []Account{}
	for _, account := range profile.Accounts {
		host := strings.TrimSpace(account.Host)
		host = strings.TrimPrefix(host, "https://")
		host = strings.TrimPrefix(host, "http://")
		host = strings.TrimRight(host, "/")
		username := strings.TrimSpace(account.Username)
		if host == "" || username == "" || strings.ContainsAny(host, " \t") || strings.ContainsAny(username, " \t") {
			return errors.New("GIT_PROFILE_INVALID: accounts must be non-blank host/username pairs")
		}
		fold := strings.ToLower(host)
		if seenHost[fold] {
			continue
		}
		seenHost[fold] = true
		accounts = append(accounts, Account{Host: host, Username: username})
	}
	profile.Accounts = accounts
	if existing := s.Profiles[profile.ID]; existing != nil && len(existing.Gitdirs) > 0 {
		profile.Gitdirs = existing.Gitdirs
	}
	s.Profiles[profile.ID] = &profile
	return nil
}

func (s *Store) MatchIdentity(name, email string) string {
	if name == "" && email == "" {
		return ""
	}
	for id, profile := range s.Profiles {
		if profile.Name == name && profile.Email == email {
			return id
		}
	}
	return ""
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
