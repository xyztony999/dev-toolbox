//go:build gui && !windows

package desktop

import webview "github.com/webview/webview_go"

// setWindowIcon is a no-op off Windows; the GTK window icon is supplied by
// the desktop environment entry (.desktop file) once packaged.
func setWindowIcon(webview.WebView) {}
