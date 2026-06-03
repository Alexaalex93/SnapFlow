# SnapFlow — Codebase Documentation

SnapFlow is a Windows window-manager inspired by [Rectangle](https://github.com/rxhanson/Rectangle) (macOS).
It runs as a system-tray application, intercepts window-drag events and keyboard shortcuts,
and repositions windows into predefined snap positions (halves, thirds, quarters, sixths, eighths, …).

---

## Architecture overview

```
main.go          entry-point, startup, hotkey registration, Win32 message loop
config.go        AppConfig struct, JSON persistence (~/.config/SnapFlow/config.json)
actions.go       WindowAction enum + Calculate() — pure geometry, no Win32 calls
snapzones.go     SnapZonesForWork() — maps screen edges/corners to WindowActions
gesture.go       GestureInterpreter — drag tracking, compound-thirds logic
drag.go          DragManager — WinEvent hooks, calls gesture + overlay + window
overlay.go       OverlayRenderer — semi-transparent blue snap-preview window
window.go        applyResize / resizeToRect + isZonableWindow filter
hotkey.go        RegisterHotKey wrapper, Win32 message loop (msgLoop)
conflict.go      Shortcut-conflict dialog + GPU settings helper
history.go       WindowHistory — stores pre-snap rect for ActionRestore
autorun.go       Windows Run registry key enable/disable
winsnap.go       Windows AeroSnap registry disable/enable
tray.go          System tray icon + context menu
settings.go      Settings window (WebView2): data types, Go←→JS bindings
settings_html.go settingsHTML const — full HTML/CSS/JS for the settings UI
generate.go      //go:generate goversioninfo — embeds icon + version into .exe
monitor.go       EnumMonitors helper
w32ex/           Thin wrappers for Win32 functions missing from gonutz/w32
```

---

## Data flow

### Keyboard shortcut → snap
```
WM_HOTKEY (main thread msgLoop)
  → HotKey.callback()
  → makeActionCallback(action)
  → executeAction(hwnd, action)
  → action.Calculate(work, cur, cfg)   [pure geometry]
  → resizeToRect(hwnd, target)          [Win32 SetWindowPos]
```

### Drag → snap
```
WinEvent EVENT_SYSTEM_MOVESIZESTART  (main thread, WINEVENT_OUTOFCONTEXT)
  → DragManager.onMoveStart(hwnd)
      isZonableWindow(hwnd)            [filter: skip system windows]
      GestureInterpreter.Begin(hwnd, work)

WinEvent EVENT_OBJECT_LOCATIONCHANGE  (main thread, per mouse-move)
  → DragManager.onLocationChange(hwnd)
      GestureInterpreter.Update(cursor, work)
        ActionForCursor(cursor, zones) → WindowAction
        bottomThirdsAction() if in bottom edge
      action.Calculate(work, preSnap, cfg) → target RECT
      OverlayRenderer.Show(target)     [blue preview]

WinEvent EVENT_SYSTEM_MOVESIZEEND    (main thread)
  → DragManager.onMoveEnd(hwnd)
      GestureInterpreter.CurrentAction()
      GestureInterpreter.End()
      windowHistory.Save(hwnd, current)
      resizeToRect(hwnd, target)
```

### Settings save → live config update
```
WebView2 goroutine (goSaveSettings JS binding)
  → JSON unmarshal into SavePayload
  → update appCfg fields
  → appCfgStore.Save(appCfg)
  → dragMgr.gest.UpdateConfig(appCfg)   [RWMutex protected]
```

---

## Key files in detail

### `actions.go`

Contains the `WindowAction` enum (integer values compatible with Rectangle's Swift enum),
and `(a WindowAction).Calculate(work, cur w32.RECT, cfg *AppConfig) w32.RECT`.

`Calculate` is **pure geometry** — no Win32 calls, no side effects. Takes:
- `work` — monitor work area (excludes taskbar)
- `cur` — current window frame rect (DWM extended frame bounds, not `GetWindowRect`)
- `cfg` — gap size, AlmostMaximize percentages, resize step sizes

Helper functions:
- `colRect(work, col, totalCols)` — horizontal grid column bounds
- `rowRect(work, row, totalRows)` — vertical grid row bounds
- `gridCell(work, col, row, totalCols, totalRows)` — grid cell rect
- `applyGaps(r, edges, halfGap)` — shrinks a rect by gap/2 on specified edges
- `clampToWork(r, work)` — translates/shrinks r to fit inside work area

**Known limitation**: `ActionMaximize` returns `work` from `Calculate` (for overlay preview only);
the actual maximize uses `SW_MAXIMIZE` in `maximize()`.

### `snapzones.go`

`SnapZonesForWork(work, cfg)` returns `[]SnapZone` — a list of `{Region w32.RECT, Action WindowAction}`.

The zones cover screen edges and corners. Layout:
```
[TL corner][────── Top edge (Maximize) ──────][TR corner]
[Left edge top]                          [Right edge top]
[Left edge mid (LeftHalf)]               [Right edge mid (RightHalf)]
[Left edge bot]                          [Right edge bot]
[BL corner][── Bottom edge (compound thirds) ─][BR corner]
```

Zone assignments are configurable per-zone in `AppConfig.Zone*` fields.
The `"compoundThirds"` zone value maps to `ActionBottomHalf` internally —
the gesture engine then activates the bottom-thirds tracker on that action.

### `gesture.go`

`GestureInterpreter` holds the drag state for one active drag.

**Compound-thirds logic** (`bottomThirdsAction`):
When the cursor is in the bottom edge zone, `xMin`/`xMax` track the horizontal
extent of the cursor path. The action is derived from which thirds were covered:
```
xMin..xMax covers [first third only]          → FirstThird
xMin..xMax covers [center third only]         → CenterThird
xMin..xMax covers [last third only]           → LastThird
xMin..xMax covers [first + center]            → FirstTwoThirds
xMin..xMax covers [center + last]             → LastTwoThirds
xMin..xMax covers [all three]                 → Maximize
```

**Threading**: `GestureInterpreter` is called from the WinEvent callback (main OS thread)
for `Begin`/`Update`/`End`/`CurrentAction`, and from the WebView goroutine for `UpdateConfig`.
A `sync.RWMutex` (`cfgMu`) protects the `cfg` pointer.

### `drag.go`

`DragManager` installs three `SetWinEventHook` hooks and translates them into
`GestureInterpreter` calls + `OverlayRenderer` calls + `resizeToRect` calls.

**Important invariant**: `d.active` is only set to `true` inside the mutex in
`onMoveStart`, and only if `GetWindowRect` succeeds. This guarantees `d.preSnap`
is always valid when `onMoveEnd` uses it.

**Throttling**: `onLocationChange` debounces overlay hide calls by 80ms to avoid
flickering when the cursor briefly leaves a snap zone.

### `overlay.go`

The overlay is a `WS_POPUP | WS_EX_LAYERED | WS_EX_TOPMOST | WS_EX_TRANSPARENT` window.
Alpha fades in/out via `SetLayeredWindowAttributes(LWA_ALPHA)` in an animation goroutine
running at ~70 fps.

**Painting**: `WM_ERASEBKGND` returns 1 (suppress default). `WM_PAINT` paints the
solid blue (`#2563EB`) fill using the HDC from `BeginPaint`. `InvalidateRect` +
`UpdateWindow` are called immediately after `SetWindowPos` to force a repaint before
the fade-in starts (prevents the black-flash on first show with DWM composition).

**Corner radius**: Applied via `SetWindowRgn(CreateRoundRectRgn(...))`.

### `window.go`

`isZonableWindow(hwnd)` — the main window filter. A window is zonal if:
1. It is the root window (`GetAncestor(GA_ROOT) == hwnd`)
2. It is visible
3. It is not the desktop or shell window
4. Its class name is not in `isSystemClassName`
5. It is not `WS_CHILD` or `WS_DISABLED`
6. It is not `WS_EX_TOOLWINDOW` (except `ApplicationFrameWindow` — UWP host)
7. It is not a `WS_POPUP` without `WS_CAPTION`

`applyResize(hwnd, f)` — the resize engine:
1. Gets the DWM extended frame bounds (`DwmGetWindowAttributeEXTENDED_FRAME_BOUNDS`)
   which excludes the invisible Win10/11 shadow border
2. Computes the invisible-border delta between `GetWindowRect` and frame bounds
3. Calls `f(work, frameBounds)` to get the target frame rect
4. Adds back the border offsets before calling `SetWindowPos`

This correctly handles the invisible 7–8 px shadow border on Windows 10/11.

### `history.go`

`WindowHistory` is a `map[w32.HWND]w32.RECT` protected by a `sync.Mutex`.
`Save` stores the current rect before a snap. `Pop` retrieves and deletes it.
Used by `ActionRestore` to undo the last snap.

**Note**: The map grows unboundedly. Closed windows stay in the map until the
corresponding HWND is reused (very rare in practice, but worth noting).

### `config.go`

`AppConfig` fields:

| Field | Default | Description |
|---|---|---|
| `EdgeThresholdPx` | 46 | Pixels from edge to activate snap zone |
| `CornerSnapAreaSize` | 20 | Corner hot-zone square size (px) |
| `ShortEdgeSnapAreaSize` | 145 | Upper/lower band height on left/right edges |
| `GapSize` | 0 | Gap between snapped windows (px) |
| `AlmostMaximizeWidth` | 0.9 | Almost-maximize width as fraction of monitor |
| `AlmostMaximizeHeight` | 0.9 | Almost-maximize height as fraction of monitor |
| `SizeOffset` | 30 | Step size for Larger/Smaller/LargerHeight/SmallerHeight |
| `WidthStepSize` | 30 | Step size for LargerWidth/SmallerWidth |
| `FootprintAlpha` | 0.3 | Persisted but currently unused (overlay alpha is hardcoded) |
| `ShortcutSet` | "rectangle" | Persisted but currently unused |
| `SnapByDragging` | true | Enable drag-to-snap |
| `RestoreSize` | false | Restore window size when unsnapping (not yet implemented) |
| `AnimateFootprint` | true | Animate the snap preview overlay |
| `Zone*` | various | Per-zone action names (see `zoneNameToAction` in snapzones.go) |

Config is stored at `%AppData%\SnapFlow\config.json`.
`fillDefaults` backfills any zero/empty values from `defaultConfig()` on load.

### `settings.go` + `settings_html.go`

The settings window uses **WebView2** (Edge-based embedded browser) via `go-webview2`.
The HTML/CSS/JS lives entirely in `settingsHTML` const in `settings_html.go`.

Go→JS communication:
- `w.Init(...)` — injects `window._shortcuts`, `window._general`, `window._snapAreas`, `window._winSnap` before page load
- `w.Bind(name, fn)` — exposes Go functions callable from JS: `goSaveSettings`, `goOpenConfigFile`, `goSetWindowsSnap`
- `w.Eval(js)` — called after save to trigger `window.showSaved()`

The HTML is written to a temp file (`%TEMP%\snapflow_settings.html`) and navigated to via
`file://` because WebView2 blocks `data:` URLs by default.

The settings window is **single-instance** (guarded by `settingsOpen atomic.Bool`).

---

## Threading model

SnapFlow uses two OS threads and several goroutines:

| Thread/Goroutine | What runs there |
|---|---|
| Main OS thread (locked) | Win32 message loop, WM_HOTKEY, WinEvent callbacks, tray messages, overlay WndProc |
| WebView goroutine | Settings window (WebView2 runs its own message pump in `w.Run()`) |
| Overlay animation goroutine | `OverlayRenderer.animLoop()` — reads cmdCh, calls SetLayeredWindowAttributes |
| Signal goroutine | os/signal watcher for Ctrl-C |

**WinEvent hooks** are installed with `WINEVENT_OUTOFCONTEXT`, meaning callbacks fire
on the thread that called `SetWinEventHook` — which is the main thread. All drag callbacks
therefore run on the main OS thread, same as the message loop.

**`GestureInterpreter.UpdateConfig`** is the only cross-goroutine write (WebView → main).
It is protected by `cfgMu sync.RWMutex`.

**`OverlayRenderer`** uses a buffered channel (`cmdCh`) to pass commands from any goroutine
to the animation loop goroutine.

---

## Build system

### Local build
```powershell
# Install tools (once)
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest

# Regenerate icon (if changed)
.\scripts\make-icon.ps1

# Embed icon + version info into .exe
go generate

# Build
go build -ldflags "-H=windowsgui -s -w" -o snapflow.exe .
```

### Full release build (exe + installer)
```powershell
.\scripts\build-release.ps1 -Version 1.2.3
# Output: dist\snapflow.exe, dist\SnapFlow-1.2.3-Setup.exe
```

### CI/CD (GitHub Actions)
- **`ci.yml`**: runs on every push. Lint (gofmt, go mod tidy, go vet) on Ubuntu; build + test on Windows.
- **`release.yml`**: runs on every push to `main` and on `v*` tags.
  - Regenerates icon (Windows PowerShell 5.1, needed for System.Drawing)
  - Installs goversioninfo, patches versioninfo.json, runs `go generate`
  - Builds exe, builds installer (Inno Setup via Chocolatey)
  - On `main` push: updates rolling "latest" pre-release
  - On `v*` tag: creates proper versioned release

### `go generate`
`generate.go` contains `//go:generate goversioninfo -64=true -o resource.syso`.
`goversioninfo` reads `versioninfo.json` and `assets/icon.ico` to produce
`resource.syso`, which is automatically linked by `go build` to embed:
- Application icon (shown in Explorer, taskbar, Alt-Tab)
- PE version info (shown in Properties dialog)

---

## Known issues / TODOs

| Issue | File | Severity |
|---|---|---|
| `WindowHistory` grows unboundedly (closed windows never evicted) | history.go | Minor |
| `FootprintAlpha` config field is persisted but never read (overlay alpha hardcoded at 165/255) | config.go, overlay.go | Minor |
| `ShortcutSet` config field is persisted but never read | config.go | Minor |
| `RestoreSize` config field is read by settings UI but the undo-on-unsnap feature is not implemented | config.go | Minor |
| `DisplayName()` on `WindowAction` is defined but never called | actions.go | Minor (dead code) |
| `AnimateFootprint` config is persisted and shown in UI but the overlay animation is not conditionally disabled | config.go | Minor |
| Hotkeys are hardcoded in `registerAllHotKeys()`; the settings UI shows them and allows editing, but changes are not persisted back to RegisterHotKey calls | settings.go, main.go | Major |

---

## Adding a new snap action

1. Add a constant to `actions.go` `const` block (use the next free integer)
2. Add a `case` to `(a WindowAction).Calculate()` returning the target `w32.RECT`
3. Add to `IsDragSnappable()` exclusion list if it should not be drag-triggerable
4. Add an entry to `DisplayName()` map
5. Add to `allShortcutEntries()` in `settings.go` (with section name and default hotkey)
6. Add the diagram coordinates to the `ACT` map in `settingsHTML` (JS)
7. If zone-configurable, add the string key to `zoneNameToAction` in `snapzones.go`
   and to `ALL_ZONE_OPTS` in `settingsHTML`

---

## Adding a new settings field

1. Add field to `AppConfig` in `config.go`
2. Set default in `defaultConfig()`
3. Add backfill in `fillDefaults()`
4. If exposed in UI: add to `GeneralSettings` or `SnapAreasSettings` struct in `settings.go`,
   read it in `runSettingsWindow` JSON inject, write it in `goSaveSettings` handler
5. Add a key to both `LANGS.en` and `LANGS.es` in `settingsHTML`
