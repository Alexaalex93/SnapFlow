package main

import (
	"fmt"

	"github.com/gonutz/w32/v2"
)

func showShortcutConflictDialog(failed []HotKey) {
	hasArrow := false
	for _, hk := range failed {
		if hk.vk == 0x25 || hk.vk == 0x26 || hk.vk == 0x27 || hk.vk == 0x28 {
			hasArrow = true
			break
		}
	}

	msg := fmt.Sprintf("%d shortcut(s) already in use by another app:\n\n", len(failed))
	for _, hk := range failed {
		msg += fmt.Sprintf("  %-28s  %s\n", hk.String(), hk.label)
	}
	msg += "\nDrag-to-snap and the other shortcuts still work.\n"
	if hasArrow {
		msg += "\nCtrl+Alt+Arrow is taken by your GPU driver (screen rotation).\n"
		msg += "To fix: open your GPU control panel → Hotkeys → disable rotation shortcuts.\n\n"
		msg += "After disabling, restart SnapFlow."
	}

	buttons := w32.MB_ICONWARNING | w32.MB_OK
	if hasArrow {
		// Show a button to open GPU settings.
		buttons = w32.MB_ICONWARNING | w32.MB_YESNO
		msg += "\n\nClick Yes to open GPU/Display Settings now."
		result := w32.MessageBox(0, msg, appName+" — Shortcut Conflict", uint(buttons))
		if result == w32.IDYES {
			openGPUSettings()
		}
		return
	}
	w32.MessageBox(0, msg, appName+" — Shortcut Conflict", uint(buttons))
}

func openGPUSettings() {
	for _, exe := range []string{"igcc.exe", "igfxCPL.exe", "RadeonSettings.exe", "CCC.exe"} {
		if err := w32.ShellExecute(0, "open", exe, "", "", w32.SW_SHOWNORMAL); err == nil {
			return
		}
	}
	w32.ShellExecute(0, "open", "ms-settings:display", "", "", w32.SW_SHOWNORMAL)
}
