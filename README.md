# SnapFlow

SnapFlow is a commercial derivative of [RectangleWin](https://github.com/ahmetb/RectangleWin), focused on fluid mouse-driven window snapping on Windows.

It is designed to feel closer to Rectangle on macOS than static zone tools by prioritizing:
- drag-to-edge gestures
- live translucent preview
- dynamic target morphing while you keep dragging

SnapFlow is distributed under a freemium model (Free + one-time Pro), while preserving Apache 2.0 attribution and compliance for the upstream base.

## What SnapFlow Is

- A modern Windows snap utility with dynamic edge gestures
- A tray app with keyboard + drag snapping workflows
- A pragmatic, lightweight utility that runs without admin privileges

## What SnapFlow Is Not

- Not a full tiling window manager
- Not a static grid editor
- Not a FancyZones clone

## Features

### Free Tier

- Left/right half snapping
- Top/bottom half snapping
- Basic corner snapping (quadrants)
- Basic drag-to-edge preview and snap
- Core keyboard shortcuts for basic layouts

### Pro Tier (one-time upgrade)

- Thirds and two-thirds layouts
- Dynamic edge morphing (for example one-third <-> two-thirds while dragging on an edge)
- Advanced gesture behavior for richer edge interactions
- Extensible entitlement architecture for future premium features

## Keyboard Bindings

- **Snap to edges**: <kbd>Win</kbd> + <kbd>Alt</kbd> + <kbd>&larr;</kbd><kbd>&rarr;</kbd><kbd>&uarr;</kbd><kbd>&darr;</kbd>
- **Corner snapping**: <kbd>Win</kbd> + <kbd>Ctrl</kbd> + <kbd>Alt</kbd> + <kbd>&larr;</kbd><kbd>&uarr;</kbd><kbd>&darr;</kbd><kbd>&rarr;</kbd>
- **Center window**: <kbd>Win</kbd> + <kbd>Alt</kbd> + <kbd>C</kbd>
- **Maximize**: <kbd>Win</kbd> + <kbd>Shift</kbd> + <kbd>F</kbd>
- **Always On Top toggle**: <kbd>Win</kbd> + <kbd>Alt</kbd> + <kbd>A</kbd>

Note: In Free tier, advanced thirds/two-thirds keyboard cycling is entitlement-gated.

## Drag UX

- Start dragging a top-level window
- Move to an edge or corner to trigger a snap preview
- Keep dragging near the same edge to morph target layouts
- Release mouse to commit
- Leave snap regions to cancel preview

## Configuration and Licensing

SnapFlow stores versioned configuration at:
- `%AppData%\\SnapFlow\\config.json`

Current schema includes:
- `version`
- `pro_license_key`
- `upgrade_entry_seen`

License validation is local-only in this version (no activation server required yet). The architecture is intentionally ready for future activation hardening.

## Known Limitations / Rectangle Parity Notes

Exact parity with Rectangle on macOS is not always possible on Windows due to platform-level differences:
- Windows move/size notifications are not identical to macOS drag event semantics.
- Some app frameworks (custom-drawn/borderless windows) may emit move/size events differently.
- Invisible frame and DPI behavior vary by framework and OS version.
- Drag behavior for maximized/restore transitions depends on native Windows window-manager handling.

SnapFlow targets the closest reliable Rectangle-like interaction Windows APIs allow without introducing brittle hooks or a heavyweight compositor model.

## Packaging and Microsoft Store Readiness

Current code is aligned for future MSIX packaging:
- user-level app behavior (no admin required)
- no privileged APIs required for core snapping
- lightweight dependency footprint

Potential Store submission follow-ups:
- final app identity assets and publisher metadata
- packaging/signing pipeline and validation checks
- final privacy/support URLs for listing completeness

## Attribution and License Compliance

SnapFlow is based on RectangleWin by Ahmet Alp Balkan.

This codebase retains and honors Apache License 2.0 requirements. See [LICENSE](./LICENSE).

## Development

With Go installed, build on Windows:

```sh
go generate
GOOS=windows go build -ldflags -H=windowsgui .
```
