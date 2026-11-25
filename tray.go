package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"unsafe"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

//go:embed assets/snapflow_tray.ico
var trayIconData []byte

const (
	wmTrayNotify       = w32.WM_APP + 100
	wmTrayOpenSettings = w32.WM_APP + 101 // sent by a second SnapFlow launch
	trayTimerRetry     = 1
)

const (
	menuSettings     = 1001
	menuAutoRun      = 1002
	menuOpenConfig   = 1003
	menuQuit         = 1004
	menuDisableWSnap = 1005
	menuCheckUpdate  = 1006
	menuSnapDrag     = 1007
)

var (
	trayShell32 = syscall.NewLazyDLL("shell32.dll")

	procShellNotifyW = trayShell32.NewProc("Shell_NotifyIconW")
	procCreatePopup  = user32DLL.NewProc("CreatePopupMenu")
	procAppendMenuW  = user32DLL.NewProc("AppendMenuW")
	procTrackPopup   = user32DLL.NewProc("TrackPopupMenu")
	procDestroyMenuP = user32DLL.NewProc("DestroyMenu")
	procLoadImageW   = user32DLL.NewProc("LoadImageW")
	procLoadIconW    = user32DLL.NewProc("LoadIconW")
	procSetTimer     = user32DLL.NewProc("SetTimer")
	procKillTimer    = user32DLL.NewProc("KillTimer")

	trayMu    sync.Mutex // guards trayNID / balloon state (notifications come from goroutines too)
	trayHWND  w32.HWND
	trayNID   nid
	trayAdded bool

	// taskbarCreatedMsg is broadcast when Explorer (re)starts. Without
	// handling it the tray icon silently vanishes after an Explorer crash
	// or restart (or when we start before the taskbar at login).
	taskbarCreatedMsg uint32

	// What a click on the current notification balloon should do.
	balloonAction func()
)

// nid is NOTIFYICONDATAW.
type nid struct {
	cbSize           uint32
	hWnd             uintptr
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            uintptr
	szTip            [128]uint16
	dwState          uint32
	dwStateMask      uint32
	szInfo           [256]uint16
	uVersion         uint32
	szInfoTitle      [64]uint16
	dwInfoFlags      uint32
	guidItem         [16]byte
	hBalloonIcon     uintptr
}

const (
	nimAdd    = 0
	nimModify = 1
	nimDelete = 2
	nifMsg    = 0x01
	nifIcon   = 0x02
	nifTip    = 0x04
	nifInfo   = 0x10
	niifInfo  = 0x01
	niifWarn  = 0x02
	niifUser  = 0x04

	ninBalloonUserClick = 0x0405

	mfString    = 0
	mfSeparator = 0x800
	mfChecked   = 8

	tpmReturnCmd   = 0x0100
	tpmRightButton = 0x0002
	tpmBottomAlign = 0x0020
)

var trayWndProc = syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
	m := uint32(msg)
	if taskbarCreatedMsg != 0 && m == taskbarCreatedMsg {
		trayMu.Lock()
		trayAdded = false
		trayMu.Unlock()
		addTrayIcon()
		return 0
	}
	switch m {
	case wmTrayNotify:
		switch uint32(lParam) {
		case 0x0205: // WM_RBUTTONUP
			showTrayMenu(w32.HWND(hwnd))
		case 0x0202, 0x0203: // WM_LBUTTONUP, WM_LBUTTONDBLCLK
			openSettings()
		case ninBalloonUserClick:
			trayMu.Lock()
			act := balloonAction
			balloonAction = nil
			trayMu.Unlock()
			if act != nil {
				act()
			}
		}
		return 0

	case wmTrayOpenSettings:
		openSettings()
		return 0

	case w32.WM_TIMER:
		if wParam == trayTimerRetry && addTrayIcon() {
			procKillTimer.Call(hwnd, trayTimerRetry)
		}
		return 0

	case w32.WM_COMMAND:
		switch wParam & 0xFFFF {
		case menuSettings:
			openSettings()
		case menuSnapDrag:
			c := appCfg.clone()
			c.SnapByDragging = !c.SnapByDragging
			if appCfgStore != nil {
				_ = appCfgStore.Save(c)
			}
			applyConfig(c)
		case menuAutoRun:
			if en, _ := AutoRunEnabled(); en {
				AutoRunDisable()
			} else {
				AutoRunEnable()
			}
		case menuOpenConfig:
			if appCfgStore != nil {
				w32.ShellExecute(0, "open", appCfgStore.Path(), "", "", w32.SW_SHOWNORMAL)
			}
		case menuDisableWSnap:
			setWindowsSnap(!WindowsSnapEnabled())
		case menuCheckUpdate:
			go checkForUpdates(false)
		case menuQuit:
			removeTrayIcon()
			postToMain(wmDoQuit, 0, 0)
		}
		return 0

	case w32.WM_SETTINGCHANGE:
		// Windows Snap may have been toggled in Windows' own Settings.
		if wParam == w32ex.SPI_SETWINARRANGING {
			dragRestore.refresh()
		}

	case w32.WM_DESTROY:
		removeTrayIcon()
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
})

