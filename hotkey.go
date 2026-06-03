package main

import (
	"fmt"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
)

const (
	MOD_ALT      = 0x0001
	MOD_CONTROL  = 0x0002
	MOD_SHIFT    = 0x0004
	MOD_WIN      = 0x0008
	MOD_NOREPEAT = 0x4000
)

var hotkeyMap = make(map[int]*HotKey)

type HotKey struct {
	id       int
	mod, vk  int
	label    string
	action   WindowAction
	callback func()
}

func (h HotKey) String() string {
	s := ""
	if h.mod&MOD_CONTROL != 0 {
		s += "Ctrl+"
	}
	if h.mod&MOD_ALT != 0 {
		s += "Alt+"
	}
	if h.mod&MOD_SHIFT != 0 {
		s += "Shift+"
	}
	if h.mod&MOD_WIN != 0 {
		s += "Win+"
	}
	return s + keyName(h.vk)
}

func RegisterHotKey(h HotKey) bool {
	if _, ok := hotkeyMap[h.id]; ok {
		panic(fmt.Sprintf("duplicate hotkey id: %d", h.id))
	}
	if w32ex.RegisterHotKey(0, h.id, h.mod, h.vk) {
		hotkeyMap[h.id] = &h
		return true
	}
	return false
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
			// WM_QUIT
			return nil
		}

		switch m.Message {
		case w32.WM_HOTKEY:
			if h, ok := hotkeyMap[int(m.WParam)]; ok {
				h.callback()
			}

		case wmOpenSettings:
			openSettings()
		case wmOpenConfig:
			if appCfgStore != nil {
				w32.ShellExecute(0, "open", appCfgStore.Path(), "", "", w32.SW_SHOWNORMAL)
			}
		case wmApplyCfg:
			// Receive a config update posted by the settings WebView goroutine.
			// Reading appCfg happens only on this thread, so no lock needed here.
			select {
			case newCfg := <-cfgUpdateCh:
				appCfg = newCfg
				if dragMgr != nil {
					dragMgr.gest.UpdateConfig(newCfg)
				}
			default:
			}

		case wmDoQuit:
			return nil

		default:
			w32.TranslateMessage(&m)
			w32.DispatchMessage(&m)
		}
	}
}

func keyName(vk int) string {
	named := map[int]string{
		0x08: "Backspace", 0x09: "Tab", 0x0D: "Enter", 0x1B: "Esc",
		0x20: "Space", 0x21: "PgUp", 0x22: "PgDn", 0x23: "End",
		0x24: "Home", 0x25: "←", 0x26: "↑", 0x27: "→",
		0x28: "↓", 0x2E: "Delete",
		0x70: "F1", 0x71: "F2", 0x72: "F3", 0x73: "F4",
		0x74: "F5", 0x75: "F6", 0x76: "F7", 0x77: "F8",
		0x78: "F9", 0x79: "F10", 0x7A: "F11", 0x7B: "F12",
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
