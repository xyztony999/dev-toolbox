package gitidentity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

// Key types / algorithms / expiries — codes match common-enums.md.
const (
	KeyTypeOpenPGP = "openpgp"
	KeyTypeSSH     = "ssh"

	AlgoED25519  = "ed25519"
	AlgoRSA4096  = "rsa4096"
	AlgoRSA3072  = "rsa3072"
	AlgoNISTP256 = "nistp256"
	AlgoECDSA256 = "ecdsa256"

	ExpiryNever = "never"
	Expiry1Y    = "1y"
	Expiry2Y    = "2y"
	Expiry5Y    = "5y"
)

var (
	openpgpAlgos = []string{AlgoED25519, AlgoRSA4096, AlgoRSA3072, AlgoNISTP256}
	sshAlgos     = []string{AlgoED25519, AlgoRSA4096, AlgoECDSA256}
	openpgpExp   = []string{ExpiryNever, Expiry1Y, Expiry2Y, Expiry5Y}
	sshExp       = []string{ExpiryNever}

	fingerprintHex = regexp.MustCompile(`(?i)^[0-9a-f]{40}$|^[0-9a-f]{64}$`)
)

// KeygenTools holds injectable tool paths and sandbox dirs. Zero value uses
// PATH lookup and the current user's default GPG/SSH locations.
type KeygenTools struct {
	GPGHome string // empty → GNUPGHOME / platform default
	SSHDir  string // empty → ~/.ssh
	GPGBin  string // empty → "gpg"
	SSHBin  string // empty → "ssh-keygen"
	Clock   func() time.Time
}

func (t KeygenTools) clock() time.Time {
	if t.Clock != nil {
		return t.Clock()
	}
	return time.Now()
}

func (t KeygenTools) gpgBin() string {
	if t.GPGBin != "" {
		return t.GPGBin
	}
	return "gpg"
}

func (t KeygenTools) sshBin() string {
	if t.SSHBin != "" {
		return t.SSHBin
	}
	return "ssh-keygen"
}

func (t KeygenTools) gpgHome() string {
	if t.GPGHome != "" {
		return t.GPGHome
	}
	return defaultGPGHome()
}

func (t KeygenTools) sshDir() (string, error) {
	if t.SSHDir != "" {
		return t.SSHDir, nil
	}
	return defaultSSHDir()
}

// --- DTOs ---

type KeygenToolStatus struct {
	KeyType        string  `json:"keyType"`
	Available      bool    `json:"available"`
	ExecutablePath *string `json:"executablePath,omitempty"`
	Version        *string `json:"version,omitempty"`
}

type KeygenAlgoGroup struct {
	KeyType string   `json:"keyType"`
	Values  []string `json:"values"`
}

type KeygenExpiryGroup struct {
	KeyType string   `json:"keyType"`
	Values  []string `json:"values"`
	Default string   `json:"default"`
}

type KeygenDefaults struct {
	OpenPGP KeygenDefaultPair `json:"openpgp"`
	SSH     KeygenDefaultPair `json:"ssh"`
}

type KeygenDefaultPair struct {
	Algorithm string `json:"algorithm"`
	Expiry    string `json:"expiry"`
}

type KeygenOptions struct {
	Tools      []KeygenToolStatus  `json:"tools"`
	Algorithms []KeygenAlgoGroup   `json:"algorithms"`
	Expiries   []KeygenExpiryGroup `json:"expiries"`
	Defaults   KeygenDefaults      `json:"defaults"`
}

type KeygenCreateRequest struct {
	ProfileID     string `json:"profileId"`
	KeyType       string `json:"keyType"`
	Algorithm     string `json:"algorithm,omitempty"`
	Expiry        string `json:"expiry,omitempty"`
	Passphrase    string `json:"passphrase,omitempty"`
	EnableGpgSign *bool  `json:"enableGpgSign,omitempty"`
	Comment       string `json:"comment,omitempty"`
	OutputPath    string `json:"outputPath,omitempty"`
	DryRun        bool   `json:"dryRun,omitempty"`
	Confirm       bool   `json:"confirm,omitempty"`
}

type KeygenProfilePatch struct {
	SigningKey string `json:"signingKey"`
	GpgFormat  string `json:"gpgFormat"`
	GpgSign    bool   `json:"gpgSign"`
}

type KeygenPlan struct {
	ProfileID      string             `json:"profileId"`
	KeyType        string             `json:"keyType"`
	ToolPath       string             `json:"toolPath"`
	CommandSummary []string           `json:"commandSummary"`
	ProfilePatch   KeygenProfilePatch `json:"profilePatch"`
	Warnings       []string           `json:"warnings,omitempty"`
	WouldFail      *string            `json:"wouldFail,omitempty"`
}

