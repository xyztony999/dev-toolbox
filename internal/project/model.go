package project

import "time"

type ScanStatus string

const (
	ScanComplete  ScanStatus = "complete"
	ScanPartial   ScanStatus = "partial"
	ScanCancelled ScanStatus = "cancelled"
)

type RequirementStatus string

const (
	RequirementResolved    RequirementStatus = "resolved"
	RequirementUnknown     RequirementStatus = "unknown"
	RequirementConflict    RequirementStatus = "conflict"
	RequirementUnsupported RequirementStatus = "unsupported"
)

type Evidence struct {
	SourcePath string    `json:"sourcePath"`
	Locator    string    `json:"locator"`
	RawValue   string    `json:"rawValue"`
	ParserID   string    `json:"parserId"`
	ObservedAt time.Time `json:"observedAt"`
}

type RuntimeRequirement struct {
	Runtime              string            `json:"runtime"`
	RawConstraint        string            `json:"rawConstraint,omitempty"`
	NormalizedConstraint string            `json:"normalizedConstraint,omitempty"`
	SourcePath           string            `json:"sourcePath,omitempty"`
	Confidence           string            `json:"confidence"`
	Status               RequirementStatus `json:"status"`
	Evidence             []Evidence        `json:"evidence,omitempty"`
}

type Repository struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	Source     string `json:"source"`
	Branch     string `json:"branch,omitempty"`
	RemoteName string `json:"remoteName,omitempty"`
	RemoteURL  string `json:"remoteUrl,omitempty"`
	UserName   string `json:"userName,omitempty"`
	UserEmail  string `json:"userEmail,omitempty"`
}

type Module struct {
	ID           string               `json:"id"`
	RelativePath string               `json:"relativePath"`
	BuildSystems []string             `json:"buildSystems"`
	Requirements []RuntimeRequirement `json:"requirements"`
}

type ScanIssue struct {
	Code         string `json:"code"`
	Severity     string `json:"severity"`
	RelativePath string `json:"relativePath,omitempty"`
	Message      string `json:"message"`
	Recoverable  bool   `json:"recoverable"`
}

type Candidate struct {
	ID             string       `json:"id"`
	DisplayName    string       `json:"displayName"`
	RootPath       string       `json:"rootPath"`
	EvidenceDigest string       `json:"evidenceDigest"`
	Repositories   []Repository `json:"repositories"`
	Modules        []Module     `json:"modules"`
	Issues         []ScanIssue  `json:"issues"`
	Status         ScanStatus   `json:"status"`
	ScannedAt      time.Time    `json:"scannedAt"`
}

type Project struct {
	ID           string       `json:"id"`
	DisplayName  string       `json:"displayName"`
	RootPath     string       `json:"rootPath"`
	Repositories []Repository `json:"repositories"`
	Modules      []Module     `json:"modules"`
	Issues       []ScanIssue  `json:"issues,omitempty"`
	Status       ScanStatus   `json:"scanStatus"`
	CreatedAt    time.Time    `json:"createdAt"`
	LastScanAt   time.Time    `json:"lastScanAt"`
	Revision     uint64       `json:"revision"`
}

type store struct {
	Version  int       `json:"version"`
	Projects []Project `json:"projects"`
}
