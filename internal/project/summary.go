package project

import (
	"time"
)

// ProjectSummary 是共享要求载荷中的最小项目投影。
// 只保留可移植字段（显示名、模块相对路径、构建系统、版本要求），
// 全部剔除本地绝对路径、仓库身份、远程 URL 与扫描状态。
type ProjectSummary struct {
	ID          string          `json:"id"`
	DisplayName string          `json:"displayName"`
	Binding     ProjectBinding  `json:"binding"`
	Modules     []SummaryModule `json:"modules"`
	ScanVersion string          `json:"scanVersion,omitempty"`
	CapturedAt  time.Time       `json:"capturedAt"`
}

// SummaryModule 是共享载荷里的模块投影：相对路径与运行时要求。
type SummaryModule struct {
	RelativePath string               `json:"relativePath"`
	BuildSystems []string             `json:"buildSystems"`
	Requirements []SummaryRequirement `json:"requirements,omitempty"`
}

// SummaryRequirement 保留运行时要求的可移植信息（名称、约束、来源的相对路径）。
type SummaryRequirement struct {
	Runtime    string `json:"runtime"`
	Constraint string `json:"constraint,omitempty"`
	SourcePath string `json:"sourcePath,omitempty"`
	Status     string `json:"status"`
}

// SummarizeShared 把项目投影为共享载荷；本地绑定项目返回不可用错误。
func SummarizeShared(p Project) (ProjectSummary, error) {
	if p.Binding != BindingShared {
		return ProjectSummary{}, ErrNotShared
	}
	modules := make([]SummaryModule, 0, len(p.Modules))
	for _, module := range p.Modules {
		requirements := make([]SummaryRequirement, 0, len(module.Requirements))
		for _, requirement := range module.Requirements {
			constraint := requirement.NormalizedConstraint
			if constraint == "" {
				constraint = requirement.RawConstraint
			}
			requirements = append(requirements, SummaryRequirement{
				Runtime:    requirement.Runtime,
				Constraint: constraint,
				SourcePath: requirement.SourcePath,
				Status:     string(requirement.Status),
			})
		}
		modules = append(modules, SummaryModule{
			RelativePath: module.RelativePath,
			BuildSystems: module.BuildSystems,
			Requirements: requirements,
		})
	}
	return ProjectSummary{
		ID:          p.ID,
		DisplayName: p.DisplayName,
		Binding:     BindingShared,
		Modules:     modules,
		ScanVersion: p.ScanToolVersion,
		CapturedAt:  time.Now().UTC(),
	}, nil
}
