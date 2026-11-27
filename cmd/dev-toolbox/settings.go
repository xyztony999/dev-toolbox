package main

import (
	"context"
	"fmt"
	"os"

	"dev-toolbox/internal/clientsettings"
	"dev-toolbox/internal/project"
)

// runSettings 实现 `dev-toolbox settings` CLI 表面；与桌面端共用同一个
// Go 应用服务，不额外维护一套规则。项目绑定范围由 project 拥有。
func runSettings(args []string) int {
	if len(args) == 0 {
		return usage("settings <show|theme|tools|export|import> [args]")
	}
	command, rest := args[0], args[1:]
	switch command {
	case "show":
		return outputSettings(func() (any, error) { return clientsettings.Get() })
	case "theme":
		return settingsTheme(rest)
	case "tools":
		return output(clientsettings.ToolStatuses(context.Background(), clientsettings.ToolStatusOptions{AllowPathLookup: true}))
	case "export":
		return settingsExport(rest)
	case "import":
		return settingsImport(rest)
	default:
		return usage("settings <show|theme|tools|export|import> [args]")
	}
}

func outputSettings(load func() (any, error)) int {
	value, err := load()
	if err != nil {
		return report(err)
	}
	return output(value)
}

func settingsTheme(args []string) int {
	if len(args) == 0 {
		return usage("settings theme <system|light|dark|custom> [--accent #hex]")
	}
	mode := clientsettings.ThemeMode(args[0])
	accent := ""
	for index := 1; index < len(args); index += 1 {
		if args[index] == "--accent" && index+1 < len(args) {
			accent = args[index+1]
			index++
		}
	}
	current, err := clientsettings.Get()
	if err != nil {
		return report(err)
	}
	updated, err := clientsettings.SetTheme(mode, accent, current.Revision)
	if err != nil {
		return report(err)
	}
	return output(map[string]any{"theme": updated.Theme, "customAccent": updated.CustomAccent})
}

func settingsExport(args []string) int {
	label := ""
	for index := 0; index < len(args); index += 1 {
		if args[index] == "--label" && index+1 < len(args) {
			label = args[index+1]
			index++
		}
	}
	snap, err := clientsettings.ExportSnapshot(label)
	if err != nil {
		return report(err)
	}
	payload, err := clientsettings.SnapshotBytes(snap)
	if err != nil {
		return report(err)
	}
	_, _ = fmt.Fprintln(os.Stdout, string(payload))
	return 0
}

func settingsImport(args []string) int {
	if len(args) == 0 {
		return usage("settings import <snapshot.json> [--confirm]")
	}
	path := args[0]
	confirm := false
	for index := 1; index < len(args); index += 1 {
		if args[index] == "--confirm" {
			confirm = true
		}
	}
	if !confirm {
		return usage("destructive: settings import <snapshot.json> --confirm")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return report(err)
	}
	current, err := clientsettings.Get()
	if err != nil {
		return report(err)
	}
	updated, record, err := clientsettings.ImportSnapshot(data, current.Revision)
	if err != nil {
		return report(err)
	}
	requirements, err := project.GetSharedRequirements()
	if err != nil {
		return report(err)
	}
	return output(map[string]any{"record": record, "requirements": requirements, "revision": updated.Revision})
}
