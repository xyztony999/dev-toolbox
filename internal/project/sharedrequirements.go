package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// sharedRequirementStore 持有从外部快照导入的共享要求（只读消费）。
// 它只保存可移植投影，不含本地路径、仓库身份或凭据；也不会反写 projects.json。
var sharedRequirementStore = struct {
	sync.Mutex
	state SharedRequirementState
}{state: SharedRequirementState{Schema: 1, Revision: 1, Payloads: []ProjectPayload{}}}

// SharedRequirementStatePath 返回外部共享要求载荷文件位置。
func SharedRequirementStatePath() (string, error) {
	override := strings.TrimSpace(os.Getenv("DEVTOOLBOX_SHARED_REQUIREMENTS"))
	if override != "" {
		return override, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "DevToolbox", "shared-requirements.json"), nil
}

// GetSharedRequirements 返回当前持有的外部共享要求，首次读取时从磁盘恢复。
func GetSharedRequirements() (SharedRequirementState, error) {
	sharedRequirementStore.Mutex.Lock()
	defer sharedRequirementStore.Mutex.Unlock()
	if sharedRequirementStore.state.Payloads == nil {
		path, err := SharedRequirementStatePath()
		if err != nil {
			return SharedRequirementState{}, err
		}
		data, err := os.ReadFile(path)
		if err == nil {
			var stored SharedRequirementState
			if json.Unmarshal(data, &stored) == nil && stored.Schema == 1 {
				sharedRequirementStore.state = stored
			}
		} else if !os.IsNotExist(err) {
			return SharedRequirementState{}, err
		}
		if sharedRequirementStore.state.Payloads == nil {
			sharedRequirementStore.state.Payloads = []ProjectPayload{}
		}
	}
	return copySharedRequirements(sharedRequirementStore.state), nil
}

// SetSharedRequirements 全量替换外部共享要求；按项目 ID 归一，幂等写入。
func SetSharedRequirements(payloads []ProjectPayload) (SharedRequirementState, error) {
	sharedRequirementStore.Mutex.Lock()
	defer sharedRequirementStore.Mutex.Unlock()
	byID := make(map[string]ProjectPayload, len(payloads))
	for _, payload := range payloads {
		if strings.TrimSpace(payload.ID) == "" {
			continue
		}
		payload.ID = strings.TrimSpace(payload.ID)
		payload.CapturedAt = time.Now().UTC()
		byID[payload.ID] = payload
	}
	next := make([]ProjectPayload, 0, len(byID))
	for _, payload := range byID {
		next = append(next, payload)
	}
	sort.SliceStable(next, func(i, j int) bool { return next[i].ID < next[j].ID })
	state := SharedRequirementState{Schema: 1, Revision: sharedRequirementStore.state.Revision + 1, Payloads: next}
	path, err := SharedRequirementStatePath()
	if err != nil {
		return SharedRequirementState{}, err
	}
	if err := writeSharedRequirements(path, state); err != nil {
		return SharedRequirementState{}, err
	}
	sharedRequirementStore.state = copySharedRequirements(state)
	return copySharedRequirements(state), nil
}

func writeSharedRequirements(path string, state SharedRequirementState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".shared-requirements-*.tmp")
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

func copySharedRequirements(state SharedRequirementState) SharedRequirementState {
	out := SharedRequirementState{Schema: state.Schema, Revision: state.Revision, Payloads: make([]ProjectPayload, 0, len(state.Payloads))}
	for _, payload := range state.Payloads {
		payload.Modules = append([]SummaryModule(nil), payload.Modules...)
		out.Payloads = append(out.Payloads, payload)
	}
	return out
}
