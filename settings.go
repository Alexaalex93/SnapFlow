package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gonutz/w32/v2"
	webview "github.com/jchv/go-webview2"
)

var settingsOpen atomic.Bool

func openSettings() {
	if !settingsOpen.CompareAndSwap(false, true) {
		return
	}
	go func() {
		runtime.LockOSThread()
		defer func() {
			settingsOpen.Store(false)
			runtime.UnlockOSThread()
		}()
		runSettingsWindow()
	}()
}

type ShortcutEntry struct {
	Action  int    `json:"action"`
	Label   string `json:"label"`
	Section string `json:"section"`
	Mod     int    `json:"mod"`
	VK      int    `json:"vk"`
	Text    string `json:"text"`
}

type GeneralSettings struct {
	EdgeThreshold   int     `json:"edgeThreshold"`
	CornerSize      int     `json:"cornerSize"`
	ShortEdgeSize   int     `json:"shortEdgeSize"`
	GapSize         int     `json:"gapSize"`
	AlmostMaxWidth  float64 `json:"almostMaxWidth"`
	AlmostMaxHeight float64 `json:"almostMaxHeight"`
	SizeStep        int     `json:"sizeStep"`
}

type SnapAreasSettings struct {
	SnapByDragging   bool   `json:"snapByDragging"`
	RestoreSize      bool   `json:"restoreSize"`
	AnimateFootprint bool   `json:"animateFootprint"`
	TopLeft          string `json:"topLeft"`
	Top              string `json:"top"`
	TopRight         string `json:"topRight"`
	Left             string `json:"left"`
	Right            string `json:"right"`
	BottomLeft       string `json:"bottomLeft"`
	Bottom           string `json:"bottom"`
	BottomRight      string `json:"bottomRight"`
}

type SavePayload struct {
	General   GeneralSettings   `json:"general"`
	SnapAreas SnapAreasSettings `json:"snapAreas"`
}

type sEntry struct {
	action  WindowAction
	label   string
	section string
	mod     int
	vk      int
}

