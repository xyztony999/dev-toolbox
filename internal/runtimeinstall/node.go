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

type nodeProvider struct{}

type nodeIndexEntry struct {
	Version string `json:"version"`
}

func (nodeProvider) option() InstallOption {
	return InstallOption{
		Tool:               "node",
		ProviderID:         "nodejs-distribution",
		Mirrors:            []MirrorOption{{ID: "official", Label: "Node.js 官方源"}, {ID: "npmmirror", Label: "npmmirror 镜像"}, {ID: "tuna", Label: "清华 TUNA 镜像"}},
		VersionSelectors:   []string{"latest", "major"},
		SupportedPlatforms: []string{"windows", "linux"},
		SupportedArchs:     []string{"amd64", "arm64"},
	}
}

func nodeBase(mirror string) (string, error) {
	switch mirrorID(mirror) {
	case "official":
		return "https://nodejs.org/dist", nil
	case "npmmirror":
		return "https://registry.npmmirror.com/-/binary/node", nil
	case "tuna":
		return "https://mirrors.tuna.tsinghua.edu.cn/nodejs-release", nil
	}
	return "", errors.New("RUNTIME_PROVIDER_UNSUPPORTED: unknown node mirror")
}

func nodePlatform() (osName, archName, ext string, err error) {
	switch runtime.GOOS {
	case "windows":
		osName = "win"
	case "linux":
		osName = "linux"
	default:
		return "", "", "", errors.New("RUNTIME_PROVIDER_UNSUPPORTED: node has no verified package for this OS")
	}
	switch runtime.GOARCH {
	case "amd64":
		archName = "x64"
	case "arm64":
		archName = "arm64"
	default:
		return "", "", "", errors.New("RUNTIME_PROVIDER_UNSUPPORTED: node has no verified package for this architecture")
	}
	ext = "tar.xz"
	if osName == "win" {
		ext = "zip"
	}
	return osName, archName, ext, nil
}

// resolveNodeVersion picks the newest non-prerelease version, optionally
// inside one major series.
func resolveNodeVersion(entries []nodeIndexEntry, selector string) (string, error) {
	prefix := ""
	if selector != "latest" {
		prefix = "v" + selector + "."
	}
	for _, entry := range entries {
		if strings.Contains(entry.Version, "-") {
			continue
		}
		if prefix != "" && !strings.HasPrefix(entry.Version, prefix) {
			continue
		}
		return strings.TrimPrefix(entry.Version, "v"), nil
	}
	return "", errors.New("RUNTIME_VERSION_UNAVAILABLE: no matching node release on the selected mirror")
}

func nodeChecksum(ctx context.Context, f *fetcher, base, version, fileName string) (string, error) {
	text, err := f.getText(ctx, base+"/v"+version+"/SHASUMS256.txt")
	if err != nil {
		return "", fmt.Errorf("RUNTIME_CHECKSUM_MISSING: node SHASUMS256.txt unavailable: %w", err)
	}
	for _, line := range strings.Split(text, "\n") {
		fields := strings.Fields(strings.TrimPrefix(strings.TrimSpace(line), "*"))
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == fileName {
			checksum := strings.ToLower(fields[0])
			if len(checksum) == 64 && isHex(checksum) {
				return checksum, nil
			}
		}
	}
	return "", errors.New("RUNTIME_CHECKSUM_MISSING: node archive has no SHASUMS256 entry")
}

func (p nodeProvider) preview(ctx context.Context, f *fetcher, selector, mirror string) (*InstallPlan, error) {
	osName, archName, ext, err := nodePlatform()
	if err != nil {
		return nil, err
	}
	base, err := nodeBase(mirror)
	if err != nil {
		return nil, err
	}
	selector, err = majorSelector(normalizeSelector(selector), true)
	if err != nil {
		return nil, err
	}
	var entries []nodeIndexEntry
	if err := f.getJSON(ctx, base+"/index.json", &entries); err != nil {
		return nil, fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: %w", err)
	}
	version, err := resolveNodeVersion(entries, selector)
	if err != nil {
		return nil, err
	}
	fileName := fmt.Sprintf("node-v%s-%s-%s.%s", version, osName, archName, ext)
	checksum, err := nodeChecksum(ctx, f, base, version, fileName)
	if err != nil {
		return nil, err
	}
	root, err := RuntimeRoot()
	if err != nil {
		return nil, err
	}
	srcs := []pkgSource{{Name: "node", FileName: fileName, URL: base + "/v" + version + "/" + fileName, Algo: "sha256", Sum: checksum}}
	return assemblePlan(ctx, f, "node", "nodejs-distribution", mirror, selector, version, osName, archName, filepath.Join(root, "node-"+version), srcs)
}

func (nodeProvider) sources(plan *InstallPlan) ([]pkgSource, error) {
	return planSources(plan)
}

func (nodeProvider) install(ctx context.Context, f *fetcher, plan *InstallPlan, run *installRun) (userstate.Installation, error) {
	srcs, err := nodeProvider{}.sources(plan)
	if err != nil {
		return userstate.Installation{}, err
	}
	stage, rootDir, err := stagePackage(ctx, f, srcs[0], run)
	defer os.RemoveAll(stage)
	if err != nil {
		return userstate.Installation{}, err
	}
	binary := "node"
	if runtime.GOOS == "windows" {
		binary = "node.exe"
	}
	binaryPath := filepath.Join(rootDir, binary)
	if !regularFile(binaryPath) {
		return userstate.Installation{}, errors.New("RUNTIME_INSTALL_LAYOUT_INVALID: node binary missing after extraction")
	}
	makeExecutable(binaryPath)
	if err := moveInto(rootDir, plan.TargetHome); err != nil {
		return userstate.Installation{}, err
	}
	run.mark("node", "done", nil)
	return userstate.Installation{
		Runtime: "node", Version: plan.ResolvedVersion, Major: majorOf(plan.ResolvedVersion),
		LocalHome: plan.TargetHome, BinaryPath: filepath.Join(plan.TargetHome, binary),
		Source: "downloaded", Label: "Node.js " + plan.ResolvedVersion, Vendor: "OpenJS", Available: true,
	}, nil
}

func (nodeProvider) checkTarget(plan *InstallPlan) error {
	return defaultCheckTarget(plan)
}
