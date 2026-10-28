package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"dev-toolbox/internal/gitidentity"
)

// runGitIdentity implements the `dev-toolbox git` CLI surface. Destructive
// operations need --confirm exactly like the other CLI domains.
func runGitIdentity(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return usage("git <profiles|save-profile|remove-profile|import-global|use|current|bind|unbind|unset|scan|audit|doctor|remote|credential|legacy> [args]")
	}
	store, err := gitidentity.Load()
	if err != nil {
		return report(err)
	}
	command, rest := args[0], args[1:]
	switch command {
	case "profiles":
		profiles, marks, err := gitidentity.ListProfiles(ctx, store)
		if err != nil {
			return report(err)
		}
		return output(map[string]any{"profiles": profiles, "marks": marks})
	case "save-profile":
		return gitSaveProfile(ctx, store, rest)
	case "remove-profile":
		if len(rest) != 2 || rest[1] != "--confirm" {
			return usage("git remove-profile <id> --confirm")
		}
		if err := gitidentity.RemoveProfile(ctx, store, rest[0]); err != nil {
			return report(err)
		}
		return output(map[string]string{"removed": rest[0]})
	case "import-global":
		if !hasFlag(rest, "--confirm") {
			return usage("git import-global [--as id] --confirm")
		}
		profile, err := gitidentity.ImportGlobalIdentity(ctx, store, flagValue(rest, "--as"))
		if err != nil {
			return report(err)
		}
		return output(profile)
	case "use":
		return gitUse(ctx, store, rest)
	case "current":
		path := "."
		prompt := false
		for _, arg := range rest {
			switch {
			case arg == "--prompt":
				prompt = true
			case !strings.HasPrefix(arg, "-"):
				path = arg
			}
		}
		effective, err := gitidentity.ResolveEffective(ctx, store, path, prompt)
		if err != nil {
			return report(err)
		}
		if prompt {
			fmt.Println(effective.PromptLine)
			return 0
		}
		return output(effective)
	case "bind":
		if len(rest) < 2 || !hasFlag(rest, "--confirm") {
			return usage("git bind <id> <dir> --confirm")
		}
		profile, _, err := gitidentity.Bind(ctx, store, rest[0], rest[1])
		if err != nil {
			return report(err)
		}
		return output(profile)
	case "unbind":
		if len(rest) < 1 || !hasFlag(rest, "--confirm") {
			return usage("git unbind <id> [dir] --confirm")
		}
		dir := ""
		for _, arg := range rest[1:] {
			if !strings.HasPrefix(arg, "-") {
				dir = arg
			}
		}
		profile, drift, err := gitidentity.Unbind(ctx, store, rest[0], dir)
		if err != nil {
			return report(err)
		}
		return output(map[string]any{"profile": profile, "drift": drift})
	case "unset":
		scope, path := "local", "."
		for _, arg := range rest {
			switch arg {
			case "--global":
				scope = "global"
			case "--local":
				scope = "local"
			case "--confirm":
			default:
				if !strings.HasPrefix(arg, "-") {
					path = arg
				}
			}
		}
		if !hasFlag(rest, "--confirm") {
			return usage("git unset [--local|--global] [path] --confirm")
		}
		return output(gitidentity.UnsetIdentity(ctx, store, scope, path))
	case "scan":
		return gitScan(ctx, store, rest)
	case "audit":
		return gitAudit(ctx, store, rest)
	case "doctor":
		doctorReport, err := gitidentity.GitDoctor(ctx, store)
		if err != nil {
			return report(err)
		}
		return output(doctorReport)
	case "remote":
		return gitRemote(ctx, rest)
	case "credential":
		return gitCredential(ctx, store, rest)
	case "legacy":
		return gitLegacy(ctx, store, rest)
	default:
		return usage("git <profiles|save-profile|remove-profile|import-global|use|current|bind|unbind|unset|scan|audit|doctor|remote|credential|legacy> [args]")
	}
}

func hasFlag(args []string, flag string) bool {
	for _, arg := range args {
		if arg == flag {
			return true
		}
	}
	return false
}

func flagValue(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
		if strings.HasPrefix(arg, flag+"=") {
			return strings.TrimPrefix(arg, flag+"=")
		}
	}
	return ""
}

func flagValues(args []string, flag string) []string {
	values := []string{}
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			values = append(values, args[i+1])
		}
	}
	return values
}