func initTray() error {
	taskbarCreatedMsg = w32ex.RegisterWindowMessage("TaskbarCreated")
	instance, _, _ := procGetModuleHandleW.Call(0)

	cn := syscall.StringToUTF16Ptr("SnapFlowTray")
	var wc wndClassExW
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.lpfnWndProc = trayWndProc
	wc.hInstance = instance
	wc.lpszClassName = cn
	if r, _, e := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 && e != syscall.Errno(1410) {
		return fmt.Errorf("RegisterClassEx: %v", e)
	}

	// A hidden top-level window (not HWND_MESSAGE, so it receives the
	// TaskbarCreated broadcast). It lives on the main thread, so msgLoop
	// dispatches its messages.
	r, _, e := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(cn)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("SnapFlow"))),
		uintptr(w32.WS_POPUP),
		0, 0, 1, 1,
		0, 0, instance, 0,
	)
	if r == 0 {
		return fmt.Errorf("CreateWindowEx tray: %v", e)
	}
	trayHWND = w32.HWND(r)

	trayMu.Lock()
	trayNID = nid{
		hWnd:             uintptr(trayHWND),
		uID:              1,
		uFlags:           nifMsg | nifIcon | nifTip,
		uCallbackMessage: wmTrayNotify,
		hIcon:            loadTrayIcon(),
	}
	trayNID.cbSize = uint32(unsafe.Sizeof(trayNID))
	setUTF16(trayNID.szTip[:], trayTipText())
	trayMu.Unlock()

	if !addTrayIcon() {
		// Explorer's taskbar may not be ready yet (early at login): retry.
		procSetTimer.Call(uintptr(trayHWND), trayTimerRetry, 2000, 0)
	}
	return nil
}

func addTrayIcon() bool {
	trayMu.Lock()
	defer trayMu.Unlock()
	if trayHWND == 0 {
		return false
	}
	if trayAdded {
		return true
	}
	n := trayNID
	n.uFlags = nifMsg | nifIcon | nifTip
	ret, _, _ := procShellNotifyW.Call(nimAdd, uintptr(unsafe.Pointer(&n)))
	if ret == 0 {
		// Already present (e.g. after TaskbarCreated race): modify instead.
		ret, _, _ = procShellNotifyW.Call(nimModify, uintptr(unsafe.Pointer(&n)))
	}
	trayAdded = ret != 0
	return trayAdded
}

func removeTrayIcon() {
	trayMu.Lock()
	defer trayMu.Unlock()
	if trayHWND == 0 || !trayAdded {
		return
	}
	procShellNotifyW.Call(nimDelete, uintptr(unsafe.Pointer(&trayNID)))
	trayAdded = false
}

func trayTipText() string {
	if c := liveCfg.Load(); c != nil && !c.SnapByDragging {
		return tr("tipPaused")
	}
	return tr("tip")
}

