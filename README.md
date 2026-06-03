# SnapFlow

SnapFlow is a free, open-source window manager for Windows inspired by [Rectangle](https://rectangleapp.com) on macOS. It provides the same fluid window snapping experience — drag-to-edge gestures, live animated previews, and keyboard shortcuts — translated natively for Windows.

## Features

- **Drag-to-snap** — drag any window to a screen edge or corner and release to snap it
- **Animated preview** — translucent blue overlay shows exactly where the window will land, with smooth fade-in/out
- **Dynamic morphing** — keep dragging along an edge to cycle between half, one-third, and two-thirds layouts
- **Keyboard shortcuts** — full set of shortcuts mirroring Rectangle's layout (Ctrl+Alt based)
- **Multi-monitor** — move windows between displays with Ctrl+Alt+Win+Arrow
- **System tray** — runs quietly in the background with autostart support
- **No admin required** — lightweight tray app, no UAC prompts

## Keyboard Shortcuts

| Action | Shortcut |
|--------|----------|
| Left Half | `Ctrl+Alt+←` |
| Right Half | `Ctrl+Alt+→` |
| Top Half | `Ctrl+Alt+↑` |
| Bottom Half | `Ctrl+Alt+↓` |
| Top Left | `Ctrl+Alt+U` |
| Top Right | `Ctrl+Alt+I` |
| Bottom Left | `Ctrl+Alt+J` |
| Bottom Right | `Ctrl+Alt+K` |
| First Third | `Ctrl+Alt+D` |
| Center Third | `Ctrl+Alt+F` |
| Last Third | `Ctrl+Alt+G` |
| First Two Thirds | `Ctrl+Alt+E` |
| Last Two Thirds | `Ctrl+Alt+T` |
| Maximize | `Ctrl+Alt+Enter` |
| Almost Maximize | `Ctrl+Alt+Shift+Enter` |
| Maximize Height | `Ctrl+Alt+Shift+↑` |
| Center | `Ctrl+Alt+C` |
| Always on Top | `Ctrl+Alt+A` |
| Next Display | `Ctrl+Alt+Win+→` |
| Previous Display | `Ctrl+Alt+Win+←` |

## Drag Zones

| Where you drag | Result |
|----------------|--------|
| Left / Right edge | Half screen (dynamic: one-third or two-thirds based on position) |
| Top edge | Top half or top thirds |
| Bottom edge | Bottom half |
| Any corner | Quarter of the screen |

## Building

Requires Go 1.21+ and Windows.

```
go build -o snapflow.exe .
```

## License

MIT
