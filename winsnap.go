package main

import (
	"fmt"
	"unsafe"

	"snapflow/w32ex"

	"golang.org/x/sys/windows/registry"
)

// Windows Snap (AeroSnap) conflicts with SnapFlow's drag-to-edge gestures.
//
// SPI_SETWINARRANGING switches it on/off immediately for the whole session
// (and persists it), so no Explorer restart or sign-out is needed. The
// Snap Assist flyout is a separate Explorer setting stored in the registry.

const regExplorer = `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Advanced`

// WindowsSnapEnabled returns true if Windows built-in snap is active.
func WindowsSnapEnabled() bool {
	var on int32
	if w32ex.SystemParametersInfo(w32ex.SPI_GETWINARRANGING, 0, unsafe.Pointer(&on), 0) {
		return on != 0
	}
	return true
}

func DisableWindowsSnap() error { return applyWindowsSnap(false) }
func EnableWindowsSnap() error  { return applyWindowsSnap(true) }

func applyWindowsSnap(on bool) error {
	v := uint32(0)
	if on {
		v = 1
	}
	if !w32ex.SystemParametersInfo(w32ex.SPI_SETWINARRANGING, v, nil,
		w32ex.SPIF_UPDATEINIFILE|w32ex.SPIF_SENDCHANGE) {
		return fmt.Errorf("SystemParametersInfo(SPI_SETWINARRANGING) failed")
	}
	if k, err := registry.OpenKey(registry.CURRENT_USER, regExplorer, registry.SET_VALUE); err == nil {
		_ = k.SetDWordValue("EnableSnapAssistFlyout", v)
		k.Close()
	}
	// With Windows Snap off, SnapFlow handles drag-to-unmaximise itself.
	dragRestore.refresh()
	return nil
}

// setWindowsSnap toggles Windows Snap and confirms with a notification.
func setWindowsSnap(on bool) {
	if err := applyWindowsSnap(on); err != nil {
		return
	}
	state := tr("winSnapDisabled")
	if on {
		state = tr("winSnapEnabled")
	}
	trayNotify(appName, fmt.Sprintf(tr("winSnapApplied"), state))
}