func gitSaveProfile(ctx context.Context, store *gitidentity.Store, args []string) int {
	if len(args) == 0 || !hasFlag(args, "--confirm") {
		return usage("git save-profile <id> --name <n> --email <e> [--signingkey k] [--gpgsign] [--format openpgp|ssh|x509] [--set k=v]... [--insteadOf original=url]... [--account host=user]... --confirm")
	}
	profile := gitidentity.Profile{ID: args[0], CustomKv: map[string]string{}}
	for i := 1; i < len(args); i++ {
		arg := args[i]
		next := func() string {
			if i+1 >= len(args) {
				return ""
			}
			i++
			return args[i]
		}
		switch {
		case arg == "--name":
			profile.Name = next()
		case arg == "--email":
			profile.Email = next()
		case arg == "--signingkey":
			profile.SigningKey = next()
		case arg == "--gpgsign":
			profile.GpgSign = true
		case arg == "--format":
			profile.GpgFormat = next()
		case arg == "--set":
			parts := strings.SplitN(next(), "=", 2)
			if len(parts) == 2 {
				profile.CustomKv[parts[0]] = parts[1]
			}
		case arg == "--insteadOf":
			parts := strings.SplitN(next(), "=", 2)
			if len(parts) == 2 {
				profile.InsteadOf = append(profile.InsteadOf, gitidentity.InsteadOf{Original: parts[0], URL: parts[1]})
			}
		case arg == "--account":
			parts := strings.SplitN(next(), "=", 2)
			if len(parts) == 2 {
				profile.Accounts = append(profile.Accounts, gitidentity.Account{Host: parts[0], Username: parts[1]})
			}
		case arg == "--confirm":
		}
	}
	if len(profile.CustomKv) == 0 {
		profile.CustomKv = nil
	}
	if err := store.Upsert(profile); err != nil {
		return report(err)
	}
	if err := store.Save(); err != nil {
		return report(err)
	}
	saved, err := store.Profile(profile.ID)
	if err != nil {
		return report(err)
	}
	return output(saved)
}

func gitUse(ctx context.Context, store *gitidentity.Store, args []string) int {
	if len(args) == 0 {
		return usage("git use <id> [--local] [--repos path]... [--dry-run] [--confirm]")
	}
	id := args[0]
	dryRun := hasFlag(args, "--dry-run")
	confirm := hasFlag(args, "--confirm")
	if !dryRun && !confirm {
		return usage("git use <id> needs --dry-run to preview or --confirm to write")
	}
	repos := flagValues(args, "--repos")
	if hasFlag(args, "--local") && len(repos) == 0 {
		changes, err := gitidentity.ApplyProfile(ctx, store, id, "local", ".", dryRun)
		if err != nil {
			return report(err)
		}
		return output(map[string]any{"scope": "local", "dryRun": dryRun, "changes": changes})
	}
	if len(repos) == 0 {
		changes, err := gitidentity.ApplyProfile(ctx, store, id, "global", "", dryRun)
		if err != nil {
			return report(err)
		}
		return output(map[string]any{"scope": "global", "dryRun": dryRun, "changes": changes})
	}
	results := gitidentity.ApplyToRepos(ctx, store, id, repos, dryRun)
	failed := 0
	for _, result := range results {
		if result.Status == "failed" {
			failed++
		}
	}
	if failed > 0 {
		if code := output(results); code != 0 {
			return code
		}
		return 4
	}
	return output(results)
}

func gitScan(ctx context.Context, store *gitidentity.Store, args []string) int {
	root := flagValue(args, "--root")
	depth := store.Settings.ScanDepth
	if raw := flagValue(args, "--depth"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return usage("git scan [--root dir] [--depth n] [--save --confirm]")
		}
		depth = parsed
	}
	if root == "" {
		root = store.Settings.ScanRoot
	}
	if root == "" {
		root = "."
	}
	if hasFlag(args, "--save") {
		if !hasFlag(args, "--confirm") {
			return usage("git scan --save needs --confirm")
		}
		if err := gitidentity.SaveScanSettings(store, root, depth); err != nil {
			return report(err)
		}
	}
	scan, err := gitidentity.ScanRepos(ctx, store, root, depth)
	if err != nil {
		return report(err)
	}
	return output(scan)
}

