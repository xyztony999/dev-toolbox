// Package toolconfig 以各运行时自身为事实源，统一暴露"环境配置"的读与
// 显式写。可写项均为工具原生用户级配置（go env / npm config / pip config /
// maven settings 托管块）；不能安全代写的（JAVA_HOME/ANDROID_HOME/
// GRADLE_USER_HOME/cargo 镜像）只读并说明杠杆。
package toolconfig

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"dev-toolbox/internal/goenv"
	"dev-toolbox/internal/userstate"
)

type Preset struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type ConfigItem struct {
	Key      string   `json:"key"`
	Label    string   `json:"label"`
	Value    string   `json:"value"`
	Editable bool     `json:"editable"`
	Source   string   `json:"source"`
	Presets  []Preset `json:"presets,omitempty"`
	Note     string   `json:"note,omitempty"`
}

type ToolConfigReport struct {
	Tool   string       `json:"tool"`
	Binary string       `json:"binary,omitempty"`
	Items  []ConfigItem `json:"items"`
	Note   string       `json:"note,omitempty"`
}

type ConfigChange struct {
	Key    string `json:"key"`
	From   string `json:"from"`
	To     string `json:"to"`
	Action string `json:"action"`
}

type ToolConfigSetResult struct {
	Change ConfigChange     `json:"change"`
	Report ToolConfigReport `json:"report"`
}

