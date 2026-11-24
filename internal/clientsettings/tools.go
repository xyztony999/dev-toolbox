package clientsettings

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// knownTools 是设置页展示顺序所依据的工具清单；rust 来自运行时目录清单。
var knownTools = []string{"node", "java", "maven", "gradle", "androidsdk", "golang", "python", "rust"}

// ToolStatus 是一个运行时工具在设置页的只读状态。
// 只做静态探测：读已登记安装与 PATH 上的可执行文件，不执行工具本身、不改环境。
type ToolStatus struct {
	Tool         string `json:"tool"`
	Available    bool   `json:"available"`
	Version      string `json:"version,omitempty"`
	Home         string `json:"home,omitempty"`
	Source       string `json:"source"`
	Configurable bool   `json:"configurable"`
	ConfigNote   string `json:"configNote,omitempty"`
}

// ToolStatusOptions 控制探测是否访问 PATH（测试可关闭）。
type ToolStatusOptions struct {
	AllowPathLookup bool
}

// ToolStatuses 返回每个工具的只读状态。
func ToolStatuses(ctx context.Context, opts ToolStatusOptions) []ToolStatus {
	out := make([]ToolStatus, 0, len(knownTools))
	for _, tool := range knownTools {
		out = append(out, inspectTool(ctx, tool, opts))
	}
	return out
}

// inspectTool 探测单个工具：优先用户级登记安装，其次 PATH（可关闭）。
func inspectTool(ctx context.Context, tool string, opts ToolStatusOptions) ToolStatus {
	status := ToolStatus{Tool: tool, Source: "none"}
	if tool == "" {
		return status
	}
	home, binary, version := registeredInstall(tool)
	if binary == "" && opts.AllowPathLookup {
		binary = pathBinary(tool)
		home = homeFromBinary(ctx, tool, binary)
		version = versionFromBinary(ctx, binary)
	}
	if binary == "" {
		status.Source = "none"
		return status
	}
	status.Available = true
	status.Home = home
	status.Version = version
	if version == "" {
		status.Version = "unknown"
	}
	if home == "" {
		status.Home = pathDirFromBinary(binary)
	}
	if registeredHome(tool) == status.Home && status.Home != "" {
		status.Source = "user"
	} else {
		status.Source = "path"
	}
	if toolconfigEditable(tool) {
		status.Configurable = true
		status.ConfigNote = "可由本产品写入工具原生用户配置"
	} else {
		status.ConfigNote = "只读；请使用工具自身的配置文件"
	}
	return status
}

func toolconfigEditable(tool string) bool {
	switch tool {
	case "golang", "node", "python", "maven":
		return true
	}
	return false
}

func registeredHome(tool string) string {
	state, err := userstateGet()
	if err != nil {
		return ""
	}
	sel, ok := state.GlobalSelections[tool]
	if !ok || sel.InstallationID == "" {
		return ""
	}
	for _, inst := range state.Installations {
		if inst.ID == sel.InstallationID {
			return inst.LocalHome
		}
	}
	return ""
}

func registeredInstall(tool string) (home, binary, version string) {
	state, err := userstateGet()
	if err != nil {
		return "", "", ""
	}
	sel, ok := state.GlobalSelections[tool]
	if !ok || sel.InstallationID == "" {
		return "", "", ""
	}
	for _, inst := range state.Installations {
		if inst.ID != sel.InstallationID {
			continue
		}
		home = inst.LocalHome
		binary = inst.BinaryPath
		version = inst.Version
		if !dirExists(home) || binary == "" {
			return "", "", ""
		}
		if !fileExists(binary) {
			binary = ""
		}
		return home, binary, version
	}
	return "", "", ""
}

func pathBinary(tool string) string {
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = map[string][]string{
			"node": {"node.exe"}, "java": {"java.exe"}, "maven": {"mvn.cmd"}, "gradle": {"gradle.bat"},
			"androidsdk": {"adb.exe"}, "golang": {"go.exe"}, "python": {"python.exe"}, "rust": {"rustc.exe"},
		}[tool]
	} else {
		candidates = map[string][]string{
			"node": {"node"}, "java": {"java"}, "maven": {"mvn"}, "gradle": {"gradle"},
			"androidsdk": {"adb"}, "golang": {"go"}, "python": {"python3", "python"}, "rust": {"rustc"},
		}[tool]
	}
	for _, name := range candidates {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

// homeFromBinary 只做路径推导，绝不在探测阶段执行工具可执行文件。
func homeFromBinary(ctx context.Context, tool, binary string) string {
	if binary == "" {
		return ""
	}
	dir := filepath.Dir(binary)
	parent := filepath.Dir(dir)
	switch tool {
	case "java":
		return parent
	case "maven", "gradle":
		return parent
	case "androidsdk":
		// bin 的上一级 platform-tools 的上一级
		return filepath.Dir(parent)
	case "golang", "node", "python", "rust":
		return parent
	}
	return parent
}

// versionFromBinary 只做静态推导（读取 release/version 文件），不执行工具。
func versionFromBinary(ctx context.Context, binary string) string {
	if binary == "" {
		return ""
	}
	dir := filepath.Dir(binary)
	for _, name := range []string{"VERSION", "version", "release"} {
		if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return ""
}

func pathDirFromBinary(binary string) string {
	if binary == "" {
		return ""
	}
	return filepath.Dir(binary)
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
