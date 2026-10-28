package gitidentity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var scanSkipDirs = map[string]bool{
	"node_modules": true, ".git": true, ".svn": true, ".hg": true, "target": true,
	"dist": true, "build": true, "vendor": true, ".venv": true, "__pycache__": true,
	".cache": true, ".npm": true, ".idea": true, ".vscode": true,
}

type RemoteInfo struct {
	Name     string   `json:"name"`
	URL      string   `json:"url,omitempty"`
	PushURLs []string `json:"pushUrls,omitempty"`
	Mirror   bool     `json:"mirror"`
	Account  string   `json:"account,omitempty"`
}

type ScanRow struct {
	Path       string       `json:"path"`
	Relative   string       `json:"relative"`
	IdentityID string       `json:"identityId,omitempty"`
	Status     string       `json:"status"` // ok | missing | manual
	HasLocal   bool         `json:"hasLocal"`
	LocalName  string       `json:"localName,omitempty"`
	LocalEmail string       `json:"localEmail,omitempty"`
	Source     ConfigSource `json:"source"`
	Remotes    []RemoteInfo `json:"remotes"`
}

type ScanSummary struct {
	Total     int `json:"total"`
	WithLocal int `json:"withLocal"`
	WithBind  int `json:"withIncludeIf"`
	Missing   int `json:"missing"`
	Manual    int `json:"manual"`
	Mirrored  int `json:"mirrored"`
}

type ScanResult struct {
	Root    string      `json:"root"`
	Rows    []ScanRow   `json:"rows"`
	Summary ScanSummary `json:"summary"`
}

func getRemotes(ctx context.Context, cwd string) []RemoteInfo {
	out, ok := gitAt(ctx, cwd).allowFail("config", "--local", "--get-regexp", `^remote\..+\.(url|pushurl)$`)
	byName := map[string]*RemoteInfo{}
	order := []string{}
	if ok && out != "" {
		for _, line := range strings.Split(out, "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), " ", 2)
			if len(parts) != 2 {
				continue
			}
			key, value := parts[0], strings.TrimSpace(parts[1])
			segments := strings.Split(key, ".")
			if len(segments) < 3 {
				continue
			}
			kind := segments[len(segments)-1]
			name := strings.Join(segments[1:len(segments)-1], ".")
			if name == "" {
				continue
			}
			entry, exists := byName[name]
			if !exists {
				entry = &RemoteInfo{Name: name}
				byName[name] = entry
				order = append(order, name)
			}
			switch kind {
			case "url", "insteadof": // "url" wins as fetch target
				if entry.URL == "" {
					entry.URL = value
				}
			case "pushurl":
				entry.PushURLs = append(entry.PushURLs, value)
			}
		}
	}
	sort.Strings(order)
	remotes := []RemoteInfo{}
	for _, name := range order {
		entry := byName[name]
		entry.Mirror = len(entry.PushURLs) >= 2
		if basis, ok := httpsBasis(entry.URL); ok {
			entry.Account, _ = gitAt(ctx, cwd).cfgGetEffective("credential." + basis + ".username")
		}
		remotes = append(remotes, *entry)
	}
	return remotes
}

func httpsBasis(url string) (string, bool) {
	lower := strings.ToLower(url)
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(lower, prefix) {
			rest := url[len(prefix):]
			if idx := strings.IndexAny(rest, "/?#"); idx >= 0 {
				rest = rest[:idx]
			}
			if rest == "" {
				return "", false
			}
			return strings.ToLower(prefix) + rest, true
		}
	}
	return "", false
}

func findRepos(root string, maxDepth int) []string {
	found := []string{}
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}
		for _, entry := range entries {
			if entry.Name() == ".git" {
				found = append(found, dir)
				return
			}
		}
		if depth <= 0 {
			return
		}
		for _, entry := range entries {
			if !entry.IsDir() || scanSkipDirs[entry.Name()] {
				continue
			}
			walk(filepath.Join(dir, entry.Name()), depth-1)
		}
	}
	walk(root, maxDepth)
	sort.Strings(found)
	return found
}

