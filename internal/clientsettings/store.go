package clientsettings

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	storeVersion     = 1
	snapshotSchema   = 1
	defaultScanDepth = 6
)

// SettingsPath 返回客户端偏好文件位置：%APPDATA%/DevToolbox/settings.json（Linux 为
// $XDG_CONFIG_HOME/DevToolbox/settings.json）。DEVTOOLBOX_SETTINGS 仅供测试隔离。
func SettingsPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv("DEVTOOLBOX_SETTINGS")); override != "" {
		return override, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "DevToolbox", "settings.json"), nil
}

// New 返回一份默认设置：跟随系统主题、空源项目清单、空快照记录。
func New() Settings {
	return Settings{
		Schema:    storeVersion,
		Revision:  1,
		Theme:     ThemeSystem,
		Snapshots: []SnapshotRecord{},
	}
}

// Get 读取设置；文件不存在时返回默认值。损坏文件返回 ErrCorrupt。
func Get() (Settings, error) {
	path, err := SettingsPath()
	if err != nil {
		return Settings{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return New(), nil
	}
	if err != nil {
		return Settings{}, err
	}
	var stored Settings
	if err := json.Unmarshal(data, &stored); err != nil || stored.Schema != storeVersion {
		return Settings{}, ErrCorrupt
	}
	return normalize(stored), nil
}

func normalize(s Settings) Settings {
	if !s.Theme.Valid() {
		s.Theme = ThemeSystem
	}
	if s.Theme != ThemeCustom {
		s.CustomAccent = ""
	}
	s.CustomAccent = strings.TrimSpace(s.CustomAccent)
	if s.Snapshots == nil {
		s.Snapshots = []SnapshotRecord{}
	}
	return s
}

func lock(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	lockPath := path + ".lock"
	deadline := time.Now().Add(10 * time.Second)
	for {
		f, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_, _ = f.WriteString("lock")
			_ = f.Close()
			return func() { _ = os.Remove(lockPath) }, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
		if time.Now().After(deadline) {
			return nil, ErrBusy
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func writeSettings(path string, s Settings) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".settings-*.tmp")
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

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func byteDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// readSettings 读取并规范化设置；文件不存在时返回 New()。
func readSettings(path string) (Settings, string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return New(), path, nil
	}
	if err != nil {
		return Settings{}, path, err
	}
	var stored Settings
	if err := json.Unmarshal(data, &stored); err != nil || stored.Schema != storeVersion {
		return Settings{}, path, ErrCorrupt
	}
	return normalize(stored), path, nil
}
