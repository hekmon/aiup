// Package msiaf provides tooling for working with MSI Afterburner profiles and configuration.
package msiaf

import (
	"testing"
)

// TestSettings_GetFanControlCurve tests that fan curve accessors work after config parsing.
// This is an integration test verifying that ParseGlobalConfig populates curve data.
func TestSettings_GetFanControlCurve(t *testing.T) {
	hexStr := "0000010004000000000000000000F0410000204200004842000048420000A0420000A0420000B4420000C8420000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	settings := &Settings{}
	parseSettingsField(settings, "SwAutoFanControlCurve", hexStr)

	curve := settings.GetFanControlCurve()
	if curve == nil {
		t.Fatal("GetFanControlCurve returned nil")
	}

	err := settings.GetFanControlCurveError()
	if err != nil {
		t.Errorf("GetFanControlCurveError returned error: %v", err)
	}

	if len(curve.Points) != 4 {
		t.Errorf("Expected 4 points, got %d", len(curve.Points))
	}
}

// TestParseGlobalConfig_FanCurves tests fan curve parsing from actual config file.
// This integration test verifies the complete config parsing pipeline.
func TestParseGlobalConfig_FanCurves(t *testing.T) {
	config, err := ParseGlobalConfig("../../LocalProfiles/MSIAfterburner.cfg")
	if err != nil {
		t.Fatalf("Failed to parse config: %v", err)
	}

	curve := config.Settings.GetFanControlCurve()
	if curve == nil {
		t.Fatal("GetFanControlCurve returned nil")
	}

	// Check for parse errors
	if err := config.Settings.GetFanControlCurveError(); err != nil {
		t.Errorf("GetFanControlCurveError returned error: %v", err)
	}

	if len(curve.Points) != 4 {
		t.Errorf("Expected 4 points, got %d", len(curve.Points))
	}

	// Verify first and last points
	if curve.Points[0].Temperature != 30.0 {
		t.Errorf("Expected first point temp 30.0, got %.2f", curve.Points[0].Temperature)
	}
	if curve.Points[len(curve.Points)-1].FanSpeed != 100.0 {
		t.Errorf("Expected last point fan speed 100.0, got %.2f", curve.Points[len(curve.Points)-1].FanSpeed)
	}

	// Curve2 should be identical in the test file
	curve2 := config.Settings.GetFanControlCurve2()
	if curve2 == nil {
		t.Fatal("GetFanControlCurve2 returned nil")
	}

	if err := config.Settings.GetFanControlCurve2Error(); err != nil {
		t.Errorf("GetFanControlCurve2Error returned error: %v", err)
	}

	if len(curve2.Points) != len(curve.Points) {
		t.Errorf("Curve2 points mismatch: expected %d, got %d", len(curve.Points), len(curve2.Points))
	}
}
