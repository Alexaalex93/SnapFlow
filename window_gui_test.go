package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
	webview "github.com/jchv/go-webview2"
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
	o.Show(r, true)
	time.Sleep(400 * time.Millisecond)
	if !w32.IsWindowVisible(o.hwnd) {
		t.Fatal("overlay not visible after Show")
	}
	if wr := w32.GetWindowRect(o.hwnd); wr == nil || *wr != r {
		t.Fatalf("overlay at %+v, want %+v", wr, r)
	}

	// Retarget: glides to the new rect.
	r2 := w32.RECT{Left: work.Left + 300, Top: work.Top + 150, Right: work.Left + 900, Bottom: work.Top + 650}
	o.Show(r2, false)
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

// Square corners must work on a window owned by ANOTHER process (the real
// case). A red WinForms window is opened via PowerShell; the pixel just
// inside its bottom-left corner is outside the rounded shape (background
// shows through) until SnapFlow squares the corners, then it is red.
func TestGUI_SquareCornersCrossProcess(t *testing.T) {
	if os.Getenv("SNAPFLOW_GUI_TESTS") != "1" {
		t.Skip("set SNAPFLOW_GUI_TESTS=1 to run real-window tests")
	}
	w32ex.SetProcessDPIAwareBestEffort()
	const title = "SnapFlow corner test"
	script := `Add-Type -AssemblyName System.Windows.Forms;` +
		`$f=New-Object Windows.Forms.Form;$f.Text='` + title + `';$f.BackColor='Red';` +
		`$f.StartPosition='Manual';$f.Left=300;$f.Top=300;$f.Width=500;$f.Height=350;` +
		`$f.TopMost=$true;$t=New-Object Windows.Forms.Timer;$t.Interval=6000;$t.add_Tick({$f.Close()});$t.Start();` +
		`[Windows.Forms.Application]::Run($f)`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()

	var hwnd w32.HWND
	for i := 0; i < 100 && hwnd == 0; i++ {
		time.Sleep(100 * time.Millisecond)
		hwnd = w32.FindWindow("", title)
	}
	if hwnd == 0 {
		t.Fatal("test window did not appear")
	}
	time.Sleep(500 * time.Millisecond)

	f, _ := frameRect(hwnd)
	probe := func() (r, g, b byte) {
		hdc := w32.GetDC(0)
		defer w32.ReleaseDC(0, hdc)
		c, _, _ := gdi32DLL.NewProc("GetPixel").Call(uintptr(hdc), uintptr(f.Left+1), uintptr(f.Bottom-2))
		return byte(c), byte(c >> 8), byte(c >> 16)
	}
	isRed := func(r, g, b byte) bool { return r > 200 && g < 60 && b < 60 }

	if r, g, b := probe(); isRed(r, g, b) {
		t.Skip("corner already square (Windows 10 or rounding disabled) — nothing to verify")
	}
	appCfg = defaultConfig()
	squareCorners(hwnd)
	time.Sleep(300 * time.Millisecond)
	if r, g, b := probe(); !isRed(r, g, b) {
		t.Fatalf("corner pixel after squaring = (%d,%d,%d), want red", r, g, b)
	}
	roundCorners(hwnd)
	time.Sleep(300 * time.Millisecond)
	if r, g, b := probe(); isRed(r, g, b) {
		t.Fatal("corner still square after roundCorners")
	}
}

// Animated move of a window owned by another process: the real window is
// parked off-screen, the proxy thumbnail (red content, not black) is visible
// mid-flight, a retarget mid-flight works, and the window lands pixel-exact.
func TestGUI_WindowAnimationCrossProcess(t *testing.T) {
	if os.Getenv("SNAPFLOW_GUI_TESTS") != "1" {
		t.Skip("set SNAPFLOW_GUI_TESTS=1 to run real-window tests")
	}
	w32ex.SetProcessDPIAwareBestEffort()
	if !systemAnimationsEnabled() {
		t.Skip("Windows animation effects are off")
	}
	appCfg = defaultConfig()
	liveCfg.Store(appCfg)
	a, err := NewWindowAnimator()
	if err != nil {
		t.Fatal(err)
	}
	windowAnim = a
	defer func() { a.Close(); windowAnim = nil }()

	hwnd, stop := startRedWindow(t, "SnapFlow animation test")
	defer stop()

	from, _ := frameRect(hwnd)
	work, _ := monitorWork(w32.MonitorFromRect(&from, w32.MONITOR_DEFAULTTONEAREST))
	target := ActionLeftHalf.Calculate(work, from, appCfg)
	expected := extendFlushEdges(target, work)

	start := time.Now()
	if got, _ := placeWindow(hwnd, from, target); got != expected {
		t.Fatalf("placeWindow returned %+v, want expected %+v", got, expected)
	}
	if lf, _ := currentFrame(hwnd); lf != target {
		t.Fatalf("logical frame mid-flight = %+v, want %+v", lf, target)
	}

	time.Sleep(windowAnimDur / 2)
	if real, _ := frameRect(hwnd); w32.MonitorFromRect(&real, w32ex.MONITOR_DEFAULTTONULL) != 0 {
		t.Errorf("real window should be parked off-screen mid-flight, is at %+v", real)
	}
	e := easeOutCubic(progress(start, windowAnimDur, time.Now()))
	mid := lerpRect(toFrect(from), toFrect(expected), e).round()
	if r, g, b := screenPixel(mid.Left+mid.Width()/2, mid.Top+mid.Height()/2); !(r > 200 && g < 60 && b < 60) {
		t.Errorf("proxy thumbnail not visible mid-flight: pixel (%d,%d,%d), want red", r, g, b)
	}

	// Retarget mid-flight.
	target2 := ActionBottomRightSixth.Calculate(work, from, appCfg)
	placeWindow(hwnd, target, target2)
	time.Sleep(windowAnimDur + 400*time.Millisecond)

	got, _ := frameRect(hwnd)
	if want := extendFlushEdges(target2, work); !rectNear(got, want, 0) {
		t.Fatalf("landed at %+v, want %+v", got, want)
	}
	if w32.IsWindowVisible(a.proxy) {
		t.Error("proxy still visible after landing")
	}
	if a.inFlight(hwnd) {
		t.Error("window still marked in flight")
	}
}

