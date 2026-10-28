package runtimecatalog

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"dev-toolbox/internal/userstate"
)

var tools = []string{"node", "java", "maven", "gradle", "androidsdk", "golang", "python"}

type Candidate struct {
	CandidateID  string   `json:"candidateId"`
	Tool         string   `json:"tool"`
	Version      string   `json:"version"`
	Major        string   `json:"major"`
	Home         string   `json:"home"`
	Binary       string   `json:"binary,omitempty"`
	Source       string   `json:"source"`
	Vendor       string   `json:"vendor,omitempty"`
	Evidence     []string `json:"evidence"`
	Availability string   `json:"availability"`
	Confidence   string   `json:"confidence"`
	Digest       string   `json:"digest"`
}

type Issue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Result struct {
	ScanID     string      `json:"scanId"`
	Status     string      `json:"status"`
	Candidates []Candidate `json:"candidates"`
	Issues     []Issue     `json:"issues"`
}

func Discover(ctx context.Context, requestedTools []string, roots []string) (Result, error) {
	selected := map[string]bool{}
	if len(requestedTools) == 0 {
		for _, tool := range tools {
			selected[tool] = true
		}
	} else {
		for _, tool := range requestedTools {
			if !knownTool(tool) {
				return Result{}, fmt.Errorf("RUNTIME_INVALID_TOOL: %s", tool)
			}
			selected[tool] = true
		}
	}
	result := Result{Status: "complete", Candidates: []Candidate{}, Issues: []Issue{}}
	seen := map[string]int{}
	add := func(tool, home, source string) {
		candidate, err := Inspect(tool, home, source)
		if err != nil {
			if errors.Is(err, os.ErrPermission) {
				result.Issues = append(result.Issues, Issue{Path: home, Code: "RUNTIME_CANDIDATE_INVALID", Message: err.Error()})
			}
			return
		}
		key := strings.ToLower(tool + "\x00" + filepath.Clean(candidate.Home))
		if runtime.GOOS != "windows" {
			key = tool + "\x00" + filepath.Clean(candidate.Home)
		}
		if index, ok := seen[key]; ok {
			result.Candidates[index].Evidence = appendUnique(result.Candidates[index].Evidence, candidate.Evidence...)
			result.Candidates[index].Digest = candidateDigest(result.Candidates[index])
			return
		}
		seen[key] = len(result.Candidates)
		result.Candidates = append(result.Candidates, candidate)
	}

	for tool := range selected {
		if err := ctx.Err(); err != nil {
			result.Status = "cancelled"
			return result, err
		}
		for _, home := range pathHomes(tool) {
			add(tool, home, "path")
		}
		for _, home := range commonHomes(tool) {
			if strings.TrimSpace(home) == "" {
				continue
			}
			add(tool, home, "system")
			if entries, readErr := os.ReadDir(home); readErr == nil {
				for index, entry := range entries {
					if index >= 200 {
						break
					}
					if entry.IsDir() {
						add(tool, filepath.Join(home, entry.Name()), "system")
					}
				}
			}
		}
	}
	for _, root := range roots {
		if err := ctx.Err(); err != nil {
			result.Status = "cancelled"
			return result, err
		}
		info, err := os.Stat(root)
		if err != nil {
			result.Issues = append(result.Issues, Issue{Path: root, Code: "RUNTIME_PERMISSION_DENIED", Message: err.Error()})
			continue
		}
		if !info.IsDir() {
			result.Issues = append(result.Issues, Issue{Path: root, Code: "RUNTIME_INVALID_PATH", Message: "指定路径不是目录"})
			continue
		}
		for tool := range selected {
			add(tool, root, "user")
			entries, readErr := os.ReadDir(root)
			if readErr != nil {
				result.Issues = append(result.Issues, Issue{Path: root, Code: "RUNTIME_PERMISSION_DENIED", Message: readErr.Error()})
				continue
			}
			for index, entry := range entries {
				if index >= 200 {
					result.Issues = append(result.Issues, Issue{Path: root, Code: "RUNTIME_SCAN_LIMIT", Message: "目录直接子项超过 200，已停止读取"})
					break
				}
				if entry.IsDir() {
					add(tool, filepath.Join(root, entry.Name()), "user")
				}
			}
		}
	}
	for i := range result.Candidates {
		result.Candidates[i].CandidateID = fingerprint(result.Candidates[i])
		result.Candidates[i].Digest = result.Candidates[i].CandidateID
	}
	sort.Slice(result.Candidates, func(i, j int) bool {
		if result.Candidates[i].Tool != result.Candidates[j].Tool {
			return result.Candidates[i].Tool < result.Candidates[j].Tool
		}
		return strings.ToLower(result.Candidates[i].Home) < strings.ToLower(result.Candidates[j].Home)
	})
	if len(result.Issues) > 0 {
		result.Status = "partial"
	}
	result.ScanID = candidateDigest(result.Candidates)
	return result, nil
}

