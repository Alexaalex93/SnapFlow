package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
	webview "github.com/jchv/go-webview2"
)

func buildShortcutEntries(cfg *AppConfig) []ShortcutEntry {
	defaults := allShortcutEntries()
	eff := effectiveShortcuts(cfg)
	out := make([]ShortcutEntry, 0, len(eff))
	for i, e := range eff {
		d := defaults[i]
		out = append(out, ShortcutEntry{
			Action:  int(e.action),
			Label:   e.label,
			Section: e.section,
			Mod:     e.mod,
			VK:      e.vk,
			Text:    shortcutChipText(e.mod, e.vk),
			DefMod:  d.mod,
			DefVK:   d.vk,
			DefText: shortcutChipText(d.mod, d.vk),
		})
	}
	return out
}

func shortcutChipText(mod, vk int) string {
	if vk == 0 {
		return ""
	}
	s := ""
	if mod&MOD_CONTROL != 0 {
		s += "Ctrl+"
	}
	if mod&MOD_ALT != 0 {
		s += "Alt+"
	}
	if mod&MOD_SHIFT != 0 {
		s += "Shift+"
	}
	if mod&MOD_WIN != 0 {
		s += "Win+"
	}
	return s + keyName(vk)
}

// shortcutOverrides converts the full list from the UI into the sparse
// config map: only bindings that differ from the default are stored.
func shortcutOverrides(list []ShortcutBinding) map[string]ShortcutConfig {
	def := map[WindowAction]sEntry{}
	for _, e := range allShortcutEntries() {
		def[e.action] = e
	}
	out := map[string]ShortcutConfig{}
	for _, b := range list {
		d, ok := def[WindowAction(b.Action)]
		if !ok {
			continue
		}
		mod, vk := b.Mod&modMask, b.VK
		if vk == 0 {
			mod = 0
		}
		if mod == d.mod && vk == d.vk {
			continue
		}
		out[strconv.Itoa(b.Action)] = ShortcutConfig{Mod: mod, VK: vk}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func runSettingsWindow() {
	cfg := liveCfg.Load()
	if cfg == nil {
		return
	}
	dataDir := ""
	if appCfgStore != nil {
		dataDir = filepath.Join(filepath.Dir(appCfgStore.Path()), "WebView2")
	}

	// The process is per-monitor DPI aware, so sizes are physical pixels:
	// scale the logical 760×660 window to the monitor under the cursor.
	scale := 1.0
	if pt, ok := w32ex.GetCursorPos(); ok {
		scale = float64(w32ex.GetDpiForMonitor(w32ex.MonitorFromPoint(pt, w32ex.MONITOR_DEFAULTTONEAREST))) / 96
	}
	width, height := uint(760*scale), uint(660*scale)

	w := webview.NewWithOptions(webview.WebViewOptions{
		Debug:     false,
		DataPath:  dataDir,
		AutoFocus: true,
		WindowOptions: webview.WindowOptions{
			Title:  appName + " — Settings",
			Width:  width,
			Height: height,
			Center: true,
		},
	})
	if w == nil {
		result := w32.MessageBox(0,
			"The Settings window requires Microsoft WebView2 Runtime,\n"+
				"which is not available on this computer.\n\n"+
				"You can still configure SnapFlow by editing the config file directly.\n\n"+
				"Open config file now?",
			appName+" — Settings",
			w32.MB_ICONWARNING|w32.MB_YESNO)
		if result == w32.IDYES {
			postToMain(wmOpenConfig, 0, 0)
		}
		return
	}
	defer w.Destroy()
	settingsHWND.Store(uintptr(w.Window()))
	w.SetSize(int(560*scale), int(480*scale), webview.HintMin)

	w.Bind("goSaveSettings", func(payload string) {
		var p SavePayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			fmt.Fprintf(os.Stderr, "settings save: %v\n", err)
			return
		}

		// Build a NEW config from the latest one — never mutate the config
		// the main thread is reading. submitConfig hands it over safely.
		newCfg := liveCfg.Load().clone()
		g := p.General
		newCfg.EdgeThresholdPx = g.EdgeThreshold
		newCfg.CornerSnapAreaSize = g.CornerSize
		newCfg.ShortEdgeSnapAreaSize = g.ShortEdgeSize
		newCfg.GapSize = g.GapSize
		newCfg.AlmostMaximizeWidth = g.AlmostMaxWidth / 100.0
		newCfg.AlmostMaximizeHeight = g.AlmostMaxHeight / 100.0
		newCfg.SizeOffset = g.SizeStep
		newCfg.WidthStepSize = g.SizeStep
		newCfg.CycleHalves = g.CycleHalves
		if g.PreviewOpacity > 0 {
			newCfg.FootprintAlpha = float64(g.PreviewOpacity) / 100.0
		}

		sa := p.SnapAreas
		newCfg.SnapByDragging = sa.SnapByDragging
		newCfg.RestoreSize = sa.RestoreSize
		newCfg.AnimateFootprint = sa.AnimateFootprint
		newCfg.SquareCorners = sa.SquareCorners
		newCfg.AnimateWindows = sa.AnimateWindows
		newCfg.ZoneTopLeft = sa.TopLeft
		newCfg.ZoneTop = sa.Top
		newCfg.ZoneTopRight = sa.TopRight
		newCfg.ZoneLeft = sa.Left
		newCfg.ZoneRight = sa.Right
		newCfg.ZoneBottomLeft = sa.BottomLeft
		newCfg.ZoneBottom = sa.Bottom
		newCfg.ZoneBottomRight = sa.BottomRight

		if p.Shortcuts != nil {
			newCfg.Shortcuts = shortcutOverrides(p.Shortcuts)
		}
		if p.Language == "en" || p.Language == "es" {
			newCfg.Language = p.Language
		}

		submitConfig(fillDefaults(newCfg))

		// Eval/Terminate must not run inside a Bind callback (WebView2
		// deadlock). Dispatch queues them on the UI thread after we return.
		go func() {
			w.Dispatch(func() { w.Eval("window.showSaved()") })
			time.Sleep(650 * time.Millisecond)
			w.Dispatch(func() { w.Terminate() })
		}()
	})

	w.Bind("goSetLanguage", func(lang string) {
		if lang != "en" && lang != "es" {
			return
		}
		c := liveCfg.Load().clone()
		c.Language = lang
		submitConfig(c)
	})

	// While the user records a shortcut, our own global hotkeys would
	// intercept the keys before the page sees them.
	w.Bind("goSuspendHotkeys", func(suspend bool) {
		v := uintptr(0)
		if suspend {
			v = 1
		}
		postToMain(wmSuspendHotkeys, v, 0)
	})

	w.Bind("goShortcutText", func(mod, vk int) string {
		return shortcutChipText(mod&modMask, vk)
	})

	// window.close() does not close the native host window in WebView2.
	w.Bind("goClose", func() {
		go w.Dispatch(func() { w.Terminate() })
	})

	w.Bind("goOpenConfigFile", func() {
		postToMain(wmOpenConfig, 0, 0)
	})

	w.Bind("goSetWindowsSnap", func(enabled bool) bool {
		if err := applyWindowsSnap(enabled); err != nil {
			return WindowsSnapEnabled()
		}
		return enabled
	})

	// goCheckUpdates queries GitHub and returns a result string:
	//   "uptodate"        — already on latest version
	//   "new:1.2.3"       — newer version available
	//   "error: <msg>"    — network or API error
	w.Bind("goCheckUpdates", func() string {
		latest, err := fetchLatestVersion()
		if err != nil {
			return "error: " + err.Error()
		}
		current := strings.TrimPrefix(appVersion, "v")
		latestClean := strings.TrimPrefix(latest, "v")
		if isNewerVersion(latestClean, current) {
			return "new:" + latestClean
		}
		return "uptodate"
	})

	w.Bind("goOpenRepo", func() {
		openReleasesPage()
	})

	w.Bind("goSetAutorun", func(enabled bool) bool {
		if enabled {
			_ = AutoRunEnable()
		} else {
			_ = AutoRunDisable()
		}
		on, _ := AutoRunEnabled()
		return on
	})

	autorun, _ := AutoRunEnabled()
	inject := map[string]interface{}{
		"_shortcuts": buildShortcutEntries(cfg),
		"_general": GeneralSettings{
			EdgeThreshold:   cfg.EdgeThresholdPx,
			CornerSize:      cfg.CornerSnapAreaSize,
			ShortEdgeSize:   cfg.ShortEdgeSnapAreaSize,
			GapSize:         cfg.GapSize,
			AlmostMaxWidth:  cfg.AlmostMaximizeWidth * 100,
			AlmostMaxHeight: cfg.AlmostMaximizeHeight * 100,
			SizeStep:        cfg.SizeOffset,
			CycleHalves:     cfg.CycleHalves,
			PreviewOpacity:  int(cfg.FootprintAlpha*100 + 0.5),
		},
		"_snapAreas": SnapAreasSettings{
			SnapByDragging:   cfg.SnapByDragging,
			RestoreSize:      cfg.RestoreSize,
			AnimateFootprint: cfg.AnimateFootprint,
			SquareCorners:    cfg.SquareCorners,
			AnimateWindows:   cfg.AnimateWindows,
			TopLeft:          cfg.ZoneTopLeft,
			Top:              cfg.ZoneTop,
			TopRight:         cfg.ZoneTopRight,
			Left:             cfg.ZoneLeft,
			Right:            cfg.ZoneRight,
			BottomLeft:       cfg.ZoneBottomLeft,
			Bottom:           cfg.ZoneBottom,
			BottomRight:      cfg.ZoneBottomRight,
		},
		"_winSnap": WindowsSnapEnabled(),
		"_version": appVersion,
		"_autorun": autorun,
		"_lang":    uiLang(),
	}
	var js strings.Builder
	for k, v := range inject {
		b, _ := json.Marshal(v)
		fmt.Fprintf(&js, "window.%s = %s;\n", k, b)
	}
	w.Init(js.String())

	tmpPath := filepath.Join(os.TempDir(), "snapflow_settings.html")
	if err := os.WriteFile(tmpPath, []byte(settingsHTML), 0o644); err != nil {
		fmt.Printf("settings html write: %v\n", err)
		return
	}
	defer os.Remove(tmpPath)

	w.Navigate("file:///" + strings.ReplaceAll(tmpPath, "\\", "/"))
	w.Run()
}