func allShortcutEntries() []sEntry {
	ca := MOD_CONTROL | MOD_ALT
	cas := MOD_CONTROL | MOD_ALT | MOD_SHIFT
	caw := MOD_CONTROL | MOD_ALT | MOD_WIN

	return []sEntry{
		{ActionLeftHalf, "Left Half", "Halves", ca, 0x25},
		{ActionRightHalf, "Right Half", "Halves", ca, 0x27},
		{ActionTopHalf, "Top Half", "Halves", ca, 0x26},
		{ActionBottomHalf, "Bottom Half", "Halves", ca, 0x28},
		{ActionCenterHalf, "Center Half", "Halves", 0, 0},
		{ActionTopLeft, "Top Left", "Quarters", ca, 0x55},
		{ActionTopRight, "Top Right", "Quarters", ca, 0x49},
		{ActionBottomLeft, "Bottom Left", "Quarters", ca, 0x4A},
		{ActionBottomRight, "Bottom Right", "Quarters", ca, 0x4B},
		{ActionFirstThird, "First Third", "Thirds", ca, 0x44},
		{ActionCenterThird, "Center Third", "Thirds", ca, 0x46},
		{ActionLastThird, "Last Third", "Thirds", ca, 0x47},
		{ActionFirstTwoThirds, "First Two Thirds", "Thirds", ca, 0x45},
		{ActionCenterTwoThirds, "Center Two Thirds", "Thirds", ca, 0x52},
		{ActionLastTwoThirds, "Last Two Thirds", "Thirds", ca, 0x54},
		{ActionTopVerticalThird, "Top Third", "Vertical Thirds", 0, 0},
		{ActionMiddleVerticalThird, "Middle Third", "Vertical Thirds", 0, 0},
		{ActionBottomVerticalThird, "Bottom Third", "Vertical Thirds", 0, 0},
		{ActionTopVerticalTwoThirds, "Top Two Thirds", "Vertical Thirds", 0, 0},
		{ActionBottomVerticalTwoThirds, "Bottom Two Thirds", "Vertical Thirds", 0, 0},
		{ActionFirstFourth, "First Fourth", "Fourths", 0, 0},
		{ActionSecondFourth, "Second Fourth", "Fourths", 0, 0},
		{ActionThirdFourth, "Third Fourth", "Fourths", 0, 0},
		{ActionLastFourth, "Last Fourth", "Fourths", 0, 0},
		{ActionFirstThreeFourths, "First Three Fourths", "Fourths", 0, 0},
		{ActionCenterThreeFourths, "Center Three Fourths", "Fourths", 0, 0},
		{ActionLastThreeFourths, "Last Three Fourths", "Fourths", 0, 0},
		{ActionTopLeftSixth, "Top Left Sixth", "Sixths", 0, 0},
		{ActionTopCenterSixth, "Top Center Sixth", "Sixths", 0, 0},
		{ActionTopRightSixth, "Top Right Sixth", "Sixths", 0, 0},
		{ActionBottomLeftSixth, "Bottom Left Sixth", "Sixths", 0, 0},
		{ActionBottomCenterSixth, "Bottom Center Sixth", "Sixths", 0, 0},
		{ActionBottomRightSixth, "Bottom Right Sixth", "Sixths", 0, 0},
		{ActionTopLeftEighth, "Top Left Eighth", "Eighths", 0, 0},
		{ActionTopCenterLeftEighth, "Top Center-Left Eighth", "Eighths", 0, 0},
		{ActionTopCenterRightEighth, "Top Center-Right Eighth", "Eighths", 0, 0},
		{ActionTopRightEighth, "Top Right Eighth", "Eighths", 0, 0},
		{ActionBottomLeftEighth, "Bottom Left Eighth", "Eighths", 0, 0},
		{ActionBottomCenterLeftEighth, "Bottom Center-Left Eighth", "Eighths", 0, 0},
		{ActionBottomCenterRightEighth, "Bottom Center-Right Eighth", "Eighths", 0, 0},
		{ActionBottomRightEighth, "Bottom Right Eighth", "Eighths", 0, 0},
		{ActionTopLeftNinth, "Top Left Ninth", "Ninths", 0, 0},
		{ActionTopCenterNinth, "Top Center Ninth", "Ninths", 0, 0},
		{ActionTopRightNinth, "Top Right Ninth", "Ninths", 0, 0},
		{ActionMiddleLeftNinth, "Middle Left Ninth", "Ninths", 0, 0},
		{ActionMiddleCenterNinth, "Middle Center Ninth", "Ninths", 0, 0},
		{ActionMiddleRightNinth, "Middle Right Ninth", "Ninths", 0, 0},
		{ActionBottomLeftNinth, "Bottom Left Ninth", "Ninths", 0, 0},
		{ActionBottomCenterNinth, "Bottom Center Ninth", "Ninths", 0, 0},
		{ActionBottomRightNinth, "Bottom Right Ninth", "Ninths", 0, 0},
		{ActionTopLeftThird, "Top Left Third", "Corner Thirds", 0, 0},
		{ActionTopRightThird, "Top Right Third", "Corner Thirds", 0, 0},
		{ActionBottomLeftThird, "Bottom Left Third", "Corner Thirds", 0, 0},
		{ActionBottomRightThird, "Bottom Right Third", "Corner Thirds", 0, 0},
		{ActionTopLeftTwelfth, "Top Left Twelfth", "Twelfths", 0, 0},
		{ActionTopCenterLeftTwelfth, "Top Center-Left Twelfth", "Twelfths", 0, 0},
		{ActionTopCenterRightTwelfth, "Top Center-Right Twelfth", "Twelfths", 0, 0},
		{ActionTopRightTwelfth, "Top Right Twelfth", "Twelfths", 0, 0},
		{ActionMiddleLeftTwelfth, "Middle Left Twelfth", "Twelfths", 0, 0},
		{ActionMiddleCenterLeftTwelfth, "Middle Center-Left Twelfth", "Twelfths", 0, 0},
		{ActionMiddleCenterRightTwelfth, "Middle Center-Right Twelfth", "Twelfths", 0, 0},
		{ActionMiddleRightTwelfth, "Middle Right Twelfth", "Twelfths", 0, 0},
		{ActionBottomLeftTwelfth, "Bottom Left Twelfth", "Twelfths", 0, 0},
		{ActionBottomCenterLeftTwelfth, "Bottom Center-Left Twelfth", "Twelfths", 0, 0},
		{ActionBottomCenterRightTwelfth, "Bottom Center-Right Twelfth", "Twelfths", 0, 0},
		{ActionBottomRightTwelfth, "Bottom Right Twelfth", "Twelfths", 0, 0},
		{ActionTopLeftSixteenth, "Top Left Sixteenth", "Sixteenths", 0, 0},
		{ActionTopCenterLeftSixteenth, "Top Center-Left Sixteenth", "Sixteenths", 0, 0},
		{ActionTopCenterRightSixteenth, "Top Center-Right Sixteenth", "Sixteenths", 0, 0},
		{ActionTopRightSixteenth, "Top Right Sixteenth", "Sixteenths", 0, 0},
		{ActionUpperMiddleLeftSixteenth, "Upper-Mid Left Sixteenth", "Sixteenths", 0, 0},
		{ActionUpperMiddleCenterLeftSixteenth, "Upper-Mid Center-Left Sixteenth", "Sixteenths", 0, 0},
		{ActionUpperMiddleCenterRightSixteenth, "Upper-Mid Center-Right Sixteenth", "Sixteenths", 0, 0},
		{ActionUpperMiddleRightSixteenth, "Upper-Mid Right Sixteenth", "Sixteenths", 0, 0},
		{ActionLowerMiddleLeftSixteenth, "Lower-Mid Left Sixteenth", "Sixteenths", 0, 0},
		{ActionLowerMiddleCenterLeftSixteenth, "Lower-Mid Center-Left Sixteenth", "Sixteenths", 0, 0},
		{ActionLowerMiddleCenterRightSixteenth, "Lower-Mid Center-Right Sixteenth", "Sixteenths", 0, 0},
		{ActionLowerMiddleRightSixteenth, "Lower-Mid Right Sixteenth", "Sixteenths", 0, 0},
		{ActionBottomLeftSixteenth, "Bottom Left Sixteenth", "Sixteenths", 0, 0},
		{ActionBottomCenterLeftSixteenth, "Bottom Center-Left Sixteenth", "Sixteenths", 0, 0},
		{ActionBottomCenterRightSixteenth, "Bottom Center-Right Sixteenth", "Sixteenths", 0, 0},
		{ActionBottomRightSixteenth, "Bottom Right Sixteenth", "Sixteenths", 0, 0},
		{ActionMaximize, "Maximize", "Resize", ca, 0x0D},
		{ActionAlmostMaximize, "Almost Maximize", "Resize", cas, 0x0D},
		{ActionMaximizeHeight, "Maximize Height", "Resize", cas, 0x26},
		{ActionLarger, "Larger", "Resize", ca, 0xBB},
		{ActionSmaller, "Smaller", "Resize", ca, 0xBD},
		{ActionLargerWidth, "Larger Width", "Resize", 0, 0},
		{ActionSmallerWidth, "Smaller Width", "Resize", 0, 0},
		{ActionLargerHeight, "Larger Height", "Resize", 0, 0},
		{ActionSmallerHeight, "Smaller Height", "Resize", 0, 0},
		{ActionCenter, "Center", "Center", ca, 0x43},
		{ActionCenterProminently, "Center Prominently", "Center", 0, 0},
		{ActionMoveLeft, "Move Left", "Move", 0, 0},
		{ActionMoveRight, "Move Right", "Move", 0, 0},
		{ActionMoveUp, "Move Up", "Move", 0, 0},
		{ActionMoveDown, "Move Down", "Move", 0, 0},
		{ActionNextDisplay, "Next Display", "Display", caw, 0x27},
		{ActionPreviousDisplay, "Previous Display", "Display", caw, 0x25},
		{ActionRestore, "Restore", "Special", ca, 0x08},
	}
}

