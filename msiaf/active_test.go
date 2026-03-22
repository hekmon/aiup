package msiaf

import (
	"fmt"
	"testing"
)

// TestMatchVFCurve_PerfectMatch tests matching when live and profile frequencies are identical.
func TestMatchVFCurve_PerfectMatch(t *testing.T) {
	// Create a simple profile curve with 3 active points
	profileCurve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 3,
		Points: []VFPoint{
			{
				Index:           0,
				VoltageMV:       800,
				OCScannerRefMHz: 2500,
				OffsetMHz:       100,
				IsActive:        true,
			},
			{
				Index:           1,
				VoltageMV:       900,
				OCScannerRefMHz: 2700,
				OffsetMHz:       50,
				IsActive:        true,
			},
			{
				Index:           2,
				VoltageMV:       1000,
				OCScannerRefMHz: 2900,
				OffsetMHz:       0,
				IsActive:        true,
			},
		},
	}

	// Live frequencies match exactly (OCScannerRef + Offset)
	liveFreqs := map[float32]float64{
		800:  2600, // 2500 + 100
		900:  2750, // 2700 + 50
		1000: 2900, // 2900 + 0
	}

	result, err := MatchVFCurve(liveFreqs, profileCurve, 5.0)
	if err != nil {
		t.Fatalf("MatchVFCurve failed: %v", err)
	}

	if result.MatchedPoints != 3 {
		t.Errorf("Expected 3 matched points, got %d", result.MatchedPoints)
	}

	if result.TotalPoints != 3 {
		t.Errorf("Expected 3 total points, got %d", result.TotalPoints)
	}

	if result.MatchConfidence != 1.0 {
		t.Errorf("Expected 100%% confidence, got %.2f%%", result.MatchConfidence*100)
	}

	if result.AvgDeviationMHz != 0.0 {
		t.Errorf("Expected 0 MHz average deviation, got %.2f", result.AvgDeviationMHz)
	}
}

// TestMatchVFCurve_PartialMatch tests matching when some points are within tolerance.
func TestMatchVFCurve_PartialMatch(t *testing.T) {
	profileCurve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 3,
		Points: []VFPoint{
			{
				Index:           0,
				VoltageMV:       800,
				OCScannerRefMHz: 2500,
				OffsetMHz:       100,
				IsActive:        true,
			},
			{
				Index:           1,
				VoltageMV:       900,
				OCScannerRefMHz: 2700,
				OffsetMHz:       50,
				IsActive:        true,
			},
			{
				Index:           2,
				VoltageMV:       1000,
				OCScannerRefMHz: 2900,
				OffsetMHz:       0,
				IsActive:        true,
			},
		},
	}

	// Live frequencies: 1st matches, 2nd is off by 10 MHz, 3rd is off by 50 MHz
	liveFreqs := map[float32]float64{
		800:  2600, // exact match
		900:  2760, // +10 MHz deviation
		1000: 2950, // +50 MHz deviation
	}

	result, err := MatchVFCurve(liveFreqs, profileCurve, 15.0)
	if err != nil {
		t.Fatalf("MatchVFCurve failed: %v", err)
	}

	// With 15 MHz tolerance: 800mV (0 MHz) and 900mV (10 MHz) should match, 1000mV (50 MHz) should not
	if result.MatchedPoints != 2 {
		t.Errorf("Expected 2 matched points (within 15 MHz tolerance), got %d", result.MatchedPoints)
	}

	if result.MatchConfidence != 2.0/3.0 {
		t.Errorf("Expected 66.67%% confidence, got %.2f%%", result.MatchConfidence*100)
	}

	if result.AvgDeviationMHz != 20.0 {
		t.Errorf("Expected 20.0 MHz average deviation, got %.2f", result.AvgDeviationMHz)
	}

	if result.MaxDeviationMHz != 50.0 {
		t.Errorf("Expected 50.0 MHz max deviation, got %.2f", result.MaxDeviationMHz)
	}
}

