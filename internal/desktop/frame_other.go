//go:build gui && !windows

package desktop

func installClientFrame(uintptr) {}

func windowCommand(string) map[string]any {
	return map[string]any{"maximized": false}
}
