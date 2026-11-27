// Package clientsettings 维护 DevToolbox 客户端自身的偏好设置：外观主题、源项目清单、
// 运行时工具状态与会话快照导入导出。它只写 DevToolbox 拥有的 settings.json（用户配置目录下），
// 不修改 PATH、注册表、shell profile 或用户 Git 配置。
//
// 产品纪律映射：
//   - 主题（外观）：仅客户端侧持久化，不触碰系统主题。
//   - 源项目：本机绑定与共享清单分离；条目不含个人身份选择或凭据（AGENTS.md 规则 5）。
//   - 运行时工具：只读探测，隔离环境询问，不切换全局默认、不改真实 PATH（规则 9）。
//   - 会话快照：导出/导入带来源、���览与幂等记录，��入前需确认（规则 8）。
package clientsettings

import (
	"errors"
	"time"

	"dev-toolbox/internal/project"
)

// ThemeMode 是客户端外观偏好；custom 需要配合 CustomAccent 使用。
type ThemeMode string

const (
	ThemeSystem ThemeMode = "system"
	ThemeLight  ThemeMode = "light"
	ThemeDark   ThemeMode = "dark"
	ThemeCustom ThemeMode = "custom"
)

func (t ThemeMode) Valid() bool {
	switch t {
	case ThemeSystem, ThemeLight, ThemeDark, ThemeCustom:
		return true
	}
	return false
}

// SnapshotRecord 记录一次会话快照导入，幂等：同一摘要重复导入只计入 skipped。
type SnapshotRecord struct {
	SourceDigest string    `json:"sourceDigest"`
	Label        string    `json:"label,omitempty"`
	BackupPath   string    `json:"backupPath,omitempty"`
	ImportedAt   time.Time `json:"importedAt"`
	Applied      int       `json:"applied"`
	Skipped      int       `json:"skipped"`
}

// Settings 是客户端偏好文件的结构。
type Settings struct {
	Schema       int              `json:"schema"`
	Revision     uint64           `json:"revision"`
	Theme        ThemeMode        `json:"theme"`
	CustomAccent string           `json:"customAccent,omitempty"`
	Snapshots    []SnapshotRecord `json:"snapshots"`
}

// Snapshot 是可导出的会话状态：只含客户端偏好与共享要求，
// 不含本地路径、仓库身份或凭据。
type Snapshot struct {
	Schema       int                      `json:"schema"`
	Label        string                   `json:"label,omitempty"`
	ExportedAt   time.Time                `json:"exportedAt"`
	Theme        ThemeMode                `json:"theme"`
	CustomAccent string                   `json:"customAccent,omitempty"`
	Requirements []project.ProjectSummary `json:"requirements,omitempty"`
	Digest       string                   `json:"digest"`
}

var (
	ErrCorrupt     = errors.New("CLIENT_SETTINGS_CORRUPT")
	ErrConflict    = errors.New("CLIENT_SETTINGS_REVISION_CONFLICT")
	ErrConfirm     = errors.New("CONFIRMATION_REQUIRED")
	ErrBadTheme    = errors.New("CLIENT_SETTINGS_THEME_INVALID")
	ErrBadAccent   = errors.New("CLIENT_SETTINGS_ACCENT_INVALID")
	ErrBadDigest   = errors.New("CLIENT_SETTINGS_DIGEST_MISMATCH")
	ErrBadSnapshot = errors.New("CLIENT_SETTINGS_SNAPSHOT_INVALID")
)
