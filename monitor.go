package main

import (
	"sync"
	"syscall"

	"github.com/gonutz/w32/v2"
)

// EnumDisplayMonitors calls back synchronously on the calling thread, so one
// shared native callback + a mutex-guarded collector is enough. (Creating a
// new syscall callback per call leaks: Go caps them and never frees them, so
// the old code would eventually panic after a few thousand monitor switches.)
var (
	enumMonMu       sync.Mutex
	enumMonCollect  []w32.HMONITOR
	enumMonCallback = syscall.NewCallback(func(h, _, _, _ uintptr) uintptr {
		enumMonCollect = append(enumMonCollect, w32.HMONITOR(h))
		return 1
	})
)

// Monitors returns all display monitors in system enumeration order.
func Monitors() []w32.HMONITOR {
	enumMonMu.Lock()
	defer enumMonMu.Unlock()
	enumMonCollect = nil
	w32.EnumDisplayMonitors(0, nil, enumMonCallback, 0)
	out := enumMonCollect
	enumMonCollect = nil
	return out
}

// monitorWork returns the work area (screen minus taskbar) of a monitor.
func monitorWork(mon w32.HMONITOR) (w32.RECT, bool) {
	if mon == 0 {
		return w32.RECT{}, false
	}
	var info w32.MONITORINFO
	if !w32.GetMonitorInfo(mon, &info) {
		return w32.RECT{}, false
	}
	if info.RcWork.Width() <= 0 || info.RcWork.Height() <= 0 {
		return w32.RECT{}, false
	}
	return info.RcWork, true
}
