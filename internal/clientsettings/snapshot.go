package clientsettings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"dev-toolbox/internal/project"
)

// ExportSnapshot 生成可导出的会话快照。
// 只导出 shared 绑定项目的可移植投影（模块相对路径 + 构建系统 + 版本要求），
// 全部剔除本地绝对路径、仓库身份、远程 URL 与扫描状态。
func ExportSnapshot(label string) (Snapshot, error) {
	current, err := Get()
	if err != nil {
		return Snapshot{}, err
	}
	items, err := project.List()
	if err != nil {
		return Snapshot{}, err
	}
	requirements := make([]project.ProjectSummary, 0, len(items))
	skipped := 0
	for _, item := range items {
		summary, sumErr := project.SummarizeShared(item)
		if sumErr != nil {
			skipped++
			continue
		}
		requirements = append(requirements, summary)
	}
	snap := Snapshot{
		Schema:       snapshotSchema,
		Label:        label,
		ExportedAt:   time.Now().UTC(),
		Theme:        current.Theme,
		CustomAccent: current.CustomAccent,
		Requirements: requirements,
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
		return Snapshot{}, ErrBadTheme
	}
	if snap.Theme == ThemeCustom {
		if _, err := NormalizeAccent(snap.CustomAccent); err != nil {
			return Snapshot{}, err
		}
	} else {
		snap.CustomAccent = ""
	}
	for _, summary := range snap.Requirements {
		if summary.ID == "" {
			return Snapshot{}, ErrBadSnapshot
		}
	}
	if snap.Digest == "" || snap.Digest != snapshotDigest(snap) {
		return Snapshot{}, ErrBadDigest
	}
	return snap, nil
}

// ImportSnapshot 导入快照：写入设置并把原始副本备份到配置目录。
// 共享要求写入 shared-requirements.json（只读消费），不回写 projects.json。
// 幂等：同一摘要重复导入只返回原记录，不重复备份。
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
	applied := 1 // 主题+强调色按 1 项计入
	payloads := make([]project.ProjectPayload, 0, len(snap.Requirements))
	for _, summary := range snap.Requirements {
		modules := make([]project.SummaryModule, 0, len(summary.Modules))
		modules = append(modules, summary.Modules...)
		payloads = append(payloads, project.ProjectPayload{
			ID:          summary.ID,
			DisplayName: summary.DisplayName,
			Modules:     modules,
			ScanVersion: summary.ScanVersion,
		})
	}
	state, err := project.SetSharedRequirements(payloads)
	if err != nil {
		return Settings{}, SnapshotRecord{}, err
	}
	applied += len(state.Payloads)
	current.Theme = snap.Theme
	current.CustomAccent = snap.CustomAccent
	current.Revision++
	record := SnapshotRecord{
		SourceDigest: snap.Digest,
		Label:        snap.Label,
		BackupPath:   backupPath,
		ImportedAt:   time.Now().UTC(),
		Applied:      applied,
		Skipped:      len(snap.Requirements) - len(state.Payloads),
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

// snapshotDigest 覆盖主题、强调色与共享要求，不含导出时间戳。
func snapshotDigest(snap Snapshot) string {
	payload := struct {
		Schema       int                      `json:"schema"`
		Theme        ThemeMode                `json:"theme"`
		CustomAccent string                   `json:"customAccent,omitempty"`
		Requirements []project.ProjectSummary `json:"requirements,omitempty"`
	}{Schema: snap.Schema, Theme: snap.Theme, CustomAccent: snap.CustomAccent, Requirements: snap.Requirements}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	return byteDigest(data)
}
