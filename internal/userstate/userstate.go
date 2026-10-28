package userstate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

var (
	ErrUnknownSchema = errors.New("LEGACY_UNKNOWN_SCHEMA")
	ErrSourceChanged = errors.New("LEGACY_SOURCE_CHANGED")
	ErrConfirmation  = errors.New("LEGACY_CONFIRMATION_REQUIRED")
	ErrNotFound      = errors.New("RUNTIME_INSTALLATION_NOT_FOUND")
	ErrUnavailable   = errors.New("RUNTIME_INSTALLATION_UNAVAILABLE")
)

var toolNames = []string{"node", "java", "maven", "gradle", "androidsdk", "golang", "python"}

type Selection struct {
	InstallationID string `json:"installationId,omitempty"`
	UnresolvedHome string `json:"unresolvedHome,omitempty"`
	Status         string `json:"status"` // active, unresolved, or empty
	Source         string `json:"source"` // user or legacy
}

type Installation struct {
	ID         string `json:"id"`
	Runtime    string `json:"runtime"`
	Version    string `json:"version"`
	Major      string `json:"major"`
	LocalHome  string `json:"localHome"`
	BinaryPath string `json:"binaryPath,omitempty"`
	Source     string `json:"source"`
	Label      string `json:"label,omitempty"`
	Vendor     string `json:"vendor,omitempty"`
	Available  bool   `json:"available"`
}

type ImportRecord struct {
	SourceKind   string    `json:"sourceKind"`
	SourceDigest string    `json:"sourceDigest"`
	SourceSchema int       `json:"sourceSchema"`
	BackupPath   string    `json:"backupPath"`
	ImportedAt   time.Time `json:"importedAt"`
	Created      int       `json:"created"`
	Linked       int       `json:"linked"`
	Unresolved   int       `json:"unresolved"`
	Conflicts    int       `json:"conflicts"`
	Unsupported  int       `json:"unsupported"`
}

type State struct {
	Schema           int                  `json:"schema"`
	Revision         uint64               `json:"revision"`
	GlobalSelections map[string]Selection `json:"globalSelections"`
	Installations    []Installation       `json:"installations"`
	LegacyImports    []ImportRecord       `json:"legacyImports"`
}

type LegacyPreview struct {
	SourceID        string               `json:"sourceId"`
	SourcePath      string               `json:"sourcePath"`
	SourceSchema    int                  `json:"sourceSchema"`
	SourceDigest    string               `json:"sourceDigest"`
	Selections      map[string]Selection `json:"selections"`
	Installations   []Installation       `json:"installations"`
	Unresolved      []string             `json:"unresolved"`
	Unsupported     []string             `json:"unsupported"`
	DuplicateCount  int                  `json:"duplicateCount"`
	BackupDirectory string               `json:"backupDirectory"`
}

type ImportOptions struct {
	ExpectedDigest string
	ConfirmBackup  bool
	ConfirmImport  bool
}

type RegistrationCounts struct {
	Created int `json:"created"`
	Linked  int `json:"linked"`
	Skipped int `json:"skipped"`
}

// RegisterInstallations adds explicitly confirmed, statically inspected local runtimes.
// It never changes the user's selected global defaults.
func RegisterInstallations(items []Installation, expectedRevision uint64, confirmed bool) (State, RegistrationCounts, error) {
	if !confirmed {
		return State{}, RegistrationCounts{}, ErrConfirmation
	}
	path, err := StatePath()
	if err != nil {
		return State{}, RegistrationCounts{}, err
	}
	unlock, err := lock(path)
	if err != nil {
		return State{}, RegistrationCounts{}, err
	}
	defer unlock()
	state, path, err := readState()
	if err != nil {
		return State{}, RegistrationCounts{}, err
	}
	if state.Revision != expectedRevision {
		return State{}, RegistrationCounts{}, errors.New("USER_RUNTIME_REVISION_CONFLICT")
	}
	counts := RegistrationCounts{}
	for _, item := range items {
		if !knownTool(item.Runtime) || strings.TrimSpace(item.LocalHome) == "" {
			counts.Skipped++
			continue
		}
		home, err := filepath.Abs(filepath.Clean(item.LocalHome))
		if err != nil || !isDirectory(home) {
			counts.Skipped++
			continue
		}
		item.LocalHome = home
		item.ID = installationID(item.Runtime, home)
		item.Available = true
		if existing := findInstallation(state.Installations, item.ID); existing != nil {
			counts.Linked++
			continue
		}
		state.Installations = append(state.Installations, item)
		counts.Created++
	}
	if counts.Created > 0 {
		state.Revision++
		if err := writeState(path, state); err != nil {
			return State{}, RegistrationCounts{}, err
		}
	}
	return state, counts, nil
}