type KeygenResult struct {
	Profile          *Profile `json:"profile,omitempty"`
	KeyType          string   `json:"keyType"`
	SigningKey       string   `json:"signingKey"`
	Fingerprint      *string  `json:"fingerprint,omitempty"`
	PublicKeyPath    *string  `json:"publicKeyPath,omitempty"`
	PublicKeyPreview *string  `json:"publicKeyPreview,omitempty"`
	LocalKeyCreated  bool     `json:"localKeyCreated"`
	ProfileUpdated   bool     `json:"profileUpdated"`
	Warning          *string  `json:"warning,omitempty"`
}

type LocalKeySummary struct {
	KeyType       string  `json:"keyType"`
	ID            string  `json:"id"`
	Fingerprint   string  `json:"fingerprint"`
	UIDOrComment  string  `json:"uidOrComment"`
	PublicKeyPath *string `json:"publicKeyPath,omitempty"`
	CreatedAt     *string `json:"createdAt,omitempty"`
}

type KeyExportRequest struct {
	KeyType   string `json:"keyType"`
	ID        string `json:"id"`
	WritePath string `json:"writePath,omitempty"`
	Overwrite bool   `json:"overwrite,omitempty"`
	Confirm   bool   `json:"confirm,omitempty"`
}

type PublicKeyExport struct {
	Text        string  `json:"text"`
	WrittenPath *string `json:"writtenPath,omitempty"`
}

// KeygenProfileWriteError is returned when the key material exists locally
// but the profile store could not be updated (partial success).
type KeygenProfileWriteError struct {
	Result KeygenResult
	Cause  error
}

func (e *KeygenProfileWriteError) Error() string {
	msg := "GIT_KEYGEN_PROFILE_WRITE_FAILED: 密钥已在本机生成，但档案未更新"
	if e.Cause != nil {
		msg += "（" + e.Cause.Error() + "）"
	}
	return msg
}

func (e *KeygenProfileWriteError) Unwrap() error { return e.Cause }

// --- Options ---

// KeygenOptionsProbe returns tool availability and selectable enums.
func KeygenOptionsProbe(ctx context.Context, tools KeygenTools) KeygenOptions {
	gpgPath, gpgVer, gpgOK := probeToolVersion(ctx, tools.gpgBin())
	sshPath, sshVer, sshOK := probeToolVersion(ctx, tools.sshBin())
	return KeygenOptions{
		Tools: []KeygenToolStatus{
			toolStatus(KeyTypeOpenPGP, gpgOK, gpgPath, gpgVer),
			toolStatus(KeyTypeSSH, sshOK, sshPath, sshVer),
		},
		Algorithms: []KeygenAlgoGroup{
			{KeyType: KeyTypeOpenPGP, Values: append([]string{}, openpgpAlgos...)},
			{KeyType: KeyTypeSSH, Values: append([]string{}, sshAlgos...)},
		},
		Expiries: []KeygenExpiryGroup{
			{KeyType: KeyTypeOpenPGP, Values: append([]string{}, openpgpExp...), Default: Expiry2Y},
			{KeyType: KeyTypeSSH, Values: append([]string{}, sshExp...), Default: ExpiryNever},
		},
		Defaults: KeygenDefaults{
			OpenPGP: KeygenDefaultPair{Algorithm: AlgoED25519, Expiry: Expiry2Y},
			SSH:     KeygenDefaultPair{Algorithm: AlgoED25519, Expiry: ExpiryNever},
		},
	}
}

func toolStatus(keyType string, ok bool, path, version string) KeygenToolStatus {
	st := KeygenToolStatus{KeyType: keyType, Available: ok}
	if path != "" {
		p := path
		st.ExecutablePath = &p
	}
	if version != "" {
		v := version
		st.Version = &v
	}
	return st
}

// --- Preview / Create ---

// KeygenPreview validates parameters and returns a dry-run plan (no generation, no store write).
func KeygenPreview(ctx context.Context, store *Store, tools KeygenTools, req KeygenCreateRequest) (KeygenPlan, error) {
	normalized, failCode, warnings, toolPath, summary, err := prepareKeygen(ctx, store, tools, req)
	plan := KeygenPlan{
		ProfileID:      req.ProfileID,
		KeyType:        req.KeyType,
		ToolPath:       toolPath,
		CommandSummary: summary,
		Warnings:       warnings,
		ProfilePatch: KeygenProfilePatch{
			SigningKey: "(pending)",
			GpgFormat:  req.KeyType,
			GpgSign:    true,
		},
	}
	if normalized.ProfileID != "" {
		plan.ProfilePatch = patchFrom(normalized, "(pending)")
	}
	if failCode != "" {
		code := failCodeOnly(failCode)
		plan.WouldFail = &code
		return plan, nil
	}
	if err != nil {
		return plan, err
	}
	return plan, nil
}