func buildShortcutEntries() []ShortcutEntry {
	out := make([]ShortcutEntry, 0)
	for _, e := range allShortcutEntries() {
		out = append(out, ShortcutEntry{
			Action:  int(e.action),
			Label:   e.label,
			Section: e.section,
			Mod:     e.mod,
			VK:      e.vk,
			Text:    shortcutChipText(e.mod, e.vk),
		})
	}
	return out
}

func shortcutChipText(mod, vk int) string {
	if mod == 0 && vk == 0 {
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

func runSettingsWindow() {
	w := webview.NewWithOptions(webview.WebViewOptions{Debug: false, Window: nil})
	if w == nil {
		fmt.Println("warn: WebView2 not available")
		return
	}
	defer w.Destroy()

	w.SetTitle(appName + " — Settings")
	w.SetSize(720, 640, webview.HintNone)

	w.Bind("goSaveSettings", func(payload string) {
		var p SavePayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			fmt.Fprintf(os.Stderr, "settings save: %v\n", err)
			return
		}

		// Build a new config value — do NOT mutate appCfg directly here because
		// this callback runs in the WebView goroutine while the main OS thread
		// reads appCfg from WinEvent callbacks. We post the new config through
		// cfgUpdateCh so the main thread applies it safely via wmApplyCfg.
		g := p.General
		newCfg := *appCfg // shallow copy of current values
		newCfg.EdgeThresholdPx = g.EdgeThreshold
		newCfg.CornerSnapAreaSize = g.CornerSize
		newCfg.ShortEdgeSnapAreaSize = g.ShortEdgeSize
		newCfg.GapSize = g.GapSize
		newCfg.AlmostMaximizeWidth = g.AlmostMaxWidth / 100.0
		newCfg.AlmostMaximizeHeight = g.AlmostMaxHeight / 100.0
		newCfg.SizeOffset = g.SizeStep

		sa := p.SnapAreas
		newCfg.SnapByDragging = sa.SnapByDragging
		newCfg.RestoreSize = sa.RestoreSize
		newCfg.AnimateFootprint = sa.AnimateFootprint
		newCfg.ZoneTopLeft = sa.TopLeft
		newCfg.ZoneTop = sa.Top
		newCfg.ZoneTopRight = sa.TopRight
		newCfg.ZoneLeft = sa.Left
		newCfg.ZoneRight = sa.Right
		newCfg.ZoneBottomLeft = sa.BottomLeft
		newCfg.ZoneBottom = sa.Bottom
		newCfg.ZoneBottomRight = sa.BottomRight

		if appCfgStore != nil {
			if err := appCfgStore.Save(&newCfg); err != nil {
				fmt.Fprintf(os.Stderr, "settings save to disk: %v\n", err)
			}
		}

		// Deliver to main thread (non-blocking: drop if channel already full,
		// meaning a previous save hasn't been consumed yet — the next will win).
		select {
		case cfgUpdateCh <- &newCfg:
		default:
		}
		postToMain(wmApplyCfg, 0, 0)

		// w.Eval() cannot be called from inside a w.Bind() callback —
		// it tries to dispatch back to the WebView thread that is currently
		// waiting for this callback to return, causing a deadlock.
		// Run on a separate goroutine so the callback returns first.
		go func() {
			time.Sleep(30 * time.Millisecond) // let the callback return
			w.Eval("window.showSaved()")
			time.Sleep(600 * time.Millisecond) // show "Saved ✓" briefly
			w.Terminate()                      // close the window
		}()
	})

	w.Bind("goOpenConfigFile", func() {
		if appCfgStore != nil {
			postToMain(wmOpenConfig, 0, 0)
		}
	})

	w.Bind("goSetWindowsSnap", func(enabled bool) {
		if enabled {
			_ = EnableWindowsSnap()
		} else {
			_ = DisableWindowsSnap()
		}
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
		w32.ShellExecute(0, "open", releasesURL, "", "", w32.SW_SHOWNORMAL)
	})

	winSnap, _ := json.Marshal(WindowsSnapEnabled())
	shortcuts, _ := json.Marshal(buildShortcutEntries())
	general, _ := json.Marshal(GeneralSettings{
		EdgeThreshold:   appCfg.EdgeThresholdPx,
		CornerSize:      appCfg.CornerSnapAreaSize,
		ShortEdgeSize:   appCfg.ShortEdgeSnapAreaSize,
		GapSize:         appCfg.GapSize,
		AlmostMaxWidth:  appCfg.AlmostMaximizeWidth * 100,
		AlmostMaxHeight: appCfg.AlmostMaximizeHeight * 100,
		SizeStep:        appCfg.SizeOffset,
	})
	snapAreas, _ := json.Marshal(SnapAreasSettings{
		SnapByDragging:   appCfg.SnapByDragging,
		RestoreSize:      appCfg.RestoreSize,
		AnimateFootprint: appCfg.AnimateFootprint,
		TopLeft:          appCfg.ZoneTopLeft,
		Top:              appCfg.ZoneTop,
		TopRight:         appCfg.ZoneTopRight,
		Left:             appCfg.ZoneLeft,
		Right:            appCfg.ZoneRight,
		BottomLeft:       appCfg.ZoneBottomLeft,
		Bottom:           appCfg.ZoneBottom,
		BottomRight:      appCfg.ZoneBottomRight,
	})
	version, _ := json.Marshal(appVersion)
	w.Init(fmt.Sprintf(`
		window._shortcuts = %s;
		window._general   = %s;
		window._snapAreas = %s;
		window._winSnap   = %s;
		window._version   = %s;
	`, string(shortcuts), string(general), string(snapAreas), string(winSnap), string(version)))

	tmpPath := filepath.Join(os.TempDir(), "snapflow_settings.html")
	if err := os.WriteFile(tmpPath, []byte(settingsHTML), 0644); err != nil {
		fmt.Printf("settings html write: %v\n", err)
		return
	}
	defer os.Remove(tmpPath)

	fileURL := "file:///" + strings.ReplaceAll(tmpPath, "\\", "/")
	w.Navigate(fileURL)
	w.Run()
}