type legacyState struct {
	Schema   int               `json:"schema"`
	Current  map[string]string `json:"current"`
	Runtimes []legacyRuntime   `json:"runtimes"`
}

type legacyRuntime struct {
	Tool    string `json:"tool"`
	Version string `json:"version"`
	Major   string `json:"major"`
	Home    string `json:"home"`
	Binary  string `json:"binary"`
	Source  string `json:"source"`
	Label   string `json:"label,omitempty"`
	Vendor  string `json:"vendor,omitempty"`
}

type diskState struct {
	Schema           int                  `json:"schema"`
	Revision         uint64               `json:"revision"`
	GlobalSelections map[string]Selection `json:"globalSelections"`
	Installations    []Installation       `json:"installations"`
	LegacyImports    []ImportRecord       `json:"legacyImports"`
}

func StatePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "DevToolbox", "user-runtime-state.json"), nil
}

func LegacyStatePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "devswitch", "state.json"), nil
}

func NewState() State {
	selections := make(map[string]Selection, len(toolNames))
	for _, name := range toolNames {
		selections[name] = Selection{Status: "empty"}
	}
	return State{Schema: 1, GlobalSelections: selections, Installations: []Installation{}, LegacyImports: []ImportRecord{}}
}

func Get() (State, error) {
	s, _, err := readState()
	if err != nil {
		return State{}, err
	}
	for i := range s.Installations {
		s.Installations[i].Available = isDirectory(s.Installations[i].LocalHome)
	}
	for runtimeName, selection := range s.GlobalSelections {
		if selection.InstallationID == "" {
			continue
		}
		installation := findInstallationByID(s.Installations, selection.InstallationID)
		if installation != nil && !installation.Available {
			selection.Status = "unresolved"
			selection.UnresolvedHome = installation.LocalHome
			s.GlobalSelections[runtimeName] = selection
		}
	}
	return s, err
}

func InspectLegacy() (map[string]any, error) {
	path, err := LegacyStatePath()
	if err != nil {
		return nil, err
	}
	data, err := readLegacyBytes(path)
	if err != nil {
		return map[string]any{"found": false, "sourcePath": path, "readable": false}, nil
	}
	var old legacyState
	if err := json.Unmarshal(data, &old); err != nil {
		return map[string]any{"found": true, "sourcePath": path, "readable": true, "valid": false}, nil
	}
	return map[string]any{"found": true, "sourcePath": path, "readable": true, "valid": old.Schema == 1, "schema": old.Schema, "sourceDigest": digest(data)}, nil
}

func PreviewLegacy(ctx context.Context) (LegacyPreview, error) {
	path, err := LegacyStatePath()
	if err != nil {
		return LegacyPreview{}, err
	}
	data, err := readLegacyBytes(path)
	if err != nil {
		return LegacyPreview{}, fmt.Errorf("read DevSwitch state: %w", err)
	}
	old, err := decodeLegacy(data)
	if err != nil {
		return LegacyPreview{}, err
	}
	if err := ctx.Err(); err != nil {
		return LegacyPreview{}, err
	}
	preview := mapLegacy(ctx, path, old, digest(data))
	if err := ctx.Err(); err != nil {
		return LegacyPreview{}, err
	}
	statePath, err := StatePath()
	if err != nil {
		return LegacyPreview{}, err
	}
	preview.BackupDirectory = filepath.Join(filepath.Dir(statePath), "backups", "devswitch")
	return preview, nil
}