// KeygenCreate generates a key and writes signing fields back to the profile.
func KeygenCreate(ctx context.Context, store *Store, tools KeygenTools, req KeygenCreateRequest) (KeygenResult, error) {
	defer clearPassphrase(&req.Passphrase)
	if !req.Confirm {
		return KeygenResult{}, errors.New("CONFIRMATION_REQUIRED")
	}
	normalized, failCode, _, _, _, err := prepareKeygen(ctx, store, tools, req)
	if err != nil {
		return KeygenResult{}, err
	}
	if failCode != "" {
		return KeygenResult{}, errors.New(strings.TrimSpace(failCode))
	}

	var (
		signingKey  string
		fingerprint string
		pubPath     string
		pubPreview  string
	)
	switch normalized.KeyType {
	case KeyTypeOpenPGP:
		fingerprint, err = createOpenPGP(ctx, store, tools, normalized)
		signingKey = fingerprint
	case KeyTypeSSH:
		signingKey, fingerprint, pubPath, pubPreview, err = createSSH(ctx, tools, normalized)
	default:
		err = errors.New("GIT_KEYGEN_PARAMS_INVALID: 不支持的密钥类型")
	}
	if err != nil {
		return KeygenResult{}, err
	}

	result := KeygenResult{
		KeyType:         normalized.KeyType,
		SigningKey:      signingKey,
		LocalKeyCreated: true,
	}
	if fingerprint != "" {
		fp := fingerprint
		result.Fingerprint = &fp
	}
	if pubPath != "" {
		p := pubPath
		result.PublicKeyPath = &p
	}
	if pubPreview != "" {
		p := pubPreview
		result.PublicKeyPreview = &p
	}

	enableSign := true
	if normalized.EnableGpgSign != nil {
		enableSign = *normalized.EnableGpgSign
	}
	updated, writeErr := writeSigningFields(store, normalized.ProfileID, signingKey, normalized.KeyType, enableSign)
	if writeErr != nil {
		warn := "密钥已在本机生成，档案未更新；可用 list/select 补写"
		result.Warning = &warn
		return result, &KeygenProfileWriteError{Result: result, Cause: writeErr}
	}
	result.Profile = updated
	result.ProfileUpdated = true
	return result, nil
}

type normalizedKeygen struct {
	ProfileID     string
	KeyType       string
	Algorithm     string
	Expiry        string
	Passphrase    string
	EnableGpgSign *bool
	Comment       string
	OutputPath    string
	Profile       *Profile
}

func prepareKeygen(ctx context.Context, store *Store, tools KeygenTools, req KeygenCreateRequest) (normalizedKeygen, string, []string, string, []string, error) {
	var empty normalizedKeygen
	warnings := []string{}
	if strings.TrimSpace(req.ProfileID) == "" {
		return empty, "GIT_KEYGEN_PARAMS_INVALID: 缺少 profileId", warnings, "", nil, nil
	}
	profile, err := store.Profile(req.ProfileID)
	if err != nil {
		return empty, "GIT_PROFILE_NOT_FOUND: 档案不存在", warnings, "", nil, nil
	}
	keyType := strings.TrimSpace(req.KeyType)
	if keyType != KeyTypeOpenPGP && keyType != KeyTypeSSH {
		return empty, "GIT_KEYGEN_PARAMS_INVALID: keyType 须为 openpgp 或 ssh", warnings, "", nil, nil
	}

	algo := strings.TrimSpace(req.Algorithm)
	expiry := strings.TrimSpace(req.Expiry)
	if keyType == KeyTypeOpenPGP {
		if algo == "" {
			algo = AlgoED25519
		}
		if expiry == "" {
			expiry = Expiry2Y
		}
		if !containsStr(openpgpAlgos, algo) {
			return empty, "GIT_KEYGEN_PARAMS_INVALID: OpenPGP 不支持该算法", warnings, "", nil, nil
		}
		if !containsStr(openpgpExp, expiry) {
			return empty, "GIT_KEYGEN_PARAMS_INVALID: OpenPGP 不支持该有效期", warnings, "", nil, nil
		}
		if strings.TrimSpace(profile.Name) == "" || strings.TrimSpace(profile.Email) == "" {
			return empty, "GIT_PROFILE_INVALID: OpenPGP 需要档案含 name 与 email", warnings, "", nil, nil
		}
	} else {
		if algo == "" {
			algo = AlgoED25519
		}
		if expiry == "" {
			expiry = ExpiryNever
		}
		if !containsStr(sshAlgos, algo) {
			return empty, "GIT_KEYGEN_PARAMS_INVALID: SSH 不支持该算法", warnings, "", nil, nil
		}
		if expiry != ExpiryNever {
			return empty, "GIT_KEYGEN_PARAMS_INVALID: SSH 有效期仅支持 never", warnings, "", nil, nil
		}
	}

	bin := tools.gpgBin()
	if keyType == KeyTypeSSH {
		bin = tools.sshBin()
	}
	path, _, ok := probeToolVersion(ctx, bin)
	if !ok {
		code := fmt.Sprintf("GIT_KEYGEN_TOOL_MISSING: 未找到 %s", bin)
		return empty, code, warnings, "", nil, nil
	}

	comment := strings.TrimSpace(req.Comment)
	if comment == "" && keyType == KeyTypeSSH {
		comment = "devtoolbox:" + req.ProfileID
	}
	outputPath := strings.TrimSpace(req.OutputPath)
	if keyType == KeyTypeSSH && outputPath != "" {
		if err := validateSSHOutputPath(outputPath); err != nil {
			return empty, err.Error(), warnings, path, nil, nil
		}
	}

	if profile.SigningKey != "" {
		warnings = append(warnings, "将覆盖档案现有 signingKey")
	}

	n := normalizedKeygen{
		ProfileID:     req.ProfileID,
		KeyType:       keyType,
		Algorithm:     algo,
		Expiry:        expiry,
		Passphrase:    req.Passphrase,
		EnableGpgSign: req.EnableGpgSign,
		Comment:       comment,
		OutputPath:    outputPath,
		Profile:       profile,
	}
	summary := buildCommandSummary(n, path)
	return n, "", warnings, path, summary, nil
}

