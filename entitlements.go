package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type Feature string

const (
	FeatureBasicDragSnap       Feature = "basic_drag_snap"
	FeatureBasicKeyboard       Feature = "basic_keyboard"
	FeatureThirdsAndTwoThirds  Feature = "thirds_two_thirds"
	FeatureDynamicMorphing     Feature = "dynamic_morphing"
	FeatureAdvancedGestures    Feature = "advanced_gestures"
	FeatureAdvancedMultiMonitor Feature = "advanced_multi_monitor"
)

type Entitlements struct {
	pro bool
}

func NewEntitlements(pro bool) *Entitlements {
	return &Entitlements{pro: pro}
}

func (e *Entitlements) IsPro() bool { return e != nil && e.pro }

func (e *Entitlements) Enabled(f Feature) bool {
	if e == nil {
		return false
	}
	switch f {
	case FeatureBasicDragSnap, FeatureBasicKeyboard:
		return true
	case FeatureThirdsAndTwoThirds, FeatureDynamicMorphing, FeatureAdvancedGestures, FeatureAdvancedMultiMonitor:
		return e.pro
	default:
		return false
	}
}

type LicenseManager struct {
	config *AppConfig
}

func NewLicenseManager(cfg *AppConfig) *LicenseManager {
	return &LicenseManager{config: cfg}
}

func (m *LicenseManager) ProUnlocked() bool {
	if m == nil || m.config == nil {
		return false
	}
	return validateProLicense(m.config.ProLicenseKey)
}

func validateProLicense(key string) bool {
	parts := strings.Split(strings.TrimSpace(key), "-")
	if len(parts) != 3 {
		return false
	}
	if !strings.EqualFold(parts[0], "SFPRO") {
		return false
	}
	payload := strings.ToUpper(strings.TrimSpace(parts[1]))
	sig := strings.ToLower(strings.TrimSpace(parts[2]))
	if payload == "" || len(sig) < 8 {
		return false
	}
	mac := hmac.New(sha256.New, []byte("snapflow-local-license-v1"))
	mac.Write([]byte(payload))
	expected := hex.EncodeToString(mac.Sum(nil))[:8]
	return hmac.Equal([]byte(sig), []byte(expected))
}