func runTool(ctx context.Context, bin string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.SysProcAttr = spawnAttrs()
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("TOOL_CONFIG_WRITE_FAILED: %s", strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("TOOL_CONFIG_WRITE_FAILED: %v", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// defaultBinary 取某工具全局默认安装的二进制路径（无则空）。
func defaultBinary(tool string) string {
	state, err := userstate.Get()
	if err != nil {
		return ""
	}
	sel, ok := state.GlobalSelections[tool]
	if !ok || sel.InstallationID == "" {
		return ""
	}
	for _, inst := range state.Installations {
		if inst.ID == sel.InstallationID {
			return inst.BinaryPath
		}
	}
	return ""
}

func defaultHome(tool string) string {
	state, err := userstate.Get()
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

func lookPath(names ...string) string {
	for _, n := range names {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
	}
	return ""
}

func npmBinary() string {
	if bin := defaultBinary("node"); bin != "" {
		dir := filepath.Dir(bin)
		if p := filepath.Join(dir, "npm.cmd"); fileExists(p) {
			return p
		}
		if p := filepath.Join(dir, "npm"); fileExists(p) {
			return p
		}
	}
	return lookPath("npm.cmd", "npm")
}

func pythonBinary() string {
	if bin := defaultBinary("python"); bin != "" {
		return bin
	}
	return lookPath("python3", "python")
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

var registryPresets = []Preset{
	{ID: "npmjs", Label: "官方 registry.npmjs.org", Value: "https://registry.npmjs.org/"},
	{ID: "npmmirror", Label: "npmmirror（国内）", Value: "https://registry.npmmirror.com/"},
}

var pipPresets = []Preset{
	{ID: "pypi", Label: "官方 pypi.org", Value: "https://pypi.org/simple"},
	{ID: "tsinghua", Label: "清华 TUNA", Value: "https://pypi.tuna.tsinghua.edu.cn/simple"},
	{ID: "aliyun", Label: "阿里云镜像", Value: "https://mirrors.aliyun.com/pypi/simple/"},
}

// Read 返回某工具的环境配置报告；工具不可用时 items 空 + note，不报错。
func Read(ctx context.Context, tool string) (ToolConfigReport, error) {
	switch tool {
	case "golang":
		rep, err := goenv.Read(ctx)
		if err != nil {
			return ToolConfigReport{Tool: tool, Note: "go 不可用：" + err.Error()}, nil
		}
		items := make([]ConfigItem, 0, len(rep.Editable)+4)
		for _, k := range rep.Editable {
			items = append(items, ConfigItem{Key: k, Label: k, Value: rep.Values[k], Editable: true, Source: "go env", Presets: proxyPresetsIf(k, rep)})
		}
		for _, k := range []string{"GOROOT", "GOMODCACHE", "GOOS", "GOARCH"} {
			items = append(items, ConfigItem{Key: k, Label: k, Value: rep.Values[k], Source: "go env（只读）"})
		}
		return ToolConfigReport{Tool: tool, Binary: rep.GoBinary, Items: items}, nil
	case "node":
		return readNode(ctx)
	case "python":
		return readPython(ctx)
	case "maven":
		return readMaven()
	case "java":
		home := defaultHome("java")
		note := "JAVA_HOME 由全局默认 java 安装推导；会话注入以此为准。切换默认版本即改变它，产品不写系统环境变量。"
		if home == "" {
			home = os.Getenv("JAVA_HOME")
			note = "未设全局默认 java；当前显示进程环境 JAVA_HOME。在「默认版本」选择一个安装后由它推导。"
		}
		return ToolConfigReport{Tool: tool, Items: []ConfigItem{{Key: "JAVA_HOME", Label: "JAVA_HOME", Value: home, Source: "只读（默认安装推导）", Note: note}}}, nil
	case "androidsdk":
		home := os.Getenv("ANDROID_HOME")
		src := "进程环境 ANDROID_HOME"
		if home == "" {
			home = defaultHome("androidsdk")
			src = "只读（默认安装推导）"
		}
		return ToolConfigReport{Tool: tool, Items: []ConfigItem{{Key: "ANDROID_HOME", Label: "ANDROID_HOME", Value: home, Source: src, Note: "产品不写系统环境变量；会话注入以默认安装/该变量为准。"}}}, nil
	case "gradle":
		home := os.Getenv("GRADLE_USER_HOME")
		src := "进程环境 GRADLE_USER_HOME"
		if home == "" {
			if h, err := os.UserHomeDir(); err == nil {
				home = filepath.Join(h, ".gradle")
			}
			src = "只读（默认 ~/.gradle）"
		}
		return ToolConfigReport{Tool: tool, Items: []ConfigItem{{Key: "GRADLE_USER_HOME", Label: "GRADLE_USER_HOME", Value: home, Source: src, Note: "Gradle 无安全的全局镜像开关；wrapper 分发镜像需在 gradle-wrapper.properties 中按项目调整。"}}}, nil
	case "rust":
		return readRust()
	default:
		return ToolConfigReport{Tool: tool, Note: "该工具暂无可管理的环境配置"}, nil
	}
}

func proxyPresetsIf(key string, rep goenv.Report) []Preset {
	if key != "GOPROXY" {
		return nil
	}
	out := make([]Preset, 0, len(rep.Presets))
	for _, p := range rep.Presets {
		out = append(out, Preset(p))
	}
	return out
}

func readNode(ctx context.Context) (ToolConfigReport, error) {
	bin := npmBinary()
	if bin == "" {
		return ToolConfigReport{Tool: "node", Note: "未找到 npm；请先安装或登记 node。"}, nil
	}
	value, err := runTool(ctx, bin, "config", "get", "registry")
	if err != nil {
		return ToolConfigReport{Tool: "node", Binary: bin, Note: "读取 npm registry 失败：" + err.Error()}, nil
	}
	if value == "undefined" {
		value = ""
	}
	return ToolConfigReport{Tool: "node", Binary: bin, Items: []ConfigItem{{
		Key: "registry", Label: "npm registry", Value: value, Editable: true,
		Source: "npm config（~/.npmrc）", Presets: registryPresets,
	}}}, nil
}

func readPython(ctx context.Context) (ToolConfigReport, error) {
	bin := pythonBinary()
	if bin == "" {
		return ToolConfigReport{Tool: "python", Note: "未找到 python；请先安装或登记。"}, nil
	}
	value, err := runTool(ctx, bin, "-m", "pip", "config", "get", "global.index-url")
	if err != nil {
		value = "" // 未设置时 pip 以非零退出，视为空
	}
	return ToolConfigReport{Tool: "python", Binary: bin, Items: []ConfigItem{{
		Key: "global.index-url", Label: "pip index-url", Value: value, Editable: true,
		Source: "pip config（用户级）", Presets: pipPresets,
	}}}, nil
}

func readRust() (ToolConfigReport, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return ToolConfigReport{Tool: "rust", Note: "无法定位用户目录"}, nil
	}
	cfg := filepath.Join(home, ".cargo", "config.toml")
	value := ""
	if data, err := os.ReadFile(cfg); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			trim := strings.TrimSpace(line)
			if strings.HasPrefix(trim, "replace-with") {
				if idx := strings.Index(trim, "="); idx >= 0 {
					value = strings.Trim(strings.TrimSpace(trim[idx+1:]), `"'`)
				}
			}
		}
	}
	return ToolConfigReport{Tool: "rust", Items: []ConfigItem{{
		Key: "cargo.mirror", Label: "cargo 镜像（replace-with）", Value: value, Source: "~/.cargo/config.toml（只读）",
		Note: "cargo 镜像需编辑 config.toml 的 [source] 段，结构多样，产品只读不代写。",
	}}}, nil
}

// Set 预览或显式写入一个可写键。
func Set(ctx context.Context, tool, key, value string, unset, dryRun, confirm bool) (ToolConfigSetResult, error) {
	before, err := Read(ctx, tool)
	if err != nil {
		return ToolConfigSetResult{}, err
	}
	var item *ConfigItem
	for i := range before.Items {
		if before.Items[i].Key == key {
			item = &before.Items[i]
			break
		}
	}
	if item == nil || !item.Editable {
		return ToolConfigSetResult{}, fmt.Errorf("TOOL_CONFIG_KEY_NOT_EDITABLE: %s/%s", tool, key)
	}
	change := ConfigChange{Key: key, From: item.Value}
	if unset {
		change.Action, change.To = "unset", ""
	} else {
		change.Action, change.To = "set", strings.TrimSpace(value)
	}
	if dryRun {
		return ToolConfigSetResult{Change: change, Report: before}, nil
	}
	if !confirm {
		return ToolConfigSetResult{}, errors.New("CONFIRMATION_REQUIRED")
	}
	switch tool {
	case "golang":
		if _, err := goenv.Set(ctx, key, change.To, unset, false, true); err != nil {
			return ToolConfigSetResult{}, err
		}
	case "node":
		if err := writeNode(ctx, change.To, unset); err != nil {
			return ToolConfigSetResult{}, err
		}
	case "python":
		if err := writePython(ctx, change.To, unset); err != nil {
			return ToolConfigSetResult{}, err
		}
	case "maven":
		if err := writeMavenMirror(change.To, unset); err != nil {
			return ToolConfigSetResult{}, err
		}
	default:
		return ToolConfigSetResult{}, fmt.Errorf("TOOL_CONFIG_KEY_NOT_EDITABLE: %s/%s", tool, key)
	}
	after, err := Read(ctx, tool)
	if err != nil {
		return ToolConfigSetResult{}, err
	}
	return ToolConfigSetResult{Change: change, Report: after}, nil
}

func writeNode(ctx context.Context, value string, unset bool) error {
	bin := npmBinary()
	if bin == "" {
		return errors.New("TOOL_NOT_AVAILABLE: npm")
	}
	if unset {
		_, err := runTool(ctx, bin, "config", "delete", "registry")
		return err
	}
	_, err := runTool(ctx, bin, "config", "set", "registry", value)
	return err
}

func writePython(ctx context.Context, value string, unset bool) error {
	bin := pythonBinary()
	if bin == "" {
		return errors.New("TOOL_NOT_AVAILABLE: python")
	}
	if unset {
		_, err := runTool(ctx, bin, "-m", "pip", "config", "unset", "global.index-url")
		return err
	}
	_, err := runTool(ctx, bin, "-m", "pip", "config", "set", "global.index-url", value)
	return err
}