func buildCommandSummary(n normalizedKeygen, toolPath string) []string {
	passSlot := "<passphrase>"
	if n.Passphrase == "" {
		passSlot = "<empty>"
	}
	switch n.KeyType {
	case KeyTypeOpenPGP:
		return []string{
			fmt.Sprintf("%s --batch --pinentry-mode loopback --passphrase-fd 0 --quick-generate-key %q %s default %s",
				toolPath, n.Profile.Name+" <"+n.Profile.Email+">", gpgAlgoArg(n.Algorithm), gpgExpireArg(n.Expiry)),
			"口令经 stdin 传入：" + passSlot,
			"写回档案 signingKey=指纹, gpgFormat=openpgp",
		}
	case KeyTypeSSH:
		out := n.OutputPath
		if out == "" {
			out = "{UserSSHDir}/devtoolbox_" + n.ProfileID + "_{timestamp}"
		}
		return []string{
			fmt.Sprintf("%s %s -f %s -C %q -N %s", toolPath, sshTypeArgsSummary(n.Algorithm), out, n.Comment, passSlot),
			"写回档案 signingKey=公钥绝对路径, gpgFormat=ssh",
		}
	}
	return nil
}

func patchFrom(n normalizedKeygen, signingKey string) KeygenProfilePatch {
	enable := true
	if n.EnableGpgSign != nil {
		enable = *n.EnableGpgSign
	}
	format := KeyTypeOpenPGP
	if n.KeyType == KeyTypeSSH {
		format = KeyTypeSSH
	}
	return KeygenProfilePatch{SigningKey: signingKey, GpgFormat: format, GpgSign: enable}
}

func createOpenPGP(ctx context.Context, store *Store, tools KeygenTools, n normalizedKeygen) (string, error) {
	_ = store
	home := tools.gpgHome()
	if home != "" {
		if err := prepareGPGHome(home); err != nil {
			return "", fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: 无法准备 GPG 目录")
		}
	}
	if err := ensureGPGAgent(ctx, tools, home); err != nil {
		return "", err
	}
	uid := n.Profile.Name + " <" + n.Profile.Email + ">"
	args := gpgArgs(home, "--batch", "--yes",
		"--pinentry-mode", "loopback",
		"--passphrase-fd", "0",
		"--quick-generate-key", uid, gpgAlgoArg(n.Algorithm), "default", gpgExpireArg(n.Expiry))
	stdin := n.Passphrase + "\n"
	stdout, stderr, err := runTool(ctx, "", gpgEnv(home), stdin, tools.gpgBin(), args...)
	if err != nil || gpgGenerationFailed(stderr, stdout) {
		detail := truncateToolErr(stderr)
		if detail == "" {
			detail = truncateToolErr(stdout)
		}
		if looksLikePassphraseReject(detail) {
			return "", fmt.Errorf("GIT_KEYGEN_PASSPHRASE_REJECTED: 口令未通过工具策略")
		}
		if strings.Contains(strings.ToLower(detail), "filename too long") {
			return "", fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: GPG 主目录路径过长，无法启动 gpg-agent")
		}
		if detail == "" {
			return "", fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: gpg 生成失败")
		}
		return "", fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: %s", detail)
	}
	fp, err := findNewestOpenPGPFingerprint(ctx, tools, uid)
	if err != nil {
		return "", err
	}
	return fp, nil
}

