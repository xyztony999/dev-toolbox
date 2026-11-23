//go:build gui && windows

package desktop

import (
	"syscall"
	"unsafe"
)

// 无边框客户区，对齐 engine-gui 的 decorations:false：
// 系统标题栏不再占一行，自绘标题栏就是窗口标题栏。
// WM_NCCALCSIZE 把整窗收成客户区；边缘命中交给 WM_NCHITTEST
// （子窗口先让出），拖动由标题栏按下后转入 HTCAPTION。

const (
	gwlpWndProc   = ^uintptr(3) // -4
	htTransparent = ^uintptr(0)

	wmNcCalcSize     = 0x0083
	wmNcHitTest      = 0x0084
	wmNcLButtonDown  = 0x00A1
	wmClose          = 0x0010
	wmParentNotify   = 0x0210
	htCaption        = 2
	swMinimize       = 6
	swMaximize       = 3
	swRestore        = 9
	swpNoSize        = 0x0001
	swpNoMove        = 0x0002
	swpNoZOrder      = 0x0004
	swpFrameChanged  = 0x0020
	smCxFrame        = 32
	smCxPaddedBorder = 92
	gwChild          = 5
	gwHwndNext       = 2
	monitorNearest   = 2
)

var (
	procSetWindowLongPtrW = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProcW   = user32.NewProc("CallWindowProcW")
	procGetSystemMetrics  = user32.NewProc("GetSystemMetrics")
	procIsZoomed          = user32.NewProc("IsZoomed")
	procShowWindow        = user32.NewProc("ShowWindow")
	procPostMessageW      = user32.NewProc("PostMessageW")
	procReleaseCapture    = user32.NewProc("ReleaseCapture")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
	procMonitorFromWindow = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW   = user32.NewProc("GetMonitorInfoW")
	procDefWindowProcW    = user32.NewProc("DefWindowProcW")
	procGetWindow         = user32.NewProc("GetWindow")
	procScreenToClient    = user32.NewProc("ScreenToClient")
	procGetClientRect     = user32.NewProc("GetClientRect")

	dwmapi                 = syscall.NewLazyDLL("dwmapi.dll")
	procDwmExtendFrame     = dwmapi.NewProc("DwmExtendFrameIntoClientArea")
	procDwmSetWindowAttrib = dwmapi.NewProc("DwmSetWindowAttribute")

	origWndProc uintptr
	clientHwnd  uintptr
	wndProcCB   = syscall.NewCallback(clientWndProc)
	childProcCB uintptr
	// 子窗口原过程。只在 UI 线程读写。
	childProcs = map[uintptr]uintptr{}
)

type winRect struct {
	Left, Top, Right, Bottom int32
}

type ncCalcSizeParams struct {
	Rgrc  [3]winRect
	Lppos uintptr
}

type monitorInfo struct {
	CbSize    uint32
	RcMonitor winRect
	RcWork    winRect
	DwFlags   uint32
}

type winPoint struct {
	X, Y int32
}

func init() {
	childProcCB = syscall.NewCallback(childWndProc)
}

func installClientFrame(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	clientHwnd = hwnd
	prev, _, _ := procSetWindowLongPtrW.Call(hwnd, gwlpWndProc, wndProcCB)
	origWndProc = prev
	subclassChildren(hwnd)
	var margins struct{ Left, Right, Top, Bottom int32 }
	margins.Left, margins.Right, margins.Top, margins.Bottom = 1, 1, 1, 1
	procDwmExtendFrame.Call(hwnd, uintptr(unsafe.Pointer(&margins)))
	// DWMWA_WINDOW_CORNER_PREFERENCE = 33, DWMWCP_ROUND = 2
	round := int32(2)
	procDwmSetWindowAttrib.Call(hwnd, 33, uintptr(unsafe.Pointer(&round)), unsafe.Sizeof(round))
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoZOrder|swpFrameChanged)
}

