package gitidentity

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// legacy import: the gitid store is only ever read; import is preview ->
// backup -> idempotent by source digest (same pattern as user-runtime-state
// migration, ADR-008).

type legacyProfile struct {
	Name       string            `json:"name"`
	Email      string            `json:"email"`
	SigningKey string            `json:"signingkey,omitempty"`
	GpgSign    bool              `json:"gpgsign,omitempty"`
	GpgFormat  string            `json:"gpgFormat,omitempty"`
	InsteadOf  []InsteadOf       `json:"insteadOf,omitempty"`
	Accounts   []Account         `json:"accounts,omitempty"`
	Extra      map[string]string `json:"extra,omitempty"`
	Gitdirs    []string          `json:"gitdirs,omitempty"`
}

type legacyStore struct {
	Version    int                       `json:"version"`
	Identities map[string]*legacyProfile `json:"identities"`
}

type LegacyPreview struct {
	Found           bool      `json:"found"`
	SourcePath      string    `json:"sourcePath,omitempty"`
	Readable        bool      `json:"readable"`
	Valid           bool      `json:"valid"`
	SourceDigest    string    `json:"sourceDigest,omitempty"`
	Profiles        []Profile `json:"profiles"`
	IdConflicts     []string  `json:"idConflicts"`
	DuplicateEmails []string  `json:"duplicateEmails"`
	BackupDirectory string    `json:"backupDirectory,omitempty"`
}

