// Package msiaf provides tooling for working with MSI Afterburner profiles and configuration.
package msiaf

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseHardwareProfile_Integration tests parsing of actual hardware profile files.
func TestParseHardwareProfile_Integration(t *testing.T) {
	// Find the RTX 5090 profile file
	profilePath := filepath.Join("..", "LocalProfiles", "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg")

	// Check if file exists first
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Skipf("Profile file not found: %s", profilePath)
	}

	profile, err := ParseHardwareProfile(profilePath)
	if err != nil {
		t.Fatalf("Failed to parse hardware profile: %v", err)
	}

	// Verify Startup section
	if profile.Startup.GetFormat() != 2 {
		t.Errorf("Expected Startup.Format=2, got %d", profile.Startup.GetFormat())
	}
	if profile.Startup.GetPowerLimit() != 100 {
		t.Errorf("Expected Startup.PowerLimit=100, got %d", profile.Startup.GetPowerLimit())
	}
	if profile.Startup.GetCoreClkBoost() != 1000000 {
		t.Errorf("Expected Startup.CoreClkBoost=1000000, got %d", profile.Startup.GetCoreClkBoost())
	}
	if profile.Startup.GetMemClkBoost() != 3000000 {
		t.Errorf("Expected Startup.MemClkBoost=3000000, got %d", profile.Startup.GetMemClkBoost())
	}
	if profile.Startup.GetFanMode() != 1 {
		t.Errorf("Expected Startup.FanMode=1, got %d", profile.Startup.GetFanMode())
	}
	if profile.Startup.GetFanSpeed() != 30 {
		t.Errorf("Expected Startup.FanSpeed=30, got %d", profile.Startup.GetFanSpeed())
	}

	// Verify fields are actually set (not nil)
	if profile.Startup.Format == nil {
		t.Error("Expected Startup.Format to be non-nil")
	}
	if profile.Startup.PowerLimit == nil {
		t.Error("Expected Startup.PowerLimit to be non-nil")
	}

	// Verify VFCurve is parsed (should be non-empty)
	if len(profile.Startup.VFCurve) == 0 {
		t.Error("Expected Startup.VFCurve to be non-empty")
	}

	// Verify Profiles array is initialized
	if len(profile.Profiles) != 5 {
		t.Errorf("Expected 5 profile slots, got %d", len(profile.Profiles))
	}

	// Verify Profile1 is populated (should have same data as Startup in test file)
	if profile.Profiles[0].IsEmpty {
		t.Error("Expected Profile1 to be populated")
	}
	if profile.Profiles[0].GetPowerLimit() != 100 {
		t.Errorf("Expected Profile1.PowerLimit=100, got %d", profile.Profiles[0].GetPowerLimit())
	}

	// Verify Defaults section exists
	if profile.Defaults.GetFormat() != 2 {
		t.Errorf("Expected Defaults.Format=2, got %d", profile.Defaults.GetFormat())
	}

	// Verify PreSuspendedMode has Format but empty values
	if profile.PreSuspendedMode.Format == nil {
		t.Error("Expected PreSuspendedMode.Format to be non-nil")
	}
	if profile.PreSuspendedMode.PowerLimit != nil {
		t.Errorf("Expected PreSuspendedMode.PowerLimit to be nil (empty), got %v", *profile.PreSuspendedMode.PowerLimit)
	}

	// Verify FilePath is stored
	if profile.FilePath() != profilePath {
		t.Errorf("Expected FilePath=%s, got %s", profilePath, profile.FilePath())
	}
}

// TestParseHardwareProfile_EmptyProfile tests parsing of minimal/empty profile.
func TestParseHardwareProfile_EmptyProfile(t *testing.T) {
	// Find the placeholder profile file
	profilePath := filepath.Join("..", "LocalProfiles", "VEN_0000&DEV_0000&SUBSYS_00000000&REV_00&BUS_0&DEV_0&FN_0.cfg")

	// Check if file exists first
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Skipf("Profile file not found: %s", profilePath)
	}

	profile, err := ParseHardwareProfile(profilePath)
	if err != nil {
		t.Fatalf("Failed to parse hardware profile: %v", err)
	}

	// Verify basic structure is initialized
	if profile.Startup.GetFormat() != 2 {
		t.Errorf("Expected Startup.Format=2, got %d", profile.Startup.GetFormat())
	}
	if profile.Startup.GetPowerLimit() != 100 {
		t.Errorf("Expected Startup.PowerLimit=100, got %d", profile.Startup.GetPowerLimit())
	}
}

