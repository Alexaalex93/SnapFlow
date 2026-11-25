package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// configVersion history:
//
//	2 — per-zone actions
//	3 — custom shortcuts, cycleHalves, language; restoreSize now implemented
//	    (migrated to on, matching Rectangle's default)
const configVersion = 3

// ShortcutConfig is one user-defined hotkey. Mod uses MOD_* flags, VK is a
// Windows virtual-key code. {0,0} means "no shortcut" (disabled).
type ShortcutConfig struct {
	Mod int `json:"mod"`
	VK  int `json:"vk"`
}

// AppConfig holds all user-configurable settings.
// Field names and defaults mirror Rectangle's Defaults.swift.
type AppConfig struct {
	Version int `json:"version"`

	// ── Snap behavior ───────────────────────────────────────────────────────
	// EdgeThresholdPx: pixels from screen edge to activate snap zone.
	EdgeThresholdPx int `json:"edgeThresholdPx"`
	// CornerSnapAreaSize: square corner hot-zone size in pixels (default 20).
	CornerSnapAreaSize int `json:"cornerSnapAreaSize"`
	// ShortEdgeSnapAreaSize: height of the upper/lower thirds on left/right edges (default 145).
	ShortEdgeSnapAreaSize int `json:"shortEdgeSnapAreaSize"`

	// ── Gap between windows ─────────────────────────────────────────────────
	GapSize int `json:"gapSize"` // pixels between snapped windows (default 0)

	// ── Almost Maximize ─────────────────────────────────────────────────────
	AlmostMaximizeWidth  float64 `json:"almostMaximizeWidth"`  // 0.0-1.0, default 0.9
	AlmostMaximizeHeight float64 `json:"almostMaximizeHeight"` // 0.0-1.0, default 0.9

	// ── Resize step ─────────────────────────────────────────────────────────
	SizeOffset    int `json:"sizeOffset"`    // pixels for larger/smaller (default 30)
	WidthStepSize int `json:"widthStepSize"` // pixels for largerWidth/smallerWidth (default 30)

	// ── Footprint appearance ─────────────────────────────────────────────────
	FootprintAlpha float64 `json:"footprintAlpha"` // fill opacity of the snap preview (0.1-0.9)

	// ── Keyboard shortcut set ───────────────────────────────────────────────
	ShortcutSet string `json:"shortcutSet"`
	// Shortcuts overrides default hotkeys, keyed by WindowAction number.
	// Missing key = default shortcut; {0,0} = disabled.
	Shortcuts map[string]ShortcutConfig `json:"shortcuts,omitempty"`
	// CycleHalves: pressing Left/Right Half again cycles ½ → ⅔ → ⅓.
	CycleHalves bool `json:"cycleHalves"`

	// ── Snap behavior toggles ────────────────────────────────────────────────
	SnapByDragging   bool `json:"snapByDragging"`   // default true
	RestoreSize      bool `json:"restoreSize"`      // restore original size when dragging a snapped window out
	AnimateFootprint bool `json:"animateFootprint"` // default true

	// ── Per-zone snap action ─────────────────────────────────────────────────
	// Each field is a WindowAction name (e.g. "leftHalf", "maximize", "none").
	ZoneTopLeft     string `json:"zoneTopLeft"`
	ZoneTop         string `json:"zoneTop"`
	ZoneTopRight    string `json:"zoneTopRight"`
	ZoneLeft        string `json:"zoneLeft"`
	ZoneRight       string `json:"zoneRight"`
	ZoneBottomLeft  string `json:"zoneBottomLeft"`
	ZoneBottom      string `json:"zoneBottom"`
	ZoneBottomRight string `json:"zoneBottomRight"`

	// ── UI ───────────────────────────────────────────────────────────────────
	Language string `json:"language"` // "en", "es" or "" (follow Windows)
	// NotifiedConflicts remembers which hotkey conflicts the user was
	// already told about, so the warning isn't repeated on every start.
	NotifiedConflicts string `json:"notifiedConflicts,omitempty"`
}

type ConfigStore struct {
	path string
}

func NewConfigStore() (*ConfigStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("user config dir: %w", err)
	}
	appDir := filepath.Join(dir, appName)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	return &ConfigStore{path: filepath.Join(appDir, "config.json")}, nil
}

// errConfigReset is returned (together with a usable default config) when
// the file existed but could not be parsed. The broken file is kept as
// config.json.bad so the user's edits aren't lost.
var errConfigReset = errors.New("config file was invalid; defaults restored (old file saved as config.json.bad)")

