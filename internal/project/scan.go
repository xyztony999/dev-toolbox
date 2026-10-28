package project

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	maxDepth = 12
	maxFiles = 50000
	maxBytes = 1 << 20
)

var ignoredDirectories = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".venv": true,
	"venv": true, "__pycache__": true, "dist": true, "build": true,
	"target": true, ".gradle": true, "coverage": true, "bin": true,
}

type moduleFiles struct {
	path  string
	files map[string]string
}

func scan(ctx context.Context, root string) (Candidate, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return Candidate{}, fmt.Errorf("PROJECT_INVALID_PATH: empty path")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return Candidate{}, fmt.Errorf("PROJECT_INVALID_PATH: %w", err)
	}
	canonical, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return Candidate{}, fmt.Errorf("PROJECT_INVALID_PATH: %w", err)
	}
	info, err := os.Stat(canonical)
	if err != nil || !info.IsDir() {
		return Candidate{}, fmt.Errorf("PROJECT_INVALID_PATH: expected a directory")
	}

	candidate := Candidate{
		ID:           stableID(canonical),
		DisplayName:  filepath.Base(canonical),
		RootPath:     canonical,
		Status:       ScanComplete,
		ScannedAt:    time.Now().UTC(),
		Repositories: []Repository{},
		Modules:      []Module{},
		Issues:       []ScanIssue{},
	}
	modules := map[string]*moduleFiles{}
	fileCount := 0
	walkErr := filepath.WalkDir(canonical, func(path string, entry fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			candidate.Status = ScanCancelled
			candidate.Issues = append(candidate.Issues, ScanIssue{Code: "PROJECT_SCAN_CANCELLED", Severity: "info", Message: "扫描已取消", Recoverable: true})
			return filepath.SkipAll
		}
		if walkErr != nil {
			rel, _ := filepath.Rel(canonical, path)
			candidate.Issues = append(candidate.Issues, ScanIssue{
				Code: "PROJECT_PERMISSION_DENIED", Severity: "warning", RelativePath: filepath.ToSlash(rel),
				Message: "部分路径无法读取", Recoverable: true,
			})
			candidate.Status = ScanPartial
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(canonical, path)
		if err != nil {
			return err
		}
		depth := 0
		if rel != "." {
			depth = strings.Count(filepath.Clean(rel), string(filepath.Separator)) + 1
		}
		if entry.IsDir() {
			if rel != "." && ignoredDirectories[strings.ToLower(entry.Name())] {
				return filepath.SkipDir
			}
			if depth > maxDepth {
				candidate.Issues = append(candidate.Issues, ScanIssue{Code: "PROJECT_SCAN_DEPTH_LIMIT", Severity: "info", RelativePath: filepath.ToSlash(rel), Message: "已达到最大扫描深度", Recoverable: true})
				candidate.Status = ScanPartial
				return filepath.SkipDir
			}
			if kind, ok := repositoryKind(path); ok {
				if rel != "." && kind == "normal" {
					kind = "nested"
				}
				candidate.Repositories = append(candidate.Repositories, Repository{
					ID: stableID(path), Path: canonicalPath(path, canonical), Kind: kind, Source: ".git metadata",
				})
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 || !entry.Type().IsRegular() {
			return nil
		}
		if !isManifest(entry.Name()) {
			return nil
		}
		fileCount++
		if fileCount > maxFiles {
			candidate.Issues = append(candidate.Issues, ScanIssue{Code: "PROJECT_SCAN_FILE_LIMIT", Severity: "warning", Message: "已达到最大清单数量", Recoverable: true})
			candidate.Status = ScanPartial
			return filepath.SkipAll
		}
		st, err := entry.Info()
		if err != nil || st.Size() > maxBytes {
			candidate.Issues = append(candidate.Issues, ScanIssue{Code: "PROJECT_MANIFEST_UNREADABLE", Severity: "warning", RelativePath: filepath.ToSlash(rel), Message: "清单不可读取或超过大小限制", Recoverable: true})
			candidate.Status = ScanPartial
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			candidate.Issues = append(candidate.Issues, ScanIssue{Code: "PROJECT_PERMISSION_DENIED", Severity: "warning", RelativePath: filepath.ToSlash(rel), Message: "清单不可读取", Recoverable: true})
			candidate.Status = ScanPartial
			return nil
		}
		modulePath := filepath.Dir(rel)
		if modulePath == "" {
			modulePath = "."
		}
		moduleKey := filepath.Clean(modulePath)
		m := modules[moduleKey]
		if m == nil {
			m = &moduleFiles{path: moduleKey, files: map[string]string{}}
			modules[moduleKey] = m
		}
		m.files[filepath.Base(path)] = string(data)
		return nil
	})
	if walkErr != nil {
		return Candidate{}, fmt.Errorf("PROJECT_SCAN_FAILED: %w", walkErr)
	}

	for _, files := range modules {
		candidate.Modules = append(candidate.Modules, makeModule(files))
	}
	sort.Slice(candidate.Repositories, func(i, j int) bool { return candidate.Repositories[i].Path < candidate.Repositories[j].Path })
	sort.Slice(candidate.Modules, func(i, j int) bool { return candidate.Modules[i].RelativePath < candidate.Modules[j].RelativePath })
	candidate.EvidenceDigest = digestCandidate(candidate)
	return candidate, nil
}

func makeModule(files *moduleFiles) Module {
	builds := map[string]bool{}
	for name := range files.files {
		switch strings.ToLower(name) {
		case "package.json":
			builds["node"] = true
		case ".nvmrc":
			builds["node"] = true
		case "pyproject.toml", "requirements.txt", "setup.py", "setup.cfg":
			builds["python"] = true
		case "go.mod":
			builds["go"] = true
		case "pom.xml":
			builds["maven"] = true
			if _, ok := files.files["mvnw"]; ok {
				builds["maven-wrapper"] = true
			}
		case "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts":
			builds["gradle"] = true
			if strings.Contains(strings.ToLower(files.files[name]), "com.android.application") || strings.Contains(strings.ToLower(files.files[name]), "com.android.library") {
				builds["android-gradle"] = true
			}
			if _, ok := files.files["gradlew"]; ok {
				builds["gradle-wrapper"] = true
			}
		}
	}
	var requirements []RuntimeRequirement
	for name, raw := range files.files {
		switch strings.ToLower(name) {
		case "package.json":
			var data struct {
				Engines map[string]string `json:"engines"`
			}
			if json.Unmarshal([]byte(raw), &data) == nil {
				if v := strings.TrimSpace(data.Engines["node"]); v != "" {
					requirements = append(requirements, requirement("node", v, files.path, name, "package.engines.node", "high", "package-json"))
				}
			}
		case ".nvmrc":
			if v := strings.TrimSpace(raw); v != "" {
				requirements = append(requirements, requirement("node", v, files.path, name, "file", "medium", "nvmrc"))
			}
		case "pyproject.toml":
			if v := findPythonConstraint(raw); v != "" {
				requirements = append(requirements, requirement("python", v, files.path, name, "project.requires-python", "high", "pyproject"))
			}
		case "go.mod":
			if v := regexp.MustCompile(`(?m)^go\s+([^\s]+)`).FindStringSubmatch(raw); len(v) == 2 {
				requirements = append(requirements, requirement("go", v[1], files.path, name, "go directive", "high", "go-mod"))
			}
		case "pom.xml":
			for _, item := range findMavenJavaVersions(raw) {
				requirements = append(requirements, requirement("java", item.value, files.path, name, item.locator, "high", "maven-pom"))
			}
		case "build.gradle", "build.gradle.kts":
			for _, v := range findGradleJavaVersions(raw) {
				requirements = append(requirements, requirement("java", v, files.path, name, "java toolchain", "medium", "gradle"))
			}
			for _, v := range findAndroidSDKVersions(raw) {
				requirements = append(requirements, requirement("androidsdk", v.value, files.path, name, v.locator, "medium", "gradle-android"))
			}
		}
	}
	for name := range files.files {
		if strings.EqualFold(name, "pom.xml") {
			if _, ok := files.files["mvnw"]; ok {
				builds["maven-wrapper"] = true
			}
		}
		if strings.EqualFold(name, "build.gradle") || strings.EqualFold(name, "build.gradle.kts") {
			if _, ok := files.files["gradlew"]; ok {
				builds["gradle-wrapper"] = true
			}
		}
	}
	buildList := make([]string, 0, len(builds))
	for name := range builds {
		buildList = append(buildList, name)
	}
	sort.Strings(buildList)
	markConflicts(requirements)
	seenRuntime := map[string]bool{}
	for _, req := range requirements {
		seenRuntime[req.Runtime] = true
	}
	for _, build := range buildList {
		runtime := ""
		switch build {
		case "node":
			runtime = "node"
		case "python":
			runtime = "python"
		case "go":
			runtime = "go"
		case "maven", "gradle":
			runtime = "java"
		case "android-gradle":
			runtime = "androidsdk"
		}
		if runtime != "" && !seenRuntime[runtime] {
			requirements = append(requirements, RuntimeRequirement{Runtime: runtime, Confidence: "low", Status: RequirementUnknown})
			seenRuntime[runtime] = true
		}
	}
	sort.Slice(requirements, func(i, j int) bool {
		if requirements[i].Runtime != requirements[j].Runtime {
			return requirements[i].Runtime < requirements[j].Runtime
		}
		return requirements[i].SourcePath < requirements[j].SourcePath
	})
	rel := "."
	if files.path != "." {
		rel = filepath.ToSlash(files.path)
	}
	return Module{ID: stableID(rel), RelativePath: rel, BuildSystems: buildList, Requirements: requirements}
}

func requirement(runtime, raw, modulePath, filename, locator, confidence, parser string) RuntimeRequirement {
	source := filename
	if modulePath != "." {
		source = filepath.ToSlash(filepath.Join(modulePath, filename))
	}
	return RuntimeRequirement{
		Runtime: runtime, RawConstraint: strings.TrimSpace(raw), Confidence: confidence,
		Status: RequirementResolved, SourcePath: source,
		Evidence: []Evidence{{SourcePath: source, Locator: locator, RawValue: strings.TrimSpace(raw), ParserID: parser, ObservedAt: time.Now().UTC()}},
	}
}

func markConflicts(reqs []RuntimeRequirement) {
	values := map[string]map[string]bool{}
	for _, r := range reqs {
		if values[r.Runtime] == nil {
			values[r.Runtime] = map[string]bool{}
		}
		values[r.Runtime][strings.ToLower(strings.TrimSpace(r.RawConstraint))] = true
	}
	for i := range reqs {
		if len(values[reqs[i].Runtime]) > 1 {
			allExact := true
			for raw := range values[reqs[i].Runtime] {
				if !regexp.MustCompile(`(?i)^v?[0-9]+(?:\.[0-9]+){0,2}$`).MatchString(raw) {
					allExact = false
					break
				}
			}
			if allExact {
				reqs[i].Status = RequirementConflict
			} else {
				reqs[i].Status = RequirementUnknown
			}
		}
	}
}

func digestCandidate(c Candidate) string {
	copy := c
	copy.EvidenceDigest = ""
	copy.ScannedAt = time.Time{}
	for i := range copy.Modules {
		for j := range copy.Modules[i].Requirements {
			for k := range copy.Modules[i].Requirements[j].Evidence {
				copy.Modules[i].Requirements[j].Evidence[k].ObservedAt = time.Time{}
			}
		}
	}
	b, _ := json.Marshal(copy)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func isManifest(name string) bool {
	switch strings.ToLower(name) {
	case "package.json", ".nvmrc", "pyproject.toml", "requirements.txt", "setup.py", "setup.cfg", "go.mod", "pom.xml", "mvnw", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts", "gradlew":
		return true
	default:
		return false
	}
}

func repositoryKind(dir string) (string, bool) {
	marker := filepath.Join(dir, ".git")
	info, err := os.Lstat(marker)
	if err != nil {
		return "", false
	}
	if info.IsDir() {
		if _, err := os.Stat(filepath.Join(marker, "HEAD")); err != nil {
			return "", false
		}
		if _, err := os.Stat(filepath.Join(marker, "config")); err != nil {
			return "", false
		}
		return "normal", true
	}
	if !info.Mode().IsRegular() || info.Size() > 4096 {
		return "", false
	}
	b, err := os.ReadFile(marker)
	if err != nil {
		return "", false
	}
	line := strings.TrimSpace(string(b))
	if !strings.HasPrefix(strings.ToLower(line), "gitdir:") {
		return "", false
	}
	gitDir := strings.TrimSpace(line[len("gitdir:"):])
	if gitDir == "" {
		return "", false
	}
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(dir, gitDir)
	}
	if st, err := os.Stat(gitDir); err == nil && st.IsDir() {
		return "worktree", true
	}
	return "", false
}

func canonicalPath(path, root string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		if rel == "." {
			return root
		}
		return filepath.Join(root, rel)
	}
	return path
}