// TestGetProfile tests the GetProfile accessor method.
func TestGetProfile(t *testing.T) {
	profilePath := filepath.Join("..", "LocalProfiles", "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg")

	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Skipf("Profile file not found: %s", profilePath)
	}

	profile, err := ParseHardwareProfile(profilePath)
	if err != nil {
		t.Fatalf("Failed to parse hardware profile: %v", err)
	}

	// Test valid slot numbers
	for i := 1; i <= 5; i++ {
		slot := profile.GetProfile(i)
		if slot == nil {
			t.Errorf("GetProfile(%d) returned nil", i)
		}
		if slot.SlotNumber != i {
			t.Errorf("GetProfile(%d).SlotNumber = %d, expected %d", i, slot.SlotNumber, i)
		}
	}

	// Test invalid slot numbers
	if slot := profile.GetProfile(0); slot != nil {
		t.Error("GetProfile(0) should return nil")
	}
	if slot := profile.GetProfile(6); slot != nil {
		t.Error("GetProfile(6) should return nil")
	}
	if slot := profile.GetProfile(-1); slot != nil {
		t.Error("GetProfile(-1) should return nil")
	}
}

// TestGetCurrentSettings tests the GetCurrentSettings accessor method.
func TestGetCurrentSettings(t *testing.T) {
	profilePath := filepath.Join("..", "LocalProfiles", "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg")

	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Skipf("Profile file not found: %s", profilePath)
	}

	profile, err := ParseHardwareProfile(profilePath)
	if err != nil {
		t.Fatalf("Failed to parse hardware profile: %v", err)
	}

	current := profile.GetCurrentSettings()
	if current == nil {
		t.Fatal("GetCurrentSettings returned nil")
	}
	if current.GetPowerLimit() != 100 {
		t.Errorf("Expected current settings PowerLimit=100, got %d", current.GetPowerLimit())
	}
}

// TestGetDefaults tests the GetDefaults accessor method.
func TestGetDefaults(t *testing.T) {
	profilePath := filepath.Join("..", "LocalProfiles", "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg")

	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Skipf("Profile file not found: %s", profilePath)
	}

	profile, err := ParseHardwareProfile(profilePath)
	if err != nil {
		t.Fatalf("Failed to parse hardware profile: %v", err)
	}

	defaults := profile.GetDefaults()
	if defaults == nil {
		t.Fatal("GetDefaults returned nil")
	}
	if defaults.GetFormat() != 2 {
		t.Errorf("Expected defaults Format=2, got %d", defaults.GetFormat())
	}
}

// TestHardwareProfileInfo_LoadProfile tests the LoadProfile method on HardwareProfileInfo.
func TestHardwareProfileInfo_LoadProfile(t *testing.T) {
	profilePath := filepath.Join("..", "LocalProfiles", "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg")

	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Skipf("Profile file not found: %s", profilePath)
	}

	info := HardwareProfileInfo{
		FilePath: profilePath,
	}

	profile, err := info.LoadProfile()
	if err != nil {
		t.Fatalf("LoadProfile failed: %v", err)
	}

	if profile == nil {
		t.Fatal("LoadProfile returned nil")
	}
	if profile.FilePath() != profilePath {
		t.Errorf("Expected FilePath=%s, got %s", profilePath, profile.FilePath())
	}
}

// TestProfileSection_PointerFields tests that pointer fields distinguish nil vs set values.
func TestProfileSection_PointerFields(t *testing.T) {
	ps := &ProfileSection{}

	// Empty section should have all nil pointers
	if ps.PowerLimit != nil {
		t.Error("Empty section PowerLimit should be nil")
	}
	if ps.CoreClkBoost != nil {
		t.Error("Empty section CoreClkBoost should be nil")
	}

	// Add a field using setter
	ps.SetPowerLimit(100)

	if ps.PowerLimit == nil {
		t.Error("PowerLimit should be non-nil after setting")
	}
	if *ps.PowerLimit != 100 {
		t.Errorf("Expected PowerLimit=100, got %d", *ps.PowerLimit)
	}

	// Zero value should still be distinguishable from nil
	ps.SetCoreClkBoost(0)
	if ps.CoreClkBoost == nil {
		t.Error("CoreClkBoost should be non-nil even when set to 0")
	}
	if *ps.CoreClkBoost != 0 {
		t.Errorf("Expected CoreClkBoost=0, got %d", *ps.CoreClkBoost)
	}
}

// TestProfileSection_HasSettings tests the HasSettings method.
func TestProfileSection_HasSettings(t *testing.T) {
	ps := &ProfileSection{}

	// Empty section should have no settings
	if ps.HasSettings() {
		t.Error("Empty section should not have settings")
	}

	// Add a field using setter
	ps.SetPowerLimit(100)

	if !ps.HasSettings() {
		t.Error("Section with fields should have settings")
	}
}

// TestParseProfileSection_VFCurve tests VFCurve hex blob parsing.
func TestParseProfileSection_VFCurve(t *testing.T) {
	ps := &ProfileSection{}

	// Test with a sample VFCurve hex string (shortened version)
	hexCurve := "000002007F000000000000000000E14300006143"
	parseProfileSection(ps, "VFCurve", hexCurve)

	if ps.VFCurve == nil {
		t.Fatal("VFCurve should not be nil")
	}
	if len(ps.VFCurve) == 0 {
		t.Error("VFCurve should not be empty")
	}

	// Verify it was decoded from hex
	expectedLen := len(hexCurve) / 2
	if len(ps.VFCurve) != expectedLen {
		t.Errorf("Expected VFCurve length %d, got %d", expectedLen, len(ps.VFCurve))
	}
}

