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

// Android SDK is a composite install: platform-tools plus cmdline-tools,
// published under one product-owned sdk root. A failure in the second stage
// keeps the first stage in place and surfaces a partial task state that can
// be retried; the retry reuses matching pieces instead of re-downloading.
type androidSDKProvider struct{}

func (androidSDKProvider) option() InstallOption {
	return InstallOption{
		Tool:               "androidsdk",
		ProviderID:         "google-android-sdk",
		Mirrors:            []MirrorOption{{ID: "official", Label: "Google 官方源"}, {ID: "tencent", Label: "腾讯云镜像"}},
		VersionSelectors:   []string{"latest"},
		SupportedPlatforms: []string{"windows", "linux"},
		SupportedArchs:     []string{"amd64"},
	}
}

var cmdlineToolsNameRe = regexp.MustCompile(`^commandlinetools-(?:win|linux)-([0-9]+)_latest\.zip$`)

func androidBase(mirror string) (string, error) {
	switch mirrorID(mirror) {
	case "", "official":
		return "https://dl.google.com/android/repository", nil
	case "tencent":
		return "https://mirrors.cloud.tencent.com/AndroidSDK", nil
	}
	return "", errors.New("RUNTIME_PROVIDER_UNSUPPORTED: unknown android sdk mirror")
}

func androidPlatform() (platform string, err error) {
	switch runtime.GOOS {
	case "windows":
		return "windows", nil
	case "linux":
		return "linux", nil
	}
	return "", errors.New("RUNTIME_PROVIDER_UNSUPPORTED: android sdk has no verified package for this OS")
}

type manifestArchive struct {
	URL      string
	Checksum string
	Size     int64
}

