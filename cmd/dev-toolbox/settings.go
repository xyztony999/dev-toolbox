package main

import (
	"context"
	"fmt"
	"os"

	"dev-toolbox/internal/clientsettings"
)

// runSettings 实现 `dev-toolbox settings` CLI 表面；与桌面端共用同一个
// Go 应用服务（internal/clientsettings），不额外维护规则。
func runSettings(args []string) int {
	if len(args) == 0 {
		return usage("settings <show|theme|source|export|import|tools> [args]")
	}
	command, rest := args[0], args[1:]
	switch command {
	case "show":
		current, err := clientsettings.Get()
		if err != nil {
			return report(err)
		}
		return output(current)
	case "theme":
		return settingsTheme(rest)
	case "source":
		return settingsSource(rest)
	case "tools":
		return output(clientsettings.ToolStatuses(context.Background(), clientsettings.ToolStatusOptions{AllowPathLookup: true}))
	case "export":
		return settingsExport(rest)
	case "import":
		return settingsImport(rest)
	default:
		return usage("settings <show|theme|source|export|import|tools> [args]")
	}
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

func settingsSource(args []string) int {
	current, err := clientsettings.Get()
	if err != nil {
		return report(err)
	}
	switch {
	case len(args) >= 1 && args[0] == "add":
		if len(args) < 2 {
			return usage("settings source add <path> [--label 名] [--note 说明] [--local|--shared]")
		}
		binding := clientsettings.BindingLocal
		label, note := "", ""
		for index := 2; index < len(args); index += 1 {
			switch args[index] {
			case "--shared":
				binding = clientsettings.BindingShared
			case "--local":
				binding = clientsettings.BindingLocal
			case "--label":
				if index+1 < len(args) {
					label = args[index+1]
					index++
				}
			case "--note":
				if index+1 < len(args) {
					note = args[index+1]
					index++
				}
			}
		}
		updated, err := clientsettings.AddSourceProject(args[1], label, note, binding, current.Revision)
		if err != nil {
			return report(err)
		}
		return output(updated.SourceProjects)
	case len(args) >= 1 && args[0] == "remove":
		if len(args) < 2 {
			return usage("settings source remove <path>")
		}
		updated, removed, err := clientsettings.RemoveSourceProject(args[1], current.Revision)
		if err != nil {
			return report(err)
		}
		return output(map[string]any{"removed": removed, "sourceProjects": updated.SourceProjects})
	case len(args) == 0:
		return output(current.SourceProjects)
	default:
		return usage("settings source <add|remove|> [args]")
	}
}

func settingsExport(args []string) int {
	includeLocal := false
	label := ""
	for index := 0; index < len(args); index += 1 {
		switch args[index] {
		case "--include-local":
			includeLocal = true
		case "--label":
			if index+1 < len(args) {
				label = args[index+1]
				index++
			}
		}
	}
	snap, err := clientsettings.ExportSnapshot(label, includeLocal)
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
	return output(map[string]any{"record": record, "revision": updated.Revision})
}
