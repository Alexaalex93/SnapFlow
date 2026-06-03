package main

import "testing"

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
		EdgeThresholdPx:    30,
		CornerSnapAreaSize: 15,
		AlmostMaximizeWidth:  0.8,
		AlmostMaximizeHeight: 0.85,
		SizeOffset:           20,
		ShortcutSet:         "rectangle",
	}
	out := fillDefaults(cfg)
	if out.EdgeThresholdPx != 30 {
		t.Errorf("expected EdgeThresholdPx=30, got %d", out.EdgeThresholdPx)
	}
	if out.AlmostMaximizeWidth != 0.8 {
		t.Errorf("expected AlmostMaximizeWidth=0.8, got %f", out.AlmostMaximizeWidth)
	}
}
