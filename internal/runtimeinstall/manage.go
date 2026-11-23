package runtimeinstall

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"dev-toolbox/internal/userstate"
)

// RemovePreview 描述一次卸载/注销将发生什么；归属由服务端判定，前端不推断。
type RemovePreview struct {
	ID             string `json:"id"`
	Runtime        string `json:"runtime"`
	Version        string `json:"version"`
	Source         string `json:"source"`
	LocalHome      string `json:"localHome"`
	OwnedByProduct bool   `json:"ownedByProduct"`
	WillDeleteDir  bool   `json:"willDeleteDir"`
	IsDefault      bool   `json:"isDefault"`
}

type RemoveResult struct {
	Preview    RemovePreview   `json:"preview"`
	State      userstate.State `json:"state"`
	Removed    bool            `json:"removed"`
	DirDeleted bool            `json:"dirDeleted"`
	WasDefault bool            `json:"wasDefault"`
}

// withinRoot 判断 dir 是否严格位于产品自有运行时根目录内（不含根本身）。
func withinRoot(dir, root string) bool {
	if dir == "" || root == "" {
		return false
	}
	d, err1 := filepath.Abs(dir)
	r, err2 := filepath.Abs(root)
	if err1 != nil || err2 != nil {
		return false
	}
	d = filepath.Clean(d)
	r = filepath.Clean(r)
	if runtime.GOOS == "windows" {
		d = strings.ToLower(d)
		r = strings.ToLower(r)
	}
	return strings.HasPrefix(d, r+string(filepath.Separator))
}

// RemoveInstallation 卸载/注销一个安装。仅产品自有（downloaded 且在 RuntimeRoot
// 内）的目录可删；外部登记目录只移除记录。dryRun 只预览；写入需 confirm。
func RemoveInstallation(id string, expectedRevision uint64, deleteFiles, dryRun, confirm bool) (RemoveResult, error) {
	state, err := userstate.Get()
	if err != nil {
		return RemoveResult{}, err
	}
	var inst *userstate.Installation
	for i := range state.Installations {
		if state.Installations[i].ID == id {
			inst = &state.Installations[i]
			break
		}
	}
	if inst == nil {
		return RemoveResult{}, errors.New("RUNTIME_INSTALL_NOT_FOUND")
	}
	root, rootErr := RuntimeRoot()
	owned := rootErr == nil && inst.Source == "downloaded" && withinRoot(inst.LocalHome, root)
	if deleteFiles && !owned {
		return RemoveResult{}, errors.New("RUNTIME_REMOVE_NOT_OWNED: external installations are never deleted; unregister only")
	}
	isDefault := false
	if sel, ok := state.GlobalSelections[inst.Runtime]; ok && sel.InstallationID == id {
		isDefault = true
	}
	preview := RemovePreview{
		ID: inst.ID, Runtime: inst.Runtime, Version: inst.Version, Source: inst.Source,
		LocalHome: inst.LocalHome, OwnedByProduct: owned, WillDeleteDir: owned && deleteFiles, IsDefault: isDefault,
	}
	if dryRun {
		return RemoveResult{Preview: preview}, nil
	}
	if !confirm {
		return RemoveResult{}, errors.New("CONFIRMATION_REQUIRED")
	}
	dirDeleted := false
	if preview.WillDeleteDir {
		if err := os.RemoveAll(inst.LocalHome); err != nil {
			return RemoveResult{}, fmt.Errorf("RUNTIME_REMOVE_FAILED: %v", err)
		}
		dirDeleted = true
	}
	newState, err := userstate.RemoveRecord(id, expectedRevision, true)
	if err != nil {
		return RemoveResult{}, err
	}
	return RemoveResult{Preview: preview, State: newState, Removed: true, DirDeleted: dirDeleted, WasDefault: isDefault}, nil
}
