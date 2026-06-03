package main

import (
	_ "embed"
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/gonutz/w32/v2"
)

//go:embed assets/snapflow_tray.ico
var trayIconData []byte

const wmTrayNotify = w32.WM_APP + 100

const (
	menuSettings     = 1001
	menuAutoRun      = 1002
	menuOpenConfig   = 1003
	menuQuit         = 1004
	menuDisableWSnap = 1005
	menuCheckUpdate  = 1006
)

var (
	trayUser32  = syscall.NewLazyDLL("user32.dll")
	trayShell32 = syscall.NewLazyDLL("shell32.dll")

	procShellNotifyW = trayShell32.NewProc("Shell_NotifyIconW")
	procCreatePopup  = trayUser32.NewProc("CreatePopupMenu")
	procAppendMenuW  = trayUser32.NewProc("AppendMenuW")
	procTrackPopup   = trayUser32.NewProc("TrackPopupMenu")
	procDestroyMenuP = trayUser32.NewProc("DestroyMenu")
	procPostMsgTray  = trayUser32.NewProc("PostMessageW")
	procSetFgTray    = trayUser32.NewProc("SetForegroundWindow")
	procGetCurTray   = trayUser32.NewProc("GetCursorPos")

	trayHWND w32.HWND
	trayNID  nid
)

// nid is a simplified NOTIFYICONDATAW.
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
	nimDelete = 2
	nifMsg    = 1
	nifIcon   = 2
	nifTip    = 4

	mfString    = 0
	mfSeparator = 0x800
	mfChecked   = 8

	tpmReturnCmd   = 0x0100
	tpmRightButton = 0x0002
	tpmBottomAlign = 0x0020
)

var trayWndProc = syscall.NewCallback(func(hwnd, msg, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case wmTrayNotify:
		switch uint32(lParam) {
		case 0x0205: // WM_RBUTTONUP
			showTrayMenu(w32.HWND(hwnd))
		case 0x0203: // WM_LBUTTONDBLCLK
			postToMain(wmOpenSettings, 0, 0)
		}
		return 0

	case w32.WM_COMMAND:
		switch wParam & 0xFFFF {
		case menuSettings:
			postToMain(wmOpenSettings, 0, 0)
		case menuAutoRun:
			en, _ := AutoRunEnabled()
			if en {
				AutoRunDisable()
			} else {
				AutoRunEnable()
			}
		case menuOpenConfig:
			if appCfgStore != nil {
				w32.ShellExecute(0, "open", appCfgStore.Path(), "", "", w32.SW_SHOWNORMAL)
			}
		case menuDisableWSnap:
			if WindowsSnapEnabled() {
				DisableWindowsSnap()
			} else {
				EnableWindowsSnap()
			}
		case menuCheckUpdate:
			go checkForUpdates(false)
		case menuQuit:
			removeTrayIcon()
			postToMain(wmDoQuit, 0, 0)
		}
		return 0

	case w32.WM_DESTROY:
		removeTrayIcon()
		return 0
	}
	r, _, _ := procDefWindowProcW.Call(hwnd, msg, wParam, lParam)
	return r
})

func initTray() error {
	instance, _, _ := procGetModuleHandleW.Call(0)

	// Register window class for the hidden tray message window.
	cn := syscall.StringToUTF16Ptr("SnapFlowTray")
	var wc wndClassExW
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.lpfnWndProc = trayWndProc
	wc.hInstance = instance
	wc.lpszClassName = cn
	if r, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		e := syscall.GetLastError()
		if e != syscall.Errno(1410) { // ERROR_CLASS_ALREADY_EXISTS is fine
			return fmt.Errorf("RegisterClassEx: %v", e)
		}
	}

	// Create a hidden (WS_POPUP, not visible) window to receive tray messages.
	// Using 0 as parent (desktop) instead of HWND_MESSAGE so our existing
	// DispatchMessage in msgLoop can deliver messages to it.
	r, _, e := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(cn)),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("SnapFlow"))),
		uintptr(w32.WS_POPUP), // invisible popup window
		0, 0, 1, 1,
		0, 0, instance, 0,
	)
	if r == 0 {
		return fmt.Errorf("CreateWindowEx tray: %v (LastError=%d)", e, w32.GetLastError())
	}
	trayHWND = w32.HWND(r)
	fmt.Printf("tray window created: 0x%X\n", trayHWND)

	// Load icon: try the embedded .ico data, fall back to system icon.
	icon := loadTrayIcon()
	fmt.Printf("tray icon handle: 0x%X\n", icon)

	// Build tooltip string.
	tipStr := appName + " — Window Manager"
	tip := syscall.StringToUTF16Ptr(tipStr)

	trayNID = nid{
		hWnd:             uintptr(trayHWND),
		uID:              1,
		uFlags:           nifMsg | nifIcon | nifTip,
		uCallbackMessage: wmTrayNotify,
		hIcon:            icon,
	}
	trayNID.cbSize = uint32(unsafe.Sizeof(trayNID))
	// Copy tooltip (max 127 chars + null).
	tipSlice := (*[128]uint16)(unsafe.Pointer(tip))
	for i := 0; i < 127 && tipSlice[i] != 0; i++ {
		trayNID.szTip[i] = tipSlice[i]
	}

	ret, _, _ := procShellNotifyW.Call(nimAdd, uintptr(unsafe.Pointer(&trayNID)))
	fmt.Printf("Shell_NotifyIcon result: %d (LastError=%d)\n", ret, w32.GetLastError())
	if ret == 0 {
		return fmt.Errorf("Shell_NotifyIcon NIM_ADD failed: %d", w32.GetLastError())
	}
	fmt.Println("tray icon added successfully")
	return nil
}

