package w32ex

import (
	"sync"
	"syscall"
	"unsafe"

	"github.com/gonutz/w32/v2"
)

const (
	EVENT_SYSTEM_MOVESIZESTART  = 0x000A
	EVENT_SYSTEM_MOVESIZEEND    = 0x000B
	EVENT_OBJECT_LOCATIONCHANGE = 0x800B

	OBJID_WINDOW = 0

	WINEVENT_OUTOFCONTEXT   = 0
	WINEVENT_SKIPOWNPROCESS = 2

	MONITOR_DEFAULTTONULL    = 0
	MONITOR_DEFAULTTOPRIMARY = 1
	MONITOR_DEFAULTTONEAREST = 2

	LWA_ALPHA = 0x2
)

var (
	procSetWinEventHook            = user32.NewProc("SetWinEventHook")
	procUnhookWinEvent             = user32.NewProc("UnhookWinEvent")
	procGetCursorPos               = user32.NewProc("GetCursorPos")
	procMonitorFromPoint           = user32.NewProc("MonitorFromPoint")
	procSetLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
)

type WinEventHook uintptr

type WinEventProc func(hook WinEventHook, event uint32, hwnd w32.HWND, idObject int32, idChild int32, eventThread uint32, eventTime uint32)

// Go can only create a limited number of syscall callbacks per process and
// never frees them. All hooks therefore share ONE native callback that
// dispatches to the Go handler registered for the hook handle. This lets
// SnapFlow install/remove short-lived hooks (one per drag) indefinitely.
var (
	winEventMu       sync.RWMutex
	winEventHandlers = map[WinEventHook]WinEventProc{}
	winEventOnce     sync.Once
	winEventCallback uintptr
)

func sharedWinEventCallback() uintptr {
	winEventOnce.Do(func() {
		winEventCallback = syscall.NewCallback(func(hook, event, hwnd, idObject, idChild, eventThread, eventTime uintptr) uintptr {
			winEventMu.RLock()
			cb := winEventHandlers[WinEventHook(hook)]
			winEventMu.RUnlock()
			if cb != nil {
				cb(WinEventHook(hook), uint32(event), w32.HWND(hwnd), int32(idObject), int32(idChild), uint32(eventThread), uint32(eventTime))
			}
			return 0
		})
	})
	return winEventCallback
}

// SetWinEventHook installs a hook. process/thread = 0 means "all".
// Out-of-context callbacks are delivered on the calling thread, which must
// pump messages.
func SetWinEventHook(eventMin, eventMax uint32, process, thread uint32, flags uint32, cb WinEventProc) WinEventHook {
	if cb == nil {
		return 0
	}
	// Hold the lock across installation so an event arriving immediately
	// (same thread, so only possible re-entrantly) finds its handler.
	winEventMu.Lock()
	defer winEventMu.Unlock()
	r1, _, _ := procSetWinEventHook.Call(
		uintptr(eventMin),
		uintptr(eventMax),
		0,
		sharedWinEventCallback(),
		uintptr(process),
		uintptr(thread),
		uintptr(flags),
	)
	h := WinEventHook(r1)
	if h != 0 {
		winEventHandlers[h] = cb
	}
	return h
}

func UnhookWinEvent(h WinEventHook) bool {
	if h == 0 {
		return false
	}
	r1, _, _ := procUnhookWinEvent.Call(uintptr(h))
	winEventMu.Lock()
	delete(winEventHandlers, h)
	winEventMu.Unlock()
	return r1 != 0
}

func GetCursorPos() (w32.POINT, bool) {
	var pt w32.POINT
	r1, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	return pt, r1 != 0
}

func MonitorFromPoint(pt w32.POINT, flags uint32) w32.HMONITOR {
	r1, _, _ := procMonitorFromPoint.Call(*(*uintptr)(unsafe.Pointer(&pt)), uintptr(flags))
	return w32.HMONITOR(r1)
}

func SetLayeredWindowAttributes(hwnd w32.HWND, colorKey uint32, alpha byte, flags uint32) bool {
	r1, _, _ := procSetLayeredWindowAttributes.Call(uintptr(hwnd), uintptr(colorKey), uintptr(alpha), uintptr(flags))
	return r1 != 0
}