func createSSH(ctx context.Context, tools KeygenTools, n normalizedKeygen) (signingKey, fingerprint, pubPath, preview string, err error) {
	outPath := n.OutputPath
	if outPath == "" {
		dir, dirErr := tools.sshDir()
		if dirErr != nil {
			return "", "", "", "", fmt.Errorf("GIT_KEYGEN_PARAMS_INVALID: 无法解析 SSH 目录")
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return "", "", "", "", fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: 无法创建 SSH 目录")
		}
		ts := tools.clock().Format("20060102150405")
		outPath = filepath.Join(dir, fmt.Sprintf("devtoolbox_%s_%s", sanitizeFilePart(n.ProfileID), ts))
	} else {
		if err := validateSSHOutputPath(outPath); err != nil {
			return "", "", "", "", err
		}
	}
	abs, absErr := filepath.Abs(outPath)
	if absErr != nil {
		return "", "", "", "", fmt.Errorf("GIT_KEYGEN_PARAMS_INVALID: 输出路径无效")
	}
	if _, statErr := os.Stat(abs); statErr == nil {
		return "", "", "", "", errors.New("GIT_KEYGEN_PARAMS_INVALID: 私钥路径已存在")
	}
	if _, statErr := os.Stat(abs + ".pub"); statErr == nil {
		return "", "", "", "", errors.New("GIT_KEYGEN_PARAMS_INVALID: 公钥路径已存在")
	}

	args := sshKeygenArgs(n.Algorithm, abs, n.Comment, n.Passphrase)
	_, stderr, runErr := runTool(ctx, "", nil, "", tools.sshBin(), args...)
	if runErr != nil {
		detail := truncateToolErr(stderr)
		if looksLikePassphraseReject(detail) {
			return "", "", "", "", fmt.Errorf("GIT_KEYGEN_PASSPHRASE_REJECTED: 口令未通过工具策略")
		}
		if detail == "" {
			return "", "", "", "", fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: ssh-keygen 生成失败")
		}
		return "", "", "", "", fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: %s", detail)
	}
	pub := abs + ".pub"
	pubAbs, _ := filepath.Abs(pub)
	text, readErr := os.ReadFile(pubAbs)
	if readErr != nil {
		return "", "", "", "", fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: 公钥文件未生成")
	}
	preview = truncatePubPreview(string(text))
	fp := sshFingerprint(ctx, tools, pubAbs)
	return pubAbs, fp, pubAbs, preview, nil
}

func sshKeygenArgs(algo, outPath, comment, passphrase string) []string {
	args := []string{"-q"}
	switch algo {
	case AlgoED25519:
		args = append(args, "-t", "ed25519")
	case AlgoRSA4096:
		args = append(args, "-t", "rsa", "-b", "4096")
	case AlgoECDSA256:
		args = append(args, "-t", "ecdsa", "-b", "256")
	}
	args = append(args, "-f", outPath, "-C", comment, "-N", passphrase)
	return args
}

func sshTypeArgsSummary(algo string) string {
	switch algo {
	case AlgoRSA4096:
		return "-t rsa -b 4096"
	case AlgoECDSA256:
		return "-t ecdsa -b 256"
	default:
		return "-t ed25519"
	}
}

func sshFingerprint(ctx context.Context, tools KeygenTools, pubPath string) string {
	out, _, err := runTool(ctx, "", nil, "", tools.sshBin(), "-lf", pubPath, "-E", "sha256")
	if err != nil {
		return ""
	}
	// Format: 256 SHA256:xxxx comment (ED25519)
	fields := strings.Fields(strings.TrimSpace(out))
	for _, f := range fields {
		if strings.HasPrefix(f, "SHA256:") {
			return f
		}
	}
	if len(fields) >= 2 {
		return fields[1]
	}
	return strings.TrimSpace(out)
}

func writeSigningFields(store *Store, profileID, signingKey, keyType string, gpgSign bool) (*Profile, error) {
	unlock, err := store.lock()
	if err != nil {
		return nil, err
	}
	defer unlock()

	existing, err := store.Profile(profileID)
	if err != nil {
		return nil, err
	}
	updated := *existing
	updated.SigningKey = signingKey
	updated.GpgFormat = keyType
	updated.GpgSign = gpgSign
	if err := store.Upsert(updated); err != nil {
		return nil, err
	}
	if err := store.Save(); err != nil {
		return nil, fmt.Errorf("GIT_KEYGEN_PROFILE_WRITE_FAILED: %v", err)
	}
	saved, err := store.Profile(profileID)
	if err != nil {
		return nil, err
	}
	cp := *saved
	return &cp, nil
}

// --- List / Select / Export ---