// ScanRepos walks root (up to depth) and read-only audits each repository's
// identity and mirror status. No project command is ever executed.
func ScanRepos(ctx context.Context, store *Store, root string, depth int) (*ScanResult, error) {
	if depth < 1 || depth > maxScanDepth {
		return nil, fmt.Errorf("GIT_SCAN_ROOT_INVALID: depth must be 1-%d", maxScanDepth)
	}
	resolved := expandUserPath(root)
	if info, err := os.Stat(resolved); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("GIT_SCAN_ROOT_INVALID: directory does not exist: %s", resolved)
	}
	base, err := filepath.Abs(resolved)
	if err != nil {
		return nil, err
	}
	result := &ScanResult{Root: base, Rows: []ScanRow{}}
	for _, repo := range findRepos(base, depth) {
		call := gitAt(ctx, repo)
		localName, _ := call.cfgGet("local", "user.name")
		localEmail, _ := call.cfgGet("local", "user.email")
		effectiveName, _ := call.cfgGetEffective("user.name")
		effectiveEmail, _ := call.cfgGetEffective("user.email")
		hasLocal := localName != "" || localEmail != ""
		identityID := store.MatchIdentity(effectiveName, effectiveEmail)
		status := "ok"
		if effectiveName == "" || effectiveEmail == "" {
			status = "missing"
		} else if identityID == "" {
			status = "manual"
		}
		row := ScanRow{
			Path: repo, Relative: relativeOrDot(base, repo), IdentityID: identityID, Status: status,
			HasLocal: hasLocal, LocalName: localName, LocalEmail: localEmail, Remotes: getRemotes(ctx, repo),
		}
		nameOrigin, nameSource := cfgGetOrigin(ctx, "user.name", repo)
		emailOrigin, emailSource := cfgGetOrigin(ctx, "user.email", repo)
		row.Source = nameSource
		if nameSource == SourceNone {
			row.Source = emailSource
		}
		if hasLocal && row.Source != SourceLocal && localName == nameOrigin && localEmail == emailOrigin {
			row.Source = SourceLocal
		}
		result.Rows = append(result.Rows, row)
	}
	result.Summary = ScanSummary{
		Total: len(result.Rows),
	}
	for _, row := range result.Rows {
		if row.HasLocal {
			result.Summary.WithLocal++
		}
		if row.Source == SourceIncludeIf {
			result.Summary.WithBind++
		}
		if row.Status == "missing" {
			result.Summary.Missing++
		}
		if row.Status == "manual" {
			result.Summary.Manual++
		}
		for _, remote := range row.Remotes {
			if remote.Mirror {
				result.Summary.Mirrored++
				break
			}
		}
	}
	return result, nil
}

func relativeOrDot(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil || rel == "" {
		return "."
	}
	return rel
}

// SaveScanSettings records the default scan root and depth.
func SaveScanSettings(store *Store, root string, depth int) error {
	if depth < 1 || depth > maxScanDepth {
		return fmt.Errorf("GIT_SCAN_ROOT_INVALID: depth must be 1-%d", maxScanDepth)
	}
	store.Settings.ScanRoot = strings.TrimSpace(root)
	store.Settings.ScanDepth = depth
	return store.Save()
}

// ImportScanManuals absorbs scanned hand-configured identities (status
// manual) as profiles after preview+confirm; same name+email is skipped.
func ImportScanManuals(ctx context.Context, store *Store, root string, depth int, onlyPaths []string, dryRun bool) ([]Profile, error) {
	result, err := ScanRepos(ctx, store, root, depth)
	if err != nil {
		return nil, err
	}
	wanted := map[string]bool{}
	for _, path := range onlyPaths {
		abs, err := filepath.Abs(expandUserPath(strings.TrimSpace(path)))
		if err == nil {
			wanted[strings.ToLower(abs)] = true
		}
	}
	imported := []Profile{}
	for _, row := range result.Rows {
		if row.Status != "manual" {
			continue
		}
		if len(wanted) > 0 && !wanted[strings.ToLower(row.Path)] {
			continue
		}
		call := gitAt(ctx, row.Path)
		name, _ := call.cfgGetEffective("user.name")
		email, _ := call.cfgGetEffective("user.email")
		if name == "" || email == "" {
			continue
		}
		if store.MatchIdentity(name, email) != "" {
			continue
		}
		profile := Profile{ID: uniqueProfileID(store, idFromEmail(email)), Name: name, Email: email}
		if key, ok := call.cfgGetEffective("user.signingkey"); ok {
			profile.SigningKey = key
		}
		if sign, ok := call.cfgGetEffective("commit.gpgsign"); ok && sign == "true" {
			profile.GpgSign = true
		}
		if format, ok := call.cfgGetEffective("gpg.format"); ok && gpgFormats[format] {
			profile.GpgFormat = format
		}
		imported = append(imported, profile)
	}
	if dryRun {
		return imported, nil
	}
	for _, profile := range imported {
		if err := store.Upsert(profile); err != nil {
			return nil, err
		}
	}
	if len(imported) > 0 {
		if err := store.Save(); err != nil {
			return nil, err
		}
	}
	return imported, nil
}

func idFromEmail(email string) string {
	local := strings.SplitN(email, "@", 2)[0]
	local = strings.ToLower(local)
	out := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		return '-'
	}, local)
	out = strings.Trim(out, "-")
	if out == "" || !idPattern.MatchString(out) {
		return "imported"
	}
	return out
}

func uniqueProfileID(store *Store, base string) string {
	if store.Profiles[base] == nil {
		return base
	}
	for n := 2; ; n++ {
		candidate := fmt.Sprintf("%s-%d", base, n)
		if store.Profiles[candidate] == nil {
			return candidate
		}
	}
}

