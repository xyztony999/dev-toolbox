//go:build gui && !windows

package desktop

import (
	"os/exec"
	"strings"
)

// pickFolderDialog 优先使用 zenity 的目录选择对话框；未安装时报不可用。
func pickFolderDialog() (string, error) {
	cmd := exec.Command("zenity", "--file-selection", "--directory")
	out, err := cmd.Output()
	if err != nil {
		return "", errCancelled
	}
	path := strings.TrimSpace(string(out))
	if path == "" {
		return "", errCancelled
	}
	return path, nil
}
