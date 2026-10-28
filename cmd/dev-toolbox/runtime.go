package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"dev-toolbox/internal/runtimecatalog"
	"dev-toolbox/internal/userstate"
)

func runRuntime(ctx context.Context, args []string) int {
	if len(args) >= 1 && args[0] == "install" {
		return runRuntimeInstall(ctx, args[1:])
	}
	if len(args) >= 1 && args[0] == "discover" {
		roots := []string{}
		if len(args) > 2 {
			return usage("runtime discover [directory]")
		}
		if len(args) == 2 {
			roots = append(roots, args[1])
		}
		result, err := runtimecatalog.Discover(ctx, nil, roots)
		if err != nil {
			return report(err)
		}
		if code := output(result); code != 0 {
			return code
		}
		if len(result.Candidates) == 0 {
			return 0
		}
		fmt.Fprintf(os.Stderr, "登记全部 %d 个候选运行时到 DevToolbox 用户清单？[y/N] ", len(result.Candidates))
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(line), "y") && !strings.EqualFold(strings.TrimSpace(line), "yes") {
			fmt.Fprintln(os.Stderr, "已取消")
			return 3
		}
		state, err := userstate.Get()
		if err != nil {
			return report(err)
		}
		refs := make([]runtimecatalog.CandidateRef, 0, len(result.Candidates))
		for _, item := range result.Candidates {
			refs = append(refs, runtimecatalog.CandidateRef{CandidateID: item.CandidateID, Digest: item.Digest, Tool: item.Tool, Home: item.Home, Source: item.Source})
		}
		_, counts, err := runtimecatalog.Register(ctx, refs, state.Revision, true)
		if err != nil {
			return report(err)
		}
		return output(counts)
	}
	if len(args) == 3 && args[0] == "inspect" {
		candidate, err := runtimecatalog.Inspect(args[1], args[2], "user")
		if err != nil {
			return report(err)
		}
		return output(candidate)
	}
	if len(args) == 4 && args[0] == "register" && args[3] == "--confirm" {
		candidate, err := runtimecatalog.Inspect(args[1], args[2], "user")
		if err != nil {
			return report(err)
		}
		state, err := userstate.Get()
		if err != nil {
			return report(err)
		}
		_, counts, err := runtimecatalog.Register(ctx, []runtimecatalog.CandidateRef{{CandidateID: candidate.CandidateID, Digest: candidate.Digest, Tool: candidate.Tool, Home: candidate.Home, Source: candidate.Source}}, state.Revision, true)
		if err != nil {
			return report(err)
		}
		return output(counts)
	}
	if len(args) == 1 && args[0] == "state" {
		state, err := userstate.Get()
		if err != nil {
			return report(err)
		}
		return output(state)
	}
	if len(args) >= 2 && args[0] == "legacy" && args[1] == "devswitch" {
		if len(args) != 3 {
			return usage("runtime legacy devswitch <inspect|preview|import>")
		}
		switch args[2] {
		case "inspect":
			result, err := userstate.InspectLegacy()
			if err != nil {
				return report(err)
			}
			return output(result)
		case "preview":
			preview, err := userstate.PreviewLegacy(ctx)
			if err != nil {
				return report(err)
			}
			return output(preview)
		case "import":
			preview, err := userstate.PreviewLegacy(ctx)
			if err != nil {
				return report(err)
			}
			if code := output(preview); code != 0 {
				return code
			}
			fmt.Fprint(os.Stderr, "将备份 DevSwitch state.json 并导入用户级默认选择。继续？[y/N] ")
			line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
			if !strings.EqualFold(strings.TrimSpace(line), "y") && !strings.EqualFold(strings.TrimSpace(line), "yes") {
				fmt.Fprintln(os.Stderr, "已取消")
				return 3
			}
			record, err := userstate.ImportLegacy(ctx, userstate.ImportOptions{ExpectedDigest: preview.SourceDigest, ConfirmBackup: true, ConfirmImport: true})
			if err != nil {
				return report(err)
			}
			return output(record)
		}
	}
	if len(args) == 4 && args[0] == "set-default" && args[3] == "--confirm" {
		state, err := userstate.Get()
		if err != nil {
			return report(err)
		}
		updated, err := userstate.SetGlobalDefault(args[1], args[2], state.Revision, true)
		if err != nil {
			return report(err)
		}
		return output(updated)
	}
	return usage("runtime <install|state|legacy devswitch inspect|preview|import|set-default <tool> <installation-id> --confirm>")
}
