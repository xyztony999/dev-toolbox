package main

import (
	"context"
	"strings"

	"dev-toolbox/internal/session"
)

// runSession 实现 session preview/open CLI。
func runSession(ctx context.Context, args []string) int {
	if len(args) == 0 {
		return usage("session <preview [path]|open [path] [--terminal id]>")
	}
	path := "."
	if len(args) > 1 && !strings.HasPrefix(args[1], "-") {
		path = args[1]
	}
	switch args[0] {
	case "preview":
		preview, err := session.ResolveContext(ctx, path)
		if err != nil {
			return report(err)
		}
		return output(preview)
	case "open":
		terminalID := flagValue(args, "--terminal")
		result, err := session.Open(ctx, path, terminalID)
		if err != nil {
			return report(err)
		}
		return output(result)
	default:
		return usage("session <preview [path]|open [path] [--terminal id]>")
	}
}
