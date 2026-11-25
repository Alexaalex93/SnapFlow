package main

import (
	"runtime"
	"sync/atomic"
	"syscall"
	"unsafe"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

// ── Drag a maximised window to un-maximise it ─────────────────────────────────
//
// Dragging a maximised window's title bar normally restores it and lets you
// move it — but that is part of Windows Snap, and SnapFlow recommends turning
// Windows Snap off. With it off, a maximised window simply doesn't move: you
// had to double-click it first.
//
// SnapFlow re-implements it: a low-level mouse hook (only installed while
// Windows Snap is off and drag snapping is on) notices a press on a
// maximised window followed by a drag. Then, off the hook:
//   - WM_NCHITTEST confirms the press was on the title bar (HTCAPTION), so
//     dragging a Chrome tab, a scrollbar or any content is left alone,
//   - the window is restored to its normal size *under the cursor*, keeping
//     the cursor at the same relative spot of the title bar,
//   - WM_NCLBUTTONDOWN(HTCAPTION) starts the regular system move loop, so the
//     drag continues seamlessly with the button still held — and SnapFlow's
//     snap zones work as usual.
//
// The hook callback itself only does cheap, non-blocking calls; everything
// that talks to the target window runs on the hook thread's message loop.

const (
	whMouseLL       = 14
	wmMouseMove     = 0x0200
	wmLButtonDown   = 0x0201
	wmLButtonUp     = 0x0202
	wmNCLButtonDown = 0x00A1
	htCaption       = 2

	wmDragRestoreGo   = w32.WM_APP + 60
	wmDragRestoreHook = w32.WM_APP + 61 // wParam 1 = install, 0 = remove
)

var (
	procSetWindowsHookExW   = user32DLL.NewProc("SetWindowsHookExW")
	procUnhookWindowsHookEx = user32DLL.NewProc("UnhookWindowsHookEx")
	procCallNextHookEx      = user32DLL.NewProc("CallNextHookEx")
	procWindowFromPoint     = user32DLL.NewProc("WindowFromPoint")
)

type msllHookStruct struct {
	pt        w32.POINT
	mouseData uint32
	flags     uint32
	time      uint32
	extra     uintptr
}

type dragRestorer struct {
	threadID uint32
	hook     uintptr
	enabled  atomic.Bool

	// Hook-thread-only state.
	armed  bool
	fired  bool
	hwnd   w32.HWND
	downPt w32.POINT
	thresh int32
}

var dragRestore *dragRestorer

// The hook callback must be a single, process-lifetime syscall callback.
var dragRestoreHookProc = syscall.NewCallback(func(nCode, wParam uintptr, info *msllHookStruct) uintptr {
	if d := dragRestore; d != nil && int32(nCode) >= 0 && d.enabled.Load() {
		d.onMouse(uint32(wParam), info)
	}
	r, _, _ := procCallNextHookEx.Call(0, nCode, wParam, uintptr(unsafe.Pointer(info)))
	return r
})

func startDragRestorer() *dragRestorer {
	d := &dragRestorer{thresh: int32(w32.GetSystemMetrics(w32.SM_CXDRAG))}
	if d.thresh < 4 {
		d.thresh = 4
	}
	ready := make(chan struct{})
	go d.run(ready)
	<-ready
	return d
}

func (d *dragRestorer) run(ready chan<- struct{}) {
	runtime.LockOSThread()
	d.threadID = w32ex.GetCurrentThreadID()
	var m w32.MSG
	peekMessage(&m) // create the message queue before anyone posts to it
	close(ready)
	for {
		r := w32.GetMessage(&m, 0, 0, 0)
		if r == 0 || r == -1 {
			d.setHook(false)
			return
		}
		switch {
		case m.Hwnd == 0 && m.Message == wmDragRestoreHook:
			d.setHook(m.WParam != 0)
		case m.Hwnd == 0 && m.Message == wmDragRestoreGo:
			d.restoreAndDrag(d.hwnd, d.downPt, true)
		default:
			w32.TranslateMessage(&m)
			w32.DispatchMessage(&m)
		}
	}
}

func (d *dragRestorer) setHook(on bool) {
	if on && d.hook == 0 {
		instance, _, _ := procGetModuleHandleW.Call(0)
		d.hook, _, _ = procSetWindowsHookExW.Call(whMouseLL, dragRestoreHookProc, instance, 0)
	} else if !on && d.hook != 0 {
		procUnhookWindowsHookEx.Call(d.hook)
		d.hook = 0
	}
	d.enabled.Store(d.hook != 0)
}

// refresh installs the hook only when it is needed: Windows Snap off (it
// does this natively otherwise) and drag snapping on. Safe from any goroutine.
func (d *dragRestorer) refresh() {
	if d == nil {
		return
	}
	need := !WindowsSnapEnabled()
	if c := liveCfg.Load(); c != nil && !c.SnapByDragging {
		need = false
	}
	v := uintptr(0)
	if need {
		v = 1
	}
	procPostThreadMessageW.Call(uintptr(d.threadID), wmDragRestoreHook, v, 0)
}

func (d *dragRestorer) stop() {
	if d != nil {
		procPostThreadMessageW.Call(uintptr(d.threadID), wmQuit, 0, 0)
	}
}

// onMouse runs inside the low-level hook: keep it cheap and non-blocking.
func (d *dragRestorer) onMouse(msg uint32, info *msllHookStruct) {
	switch msg {
	case wmLButtonDown:
		d.armed, d.fired = false, false
		h, _, _ := procWindowFromPoint.Call(*(*uintptr)(unsafe.Pointer(&info.pt)))
		if h == 0 {
			return
		}
		root := w32ex.GetAncestor(w32.HWND(h), w32ex.GA_ROOT)
		if root != 0 && w32ex.IsZoomed(root) {
			d.armed, d.hwnd, d.downPt = true, root, info.pt
		}
	case wmMouseMove:
		if d.armed && !d.fired {
			dx, dy := info.pt.X-d.downPt.X, info.pt.Y-d.downPt.Y
			if dx*dx+dy*dy >= d.thresh*d.thresh {
				d.fired = true
				procPostThreadMessageW.Call(uintptr(d.threadID), wmDragRestoreGo, 0, 0)
			}
		}
	case wmLButtonUp:
		d.armed = false
	}
}

// restoreAndDrag un-maximises hwnd under the cursor and hands the drag to
// the system move loop. interactive=false (tests only) skips the mouse-button
// checks and does not start the move loop.
func (d *dragRestorer) restoreAndDrag(hwnd w32.HWND, downPt w32.POINT, interactive bool) bool {
	if hwnd == 0 || !w32ex.IsZoomed(hwnd) || !isZonableWindow(hwnd) {
		return false
	}
	const vkLButton = 0x01
	if interactive && w32.GetAsyncKeyState(vkLButton)&0x8000 == 0 {
		return false // already released: it was a click, not a drag
	}
	lp := uintptr(uint16(int16(downPt.X))) | uintptr(uint16(int16(downPt.Y)))<<16
	if hit, ok := w32ex.SendMessageTimeout(hwnd, w32ex.WM_NCHITTEST, 0, lp, 100); !ok || hit != htCaption {
		return false // tab, button, content… not ours to handle
	}
	maxFrame, ok := frameRect(hwnd)
	if !ok || maxFrame.Width() <= 0 {
		return false
	}

	w32.ShowWindow(hwnd, w32.SW_RESTORE)
	normal, ok := frameRect(hwnd)
	if !ok {
		return false
	}
	cursor, _ := w32ex.GetCursorPos()
	nf := grabFrame(maxFrame, normal, downPt, cursor)
	if in, ok := windowInsets(hwnd); ok {
		setWindowFrame(hwnd, nf, in, 0)
	}

	if !interactive || w32.GetAsyncKeyState(vkLButton)&0x8000 == 0 {
		return true // released meanwhile: restored, but nothing to drag
	}
	cp := uintptr(uint16(int16(cursor.X))) | uintptr(uint16(int16(cursor.Y)))<<16
	w32.PostMessage(hwnd, wmNCLButtonDown, htCaption, cp)
	return true
}

// grabFrame positions a frame of normal's size so that the cursor sits at
// the same relative horizontal spot and the same height of the title bar as
// where the maximised window was grabbed. Pure geometry.
func grabFrame(maxFrame, normal w32.RECT, downPt, cursor w32.POINT) w32.RECT {
	w, h := normal.Width(), normal.Height()
	rel := float64(downPt.X-maxFrame.Left) / float64(maxFrame.Width())
	if rel < 0 {
		rel = 0
	} else if rel > 1 {
		rel = 1
	}
	left := cursor.X - int32(rel*float64(w))
	offY := downPt.Y - maxFrame.Top
	if offY < 4 {
		offY = 4
	}
	if offY > h/2 {
		offY = h / 2
	}
	top := cursor.Y - offY
	return w32.RECT{Left: left, Top: top, Right: left + w, Bottom: top + h}
}
