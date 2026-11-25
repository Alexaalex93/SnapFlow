package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

// appVersion is overwritten at build time via:
//
//	go build -ldflags "-X main.appVersion=1.2.3"
//
// Must be a var (not const) for ldflags -X to work.
var appVersion = "dev"

const (
	appName = "SnapFlow"

	wmOpenSettings   = w32.WM_APP + 1
	wmDoQuit         = w32.WM_APP + 2
	wmOpenConfig     = w32.WM_APP + 3
	wmApplyCfg       = w32.WM_APP + 4
	wmSuspendHotkeys = w32.WM_APP + 5
)

var errHookInstall = errors.New("failed to install WinEvent hook")

// liveCfg mirrors appCfg for goroutines other than the main thread
// (settings window, update checker, placement verification). appCfg itself
// is only read/written on the main thread.
var liveCfg atomic.Pointer[AppConfig]

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
		// Second launch (e.g. user double-clicked the shortcut again):
		// instead of an error box, just open the running instance's settings.
		if hwnd := w32.FindWindow("SnapFlowTray", ""); hwnd != 0 {
			w32.PostMessage(hwnd, wmTrayOpenSettings, 0, 0)
		}
		os.Exit(0)
	}

	if !w32ex.SetProcessDPIAwareBestEffort() {
		fmt.Fprintln(os.Stderr, "warn: DPI awareness failed")
	}
	cfgErr := startup()
	if cfgErr != nil && !errors.Is(cfgErr, errConfigReset) {
		w32.MessageBox(0, "SnapFlow could not start:\n\n"+cfgErr.Error(), appName, w32.MB_ICONERROR|w32.MB_OK)
		os.Exit(1)
	}

	// Tray first, so the hotkey-conflict notice can use a tray balloon
	// instead of a blocking dialog.
	if err := initTray(); err != nil {
		fmt.Fprintf(os.Stderr, "warn: tray: %v\n", err)
	}
	if errors.Is(cfgErr, errConfigReset) {
		trayNotify(appName, tr("cfgReset"))
	}

	if failed := registerHotKeys(appCfg); len(failed) > 0 {
		notifyShortcutConflicts(failed, false)
	}

	// Silent update check on startup — only notifies if a new version
	// is available. Runs in background so it never delays startup.
	go func() {
		time.Sleep(8 * time.Second)
		checkForUpdates(true)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		removeTrayIcon()
		postToMain(wmDoQuit, 0, 0)
	}()

	err := msgLoop()
	shutdown()
	if err != nil {
		fmt.Fprintf(os.Stderr, "message loop: %v\n", err)
		os.Exit(1)
	}
}

// startup loads config and starts the drag engine. A config reset is
// reported (errConfigReset) but is not fatal.
func startup() error {
	var err error
	appCfgStore, err = NewConfigStore()
	if err != nil {
		return fmt.Errorf("config store: %w", err)
	}
	cfg, loadErr := appCfgStore.Load()
	if loadErr != nil && !errors.Is(loadErr, errConfigReset) {
		return fmt.Errorf("load config: %w", loadErr)
	}
	appCfg = cfg
	liveCfg.Store(cfg)
	_ = appCfgStore.Save(appCfg) // persist migrated/backfilled values

	overlay, err = NewOverlayRenderer()
	if err != nil {
		return fmt.Errorf("overlay: %w", err)
	}
	overlay.Configure(appCfg.AnimateFootprint, appCfg.FootprintAlpha)

	gest := NewGestureInterpreter(appCfg)
	dragMgr = NewDragManager(gest, overlay)
	if err := dragMgr.Start(); err != nil {
		return err
	}
	return loadErr
}

func shutdown() {
	unregisterAllHotKeys()
	if dragMgr != nil {
		dragMgr.Stop()
	}
	if overlay != nil {
		overlay.Close()
	}
	removeTrayIcon()
}

// runAction applies a WindowAction to the foreground window (hotkey path).
func runAction(action WindowAction) {
	hwnd := w32.GetForegroundWindow()
	if !isZonableWindow(hwnd) {
		return
	}
	executeAction(hwnd, action)
}

// halfCycles: pressing the same half shortcut repeatedly walks through
// these sizes (Rectangle's default "subsequent execution" behaviour).
var halfCycles = map[WindowAction][]WindowAction{
	ActionLeftHalf:  {ActionLeftHalf, ActionFirstTwoThirds, ActionFirstThird},
	ActionRightHalf: {ActionRightHalf, ActionLastTwoThirds, ActionLastThird},
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
		} else if w32ex.IsZoomed(hwnd) {
			w32.ShowWindow(hwnd, w32.SW_RESTORE)
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
		moveToDisplayByIndex(hwnd, int(action-ActionDisplayOne))
		return
	}

	work, ok := monitorWork(w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST))
	if !ok {
		return
	}
	// Geometry works on the visible frame (not GetWindowRect), otherwise
	// size-relative actions like Center/Larger grew the window by the
	// invisible border on every press.
	cur, ok := frameRect(hwnd)
	if !ok {
		return
	}

	effective, cycleIdx := action, -1
	if cycle, isCycle := halfCycles[action]; isCycle && appCfg.CycleHalves {
		cycleIdx = windowHistory.NextCycle(hwnd, action, cur, len(cycle))
		effective = cycle[cycleIdx]
	}

	target := effective.Calculate(work, cur, appCfg)
	got, err := resizeToRect(hwnd, target)
	if err != nil {
		fmt.Printf("warn: action %d: %v\n", action, err)
		return
	}
	if got.Width() > 0 {
		windowHistory.RecordSnap(hwnd, cur, got)
		if cycleIdx >= 0 {
			windowHistory.SetCycle(hwnd, action, cycleIdx)
		}
	}
}

func postToMain(msg uint32, wParam, lParam uintptr) {
	w32ex.PostThreadMessage(mainThreadID, msg, wParam, lParam)
}