// manifestArchives extracts <archive> entries (url/checksum/size) for one
// remotePackage path from an Android repository manifest.
func manifestArchives(manifest []byte, packagePath string) ([]manifestArchive, error) {
	blockRe := regexp.MustCompile(`(?s)<remotePackage[^>]*path="` + regexp.QuoteMeta(packagePath) + `".*?</remotePackage>`)
	block := blockRe.Find(manifest)
	if block == nil {
		return nil, fmt.Errorf("manifest has no package %q", packagePath)
	}
	archiveRe := regexp.MustCompile(`(?s)<archive>(.*?)</archive>`)
	urlRe := regexp.MustCompile(`<url>([^<]+)</url>`)
	checksumRe := regexp.MustCompile(`<checksum>([^<]+)</checksum>`)
	sizeRe := regexp.MustCompile(`<size>([0-9]+)</size>`)
	entries := []manifestArchive{}
	for _, match := range archiveRe.FindAllSubmatch(block, -1) {
		body := match[1]
		entry := manifestArchive{}
		if m := urlRe.FindSubmatch(body); m != nil {
			entry.URL = string(m[1])
		}
		if m := checksumRe.FindSubmatch(body); m != nil {
			entry.Checksum = strings.ToLower(strings.TrimSpace(string(m[1])))
		}
		if m := sizeRe.FindSubmatch(body); m != nil {
			fmt.Sscanf(string(m[1]), "%d", &entry.Size)
		}
		if entry.URL != "" {
			entries = append(entries, entry)
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("manifest package %q has no archive entries", packagePath)
	}
	return entries, nil
}

// manifestChecksum finds the checksum for one archive file. Only sha256 and
// sha512 are acceptable; a sha1 manifest entry means no verifiable digest.
func manifestChecksum(entries []manifestArchive, fileName string) (string, int64, error) {
	for _, entry := range entries {
		if entry.URL != fileName {
			continue
		}
		switch len(entry.Checksum) {
		case 64, 128:
			if isHex(entry.Checksum) {
				algo := "sha256"
				if len(entry.Checksum) == 128 {
					algo = "sha512"
				}
				return algo, entry.Size, nil
			}
		}
		return "", 0, errors.New("RUNTIME_CHECKSUM_MISSING: manifest digest is not sha256/sha512")
	}
	return "", 0, errors.New("RUNTIME_CHECKSUM_MISSING: manifest has no digest for " + fileName)
}

func (p androidSDKProvider) preview(ctx context.Context, f *fetcher, selector, mirror string) (*InstallPlan, error) {
	platform, err := androidPlatform()
	if err != nil {
		return nil, err
	}
	if runtime.GOARCH != "amd64" {
		return nil, errors.New("RUNTIME_PROVIDER_UNSUPPORTED: android sdk provider evidence covers x64 only")
	}
	if normalizeSelector(selector) != "latest" {
		return nil, errors.New("RUNTIME_VERSION_UNAVAILABLE: android sdk only offers the latest channel")
	}
	base, err := androidBase(mirror)
	if err != nil {
		return nil, err
	}
	manifest, err := f.getText(ctx, base+"/repository2-3.xml")
	if err != nil {
		return nil, fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: %w", err)
	}
	ptName := "platform-tools-latest-" + platform + ".zip"
	ptEntries, err := manifestArchives([]byte(manifest), "platform-tools")
	if err != nil {
		return nil, fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: %w", err)
	}
	ptAlgo, _, err := manifestChecksum(ptEntries, ptName)
	if err != nil {
		return nil, err
	}
	ctEntries, err := manifestArchives([]byte(manifest), "cmdline-tools;latest")
	if err != nil {
		return nil, fmt.Errorf("RUNTIME_PROVIDER_UNAVAILABLE: %w", err)
	}
	ctName := ""
	for _, entry := range ctEntries {
		if m := cmdlineToolsNameRe.FindStringSubmatch(entry.URL); m != nil {
			ctName = entry.URL
			break
		}
	}
	if ctName == "" {
		return nil, errors.New("RUNTIME_VERSION_UNAVAILABLE: manifest has no cmdline-tools package for this platform")
	}
	ctAlgo, _, err := manifestChecksum(ctEntries, ctName)
	if err != nil {
		return nil, err
	}
	build := cmdlineToolsNameRe.FindStringSubmatch(ctName)[1]
	root, err := RuntimeRoot()
	if err != nil {
		return nil, err
	}
	srcs := []pkgSource{
		{Name: "platform-tools", FileName: ptName, URL: base + "/" + ptName, Algo: ptAlgo},
		{Name: "cmdline-tools", FileName: ctName, URL: base + "/" + ctName, Algo: ctAlgo},
	}
	plan, err := assemblePlan(ctx, f, "androidsdk", "google-android-sdk", mirror, "latest", build, platform, "amd64", filepath.Join(root, "android-sdk"), srcs)
	if err != nil {
		return nil, err
	}
	// The sdk root is a composite directory; its presence alone is not a
	// conflict (see checkTarget), so hide the misleading flag for previews.
	plan.TargetExists = false
	return plan, nil
}

func (androidSDKProvider) sources(plan *InstallPlan) ([]pkgSource, error) {
	return planSources(plan)
}

func androidComplete(sdkRoot string) bool {
	adb := filepath.Join(sdkRoot, "platform-tools", "adb")
	manager := filepath.Join(sdkRoot, "cmdline-tools", "latest", "bin", "sdkmanager")
	if runtime.GOOS == "windows" {
		adb += ".exe"
		manager += ".bat"
	}
	return regularFile(adb) && regularFile(manager)
}

func (p androidSDKProvider) install(ctx context.Context, f *fetcher, plan *InstallPlan, run *installRun) (userstate.Installation, error) {
	srcs, err := p.sources(plan)
	if err != nil {
		return userstate.Installation{}, err
	}
	if len(srcs) != 2 {
		return userstate.Installation{}, errors.New("RUNTIME_PLAN_STALE: android sdk plan needs two packages")
	}
	sdkRoot := plan.TargetHome
	if err := os.MkdirAll(sdkRoot, 0o700); err != nil {
		return userstate.Installation{}, err
	}
	adbName := "adb"
	if runtime.GOOS == "windows" {
		adbName = "adb.exe"
	}
	ptDir := filepath.Join(sdkRoot, "platform-tools")
	if regularFile(filepath.Join(ptDir, adbName)) {
		run.mark("platform-tools", "reused", nil)
	} else {
		run.mark("platform-tools", "started", nil)
		stage, rootDir, err := stagePackage(ctx, f, srcs[0], run)
		defer os.RemoveAll(stage)
		if err != nil {
			run.mark("platform-tools", "failed", taskError(err))
			return userstate.Installation{}, err
		}
		if err := moveInto(rootDir, ptDir); err != nil {
			run.mark("platform-tools", "failed", taskError(err))
			return userstate.Installation{}, err
		}
		if runtime.GOOS != "windows" {
			makeExecutable(filepath.Join(ptDir, "adb"))
			makeExecutable(filepath.Join(ptDir, "fastboot"))
		}
		run.mark("platform-tools", "done", nil)
	}

	managerName := "sdkmanager"
	if runtime.GOOS == "windows" {
		managerName = "sdkmanager.bat"
	}
	ctLatest := filepath.Join(sdkRoot, "cmdline-tools", "latest")
	if regularFile(filepath.Join(ctLatest, "bin", managerName)) {
		run.mark("cmdline-tools", "reused", nil)
	} else {
		run.mark("cmdline-tools", "started", nil)
		stage, rootDir, err := stagePackage(ctx, f, srcs[1], run)
		defer os.RemoveAll(stage)
		if err != nil {
			run.mark("cmdline-tools", "failed", taskError(err))
			return userstate.Installation{}, err
		}
		if err := os.MkdirAll(filepath.Join(sdkRoot, "cmdline-tools"), 0o700); err != nil {
			run.mark("cmdline-tools", "failed", taskError(err))
			return userstate.Installation{}, err
		}
		if err := moveInto(rootDir, ctLatest); err != nil {
			run.mark("cmdline-tools", "failed", taskError(err))
			return userstate.Installation{}, err
		}
		if runtime.GOOS != "windows" {
			makeExecutable(filepath.Join(ctLatest, "bin", "sdkmanager"))
			makeExecutable(filepath.Join(ctLatest, "bin", "avdmanager"))
		}
		run.mark("cmdline-tools", "done", nil)
	}
	return userstate.Installation{
		Runtime: "androidsdk", Version: plan.ResolvedVersion,
		LocalHome: sdkRoot, BinaryPath: filepath.Join(ptDir, adbName),
		Source: "downloaded", Label: "Android SDK (platform-tools + cmdline-tools " + plan.ResolvedVersion + ")", Vendor: "Google", Available: true,
	}, nil
}

// checkTarget tolerates a partial product-owned sdk root so retries can
// resume, but rejects a complete installation.
func (androidSDKProvider) checkTarget(plan *InstallPlan) error {
	if _, err := os.Lstat(plan.TargetHome); err == nil {
		if androidComplete(plan.TargetHome) {
			return errors.New("RUNTIME_TARGET_CONFLICT: android sdk root already installed")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
