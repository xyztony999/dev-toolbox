package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"dev-toolbox/internal/runtimeinstall"
)

// runRuntimeInstall implements the runtime install CLI surface. Install
// commands only accept tool selectors, mirror ids and plan ids produced by
// preview in this process; arbitrary URLs and paths are rejected by design.
func runRuntimeInstall(ctx context.Context, args []string) int {
	mgr := runtimeinstall.Default
	if len(args) == 0 {
		return usage("runtime install options [tool] | runtime install preview <tool> <selector> [--mirror id] | runtime install start <plan-id> <digest> --confirm | runtime install status|cancel <task-id> | runtime install <tool> <selector> [--mirror id] --confirm")
	}
	switch args[0] {
	case "options":
		tool := ""
		switch len(args) {
		case 1:
		case 2:
			tool = args[1]
		default:
			return usage("runtime install options [tool]")
		}
		options, err := mgr.Options(tool)
		if err != nil {
			return report(err)
		}
		return output(options)
	case "preview":
		if len(args) != 3 && len(args) != 5 {
			return usage("runtime install preview <tool> <selector> [--mirror id]")
		}
		mirror := ""
		if len(args) == 5 {
			if args[3] != "--mirror" {
				return usage("runtime install preview <tool> <selector> [--mirror id]")
			}
			mirror = args[4]
		}
		plan, err := mgr.Preview(ctx, args[1], args[2], mirror, 0)
		if err != nil {
			return report(err)
		}
		return output(plan)
	case "start":
		if len(args) != 4 || args[3] != "--confirm" {
			return usage("runtime install start <plan-id> <digest> --confirm")
		}
		task, err := mgr.Start(args[1], args[2], 0, true)
		if err != nil {
			return report(err)
		}
		return followInstall(ctx, mgr, task.TaskID)
	case "status":
		if len(args) != 2 {
			return usage("runtime install status <task-id>")
		}
		task, err := mgr.Status(args[1])
		if err != nil {
			return report(err)
		}
		return output(task)
	case "cancel":
		if len(args) != 2 {
			return usage("runtime install cancel <task-id>")
		}
		task, err := mgr.Cancel(args[1])
		if err != nil {
			return report(err)
		}
		return output(task)
	default:
		if len(args) < 3 {
			return usage("runtime install <tool> <selector> [--mirror id] --confirm")
		}
		tool, selector := args[0], args[1]
		rest := args[2:]
		mirror := ""
		if len(rest) >= 2 && rest[0] == "--mirror" {
			mirror = rest[1]
			rest = rest[2:]
		}
		if len(rest) != 1 || rest[0] != "--confirm" {
			return usage("runtime install <tool> <selector> [--mirror id] --confirm")
		}
		plan, err := mgr.Preview(ctx, tool, selector, mirror, 0)
		if err != nil {
			return report(err)
		}
		fmt.Fprintf(os.Stderr, "预览：%s %s（%s/%s，镜像 %s）\n目标目录：%s\n预计体积：%d 字节；安装不切换全局默认、不改 PATH 或注册表\n",
			plan.Tool, plan.ResolvedVersion, plan.OS, plan.Arch, plan.Mirror, plan.TargetHome, plan.EstimatedBytes)
		task, err := mgr.Start(plan.PlanID, plan.PlanDigest, 0, true)
		if err != nil {
			return report(err)
		}
		return followInstall(ctx, mgr, task.TaskID)
	}
}

// followInstall streams task progress to stderr until the task terminates.
// Interrupting the CLI cancels the cooperative task.
func followInstall(ctx context.Context, mgr *runtimeinstall.Manager, taskID string) int {
	last := ""
	for {
		task, err := mgr.Status(taskID)
		if err != nil {
			return report(err)
		}
		line := fmt.Sprintf("%s", task.State)
		if task.Progress.Package != "" {
			line += " " + task.Progress.Package
		}
		if task.Progress.TotalBytes > 0 {
			line += fmt.Sprintf(" %d/%d", task.Progress.ReceivedBytes, task.Progress.TotalBytes)
		}
		if line != last {
			fmt.Fprintln(os.Stderr, line)
			last = line
		}
		switch task.State {
		case runtimeinstall.StateComplete:
			return output(task)
		case runtimeinstall.StateFailed, runtimeinstall.StateCancelled, runtimeinstall.StatePartial:
			if code := output(task); code != 0 {
				return code
			}
			return 1
		}
		select {
		case <-ctx.Done():
			_, _ = mgr.Cancel(taskID)
		case <-time.After(300 * time.Millisecond):
		}
	}
}