// KeyList returns local key summaries for the given type.
func KeyList(ctx context.Context, tools KeygenTools, keyType string) ([]LocalKeySummary, error) {
	switch strings.TrimSpace(keyType) {
	case KeyTypeOpenPGP:
		return listOpenPGP(ctx, tools)
	case KeyTypeSSH:
		return listSSH(ctx, tools)
	default:
		return nil, errors.New("GIT_KEYGEN_PARAMS_INVALID: keyType 须为 openpgp 或 ssh")
	}
}

func listOpenPGP(ctx context.Context, tools KeygenTools) ([]LocalKeySummary, error) {
	if _, _, ok := probeToolVersion(ctx, tools.gpgBin()); !ok {
		return nil, fmt.Errorf("GIT_KEYGEN_TOOL_MISSING: 未找到 gpg")
	}
	home := tools.gpgHome()
	out, stderr, err := runTool(ctx, "", gpgEnv(home), "", tools.gpgBin(),
		gpgArgs(home, "--batch", "--list-secret-keys", "--with-colons", "--fixed-list-mode")...)
	if err != nil {
		detail := truncateToolErr(stderr)
		if detail == "" {
			return nil, fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: 无法列出 OpenPGP 密钥")
		}
		return nil, fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: %s", detail)
	}
	return parseGPGColonSecretKeys(out), nil
}

func parseGPGColonSecretKeys(out string) []LocalKeySummary {
	var results []LocalKeySummary
	var cur *LocalKeySummary
	var created string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Split(line, ":")
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "sec":
			if cur != nil && cur.ID != "" {
				results = append(results, *cur)
			}
			cur = &LocalKeySummary{KeyType: KeyTypeOpenPGP}
			created = ""
			if len(fields) > 5 && fields[5] != "" {
				if ts, err := parseUnixTS(fields[5]); err == nil {
					created = ts
				}
			}
		case "fpr", "fp2":
			if cur == nil || len(fields) < 10 {
				continue
			}
			fp := strings.TrimSpace(fields[9])
			if fp == "" {
				continue
			}
			if cur.ID == "" {
				cur.ID = fp
				cur.Fingerprint = fp
				if created != "" {
					c := created
					cur.CreatedAt = &c
				}
			}
		case "uid":
			if cur == nil {
				continue
			}
			uid := ""
			if len(fields) > 9 {
				uid = fields[9]
			}
			if cur.UIDOrComment == "" {
				cur.UIDOrComment = uid
			}
		}
	}
	if cur != nil && cur.ID != "" {
		results = append(results, *cur)
	}
	return results
}

func listSSH(ctx context.Context, tools KeygenTools) ([]LocalKeySummary, error) {
	_ = ctx
	dir, err := tools.sshDir()
	if err != nil {
		return nil, fmt.Errorf("GIT_KEYGEN_PARAMS_INVALID: 无法解析 SSH 目录")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []LocalKeySummary{}, nil
		}
		return nil, fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: 无法读取 SSH 目录")
	}
	type item struct {
		sum      LocalKeySummary
		priority int
	}
	var items []item
	for _, ent := range entries {
		if ent.IsDir() {
			continue
		}
		name := ent.Name()
		if !strings.HasSuffix(name, ".pub") {
			continue
		}
		full := filepath.Join(dir, name)
		abs, absErr := filepath.Abs(full)
		if absErr != nil {
			continue
		}
		data, readErr := os.ReadFile(abs)
		if readErr != nil {
			continue
		}
		comment := sshCommentFromPub(string(data))
		fp := sshFingerprint(ctx, tools, abs)
		if fp == "" {
			fp = abs
		}
		pri := 1
		base := strings.TrimSuffix(name, ".pub")
		if strings.HasPrefix(base, "devtoolbox_") {
			pri = 0
		}
		var created *string
		if info, infoErr := ent.Info(); infoErr == nil {
			c := info.ModTime().UTC().Format(time.RFC3339)
			created = &c
		}
		pub := abs
		items = append(items, item{
			priority: pri,
			sum: LocalKeySummary{
				KeyType:       KeyTypeSSH,
				ID:            abs,
				Fingerprint:   fp,
				UIDOrComment:  comment,
				PublicKeyPath: &pub,
				CreatedAt:     created,
			},
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].priority != items[j].priority {
			return items[i].priority < items[j].priority
		}
		return items[i].sum.ID < items[j].sum.ID
	})
	out := make([]LocalKeySummary, 0, len(items))
	for _, it := range items {
		out = append(out, it.sum)
	}
	return out, nil
}