func ImportLegacy(ctx context.Context, options ImportOptions) (ImportRecord, error) {
	if !options.ConfirmBackup || !options.ConfirmImport {
		return ImportRecord{}, ErrConfirmation
	}
	path, err := LegacyStatePath()
	if err != nil {
		return ImportRecord{}, err
	}
	data, err := readLegacyBytes(path)
	if err != nil {
		return ImportRecord{}, fmt.Errorf("read DevSwitch state: %w", err)
	}
	actualDigest := digest(data)
	if actualDigest != options.ExpectedDigest {
		return ImportRecord{}, ErrSourceChanged
	}
	old, err := decodeLegacy(data)
	if err != nil {
		return ImportRecord{}, err
	}
	preview := mapLegacy(ctx, path, old, actualDigest)
	if err := ctx.Err(); err != nil {
		return ImportRecord{}, err
	}

	statePath, err := StatePath()
	if err != nil {
		return ImportRecord{}, err
	}
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return ImportRecord{}, err
	}
	unlock, err := lock(statePath)
	if err != nil {
		return ImportRecord{}, err
	}
	defer unlock()
	state, _, err := readState()
	if err != nil {
		return ImportRecord{}, err
	}
	for _, record := range state.LegacyImports {
		if record.SourceKind == "devswitch" && record.SourceDigest == actualDigest {
			return record, nil
		}
	}
	latest, err := readLegacyBytes(path)
	if err != nil {
		return ImportRecord{}, err
	}
	if digest(latest) != actualDigest {
		return ImportRecord{}, ErrSourceChanged
	}

	backupDir := filepath.Join(filepath.Dir(statePath), "backups", "devswitch")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return ImportRecord{}, err
	}
	backupPath := filepath.Join(backupDir, "state-"+actualDigest[:16]+".json")
	if err := writeBackup(backupPath, data); err != nil {
		return ImportRecord{}, err
	}

	counts := ImportRecord{SourceKind: "devswitch", SourceDigest: actualDigest, SourceSchema: old.Schema, BackupPath: backupPath, ImportedAt: time.Now().UTC()}
	counts.Unsupported = len(preview.Unsupported)
	for _, installation := range preview.Installations {
		if existing := findInstallation(state.Installations, installation.ID); existing != nil {
			if existing.Label == "" {
				existing.Label = installation.Label
			}
			if existing.Vendor == "" {
				existing.Vendor = installation.Vendor
			}
			counts.Linked++
		} else {
			state.Installations = append(state.Installations, installation)
			counts.Created++
		}
	}
	for _, tool := range toolNames {
		legacy := preview.Selections[tool]
		if legacy.Status == "empty" {
			continue
		}
		current := state.GlobalSelections[tool]
		if current.Status != "" && current.Status != "empty" {
			counts.Conflicts++
			continue
		}
		legacy.Source = "legacy"
		state.GlobalSelections[tool] = legacy
		if legacy.Status == "unresolved" {
			counts.Unresolved++
		}
	}
	state.Revision++
	state.LegacyImports = append(state.LegacyImports, counts)
	if err := writeState(statePath, state); err != nil {
		return ImportRecord{}, err
	}
	return counts, nil
}

func SetGlobalDefault(runtimeName, installationID string, expectedRevision uint64, confirmed bool) (State, error) {
	if !confirmed {
		return State{}, ErrConfirmation
	}
	path, err := StatePath()
	if err != nil {
		return State{}, err
	}
	unlock, err := lock(path)
	if err != nil {
		return State{}, err
	}
	defer unlock()
	state, path, err := readState()
	if err != nil {
		return State{}, err
	}
	if state.Revision != expectedRevision {
		return State{}, errors.New("USER_RUNTIME_REVISION_CONFLICT")
	}
	installation := findInstallationByID(state.Installations, installationID)
	if installation == nil {
		return State{}, ErrNotFound
	}
	if installation.Runtime != runtimeName {
		return State{}, errors.New("RUNTIME_TOOL_MISMATCH")
	}
	if !isDirectory(installation.LocalHome) {
		return State{}, ErrUnavailable
	}
	state.GlobalSelections[runtimeName] = Selection{InstallationID: installationID, Status: "active", Source: "user"}
	state.Revision++
	if err := writeState(path, state); err != nil {
		return State{}, err
	}
	return state, nil
}

func decodeLegacy(data []byte) (legacyState, error) {
	var old legacyState
	if err := json.Unmarshal(data, &old); err != nil {
		return old, fmt.Errorf("LEGACY_STATE_INVALID: invalid state file")
	}
	if old.Schema != 1 {
		return old, ErrUnknownSchema
	}
	return old, nil
}

func readLegacyBytes(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, fmt.Errorf("read DevSwitch state: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > 10<<20 {
		return nil, errors.New("LEGACY_SOURCE_INVALID_PATH")
	}
	return os.ReadFile(path)
}

