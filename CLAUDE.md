# SnapFlow — Codebase Documentation

SnapFlow is a Windows window-manager inspired by [Rectangle](https://github.com/rxhanson/Rectangle) (macOS).
It runs as a system-tray application, intercepts window-drag events and keyboard shortcuts,
and repositions windows into predefined snap positions (halves, thirds, quarters, sixths, eighths, …).

---

## Architecture overview

```
main.go            entry-point, startup/shutdown, executeAction (hotkey dispatch, ½→⅔→⅓ cycling)
config.go          AppConfig, JSON persistence (atomic save, migration, broken-file recovery)
actions.go         WindowAction enum + Calculate() — pure geometry, no Win32 calls
snapzones.go       SnapZonesForWork() — maps screen edges/corners to WindowActions
gesture.go         GestureInterpreter — zone lookup + reversible compound thirds
drag.go            DragManager — WinEvent hooks, move-vs-resize detection, unsnap-restore
overlay.go         OverlayRenderer — animated per-pixel-alpha snap preview (own thread)
window.go          resizeToRect / placeFrame (pixel-exact placement), multi-monitor, window filter
history.go         WindowHistory — restore rect, last snapped rect, cycle state
corners.go         square corners on snapped windows (DWMWA_WINDOW_CORNER_PREFERENCE)
hotkey.go          configurable hotkeys (register/re-register), main message loop, applyConfig
conflict.go        hotkey-conflict + elevated-window notifications (tray balloons)
tray.go            tray icon/menu, notifications, TaskbarCreated re-registration
i18n.go            EN/ES strings for native UI (tray, notifications)
autorun.go         HKCU Run key (quoted os.Executable path)
winsnap.go         Windows Snap on/off via SPI_SETWINARRANGING (applies instantly)
update.go          GitHub release check (notification, never a modal dialog)
settings.go        settings data types, openSettings (single instance + focus), submitConfig
settings_window.go WebView2 window: bindings, JSON injection, shortcut overrides
settings_html.go   settingsHTML const — full HTML/CSS/JS for the settings UI
monitor.go         Monitors() / monitorWork() — leak-free EnumDisplayMonitors
win32.go           shared raw user32/gdi32 procs + WNDCLASSEXW
generate.go        //go:generate goversioninfo — embeds icon + version into .exe
w32ex/             Win32 wrappers missing from gonutz/w32 (cached procs, shared WinEvent callback)
```

---

## Data flow

### Keyboard shortcut → snap
```
WM_HOTKEY (main thread msgLoop, hwnd=0)
  → runAction(action) → executeAction(foregroundHwnd, action)
      frameRect(hwnd)                      [visible frame, not GetWindowRect]
      halfCycles: repeated Left/Right Half → ½, ⅔, ⅓ (WindowHistory.NextCycle)
      action.Calculate(work, frame, cfg)   [pure geometry]
      resizeToRect(hwnd, target)           [placeFrame + delayed verification]
      windowHistory.RecordSnap(before, got)
```

### Drag → snap
```
EVENT_SYSTEM_MOVESIZESTART (global hook, main thread)
  → onMoveStart: isZonableWindow, WM_NCHITTEST (ignore border *resizes*),
    unsnap-restore (RestoreSize), install per-process LOCATIONCHANGE hook,
    GestureInterpreter.Begin
EVENT_OBJECT_LOCATIONCHANGE (hook scoped to the dragged window's process)
  → onLocationChange: 4px dead zone, gest.Update(cursor) → overlay.Show/Hide
EVENT_SYSTEM_MOVESIZEEND
  → onMoveEnd: final gest.Update at release point, Esc-cancel detection,
    resizeToRect / maximize, RecordSnap, THEN overlay.Hide (fades over the
    placed window), unhook LOCATIONCHANGE
```

### Settings save → live config update
```
WebView2 goroutine (goSaveSettings)
  → newCfg := liveCfg.Load().clone() + payload  (never mutates the live config)
  → submitConfig: liveCfg.Store, atomic save, pendingCfg.Store, post wmApplyCfg
main thread msgLoop (wmApplyCfg)
  → applyConfig: appCfg = newCfg, gesture.UpdateConfig, overlay.Configure,
    re-register hotkeys if they changed (conflicts → notification)
```

---

## Key files in detail

### `window.go` — placement engine

All geometry is in **visible-frame** coordinates (`DwmGetWindowAttribute(EXTENDED_FRAME_BOUNDS)`,
falling back to `GetWindowRect`). The process is **Per-Monitor-V2 DPI aware**, so frame
bounds, window rects and work areas are all physical pixels — **never rescale them**
(the old `scaleRect(GetDpiForWindow/GetDeviceCaps)` broke DPI-unaware apps and mixed-DPI setups).

`placeFrame(hwnd, target, work)`:
1. `SW_RESTORE` if maximised/minimised (border metrics differ in those states)
2. `extendFlushEdges`: edges flush with the work area go 1px outward (hides the Win11 1px accent border)
3. position with the current invisible-border insets
4. second pass if the result differs (DPI change after a cross-monitor move)
5. min-size apps: `reanchor` keeps an oversized window hugging the intended edge and inside the work area

`resizeToRect` additionally schedules a verification 150ms later (per-window generation
counter, skipped while dragging) for apps that re-apply their own bounds.
`SetWindowPos` returning ERROR_ACCESS_DENIED = elevated target → one notification per session.

### `drag.go`

- **Move vs resize**: MOVESIZESTART fires for both. `WM_NCHITTEST` (via `SendMessageTimeout`,
  60ms, abort-if-hung) on the cursor; HTLEFT..HTBOTTOMRIGHT / HTBORDER → ignored.
- **Clicks aren't drags**: a 4px dead zone; clicking a title bar that sits on a snap edge no longer snaps.
- **Esc cancel**: if the frame at release equals the start frame after moving → no snap.
- **Unsnap-restore**: a window still at its snapped rect gets its original size back when a drag
  starts (cursor kept at the same relative title-bar position); fallback at drop if the app's move loop overrode it.
- The LOCATIONCHANGE hook is installed per drag and scoped to the window's process (a global one fires constantly for carets/progress bars).

### `gesture.go`

Zone lookup via `ActionForCursor`. The bottom edge with `"compoundThirds"` yields the internal
marker `actionCompoundThirds` (-2), resolved as a span from the **anchor third** (where the
cursor entered the bottom zone) to the current third — reversible:
```
single third → First/Center/Last Third
first+center → First Two Thirds     center+last → Last Two Thirds     all → Maximize
```
A plain `"bottomHalf"` setting is a real Bottom Half; `"none"` disables a zone.

### `overlay.go`

Own locked OS thread with its own window + message pump (idle in `GetMessage`, woken by
`PostThreadMessage`). Rendering: `UpdateLayeredWindow` with a reusable top-down 32bpp DIB,
anti-aliased rounded rect via SDF (only corner bands per-pixel; row middles are memmove fills —
~1ms for 1440p). Animations (ease-out cubic, DwmFlush-paced): appear = fade + 94%→100% zoom,
retarget = glide, hide = fade. Colour = Windows accent (`HKCU\Software\Microsoft\Windows\DWM\AccentColor`),
fill opacity = `FootprintAlpha`, `AnimateFootprint=false` → instant.

### `corners.go`

Windows 11 squares the corners of windows it snaps. After a placement, `updateCorners` sets
`DWMWCP_DONOTROUND` when `SquareCorners` is on and the action `IsTiling()` (grid positions;
not Center/Almost Maximize/resize steps). Works cross-process. Only windows SnapFlow squared
are reset to `DWMWCP_DEFAULT` (on Restore, unsnap-drag, free drop, option off, or quit), so
apps that pick their own corner style are never touched. No-op on Windows 10.

### `history.go`

Per window: `restore` (free-floating frame before the first snap), `snapped` (frame SnapFlow
last applied), cycle state. Snapping an already-snapped window keeps the original restore rect.
Entries for dead HWNDs are pruned once the map exceeds 128.

### `hotkey.go` / shortcuts

Defaults live in `allShortcutEntries()` (settings.go). `AppConfig.Shortcuts` stores only
overrides (`"<action>": {mod, vk}`, `{0,0}` = disabled). `registerHotKeys(cfg)` unregisters
everything and re-registers (ids = 1000+action, `MOD_NOREPEAT`). While the settings page records a
shortcut it posts `wmSuspendHotkeys` so the keys reach the page.

### `settings*.go` + `settings_html.go`

WebView2 via `go-webview2`, data dir `%AppData%\SnapFlow\WebView2`, size scaled by monitor DPI,
centered; a second open focuses the existing window. Injected globals: `_shortcuts`
(with defaults), `_general`, `_snapAreas`, `_winSnap`, `_version`, `_autorun`, `_lang`.
Bindings: `goSaveSettings`, `goSetLanguage`, `goSuspendHotkeys`, `goShortcutText`, `goClose`,
`goOpenConfigFile`, `goSetWindowsSnap`, `goSetAutorun`, `goCheckUpdates`, `goOpenRepo`.
**Never call `w.Eval`/`w.Terminate` inside a Bind callback** — use `w.Dispatch` from a goroutine.
The General tab is rebuilt on language change; `readGeneral()` preserves unsaved edits.

---

## Threading model

| Thread/Goroutine | What runs there |
|---|---|
| Main OS thread (locked) | msgLoop, WM_HOTKEY, WinEvent callbacks, tray window, applyConfig. Sole owner of `appCfg`. |
| Overlay thread (locked) | overlay window, its message pump, animation + rendering |
| WebView goroutine (locked) | settings window (`w.Run()`) |
| Short goroutines | placement verification, update check |

Cross-thread config access uses `liveCfg atomic.Pointer[AppConfig]` (read anywhere) and
`pendingCfg` (handoff to the main thread). `GestureInterpreter.cfg` has its own RWMutex.
Tray state (`trayNID`, balloon action) is mutex-guarded; notifications can come from any goroutine.

**Syscall callbacks**: Go can only create a limited number and never frees them. WinEvent hooks share
one native callback (dispatch map in `w32ex`), `EnumDisplayMonitors` uses one shared callback.
Never call `syscall.NewCallback` per operation.

---

## Testing

```powershell
go test ./...                                   # pure logic (CI)
$env:SNAPFLOW_GUI_TESTS=1; go test -run GUI -v . # real windows: pixel-exact placement, overlay
```
`window_gui_test.go` briefly shows a window and the overlay; skipped unless the env var is set.

---

## Build system

### Local build
```powershell
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest   # once
.\scripts\make-icon.ps1     # Windows PowerShell 5.1 (System.Drawing)
go generate
go build -ldflags "-H=windowsgui -s -w -X main.appVersion=1.2.3" -o snapflow.exe .
```

### CI/CD (GitHub Actions)
- **`ci.yml`**: lint (gofmt, go mod tidy, `GOOS=windows go vet`) on Ubuntu; build + test on Windows.
- **`release.yml`**: every push to `main` → version `MAJOR.MINOR` (from `VERSION`) `.` commit count,
  builds exe + Inno Setup installer, publishes release `vX.Y.Z`.

---

## Known issues / TODOs

| Issue | File | Severity |
|---|---|---|
| Windows of elevated apps can't be moved (UIPI) unless SnapFlow runs elevated — user is notified once | window.go | By design |
| Shortcut/zone labels in the settings list are English-only | settings.go | Minor |
| `ShortcutSet` config field is persisted but never read | config.go | Minor |
| `DisplayName()` on `WindowAction` is unused | actions.go | Minor (dead code) |

---

## Adding a new snap action

1. Add a constant to `actions.go` (next free integer)
2. Add a `case` to `Calculate()`; if it depends on the current frame, add it to `UsesCurrentRect()`
3. Add to `IsDragSnappable()` exclusions if it must not be drag-triggered
4. Add to `allShortcutEntries()` in `settings.go` (section + default hotkey or `0, 0`)
5. Add diagram coords to `ACT` in `settingsHTML`
6. If zone-configurable: `zoneNameToAction` (snapzones.go) + `ALL_ZONE_OPTS` (settingsHTML)

## Adding a new settings field

1. Field in `AppConfig`, default in `defaultConfig()`, validation in `fillDefaults()`
   (bump `configVersion` + `migrate()` if old files need a different value)
2. UI: add to `GeneralSettings`/`SnapAreasSettings`, inject in `runSettingsWindow`, apply in `goSaveSettings`
3. JS: `buildGeneral`/`readGeneral`/`save`/`resetDefaults`, keys in both `LANGS.en` and `LANGS.es`
4. If the native side reacts live, handle it in `applyConfig` (hotkey.go)
