// Package session 实现会话启动器：按用户级全局运行时选择计算注入环境，
// 并以该环境在指定 cwd 启动终端进程。注入只作用于启动的子进程树——
// 绝不写用户/系统环境变量、注册表或 shell profile（ADR-007）。
package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dev-toolbox/internal/gitidentity"
	"dev-toolbox/internal/userstate"
)

type ToolState struct {
	Tool    string `json:"tool"`
	Status  string `json:"status"` // injected | skipped
	Version string `json:"version,omitempty"`
	BinDir  string `json:"binDir,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type TerminalInfo struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	ExePath string `json:"exePath"`
}

type Preview struct {
	Cwd          string             `json:"cwd"`
	Tools        []ToolState        `json:"tools"`
	JavaHome     string             `json:"javaHome,omitempty"`
	PathPrepends []string           `json:"pathPrepends"`
	GitIdentity  GitIdentityPreview `json:"gitIdentity"`
	Terminals    []TerminalInfo     `json:"terminals"`
}

type GitIdentityPreview struct {
	Resolved  string `json:"resolved"` // profile id | manual | none
	ProfileID string `json:"profileId,omitempty"`
}

type OpenResult struct {
	TerminalID string `json:"terminalId"`
	Pid        int    `json:"pid"`
}

// binDirFor 计算某工具安装的 PATH 注入目录（各发行版布局差异在此收敛）。
func binDirFor(tool, home string) string {
	switch tool {
	case "node", "python", "rust":
		return home
	case "java", "maven", "gradle", "golang":
		return filepath.Join(home, "bin")
	case "androidsdk":
		return filepath.Join(home, "platform-tools")
	}
	return home
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// ResolveContext 只读求值：八工具逐项结论 + PATH 前置项 + JAVA_HOME。
// 未选择或路径失效的工具如实标 skipped，不静默回退也不拦截启动。
func ResolveContext(ctx context.Context, cwd string) (*Preview, error) {
	if cwd == "" {
		cwd = "."
	}
	absolute, err := filepath.Abs(cwd)
	if err != nil || !dirExists(absolute) {
		return nil, fmt.Errorf("SESSION_CWD_INVALID: %s", cwd)
	}
	state, err := userstate.Get()
	if err != nil {
		return nil, err
	}
	preview := &Preview{Cwd: absolute, Tools: []ToolState{}, PathPrepends: []string{}}
	seen := map[string]bool{}
	byID := map[string]userstate.Installation{}
	for _, item := range state.Installations {
		byID[item.ID] = item
	}
	for _, tool := range userstate.ToolNames() {
		selection := state.GlobalSelections[tool]
		if selection.InstallationID == "" {
			reason := "未选择"
			if selection.Status == "unresolved" {
				reason = "待重新关联：" + selection.UnresolvedHome
			}
			preview.Tools = append(preview.Tools, ToolState{Tool: tool, Status: "skipped", Reason: reason})
			continue
		}
		installation, ok := byID[selection.InstallationID]
		if !ok {
			preview.Tools = append(preview.Tools, ToolState{Tool: tool, Status: "skipped", Reason: "所选安装不在清单中"})
			continue
		}
		binDir := binDirFor(tool, installation.LocalHome)
		if !installation.Available || !dirExists(binDir) {
			preview.Tools = append(preview.Tools, ToolState{Tool: tool, Status: "skipped", Reason: "路径不可用：" + installation.LocalHome})
			continue
		}
		preview.Tools = append(preview.Tools, ToolState{Tool: tool, Status: "injected", Version: installation.Version, BinDir: binDir})
		key := strings.ToLower(binDir)
		if !seen[key] {
			seen[key] = true
			preview.PathPrepends = append(preview.PathPrepends, binDir)
		}
		if tool == "java" {
			preview.JavaHome = installation.LocalHome
		}
	}
	if store, err := gitidentity.Load(); err == nil {
		if effective, err := gitidentity.ResolveEffective(ctx, store, absolute, false); err == nil {
			preview.GitIdentity = GitIdentityPreview{Resolved: effective.Resolved, ProfileID: effective.ProfileID}
		}
	}
	preview.Terminals = DetectTerminals()
	return preview, nil
}

// BuildEnv 组装启动环境：父环境 + PATH 前置 + JAVA_HOME（仅本进程树）。
func BuildEnv(parent []string, prepends []string, javaHome string) []string {
	sep := string(os.PathListSeparator)
	newPath := strings.Join(prepends, sep)
	var parentPath string
	filtered := make([]string, 0, len(parent)+2)
	for _, entry := range parent {
		name, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(name, "PATH") {
			parentPath = value
			continue
		}
		if found && javaHome != "" && strings.EqualFold(name, "JAVA_HOME") {
			continue
		}
		filtered = append(filtered, entry)
	}
	merged := newPath
	if parentPath != "" {
		merged = newPath + sep + parentPath
	}
	filtered = append(filtered, "PATH="+merged)
	if javaHome != "" {
		filtered = append(filtered, "JAVA_HOME="+javaHome)
	}
	return filtered
}
