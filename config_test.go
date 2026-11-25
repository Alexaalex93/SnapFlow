package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gonutz/w32/v2"
)

func TestFillDefaults_SetsAll(t *testing.T) {
	cfg := fillDefaults(&AppConfig{Version: 1})
	if cfg.EdgeThresholdPx <= 0 {
		t.Fatal("EdgeThresholdPx should be defaulted")
	}
	if cfg.AlmostMaximizeWidth <= 0 {
		t.Fatal("AlmostMaximizeWidth should be defaulted")
	}
	if cfg.ShortcutSet == "" {
		t.Fatal("ShortcutSet should be defaulted")
	}
}

func TestFillDefaults_PreservesValidValues(t *testing.T) {
	cfg := &AppConfig{
		EdgeThresholdPx:      30,
		CornerSnapAreaSize:   15,
		AlmostMaximizeWidth:  0.8,
		AlmostMaximizeHeight: 0.85,
		SizeOffset:           20,
		ShortcutSet:          "rectangle",
	}
	out := fillDefaults(cfg)
	if out.EdgeThresholdPx != 30 {
		t.Errorf("expected EdgeThresholdPx=30, got %d", out.EdgeThresholdPx)
	}
	if out.AlmostMaximizeWidth != 0.8 {
		t.Errorf("expected AlmostMaximizeWidth=0.8, got %f", out.AlmostMaximizeWidth)
	}
}

func TestLoad_InvalidJSONFallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	s := &ConfigStore{path: filepath.Join(dir, "config.json")}
	if err := os.WriteFile(s.path, []byte("{ this is not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := s.Load()
	if err != errConfigReset {
		t.Fatalf("expected errConfigReset, got %v", err)
	}
	if cfg == nil || cfg.EdgeThresholdPx != defaultConfig().EdgeThresholdPx {
		t.Fatal("expected usable default config")
	}
	if _, err := os.Stat(s.path + ".bad"); err != nil {
		t.Fatal("broken file should be preserved as config.json.bad")
	}
}

func TestLoad_MigratesV2(t *testing.T) {
	dir := t.TempDir()
	s := &ConfigStore{path: filepath.Join(dir, "config.json")}
	os.WriteFile(s.path, []byte(`{"version":2,"restoreSize":false,"gapSize":6}`), 0o644)
	cfg, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.RestoreSize || !cfg.CycleHalves || cfg.GapSize != 6 || cfg.Version != configVersion {
		t.Fatalf("bad migration: %+v", cfg)
	}
}

func TestSaveLoadRoundTrip_Shortcuts(t *testing.T) {
	dir := t.TempDir()
	s := &ConfigStore{path: filepath.Join(dir, "config.json")}
	cfg := defaultConfig()
	cfg.Shortcuts = map[string]ShortcutConfig{"0": {Mod: MOD_WIN | MOD_ALT, VK: 0x25}}
	if err := s.Save(cfg); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range effectiveShortcuts(got) {
		if e.action == ActionLeftHalf && (e.mod != MOD_WIN|MOD_ALT || e.vk != 0x25) {
			t.Fatalf("override not applied: %+v", e)
		}
	}
}

func TestShortcutOverrides_OnlyStoresChanges(t *testing.T) {
	var list []ShortcutBinding
	for _, e := range allShortcutEntries() {
		list = append(list, ShortcutBinding{Action: int(e.action), Mod: e.mod, VK: e.vk})
	}
	if m := shortcutOverrides(list); m != nil {
		t.Fatalf("unchanged list should produce no overrides, got %v", m)
	}
	list[0].VK = 0 // disable Left Half
	m := shortcutOverrides(list)
	if len(m) != 1 || m["0"] != (ShortcutConfig{}) {
		t.Fatalf("expected a single 'disabled' override, got %v", m)
	}
}

func TestHistory_KeepsOriginalRestoreRect(t *testing.T) {
	h := &WindowHistory{entries: map[w32.HWND]*historyEntry{}}
	free := w32.RECT{Left: 100, Top: 100, Right: 900, Bottom: 700}
	left := w32.RECT{Left: 0, Top: 0, Right: 960, Bottom: 1040}
	topLeft := w32.RECT{Left: 0, Top: 0, Right: 960, Bottom: 520}

	h.RecordSnap(1, free, left)
	h.RecordSnap(1, left, topLeft) // snapped again from the snapped spot
	if r, ok := h.Snapped(1, topLeft); !ok || r != free {
		t.Fatalf("restore rect should still be the original free rect, got %+v %v", r, ok)
	}
	if _, ok := h.Snapped(1, free); ok {
		t.Fatal("window at a different spot must not count as snapped")
	}
}

func TestHistory_Cycle(t *testing.T) {
	h := &WindowHistory{entries: map[w32.HWND]*historyEntry{}}
	a := w32.RECT{Left: 0, Top: 0, Right: 960, Bottom: 1040}
	b := w32.RECT{Left: 0, Top: 0, Right: 1280, Bottom: 1040}
	if i := h.NextCycle(1, ActionLeftHalf, a, 3); i != 0 {
		t.Fatalf("first press should be step 0, got %d", i)
	}
	h.RecordSnap(1, w32.RECT{Left: 50, Top: 50, Right: 500, Bottom: 500}, a)
	h.SetCycle(1, ActionLeftHalf, 0)
	if i := h.NextCycle(1, ActionLeftHalf, a, 3); i != 1 {
		t.Fatalf("second press should be step 1, got %d", i)
	}
	h.RecordSnap(1, a, b)
	h.SetCycle(1, ActionLeftHalf, 1)
	if i := h.NextCycle(1, ActionRightHalf, b, 3); i != 0 {
		t.Fatalf("a different shortcut restarts its cycle, got %d", i)
	}
}

func TestIsNewerVersion(t *testing.T) {
	if !isNewerVersion("1.0.120", "1.0.99") || isNewerVersion("1.0.99", "1.0.120") {
		t.Fatal("numeric comparison broken")
	}
	if isNewerVersion("9.9.9", "dev") {
		t.Fatal("dev builds must not be offered updates")
	}
}