func removeTrayIcon() {
	if trayHWND == 0 {
		return
	}
	procShellNotifyW.Call(nimDelete, uintptr(unsafe.Pointer(&trayNID)))
}

func loadTrayIcon() uintptr {
	// Try loading from the embedded .ico file bytes.
	// Write to a temp file then LoadImage from file (most reliable approach).
	if len(trayIconData) > 0 {
		tmpPath := os.TempDir() + "\\snapflow_tray_tmp.ico"
		if err := os.WriteFile(tmpPath, trayIconData, 0o644); err == nil {
			defer os.Remove(tmpPath)
			ptr := syscall.StringToUTF16Ptr(tmpPath)
			icon, _, _ := trayUser32.NewProc("LoadImageW").Call(
				0,
				uintptr(unsafe.Pointer(ptr)),
				1, // IMAGE_ICON
				0, 0,
				0x00000010, // LR_LOADFROMFILE
			)
			if icon != 0 {
				return icon
			}
			fmt.Printf("LoadImage from file failed: %d\n", w32.GetLastError())
		}
	}
	// Fallback: Windows application icon.
	icon, _, _ := trayUser32.NewProc("LoadIconW").Call(0, 32512)
	return icon
}

func showTrayMenu(hwnd w32.HWND) {
	hMenu, _, _ := procCreatePopup.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenuP.Call(hMenu)

	add := func(id uintptr, label string) {
		p := syscall.StringToUTF16Ptr(label)
		procAppendMenuW.Call(hMenu, mfString, id, uintptr(unsafe.Pointer(p)))
	}
	addChecked := func(id uintptr, label string, checked bool) {
		p := syscall.StringToUTF16Ptr(label)
		f := uintptr(mfString)
		if checked {
			f |= mfChecked
		}
		procAppendMenuW.Call(hMenu, f, id, uintptr(unsafe.Pointer(p)))
	}
	sep := func() { procAppendMenuW.Call(hMenu, mfSeparator, 0, 0) }

	add(menuSettings, "Settings…")
	add(menuCheckUpdate, "Check for updates…")
	sep()
	// Show Windows Snap toggle — it interferes with SnapFlow drag gestures.
	snapOn := WindowsSnapEnabled()
	if snapOn {
		add(menuDisableWSnap, "Disable Windows Snap  ⚠")
	} else {
		add(menuDisableWSnap, "Re-enable Windows Snap")
	}
	sep()
	autorun, _ := AutoRunEnabled()
	addChecked(menuAutoRun, "Run on startup", autorun)
	add(menuOpenConfig, "Open Config File")
	sep()
	add(menuQuit, "Quit SnapFlow")

	procSetFgTray.Call(uintptr(hwnd))

	var pt struct{ x, y int32 }
	procGetCurTray.Call(uintptr(unsafe.Pointer(&pt)))

	cmd, _, _ := procTrackPopup.Call(
		hMenu,
		tpmReturnCmd|tpmRightButton|tpmBottomAlign,
		uintptr(pt.x), uintptr(pt.y),
		0, uintptr(hwnd), 0,
	)
	if cmd != 0 {
		procPostMsgTray.Call(uintptr(hwnd), w32.WM_COMMAND, cmd, 0)
	}
}