func (s *ConfigStore) Load() (*AppConfig, error) {
	cfg := defaultConfig()
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(b, cfg); err != nil {
		_ = os.WriteFile(s.path+".bad", b, 0o644)
		return defaultConfig(), errConfigReset
	}
	return migrate(fillDefaults(cfg)), nil
}

// Save writes atomically (temp file + rename) so a crash or power loss
// mid-write can never leave a truncated config behind.
func (s *ConfigStore) Save(cfg *AppConfig) error {
	cfg.Version = configVersion
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *ConfigStore) Path() string { return s.path }

func defaultConfig() *AppConfig {
	return &AppConfig{
		Version:               configVersion,
		EdgeThresholdPx:       12,
		CornerSnapAreaSize:    20,
		ShortEdgeSnapAreaSize: 145,
		GapSize:               0,
		AlmostMaximizeWidth:   0.9,
		AlmostMaximizeHeight:  0.9,
		SizeOffset:            30,
		WidthStepSize:         30,
		FootprintAlpha:        0.3,
		ShortcutSet:           "rectangle",
		CycleHalves:           true,
		SnapByDragging:        true,
		RestoreSize:           true,
		AnimateFootprint:      true,
		ZoneTopLeft:           "topLeft",
		ZoneTop:               "maximize",
		ZoneTopRight:          "topRight",
		ZoneLeft:              "leftHalf",
		ZoneRight:             "rightHalf",
		ZoneBottomLeft:        "bottomLeft",
		ZoneBottom:            "compoundThirds",
		ZoneBottomRight:       "bottomRight",
	}
}

func migrate(cfg *AppConfig) *AppConfig {
	if cfg.Version < 3 {
		// restoreSize was shown in the UI but never implemented before v3,
		// so the stored value was meaningless: switch to the real default.
		cfg.RestoreSize = true
		cfg.CycleHalves = true
	}
	cfg.Version = configVersion
	return cfg
}

func fillDefaults(cfg *AppConfig) *AppConfig {
	d := defaultConfig()
	if cfg.EdgeThresholdPx <= 0 {
		cfg.EdgeThresholdPx = d.EdgeThresholdPx
	}
	if cfg.CornerSnapAreaSize <= 0 {
		cfg.CornerSnapAreaSize = d.CornerSnapAreaSize
	}
	if cfg.ShortEdgeSnapAreaSize <= 0 {
		cfg.ShortEdgeSnapAreaSize = d.ShortEdgeSnapAreaSize
	}
	if cfg.GapSize < 0 {
		cfg.GapSize = 0
	}
	if cfg.AlmostMaximizeWidth <= 0 || cfg.AlmostMaximizeWidth > 1 {
		cfg.AlmostMaximizeWidth = d.AlmostMaximizeWidth
	}
	if cfg.AlmostMaximizeHeight <= 0 || cfg.AlmostMaximizeHeight > 1 {
		cfg.AlmostMaximizeHeight = d.AlmostMaximizeHeight
	}
	if cfg.SizeOffset <= 0 {
		cfg.SizeOffset = d.SizeOffset
	}
	if cfg.WidthStepSize <= 0 {
		cfg.WidthStepSize = d.WidthStepSize
	}
	if cfg.FootprintAlpha <= 0 || cfg.FootprintAlpha > 0.95 {
		cfg.FootprintAlpha = d.FootprintAlpha
	}
	if cfg.ShortcutSet == "" {
		cfg.ShortcutSet = d.ShortcutSet
	}
	if cfg.Language != "en" && cfg.Language != "es" {
		cfg.Language = ""
	}
	zones := map[*string]string{
		&cfg.ZoneTopLeft: d.ZoneTopLeft, &cfg.ZoneTop: d.ZoneTop,
		&cfg.ZoneTopRight: d.ZoneTopRight, &cfg.ZoneLeft: d.ZoneLeft,
		&cfg.ZoneRight: d.ZoneRight, &cfg.ZoneBottomLeft: d.ZoneBottomLeft,
		&cfg.ZoneBottom: d.ZoneBottom, &cfg.ZoneBottomRight: d.ZoneBottomRight,
	}
	for z, def := range zones {
		if *z == "" {
			*z = def
		}
	}
	return cfg
}

// clone returns a deep copy (the Shortcuts map is not shared).
func (c *AppConfig) clone() *AppConfig {
	n := *c
	if c.Shortcuts != nil {
		n.Shortcuts = make(map[string]ShortcutConfig, len(c.Shortcuts))
		for k, v := range c.Shortcuts {
			n.Shortcuts[k] = v
		}
	}
	return &n
}
