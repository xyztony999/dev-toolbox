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

// Python from python-build-standalone via the npmmirror binary mirror,
// matching the DevSwitch provider behavior (install_only archives, complete
// CPython with pip). The source provider evidence covers x64 only.
type pythonProvider struct{}

type pbsEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

var pbsVersionRe = regexp.MustCompile(`^cpython-(\d+\.\d+(?:\.\d+)?)\+(\d{8})-`)
var pbsDateDirRe = regexp.MustCompile(`^(\d{8})/$`)

func (pythonProvider) option() InstallOption {
	return InstallOption{
		Tool:               "python",
		ProviderID:         "python-build-standalone",
		Mirrors:            []MirrorOption{{ID: "official", Label: "npmmirror python-build-standalone 镜像"}},
		VersionSelectors:   []string{"latest", "prefix"},
		SupportedPlatforms: []string{"windows", "linux"},
		SupportedArchs:     []string{"amd64"},
	}
}

func pbsPlatform() (platform, ext string, err error) {
	switch runtime.GOOS {
	case "windows":
		return "x86_64-pc-windows-msvc", "tar.gz", nil
	case "linux":
		return "x86_64-unknown-linux-gnu", "tar.gz", nil
	}
	return "", "", errors.New("RUNTIME_PROVIDER_UNSUPPORTED: python provider evidence covers windows/linux only")
}

type pbsMatch struct {
	version, date, fileName string
}

// findPbsBuild walks the two-level mirror listing (date directories) from
// newest to oldest, at most six directories deep, honoring the selector.
func findPbsBuild(ctx context.Context, f *fetcher, base, platform, selector string) (*pbsMatch, error) {
	var rootEntries []pbsEntry
	if err := f.getJSON(ctx, base+"/", &rootEntries); err != nil {
		return nil, fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: %w", err)
	}
	dates := []string{}
	for _, entry := range rootEntries {
		if entry.Type != "dir" {
			continue
		}
		if m := pbsDateDirRe.FindStringSubmatch(entry.Name); m != nil {
			dates = append(dates, m[1])
		}
	}
	suffix := platform + "-install_only.tar.gz"
	attempts := 0
	for i := len(dates) - 1; i >= 0 && attempts < 6; i-- {
		attempts++
		var entries []pbsEntry
		if err := f.getJSON(ctx, base+"/"+dates[i]+"/", &entries); err != nil {
			continue
		}
		for _, entry := range entries {
			if strings.Contains(entry.Name, "freethreaded") || !strings.HasSuffix(entry.Name, suffix) {
				continue
			}
			match := pbsVersionRe.FindStringSubmatch(entry.Name)
			if match == nil {
				continue
			}
			if selector != "latest" && !prefixMatch(match[1], selector) {
				continue
			}
			return &pbsMatch{version: match[1], date: dates[i], fileName: entry.Name}, nil
		}
		if selector == "latest" {
			break
		}
	}
	return nil, errors.New("RUNTIME_VERSION_UNAVAILABLE: no matching python-build-standalone archive")
}

func (pythonProvider) preview(ctx context.Context, f *fetcher, selector, mirror string) (*InstallPlan, error) {
	if mirrorID(mirror) != "" && mirrorID(mirror) != "official" {
		return nil, errors.New("RUNTIME_PROVIDER_UNSUPPORTED: unknown python mirror")
	}
	if runtime.GOARCH != "amd64" {
		return nil, errors.New("RUNTIME_PROVIDER_UNSUPPORTED: python provider evidence covers x64 only")
	}
	osName := "linux"
	if runtime.GOOS == "windows" {
		osName = "windows"
	}
	platform, _, err := pbsPlatform()
	if err != nil {
		return nil, err
	}
	selector, err = prefixSelector(normalizeSelector(selector))
	if err != nil {
		return nil, err
	}
	base := "https://registry.npmmirror.com/-/binary/python-build-standalone"
	match, err := findPbsBuild(ctx, f, base, platform, selector)
	if err != nil {
		return nil, err
	}
	archiveURL := base + "/" + match.date + "/" + match.fileName
	algo, sum, err := sidecarChecksum(ctx, f, archiveURL)
	if err != nil {
		return nil, err
	}
	root, err := RuntimeRoot()
	if err != nil {
		return nil, err
	}
	srcs := []pkgSource{{Name: "python", FileName: match.fileName, URL: archiveURL, Algo: algo, Sum: sum}}
	return assemblePlan(ctx, f, "python", "python-build-standalone", mirror, selector, match.version, osName, "amd64", filepath.Join(root, "python-"+match.version), srcs)
}

func (pythonProvider) sources(plan *InstallPlan) ([]pkgSource, error) {
	return planSources(plan)
}

func (pythonProvider) install(ctx context.Context, f *fetcher, plan *InstallPlan, run *installRun) (userstate.Installation, error) {
	srcs, err := pythonProvider{}.sources(plan)
	if err != nil {
		return userstate.Installation{}, err
	}
	stage, rootDir, err := stagePackage(ctx, f, srcs[0], run)
	defer os.RemoveAll(stage)
	if err != nil {
		return userstate.Installation{}, err
	}
	binary := filepath.Join("bin", "python3")
	labelBin := "python3"
	if plan.OS == "windows" {
		binary = "python.exe"
		labelBin = "python.exe"
	}
	if !regularFile(filepath.Join(rootDir, binary)) {
		return userstate.Installation{}, errors.New("RUNTIME_INSTALL_LAYOUT_INVALID: python binary missing after extraction")
	}
	makeExecutable(filepath.Join(rootDir, binary))
	if err := moveInto(rootDir, plan.TargetHome); err != nil {
		return userstate.Installation{}, err
	}
	run.mark("python", "done", nil)
	return userstate.Installation{
		Runtime: "python", Version: plan.ResolvedVersion, Major: majorOf(plan.ResolvedVersion),
		LocalHome: plan.TargetHome, BinaryPath: filepath.Join(plan.TargetHome, labelBin),
		Source: "downloaded", Label: "CPython " + plan.ResolvedVersion + " (python-build-standalone)", Vendor: "astral-sh/python-build-standalone", Available: true,
	}, nil
}

func (pythonProvider) checkTarget(plan *InstallPlan) error {
	return defaultCheckTarget(plan)
}
