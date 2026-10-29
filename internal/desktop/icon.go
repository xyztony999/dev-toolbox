//go:build gui && windows

package desktop

import (
	_ "embed"
	"encoding/binary"
	"syscall"
	"unsafe"

	webview "github.com/webview/webview_go"
)

//go:embed icon.ico
var iconICO []byte

var (
	user32                       = syscall.NewLazyDLL("user32.dll")
	procCreateIconFromResourceEx = user32.NewProc("CreateIconFromResourceEx")
	procSendMessageW             = user32.NewProc("SendMessageW")
)

const (
	wmSetIcon      = 0x0080
	iconSmallParam = 0
	iconBigParam   = 1
	iconVersion    = 0x00030000
)

// extractIconPNG returns the ICO entry closest to the wanted size; modern
// Windows builds HICONs directly from PNG-compressed entries.
func extractIconPNG(want int) []byte {
	if len(iconICO) < 6 || iconICO[2] != 1 {
		return nil
	}
	count := int(binary.LittleEndian.Uint16(iconICO[4:6]))
	best, bestDiff := -1, 1<<30
	for i := 0; i < count; i++ {
		entry := iconICO[6+16*i : 6+16*i+16]
		width := int(entry[0])
		if width == 0 {
			width = 256
		}
		diff := width - want
		if diff < 0 {
			diff = -diff
		}
		if diff < bestDiff {
			bestDiff, best = diff, i
		}
	}
	if best < 0 {
		return nil
	}
	entry := iconICO[6+16*best : 6+16*best+16]
	size := binary.LittleEndian.Uint32(entry[8:12])
	offset := binary.LittleEndian.Uint32(entry[12:16])
	if uint64(offset)+uint64(size) > uint64(len(iconICO)) {
		return nil
	}
	return iconICO[offset : offset+size]
}

func createHICON(png []byte) uintptr {
	if len(png) == 0 {
		return 0
	}
	handle, _, _ := procCreateIconFromResourceEx.Call(
		uintptr(unsafe.Pointer(&png[0])), uintptr(len(png)),
		1, iconVersion, 0, 0, 0)
	return handle
}

// setWindowIcon applies the embedded application icon to the host window:
// title bar, task bar and alt-tab all follow WM_SETICON.
func setWindowIcon(w webview.WebView) {
	hwnd := uintptr(w.Window())
	if hwnd == 0 {
		return
	}
	if big := createHICON(extractIconPNG(256)); big != 0 {
		procSendMessageW.Call(hwnd, wmSetIcon, iconBigParam, big)
	}
	if small := createHICON(extractIconPNG(32)); small != 0 {
		procSendMessageW.Call(hwnd, wmSetIcon, iconSmallParam, small)
	}
}