func windowCommand(action string) map[string]any {
	hwnd := clientHwnd
	if hwnd == 0 {
		return map[string]any{"maximized": false}
	}
	switch action {
	case "minimize":
		procShowWindow.Call(hwnd, swMinimize)
	case "toggleMaximize":
		if zoomed(hwnd) {
			procShowWindow.Call(hwnd, swRestore)
		} else {
			procShowWindow.Call(hwnd, swMaximize)
		}
	case "close":
		procPostMessageW.Call(hwnd, wmClose, 0, 0)
	case "drag":
		procReleaseCapture.Call()
		procSendMessageW.Call(hwnd, wmNcLButtonDown, htCaption, 0)
	}
	return map[string]any{"maximized": zoomed(hwnd)}
}

func zoomed(hwnd uintptr) bool {
	r, _, _ := procIsZoomed.Call(hwnd)
	return r != 0
}

func clientWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch msg {
	case wmNcCalcSize:
		if wp != 0 {
			if zoomed(hwnd) {
				if work, ok := workArea(hwnd); ok {
					(*ncCalcSizeParams)(unsafe.Pointer(lp)).Rgrc[0] = work
				}
			}
			return 0
		}
	case wmNcHitTest:
		if code, ok := edgeHit(hwnd, lp); ok {
			return code
		}
	case wmParentNotify:
		subclassChildren(hwnd)
	}
	return callPrev(origWndProc, hwnd, msg, wp, lp)
}

func childWndProc(hwnd, msg, wp, lp uintptr) uintptr {
	switch msg {
	case wmNcHitTest:
		if _, ok := edgeHit(clientHwnd, lp); ok {
			return htTransparent
		}
	case wmParentNotify:
		subclassChildren(clientHwnd)
	}
	return callPrev(childProcs[hwnd], hwnd, msg, wp, lp)
}

func callPrev(prev, hwnd, msg, wp, lp uintptr) uintptr {
	if prev != 0 {
		r, _, _ := procCallWindowProcW.Call(prev, hwnd, msg, wp, lp)
		return r
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wp, lp)
	return r
}

func subclassChildren(root uintptr) {
	if root == 0 {
		return
	}
	var walk func(uintptr)
	walk = func(hwnd uintptr) {
		if hwnd != clientHwnd {
			if _, done := childProcs[hwnd]; !done {
				prev, _, _ := procSetWindowLongPtrW.Call(hwnd, gwlpWndProc, childProcCB)
				if prev != 0 {
					childProcs[hwnd] = prev
				}
			}
		}
		child, _, _ := procGetWindow.Call(hwnd, gwChild)
		for child != 0 {
			walk(child)
			child, _, _ = procGetWindow.Call(child, gwHwndNext)
		}
	}
	child, _, _ := procGetWindow.Call(root, gwChild)
	for child != 0 {
		walk(child)
		child, _, _ = procGetWindow.Call(child, gwHwndNext)
	}
}

func edgeHit(hwnd, lp uintptr) (uintptr, bool) {
	if hwnd == 0 || zoomed(hwnd) {
		return 0, false
	}
	pt := winPoint{X: int32(int16(lp & 0xFFFF)), Y: int32(int16((lp >> 16) & 0xFFFF))}
	procScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
	var rc winRect
	procGetClientRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
	pad := framePad()
	left := pt.X >= 0 && pt.X < pad
	right := pt.X < rc.Right && pt.X >= rc.Right-pad
	top := pt.Y >= 0 && pt.Y < pad
	bottom := pt.Y < rc.Bottom && pt.Y >= rc.Bottom-pad
	switch {
	case top && left:
		return 13, true // HTTOPLEFT
	case top && right:
		return 14, true
	case bottom && left:
		return 16, true
	case bottom && right:
		return 17, true
	case left:
		return 10, true
	case right:
		return 11, true
	case top:
		return 12, true
	case bottom:
		return 15, true
	default:
		return 0, false
	}
}

func framePad() int32 {
	frame, _, _ := procGetSystemMetrics.Call(smCxFrame)
	border, _, _ := procGetSystemMetrics.Call(smCxPaddedBorder)
	pad := int32(frame + border)
	if pad < 6 {
		return 6
	}
	return pad
}

func workArea(hwnd uintptr) (winRect, bool) {
	mon, _, _ := procMonitorFromWindow.Call(hwnd, monitorNearest)
	if mon == 0 {
		return winRect{}, false
	}
	var info monitorInfo
	info.CbSize = uint32(unsafe.Sizeof(info))
	ok, _, _ := procGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return winRect{}, false
	}
	return info.RcWork, true
}
