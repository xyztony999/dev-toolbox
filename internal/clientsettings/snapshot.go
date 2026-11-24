package clientsettings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ExportSnapshot 生成可导出的会话快照：只含客户端自身偏好，不含 token 或身份选择。
func ExportSnapshot(label string, includeSharedProjects bool) (Snapshot, error) {
	current, err := Get()
	if err != nil {
		return Snapshot{}, err
	}
	projects := make([]SourceProject, 0, len(current.SourceProjects))
	for _, item := range current.SourceProjects {
		if !includeSharedProjects && item.Binding != BindingShared {
			continue
		}
		projects = append(projects, item)
	}
	snap := Snapshot{
		Schema:         snapshotSchema,
		Label:          label,
		ExportedAt:     time.Now().UTC(),
		Theme:          current.Theme,
		CustomAccent:   current.CustomAccent,
		SourceProjects: projects,
	}
	snap.Digest = snapshotDigest(snap)
	return snap, nil
}

// SnapshotBytes 序列化快照供下载。
func SnapshotBytes(snap Snapshot) ([]byte, error) {
	return json.MarshalIndent(snap, "", "  ")
}

// ParseSnapshot 解析并校验一份快照文件；摘要不匹配时拒绝。
func ParseSnapshot(data []byte) (Snapshot, error) {
	var snap Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return Snapshot{}, ErrBadSnapshot
	}
	if snap.Schema != snapshotSchema {
		return Snapshot{}, fmt.Errorf("%s: schema %d", ErrBadSnapshot.Error(), snap.Schema)
	}
	if !snap.Theme.Valid() {
		return Snapshot{}, ErrBadSnapshot
	}
	if snap.Theme == ThemeCustom {
		if _, err := NormalizeAccent(snap.CustomAccent); err != nil {
			return Snapshot{}, err
		}
	} else {
		snap.CustomAccent = ""
	}
	for i := range snap.SourceProjects {
		if !snap.SourceProjects[i].Binding.Valid() {
			return Snapshot{}, ErrBadBinding
		}
		canonical, err := canonicalPath(snap.SourceProjects[i].Path)
		if err != nil {
			return Snapshot{}, err
		}
		snap.SourceProjects[i].Path = canonical
	}
	if snap.Digest == "" || snap.Digest != snapshotDigest(snap) {
		return Snapshot{}, ErrBadDigest
	}
	return snap, nil
}

// ImportSnapshot 导入快照：写入设置并把原始副本备份到配置目录。
// 幂等：同一摘要重复导入只报告 skipped，不重复写备份。
func ImportSnapshot(data []byte, expectedRevision uint64) (Settings, SnapshotRecord, error) {
	snap, err := ParseSnapshot(data)
	if err != nil {
		return Settings{}, SnapshotRecord{}, err
	}
	settingsPath, err := SettingsPath()
	if err != nil {
		return Settings{}, SnapshotRecord{}, err
	}
	unlock, err := lock(settingsPath)
	if err != nil {
		return Settings{}, SnapshotRecord{}, err
	}
	defer unlock()
	current, _, err := readSettings(settingsPath)
	if err != nil {
		return Settings{}, SnapshotRecord{}, err
	}
	if current.Revision != expectedRevision && expectedRevision != 0 {
		return Settings{}, SnapshotRecord{}, ErrConflict
	}
	for _, rec := range current.Snapshots {
		if rec.SourceDigest == snap.Digest {
			return current, rec, nil
		}
	}
	backupPath, err := backUpSnapshot(settingsPath, snap)
	if err != nil {
		return Settings{}, SnapshotRecord{}, err
	}
	applied := 0
	current.Theme = snap.Theme
	current.CustomAccent = snap.CustomAccent
	applied++
	for _, item := range snap.SourceProjects {
		found := false
		for _, existing := range current.SourceProjects {
			if samePath(existing.Path, item.Path) {
				found = true
				break
			}
		}
		if found {
			continue
		}
		current.SourceProjects = append(current.SourceProjects, item)
		applied++
	}
	sortProjects(current.SourceProjects)
	current.Revision++
	record := SnapshotRecord{
		SourceDigest: snap.Digest,
		Label:        snap.Label,
		BackupPath:   backupPath,
		ImportedAt:   time.Now().UTC(),
		Applied:      applied,
		Skipped:      len(snap.SourceProjects) - (applied - 1),
	}
	current.Snapshots = append(current.Snapshots, record)
	if err := writeSettings(settingsPath, current); err != nil {
		return Settings{}, SnapshotRecord{}, err
	}
	return current, record, nil
}

func backUpSnapshot(settingsPath string, snap Snapshot) (string, error) {
	dir := filepath.Join(filepath.Dir(settingsPath), "backups", "settings")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	target := filepath.Join(dir, "snapshot-"+snap.Digest+".json")
	if _, err := os.Stat(target); err == nil {
		return target, nil
	}
	data, err := SnapshotBytes(snap)
	if err != nil {
		return "", err
	}
	f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return target, nil
		}
		return "", err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return "", err
	}
	return target, f.Close()
}

// snapshotDigest 覆盖主题、强调色与共享项目路径，不含导出时间戳。
func snapshotDigest(snap Snapshot) string {
	payload := struct {
		Schema         int             `json:"schema"`
		Theme          ThemeMode       `json:"theme"`
		CustomAccent   string          `json:"customAccent,omitempty"`
		SourceProjects []SourceProject `json:"sourceProjects"`
	}{Schema: snap.Schema, Theme: snap.Theme, CustomAccent: snap.CustomAccent, SourceProjects: snap.SourceProjects}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return byteDigest(data)
}
