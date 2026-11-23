// Package goenv 以 go 自身为事实源管理用户级 Go 环境配置。
// 读=`go env -json`（只读内省）；写=`go env -w/-u`（显式用户动作，
// 只写 go 的用户 env 文件 GOENV，不改系统 PATH/注册表/shell profile）。
package goenv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"dev-toolbox/internal/userstate"
)

var editable = []string{"GOPROXY", "GOPATH", "GOBIN"}

type ProxyPreset struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

var presets = []ProxyPreset{
	{ID: "default", Label: "官方 proxy.golang.org", Value: "https://proxy.golang.org,direct"},
	{ID: "cn", Label: "goproxy.cn（国内）", Value: "https://goproxy.cn,direct"},
	{ID: "aliyun", Label: "阿里云镜像", Value: "https://mirrors.aliyun.com/goproxy/,direct"},
	{ID: "direct", Label: "direct（离线/直连）", Value: "direct"},
}

type Report struct {
	GoBinary  string            `json:"goBinary"`
	GoVersion string            `json:"goVersion"`
	EnvFile   string            `json:"envFile"`
	Values    map[string]string `json:"values"`
	Editable  []string          `json:"editable"`
	Presets   []ProxyPreset     `json:"presets"`
}

type Change struct {
	Key    string `json:"key"`
	From   string `json:"from"`
	To     string `json:"to"`
	Action string `json:"action"` // set | unset
}

type SetResult struct {
	Change Change `json:"change"`
	Report Report `json:"report"`
}

func isEditable(key string) bool {
	for _, k := range editable {
		if k == key {
			return true
		}
	}
	return false
}

// locateGo 优先用户级全局默认 golang 安装的二进制，回退 PATH 上的 go。
func locateGo() (string, error) {
	if state, err := userstate.Get(); err == nil {
		if sel, ok := state.GlobalSelections["golang"]; ok && sel.InstallationID != "" {
			for _, inst := range state.Installations {
				if inst.ID == sel.InstallationID && inst.BinaryPath != "" {
					return inst.BinaryPath, nil
				}
			}
		}
	}
	if path, err := exec.LookPath("go"); err == nil {
		return path, nil
	}
	return "", errors.New("GO_NOT_AVAILABLE: no go binary found; install or register one first")
}

func runGo(ctx context.Context, bin string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.SysProcAttr = spawnAttrs()
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("GO_ENV_WRITE_FAILED: %v", err)
	}
	return string(out), nil
}

// Read 只读内省 go env。
func Read(ctx context.Context) (Report, error) {
	bin, err := locateGo()
	if err != nil {
		return Report{}, err
	}
	out, err := runGo(ctx, bin, "env", "-json")
	if err != nil {
		return Report{}, errors.New("GO_NOT_AVAILABLE: " + err.Error())
	}
	var values map[string]string
	if err := json.Unmarshal([]byte(out), &values); err != nil {
		return Report{}, fmt.Errorf("GO_NOT_AVAILABLE: parse go env: %v", err)
	}
	return Report{
		GoBinary:  bin,
		GoVersion: values["GOVERSION"],
		EnvFile:   values["GOENV"],
		Values:    values,
		Editable:  editable,
		Presets:   presets,
	}, nil
}

// Set 预览（dryRun）或显式写入（confirm）一个可编辑键。
func Set(ctx context.Context, key, value string, unset, dryRun, confirm bool) (SetResult, error) {
	key = strings.ToUpper(strings.TrimSpace(key))
	if !isEditable(key) {
		return SetResult{}, fmt.Errorf("GO_ENV_KEY_NOT_EDITABLE: %s", key)
	}
	before, err := Read(ctx)
	if err != nil {
		return SetResult{}, err
	}
	change := Change{Key: key, From: before.Values[key]}
	if unset {
		change.Action = "unset"
		change.To = ""
	} else {
		change.Action = "set"
		change.To = strings.TrimSpace(value)
	}
	if dryRun {
		return SetResult{Change: change, Report: before}, nil
	}
	if !confirm {
		return SetResult{}, errors.New("CONFIRMATION_REQUIRED")
	}
	if unset {
		_, err = runGo(ctx, before.GoBinary, "env", "-u", key)
	} else {
		_, err = runGo(ctx, before.GoBinary, "env", "-w", key+"="+change.To)
	}
	if err != nil {
		return SetResult{}, err
	}
	after, err := Read(ctx)
	if err != nil {
		return SetResult{}, err
	}
	return SetResult{Change: change, Report: after}, nil
}