// KeySelect writes an existing local key into the profile (no generation).
func KeySelect(ctx context.Context, store *Store, tools KeygenTools, profileID, keyType, id string, enableGpgSign *bool, confirm bool) (KeygenResult, error) {
	if !confirm {
		return KeygenResult{}, errors.New("CONFIRMATION_REQUIRED")
	}
	if strings.TrimSpace(profileID) == "" || strings.TrimSpace(id) == "" {
		return KeygenResult{}, errors.New("GIT_KEYGEN_PARAMS_INVALID: 缺少 profileId 或 id")
	}
	if _, err := store.Profile(profileID); err != nil {
		return KeygenResult{}, err
	}
	keys, err := KeyList(ctx, tools, keyType)
	if err != nil {
		return KeygenResult{}, err
	}
	var found *LocalKeySummary
	for i := range keys {
		if keys[i].ID == id || strings.EqualFold(keys[i].Fingerprint, id) {
			found = &keys[i]
			break
		}
	}
	if found == nil {
		return KeygenResult{}, fmt.Errorf("GIT_KEY_NOT_FOUND: 未找到密钥 %s", id)
	}
	signingKey := found.ID
	if keyType == KeyTypeSSH && found.PublicKeyPath != nil {
		signingKey = *found.PublicKeyPath
	}
	enable := true
	if enableGpgSign != nil {
		enable = *enableGpgSign
	}
	updated, writeErr := writeSigningFields(store, profileID, signingKey, keyType, enable)
	if writeErr != nil {
		result := KeygenResult{
			KeyType:         keyType,
			SigningKey:      signingKey,
			LocalKeyCreated: false,
			ProfileUpdated:  false,
		}
		if found.Fingerprint != "" {
			fp := found.Fingerprint
			result.Fingerprint = &fp
		}
		return result, &KeygenProfileWriteError{Result: result, Cause: writeErr}
	}
	result := KeygenResult{
		Profile:        updated,
		KeyType:        keyType,
		SigningKey:     signingKey,
		ProfileUpdated: true,
	}
	if found.Fingerprint != "" {
		fp := found.Fingerprint
		result.Fingerprint = &fp
	}
	if found.PublicKeyPath != nil {
		p := *found.PublicKeyPath
		result.PublicKeyPath = &p
	}
	return result, nil
}

// KeyExport returns public key text, optionally writing to writePath.
func KeyExport(ctx context.Context, tools KeygenTools, req KeyExportRequest) (PublicKeyExport, error) {
	if strings.TrimSpace(req.ID) == "" {
		return PublicKeyExport{}, errors.New("GIT_KEYGEN_PARAMS_INVALID: 缺少 id")
	}
	text, err := exportPublicKeyText(ctx, tools, req.KeyType, req.ID)
	if err != nil {
		return PublicKeyExport{}, err
	}
	result := PublicKeyExport{Text: text}
	if strings.TrimSpace(req.WritePath) == "" {
		return result, nil
	}
	if !req.Confirm {
		return PublicKeyExport{}, errors.New("CONFIRMATION_REQUIRED")
	}
	path := req.WritePath
	abs, absErr := filepath.Abs(path)
	if absErr != nil {
		return PublicKeyExport{}, errors.New("GIT_KEY_EXPORT_PATH_INVALID: 导出路径无效")
	}
	parent := filepath.Dir(abs)
	if info, statErr := os.Stat(parent); statErr != nil || !info.IsDir() {
		return PublicKeyExport{}, errors.New("GIT_KEY_EXPORT_PATH_INVALID: 父目录不存在")
	}
	if _, statErr := os.Stat(abs); statErr == nil && !req.Overwrite {
		return PublicKeyExport{}, errors.New("GIT_KEY_EXPORT_EXISTS: 目标文件已存在，需 overwrite")
	}
	if err := os.WriteFile(abs, []byte(text), 0o644); err != nil {
		return PublicKeyExport{}, fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: 写入导出文件失败")
	}
	result.WrittenPath = &abs
	return result, nil
}

