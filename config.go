package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const configVersion = 2

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
	FootprintAlpha float64 `json:"footprintAlpha"`

	// ── Keyboard shortcut set ───────────────────────────────────────────────
	ShortcutSet string `json:"shortcutSet"`

	// ── Snap behavior toggles ────────────────────────────────────────────────
	SnapByDragging   bool `json:"snapByDragging"`   // default true
	RestoreSize      bool `json:"restoreSize"`      // restore on unsnap
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

func (s *ConfigStore) Load() (*AppConfig, error) {
	cfg := defaultConfig()
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return fillDefaults(cfg), nil
}

func (s *ConfigStore) Save(cfg *AppConfig) error {
	cfg.Version = configVersion
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(s.path, b, 0o644)
}

func (s *ConfigStore) Path() string { return s.path }

func defaultConfig() *AppConfig {
	return &AppConfig{
		Version:               configVersion,
		EdgeThresholdPx:       46,
		CornerSnapAreaSize:    20,
		ShortEdgeSnapAreaSize: 145,
		GapSize:               0,
		AlmostMaximizeWidth:   0.9,
		AlmostMaximizeHeight:  0.9,
		SizeOffset:            30,
		WidthStepSize:         30,
		FootprintAlpha:        0.3,
		ShortcutSet:           "rectangle",
		SnapByDragging:        true,
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
	if cfg.AlmostMaximizeWidth <= 0 {
		cfg.AlmostMaximizeWidth = d.AlmostMaximizeWidth
	}
	if cfg.AlmostMaximizeHeight <= 0 {
		cfg.AlmostMaximizeHeight = d.AlmostMaximizeHeight
	}
	if cfg.SizeOffset <= 0 {
		cfg.SizeOffset = d.SizeOffset
	}
	if cfg.WidthStepSize <= 0 {
		cfg.WidthStepSize = d.WidthStepSize
	}
	if cfg.FootprintAlpha <= 0 {
		cfg.FootprintAlpha = d.FootprintAlpha
	}
	if cfg.ShortcutSet == "" {
		cfg.ShortcutSet = d.ShortcutSet
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
