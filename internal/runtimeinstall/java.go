package runtimeinstall

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"dev-toolbox/internal/userstate"
)

type javaProvider struct{}

type adoptiumReleases struct {
	MostRecentFeatureRelease int `json:"most_recent_feature_release"`
}

type adoptiumAsset struct {
	Version struct {
		Semver string `json:"semver"`
	} `json:"version"`
	Binary struct {
		Package struct {
			Name     string `json:"name"`
			Link     string `json:"link"`
			Checksum string `json:"checksum"`
			Size     int64  `json:"size"`
		} `json:"package"`
	} `json:"binary"`
}

func (javaProvider) option() InstallOption {
	return InstallOption{
		Tool:               "java",
		ProviderID:         "adoptium-temurin",
		Mirrors:            []MirrorOption{{ID: "official", Label: "Adoptium 官方源"}, {ID: "tuna", Label: "清华 TUNA 镜像"}},
		VersionSelectors:   []string{"latest", "major"},
		SupportedPlatforms: []string{"windows", "linux"},
		SupportedArchs:     []string{"amd64", "arm64"},
	}
}

func javaPlatform() (osName, apiArch, ext string, err error) {
	switch runtime.GOOS {
	case "windows":
		osName = "windows"
	case "linux":
		osName = "linux"
	default:
		return "", "", "", errors.New("RUNTIME_PROVIDER_UNSUPPORTED: temurin has no verified package for this OS")
	}
	switch runtime.GOARCH {
	case "amd64":
		apiArch = "x64"
	case "arm64":
		apiArch = "aarch64"
	default:
		return "", "", "", errors.New("RUNTIME_PROVIDER_UNSUPPORTED: temurin has no verified package for this architecture")
	}
	ext = "tar.gz"
	if osName == "windows" {
		ext = "zip"
	}
	return osName, apiArch, ext, nil
}

func javaAsset(ctx context.Context, f *fetcher, major int) (*adoptiumAsset, error) {
	osName, apiArch, _, err := javaPlatform()
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://api.adoptium.net/v3/assets/latest/%d/hotspot?os=%s&architecture=%s&image_type=jdk&vendor=eclipse", major, osName, apiArch)
	var assets []adoptiumAsset
	if err := f.getJSON(ctx, url, &assets); err != nil {
		return nil, fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: %w", err)
	}
	if len(assets) == 0 {
		return nil, fmt.Errorf("RUNTIME_VERSION_UNAVAILABLE: no Temurin JDK %d build for %s/%s", major, osName, apiArch)
	}
	return &assets[0], nil
}

func (p javaProvider) preview(ctx context.Context, f *fetcher, selector, mirror string) (*InstallPlan, error) {
	osName, apiArch, _, err := javaPlatform()
	if err != nil {
		return nil, err
	}
	selector = normalizeSelector(selector)
	major := 0
	if selector == "latest" {
		var releases adoptiumReleases
		if err := f.getJSON(ctx, "https://api.adoptium.net/v3/info/available_releases", &releases); err != nil {
			return nil, fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: %w", err)
		}
		if releases.MostRecentFeatureRelease == 0 {
			return nil, errors.New("RUNTIME_VERSION_UNAVAILABLE: provider did not report a latest JDK release")
		}
		major = releases.MostRecentFeatureRelease
		selector = fmt.Sprintf("%d", major)
	} else {
		if !numericSelectorRe.MatchString(selector) {
			return nil, fmt.Errorf("RUNTIME_VERSION_UNAVAILABLE: invalid major selector %q", selector)
		}
		major, err = parsePositiveInt(selector)
		if err != nil {
			return nil, err
		}
	}
	asset, err := javaAsset(ctx, f, major)
	if err != nil {
		return nil, err
	}
	pkg := asset.Binary.Package
	if pkg.Name == "" || pkg.Link == "" {
		return nil, errors.New("RUNTIME_PROVIDER_UNAVAILABLE: adoptium asset misses package metadata")
	}
	checksum := strings.ToLower(strings.TrimSpace(pkg.Checksum))
	if len(checksum) != 64 || !isHex(checksum) {
		return nil, errors.New("RUNTIME_CHECKSUM_MISSING: temurin package has no usable sha256")
	}
	url := pkg.Link
	if mirrorID(mirror) == "tuna" {
		url = fmt.Sprintf("https://mirrors.tuna.tsinghua.edu.cn/Adoptium/%d/jdk/%s/%s/%s", major, apiArch, osName, pkg.Name)
	} else if mirrorID(mirror) != "official" {
		return nil, errors.New("RUNTIME_PROVIDER_UNSUPPORTED: unknown java mirror")
	}
	root, err := RuntimeRoot()
	if err != nil {
		return nil, err
	}
	srcs := []pkgSource{{Name: "jdk", FileName: pkg.Name, URL: url, Algo: "sha256", Sum: checksum}}
	version := asset.Version.Semver
	if version == "" {
		version = selector
	}
	return assemblePlan(ctx, f, "java", "adoptium-temurin", mirror, selector, version, osName, apiArch, filepath.Join(root, "jdk-"+version), srcs)
}

func (javaProvider) sources(plan *InstallPlan) ([]pkgSource, error) {
	return planSources(plan)
}

func (javaProvider) install(ctx context.Context, f *fetcher, plan *InstallPlan, run *installRun) (userstate.Installation, error) {
	srcs, err := javaProvider{}.sources(plan)
	if err != nil {
		return userstate.Installation{}, err
	}
	stage, rootDir, err := stagePackage(ctx, f, srcs[0], run)
	defer os.RemoveAll(stage)
	if err != nil {
		return userstate.Installation{}, err
	}
	binary := filepath.Join("bin", "java")
	if plan.OS == "windows" {
		binary = filepath.Join("bin", "java.exe")
	}
	if !regularFile(filepath.Join(rootDir, binary)) {
		return userstate.Installation{}, errors.New("RUNTIME_INSTALL_LAYOUT_INVALID: java binary missing after extraction")
	}
	makeExecutable(filepath.Join(rootDir, binary))
	if err := moveInto(rootDir, plan.TargetHome); err != nil {
		return userstate.Installation{}, err
	}
	run.mark("jdk", "done", nil)
	return userstate.Installation{
		Runtime: "java", Version: plan.ResolvedVersion, Major: majorOf(plan.ResolvedVersion),
		LocalHome: plan.TargetHome, BinaryPath: filepath.Join(plan.TargetHome, binary),
		Source: "downloaded", Label: "Temurin JDK " + plan.ResolvedVersion, Vendor: "Eclipse Adoptium", Available: true,
	}, nil
}

func (javaProvider) checkTarget(plan *InstallPlan) error {
	return defaultCheckTarget(plan)
}
