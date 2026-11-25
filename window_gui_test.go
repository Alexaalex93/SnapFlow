package main

import (
	"os"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

// Real-window placement tests. They briefly show a window on screen, so they
// only run when SNAPFLOW_GUI_TESTS=1 (skipped in CI):
//
//	$env:SNAPFLOW_GUI_TESTS=1; go test -run GUI -v .
func TestGUI_PlacementIsPixelExact(t *testing.T) {
	if os.Getenv("SNAPFLOW_GUI_TESTS") != "1" {
		t.Skip("set SNAPFLOW_GUI_TESTS=1 to run real-window tests")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	w32ex.SetProcessDPIAwareBestEffort()
	appCfg = defaultConfig()
	liveCfg.Store(appCfg)

	hwnd := createTestWindow(t)
	defer procDestroyWindow.Call(uintptr(hwnd))
	pump(150 * time.Millisecond)

	work, ok := monitorWork(w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST))
	if !ok {
		t.Fatal("no work area")
	}

	check := func(name string, action WindowAction) {
		t.Helper()
		target := action.Calculate(work, work, appCfg)
		if _, err := resizeToRect(hwnd, target); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		pump(250 * time.Millisecond) // let the delayed verification run too
		got, _ := frameRect(hwnd)
		want := extendFlushEdges(target, work)
		if !rectNear(got, want, 0) {
			t.Errorf("%s: frame %+v, want %+v", name, got, want)
		}
	}

	check("left half", ActionLeftHalf)
	check("right half", ActionRightHalf)
	check("center third", ActionCenterThird)
	check("bottom-right sixth", ActionBottomRightSixth)

	// From a maximised window (different border metrics) straight into a snap.
	w32.ShowWindow(hwnd, w32.SW_MAXIMIZE)
	pump(300 * time.Millisecond)
	check("maximized → first two thirds", ActionFirstTwoThirds)

	// Center must not grow the window by the invisible border each press.
	before, _ := frameRect(hwnd)
	for i := 0; i < 3; i++ {
		executeAction(hwnd, ActionCenter)
		pump(50 * time.Millisecond)
	}
	after, _ := frameRect(hwnd)
	if after.Width() != before.Width() || after.Height() != before.Height() {
		t.Errorf("Center changed the size: %dx%d → %dx%d", before.Width(), before.Height(), after.Width(), after.Height())
	}

	// Restore brings back the original free-floating size.
	free := w32.RECT{Left: work.Left + 100, Top: work.Top + 100, Right: work.Left + 900, Bottom: work.Top + 700}
	w32.ShowWindow(hwnd, w32.SW_RESTORE)
	resizeToRect(hwnd, free)
	pump(250 * time.Millisecond)
	windowHistory.Forget(hwnd)
	executeAction(hwnd, ActionLeftHalf)
	executeAction(hwnd, ActionTopLeft)
	pump(100 * time.Millisecond)
	executeAction(hwnd, ActionRestore)
	pump(250 * time.Millisecond)
	if got, _ := frameRect(hwnd); !rectNear(got, free, 0) {
		t.Errorf("restore: frame %+v, want %+v", got, free)
	}

	// Repeated Left Half cycles ½ → ⅔ → ⅓.
	windowHistory.Forget(hwnd)
	want := []WindowAction{ActionLeftHalf, ActionFirstTwoThirds, ActionFirstThird, ActionLeftHalf}
	for i, a := range want {
		executeAction(hwnd, ActionLeftHalf)
		pump(60 * time.Millisecond)
		got, _ := frameRect(hwnd)
		if exp := extendFlushEdges(a.Calculate(work, work, appCfg), work); !rectNear(got, exp, 0) {
			t.Errorf("cycle press %d: frame %+v, want %+v", i+1, got, exp)
		}
	}
}

func TestGUI_OverlayShowsAndHides(t *testing.T) {
	if os.Getenv("SNAPFLOW_GUI_TESTS") != "1" {
		t.Skip("set SNAPFLOW_GUI_TESTS=1 to run real-window tests")
	}
	w32ex.SetProcessDPIAwareBestEffort()
	o, err := NewOverlayRenderer()
	if err != nil {
		t.Fatal(err)
	}
	defer o.Close()

	work, _ := monitorWork(w32ex.MonitorFromPoint(w32.POINT{X: 0, Y: 0}, w32ex.MONITOR_DEFAULTTOPRIMARY))
	r := w32.RECT{Left: work.Left + 100, Top: work.Top + 100, Right: work.Left + 700, Bottom: work.Top + 500}
	o.Show(r)
	time.Sleep(400 * time.Millisecond)
	if !w32.IsWindowVisible(o.hwnd) {
		t.Fatal("overlay not visible after Show")
	}
	if wr := w32.GetWindowRect(o.hwnd); wr == nil || *wr != r {
		t.Fatalf("overlay at %+v, want %+v", wr, r)
	}

	// Retarget: glides to the new rect.
	r2 := w32.RECT{Left: work.Left + 300, Top: work.Top + 150, Right: work.Left + 900, Bottom: work.Top + 650}
	o.Show(r2)
	time.Sleep(400 * time.Millisecond)
	if wr := w32.GetWindowRect(o.hwnd); wr == nil || *wr != r2 {
		t.Fatalf("overlay at %+v after retarget, want %+v", wr, r2)
	}

	o.Hide()
	time.Sleep(450 * time.Millisecond)
	if w32.IsWindowVisible(o.hwnd) {
		t.Fatal("overlay still visible after Hide")
	}
}

func createTestWindow(t *testing.T) w32.HWND {
	t.Helper()
	const wsOverlappedWindow = 0x00CF0000
	const wsVisible = 0x10000000
	cls, _ := syscall.UTF16PtrFromString("Static")
	title, _ := syscall.UTF16PtrFromString("SnapFlow placement test")
	r, _, e := procCreateWindowExW.Call(0,
		uintptr(unsafe.Pointer(cls)), uintptr(unsafe.Pointer(title)),
		wsOverlappedWindow|wsVisible, 200, 200, 800, 600, 0, 0, 0, 0)
	if r == 0 {
		t.Fatalf("CreateWindowEx: %v", e)
	}
	return w32.HWND(r)
}

func pump(d time.Duration) {
	end := time.Now().Add(d)
	var m w32.MSG
	for time.Now().Before(end) {
		for peekMessage(&m) {
			w32.TranslateMessage(&m)
			w32.DispatchMessage(&m)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
