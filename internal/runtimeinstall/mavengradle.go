package runtimeinstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"dev-toolbox/internal/userstate"
)

// sidecarChecksum resolves the announced digest for archives that publish
// .sha256/.sha512 companion files. A companion that returns non-hex content
// (mirror HTML error pages) is treated as missing, never trusted.
func sidecarChecksum(ctx context.Context, f *fetcher, archiveURL string) (string, string, error) {
	for _, candidate := range []struct{ suffix, algo string }{
		{".sha256", "sha256"},
		{".sha512", "sha512"},
	} {
		text, err := f.getText(ctx, archiveURL+candidate.suffix)
		if err != nil {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(text))
		wantLen := 64
		if candidate.algo == "sha512" {
			wantLen = 128
		}
		if len(fields) > 0 && len(fields[0]) == wantLen && isHex(fields[0]) {
			return candidate.algo, strings.ToLower(fields[0]), nil
		}
		return "", "", errors.New("RUNTIME_CHECKSUM_MISSING: checksum companion content is not a digest")
	}
	return "", "", errors.New("RUNTIME_CHECKSUM_MISSING: no .sha256/.sha512 companion for archive")
}

// ---------------------------------------------------------------------------
// Maven
// ---------------------------------------------------------------------------

type mavenProvider struct{}

func (mavenProvider) option() InstallOption {
	return InstallOption{
		Tool:               "maven",
		ProviderID:         "apache-maven",
		Mirrors:            []MirrorOption{{ID: "official", Label: "Apache 官方源"}, {ID: "tuna", Label: "清华 TUNA 镜像"}},
		VersionSelectors:   []string{"latest", "major"},
		SupportedPlatforms: []string{"windows", "linux"},
		SupportedArchs:     []string{"amd64", "arm64"},
	}
}

var mavenMetaVersionRe = regexp.MustCompile(`<version>([^<]+)</version>`)
var mavenMetaReleaseRe = regexp.MustCompile(`<release>([^<]+)</release>`)

func (mavenProvider) preview(ctx context.Context, f *fetcher, selector, mirror string) (*InstallPlan, error) {
	osName := "linux"
	switch runtime.GOOS {
	case "windows":
		osName = "windows"
	case "linux":
	default:
		return nil, errors.New("RUNTIME_PROVIDER_UNSUPPORTED: maven has no verified package for this OS")
	}
	selector = normalizeSelector(selector)
	selector, err := majorSelector(selector, true)
	if err != nil {
		return nil, err
	}
	metadata, err := f.getText(ctx, "https://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/maven-metadata.xml")
	if err != nil {
		return nil, fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: %w", err)
	}
	version := ""
	if selector == "latest" {
		if m := mavenMetaReleaseRe.FindStringSubmatch(metadata); m != nil {
			version = m[1]
		}
	} else {
		for _, m := range mavenMetaVersionRe.FindAllStringSubmatch(metadata, -1) {
			if strings.HasPrefix(m[1], selector+".") {
				version = m[1]
			}
		}
	}
	if version == "" {
		return nil, fmt.Errorf("RUNTIME_VERSION_UNAVAILABLE: no maven release in series %q", selector)
	}
	major := majorOf(version)
	var base string
	switch mirrorID(mirror) {
	case "", "official":
		base = fmt.Sprintf("https://dlcdn.apache.org/maven/maven-%s/%s/binaries", major, version)
	case "tuna":
		base = fmt.Sprintf("https://mirrors.tuna.tsinghua.edu.cn/apache/maven/maven-%s/%s/binaries", major, version)
	default:
		return nil, errors.New("RUNTIME_PROVIDER_UNSUPPORTED: unknown maven mirror")
	}
	fileName := fmt.Sprintf("apache-maven-%s-bin.zip", version)
	algo, sum, err := sidecarChecksum(ctx, f, base+"/"+fileName)
	if err != nil {
		return nil, err
	}
	root, err := RuntimeRoot()
	if err != nil {
		return nil, err
	}
	srcs := []pkgSource{{Name: "maven", FileName: fileName, URL: base + "/" + fileName, Algo: algo, Sum: sum}}
	return assemblePlan(ctx, f, "maven", "apache-maven", mirror, selector, version, osName, runtime.GOARCH, filepath.Join(root, "apache-maven-"+version), srcs)
}

func (mavenProvider) sources(plan *InstallPlan) ([]pkgSource, error) {
	return planSources(plan)
}

func (mavenProvider) install(ctx context.Context, f *fetcher, plan *InstallPlan, run *installRun) (userstate.Installation, error) {
	srcs, err := mavenProvider{}.sources(plan)
	if err != nil {
		return userstate.Installation{}, err
	}
	stage, rootDir, err := stagePackage(ctx, f, srcs[0], run)
	defer os.RemoveAll(stage)
	if err != nil {
		return userstate.Installation{}, err
	}
	script := filepath.Join("bin", "mvn")
	if plan.OS == "windows" {
		script = filepath.Join("bin", "mvn.cmd")
	}
	if !regularFile(filepath.Join(rootDir, script)) {
		return userstate.Installation{}, errors.New("RUNTIME_INSTALL_LAYOUT_INVALID: mvn launcher missing after extraction")
	}
	makeExecutable(filepath.Join(rootDir, "bin", "mvn"))
	if err := moveInto(rootDir, plan.TargetHome); err != nil {
		return userstate.Installation{}, err
	}
	run.mark("maven", "done", nil)
	return userstate.Installation{
		Runtime: "maven", Version: plan.ResolvedVersion, Major: majorOf(plan.ResolvedVersion),
		LocalHome: plan.TargetHome, BinaryPath: filepath.Join(plan.TargetHome, script),
		Source: "downloaded", Label: "Apache Maven " + plan.ResolvedVersion, Vendor: "Apache", Available: true,
	}, nil
}

