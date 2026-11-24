package clientsettings

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AddSourceProject 登记一个源项目路径。binding 必须为 local 或 shared：
// shared 条目不得指向用户主目录内（AGENTS.md 规则 5：共享清单不携带私有路径）。
func AddSourceProject(path, label, note string, binding ProjectBinding, expectedRevision uint64) (Settings, error) {
	if !binding.Valid() {
		return Settings{}, fmt.Errorf("%s: %s", ErrBadBinding.Error(), binding)
	}
	canonical, err := canonicalPath(path)
	if err != nil {
		return Settings{}, err
	}
	if binding == BindingShared {
		if home, homeErr := os.UserHomeDir(); homeErr == nil && withinHome(home, canonical) {
			return Settings{}, fmt.Errorf("%s: shared entries must not point inside the user home directory", ErrBadPath.Error())
		}
	}
	if strings.ContainsAny(note, "\r\n\x00") {
		return Settings{}, ErrBadPath
	}
	entry := SourceProject{Path: canonical, Label: strings.TrimSpace(label), Binding: binding, Note: strings.TrimSpace(note)}
	settingsPath, err := SettingsPath()
	if err != nil {
		return Settings{}, err
	}
	unlock, err := lock(settingsPath)
	if err != nil {
		return Settings{}, err
	}
	defer unlock()
	current, _, err := readSettings(settingsPath)
	if err != nil {
		return Settings{}, err
	}
	if current.Revision != expectedRevision && expectedRevision != 0 {
		return Settings{}, ErrConflict
	}
	for _, existing := range current.SourceProjects {
		if samePath(existing.Path, entry.Path) {
			return Settings{}, fmt.Errorf("%s: %s already registered", ErrBadPath.Error(), entry.Path)
		}
	}
	current.SourceProjects = append(current.SourceProjects, entry)
	sortProjects(current.SourceProjects)
	current.Revision++
	if err := writeSettings(settingsPath, current); err != nil {
		return Settings{}, err
	}
	return current, nil
}

// RemoveSourceProject 移除一个源项目登记，保留幂等：不存在时原样返回。
func RemoveSourceProject(path string, expectedRevision uint64) (Settings, bool, error) {
	canonical, err := canonicalPath(path)
	if err != nil {
		return Settings{}, false, err
	}
	settingsPath, err := SettingsPath()
	if err != nil {
		return Settings{}, false, err
	}
	unlock, err := lock(settingsPath)
	if err != nil {
		return Settings{}, false, err
	}
	defer unlock()
	current, _, err := readSettings(settingsPath)
	if err != nil {
		return Settings{}, false, err
	}
	if current.Revision != expectedRevision && expectedRevision != 0 {
		return Settings{}, false, ErrConflict
	}
	kept := make([]SourceProject, 0, len(current.SourceProjects))
	removed := false
	for _, item := range current.SourceProjects {
		if samePath(item.Path, canonical) {
			removed = true
			continue
		}
		kept = append(kept, item)
	}
	if !removed {
		return current, false, nil
	}
	current.SourceProjects = kept
	current.Revision++
	if err := writeSettings(settingsPath, current); err != nil {
		return Settings{}, false, err
	}
	return current, true, nil
}

// withinHome reports whether path lives inside home.
func withinHome(home, path string) bool {
	rel, err := filepath.Rel(home, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}
