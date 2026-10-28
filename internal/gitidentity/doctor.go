package gitidentity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DoctorFinding struct {
	Check  string `json:"check"`
	Level  string `json:"level"` // info | warn | error
	Detail string `json:"detail"`
}

func gitVersionFinding(ctx context.Context) DoctorFinding {
	out, ok := gitAt(ctx, "").allowFail("--version")
	if !ok {
		return DoctorFinding{Check: "gitVersion", Level: "error", Detail: "git 命令不可用；本模块所有操作依赖 git 自身"}
	}
	return DoctorFinding{Check: "gitVersion", Level: "info", Detail: out}
}

func credentialHelperFinding(ctx context.Context) DoctorFinding {
	value, ok := gitAt(ctx, "").cfgGet("global", "credential.helper")
	if !ok {
		if value, ok = gitAt(ctx, "").cfgGetEffective("credential.helper"); !ok {
			return DoctorFinding{Check: "credentialHelper", Level: "warn", Detail: "未配置 credential.helper；凭据账号选择器没有生效载体（GCM/钥匙串/store）"}
		}
	}
	return DoctorFinding{Check: "credentialHelper", Level: "info", Detail: "credential.helper = " + value}
}

// insteadOfConflictFinding warns when one original URL prefix is rewritten
// to multiple targets — pushes would follow an unpredictable one.
func insteadOfConflictFinding(ctx context.Context) *DoctorFinding {
	rewrites := listUrlRewrites(ctx, "")
	byOriginal := map[string]map[string]bool{}
	for _, rewrite := range rewrites {
		if byOriginal[rewrite.Original] == nil {
			byOriginal[rewrite.Original] = map[string]bool{}
		}
		byOriginal[rewrite.Original][rewrite.URL] = true
	}
	originals := make([]string, 0, len(byOriginal))
	for original := range byOriginal {
		originals = append(originals, original)
	}
	sort.Strings(originals)
	for _, original := range originals {
		if len(byOriginal[original]) > 1 {
			targets := make([]string, 0)
			for url := range byOriginal[original] {
				targets = append(targets, url)
			}
			sort.Strings(targets)
			return &DoctorFinding{Check: "insteadOfConflict", Level: "warn",
				Detail: fmt.Sprintf("原始地址 %q 被重写到多个目标：%s", original, strings.Join(targets, ", "))}
		}
	}
	return nil
}

type BindingHealth struct {
	ProfileID  string        `json:"profileId"`
	Dir        string        `json:"dir"`
	DirExists  bool          `json:"dirExists"`
	SnippetOK  bool          `json:"snippetOk"`
	SnippetDir bool          `json:"snippetMissing"`
	Drift      *SnippetDrift `json:"drift,omitempty"`
}

type DoctorReport struct {
	Findings       []DoctorFinding `json:"findings"`
	Bindings       []BindingHealth `json:"bindings"`
	OrphanSnippets []string        `json:"orphanSnippets"`
	HookPresent    bool            `json:"hookPresent"`
}

// GitDoctor is read-only: it reports; it never repairs (product rule 1).
func GitDoctor(ctx context.Context, store *Store) (*DoctorReport, error) {
	report := &DoctorReport{Findings: []DoctorFinding{}, Bindings: []BindingHealth{}, OrphanSnippets: []string{}}
	report.Findings = append(report.Findings, gitVersionFinding(ctx))
	report.Findings = append(report.Findings, credentialHelperFinding(ctx))
	if conflict := insteadOfConflictFinding(ctx); conflict != nil {
		report.Findings = append(report.Findings, *conflict)
	}
	report.HookPresent = includeHookPresent(ctx)
	if len(activeBindings(store)) > 0 && !report.HookPresent {
		report.Findings = append(report.Findings, DoctorFinding{Check: "bindingHealth", Level: "error",
			Detail: "存在目录绑定，但全局 gitconfig 缺少 include 钩子；绑定不会生效"})
	}

	ids := make([]string, 0, len(store.Profiles))
	for id := range store.Profiles {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	referencedSnippets := map[string]bool{}
	for _, id := range ids {
		profile := store.Profiles[id]
		if len(identityGitdirs(profile)) == 0 {
			continue
		}
		snippet, err := snippetPath(id)
		if err != nil {
			continue
		}
		referencedSnippets[strings.ToLower(snippet)] = true
		for _, dir := range identityGitdirs(profile) {
			health := BindingHealth{ProfileID: id, Dir: dir, DirExists: true, SnippetOK: true}
			if _, err := os.Stat(strings.TrimSuffix(filepath.FromSlash(dir), string(filepath.Separator))); err != nil {
				health.DirExists = false
				report.Findings = append(report.Findings, DoctorFinding{Check: "bindingHealth", Level: "warn",
					Detail: fmt.Sprintf("绑定目录不存在：%s（身份 %s）", dir, id)})
			}
			if _, err := os.Stat(snippet); err != nil {
				health.SnippetOK = false
				health.SnippetDir = true
				report.Findings = append(report.Findings, DoctorFinding{Check: "bindingHealth", Level: "error",
					Detail: fmt.Sprintf("托管片段缺失：%s（身份 %s）", snippet, id)})
			} else if driftReport := checkSnippetDrift(ctx, id, profile); driftReport.Drift {
				copied := driftReport
				health.Drift = &copied
				report.Findings = append(report.Findings, DoctorFinding{Check: "bindingHealth", Level: "warn",
					Detail: fmt.Sprintf("片段被手改，与身份 %s 不一致（键：%s）", id, strings.Join(copied.Keys, ", "))})
			}
			report.Bindings = append(report.Bindings, health)
		}
	}
	// nesting: one bound directory inside another bound directory (different profiles)
	type pair struct{ id, dir string }
	all := []pair{}
	for _, id := range ids {
		for _, dir := range identityGitdirs(store.Profiles[id]) {
			all = append(all, pair{id, dir})
		}
	}
	for i, outer := range all {
		for j, inner := range all {
			if i == j || outer.id == inner.id {
				continue
			}
			rel, err := filepath.Rel(filepath.FromSlash(strings.TrimSuffix(outer.dir, "/")), filepath.FromSlash(strings.TrimSuffix(inner.dir, "/")))
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != "." {
				report.Findings = append(report.Findings, DoctorFinding{Check: "bindingHealth", Level: "warn",
					Detail: fmt.Sprintf("目录嵌套绑定：%s（%s）位于 %s（%s）之内", inner.dir, inner.id, outer.dir, outer.id)})
			}
		}
	}
	if base, err := BaseDir(); err == nil {
		entries, _ := os.ReadDir(filepath.Join(base, "snippets"))
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".gitconfig") {
				continue
			}
			full := filepath.Join(base, "snippets", entry.Name())
			if !referencedSnippets[strings.ToLower(full)] {
				report.OrphanSnippets = append(report.OrphanSnippets, full)
			}
		}
	}
	if len(report.OrphanSnippets) > 0 {
		report.Findings = append(report.Findings, DoctorFinding{Check: "bindingHealth", Level: "warn",
			Detail: fmt.Sprintf("发现 %d 个孤儿片段（无绑定引用）", len(report.OrphanSnippets))})
	}
	return report, nil
}

func activeBindings(store *Store) []string {
	bindings := []string{}
	for _, profile := range store.Profiles {
		bindings = append(bindings, identityGitdirs(profile)...)
	}
	return bindings
}
