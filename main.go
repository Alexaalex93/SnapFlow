package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"unsafe"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

const (
	appName    = "SnapFlow"
	appVersion = "1.0.0"

	wmOpenSettings = w32.WM_APP + 1
	wmDoQuit       = w32.WM_APP + 2
	wmOpenConfig   = w32.WM_APP + 3
)

// cfgUpdateCh carries config updates from the settings WebView goroutine to the
// main OS thread, where appCfg is also read by WinEvent callbacks. This avoids
// a data race: the channel write (settings goroutine) and the channel read
// (main thread msgLoop via wmApplyCfg) are the only two access paths.
var cfgUpdateCh = make(chan *AppConfig, 1)

const wmApplyCfg = w32.WM_APP + 4

var (
	appCfg       *AppConfig
	appCfgStore  *ConfigStore
	dragMgr      *DragManager
	overlay      *OverlayRenderer
	mainThreadID uint32
)

func main() {
	runtime.LockOSThread()
	mainThreadID = w32ex.GetCurrentThreadID()

	// Single-instance guard: exit if already running.
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	mutexName, _ := syscall.UTF16PtrFromString("SnapFlow_SingleInstance_Mutex")
	kernel32.NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(mutexName)))
	if syscall.GetLastError() == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		w32.MessageBox(0, "SnapFlow is already running.\nCheck the system tray.", appName, w32.MB_ICONINFORMATION|w32.MB_OK)
		os.Exit(0)
	}

	if !w32ex.SetProcessDPIAwareBestEffort() {
		fmt.Fprintln(os.Stderr, "warn: DPI awareness failed")
	}
	if err := startup(); err != nil {
		fmt.Fprintf(os.Stderr, "startup: %v\n", err)
		os.Exit(1)
	}

	registerAllHotKeys()

	// initTray runs on the main thread so the tray window's messages
	// are delivered by our msgLoop (same OS thread = same message queue).
	if err := initTray(); err != nil {
		fmt.Fprintf(os.Stderr, "warn: tray: %v\n", err)
	}

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		removeTrayIcon()
		postToMain(wmDoQuit, 0, 0)
	}()

	if err := msgLoop(); err != nil {
		fmt.Fprintf(os.Stderr, "message loop: %v\n", err)
		os.Exit(1)
	}
}

func startup() error {
	var err error
	appCfgStore, err = NewConfigStore()
	if err != nil {
		return fmt.Errorf("config store: %w", err)
	}
	appCfg, err = appCfgStore.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	_ = appCfgStore.Save(appCfg)

	overlay, err = NewOverlayRenderer()
	if err != nil {
		return fmt.Errorf("overlay: %w", err)
	}

	gest := NewGestureInterpreter(appCfg)
	dragMgr = NewDragManager(gest, overlay)
	return dragMgr.Start()
}

