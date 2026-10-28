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
	"time"

	"dev-toolbox/internal/userstate"
)

// provider is one fixed distribution source. preview may query metadata but
// must not download archives or write to the install root; install performs
// download → verify → extract → atomic publish and registers nothing itself.
type provider interface {
	option() InstallOption
	preview(ctx context.Context, f *fetcher, selector, mirror string) (*InstallPlan, error)
	sources(plan *InstallPlan) ([]pkgSource, error)
	install(ctx context.Context, f *fetcher, plan *InstallPlan, run *installRun) (userstate.Installation, error)
	checkTarget(plan *InstallPlan) error
}

// installRun lets a provider report progress and per-package results while
// the task manager owns the task state.
type installRun struct {
	report func(InstallProgress)
	mark   func(packageName, state string, terr *TaskError)
}

var providers = map[string]provider{}

func init() {
	for _, p := range []provider{
		nodeProvider{}, javaProvider{}, mavenProvider{}, gradleProvider{},
		androidSDKProvider{}, golangProvider{}, pythonProvider{},
	} {
		providers[p.option().Tool] = p
	}
}

func installOptions(tool string) ([]InstallOption, error) {
	if tool == "" {
		options := make([]InstallOption, 0, len(providers))
		for _, name := range []string{"node", "java", "maven", "gradle", "androidsdk", "golang", "python"} {
			if p, ok := providers[name]; ok {
				options = append(options, withHostSupport(p.option()))
			}
		}
		return options, nil
	}
	p, ok := providers[tool]
	if !ok {
		return nil, fmt.Errorf("RUNTIME_PROVIDER_UNSUPPORTED: unknown tool %q", tool)
	}
	return []InstallOption{withHostSupport(p.option())}, nil
}

// withHostSupport marks the option unsupported when this host's OS/arch has
// no provider evidence, per the D0 capability table.
func withHostSupport(option InstallOption) InstallOption {
	osName, arch := runtime.GOOS, runtime.GOARCH
	platformOK, archOK := false, false
	for _, name := range option.SupportedPlatforms {
		if name == osName {
			platformOK = true
		}
	}
	for _, name := range option.SupportedArchs {
		if name == arch {
			archOK = true
		}
	}
	if !platformOK || !archOK {
		option.Unsupported = fmt.Sprintf("provider 在本机 %s/%s 上没有已验证的发行包", osName, arch)
	}
	return option
}

// RuntimeRoot is the only install target root; Windows uses per-user
// %LOCALAPPDATA%, Linux follows XDG_DATA_HOME.
func RuntimeRoot() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return "", errors.New("RUNTIME_DATA_DIR_UNAVAILABLE")
		}
		return filepath.Join(base, "DevToolbox", "runtimes"), nil
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "dev-toolbox", "runtimes"), nil
}

var (
	numericSelectorRe = regexp.MustCompile(`^\d+$`)
	versionPrefixRe   = regexp.MustCompile(`^\d+(\.\d+)*$`)
)

func normalizeSelector(selector string) string {
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(selector), "v"))
}

// majorSelector validates "latest" or a bare major number ("22").
func majorSelector(selector string, latestAllowed bool) (string, error) {
	if selector == "latest" {
		if !latestAllowed {
			return "", errors.New("RUNTIME_VERSION_UNAVAILABLE: latest channel is not offered for this tool")
		}
		return "latest", nil
	}
	if !numericSelectorRe.MatchString(selector) {
		return "", fmt.Errorf("RUNTIME_VERSION_UNAVAILABLE: invalid major selector %q", selector)
	}
	return selector, nil
}

// prefixSelector validates "latest" or a version prefix ("1.23", "3.12.7").
func prefixSelector(selector string) (string, error) {
	if selector == "latest" {
		return "latest", nil
	}
	if !versionPrefixRe.MatchString(selector) {
		return "", fmt.Errorf("RUNTIME_VERSION_UNAVAILABLE: invalid version selector %q", selector)
	}
	return selector, nil
}

func prefixMatch(version, prefix string) bool {
	return version == prefix || strings.HasPrefix(version, prefix+".")
}

func majorOf(version string) string {
	return strings.SplitN(version, ".", 2)[0]
}

