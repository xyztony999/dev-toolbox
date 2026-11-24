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

// ProjectBinding 表示一个源项目的登记方式：local 只在本机有意义，shared 可随清单分发。
type ProjectBinding string

const (
	BindingLocal  ProjectBinding = "local"
	BindingShared ProjectBinding = "shared"
)

func (b ProjectBinding) Valid() bool {
	switch b {
	case BindingLocal, BindingShared:
		return true
	}
	return false
}

// SourceProject 是用户登记的源项目路径。它不保存任何凭据或个人身份选择。
type SourceProject struct {
	Path    string         `json:"path"`
	Label   string         `json:"label,omitempty"`
	Binding ProjectBinding `json:"binding"`
	Note    string         `json:"note,omitempty"`
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
	Schema         int              `json:"schema"`
	Revision       uint64           `json:"revision"`
	Theme          ThemeMode        `json:"theme"`
	CustomAccent   string           `json:"customAccent,omitempty"`
	SourceProjects []SourceProject  `json:"sourceProjects"`
	Snapshots      []SnapshotRecord `json:"snapshots"`
}

// Snapshot 是可导出的会话状态：只包含客户端自身偏好，不含 token / 身份选择。
type Snapshot struct {
	Schema         int             `json:"schema"`
	Label          string          `json:"label,omitempty"`
	ExportedAt     time.Time       `json:"exportedAt"`
	Theme          ThemeMode       `json:"theme"`
	CustomAccent   string          `json:"customAccent,omitempty"`
	SourceProjects []SourceProject `json:"sourceProjects"`
	Digest         string          `json:"digest"`
}

var (
	ErrCorrupt     = errors.New("CLIENT_SETTINGS_CORRUPT")
	ErrConflict    = errors.New("CLIENT_SETTINGS_REVISION_CONFLICT")
	ErrConfirm     = errors.New("CONFIRMATION_REQUIRED")
	ErrBadTheme    = errors.New("CLIENT_SETTINGS_THEME_INVALID")
	ErrBadAccent   = errors.New("CLIENT_SETTINGS_ACCENT_INVALID")
	ErrBadBinding  = errors.New("CLIENT_SETTINGS_BINDING_INVALID")
	ErrBadPath     = errors.New("CLIENT_SETTINGS_PATH_INVALID")
	ErrBadDigest   = errors.New("CLIENT_SETTINGS_DIGEST_MISMATCH")
	ErrBadSnapshot = errors.New("CLIENT_SETTINGS_SNAPSHOT_INVALID")
)