func gitAudit(ctx context.Context, store *gitidentity.Store, args []string) int {
	repo := "."
	expectID := ""
	strict := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--strict":
			strict = true
		case "--expect":
			if i+1 >= len(args) {
				return usage("git audit [repo] [--expect id] [--strict]")
			}
			i++
			expectID = args[i]
		default:
			if !strings.HasPrefix(args[i], "-") {
				repo = args[i]
			}
		}
	}
	reportData, err := gitidentity.AuditRepo(ctx, store, repo, expectID)
	if err != nil {
		return report(err)
	}
	if code := output(reportData); code != 0 {
		return code
	}
	if strict && reportData.StrictPassed != nil && !*reportData.StrictPassed {
		return 2
	}
	return 0
}

func gitRemote(ctx context.Context, args []string) int {
	if len(args) == 0 || args[0] == "list" {
		repo := "."
		if len(args) > 1 {
			repo = args[1]
		}
		return output(gitidentity.MirrorList(ctx, repo))
	}
	remoteName := flagValue(args, "--remote")
	switch args[0] {
	case "mirror":
		if len(args) < 2 || !hasFlag(args, "--confirm") {
			return usage("git remote mirror <url> [--remote name] --confirm")
		}
		urls, err := gitidentity.MirrorSet(ctx, ".", remoteName, args[1])
		if err != nil {
			return report(err)
		}
		return output(map[string]any{"remote": remoteName, "pushUrls": urls})
	case "unmirror":
		if !hasFlag(args, "--confirm") {
			return usage("git remote unmirror [--remote name] --confirm")
		}
		removed, err := gitidentity.MirrorUnset(ctx, ".", remoteName)
		if err != nil {
			return report(err)
		}
		return output(map[string]any{"remote": remoteName, "removed": removed})
	default:
		return usage("git remote <list|mirror <url>|unmirror> [--remote name] --confirm")
	}
}

func gitCredential(ctx context.Context, store *gitidentity.Store, args []string) int {
	if len(args) == 0 || args[0] == "list" {
		host := ""
		if len(args) > 1 {
			host = args[1]
		}
		return output(gitidentity.CredentialList(ctx, store, host))
	}
	switch args[0] {
	case "login":
		if len(args) < 2 {
			return usage("git credential login <host> [--username u]")
		}
		username, err := gitidentity.CredentialLogin(ctx, store, args[1], flagValue(args, "--username"))
		if err != nil {
			return report(err)
		}
		return output(map[string]string{"host": args[1], "username": username})
	case "set":
		if len(args) < 3 || !hasFlag(args, "--confirm") {
			return usage("git credential set <host> <username> --confirm （token 经 stdin 管道输入，不回显、不落盘）")
		}
		fmt.Fprintln(os.Stderr, "粘贴 token 后回车（建议 echo $token | dev-toolbox git credential set ...；token 不落盘不回显）：")
		reader := bufio.NewReader(os.Stdin)
		token, readErr := reader.ReadString('\n')
		if readErr != nil && token == "" {
			return reportStr("GIT_CREDENTIAL_REJECTED: token input unavailable")
		}
		token = strings.TrimRight(token, "\r\n")
		if err := gitidentity.CredentialStore(ctx, store, args[1], args[2], token); err != nil {
			return report(err)
		}
		return output(map[string]string{"host": args[1], "username": args[2]})
	case "remove":
		if len(args) < 2 || !hasFlag(args, "--confirm") {
			return usage("git credential remove <host> [username] --confirm")
		}
		if err := gitidentity.CredentialErase(ctx, store, args[1], flagValue(args, "--username")); err != nil {
			return report(err)
		}
		return output(map[string]string{"host": args[1]})
	default:
		return usage("git credential <list [host]|login <host>|set <host> <user>|remove <host>>")
	}
}

func gitLegacy(ctx context.Context, store *gitidentity.Store, args []string) int {
	if len(args) == 0 || args[0] == "preview" {
		preview, err := gitidentity.PreviewLegacy(ctx, store)
		if err != nil {
			return report(err)
		}
		return output(preview)
	}
	if args[0] == "import" {
		if !hasFlag(args, "--confirm") {
			return usage("git legacy import <source-digest> [--ids id1,id2] --confirm")
		}
		digest := ""
		if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
			digest = args[1]
		}
		if digest == "" {
			return usage("git legacy import <source-digest> [--ids id1,id2] --confirm")
		}
		var ids []string
		if raw := flagValue(args, "--ids"); raw != "" {
			ids = strings.Split(raw, ",")
		}
		record, err := gitidentity.LegacyImport(ctx, store, digest, ids, true, true)
		if err != nil {
			return report(err)
		}
		return output(record)
	}
	return usage("git legacy <preview|import <digest>>")
}

func reportStr(message string) int {
	fmt.Fprintln(os.Stderr, message)
	return 5
}