func parsePositiveInt(value string) (int, error) {
	n := 0
	for _, c := range value {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("RUNTIME_VERSION_UNAVAILABLE: invalid numeric selector %q", value)
		}
		n = n*10 + int(c-'0')
		if n > 1000 {
			return 0, fmt.Errorf("RUNTIME_VERSION_UNAVAILABLE: invalid numeric selector %q", value)
		}
	}
	if n == 0 {
		return 0, fmt.Errorf("RUNTIME_VERSION_UNAVAILABLE: invalid numeric selector %q", value)
	}
	return n, nil
}

// planSources pairs stored plan packages with the URLs resolved at preview
// time; a plan without them cannot be installed safely.
func planSources(plan *InstallPlan) ([]pkgSource, error) {
	if len(plan.urls) != len(plan.Packages) {
		return nil, errors.New("RUNTIME_PLAN_STALE: plan has no resolved download locations")
	}
	srcs := make([]pkgSource, 0, len(plan.Packages))
	for i, pkg := range plan.Packages {
		srcs = append(srcs, pkgSource{Name: pkg.Name, FileName: pkg.FileName, URL: plan.urls[i], Algo: pkg.ChecksumAlgorithm, Sum: pkg.Checksum})
	}
	return srcs, nil
}

func defaultCheckTarget(plan *InstallPlan) error {
	if _, err := os.Lstat(plan.TargetHome); err == nil {
		return errors.New("RUNTIME_TARGET_CONFLICT")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// assemblePlan fills the shared plan fields, estimates package sizes with
// HEAD probes, and stamps the digest and id. It never writes to disk.
func assemblePlan(ctx context.Context, f *fetcher, tool, providerID, mirror, selector, version, osName, arch, targetHome string, srcs []pkgSource) (*InstallPlan, error) {
	plan := &InstallPlan{
		Tool: tool, ProviderID: providerID, Mirror: mirrorID(mirror), RequestedSelector: selector,
		ResolvedVersion: version, OS: osName, Arch: arch, TargetHome: targetHome,
		ExpiresAt: time.Now().Add(planTTL),
	}
	for _, src := range srcs {
		pkg := InstallPackage{Name: src.Name, FileName: src.FileName, ChecksumAlgorithm: src.Algo, Checksum: src.Sum}
		if size, ok := f.head(ctx, src.URL); ok {
			pkg.EstimatedBytes = size
			plan.EstimatedBytes += size
		}
		plan.Packages = append(plan.Packages, pkg)
		plan.urls = append(plan.urls, src.URL)
	}
	plan.PlanDigest = computePlanDigest(*plan)
	plan.PlanID = plan.PlanDigest[:24]
	return plan, nil
}

func mirrorID(mirror string) string {
	if strings.TrimSpace(mirror) == "" {
		return "official"
	}
	return strings.TrimSpace(mirror)
}

// stagePackage downloads one package into a fresh staging dir under the
// product runtime root, verifies its digest and extracts the single root.
// The caller owns the returned staging dir and must remove it.
func stagePackage(ctx context.Context, f *fetcher, src pkgSource, run *installRun) (stageDir, rootDir string, err error) {
	root, err := RuntimeRoot()
	if err != nil {
		return "", "", err
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return "", "", err
	}
	stageDir, err = os.MkdirTemp(root, ".install-*")
	if err != nil {
		return "", "", err
	}
	archive := filepath.Join(stageDir, src.FileName)
	run.report(InstallProgress{Phase: StateDownloading, Package: src.Name})
	if err := f.download(ctx, src.URL, archive, func(received, total int64) {
		run.report(InstallProgress{Phase: StateDownloading, Package: src.Name, ReceivedBytes: received, TotalBytes: total})
	}); err != nil {
		return stageDir, "", err
	}
	run.report(InstallProgress{Phase: StateVerifying, Package: src.Name})
	if err := verifyChecksum(archive, src.Sum, src.Algo); err != nil {
		return stageDir, "", err
	}
	run.report(InstallProgress{Phase: StateExtracting, Package: src.Name})
	rootDir, err = extractSingleRoot(archive, filepath.Join(stageDir, "extract"))
	if err != nil {
		return stageDir, "", err
	}
	return stageDir, rootDir, nil
}