// AuthorCount is one author line in a commit audit.
type AuthorCount struct {
	Name      string `json:"name"`
	Email     string `json:"email"`
	Count     int    `json:"count"`
	ProfileID string `json:"profileId,omitempty"`
	Polluted  *bool  `json:"polluted,omitempty"`
}

type AuditReport struct {
	Repo              string        `json:"repo"`
	ExpectedProfileID string        `json:"expectedProfileId,omitempty"`
	Authors           []AuthorCount `json:"authors"`
	MailmapSuggestion string        `json:"mailmapSuggestion,omitempty"`
	StrictPassed      *bool         `json:"strictPassed,omitempty"`
	Empty             bool          `json:"empty"`
}

func buildMailmapSuggestion(authors []AuthorCount, expect *Profile) string {
	if len(authors) == 0 {
		return ""
	}
	lines := []string{}
	if expect != nil {
		lines = append(lines, fmt.Sprintf("%s <%s>", expect.Name, expect.Email))
		for _, author := range authors {
			if author.Name != expect.Name || author.Email != expect.Email {
				lines = append(lines, fmt.Sprintf("%s <%s> %s <%s>", expect.Name, expect.Email, author.Name, author.Email))
			}
		}
		return strings.Join(lines, "\n")
	}
	byEmail := map[string][]AuthorCount{}
	emails := []string{}
	for _, author := range authors {
		if _, exists := byEmail[author.Email]; !exists {
			emails = append(emails, author.Email)
		}
		byEmail[author.Email] = append(byEmail[author.Email], author)
	}
	sort.Strings(emails)
	for _, email := range emails {
		group := byEmail[email]
		sort.SliceStable(group, func(i, j int) bool {
			if group[i].Count != group[j].Count {
				return group[i].Count > group[j].Count
			}
			return group[i].Name < group[j].Name
		})
		canonical := group[0]
		lines = append(lines, fmt.Sprintf("%s <%s>", canonical.Name, email))
		for _, author := range group {
			if author.Name != canonical.Name {
				lines = append(lines, fmt.Sprintf("%s <%s> %s <%s>", canonical.Name, email, author.Name, email))
			}
		}
	}
	return strings.Join(lines, "\n")
}

// AuditRepo aggregates author distribution for all branches and marks
// authors; mailmap output is a suggestion only — .mailmap is never written
// (product rule), and strict mode is a gate, not a repair.
func AuditRepo(ctx context.Context, store *Store, repoPath, expectID string) (*AuditReport, error) {
	cwd := repoPath
	root := gitAt(ctx, cwd).repoRoot()
	if root == "" {
		return nil, fmt.Errorf("GIT_NOT_A_REPOSITORY: %s", cwd)
	}
	report := &AuditReport{Repo: root, Authors: []AuthorCount{}}
	var expect *Profile
	if expectID != "" {
		var err error
		if expect, err = store.Profile(expectID); err != nil {
			return nil, err
		}
		report.ExpectedProfileID = expectID
	}
	out, ok := gitAt(ctx, root).allowFail("log", "--all", "--format=%aN%x09%aE")
	if !ok {
		combined := strings.ToLower(out)
		empty := strings.Contains(combined, "does not have any commits") ||
			strings.Contains(combined, "no commits yet") ||
			strings.Contains(combined, "bad default revision") ||
			strings.Contains(combined, "unknown revision") ||
			strings.Contains(combined, "ambiguous argument")
		if !empty {
			return nil, errors.New("GIT_AUDIT_FAILED: git log unavailable for this repository")
		}
		report.Empty = true
		return report, nil
	}
	counts := map[string]int{}
	order := []string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		name, email := parts[0], ""
		if len(parts) == 2 {
			email = parts[1]
		}
		key := name + "\x00" + email
		if _, exists := counts[key]; !exists {
			order = append(order, key)
		}
		counts[key]++
	}
	for _, key := range order {
		segments := strings.SplitN(key, "\x00", 2)
		name, email := segments[0], segments[1]
		author := AuthorCount{Name: name, Email: email, Count: counts[key], ProfileID: store.MatchIdentity(name, email)}
		if expect != nil {
			polluted := email != expect.Email
			author.Polluted = &polluted
		}
		report.Authors = append(report.Authors, author)
	}
	sort.SliceStable(report.Authors, func(i, j int) bool {
		if report.Authors[i].Count != report.Authors[j].Count {
			return report.Authors[i].Count > report.Authors[j].Count
		}
		if report.Authors[i].Email != report.Authors[j].Email {
			return report.Authors[i].Email < report.Authors[j].Email
		}
		return report.Authors[i].Name < report.Authors[j].Name
	})
	report.MailmapSuggestion = buildMailmapSuggestion(report.Authors, expect)
	if expect != nil {
		pollutedExists := false
		for _, author := range report.Authors {
			if author.Polluted != nil && *author.Polluted {
				pollutedExists = true
			}
		}
		passed := !pollutedExists
		report.StrictPassed = &passed
	}
	return report, nil
}
