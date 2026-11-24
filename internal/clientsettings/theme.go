package clientsettings

import (
	"fmt"
	"regexp"
	"strings"
)

var accentRe = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// NormalizeAccent 校验并规范化自定义强调色；返回空字符串表示清除自定义色。
func NormalizeAccent(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !accentRe.MatchString(value) {
		return "", fmt.Errorf("%s: %q", ErrBadAccent.Error(), value)
	}
	return strings.ToLower(value), nil
}

// SetTheme 更新外观偏好，返回新设置。mode=custom 时 accent 必须是合法的十六进制色值。
func SetTheme(mode ThemeMode, accent string, expectedRevision uint64) (Settings, error) {
	if !mode.Valid() {
		return Settings{}, fmt.Errorf("%s: %s", ErrBadTheme.Error(), mode)
	}
	accent, err := NormalizeAccent(accent)
	if err != nil {
		return Settings{}, err
	}
	if mode == ThemeCustom && accent == "" {
		return Settings{}, fmt.Errorf("%s: custom theme needs an accent color", ErrBadTheme.Error())
	}
	if mode != ThemeCustom {
		accent = ""
	}
	path, err := SettingsPath()
	if err != nil {
		return Settings{}, err
	}
	unlock, err := lock(path)
	if err != nil {
		return Settings{}, err
	}
	defer unlock()
	current, _, err := readSettings(path)
	if err != nil {
		return Settings{}, err
	}
	if current.Revision != expectedRevision && expectedRevision != 0 {
		return Settings{}, ErrConflict
	}
	current.Theme = mode
	current.CustomAccent = accent
	current.Revision++
	if err := writeSettings(path, current); err != nil {
		return Settings{}, err
	}
	return current, nil
}
