package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"dev-toolbox/internal/project"
)

func run(ctx context.Context, args []string) int {
	if len(args) > 0 && args[0] == "runtime" {
		return runRuntime(ctx, args[1:])
	}
	if len(args) > 0 && args[0] == "git" {
		return runGitIdentity(ctx, args[1:])
	}
	if len(args) == 0 || args[0] != "project" {
		fmt.Fprintln(os.Stderr, "usage: dev-toolbox <project|runtime|git> ...")
		return 2
	}
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "missing project action")
		return 2
	}
	switch args[1] {
	case "scan":
		if len(args) != 3 {
			return usage("project scan <path>")
		}
		candidate, err := project.ScanPreview(ctx, args[2])
		if err != nil {
			return report(err)
		}
		if code := output(candidate); code != 0 {
			return code
		}
		return scanExitCode(candidate.Status)
	case "add":
		if len(args) < 3 || len(args) > 4 {
			return usage("project add <path> [display-name]")
		}
		candidate, err := project.ScanPreview(ctx, args[2])
		if err != nil {
			return report(err)
		}
		if code := output(candidate); code != 0 {
			return code
		}
		if candidate.Status == project.ScanCancelled {
			return 3
		}
		fmt.Fprint(os.Stderr, "登记以上项目上下文？[y/N] ")
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(line), "y") && !strings.EqualFold(strings.TrimSpace(line), "yes") {
			fmt.Fprintln(os.Stderr, "已取消")
			return 3
		}
		name := ""
		if len(args) == 4 {
			name = args[3]
		}
		p, created, err := project.Register(ctx, args[2], name, candidate.ID, candidate.EvidenceDigest)
		if err != nil {
			return report(err)
		}
		fmt.Fprintf(os.Stderr, "登记完成（created=%t）\n", created)
		if code := output(p); code != 0 {
			return code
		}
		return scanExitCode(p.Status)
	case "list":
		if len(args) != 2 {
			return usage("project list")
		}
		items, err := project.List()
		if err != nil {
			return report(err)
		}
		return output(items)
	case "show":
		if len(args) != 3 {
			return usage("project show <id>")
		}
		p, err := project.Get(args[2])
		if err != nil {
			return report(err)
		}
		return output(p)
	case "rescan":
		if len(args) != 3 {
			return usage("project rescan <id>")
		}
		p, err := project.Get(args[2])
		if err != nil {
			return report(err)
		}
		updated, err := project.Rescan(ctx, p.ID, p.Revision)
		if err != nil {
			return report(err)
		}
		if code := output(updated); code != 0 {
			return code
		}
		return scanExitCode(updated.Status)
	case "remove":
		if len(args) != 3 {
			return usage("project remove <id>")
		}
		p, err := project.Get(args[2])
		if err != nil {
			return report(err)
		}
		fmt.Fprintf(os.Stderr, "仅移除 DevToolbox 对 %q 的登记，不删除项目文件。继续？[y/N] ", p.DisplayName)
		line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if !strings.EqualFold(strings.TrimSpace(line), "y") && !strings.EqualFold(strings.TrimSpace(line), "yes") {
			fmt.Fprintln(os.Stderr, "已取消")
			return 3
		}
		removed, err := project.Unregister(p.ID, p.Revision)
		if err != nil {
			return report(err)
		}
		return output(map[string]bool{"removed": removed})
	default:
		return usage("project <scan|add|list|show|rescan|remove> [args]")
	}
}

func scanExitCode(status project.ScanStatus) int {
	switch status {
	case project.ScanCancelled:
		return 3
	case project.ScanPartial:
		return 4
	default:
		return 0
	}
}

func output(value any) int {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return report(err)
	}
	return 0
}

func usage(message string) int {
	fmt.Fprintln(os.Stderr, "usage:", message)
	return 2
}

func report(err error) int {
	fmt.Fprintln(os.Stderr, err)
	message := err.Error()
	if strings.HasPrefix(message, "PROJECT_INVALID_PATH") || strings.HasPrefix(message, "PROJECT_INVALID_REQUEST") {
		return 2
	}
	if strings.HasPrefix(message, "PROJECT_SCAN_") || strings.HasPrefix(message, "PROJECT_PERMISSION_") {
		return 4
	}
	if errors.Is(err, project.ErrNotFound) || errors.Is(err, project.ErrAlreadyExists) || errors.Is(err, project.ErrStaleScan) || errors.Is(err, project.ErrRevision) {
		return 4
	}
	return 5
}