func findLegacyStorePath() string {
	for _, candidate := range legacyGitidStorePath() {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func PreviewLegacy(ctx context.Context, store *Store) (*LegacyPreview, error) {
	preview := &LegacyPreview{Profiles: []Profile{}, IdConflicts: []string{}, DuplicateEmails: []string{}}
	source := findLegacyStorePath()
	if source == "" {
		return preview, nil
	}
	preview.Found = true
	preview.SourcePath = source
	data, err := os.ReadFile(source)
	if err != nil {
		return preview, nil
	}
	preview.Readable = true
	var old legacyStore
	if err := json.Unmarshal(data, &old); err != nil || old.Identities == nil {
		return preview, nil
	}
	preview.Valid = true
	preview.SourceDigest = digestBytes(data)

	ids := make([]string, 0, len(old.Identities))
	for id := range old.Identities {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	// Two passes: profiles whose original id is valid keep it and claim
	// their email first; invalid ids are re-derived only afterwards so a
	// valid "work" never loses its email to a garbage-id twin.
	type candidate struct {
		profile Profile
		validID bool
	}
	candidates := []candidate{}
	seenEmail := map[string]string{}
	claimedIDs := map[string]bool{}
	for _, id := range ids {
		oldProfile := old.Identities[id]
		if oldProfile == nil || oldProfile.Name == "" || oldProfile.Email == "" {
			continue
		}
		candidates = append(candidates, candidate{profile: Profile{
			ID: id, Name: oldProfile.Name, Email: oldProfile.Email,
			SigningKey: oldProfile.SigningKey, GpgSign: oldProfile.GpgSign,
			GpgFormat: oldProfile.GpgFormat, InsteadOf: oldProfile.InsteadOf,
			Accounts: oldProfile.Accounts, CustomKv: oldProfile.Extra,
		}, validID: idPattern.MatchString(id)})
	}
	for _, entry := range candidates {
		profile := entry.profile
		if !entry.validID {
			continue
		}
		if existing := store.Profiles[profile.ID]; existing != nil {
			if existing.Name != profile.Name || existing.Email != profile.Email {
				preview.IdConflicts = append(preview.IdConflicts, profile.ID)
				continue
			}
		}
		if prevID := seenEmail[profile.Email]; prevID != "" {
			preview.DuplicateEmails = append(preview.DuplicateEmails, prevID+"/"+profile.ID)
			continue
		}
		seenEmail[profile.Email] = profile.ID
		claimedIDs[profile.ID] = true
		preview.Profiles = append(preview.Profiles, profile)
	}
	for _, entry := range candidates {
		if entry.validID {
			continue
		}
		profile := entry.profile
		profile.ID = uniqueProfileID(store, idFromEmail(profile.Email))
		for claimedIDs[profile.ID] {
			profile.ID += "-x"
		}
		if existing := store.Profiles[profile.ID]; existing != nil {
			if existing.Name != profile.Name || existing.Email != profile.Email {
				preview.IdConflicts = append(preview.IdConflicts, profile.ID)
				continue
			}
		}
		if prevID := seenEmail[profile.Email]; prevID != "" {
			preview.DuplicateEmails = append(preview.DuplicateEmails, prevID+"/"+profile.ID)
			continue
		}
		seenEmail[profile.Email] = profile.ID
		claimedIDs[profile.ID] = true
		preview.Profiles = append(preview.Profiles, profile)
	}
	base, err := BaseDir()
	if err != nil {
		return preview, err
	}
	preview.BackupDirectory = filepath.Join(base, "backups", "gitid")
	return preview, nil
}

// LegacyImport backs up the source file and imports previewed profiles.
// Importing the same source digest twice is a no-op returning the original
// record; profiles with conflicting ids are skipped and counted.
func LegacyImport(ctx context.Context, store *Store, expectedDigest string, onlyIDs []string, confirmBackup, confirmImport bool) (*ImportRecord, error) {
	if !confirmBackup || !confirmImport {
		return nil, errors.New("GIT_LEGACY_CONFIRMATION_REQUIRED")
	}
	source := findLegacyStorePath()
	if source == "" {
		return nil, errors.New("GIT_LEGACY_SOURCE_CHANGED: gitid store not found")
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return nil, errors.New("GIT_LEGACY_SOURCE_CHANGED: source unreadable")
	}
	actualDigest := digestBytes(data)
	if actualDigest != expectedDigest {
		return nil, errors.New("GIT_LEGACY_SOURCE_CHANGED")
	}
	for _, record := range store.LegacyImports {
		if record.SourceKind == "gitid" && record.SourceDigest == actualDigest {
			return &record, nil
		}
	}
	var old legacyStore
	if err := json.Unmarshal(data, &old); err != nil || old.Identities == nil {
		return nil, errors.New("GIT_LEGACY_UNKNOWN_SCHEMA")
	}
	preview, err := PreviewLegacy(ctx, store)
	if err != nil {
		return nil, err
	}
	base, err := BaseDir()
	if err != nil {
		return nil, err
	}
	backupDir := filepath.Join(base, "backups", "gitid")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return nil, err
	}
	backupPath := filepath.Join(backupDir, "config-"+actualDigest[:16]+".json")
	if existing, err := os.ReadFile(backupPath); err == nil {
		if digestBytes(existing) != actualDigest {
			return nil, errors.New("GIT_LEGACY_SOURCE_CHANGED: backup path collision with different content")
		}
	} else if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return nil, err
	}
	record := &ImportRecord{SourceKind: "gitid", SourceDigest: actualDigest, BackupPath: backupPath, ImportedAt: time.Now().UTC()}
	wanted := map[string]bool{}
	for _, id := range onlyIDs {
		wanted[id] = true
	}
	for _, profile := range preview.Profiles {
		if len(wanted) > 0 && !wanted[profile.ID] {
			continue
		}
		if err := store.Upsert(profile); err != nil {
			record.Conflicts++
			continue
		}
		record.Created++
	}
	record.Skipped = len(preview.IdConflicts)
	store.LegacyImports = append(store.LegacyImports, *record)
	if err := store.Save(); err != nil {
		return nil, err
	}
	return record, nil
}

// ProfileMarks annotates a profile list with where each identity is
// currently applied (global star and local repo overrides are resolved by
// git itself).
type ProfileMarks struct {
	GlobalID  string   `json:"globalId,omitempty"`
	LocalIDs  []string `json:"localIds,omitempty"`
	BoundDirs int      `json:"boundDirs"`
}

func ListProfiles(ctx context.Context, store *Store) ([]Profile, ProfileMarks, error) {
	marks := ProfileMarks{}
	call := gitAt(ctx, "")
	name, _ := call.cfgGet("global", "user.name")
	email, _ := call.cfgGet("global", "user.email")
	marks.GlobalID = store.MatchIdentity(name, email)
	profiles := make([]Profile, 0, len(store.Profiles))
	ids := make([]string, 0, len(store.Profiles))
	for id := range store.Profiles {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		profiles = append(profiles, *store.Profiles[id])
		if len(store.Profiles[id].Gitdirs) > 0 {
			marks.BoundDirs += len(store.Profiles[id].Gitdirs)
		}
	}
	return profiles, marks, nil
}
