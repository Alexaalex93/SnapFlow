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
	shcore   = syscall.NewLazyDLL("shcore.dll")

	// Procs are resolved once and cached. Creating a LazyProc per call (as the
	// previous code did) re-ran GetProcAddress on every hot-path call.
	procRegisterHotKey                = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey              = user32.NewProc("UnregisterHotKey")
	procGetDpiForWindow               = user32.NewProc("GetDpiForWindow")
	procGetAncestor                   = user32.NewProc("GetAncestor")
	procGetShellWindow                = user32.NewProc("GetShellWindow")
	procSetProcessDPIAware            = user32.NewProc("SetProcessDPIAware")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procGetCurrentThreadId            = kernel32.NewProc("GetCurrentThreadId")
	procPostThreadMessageW            = user32.NewProc("PostThreadMessageW")
	procIsZoomed                      = user32.NewProc("IsZoomed")
	procIsIconic                      = user32.NewProc("IsIconic")
	procSendMessageTimeoutW           = user32.NewProc("SendMessageTimeoutW")
	procSystemParametersInfoW         = user32.NewProc("SystemParametersInfoW")
	procRegisterWindowMessageW        = user32.NewProc("RegisterWindowMessageW")
	procGetDpiForMonitor              = shcore.NewProc("GetDpiForMonitor")
)

func RegisterHotKey(hwnd w32.HWND, id, mod, vk int) bool {
	r1, _, _ := procRegisterHotKey.Call(uintptr(hwnd), uintptr(id), uintptr(mod), uintptr(vk))
	return r1 != 0
}

func UnregisterHotKey(hwnd w32.HWND, id int) bool {
	r1, _, _ := procUnregisterHotKey.Call(uintptr(hwnd), uintptr(id))
	return r1 != 0
}

func GetDpiForWindow(hwnd w32.HWND) int32 {
	if procGetDpiForWindow.Find() != nil {
		return 96
	}
	r1, _, _ := procGetDpiForWindow.Call(uintptr(hwnd))
	return int32(r1)
}

// GetDpiForMonitor returns the effective DPI of a monitor (96 if unavailable).
func GetDpiForMonitor(mon w32.HMONITOR) int32 {
	if mon == 0 || procGetDpiForMonitor.Find() != nil {
		return 96
	}
	var dx, dy uint32
	const mdtEffectiveDpi = 0
	hr, _, _ := procGetDpiForMonitor.Call(uintptr(mon), mdtEffectiveDpi,
		uintptr(unsafe.Pointer(&dx)), uintptr(unsafe.Pointer(&dy)))
	if hr != 0 || dy == 0 {
		return 96
	}
	return int32(dy)
}

func GetAncestor(hwnd w32.HWND, gaFlags uint) w32.HWND {
	r1, _, _ := procGetAncestor.Call(uintptr(hwnd), uintptr(gaFlags))
	return w32.HWND(r1)
}

func GetShellWindow() w32.HWND {
	r1, _, _ := procGetShellWindow.Call()
	return w32.HWND(r1)
}

func SetProcessDPIAware() bool {
	r1, _, _ := procSetProcessDPIAware.Call()
	return r1 != 0
}

// SetProcessDPIAwareBestEffort enables Per-Monitor-V2 DPI awareness so every
// coordinate we read (GetWindowRect, DWM frame bounds, monitor work areas) is
// in physical pixels — no DPI virtualisation, no manual scaling needed.
func SetProcessDPIAwareBestEffort() bool {
	const dpiAwarenessContextPerMonitorAwareV2 = ^uintptr(3) // (DPI_AWARENESS_CONTEXT)-4
	if procSetProcessDpiAwarenessContext.Find() == nil {
		r1, _, _ := procSetProcessDpiAwarenessContext.Call(dpiAwarenessContextPerMonitorAwareV2)
		if r1 != 0 {
			return true
		}
	}
	return SetProcessDPIAware()
}

func GetCurrentThreadID() uint32 {
	r, _, _ := procGetCurrentThreadId.Call()
	return uint32(r)
}

func PostThreadMessage(threadID uint32, msg uint32, wParam, lParam uintptr) bool {
	r, _, _ := procPostThreadMessageW.Call(uintptr(threadID), uintptr(msg), wParam, lParam)
	return r != 0
}

func IsZoomed(hwnd w32.HWND) bool {
	r, _, _ := procIsZoomed.Call(uintptr(hwnd))
	return r != 0
}

func IsIconic(hwnd w32.HWND) bool {
	r, _, _ := procIsIconic.Call(uintptr(hwnd))
	return r != 0
}

const (
	SMTO_ABORTIFHUNG = 0x0002
	WM_NCHITTEST     = 0x0084
)

// SendMessageTimeout sends msg with a timeout so a hung target window can
// never freeze SnapFlow's main thread. ok=false on timeout/failure.
func SendMessageTimeout(hwnd w32.HWND, msg uint32, wParam, lParam uintptr, timeoutMs uint32) (result uintptr, ok bool) {
	r, _, _ := procSendMessageTimeoutW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam,
		SMTO_ABORTIFHUNG, uintptr(timeoutMs), uintptr(unsafe.Pointer(&result)))
	return result, r != 0
}

const (
	SPI_GETWINARRANGING = 0x0082
	SPI_SETWINARRANGING = 0x0083
	SPIF_UPDATEINIFILE  = 0x01
	SPIF_SENDCHANGE     = 0x02
)

func SystemParametersInfo(action, uiParam uint32, pvParam unsafe.Pointer, winIni uint32) bool {
	r, _, _ := procSystemParametersInfoW.Call(uintptr(action), uintptr(uiParam), uintptr(pvParam), uintptr(winIni))
	return r != 0
}

func RegisterWindowMessage(name string) uint32 {
	p, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return 0
	}
	r, _, _ := procRegisterWindowMessageW.Call(uintptr(unsafe.Pointer(p)))
	return uint32(r)
}

var (
	dwmapi                    = syscall.NewLazyDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
)

const (
	DWMWA_WINDOW_CORNER_PREFERENCE = 33 // Windows 11+

	DWMWCP_DEFAULT    = 0
	DWMWCP_DONOTROUND = 1
)

// SetWindowCornerPreference sets the Windows 11 corner style of a top-level
// window (works on other processes' windows). Returns false on Windows 10,
// where the attribute doesn't exist.
func SetWindowCornerPreference(hwnd w32.HWND, pref uint32) bool {
	if procDwmSetWindowAttribute.Find() != nil {
		return false
	}
	hr, _, _ := procDwmSetWindowAttribute.Call(uintptr(hwnd), DWMWA_WINDOW_CORNER_PREFERENCE,
		uintptr(unsafe.Pointer(&pref)), unsafe.Sizeof(pref))
	return hr == 0
}
