package w32ex

import (
	"syscall"
	"unsafe"

	"github.com/gonutz/w32/v2"
)

const (
	EVENT_SYSTEM_MOVESIZESTART   = 0x000A
	EVENT_SYSTEM_MOVESIZEEND     = 0x000B
	EVENT_OBJECT_LOCATIONCHANGE  = 0x800B

	OBJID_WINDOW = 0

	WINEVENT_OUTOFCONTEXT   = 0
	WINEVENT_SKIPOWNPROCESS = 2

	MONITOR_DEFAULTTONULL    = 0
	MONITOR_DEFAULTTOPRIMARY = 1
	MONITOR_DEFAULTTONEAREST = 2

	LWA_ALPHA = 0x2
)

type WinEventHook uintptr

type WinEventProc func(hook WinEventHook, event uint32, hwnd w32.HWND, idObject int32, idChild int32, eventThread uint32, eventTime uint32)

var winEventCallbacks []WinEventProc

func SetWinEventHook(eventMin, eventMax uint32, process, thread uint32, flags uint32, cb WinEventProc) WinEventHook {
	if cb == nil {
		return 0
	}
	winEventCallbacks = append(winEventCallbacks, cb)
	proc := syscall.NewCallback(func(hook, event, hwnd, idObject, idChild, eventThread, eventTime uintptr) uintptr {
		cb(WinEventHook(hook), uint32(event), w32.HWND(hwnd), int32(idObject), int32(idChild), uint32(eventThread), uint32(eventTime))
		return 0
	})
	r1, _, _ := user32.NewProc("SetWinEventHook").Call(
		uintptr(eventMin),
		uintptr(eventMax),
		0,
		proc,
		uintptr(process),
		uintptr(thread),
		uintptr(flags),
	)
	return WinEventHook(r1)
}

func UnhookWinEvent(h WinEventHook) bool {
	r1, _, _ := user32.NewProc("UnhookWinEvent").Call(uintptr(h))
	return r1 != 0
}

func GetCursorPos() (w32.POINT, bool) {
	var pt w32.POINT
	r1, _, _ := user32.NewProc("GetCursorPos").Call(uintptr(unsafe.Pointer(&pt)))
	return pt, r1 != 0
}

func MonitorFromPoint(pt w32.POINT, flags uint32) w32.HMONITOR {
	r1, _, _ := user32.NewProc("MonitorFromPoint").Call(*(*uintptr)(unsafe.Pointer(&pt)), uintptr(flags))
	return w32.HMONITOR(r1)
}

func SetLayeredWindowAttributes(hwnd w32.HWND, colorKey uint32, alpha byte, flags uint32) bool {
	r1, _, _ := user32.NewProc("SetLayeredWindowAttributes").Call(uintptr(hwnd), uintptr(colorKey), uintptr(alpha), uintptr(flags))
	return r1 != 0
}
