package main

import (
	"fmt"
	"strconv"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

const (
	MOD_ALT      = 0x0001
	MOD_CONTROL  = 0x0002
	MOD_SHIFT    = 0x0004
	MOD_WIN      = 0x0008
	MOD_NOREPEAT = 0x4000

	modMask = MOD_ALT | MOD_CONTROL | MOD_SHIFT | MOD_WIN
)

// hotkeyMap holds the currently registered hotkeys by id. Only touched on
// the main OS thread (RegisterHotKey with hwnd=0 binds to the calling
// thread's message queue, so registration must happen there too).
var hotkeyMap = make(map[int]*HotKey)

type HotKey struct {
	id      int
	mod, vk int
	label   string
	action  WindowAction
}

func (h HotKey) String() string { return shortcutChipText(h.mod, h.vk) }

func hotkeyID(a WindowAction) int { return 1000 + int(a) }

// effectiveShortcuts returns every action with its default shortcut,
// overridden by the user's custom bindings from cfg.
func effectiveShortcuts(cfg *AppConfig) []sEntry {
	entries := allShortcutEntries()
	for i, e := range entries {
		if sc, ok := cfg.Shortcuts[strconv.Itoa(int(e.action))]; ok {
			entries[i].mod = sc.Mod & modMask
			entries[i].vk = sc.VK
		}
	}
	return entries
}

func unregisterAllHotKeys() {
	for id := range hotkeyMap {
		w32ex.UnregisterHotKey(0, id)
		delete(hotkeyMap, id)
	}
}

// registerHotKeys (re)registers all shortcuts from cfg and returns those
// that another application already owns. Must run on the main thread.
func registerHotKeys(cfg *AppConfig) []HotKey {
	unregisterAllHotKeys()
	var failed []HotKey
	for _, e := range effectiveShortcuts(cfg) {
		if e.vk == 0 {
			continue
		}
		hk := HotKey{id: hotkeyID(e.action), mod: e.mod, vk: e.vk, label: e.label, action: e.action}
		if w32ex.RegisterHotKey(0, hk.id, hk.mod|MOD_NOREPEAT, hk.vk) {
			h := hk
			hotkeyMap[hk.id] = &h
		} else {
			failed = append(failed, hk)
		}
	}
	return failed
}

// msgLoop is the main Win32 message loop. It runs on the locked main OS thread
// so WM_HOTKEY and WINEVENT_OUTOFCONTEXT callbacks are dispatched here.
func msgLoop() error {
	for {
		var m w32.MSG
		c := w32.GetMessage(&m, 0, 0, 0)
		if c == -1 {
			return fmt.Errorf("GetMessage: %d", w32.GetLastError())
		}
		if c == 0 {
			return nil // WM_QUIT
		}

		if m.Hwnd == 0 {
			switch m.Message {
			case w32.WM_HOTKEY:
				if h, ok := hotkeyMap[int(m.WParam)]; ok {
					runAction(h.action)
				}
				continue

			case wmOpenSettings:
				openSettings()
				continue

			case wmOpenConfig:
				if appCfgStore != nil {
					w32.ShellExecute(0, "open", appCfgStore.Path(), "", "", w32.SW_SHOWNORMAL)
				}
				continue

			case wmApplyCfg:
				// Config posted by the settings goroutine. appCfg is only
				// written here, on the main thread.
				if newCfg := pendingCfg.Swap(nil); newCfg != nil {
					applyConfig(newCfg)
				}
				continue

			case wmSuspendHotkeys:
				// The settings window is recording a shortcut: global hotkeys
				// would swallow the key press, so release them meanwhile.
				if m.WParam != 0 {
					unregisterAllHotKeys()
				} else {
					registerHotKeys(appCfg)
				}
				continue

			case wmDoQuit:
				return nil
			}
		}
		w32.TranslateMessage(&m)
		w32.DispatchMessage(&m)
	}
}

// applyConfig installs a new config on the main thread and propagates it.
func applyConfig(newCfg *AppConfig) {
	old := appCfg
	appCfg = newCfg
	liveCfg.Store(newCfg)
	updateTrayTip()
	if dragMgr != nil {
		dragMgr.gest.UpdateConfig(newCfg)
	}
	if overlay != nil {
		overlay.Configure(newCfg.AnimateFootprint, newCfg.FootprintAlpha)
	}
	if !newCfg.SquareCorners {
		roundAllCorners()
	}
	if old == nil || !sameShortcuts(old, newCfg) {
		if failed := registerHotKeys(newCfg); len(failed) > 0 {
			notifyShortcutConflicts(failed, true)
		}
	}
}

func sameShortcuts(a, b *AppConfig) bool {
	ea, eb := effectiveShortcuts(a), effectiveShortcuts(b)
	if len(ea) != len(eb) {
		return false
	}
	for i := range ea {
		if ea[i].mod != eb[i].mod || ea[i].vk != eb[i].vk {
			return false
		}
	}
	return true
}

func keyName(vk int) string {
	named := map[int]string{
		0x08: "Backspace", 0x09: "Tab", 0x0D: "Enter", 0x1B: "Esc",
		0x20: "Space", 0x21: "PgUp", 0x22: "PgDn", 0x23: "End",
		0x24: "Home", 0x25: "←", 0x26: "↑", 0x27: "→",
		0x28: "↓", 0x2D: "Insert", 0x2E: "Delete",
		0x70: "F1", 0x71: "F2", 0x72: "F3", 0x73: "F4",
		0x74: "F5", 0x75: "F6", 0x76: "F7", 0x77: "F8",
		0x78: "F9", 0x79: "F10", 0x7A: "F11", 0x7B: "F12",
		0x60: "Num0", 0x61: "Num1", 0x62: "Num2", 0x63: "Num3", 0x64: "Num4",
		0x65: "Num5", 0x66: "Num6", 0x67: "Num7", 0x68: "Num8", 0x69: "Num9",
		0x6A: "Num*", 0x6B: "Num+", 0x6D: "Num-", 0x6E: "Num.", 0x6F: "Num/",
		// OEM keys
		0xBA: ";", 0xBB: "=", 0xBC: ",", 0xBD: "-", 0xBE: ".", 0xBF: "/",
		0xC0: "`", 0xDB: "[", 0xDC: "\\", 0xDD: "]", 0xDE: "'",
	}
	if n, ok := named[vk]; ok {
		return n
	}
	if vk >= 0x41 && vk <= 0x5A {
		return string(rune('A' + vk - 0x41))
	}
	if vk >= 0x30 && vk <= 0x39 {
		return string(rune('0' + vk - 0x30))
	}
	return fmt.Sprintf("0x%X", vk)
}
