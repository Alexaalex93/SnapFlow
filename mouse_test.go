package main

// Opt-in tests that drive the real mouse.

import (
	"os"
	"runtime"
	"testing"
	"time"
	"unsafe"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

type mouseInput struct {
	typ       uint32
	_         uint32
	dx, dy    int32
	mouseData uint32
	flags     uint32
	time      uint32
	extra     uintptr
}

var procSendInput = user32DLL.NewProc("SendInput")

func moveAbs(x, y int32) {
	vx := int32(w32.GetSystemMetrics(76)) // SM_XVIRTUALSCREEN
	vy := int32(w32.GetSystemMetrics(77))
	vw := int32(w32.GetSystemMetrics(78))
	vh := int32(w32.GetSystemMetrics(79))
	in := mouseInput{typ: 0, flags: 0x0001 | 0x8000 | 0x4000, // MOVE|ABSOLUTE|VIRTUALDESK
		dx: int32((int64(x-vx)*65535 + int64(vw)/2) / int64(vw-1)),
		dy: int32((int64(y-vy)*65535 + int64(vh)/2) / int64(vh-1))}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func sendMouse(flags uint32) {
	in := mouseInput{typ: 0, flags: flags}
	procSendInput.Call(1, uintptr(unsafe.Pointer(&in)), unsafe.Sizeof(in))
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	s := ""
	for i > 0 {
		s = string(rune('0'+i%10)) + s
		i /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
func btoa(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
func rectStr(r w32.RECT) string {
	return "[" + itoa(int(r.Left)) + "," + itoa(int(r.Top)) + " " + itoa(int(r.Width())) + "x" + itoa(int(r.Height())) + "]"
}

// Real mouse input (SendInput): drag a MAXIMISED window by its title bar to
// the left screen edge and drop it — with Windows Snap off this must
// un-maximise it at normal size under the cursor and end as Left Half.
// It moves the real mouse, so it only runs with SNAPFLOW_MOUSE_TESTS=1:
//
//	$env:SNAPFLOW_MOUSE_TESTS=1; go test -run Mouse -v .
func TestMouse_DragMaximizedToLeftHalf(t *testing.T) {
	if os.Getenv("SNAPFLOW_MOUSE_TESTS") != "1" {
		t.Skip()
	}
	runtime.LockOSThread()
	w32ex.SetProcessDPIAwareBestEffort()
	appCfg = defaultConfig()
	appCfg.AnimateWindows = false
	liveCfg.Store(appCfg)

	hwnd, stop := startRedWindow(t, "SnapFlow drag-to-half probe")
	defer stop()
	w32.ShowWindow(hwnd, w32.SW_MAXIMIZE)
	time.Sleep(600 * time.Millisecond)
	maxF, _ := frameRect(hwnd)
	work, _ := monitorWork(w32.MonitorFromRect(&maxF, w32.MONITOR_DEFAULTTONEAREST))

	var err error
	overlay, err = NewOverlayRenderer()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { overlay.Close(); overlay = nil }()
	dragMgr = NewDragManager(NewGestureInterpreter(appCfg), overlay)
	if err := dragMgr.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { dragMgr.Stop(); dragMgr = nil }()
	dragRestore = startDragRestorer()
	defer func() { dragRestore.stop(); dragRestore = nil }()
	procPostThreadMessageW.Call(uintptr(dragRestore.threadID), wmDragRestoreHook, 1, 0)
	pump(200 * time.Millisecond)

	x0, y0 := maxF.Left+maxF.Width()/3, maxF.Top+12
	tx, ty := work.Left+2, work.Top+work.Height()/2
	moveAbs(x0, y0)
	pump(100 * time.Millisecond)
	sendMouse(0x0002)
	const steps = 40
	for i := 1; i <= steps; i++ {
		moveAbs(x0+(tx-x0)*int32(i)/steps, y0+(ty-y0)*int32(i)/steps)
		pump(15 * time.Millisecond)
	}
	pump(300 * time.Millisecond)
	sendMouse(0x0004)
	pump(600 * time.Millisecond)

	got, _ := frameRect(hwnd)
	want := extendFlushEdges(ActionLeftHalf.Calculate(work, work, appCfg), work)
	t.Logf("zoomed=%v frame=%s want=%s", w32ex.IsZoomed(hwnd), rectStr(got), rectStr(want))
	if !rectNear(got, want, 1) {
		t.Fatalf("maximised window dragged to the left edge should end as Left Half")
	}
}
