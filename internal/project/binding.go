package project

import (
	"fmt"
	"time"
)

// ProjectPayload 是快照导入方用来消费共享要求的视图。
// 与 Project 不同，它不含 RootPath / Repositories / Issues / 扫描时间戳。
type ProjectPayload struct {
	ID          string          `json:"id"`
	DisplayName string          `json:"displayName"`
	Modules     []SummaryModule `json:"modules"`
	ScanVersion string          `json:"scanVersion,omitempty"`
	CapturedAt  time.Time       `json:"capturedAt"`
}

// SharedRequirementState 是 shared-requirements.json 的结构：本机消费的外部共享要求。
type SharedRequirementState struct {
	Schema   int              `json:"schema"`
	Revision uint64           `json:"revision"`
	Payloads []ProjectPayload `json:"payloads"`
}

// SetBinding 更新一个已登记项目的绑定范围，用于设置页与 CLI。
// 幂等：设置成当前值时不递增修订号、不写文件。
func SetBinding(id string, binding ProjectBinding, expectedRevision uint64) (Project, bool, error) {
	if !binding.Valid() {
		return Project{}, false, fmt.Errorf("PROJECT_BINDING_INVALID: %s", binding)
	}
	path, err := StorePath()
	if err != nil {
		return Project{}, false, err
	}
	unlock, err := lockStore(path)
	if err != nil {
		return Project{}, false, err
	}
	defer unlock()
	s, path, err := readStore()
	if err != nil {
		return Project{}, false, err
	}
	for index := range s.Projects {
		p := &s.Projects[index]
		if p.ID != id {
			continue
		}
		if p.Revision != expectedRevision {
			return Project{}, false, ErrRevision
		}
		current := p.Binding
		if current == "" {
			current = BindingLocal
		}
		if current == binding {
			return *p, false, nil
		}
		p.Binding = binding
		p.Revision++
		if err := writeStore(path, s); err != nil {
			return Project{}, false, err
		}
		return *p, true, nil
	}
	return Project{}, false, ErrNotFound
}
