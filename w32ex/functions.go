package w32ex

import (
	"syscall"
	"unsafe"

	"github.com/gonutz/w32/v2"
)

const (
	GA_PARENT    = 1
	GA_ROOT      = 2
	GA_ROOTOWNER = 3
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
)

func RegisterHotKey(hwnd w32.HWND, id, mod, vk int) bool {
	r1, _, _ := user32.NewProc("RegisterHotKey").Call(uintptr(hwnd), uintptr(id), uintptr(mod), uintptr(vk))
	return r1 != 0
}

func GetDpiForWindow(hwnd w32.HWND) int32 {
	r1, _, _ := user32.NewProc("GetDpiForWindow").Call(uintptr(hwnd))
	return int32(r1)
}

func GetAncestor(hwnd w32.HWND, gaFlags uint) w32.HWND {
	r1, _, _ := user32.NewProc("GetAncestor").Call(uintptr(hwnd), uintptr(gaFlags))
	return w32.HWND(r1)
}

func GetShellWindow() w32.HWND {
	r1, _, _ := user32.NewProc("GetShellWindow").Call()
	return w32.HWND(r1)
}

func SetProcessDPIAware() bool {
	r1, _, _ := user32.NewProc("SetProcessDPIAware").Call()
	return r1 != 0
}

func SetProcessDPIAwareBestEffort() bool {
	const dpiAwarenessContextPerMonitorAwareV2 = ^uintptr(3)
	if p := user32.NewProc("SetProcessDpiAwarenessContext"); p != nil {
		r1, _, _ := p.Call(dpiAwarenessContextPerMonitorAwareV2)
		if r1 != 0 {
			return true
		}
	}
	return SetProcessDPIAware()
}

func AnimateWindow(hwnd w32.HWND, dwTime uint32, dwFlags uint32) bool {
	r1, _, _ := user32.NewProc("AnimateWindow").Call(uintptr(hwnd), uintptr(dwTime), uintptr(dwFlags))
	return r1 != 0
}

func GetCurrentThreadID() uint32 {
	r, _, _ := kernel32.NewProc("GetCurrentThreadId").Call()
	return uint32(r)
}

func PostThreadMessage(threadID uint32, msg uint32, wParam, lParam uintptr) bool {
	r, _, _ := user32.NewProc("PostThreadMessageW").Call(uintptr(threadID), uintptr(msg), wParam, lParam)
	return r != 0
}

func GetWindowModuleFileName(hwnd w32.HWND) string {
	var path [32768]uint16
	ret, _, _ := user32.NewProc("GetWindowModuleFileNameW").Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(&path[0])),
		uintptr(len(path)),
	)
	if ret == 0 {
		return ""
	}
	return syscall.UTF16ToString(path[:])
}
