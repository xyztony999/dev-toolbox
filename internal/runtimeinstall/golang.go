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

// Go toolchain from the golang.google.cn dl manifest, whose entries carry
// the sha256 inline (the site serves HTML for .sha256 companions). The
// source provider evidence covers x64 only.
type golangProvider struct{}

type goRelease struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	Files   []struct {
		Filename string `json:"filename"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		Kind     string `json:"kind"`
		SHA256   string `json:"sha256"`
	} `json:"files"`
}

func (golangProvider) option() InstallOption {
	return InstallOption{
		Tool:               "golang",
		ProviderID:         "golang-google-cn",
		Mirrors:            []MirrorOption{{ID: "official", Label: "Go 官方中国站"}},
		VersionSelectors:   []string{"latest", "prefix"},
		SupportedPlatforms: []string{"windows", "linux"},
		SupportedArchs:     []string{"amd64"},
	}
}

func (golangProvider) preview(ctx context.Context, f *fetcher, selector, mirror string) (*InstallPlan, error) {
	if mirrorID(mirror) != "" && mirrorID(mirror) != "official" {
		return nil, errors.New("RUNTIME_PROVIDER_UNSUPPORTED: unknown go mirror")
	}
	osName := "linux"
	switch runtime.GOOS {
	case "windows":
		osName = "windows"
	case "linux":
	default:
		return nil, errors.New("RUNTIME_PROVIDER_UNSUPPORTED: go provider evidence covers windows/linux only")
	}
	if runtime.GOARCH != "amd64" {
		return nil, errors.New("RUNTIME_PROVIDER_UNSUPPORTED: go provider evidence covers x64 only")
	}
	selector, err := prefixSelector(normalizeSelector(selector))
	if err != nil {
		return nil, err
	}
	var releases []goRelease
	if err := f.getJSON(ctx, "https://golang.google.cn/dl/?mode=json&include=all", &releases); err != nil {
		return nil, fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: %w", err)
	}
	version, fileName, checksum := "", "", ""
	for _, release := range releases {
		if !release.Stable {
			continue
		}
		name := strings.TrimPrefix(release.Version, "go")
		if selector != "latest" && !prefixMatch(name, selector) {
			continue
		}
		for _, file := range release.Files {
			if file.OS != osName || file.Arch != "amd64" || strings.HasSuffix(file.Filename, ".msi") {
				continue
			}
			digest := strings.ToLower(strings.TrimSpace(file.SHA256))
			if len(digest) != 64 || !isHex(digest) {
				continue
			}
			version, fileName, checksum = name, file.Filename, digest
			break
		}
		if version != "" {
			break
		}
	}
	if version == "" {
		return nil, errors.New("RUNTIME_VERSION_UNAVAILABLE: no matching stable go release on the manifest")
	}
	root, err := RuntimeRoot()
	if err != nil {
		return nil, err
	}
	srcs := []pkgSource{{Name: "go", FileName: fileName, URL: "https://golang.google.cn/dl/" + fileName, Algo: "sha256", Sum: checksum}}
	return assemblePlan(ctx, f, "golang", "golang-google-cn", mirror, selector, version, osName, "amd64", filepath.Join(root, "go-"+version), srcs)
}

func (golangProvider) sources(plan *InstallPlan) ([]pkgSource, error) {
	return planSources(plan)
}

func (golangProvider) install(ctx context.Context, f *fetcher, plan *InstallPlan, run *installRun) (userstate.Installation, error) {
	srcs, err := golangProvider{}.sources(plan)
	if err != nil {
		return userstate.Installation{}, err
	}
	stage, rootDir, err := stagePackage(ctx, f, srcs[0], run)
	defer os.RemoveAll(stage)
	if err != nil {
		return userstate.Installation{}, err
	}
	binary := filepath.Join("bin", "go")
	if plan.OS == "windows" {
		binary = filepath.Join("bin", "go.exe")
	}
	if !regularFile(filepath.Join(rootDir, binary)) {
		return userstate.Installation{}, errors.New("RUNTIME_INSTALL_LAYOUT_INVALID: go binary missing after extraction")
	}
	makeExecutable(filepath.Join(rootDir, binary))
	if err := moveInto(rootDir, plan.TargetHome); err != nil {
		return userstate.Installation{}, err
	}
	run.mark("go", "done", nil)
	return userstate.Installation{
		Runtime: "golang", Version: plan.ResolvedVersion, Major: majorOf(plan.ResolvedVersion),
		LocalHome: plan.TargetHome, BinaryPath: filepath.Join(plan.TargetHome, binary),
		Source: "downloaded", Label: "Go " + plan.ResolvedVersion, Vendor: "Google Go", Available: true,
	}, nil
}

func (golangProvider) checkTarget(plan *InstallPlan) error {
	return defaultCheckTarget(plan)
}
