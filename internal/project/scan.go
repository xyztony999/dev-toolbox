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

	"dev-toolbox/internal/gitidentity"
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

// shouldIgnoreDir 在精确名单之外识别 venv 族变体（.venv-workers、
// .venv-prod 等自定义虚拟环境名）：它们是可重建的依赖目录，深入只会
// 触发深度上限（partial 快照被拒收，SEG2105 用户场景即此）。
func shouldIgnoreDir(name string) bool {
	lower := strings.ToLower(name)
	if ignoredDirectories[lower] {
		return true
	}
	return strings.HasPrefix(lower, ".venv") || strings.HasPrefix(lower, "venv")
}

type moduleFiles struct {
	path  string
	files map[string]string
}

var (
	cargoRustVersion     = regexp.MustCompile(`(?m)^\s*rust-version\s*=\s*"([^"]+)"`)
	rustToolchainChannel = regexp.MustCompile(`(?m)^\s*channel\s*=\s*"([^"]+)"`)
)

// bareRustChannel 解析无扩展名 rust-toolchain 文件：首个非注释非空行即通道名。
func bareRustChannel(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return trimmed
	}
	return ""
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
			if rel != "." && shouldIgnoreDir(entry.Name()) {
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
				repo := Repository{
					ID: stableID(path), Path: canonicalPath(path, canonical), Kind: kind, Source: ".git metadata",
				}
				repo.Branch, _ = gitidentity.ReadBranch(ctx, path)
				if name, url, ok := gitidentity.ReadLocalRemote(ctx, path); ok {
					repo.RemoteName, repo.RemoteURL = name, url
				}
				repo.UserName, _ = gitidentity.ReadLocalConfig(ctx, path, "user.name")
				repo.UserEmail, _ = gitidentity.ReadLocalConfig(ctx, path, "user.email")
				candidate.Repositories = append(candidate.Repositories, repo)
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
		case "cargo.toml":
			builds["cargo"] = true
		case "rust-toolchain", "rust-toolchain.toml":
			builds["rust"] = true
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
			for _, item := range findGradleJavaVersions(raw) {
				req := requirement("java", item.constraint, files.path, name, item.locator, "medium", "gradle")
				req.Evidence[0].RawValue = item.observed
				requirements = append(requirements, req)
			}
			for _, v := range findAndroidSDKVersions(raw) {
				requirements = append(requirements, requirement("androidsdk", v.value, files.path, name, v.locator, "medium", "gradle-android"))
			}
		case "cargo.toml":
			if m := cargoRustVersion.FindStringSubmatch(raw); len(m) == 2 {
				requirements = append(requirements, requirement("rust", m[1], files.path, name, "package.rust-version", "high", "cargo"))
			}
		case "rust-toolchain":
			if v := bareRustChannel(raw); v != "" {
				requirements = append(requirements, requirement("rust", v, files.path, name, "toolchain.channel", "high", "rust-toolchain"))
			}
		case "rust-toolchain.toml":
			if m := rustToolchainChannel.FindStringSubmatch(raw); len(m) == 2 {
				requirements = append(requirements, requirement("rust", m[1], files.path, name, "toolchain.channel", "high", "rust-toolchain"))
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
		case "cargo", "rust":
			runtime = "rust"
		}
		if runtime != "" && !seenRuntime[runtime] {
			requirements = append(requirements, RuntimeRequirement{Runtime: runtime, Confidence: "low", Status: RequirementUnknown})
			seenRuntime[runtime] = true
		}
	}
	// 稳定排序：同键条目保留清单中的声明顺序（如 compileSdk 在 minSdk 前）。
	sort.SliceStable(requirements, func(i, j int) bool {
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

// ScanToolVersion 随扫描结果落盘（Project.scanToolVersion）：扫描判定
// 逻辑演进后，客户端可据此提示旧快照已过期、建议重新扫描——避免
// "判定已修、界面仍显示旧结论"的错觉（2026-11-23 androidsdk 误报即此）。
const ScanToolVersion = "3"

func markConflicts(reqs []RuntimeRequirement) {
	// 冲突按键：默认按 runtime 分组——单一语义版本（java toolchain、
	// maven 属性族、node、python 等）多来源互斥值即冲突（区间混入时
	// 降级 unknown，不擅自任选）。
	// 多属性运行时按 runtime+locator 拆组：同一属性的多个互斥值才是
	// 冲突，不同属性各自独立成立——androidsdk 的 compileSdk/minSdk、
	// gradle 字节码兼容属性对（source/targetCompatibility、jvmTarget，
	// 可与 toolchain 共存不同值）均不构成冲突（SEG2105/app 误报即此）。
	group := func(r RuntimeRequirement) string {
		if len(r.Evidence) > 0 {
			key := r.Runtime + "/" + r.Evidence[0].Locator
			switch key {
			case "androidsdk/compileSdk", "androidsdk/minSdk",
				"java/sourceCompatibility", "java/targetCompatibility", "java/jvmTarget":
				return r.Runtime + ":" + r.Evidence[0].Locator
			}
		}
		return r.Runtime
	}
	values := map[string]map[string]bool{}
	for _, r := range reqs {
		key := group(r)
		if values[key] == nil {
			values[key] = map[string]bool{}
		}
		values[key][strings.ToLower(strings.TrimSpace(r.RawConstraint))] = true
	}
	for i := range reqs {
		seen := values[group(reqs[i])]
		if len(seen) > 1 {
			allExact := true
			for raw := range seen {
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
	case "package.json", ".nvmrc", "pyproject.toml", "requirements.txt", "setup.py", "setup.cfg", "go.mod", "pom.xml", "mvnw", "build.gradle", "build.gradle.kts", "settings.gradle", "settings.gradle.kts", "gradlew", "cargo.toml", "rust-toolchain", "rust-toolchain.toml":
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

type gradleJavaVersion struct {
	constraint string // 规范化约束（进 RawConstraint，参与冲突比较）
	observed   string // 原始记号（进证据 RawValue，如 VERSION_11）
	locator    string
}

// gradleJavaPatterns 覆盖 build.gradle(.kts) 的三类 Java 版本声明：
// 1) toolchain languageVersion（权威声明）；2) compileOptions 的
// source/targetCompatibility；3) kotlinOptions 的 jvmTarget。
// source/target 与 toolchain 彼此独立、可共存不同值（如 toolchain 17 +
// source 11），冲突判定按属性拆组（见 markConflicts）。
var gradleJavaPatterns = []struct {
	re      *regexp.Regexp
	locator string
}{
	{regexp.MustCompile(`(?i)JavaLanguageVersion\.of\s*\(\s*([0-9]+)\s*\)`), "java toolchain"},
	{regexp.MustCompile(`(?i)sourceCompatibility\s*=?\s*(?:JavaVersion\.)?VERSION_([0-9]+(?:_[0-9]+)?)`), "sourceCompatibility"},
	{regexp.MustCompile(`(?i)targetCompatibility\s*=?\s*(?:JavaVersion\.)?VERSION_([0-9]+(?:_[0-9]+)?)`), "targetCompatibility"},
	{regexp.MustCompile(`(?i)jvmTarget\s*=?\s*["']?(?:JavaVersion\.)?VERSION_([0-9]+(?:_[0-9]+)?)["']?`), "jvmTarget"},
	{regexp.MustCompile(`(?i)jvmTarget\s*\(\s*["']?([0-9]+(?:\.[0-9]+)?)`), "jvmTarget"},
	{regexp.MustCompile(`(?i)sourceCompatibility\s*=?\s*["']?([0-9]+(?:\.[0-9]+)?)["']?`), "sourceCompatibility"},
	{regexp.MustCompile(`(?i)targetCompatibility\s*=?\s*["']?([0-9]+(?:\.[0-9]+)?)["']?`), "targetCompatibility"},
	{regexp.MustCompile(`(?i)jvmTarget\s*=?\s*["']?([0-9]+(?:\.[0-9]+)?)["']?`), "jvmTarget"},
}

var javaLegacyVersion = regexp.MustCompile(`^1_([0-9]+)$`)

// normalizeJavaConstraint：VERSION_1_8 / 字面量 1.8 / 单数字 8 同义
// （Gradle JavaVersion.toVersion 语义），统一为 "1.8" 避免同义误报
// 冲突；9+ 保留原样（无 "1.11" 纪元）。
func normalizeJavaConstraint(token string) string {
	t := strings.TrimSpace(token)
	switch t {
	case "6", "1_6", "1.6":
		return "1.6"
	case "7", "1_7", "1.7":
		return "1.7"
	case "8", "1_8", "1.8":
		return "1.8"
	}
	if m := javaLegacyVersion.FindStringSubmatch(t); m != nil {
		return "1." + m[1]
	}
	return t
}

func findGradleJavaVersions(raw string) []gradleJavaVersion {
	var found []gradleJavaVersion
	seen := map[string]bool{}
	for _, pattern := range gradleJavaPatterns {
		for _, match := range pattern.re.FindAllStringSubmatch(raw, -1) {
			token := strings.TrimSpace(match[1])
			if token == "" {
				continue
			}
			constraint := normalizeJavaConstraint(token)
			key := pattern.locator + "/" + constraint
			if seen[key] {
				continue
			}
			seen[key] = true
			found = append(found, gradleJavaVersion{constraint: constraint, observed: token, locator: pattern.locator})
		}
	}
	return found
}