func findPythonConstraint(raw string) string {
	re := regexp.MustCompile(`(?m)^\s*requires-python\s*=\s*["']([^"']+)["']`)
	if m := re.FindStringSubmatch(raw); len(m) == 2 {
		return m[1]
	}
	poetry := regexp.MustCompile(`(?m)^\s*python\s*=\s*["']([^"']+)["']`)
	if m := poetry.FindStringSubmatch(raw); len(m) == 2 {
		return m[1]
	}
	return ""
}

type sdkVersion struct{ value, locator string }

func findAndroidSDKVersions(raw string) []sdkVersion {
	compile := regexp.MustCompile(`(?mi)^\s*compileSdk(?:Version)?\s*(?:=\s*)?([0-9]+)`)
	min := regexp.MustCompile(`(?mi)^\s*minSdk(?:Version)?\s*(?:=\s*)?([0-9]+)`)
	var found []sdkVersion
	for _, item := range []struct {
		re   *regexp.Regexp
		name string
	}{{compile, "compileSdk"}, {min, "minSdk"}} {
		for _, match := range item.re.FindAllStringSubmatch(raw, -1) {
			if len(match) == 2 {
				found = append(found, sdkVersion{value: match[1], locator: item.name})
			}
		}
	}
	return found
}

type mavenVersion struct{ value, locator string }

func findMavenJavaVersions(raw string) []mavenVersion {
	decoder := xml.NewDecoder(strings.NewReader(raw))
	var result []mavenVersion
	var stack []string
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			stack = append(stack, t.Name.Local)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) >= 2 && stack[len(stack)-2] == "properties" {
				name := stack[len(stack)-1]
				if name == "maven.compiler.release" || name == "maven.compiler.target" || name == "java.version" {
					v := strings.TrimSpace(string(t))
					if v != "" {
						result = append(result, mavenVersion{value: v, locator: name})
					}
				}
			}
		}
	}
	return result
}

func findGradleJavaVersions(raw string) []string {
	re := regexp.MustCompile(`(?i)(?:JavaLanguageVersion\.of\s*\(\s*|languageVersion\s*=\s*JavaLanguageVersion\.of\s*\(\s*)([0-9]+)`)
	matches := re.FindAllStringSubmatch(raw, -1)
	var result []string
	seen := map[string]bool{}
	for _, m := range matches {
		if len(m) == 2 && !seen[m[1]] {
			result = append(result, m[1])
			seen[m[1]] = true
		}
	}
	return result
}