// TestParseProfileSection_AllFields tests parsing of all ProfileSection fields.
func TestParseProfileSection_AllFields(t *testing.T) {
	ps := &ProfileSection{}

	// Parse all fields
	parseProfileSection(ps, "Format", "2")
	parseProfileSection(ps, "PowerLimit", "150")
	parseProfileSection(ps, "CoreClkBoost", "500000")
	parseProfileSection(ps, "MemClkBoost", "1000000")
	parseProfileSection(ps, "FanMode", "1")
	parseProfileSection(ps, "FanSpeed", "75")
	parseProfileSection(ps, "FanMode2", "0")
	parseProfileSection(ps, "FanSpeed2", "50")

	if ps.GetFormat() != 2 {
		t.Errorf("Format: expected 2, got %d", ps.GetFormat())
	}
	if ps.GetPowerLimit() != 150 {
		t.Errorf("PowerLimit: expected 150, got %d", ps.GetPowerLimit())
	}
	if ps.GetCoreClkBoost() != 500000 {
		t.Errorf("CoreClkBoost: expected 500000, got %d", ps.GetCoreClkBoost())
	}
	if ps.GetMemClkBoost() != 1000000 {
		t.Errorf("MemClkBoost: expected 1000000, got %d", ps.GetMemClkBoost())
	}
	if ps.GetFanMode() != 1 {
		t.Errorf("FanMode: expected 1, got %d", ps.GetFanMode())
	}
	if ps.GetFanSpeed() != 75 {
		t.Errorf("FanSpeed: expected 75, got %d", ps.GetFanSpeed())
	}
	if ps.GetFanMode2() != 0 {
		t.Errorf("FanMode2: expected 0, got %d", ps.GetFanMode2())
	}
	if ps.GetFanSpeed2() != 50 {
		t.Errorf("FanSpeed2: expected 50, got %d", ps.GetFanSpeed2())
	}

	// Verify all fields are non-nil
	if ps.Format == nil || ps.PowerLimit == nil || ps.CoreClkBoost == nil {
		t.Error("Expected fields to be non-nil")
	}
}

// TestParseProfileSection_EmptyValues tests handling of empty values.
func TestParseProfileSection_EmptyValues(t *testing.T) {
	ps := &ProfileSection{}

	// Empty values should not be parsed (common in PreSuspendedMode)
	parseProfileSection(ps, "PowerLimit", "")
	parseProfileSection(ps, "CoreClkBoost", "")

	// Fields should remain nil
	if ps.PowerLimit != nil {
		t.Errorf("Empty PowerLimit should be nil, got %d", *ps.PowerLimit)
	}
	if ps.CoreClkBoost != nil {
		t.Errorf("Empty CoreClkBoost should be nil, got %d", *ps.CoreClkBoost)
	}
}

// TestParseProfileMiscSettings tests parsing of Settings section.
func TestParseProfileMiscSettings(t *testing.T) {
	s := &ProfileMiscSettings{}

	parseProfileMiscSettings(s, "CaptureDefaults", "0")
	if s.GetCaptureDefaults() != 0 {
		t.Errorf("CaptureDefaults: expected 0, got %d", s.GetCaptureDefaults())
	}
	if s.CaptureDefaults == nil {
		t.Error("CaptureDefaults should be non-nil when set")
	}

	parseProfileMiscSettings(s, "CaptureDefaults", "1")
	if s.GetCaptureDefaults() != 1 {
		t.Errorf("CaptureDefaults: expected 1, got %d", s.GetCaptureDefaults())
	}
}

// TestProfileSlot_Initialization tests that profile slots are properly initialized.
func TestProfileSlot_Initialization(t *testing.T) {
	profilePath := filepath.Join("..", "LocalProfiles", "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg")

	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		t.Skipf("Profile file not found: %s", profilePath)
	}

	profile, err := ParseHardwareProfile(profilePath)
	if err != nil {
		t.Fatalf("Failed to parse hardware profile: %v", err)
	}

	// All 5 slots should be initialized
	for i := 0; i < 5; i++ {
		if profile.Profiles[i].SlotNumber != i+1 {
			t.Errorf("Profile[%d].SlotNumber = %d, expected %d", i, profile.Profiles[i].SlotNumber, i+1)
		}
	}

	// Only Profile1 is populated in test file (Profile2-5 don't exist)
	if profile.Profiles[0].IsEmpty {
		t.Error("Profile1 should not be empty")
	}
	for i := 1; i < 5; i++ {
		if !profile.Profiles[i].IsEmpty {
			t.Errorf("Profile[%d] should be empty (not in file)", i+1)
		}
	}
}
