package main

import "testing"

func TestValidateProLicense(t *testing.T) {
	if !validateProLicense("SFPRO-ALICE-1ade6c38") {
		t.Fatalf("expected known-good key to pass validation")
	}
	if validateProLicense("bad") {
		t.Fatalf("invalid key should be rejected")
	}
}

func TestEntitlements(t *testing.T) {
	free := NewEntitlements(false)
	pro := NewEntitlements(true)
	if !free.Enabled(FeatureBasicDragSnap) {
		t.Fatalf("free should allow basic drag")
	}
	if free.Enabled(FeatureDynamicMorphing) {
		t.Fatalf("free should not allow dynamic morphing")
	}
	if !pro.Enabled(FeatureDynamicMorphing) {
		t.Fatalf("pro should allow dynamic morphing")
	}
}
