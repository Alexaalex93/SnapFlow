package main

import (
	"errors"

	"golang.org/x/sys/windows/registry"
)

// Windows Snap (AeroSnap) is controlled by two registry values.
// Disabling them prevents Windows from intercepting drag-to-edge gestures
// that would otherwise conflict with SnapFlow.

const (
	regDesktop   = `Control Panel\Desktop`
	regExplorer  = `SOFTWARE\Microsoft\Windows\CurrentVersion\Explorer\Advanced`
)

// WindowsSnapEnabled returns true if Windows built-in snap is active.
func WindowsSnapEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, regDesktop, registry.QUERY_VALUE)
	if err != nil {
		return true // assume on if can't read
	}
	defer k.Close()
	v, _, err := k.GetStringValue("WindowArrangementActive")
	if errors.Is(err, registry.ErrNotExist) {
		return true // default is on
	}
	return v != "0"
}

// DisableWindowsSnap turns off Windows AeroSnap and Snap Assist.
// The user must sign out or restart Explorer for the change to take effect,
// but the drag conflict disappears after the next Explorer restart.
func DisableWindowsSnap() error {
	// 1. Disable AeroSnap (the drag-to-edge part).
	k1, err := registry.OpenKey(registry.CURRENT_USER, regDesktop, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k1.Close()
	if err := k1.SetStringValue("WindowArrangementActive", "0"); err != nil {
		return err
	}

	// 2. Disable Snap Assist flyout (the thumbnail picker).
	k2, err := registry.OpenKey(registry.CURRENT_USER, regExplorer, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k2.Close()
	return k2.SetDWordValue("EnableSnapAssistFlyout", 0)
}

// EnableWindowsSnap re-enables Windows built-in snap.
func EnableWindowsSnap() error {
	k1, err := registry.OpenKey(registry.CURRENT_USER, regDesktop, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k1.Close()
	if err := k1.SetStringValue("WindowArrangementActive", "1"); err != nil {
		return err
	}

	k2, err := registry.OpenKey(registry.CURRENT_USER, regExplorer, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k2.Close()
	return k2.SetDWordValue("EnableSnapAssistFlyout", 1)
}