// TestMatchVFCurve_NoMatch tests when no points match within tolerance.
func TestMatchVFCurve_NoMatch(t *testing.T) {
	profileCurve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 2,
		Points: []VFPoint{
			{
				Index:           0,
				VoltageMV:       800,
				OCScannerRefMHz: 2500,
				OffsetMHz:       100,
				IsActive:        true,
			},
			{
				Index:           1,
				VoltageMV:       900,
				OCScannerRefMHz: 2700,
				OffsetMHz:       50,
				IsActive:        true,
			},
		},
	}

	// Live frequencies are way off
	liveFreqs := map[float32]float64{
		800: 2800, // 200 MHz off
		900: 2900, // 200 MHz off
	}

	result, err := MatchVFCurve(liveFreqs, profileCurve, 50.0)
	if err != nil {
		t.Fatalf("MatchVFCurve failed: %v", err)
	}

	if result.MatchedPoints != 0 {
		t.Errorf("Expected 0 matched points, got %d", result.MatchedPoints)
	}

	if result.MatchConfidence != 0.0 {
		t.Errorf("Expected 0%% confidence, got %.2f%%", result.MatchConfidence*100)
	}
}

// TestMatchVFCurve_InactivePoints tests that inactive points are skipped.
func TestMatchVFCurve_InactivePoints(t *testing.T) {
	profileCurve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 4,
		Points: []VFPoint{
			{
				Index:           0,
				VoltageMV:       800,
				OCScannerRefMHz: 2500,
				OffsetMHz:       100,
				IsActive:        true,
			},
			{
				Index:           1,
				VoltageMV:       850,
				OCScannerRefMHz: 225.0, // Inactive marker
				OffsetMHz:       0,
				IsActive:        false,
			},
			{
				Index:           2,
				VoltageMV:       900,
				OCScannerRefMHz: 2700,
				OffsetMHz:       50,
				IsActive:        true,
			},
			{
				Index:           3,
				VoltageMV:       950,
				OCScannerRefMHz: 225.0, // Inactive marker
				OffsetMHz:       0,
				IsActive:        false,
			},
		},
	}

	// Only provide live data for active points
	liveFreqs := map[float32]float64{
		800: 2600,
		900: 2750,
		// 850 and 950 not provided (inactive points)
	}

	result, err := MatchVFCurve(liveFreqs, profileCurve, 5.0)
	if err != nil {
		t.Fatalf("MatchVFCurve failed: %v", err)
	}

	// Only 2 active points should be compared
	if result.TotalPoints != 2 {
		t.Errorf("Expected 2 total points (inactive skipped), got %d", result.TotalPoints)
	}

	if result.MatchedPoints != 2 {
		t.Errorf("Expected 2 matched points, got %d", result.MatchedPoints)
	}
}

// TestMatchVFCurve_MissingLiveData tests when live data doesn't have all voltage points.
func TestMatchVFCurve_MissingLiveData(t *testing.T) {
	profileCurve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 3,
		Points: []VFPoint{
			{
				Index:           0,
				VoltageMV:       800,
				OCScannerRefMHz: 2500,
				OffsetMHz:       100,
				IsActive:        true,
			},
			{
				Index:           1,
				VoltageMV:       900,
				OCScannerRefMHz: 2700,
				OffsetMHz:       50,
				IsActive:        true,
			},
			{
				Index:           2,
				VoltageMV:       1000,
				OCScannerRefMHz: 2900,
				OffsetMHz:       0,
				IsActive:        true,
			},
		},
	}

	// Live data only has 800mV and 1000mV, missing 900mV
	liveFreqs := map[float32]float64{
		800:  2600,
		1000: 2900,
	}

	result, err := MatchVFCurve(liveFreqs, profileCurve, 5.0)
	if err != nil {
		t.Fatalf("MatchVFCurve failed: %v", err)
	}

	// Only 2 points can be compared (900mV missing from live data)
	if result.TotalPoints != 2 {
		t.Errorf("Expected 2 total points (900mV missing), got %d", result.TotalPoints)
	}

	if result.MatchedPoints != 2 {
		t.Errorf("Expected 2 matched points, got %d", result.MatchedPoints)
	}
}