func mapLegacy(ctx context.Context, sourcePath string, old legacyState, sourceDigest string) LegacyPreview {
	preview := LegacyPreview{
		SourceID: stableID(sourcePath), SourcePath: sourcePath, SourceSchema: old.Schema, SourceDigest: sourceDigest,
		Selections: make(map[string]Selection, len(toolNames)), Installations: []Installation{}, Unresolved: []string{}, Unsupported: []string{},
		BackupDirectory: "",
	}
	seen := map[string]bool{}
	for _, name := range toolNames {
		preview.Selections[name] = Selection{Status: "empty"}
	}
	for _, item := range old.Runtimes {
		if err := ctx.Err(); err != nil {
			break
		}
		if !knownTool(item.Tool) || strings.TrimSpace(item.Home) == "" {
			preview.Unsupported = append(preview.Unsupported, item.Tool)
			continue
		}
		id := installationID(item.Tool, item.Home)
		installation := Installation{
			ID: id, Runtime: item.Tool, Version: item.Version, Major: item.Major,
			LocalHome: item.Home, BinaryPath: item.Binary, Source: item.Source,
			Label: item.Label, Vendor: item.Vendor, Available: isDirectory(item.Home),
		}
		if seen[id] {
			preview.DuplicateCount++
			for i := range preview.Installations {
				if preview.Installations[i].ID == id {
					if preview.Installations[i].Label == "" {
						preview.Installations[i].Label = item.Label
					}
					if preview.Installations[i].Vendor == "" {
						preview.Installations[i].Vendor = item.Vendor
					}
					if preview.Installations[i].Source == "" {
						preview.Installations[i].Source = item.Source
					}
				}
			}
			continue
		}
		seen[id] = true
		preview.Installations = append(preview.Installations, installation)
	}
	for _, tool := range toolNames {
		home := strings.TrimSpace(old.Current[tool])
		if home == "" {
			continue
		}
		id := installationID(tool, home)
		var matched *Installation
		for i := range preview.Installations {
			if preview.Installations[i].ID == id {
				matched = &preview.Installations[i]
				break
			}
		}
		if matched != nil && matched.Available {
			preview.Selections[tool] = Selection{InstallationID: id, Status: "active", Source: "legacy"}
		} else {
			preview.Selections[tool] = Selection{UnresolvedHome: home, Status: "unresolved", Source: "legacy"}
			preview.Unresolved = append(preview.Unresolved, tool)
		}
	}
	for tool := range old.Current {
		if !knownTool(tool) {
			preview.Unsupported = append(preview.Unsupported, "current:"+tool)
		}
	}
	sort.Slice(preview.Installations, func(i, j int) bool {
		if preview.Installations[i].Runtime != preview.Installations[j].Runtime {
			return preview.Installations[i].Runtime < preview.Installations[j].Runtime
		}
		return preview.Installations[i].Version < preview.Installations[j].Version
	})
	return preview
}

func readState() (State, string, error) {
	path, err := StatePath()
	if err != nil {
		return State{}, "", err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return NewState(), path, nil
	}
	if err != nil {
		return State{}, "", err
	}
	var stored diskState
	if err := json.Unmarshal(data, &stored); err != nil || stored.Schema != 1 {
		return State{}, "", errors.New("USER_RUNTIME_STATE_CORRUPT")
	}
	if stored.GlobalSelections == nil {
		stored.GlobalSelections = NewState().GlobalSelections
	} else {
		for runtimeName, selection := range NewState().GlobalSelections {
			if _, exists := stored.GlobalSelections[runtimeName]; !exists {
				stored.GlobalSelections[runtimeName] = selection
			}
		}
	}
	if stored.Installations == nil {
		stored.Installations = []Installation{}
	}
	if stored.LegacyImports == nil {
		stored.LegacyImports = []ImportRecord{}
	}
	return State{Schema: stored.Schema, Revision: stored.Revision, GlobalSelections: stored.GlobalSelections, Installations: stored.Installations, LegacyImports: stored.LegacyImports}, path, nil
}

func writeState(path string, state State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".user-runtime-*.tmp")
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

func writeBackup(path string, data []byte) error {
	if current, err := os.ReadFile(path); err == nil {
		if digest(current) == digest(data) {
			return nil
		}
		return errors.New("LEGACY_BACKUP_COLLISION")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func lock(statePath string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(statePath), 0o700); err != nil {
		return nil, err
	}
	lockPath := statePath + ".lock"
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
			return nil, errors.New("USER_RUNTIME_STATE_BUSY")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func findInstallation(items []Installation, id string) *Installation {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

func findInstallationByID(items []Installation, id string) *Installation {
	return findInstallation(items, id)
}
func knownTool(name string) bool {
	for _, item := range toolNames {
		if item == name {
			return true
		}
	}
	return false
}
func isDirectory(path string) bool { info, err := os.Stat(path); return err == nil && info.IsDir() }
func digest(data []byte) string    { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }

func stableID(value string) string {
	value = filepath.Clean(value)
	if runtime.GOOS == "windows" {
		value = strings.ToLower(value)
	}
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:16])
}

func installationID(tool, home string) string {
	cleaned := filepath.Clean(home)
	if absolute, err := filepath.Abs(cleaned); err == nil {
		cleaned = absolute
	}
	if canonical, err := filepath.EvalSymlinks(cleaned); err == nil {
		cleaned = canonical
	}
	if runtime.GOOS == "windows" {
		cleaned = strings.ToLower(cleaned)
	}
	return stableID(tool + "\x00" + cleaned)
}