func Inspect(tool, home, source string) (Candidate, error) {
	if !knownTool(tool) {
		return Candidate{}, fmt.Errorf("RUNTIME_INVALID_TOOL: %s", tool)
	}
	absolute, err := filepath.Abs(filepath.Clean(os.ExpandEnv(strings.TrimSpace(home))))
	if err != nil {
		return Candidate{}, fmt.Errorf("RUNTIME_INVALID_PATH: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return Candidate{}, fmt.Errorf("RUNTIME_PERMISSION_DENIED: %w", err)
		}
		return Candidate{}, fmt.Errorf("RUNTIME_INVALID_PATH: %w", err)
	}
	if !info.IsDir() {
		return Candidate{}, errors.New("RUNTIME_INVALID_PATH: not a directory")
	}
	binary := expectedBinary(tool, absolute)
	if binary == "" {
		return Candidate{}, errors.New("RUNTIME_CANDIDATE_INVALID: expected executable not found")
	}
	version, vendor, evidence := staticVersion(tool, absolute)
	if evidence == "" {
		evidence = "executable:" + binary
	}
	candidate := Candidate{Tool: tool, Version: version, Major: major(tool, version), Home: absolute, Binary: binary, Source: source, Vendor: vendor, Evidence: []string{evidence}, Availability: "available", Confidence: "exact"}
	if version == "" {
		candidate.Version = "unknown"
		candidate.Confidence = "unknown"
	}
	candidate.CandidateID = fingerprint(candidate)
	candidate.Digest = candidate.CandidateID
	return candidate, nil
}

func expectedBinary(tool, home string) string {
	var rels []string
	if runtime.GOOS == "windows" {
		rels = map[string][]string{
			"node": {"node.exe"}, "java": {"bin/java.exe", "jre/bin/java.exe"}, "maven": {"bin/mvn.cmd"}, "gradle": {"bin/gradle.bat"},
			"androidsdk": {"platform-tools/adb.exe"}, "golang": {"bin/go.exe"}, "python": {"python.exe", "Scripts/python.exe"},
		}[tool]
	} else {
		rels = map[string][]string{
			"node": {"bin/node", "node"}, "java": {"bin/java", "jre/bin/java"}, "maven": {"bin/mvn"}, "gradle": {"bin/gradle"},
			"androidsdk": {"platform-tools/adb"}, "golang": {"bin/go"}, "python": {"bin/python3", "bin/python", "python3", "python"},
		}[tool]
	}
	for _, rel := range rels {
		candidate := filepath.Join(home, filepath.FromSlash(rel))
		if isFile(candidate) {
			return candidate
		}
	}
	return ""
}

func staticVersion(tool, home string) (string, string, string) {
	switch tool {
	case "java":
		data, _ := os.ReadFile(filepath.Join(home, "release"))
		vendor := ""
		version := ""
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "JAVA_VERSION=") {
				version = strings.Trim(strings.TrimPrefix(line, "JAVA_VERSION="), `"`)
			}
			if strings.HasPrefix(line, "IMPLEMENTOR=") {
				vendor = strings.Trim(strings.TrimPrefix(line, "IMPLEMENTOR="), `"`)
			}
		}
		if version != "" {
			return version, vendor, "release:JAVA_VERSION"
		}
	case "maven":
		if entries, err := os.ReadDir(filepath.Join(home, "lib")); err == nil {
			for _, entry := range entries {
				if strings.HasPrefix(entry.Name(), "maven-core-") && strings.HasSuffix(entry.Name(), ".jar") {
					version := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "maven-core-"), ".jar")
					return version, "Apache", "lib/" + entry.Name()
				}
			}
		}
	case "gradle":
		if entries, err := os.ReadDir(filepath.Join(home, "lib")); err == nil {
			for _, entry := range entries {
				for _, prefix := range []string{"gradle-core-", "gradle-launcher-"} {
					if strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), ".jar") {
						version := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), prefix), ".jar")
						return version, "Gradle", "lib/" + entry.Name()
					}
				}
			}
		}
	case "androidsdk":
		data, _ := os.ReadFile(filepath.Join(home, "platform-tools", "source.properties"))
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "Pkg.Revision") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1]), "Android", "platform-tools/source.properties"
				}
			}
		}
	case "golang":
		data, _ := os.ReadFile(filepath.Join(home, "VERSION"))
		v := strings.TrimSpace(strings.SplitN(string(data), " ", 2)[0])
		if strings.HasPrefix(v, "go") {
			v = strings.TrimPrefix(v, "go")
		}
		if v != "" {
			return v, "Go", "VERSION"
		}
	case "python":
		data, _ := os.ReadFile(filepath.Join(home, "pyvenv.cfg"))
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.ToLower(line), "version =") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					return strings.TrimSpace(parts[1]), "Python", "pyvenv.cfg"
				}
			}
		}
	}
	base := filepath.Base(home)
	for _, prefix := range []string{"node-v", "v", "go", "apache-maven-", "gradle-", "Python-"} {
		if strings.HasPrefix(strings.ToLower(base), strings.ToLower(prefix)) {
			value := strings.TrimPrefix(strings.TrimPrefix(base, prefix), "v")
			if hasDigit(value) {
				return value, "", "directory-name"
			}
		}
	}
	return "", "", ""
}