func exportPublicKeyText(ctx context.Context, tools KeygenTools, keyType, id string) (string, error) {
	switch keyType {
	case KeyTypeOpenPGP:
		if _, _, ok := probeToolVersion(ctx, tools.gpgBin()); !ok {
			return "", fmt.Errorf("GIT_KEYGEN_TOOL_MISSING: 未找到 gpg")
		}
		home := tools.gpgHome()
		out, stderr, err := runTool(ctx, "", gpgEnv(home), "", tools.gpgBin(),
			gpgArgs(home, "--batch", "--armor", "--export", id)...)
		if err != nil || strings.TrimSpace(out) == "" {
			detail := truncateToolErr(stderr)
			if detail == "" {
				return "", fmt.Errorf("GIT_KEY_NOT_FOUND: 无法导出公钥")
			}
			return "", fmt.Errorf("GIT_KEY_NOT_FOUND: %s", detail)
		}
		return strings.TrimRight(out, "\r\n") + "\n", nil
	case KeyTypeSSH:
		path := id
		if !strings.HasSuffix(path, ".pub") {
			if _, err := os.Stat(path + ".pub"); err == nil {
				path = path + ".pub"
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("GIT_KEY_NOT_FOUND: 公钥文件不存在")
		}
		return string(data), nil
	default:
		return "", errors.New("GIT_KEYGEN_PARAMS_INVALID: keyType 须为 openpgp 或 ssh")
	}
}

func findNewestOpenPGPFingerprint(ctx context.Context, tools KeygenTools, uid string) (string, error) {
	home := tools.gpgHome()
	out, stderr, err := runTool(ctx, "", gpgEnv(home), "", tools.gpgBin(),
		gpgArgs(home, "--batch", "--list-secret-keys", "--with-colons", "--fixed-list-mode", uid)...)
	keys := parseGPGColonSecretKeys(out)
	if len(keys) == 0 {
		// Fallback: list all and pick last (UID match can fail on encoding).
		allOut, allErrOut, allErr := runTool(ctx, "", gpgEnv(home), "", tools.gpgBin(),
			gpgArgs(home, "--batch", "--list-secret-keys", "--with-colons", "--fixed-list-mode")...)
		if allErr != nil && len(keys) == 0 {
			detail := truncateToolErr(stderr)
			if detail == "" {
				detail = truncateToolErr(allErrOut)
			}
			if detail == "" {
				detail = "生成后无法读取指纹"
			}
			return "", fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: %s", detail)
		}
		keys = parseGPGColonSecretKeys(allOut)
	}
	if len(keys) == 0 {
		return "", fmt.Errorf("GIT_KEYGEN_TOOL_FAILED: 生成后未找到密钥指纹")
	}
	_ = err
	return keys[len(keys)-1].Fingerprint, nil
}

func gpgArgs(home string, args ...string) []string {
	if home == "" {
		return args
	}
	return append([]string{"--homedir", home}, args...)
}

// --- helpers ---

func gpgAlgoArg(algo string) string {
	switch algo {
	case AlgoED25519:
		return "ed25519"
	case AlgoRSA4096:
		return "rsa4096"
	case AlgoRSA3072:
		return "rsa3072"
	case AlgoNISTP256:
		return "nistp256"
	default:
		return algo
	}
}

func gpgExpireArg(expiry string) string {
	if expiry == ExpiryNever {
		return "0"
	}
	return expiry
}

func validateSSHOutputPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("GIT_KEYGEN_PARAMS_INVALID: 输出路径为空")
	}
	if strings.HasSuffix(path, ".pub") {
		return errors.New("GIT_KEYGEN_PARAMS_INVALID: outputPath 应为无私钥后缀的路径")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return errors.New("GIT_KEYGEN_PARAMS_INVALID: 输出路径无效")
	}
	parent := filepath.Dir(abs)
	info, err := os.Stat(parent)
	if err != nil || !info.IsDir() {
		return errors.New("GIT_KEYGEN_PARAMS_INVALID: 输出路径的父目录不存在")
	}
	return nil
}

func sanitizeFilePart(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "profile"
	}
	return out
}

func containsStr(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func clearPassphrase(p *string) {
	if p != nil {
		*p = ""
	}
}

func failCodeOnly(msg string) string {
	fields := strings.Fields(msg)
	if len(fields) == 0 {
		return msg
	}
	return strings.TrimSuffix(fields[0], ":")
}

func looksLikePassphraseReject(detail string) bool {
	lower := strings.ToLower(detail)
	return strings.Contains(lower, "passphrase") && (strings.Contains(lower, "too short") ||
		strings.Contains(lower, "weak") || strings.Contains(lower, "rejected") ||
		strings.Contains(lower, "invalid"))
}

// gpgGenerationFailed detects batch failures that still exit 0 (common on Windows).
func gpgGenerationFailed(stderr, stdout string) bool {
	combined := strings.ToLower(stderr + "\n" + stdout)
	markers := []string{
		"key generation failed",
		"agent_genkey failed",
		"no agent running",
		"can't connect to the gpg-agent",
	}
	for _, m := range markers {
		if strings.Contains(combined, m) {
			return true
		}
	}
	return false
}

func truncatePubPreview(text string) string {
	lines := strings.Split(strings.TrimRight(text, "\r\n"), "\n")
	if len(lines) == 0 {
		return ""
	}
	line := lines[0]
	if len(line) > 120 {
		return line[:120] + "…"
	}
	return line
}

func sshCommentFromPub(text string) string {
	line := strings.TrimSpace(strings.Split(text, "\n")[0])
	fields := strings.Fields(line)
	if len(fields) >= 3 {
		return strings.Join(fields[2:], " ")
	}
	return ""
}

func parseUnixTS(raw string) (string, error) {
	var sec int64
	if _, err := fmt.Sscan(raw, &sec); err != nil {
		return "", err
	}
	return time.Unix(sec, 0).UTC().Format(time.RFC3339), nil
}

// Ensure fingerprint helper is referenced for potential callers/tests.
var _ = fingerprintHex