func (mavenProvider) checkTarget(plan *InstallPlan) error {
	return defaultCheckTarget(plan)
}

// ---------------------------------------------------------------------------
// Gradle
// ---------------------------------------------------------------------------

type gradleProvider struct{}

type gradleVersionEntry struct {
	Version      string `json:"version"`
	Snapshot     bool   `json:"snapshot"`
	RcFor        string `json:"rcFor"`
	MilestoneFor string `json:"milestoneFor"`
}

func (gradleProvider) option() InstallOption {
	return InstallOption{
		Tool:               "gradle",
		ProviderID:         "gradle-distribution",
		Mirrors:            []MirrorOption{{ID: "official", Label: "Gradle 官方源"}, {ID: "tencent", Label: "腾讯云镜像"}},
		VersionSelectors:   []string{"latest", "major"},
		SupportedPlatforms: []string{"windows", "linux"},
		SupportedArchs:     []string{"amd64", "arm64"},
	}
}

func (gradleProvider) preview(ctx context.Context, f *fetcher, selector, mirror string) (*InstallPlan, error) {
	osName := "linux"
	switch runtime.GOOS {
	case "windows":
		osName = "windows"
	case "linux":
	default:
		return nil, errors.New("RUNTIME_PROVIDER_UNSUPPORTED: gradle has no verified package for this OS")
	}
	selector = normalizeSelector(selector)
	selector, err := majorSelector(selector, true)
	if err != nil {
		return nil, err
	}
	var entries []gradleVersionEntry
	if err := f.getJSON(ctx, "https://services.gradle.org/versions/all", &entries); err != nil {
		return nil, fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: %w", err)
	}
	version := ""
	for _, entry := range entries {
		if entry.Snapshot || entry.RcFor != "" || entry.MilestoneFor != "" {
			continue
		}
		if selector == "latest" || strings.HasPrefix(entry.Version, selector+".") {
			version = entry.Version
			break
		}
	}
	if version == "" {
		return nil, fmt.Errorf("RUNTIME_VERSION_UNAVAILABLE: no gradle release in series %q", selector)
	}
	var base string
	switch mirrorID(mirror) {
	case "", "official":
		base = "https://services.gradle.org/distributions"
	case "tencent":
		base = "https://mirrors.cloud.tencent.com/gradle"
	default:
		return nil, errors.New("RUNTIME_PROVIDER_UNSUPPORTED: unknown gradle mirror")
	}
	fileName := fmt.Sprintf("gradle-%s-bin.zip", version)
	algo, sum, err := sidecarChecksum(ctx, f, base+"/"+fileName)
	if err != nil {
		return nil, err
	}
	root, err := RuntimeRoot()
	if err != nil {
		return nil, err
	}
	srcs := []pkgSource{{Name: "gradle", FileName: fileName, URL: base + "/" + fileName, Algo: algo, Sum: sum}}
	return assemblePlan(ctx, f, "gradle", "gradle-distribution", mirror, selector, version, osName, runtime.GOARCH, filepath.Join(root, "gradle-"+version), srcs)
}

func (gradleProvider) sources(plan *InstallPlan) ([]pkgSource, error) {
	return planSources(plan)
}

func (gradleProvider) install(ctx context.Context, f *fetcher, plan *InstallPlan, run *installRun) (userstate.Installation, error) {
	srcs, err := gradleProvider{}.sources(plan)
	if err != nil {
		return userstate.Installation{}, err
	}
	stage, rootDir, err := stagePackage(ctx, f, srcs[0], run)
	defer os.RemoveAll(stage)
	if err != nil {
		return userstate.Installation{}, err
	}
	script := filepath.Join("bin", "gradle")
	if plan.OS == "windows" {
		script = filepath.Join("bin", "gradle.bat")
	}
	if !regularFile(filepath.Join(rootDir, script)) {
		return userstate.Installation{}, errors.New("RUNTIME_INSTALL_LAYOUT_INVALID: gradle launcher missing after extraction")
	}
	makeExecutable(filepath.Join(rootDir, "bin", "gradle"))
	if err := moveInto(rootDir, plan.TargetHome); err != nil {
		return userstate.Installation{}, err
	}
	run.mark("gradle", "done", nil)
	return userstate.Installation{
		Runtime: "gradle", Version: plan.ResolvedVersion, Major: majorOf(plan.ResolvedVersion),
		LocalHome: plan.TargetHome, BinaryPath: filepath.Join(plan.TargetHome, script),
		Source: "downloaded", Label: "Gradle " + plan.ResolvedVersion, Vendor: "Gradle", Available: true,
	}, nil
}

func (gradleProvider) checkTarget(plan *InstallPlan) error {
	return defaultCheckTarget(plan)
}