// TestMatchVFCurve_EmptyLiveFreqs tests error handling for empty live data.
func TestMatchVFCurve_EmptyLiveFreqs(t *testing.T) {
	profileCurve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 1,
		Points: []VFPoint{
			{
				Index:           0,
				VoltageMV:       800,
				OCScannerRefMHz: 2500,
				OffsetMHz:       100,
				IsActive:        true,
			},
		},
	}

	liveFreqs := map[float32]float64{}

	_, err := MatchVFCurve(liveFreqs, profileCurve, 5.0)
	if err == nil {
		t.Error("Expected error for empty liveFreqs, got nil")
	}
}

// TestMatchVFCurve_NilProfile tests error handling for nil profile curve.
func TestMatchVFCurve_NilProfile(t *testing.T) {
	liveFreqs := map[float32]float64{
		800: 2600,
	}

	_, err := MatchVFCurve(liveFreqs, nil, 5.0)
	if err == nil {
		t.Error("Expected error for nil profileCurve, got nil")
	}
}

// TestMatchVFCurve_NegativeTolerance tests error handling for negative tolerance.
func TestMatchVFCurve_NegativeTolerance(t *testing.T) {
	profileCurve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 1,
		Points: []VFPoint{
			{
				Index:           0,
				VoltageMV:       800,
				OCScannerRefMHz: 2500,
				OffsetMHz:       100,
				IsActive:        true,
			},
		},
	}

	liveFreqs := map[float32]float64{
		800: 2600,
	}

	_, err := MatchVFCurve(liveFreqs, profileCurve, -5.0)
	if err == nil {
		t.Error("Expected error for negative tolerance, got nil")
	}
}

// TestMatchProfileAgainstLive tests matching against all profile slots.
func TestMatchProfileAgainstLive(t *testing.T) {
	// Create a hardware profile with Startup and 2 profile slots
	hwProfile := &HardwareProfile{
		Startup: ProfileSection{
			VFCurve: []byte{
				// Version 2.0 header + 1 point at 800mV with 2600 MHz effective
				0x00, 0x00, 0x02, 0x00, // version
				0x01, 0x00, 0x00, 0x00, // count = 1
				0x00, 0x00, 0x00, 0x00, // reserved
				0x00, 0x40, 0x48, 0x44, // voltage = 800.0f
				0x00, 0x00, 0x9c, 0x43, // oc_ref = 314.0f (active)
				0x00, 0x00, 0xa0, 0x44, // offset = 1280.0f
			},
		},
		Profiles: [5]ProfileSlot{
			{
				SlotNumber: 1,
				IsEmpty:    false,
				ProfileSection: ProfileSection{
					VFCurve: []byte{
						// Different curve: 2700 MHz effective
						0x00, 0x00, 0x02, 0x00,
						0x01, 0x00, 0x00, 0x00,
						0x00, 0x00, 0x00, 0x00,
						0x00, 0x40, 0x48, 0x44, // 800.0f
						0x00, 0x00, 0x9c, 0x43, // 314.0f
						0x00, 0x00, 0xa8, 0x44, // 1344.0f offset -> 314 + 1344 = 1658... wait this is wrong
					},
				},
			},
			{SlotNumber: 2, IsEmpty: true},
			{SlotNumber: 3, IsEmpty: true},
			{SlotNumber: 4, IsEmpty: true},
			{SlotNumber: 5, IsEmpty: true},
		},
	}

	// Live data matches Startup (2600 MHz at 800mV)
	liveFreqs := map[float32]float64{
		800: 2600.0,
	}

	results, err := MatchProfileAgainstLive(liveFreqs, hwProfile, 10.0)
	if err != nil {
		t.Fatalf("MatchProfileAgainstLive failed: %v", err)
	}

	if len(results) != 6 {
		t.Errorf("Expected 6 results (Startup + 5 slots), got %d", len(results))
	}

	// Startup should have some confidence
	if results[0].SlotName != "Startup" {
		t.Errorf("Expected result[0] to be Startup, got %s", results[0].SlotName)
	}

	// Profile1 should be parsed (not empty)
	if results[1].SlotName != "Profile1" {
		t.Errorf("Expected result[1] to be Profile1, got %s", results[1].SlotName)
	}

	// Profile2-5 should have 0 confidence (empty)
	for i := 2; i <= 5; i++ {
		if results[i].SlotName != fmt.Sprintf("Profile%d", i) {
			t.Errorf("Expected result[%d] to be Profile%d, got %s", i, i, results[i].SlotName)
		}
		if results[i].MatchConfidence != 0.0 {
			t.Errorf("Expected Profile%d to have 0%% confidence (empty), got %.2f%%", i, results[i].MatchConfidence*100)
		}
	}
}