// registerAllHotKeys registers Rectangle's default shortcuts translated to Windows.
//
// Mapping:  macOS Ctrl+Option → Windows Ctrl+Alt
//
//	macOS Ctrl+Option+Cmd → Windows Ctrl+Alt+Win
//
// NOTE: Ctrl+Alt+Arrow is often grabbed by Intel/AMD/NVIDIA display-rotation
// shortcuts. If registration fails the user is told which shortcuts conflicted
// and the rest still work. The graphics-driver conflict can be fixed in the
// GPU control panel ("Disable hotkeys" or "Rotate display" shortcut).
func registerAllHotKeys() {
	// Primary modifier set: Ctrl+Alt (= macOS Ctrl+Option)
	ca := MOD_CONTROL | MOD_ALT | MOD_NOREPEAT
	cas := MOD_CONTROL | MOD_ALT | MOD_SHIFT | MOD_NOREPEAT
	caw := MOD_CONTROL | MOD_ALT | MOD_WIN | MOD_NOREPEAT

	const (
		vkLeft   = 0x25
		vkRight  = 0x27
		vkUp     = 0x26
		vkDown   = 0x28
		vkReturn = 0x0D
		// macOS "Delete" key = Backspace on Windows (0x08).
		// Ctrl+Alt+Delete is system-reserved by Windows and can never be registered.
		vkBack  = 0x08
		vkEqual = 0xBB
		vkMinus = 0xBD // = and -

		vkC = 0x43
		vkD = 0x44
		vkE = 0x45
		vkF = 0x46
		vkG = 0x47
		vkI = 0x49
		vkJ = 0x4A
		vkK = 0x4B
		vkR = 0x52
		vkT = 0x54
		vkU = 0x55
	)

	hks := []HotKey{
		// ── Halves ───────────────────────────────────────────────────────────
		// Ctrl+Alt+Arrow is commonly taken by GPU rotation shortcuts.
		// If they fail the user sees which ones and can fix in GPU settings.
		{id: 1, mod: ca, vk: vkLeft, label: "Left Half", action: ActionLeftHalf},
		{id: 2, mod: ca, vk: vkRight, label: "Right Half", action: ActionRightHalf},
		{id: 3, mod: ca, vk: vkUp, label: "Top Half", action: ActionTopHalf},
		{id: 4, mod: ca, vk: vkDown, label: "Bottom Half", action: ActionBottomHalf},

		// ── Quarters ─────────────────────────────────────────────────────────
		{id: 10, mod: ca, vk: vkU, label: "Top Left", action: ActionTopLeft},
		{id: 11, mod: ca, vk: vkI, label: "Top Right", action: ActionTopRight},
		{id: 12, mod: ca, vk: vkJ, label: "Bottom Left", action: ActionBottomLeft},
		{id: 13, mod: ca, vk: vkK, label: "Bottom Right", action: ActionBottomRight},

		// ── Thirds ───────────────────────────────────────────────────────────
		{id: 20, mod: ca, vk: vkD, label: "First Third", action: ActionFirstThird},
		{id: 21, mod: ca, vk: vkE, label: "First Two Thirds", action: ActionFirstTwoThirds},
		{id: 22, mod: ca, vk: vkF, label: "Center Third", action: ActionCenterThird},
		{id: 23, mod: ca, vk: vkT, label: "Last Two Thirds", action: ActionLastTwoThirds},
		{id: 24, mod: ca, vk: vkG, label: "Last Third", action: ActionLastThird},
		{id: 25, mod: ca, vk: vkR, label: "Center Two Thirds", action: ActionCenterTwoThirds},

		// ── Maximize ─────────────────────────────────────────────────────────
		{id: 30, mod: ca, vk: vkReturn, label: "Maximize", action: ActionMaximize},
		{id: 31, mod: cas, vk: vkReturn, label: "Almost Maximize", action: ActionAlmostMaximize},
		{id: 32, mod: cas, vk: vkUp, label: "Maximize Height", action: ActionMaximizeHeight},

		// ── Center / Resize / Restore ─────────────────────────────────────────
		{id: 40, mod: ca, vk: vkC, label: "Center", action: ActionCenter},
		{id: 50, mod: ca, vk: vkEqual, label: "Larger", action: ActionLarger},
		{id: 51, mod: ca, vk: vkMinus, label: "Smaller", action: ActionSmaller},
		{id: 60, mod: ca, vk: vkBack, label: "Restore (Ctrl+Alt+Backspace)", action: ActionRestore},

		// ── Multi-monitor ────────────────────────────────────────────────────
		{id: 70, mod: caw, vk: vkRight, label: "Next Display", action: ActionNextDisplay},
		{id: 71, mod: caw, vk: vkLeft, label: "Previous Display", action: ActionPreviousDisplay},
	}

	var failed []HotKey
	for _, hk := range hks {
		hk.callback = makeActionCallback(hk.action)
		if !RegisterHotKey(hk) {
			failed = append(failed, hk)
		}
	}
	if len(failed) > 0 {
		showShortcutConflictDialog(failed)
	}
}

// makeActionCallback returns the callback function for a WindowAction hotkey.
func makeActionCallback(action WindowAction) func() {
	return func() {
		hwnd := w32.GetForegroundWindow()
		if !isZonableWindow(hwnd) {
			return
		}
		executeAction(hwnd, action)
	}
}

// executeAction applies a WindowAction to the given window.
// This is the central dispatch — translates Rectangle's WindowManager.execute().
func executeAction(hwnd w32.HWND, action WindowAction) {
	switch action {
	case ActionMaximize:
		maximize(hwnd)
		return

	case ActionRestore:
		if prev, ok := windowHistory.Pop(hwnd); ok {
			resizeToRect(hwnd, prev)
		}
		return

	case ActionNextDisplay:
		moveToDisplay(hwnd, true)
		return
	case ActionPreviousDisplay:
		moveToDisplay(hwnd, false)
		return

	case ActionDisplayOne, ActionDisplayTwo, ActionDisplayThree,
		ActionDisplayFour, ActionDisplayFive, ActionDisplaySix,
		ActionDisplaySeven, ActionDisplayEight, ActionDisplayNine:
		idx := int(action - ActionDisplayOne)
		moveToDisplayByIndex(hwnd, idx)
		return
	}

	mon := w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST)
	var monInfo w32.MONITORINFO
	if !w32.GetMonitorInfo(mon, &monInfo) {
		return
	}
	work := monInfo.RcWork
	cur := w32.GetWindowRect(hwnd)
	if cur == nil {
		return
	}

	// Save to history before modifying.
	windowHistory.Save(hwnd, *cur)

	target := action.Calculate(work, *cur, appCfg)
	if _, err := resizeToRect(hwnd, target); err != nil {
		fmt.Printf("warn: action %d: %v\n", action, err)
	}
}

func postToMain(msg uint32, wParam, lParam uintptr) {
	w32ex.PostThreadMessage(mainThreadID, msg, wParam, lParam)
}
