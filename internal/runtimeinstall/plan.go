package runtimeinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

const planTTL = 10 * time.Minute

// Task lifecycle states, aligned with the gated design
// (confirmed → downloading → verifying → extracting → registering → complete,
// plus failed / cancelled / partial branches).
const (
	StateQueued      = "confirmed"
	StateDownloading = "downloading"
	StateVerifying   = "verifying"
	StateExtracting  = "extracting"
	StateRegistering = "registering"
	StateComplete    = "complete"
	StateFailed      = "failed"
	StateCancelled   = "cancelled"
	StatePartial     = "partial"
)

func isTaskPhase(state string) bool {
	switch state {
	case StateQueued, StateDownloading, StateVerifying, StateExtracting, StateRegistering,
		StateComplete, StateFailed, StateCancelled, StatePartial:
		return true
	}
	return false
}

type MirrorOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// InstallOption is the static, read-only capability description returned by
// runtime.install.options.
type InstallOption struct {
	Tool               string         `json:"tool"`
	ProviderID         string         `json:"providerId"`
	Mirrors            []MirrorOption `json:"mirrors"`
	VersionSelectors   []string       `json:"versionSelectors"`
	SupportedPlatforms []string       `json:"supportedPlatforms"`
	SupportedArchs     []string       `json:"supportedArchs"`
	Unsupported        string         `json:"unsupported,omitempty"`
}

type InstallPackage struct {
	Name              string `json:"name"`
	FileName          string `json:"fileName"`
	ChecksumAlgorithm string `json:"checksumAlgorithm"`
	Checksum          string `json:"checksum"`
	EstimatedBytes    int64  `json:"estimatedBytes,omitempty"`
}

// InstallPlan is what a preview returns and a start confirms. Provider URLs
// are intentionally absent: they are rebuilt inside provider adapters and
// never cross the bridge.
type InstallPlan struct {
	PlanID            string           `json:"planId"`
	PlanDigest        string           `json:"planDigest"`
	Tool              string           `json:"tool"`
	ProviderID        string           `json:"providerId"`
	Mirror            string           `json:"mirror"`
	RequestedSelector string           `json:"requestedSelector"`
	ResolvedVersion   string           `json:"resolvedVersion"`
	OS                string           `json:"os"`
	Arch              string           `json:"arch"`
	Packages          []InstallPackage `json:"packages"`
	TargetHome        string           `json:"targetHome"`
	EstimatedBytes    int64            `json:"estimatedBytes,omitempty"`
	TargetExists      bool             `json:"targetExists"`
	ExpiresAt         time.Time        `json:"expiresAt"`

	// urls are the provider-resolved download locations, parallel to
	// Packages. Unexported so they never cross the CLI/WebView bridge;
	// they are still covered indirectly by the plan digest via mirror and
	// resolved version.
	urls []string
}

// computePlanDigest covers every resolved result and target field; callers
// must echo it at start so a stale or tampered plan is rejected.
func computePlanDigest(plan InstallPlan) string {
	payload := struct {
		Tool, ProviderID, Mirror, RequestedSelector, ResolvedVersion, OS, Arch string
		Packages                                                               []InstallPackage
		TargetHome                                                             string
		ExpiresAtNano                                                          int64
	}{
		Tool: plan.Tool, ProviderID: plan.ProviderID, Mirror: plan.Mirror,
		RequestedSelector: plan.RequestedSelector, ResolvedVersion: plan.ResolvedVersion,
		OS: plan.OS, Arch: plan.Arch, Packages: plan.Packages, TargetHome: plan.TargetHome,
		ExpiresAtNano: plan.ExpiresAt.UnixNano(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

type TaskError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type InstallProgress struct {
	Phase         string `json:"phase"`
	Package       string `json:"package,omitempty"`
	ReceivedBytes int64  `json:"receivedBytes,omitempty"`
	TotalBytes    int64  `json:"totalBytes,omitempty"`
}

// PackageResult reports per-package outcome; Android depends on it to show
// platform-tools vs cmdline-tools separately.
type PackageResult struct {
	Name     string     `json:"name"`
	FileName string     `json:"fileName"`
	State    string     `json:"state"` // done | reused | failed
	Error    *TaskError `json:"error,omitempty"`
}

type InstallTask struct {
	TaskID         string          `json:"taskId"`
	PlanID         string          `json:"planId"`
	PlanDigest     string          `json:"planDigest"`
	State          string          `json:"state"`
	Progress       InstallProgress `json:"progress"`
	Packages       []PackageResult `json:"packages"`
	InstallationID string          `json:"installationId,omitempty"`
	Error          *TaskError      `json:"error,omitempty"`
}

// Event is the host-push payload; sequence is monotonically increasing so
// consumers can drop stale events.
type Event struct {
	Sequence uint64      `json:"sequence"`
	TaskID   string      `json:"taskId"`
	Task     InstallTask `json:"task"`
}

// pkgSource is the internal download descriptor; it never leaves the service.
type pkgSource struct {
	Name     string
	FileName string
	URL      string
	Algo     string
	Sum      string
}

func taskError(err error) *TaskError {
	if err == nil {
		return nil
	}
	code := "RUNTIME_INSTALL_FAILED"
	for _, field := range []string{
		"RUNTIME_PROVIDER_UNSUPPORTED", "RUNTIME_VERSION_UNAVAILABLE", "RUNTIME_PLAN_EXPIRED",
		"RUNTIME_PLAN_STALE", "RUNTIME_REVISION_CONFLICT", "RUNTIME_TARGET_CONFLICT",
		"RUNTIME_CHECKSUM_MISSING", "RUNTIME_CHECKSUM_MISMATCH", "RUNTIME_ARCHIVE_UNSAFE",
		"RUNTIME_DOWNLOAD_FAILED", "RUNTIME_INSTALL_CANCELLED", "RUNTIME_CONFIRMATION_REQUIRED",
		"RUNTIME_DISK_SPACE_INSUFFICIENT", "RUNTIME_PROVIDER_UNAVAILABLE", "RUNTIME_PROVIDER_HOST_REJECTED",
		"RUNTIME_PROVIDER_ADDRESS_REJECTED", "RUNTIME_PROVIDER_REDIRECT_REJECTED", "RUNTIME_PROVIDER_URL_REJECTED",
		"RUNTIME_INSTALL_LAYOUT_INVALID", "RUNTIME_INSTALL_CANCELLED",
	} {
		if len(err.Error()) >= len(field) && err.Error()[:len(field)] == field {
			code = field
			break
		}
	}
	return &TaskError{Code: code, Message: err.Error()}
}