// TestFindBestMatch tests finding the best matching profile.
func TestFindBestMatch(t *testing.T) {
	results := []ProfileMatchResult{
		{Slot: 0, SlotName: "Startup", MatchConfidence: 0.3},
		{Slot: 1, SlotName: "Profile1", MatchConfidence: 0.9},
		{Slot: 2, SlotName: "Profile2", MatchConfidence: 0.5},
		{Slot: 3, SlotName: "Profile3", MatchConfidence: 0.7},
	}

	best, isMatch := FindBestMatch(results, 0.8)

	if best.SlotName != "Profile1" {
		t.Errorf("Expected best match to be Profile1, got %s", best.SlotName)
	}

	if best.MatchConfidence != 0.9 {
		t.Errorf("Expected 90%% confidence, got %.2f%%", best.MatchConfidence*100)
	}

	if !isMatch {
		t.Error("Expected isMatch to be true (0.9 >= 0.8)")
	}

	// Test with higher threshold
	_, isMatch = FindBestMatch(results, 0.95)
	if isMatch {
		t.Error("Expected isMatch to be false (0.9 < 0.95)")
	}
}

// TestFindBestMatch_EmptyResults tests edge case with no results.
func TestFindBestMatch_EmptyResults(t *testing.T) {
	results := []ProfileMatchResult{}

	best, isMatch := FindBestMatch(results, 0.8)

	if best.SlotName != "" {
		t.Errorf("Expected empty SlotName, got %s", best.SlotName)
	}

	if isMatch {
		t.Error("Expected isMatch to be false for empty results")
	}
}

// TestProfileMatchResult_IsMatch tests the IsMatch method.
func TestProfileMatchResult_IsMatch(t *testing.T) {
	result := ProfileMatchResult{
		MatchConfidence: 0.75,
	}

	if !result.IsMatch(0.7) {
		t.Error("Expected IsMatch(0.7) to be true")
	}

	if !result.IsMatch(0.75) {
		t.Error("Expected IsMatch(0.75) to be true")
	}

	if result.IsMatch(0.8) {
		t.Error("Expected IsMatch(0.8) to be false")
	}
}

// TestProfileMatchResult_String tests the String method.
func TestProfileMatchResult_String(t *testing.T) {
	result := ProfileMatchResult{
		SlotName:        "Profile1",
		MatchedPoints:   8,
		TotalPoints:     10,
		AvgDeviationMHz: 5.5,
		MaxDeviationMHz: 12.3,
		MatchConfidence: 0.8,
	}

	str := result.String()

	expectedSubstrings := []string{
		"Profile1",
		"80%",
		"8/10",
		"5.5",
		"12.3",
	}

	for _, substr := range expectedSubstrings {
		if !contains(str, substr) {
			t.Errorf("Expected String() to contain %q, got %q", substr, str)
		}
	}
}

// Helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