func pathHomes(tool string) []string {
	names := map[string][]string{
		"node": {"node", "node.exe"}, "java": {"java", "java.exe"}, "maven": {"mvn", "mvn.cmd"}, "gradle": {"gradle", "gradle.bat"},
		"androidsdk": {"adb", "adb.exe"}, "golang": {"go", "go.exe"}, "python": {"python", "python3", "python.exe", "python3.exe"},
	}[tool]
	out := []string{}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		for _, name := range names {
			binary := filepath.Join(dir, name)
			if !isFile(binary) {
				continue
			}
			home := filepath.Dir(dir)
			switch tool {
			case "node", "python":
				home = dir
				if strings.EqualFold(filepath.Base(dir), "bin") || strings.EqualFold(filepath.Base(dir), "scripts") {
					home = filepath.Dir(dir)
				}
			case "androidsdk":
				home = filepath.Dir(dir)
			}
			out = append(out, home)
			break
		}
	}
	return out
}

func commonHomes(tool string) []string {
	var paths []string
	if runtime.GOOS == "windows" {
		pf := os.Getenv("ProgramFiles")
		local := os.Getenv("LOCALAPPDATA")
		paths = map[string][]string{
			"node": {under(pf, "nodejs")}, "java": {under(pf, "Java")}, "maven": {under(pf, "Apache Software Foundation", "Maven")},
			"gradle": {under(pf, "Gradle")}, "androidsdk": {os.Getenv("ANDROID_HOME"), under(local, "Android", "Sdk")},
			"golang": {os.Getenv("GOROOT"), under(pf, "Go")}, "python": {under(local, "Programs", "Python")},
		}[tool]
	} else {
		home := os.Getenv("HOME")
		paths = map[string][]string{
			"node": {"/usr", "/usr/local"}, "java": {"/usr/lib/jvm"}, "maven": {"/usr/share/maven", "/usr/local"}, "gradle": {"/opt/gradle", "/usr/local"},
			"androidsdk": {os.Getenv("ANDROID_HOME"), under(home, "Android", "Sdk")}, "golang": {"/usr/local/go"}, "python": {"/usr", "/usr/local"},
		}[tool]
	}
	return paths
}

func candidateDigest(value any) string {
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func under(root string, parts ...string) string {
	if strings.TrimSpace(root) == "" {
		return ""
	}
	return filepath.Join(append([]string{root}, parts...)...)
}
func fingerprint(candidate Candidate) string {
	candidate.CandidateID = ""
	candidate.Digest = ""
	return candidateDigest(candidate)
}
func appendUnique(items []string, values ...string) []string {
	for _, value := range values {
		found := false
		for _, existing := range items {
			if existing == value {
				found = true
				break
			}
		}
		if !found {
			items = append(items, value)
		}
	}
	return items
}
func knownTool(tool string) bool {
	for _, item := range tools {
		if item == tool {
			return true
		}
	}
	return false
}
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
func hasDigit(value string) bool {
	for _, r := range value {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}
func major(tool, version string) string {
	if version == "" || version == "unknown" {
		return ""
	}
	value := strings.TrimPrefix(version, "v")
	if tool == "java" && strings.HasPrefix(value, "1.") {
		parts := strings.Split(value, ".")
		if len(parts) > 1 {
			return parts[1]
		}
	}
	return strings.Split(value, ".")[0]
}

type CandidateRef struct {
	CandidateID string `json:"candidateId"`
	Digest      string `json:"digest"`
	Tool        string `json:"tool"`
	Home        string `json:"home"`
	Source      string `json:"source"`
}

func Register(ctx context.Context, refs []CandidateRef, revision uint64, confirmed bool) (userstate.State, userstate.RegistrationCounts, error) {
	if !confirmed {
		return userstate.State{}, userstate.RegistrationCounts{}, errors.New("RUNTIME_CONFIRMATION_REQUIRED")
	}
	items := make([]userstate.Installation, 0, len(refs))
	for _, ref := range refs {
		if err := ctx.Err(); err != nil {
			return userstate.State{}, userstate.RegistrationCounts{}, err
		}
		candidate, err := Inspect(ref.Tool, ref.Home, ref.Source)
		if err != nil || candidate.CandidateID != ref.CandidateID || candidate.Digest != ref.Digest {
			return userstate.State{}, userstate.RegistrationCounts{}, errors.New("RUNTIME_CANDIDATE_STALE")
		}
		items = append(items, userstate.Installation{Runtime: candidate.Tool, Version: candidate.Version, Major: candidate.Major, LocalHome: candidate.Home, BinaryPath: candidate.Binary, Source: candidate.Source, Vendor: candidate.Vendor})
	}
	return userstate.RegisterInstallations(items, revision, confirmed)
}
