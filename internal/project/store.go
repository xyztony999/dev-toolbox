package project

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
	ErrNotFound      = errors.New("PROJECT_NOT_FOUND")
	ErrAlreadyExists = errors.New("PROJECT_ALREADY_EXISTS")
	ErrStaleScan     = errors.New("PROJECT_SCAN_STALE")
	ErrRevision      = errors.New("PROJECT_REVISION_CONFLICT")
	ErrStoreCorrupt  = errors.New("PROJECT_STORE_CORRUPT")
)

func StorePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(base, "DevToolbox", "projects.json"), nil
}

func ScanPreview(ctx context.Context, root string) (Candidate, error) {
	return scan(ctx, root)
}

func List() ([]Project, error) {
	s, _, err := readStore()
	if err != nil {
		return nil, err
	}
	sort.Slice(s.Projects, func(i, j int) bool {
		return strings.ToLower(s.Projects[i].DisplayName) < strings.ToLower(s.Projects[j].DisplayName)
	})
	return s.Projects, nil
}

func Get(id string) (Project, error) {
	s, _, err := readStore()
	if err != nil {
		return Project{}, err
	}
	for _, p := range s.Projects {
		if p.ID == id {
			return p, nil
		}
	}
	return Project{}, ErrNotFound
}

func Register(ctx context.Context, root, name, candidateID, expectedDigest string) (Project, bool, error) {
	candidate, err := scan(ctx, root)
	if err != nil {
		return Project{}, false, err
	}
	if candidate.ID != candidateID || candidate.EvidenceDigest != expectedDigest {
		return Project{}, false, ErrStaleScan
	}
	path, err := StorePath()
	if err != nil {
		return Project{}, false, err
	}
	unlock, err := lockStore(path)
	if err != nil {
		return Project{}, false, err
	}
	defer unlock()
	s, path, err := readStore()
	if err != nil {
		return Project{}, false, err
	}
	for _, p := range s.Projects {
		if samePath(p.RootPath, candidate.RootPath) {
			return p, false, nil
		}
	}
	if strings.TrimSpace(name) == "" {
		name = filepath.Base(candidate.RootPath)
	}
	now := time.Now().UTC()
	p := Project{
		ID:           stableID(candidate.RootPath),
		DisplayName:  strings.TrimSpace(name),
		RootPath:     candidate.RootPath,
		Repositories: candidate.Repositories,
		Modules:      candidate.Modules,
		Issues:       candidate.Issues,
		Status:       candidate.Status,
		CreatedAt:    now,
		LastScanAt:   candidate.ScannedAt,
		Revision:     1,
	}
	s.Projects = append(s.Projects, p)
	if err := writeStore(path, s); err != nil {
		return Project{}, false, err
	}
	return p, true, nil
}

func Rescan(ctx context.Context, id string, expectedRevision uint64) (Project, error) {
	current, err := Get(id)
	if err != nil {
		return Project{}, err
	}
	if current.Revision != expectedRevision {
		return Project{}, ErrRevision
	}
	candidate, err := scan(ctx, current.RootPath)
	if err != nil {
		return Project{}, err
	}
	path, err := StorePath()
	if err != nil {
		return Project{}, err
	}
	unlock, err := lockStore(path)
	if err != nil {
		return Project{}, err
	}
	defer unlock()
	s, path, err := readStore()
	if err != nil {
		return Project{}, err
	}
	for i := range s.Projects {
		p := &s.Projects[i]
		if p.ID == id {
			if p.Revision != expectedRevision {
				return Project{}, ErrRevision
			}
			if candidate.Status == ScanComplete {
				p.Repositories = candidate.Repositories
				p.Modules = candidate.Modules
				p.Issues = candidate.Issues
			} else {
				// Keep the last complete discovery snapshot; expose the latest attempt status and issues.
				p.Issues = candidate.Issues
			}
			p.Status = candidate.Status
			p.LastScanAt = candidate.ScannedAt
			p.Revision++
			if err := writeStore(path, s); err != nil {
				return Project{}, err
			}
			return *p, nil
		}
	}
	return Project{}, ErrNotFound
}

func Unregister(id string, expectedRevision uint64) (bool, error) {
	path, err := StorePath()
	if err != nil {
		return false, err
	}
	unlock, err := lockStore(path)
	if err != nil {
		return false, err
	}
	defer unlock()
	s, path, err := readStore()
	if err != nil {
		return false, err
	}
	for i, p := range s.Projects {
		if p.ID != id {
			continue
		}
		if p.Revision != expectedRevision {
			return false, ErrRevision
		}
		s.Projects = append(s.Projects[:i], s.Projects[i+1:]...)
		if err := writeStore(path, s); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func lockStore(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create project store directory: %w", err)
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
			return nil, fmt.Errorf("lock project store: %w", err)
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("PROJECT_STORE_BUSY: timed out waiting for project store")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func readStore() (store, string, error) {
	path, err := StorePath()
	if err != nil {
		return store{}, "", err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store{Version: 1, Projects: []Project{}}, path, nil
	}
	if err != nil {
		return store{}, "", fmt.Errorf("read project store: %w", err)
	}
	var s store
	if err := json.Unmarshal(data, &s); err != nil || s.Version != 1 {
		return store{}, "", ErrStoreCorrupt
	}
	if s.Projects == nil {
		s.Projects = []Project{}
	}
	return s, path, nil
}

func writeStore(path string, s store) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create project store directory: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".projects-*.tmp")
	if err != nil {
		return fmt.Errorf("create project store temporary file: %w", err)
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
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace project store: %w", err)
	}
	return nil
}

func stableID(s string) string {
	cleaned := filepath.Clean(s)
	if runtime.GOOS == "windows" {
		cleaned = strings.ToLower(cleaned)
	}
	sum := sha256.Sum256([]byte(cleaned))
	return hex.EncodeToString(sum[:16])
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
