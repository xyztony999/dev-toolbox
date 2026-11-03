package runtimecatalog

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// 外部工具链只读检测（toolchain-import 模块）：
// rustup 工具链与 uv 自管解释器。全程仅读目录与文件，绝不执行
// rustup/uv/rustc，也绝不写 RUSTUP_HOME 或 uv 数据目录。

var (
	rustSemverToolchain      = regexp.MustCompile(`^(\d+\.\d+\.\d+)-`)
	rustChannelToolchain     = regexp.MustCompile(`^(stable|beta|nightly)(?:-\d{4}-\d{2}-\d{2})?-`)
	pythonBuildStandaloneDir = regexp.MustCompile(`^cpython-(\d+\.\d+(?:\.\d+)?)`)
)

const maxExternalToolchains = 50

func rustupRoot() string {
	if v := strings.TrimSpace(os.Getenv("RUSTUP_HOME")); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".rustup")
}

func uvPythonRoot() string {
	if v := strings.TrimSpace(os.Getenv("UV_PYTHON_INSTALL_DIR")); v != "" {
		return v
	}
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("APPDATA"); appdata != "" {
			// 真机验证（2026-11-03，Windows）：%APPDATA%\uv\python
			return filepath.Join(appdata, "uv", "python")
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "share", "uv", "python")
}

// rustupToolchainHomes 枚举 toolchains/ 下的工具链目录。
func rustupToolchainHomes() []string {
	toolchains := filepath.Join(rustupRoot(), "toolchains")
	entries, err := os.ReadDir(toolchains)
	if err != nil {
		return nil
	}
	homes := []string{}
	for index, entry := range entries {
		if index >= maxExternalToolchains {
			break
		}
		if entry.IsDir() {
			homes = append(homes, filepath.Join(toolchains, entry.Name()))
		}
	}
	return homes
}

// uvPythonHomes 枚举 uv python 安装目录下的 cpython-* 目录。
func uvPythonHomes() []string {
	root := uvPythonRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	homes := []string{}
	for index, entry := range entries {
		if index >= maxExternalToolchains {
			break
		}
		if entry.IsDir() && pythonBuildStandaloneDir.MatchString(entry.Name()) {
			homes = append(homes, filepath.Join(root, entry.Name()))
		}
	}
	return homes
}

// markRustupDefault 只读解析 settings.toml 顶层 default_toolchain，
// 给对应 rust 候选补默认标记。受限解析：仅取目标行，坏文件安静跳过。
func markRustupDefault(result *Result) {
	data, err := os.ReadFile(filepath.Join(rustupRoot(), "settings.toml"))
	if err != nil {
		return
	}
	name := ""
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			// 顶层键只在第一个表头之前有效
			break
		}
		if strings.HasPrefix(trimmed, "default_toolchain") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				name = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			}
		}
	}
	if name == "" {
		return
	}
	for i := range result.Candidates {
		if result.Candidates[i].Tool != "rust" {
			continue
		}
		if filepath.Base(result.Candidates[i].Home) == name {
			result.Candidates[i].Evidence = appendUnique(result.Candidates[i].Evidence, "rustup default")
			result.Candidates[i].Digest = candidateDigest(result.Candidates[i])
		}
	}
}
