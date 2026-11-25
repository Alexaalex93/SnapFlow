# SnapFlow

**SnapFlow** is a free, open-source window manager for Windows inspired by [Rectangle](https://rectangleapp.com) on macOS.
Drag any window to a screen edge, watch the blue preview snap into position, and release.

> Requires Windows 10 or 11 (x64). No admin privileges needed.

---

## Download

**→ [Latest release](https://github.com/Alexaalex93/SnapFlow/releases/latest)**

Download `SnapFlow-x.x.x-Setup.exe` and run it. The installer:
- Installs to `%LOCALAPPDATA%\SnapFlow` (no UAC prompt)
- Optionally starts SnapFlow automatically at login
- Adds an uninstaller to Add/Remove Programs

**Portable**: download `snapflow.exe` and run it directly — no installation needed.

> **Requires [WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)** for the Settings window.
> Already installed on Windows 11. On Windows 10, install it from the link or let Windows Update handle it.

---

## Features

| | |
|---|---|
| 🖱 **Drag-to-snap** | Drag any window to an edge or corner — an animated preview in your Windows accent colour glides between zones and shows exactly where it will land |
| 🎯 **Compound thirds** | Drag along the bottom edge to sweep across thirds — release over 1, 2 or 3 to snap to that fraction (sweep back to shrink) |
| ↩️ **Restore on unsnap** | Drag a snapped window away and it gets its original size back |
| ⌨️ **Keyboard shortcuts** | Rectangle's layout (Ctrl+Alt based), fully customisable; press Left/Right Half again to cycle ½ → ⅔ → ⅓ |
| 🎞 **Smooth motion** | Windows glide and grow into place, their content adapting as they resize — no jump at the end |
| ⤵️ **Drag maximised windows** | Grab a maximised window's title bar and drag — it restores under the cursor, even with Windows Snap turned off |
| ◻️ **Square corners** | Snapped windows get square corners like Windows 11's own Snap; they round again when you drag them free |
| 📐 **Pixel-exact** | Edge-to-edge placement on any DPI / multi-monitor setup, including maximised and minimum-size windows |
| 🖥 **Multi-monitor** | Move windows between displays with Ctrl+Alt+Win+Arrow (halves stay halves, maximised stays maximised) |
| ⚙️ **Settings UI** | Visual settings window with live zone previews — English and Spanish |
| 🔕 **Silent** | Runs in the system tray, no admin rights, no pop-up dialogs — just quiet notifications |
| 🔄 **Autostart** | Optional launch at login, from the tray menu or Settings |

---

## Snap positions

SnapFlow supports every position Rectangle has, plus more.

### Drag zones — where to drag a window

| Zone | Default action |
|---|---|
| Top edge (centre) | Maximize |
| Top-left corner | Top Left quarter |
| Top-right corner | Top Right quarter |
| Left edge (middle) | Left Half |
| Left edge (top band) | Top Left quarter |
| Left edge (bottom band) | Bottom Left quarter |
| Right edge (middle) | Right Half |
| Right edge (top band) | Top Right quarter |
| Right edge (bottom band) | Bottom Right quarter |
| Bottom edge | **Compound thirds** (see below) |
| Bottom-left corner | Bottom Left quarter |
| Bottom-right corner | Bottom Right quarter |

**Compound thirds on the bottom edge** — drag to the bottom of the screen and sweep horizontally:
- Land in the left third → **Left 1/3**
- Sweep left + centre → **Left 2/3**
- Land in centre only → **Centre 1/3**
- Sweep centre + right → **Right 2/3**
- Land in right third → **Right 1/3**
- Sweep all three → **Maximize**

All drag zones are configurable in **Settings → Snap Areas**.

---

### Keyboard shortcuts

| Action | Shortcut |
|---|---|
| **Left Half** | `Ctrl+Alt+←` |
| **Right Half** | `Ctrl+Alt+→` |
| **Top Half** | `Ctrl+Alt+↑` |
| **Bottom Half** | `Ctrl+Alt+↓` |
| **Top Left** | `Ctrl+Alt+U` |
| **Top Right** | `Ctrl+Alt+I` |
| **Bottom Left** | `Ctrl+Alt+J` |
| **Bottom Right** | `Ctrl+Alt+K` |
| **First Third** | `Ctrl+Alt+D` |
| **Center Third** | `Ctrl+Alt+F` |
| **Last Third** | `Ctrl+Alt+G` |
| **First Two Thirds** | `Ctrl+Alt+E` |
| **Center Two Thirds** | `Ctrl+Alt+R` |
| **Last Two Thirds** | `Ctrl+Alt+T` |
| **Maximize** | `Ctrl+Alt+Enter` |
| **Almost Maximize** | `Ctrl+Alt+Shift+Enter` |
| **Maximize Height** | `Ctrl+Alt+Shift+↑` |
| **Center** | `Ctrl+Alt+C` |
| **Larger** | `Ctrl+Alt+=` |
| **Smaller** | `Ctrl+Alt+-` |
| **Restore** | `Ctrl+Alt+Backspace` |
| **Next Display** | `Ctrl+Alt+Win+→` |
| **Previous Display** | `Ctrl+Alt+Win+←` |

Shortcuts for Fourths, Sixths, Eighths, Ninths, Twelfths, Sixteenths and more can be assigned in **Settings → Shortcuts**.

---

## Settings

Open from the tray icon → **Settings**.

### Shortcuts tab
Every snap position — halves, quarters, thirds, fourths, sixths, eighths, ninths, twelfths, sixteenths — is listed with:
- A small diagram showing exactly what the position looks like
- The current keyboard shortcut
- Click the chip and press the new combination (**Backspace** clears it, **Esc** cancels, ↺ restores the default).
  Assigning a combination that another action uses moves it. A search box filters the list.

### Snap Areas tab
- **Windows Snap** — turn AeroSnap off so it doesn't conflict with SnapFlow (applies instantly)
- **Snap windows by dragging** — toggle drag-to-snap on/off (also in the tray menu)
- **Restore window size when unsnapped** — dragging a snapped window out restores its size
- **Animate window movement** — windows glide into place (follows Windows' *Animation effects* setting)
- **Square corners on snapped windows** — Windows 11 style (on by default)
- **Zone grid** — 3×3 visual grid; each zone has a live diagram and a dropdown with every action (or *None*)

### General tab
Edge threshold, corner size, gap between windows, Almost Maximize size, resize step,
½ → ⅔ → ⅓ cycling, preview opacity, launch at login.

### Language
Toggle **EN / ES** in the top-right corner of the Settings window (remembered; defaults to the Windows language).
Press **Ctrl+S** to save.

---

## Tray menu

Click the SnapFlow tray icon to open Settings; right-click for the menu:

| Item | Description |
|---|---|
| Settings | Opens the Settings window (or brings it to the front) |
| Snap windows by dragging | Pause / resume drag snapping |
| Launch at login | Toggle autostart |
| Windows Snap (conflicts with SnapFlow) | Ticked = Windows' own Snap is on. Untick it (recommended) — applies immediately |
| Open config file | Opens `config.json` in your default editor |
| Check for updates | Notifies you if a newer release exists |
| Quit | Exits SnapFlow |

If you hand-edit `config.json` and break it, SnapFlow starts with defaults and keeps your file as `config.json.bad`.
Launching SnapFlow again while it is running just opens Settings.

---

## Building from source

Requirements: **Go 1.21+**, **Windows** (cross-compilation not supported — uses Win32 APIs).

```powershell
# Quick build (no icon embedding)
go build -ldflags "-H=windowsgui" -o snapflow.exe .

# Full release build with embedded icon and installer
.\scripts\build-release.ps1 -Version 1.0.0
```

The full build requires [goversioninfo](https://github.com/josephspurrier/goversioninfo) and [Inno Setup 6](https://jrsoftware.org/isdl.php).
`build-release.ps1` installs both automatically if missing.

---

## Uninstall

**If installed via the installer**: Control Panel → Add/Remove Programs → SnapFlow → Uninstall.

**If running portable**: close SnapFlow from the tray, delete `snapflow.exe`.

Config is stored at `%AppData%\SnapFlow\config.json` — delete it to reset all settings.

---

## License

MIT — see [LICENSE](LICENSE).
