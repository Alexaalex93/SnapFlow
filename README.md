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
| 🖱 **Drag-to-snap** | Drag any window to an edge or corner — a blue preview shows exactly where it will land |
| 🎯 **Compound thirds** | Drag along the bottom edge to sweep across thirds — release over 1, 2 or 3 to snap to that fraction |
| ⌨️ **Keyboard shortcuts** | Full shortcut set matching Rectangle's layout (Ctrl+Alt based) |
| 🖥 **Multi-monitor** | Move windows between displays with Ctrl+Alt+Win+Arrow |
| ⚙️ **Settings UI** | Visual settings window with live zone previews — available in English and Spanish |
| 🔕 **Silent** | Runs in the system tray, no console window, no UAC prompts |
| 🔄 **Autostart** | Optional launch at login, togglable from the tray menu |

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
- Click the shortcut chip to reassign it

### Snap Areas tab
- **Windows Snap** — disable AeroSnap so it doesn't conflict with SnapFlow's drag zones
- **Snap windows by dragging** — toggle drag-to-snap on/off
- **Zone grid** — 3×3 visual grid; each zone has a live diagram and a dropdown with every available action

### General tab
Edge threshold, corner size, gap between windows, Almost Maximize size, resize step.

### Language
Toggle **EN / ES** in the top-right corner of the Settings window.

---

## Tray menu

Right-click the SnapFlow icon in the system tray:

| Item | Description |
|---|---|
| Settings | Opens the Settings window |
| Start at login | Toggle autostart |
| Disable Windows Snap | Turns off AeroSnap via registry (recommended) |
| Open config file | Opens `config.json` in your default editor |
| Quit | Exits SnapFlow |

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
