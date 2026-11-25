package main

import (
	"fmt"
	"os"
	"runtime"
	"sync/atomic"

	"snapflow/w32ex"

	"github.com/gonutz/w32/v2"
	webview "github.com/jchv/go-webview2"
)

// settingsTestHook lets GUI tests drive the real settings page.
var settingsTestHook func(webview.WebView)

var (
	settingsOpen atomic.Bool
	settingsHWND atomic.Uintptr // window handle while the settings window is open
)

// openSettings opens the settings window, or brings it to the front if it
// is already open (previously a second click silently did nothing).
func openSettings() {
	if !settingsOpen.CompareAndSwap(false, true) {
		if h := w32.HWND(settingsHWND.Load()); h != 0 {
			if w32ex.IsIconic(h) {
				w32.ShowWindow(h, w32.SW_RESTORE)
			}
			w32.SetForegroundWindow(h)
		}
		return
	}
	go settingsThread()
}

// settingsThread hosts the settings window on its own locked OS thread.
func settingsThread() {
	runtime.LockOSThread()
	defer func() {
		settingsHWND.Store(0)
		settingsOpen.Store(false)
		runtime.UnlockOSThread()
		// Never leave global hotkeys unregistered if the window closed
		// while a shortcut was being recorded.
		postToMain(wmSuspendHotkeys, 0, 0)
	}()
	runSettingsWindow()
}

// pendingCfg holds the newest config submitted from a non-main goroutine;
// the main thread picks it up on wmApplyCfg. Unlike a 1-slot channel with a
// non-blocking send, a newer submission can never be silently dropped.
var pendingCfg atomic.Pointer[AppConfig]

// submitConfig persists cfg and hands it to the main thread.
func submitConfig(cfg *AppConfig) {
	liveCfg.Store(cfg)
	if appCfgStore != nil {
		if err := appCfgStore.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "settings save to disk: %v\n", err)
		}
	}
	pendingCfg.Store(cfg)
	postToMain(wmApplyCfg, 0, 0)
}

type ShortcutEntry struct {
	Action  int    `json:"action"`
	Label   string `json:"label"`
	Section string `json:"section"`
	Mod     int    `json:"mod"`
	VK      int    `json:"vk"`
	Text    string `json:"text"`
	DefMod  int    `json:"defMod"`
	DefVK   int    `json:"defVk"`
	DefText string `json:"defText"`
}

type ShortcutBinding struct {
	Action int `json:"action"`
	Mod    int `json:"mod"`
	VK     int `json:"vk"`
}

type GeneralSettings struct {
	EdgeThreshold   int     `json:"edgeThreshold"`
	CornerSize      int     `json:"cornerSize"`
	ShortEdgeSize   int     `json:"shortEdgeSize"`
	GapSize         int     `json:"gapSize"`
	AlmostMaxWidth  float64 `json:"almostMaxWidth"`
	AlmostMaxHeight float64 `json:"almostMaxHeight"`
	SizeStep        int     `json:"sizeStep"`
	CycleHalves     bool    `json:"cycleHalves"`
	PreviewOpacity  int     `json:"previewOpacity"` // percent
}

type SnapAreasSettings struct {
	SnapByDragging   bool   `json:"snapByDragging"`
	RestoreSize      bool   `json:"restoreSize"`
	AnimateFootprint bool   `json:"animateFootprint"`
	SquareCorners    bool   `json:"squareCorners"`
	AnimateWindows   bool   `json:"animateWindows"`
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
	Shortcuts []ShortcutBinding `json:"shortcuts"`
	Language  string            `json:"language"`
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