// updateTrayTip refreshes the tooltip (e.g. after pausing drag snapping).
func updateTrayTip() {
	trayMu.Lock()
	defer trayMu.Unlock()
	if !trayAdded {
		return
	}
	setUTF16(trayNID.szTip[:], trayTipText())
	n := trayNID
	n.uFlags = nifTip
	procShellNotifyW.Call(nimModify, uintptr(unsafe.Pointer(&n)))
}

// trayNotify shows a non-blocking Windows notification from the tray icon.
// Safe from any goroutine. onClick (optional) runs on the main thread if
// the user clicks the notification.
func trayNotify(title, text string, onClick ...func()) {
	trayMu.Lock()
	defer trayMu.Unlock()
	if !trayAdded {
		return
	}
	balloonAction = nil
	if len(onClick) > 0 {
		balloonAction = onClick[0]
	}
	n := trayNID
	n.uFlags = nifInfo
	n.dwInfoFlags = niifInfo
	setUTF16(n.szInfoTitle[:], title)
	setUTF16(n.szInfo[:], text)
	procShellNotifyW.Call(nimModify, uintptr(unsafe.Pointer(&n)))
}

func setUTF16(dst []uint16, s string) {
	u, _ := syscall.UTF16FromString(s)
	if len(u) > len(dst) {
		u = u[:len(dst)]
		u[len(u)-1] = 0
	}
	for i := range dst {
		dst[i] = 0
	}
	copy(dst, u)
}

func loadTrayIcon() uintptr {
	// LoadImage needs a file; write the embedded .ico next to the config
	// (falls back to %TEMP%) and load it at the small-icon size.
	if len(trayIconData) > 0 {
		dir := os.TempDir()
		if appCfgStore != nil {
			dir = filepath.Dir(appCfgStore.Path())
		}
		p := filepath.Join(dir, "tray.ico")
		if err := os.WriteFile(p, trayIconData, 0o644); err == nil {
			ptr := syscall.StringToUTF16Ptr(p)
			icon, _, _ := procLoadImageW.Call(
				0, uintptr(unsafe.Pointer(ptr)),
				1, // IMAGE_ICON
				uintptr(w32.GetSystemMetrics(w32.SM_CXSMICON)),
				uintptr(w32.GetSystemMetrics(w32.SM_CYSMICON)),
				0x00000010, // LR_LOADFROMFILE
			)
			if icon != 0 {
				return icon
			}
		}
	}
	icon, _, _ := procLoadIconW.Call(0, 32512) // IDI_APPLICATION
	return icon
}

func showTrayMenu(hwnd w32.HWND) {
	hMenu, _, _ := procCreatePopup.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenuP.Call(hMenu)

	add := func(id uintptr, label string, checked bool) {
		p := syscall.StringToUTF16Ptr(label)
		f := uintptr(mfString)
		if checked {
			f |= mfChecked
		}
		procAppendMenuW.Call(hMenu, f, id, uintptr(unsafe.Pointer(p)))
	}
	sep := func() { procAppendMenuW.Call(hMenu, mfSeparator, 0, 0) }

	add(menuSettings, tr("menuSettings"), false)
	sep()
	add(menuSnapDrag, tr("menuSnapDrag"), appCfg.SnapByDragging)
	autorun, _ := AutoRunEnabled()
	add(menuAutoRun, tr("menuAutoRun"), autorun)
	// A normal checkbox showing the real state (it used to be an action
	// whose label flipped between "Disable…" and "Re-enable…").
	add(menuDisableWSnap, tr("menuWinSnap"), WindowsSnapEnabled())
	sep()
	add(menuOpenConfig, tr("menuOpenConfig"), false)
	add(menuCheckUpdate, tr("menuUpdates"), false)
	sep()
	add(menuQuit, tr("menuQuit"), false)

	w32.SetForegroundWindow(hwnd)
	pt, _ := w32ex.GetCursorPos()
	cmd, _, _ := procTrackPopup.Call(
		hMenu,
		tpmReturnCmd|tpmRightButton|tpmBottomAlign,
		uintptr(pt.X), uintptr(pt.Y),
		0, uintptr(hwnd), 0,
	)
	if cmd != 0 {
		w32.PostMessage(hwnd, w32.WM_COMMAND, cmd, 0)
	}
}
