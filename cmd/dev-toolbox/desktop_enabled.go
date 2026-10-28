//go:build gui

package main

import "dev-toolbox/internal/desktop"

func runDesktop() bool {
	desktop.Run()
	return true
}