func startRedWindow(t *testing.T, title string) (w32.HWND, func()) {
	t.Helper()
	script := `Add-Type -AssemblyName System.Windows.Forms;` +
		`$f=New-Object Windows.Forms.Form;$f.Text='` + title + `';$f.BackColor='Red';` +
		`$f.StartPosition='Manual';$f.Left=400;$f.Top=300;$f.Width=500;$f.Height=350;` +
		`$t=New-Object Windows.Forms.Timer;$t.Interval=8000;$t.add_Tick({$f.Close()});$t.Start();` +
		`[Windows.Forms.Application]::Run($f)`
	cmd := exec.Command("powershell", "-NoProfile", "-Command", script)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	var hwnd w32.HWND
	for i := 0; i < 100 && hwnd == 0; i++ {
		time.Sleep(100 * time.Millisecond)
		hwnd = w32.FindWindow("", title)
	}
	if hwnd == 0 {
		cmd.Process.Kill()
		t.Fatal("test window did not appear")
	}
	time.Sleep(400 * time.Millisecond)
	return hwnd, func() { cmd.Process.Kill() }
}

func screenPixel(x, y int32) (r, g, b byte) {
	hdc := w32.GetDC(0)
	defer w32.ReleaseDC(0, hdc)
	c, _, _ := gdi32DLL.NewProc("GetPixel").Call(uintptr(hdc), uintptr(x), uintptr(y))
	return byte(c), byte(c >> 8), byte(c >> 16)
}

// Pressing Save (and Cancel) in the real settings window must close it.
// Regression: Terminate() stopped the message loop without destroying the
// window, leaving a frozen "Saved ✓" window on screen.
func TestGUI_SettingsSaveAndCancelClose(t *testing.T) {
	if os.Getenv("SNAPFLOW_GUI_TESTS") != "1" {
		t.Skip("set SNAPFLOW_GUI_TESTS=1 to run real-window tests")
	}
	w32ex.SetProcessDPIAwareBestEffort()
	dir, _ := os.MkdirTemp("", "snapflow-settings-test")
	defer os.RemoveAll(dir) // best effort: WebView2 may still hold its lockfile
	appCfgStore = &ConfigStore{path: filepath.Join(dir, "config.json")}
	appCfg = defaultConfig()
	liveCfg.Store(appCfg)
	defer func() { settingsTestHook = nil }()

	for _, js := range []string{"save()", "goClose()"} {
		var view webview.WebView
		ready := make(chan struct{})
		settingsTestHook = func(w webview.WebView) { view = w; close(ready) }
		done := make(chan struct{})
		go func() {
			settingsThread() // exactly what openSettings runs
			close(done)
		}()
		select {
		case <-ready:
		case <-time.After(15 * time.Second):
			t.Fatal("settings window did not open (WebView2 missing?)")
		}
		hwnd := w32.HWND(settingsHWND.Load())
		time.Sleep(2500 * time.Millisecond) // let the page load
		view.Dispatch(func() { view.Eval(js) })
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatalf("%s: settings window loop never ended", js)
		}
		time.Sleep(200 * time.Millisecond)
		if w32.IsWindow(hwnd) {
			t.Fatalf("%s: settings window still exists (frozen) after closing", js)
		}
		settingsHWND.Store(0)
	}
	if _, err := os.Stat(appCfgStore.path); err != nil {
		t.Fatal("save() did not write the config file")
	}
}
