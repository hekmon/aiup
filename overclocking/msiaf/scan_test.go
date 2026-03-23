package msiaf

import (
	"os"
	"path/filepath"
	"testing"
)

// TestScanWithRealProfiles tests the Scan function against the actual MSI Afterburner profiles directory.
// This test is skipped if the real profiles directory doesn't exist.
func TestScanWithRealProfiles(t *testing.T) {
	// Path to real MSI Afterburner profiles (WSL path on Windows)
	realProfilesDir := "/mnt/c/Program Files (x86)/MSI Afterburner/Profiles/"

	// Check if directory exists, skip if not
	if _, err := os.Stat(realProfilesDir); os.IsNotExist(err) {
		t.Skip("Real MSI Afterburner profiles directory not found, skipping test")
	}

	result, err := Scan(realProfilesDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Verify global config was found
	if result.GlobalConfigPath == "" {
		t.Error("Expected GlobalConfigPath to be set, got empty string")
	}

	if result.GlobalConfigPath != filepath.Join(realProfilesDir, GlobalConfigFile) {
		t.Errorf("Expected GlobalConfigPath to be '%s', got '%s'", filepath.Join(realProfilesDir, GlobalConfigFile), result.GlobalConfigPath)
	}

	// Verify at least one hardware profile was found
	if len(result.HardwareProfiles) == 0 {
		t.Error("Expected at least one hardware profile, got none")
	}

	// Verify the NVIDIA GPU profile was found (DEV_2B85 = RTX 5090)
	foundNVIDIA := false
	for _, hp := range result.HardwareProfiles {
		if hp.VendorID == "10DE" && hp.DeviceID == "2B85" {
			foundNVIDIA = true
			if hp.BusNumber != 1 {
				t.Errorf("Expected NVIDIA GPU BusNumber to be 1, got %d", hp.BusNumber)
			}
			if hp.SubsystemID != "89EC1043" {
				t.Errorf("Expected SubsystemID to be '89EC1043', got '%s'", hp.SubsystemID)
			}
		}
	}
	if !foundNVIDIA {
		t.Error("Expected to find NVIDIA RTX 5090 profile (VEN_10DE&DEV_2B85)")
	}
}

// TestScanWithMissingGlobalConfig tests that Scan returns an error when MSIAfterburner.cfg is missing.
func TestScanWithMissingGlobalConfig(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create a fake hardware profile file (without global config)
	hwFile := filepath.Join(tmpDir, "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg")
	if err := os.WriteFile(hwFile, []byte("[Startup]"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Scan should fail because global config is missing
	_, err := Scan(tmpDir)
	if err == nil {
		t.Fatal("Expected Scan to fail when global config is missing, got nil error")
	}

	if _, ok := err.(*os.PathError); ok {
		t.Fatalf("Expected custom error message, got PathError: %v", err)
	}
}

// TestScanWithNoHardwareProfiles tests that Scan returns an error when no VEN_*.cfg files exist.
func TestScanWithNoHardwareProfiles(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create only the global config file (no hardware profiles)
	globalFile := filepath.Join(tmpDir, GlobalConfigFile)
	if err := os.WriteFile(globalFile, []byte("[Settings]"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Scan should fail because no hardware profiles exist
	_, err := Scan(tmpDir)
	if err == nil {
		t.Fatal("Expected Scan to fail when no hardware profiles exist, got nil error")
	}
}

// TestScanFiltersPlaceholderProfiles tests that placeholder profiles (DEV_0000) are filtered out.
func TestScanFiltersPlaceholderProfiles(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create global config
	globalFile := filepath.Join(tmpDir, GlobalConfigFile)
	if err := os.WriteFile(globalFile, []byte("[Settings]"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a placeholder hardware profile (DEV_0000)
	placeholderFile := filepath.Join(tmpDir, "VEN_0000&DEV_0000&SUBSYS_00000000&REV_00&BUS_0&DEV_0&FN_0.cfg")
	if err := os.WriteFile(placeholderFile, []byte("[Startup]"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a real hardware profile
	realFile := filepath.Join(tmpDir, "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg")
	if err := os.WriteFile(realFile, []byte("[Startup]"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should only have 1 hardware profile (the real one)
	if len(result.HardwareProfiles) != 1 {
		t.Errorf("Expected 1 hardware profile, got %d", len(result.HardwareProfiles))
	}

	// Should have an error/warning about the placeholder
	if len(result.Errors) != 1 {
		t.Errorf("Expected 1 error about placeholder, got %d", len(result.Errors))
	}
}

// TestScanMultipleGPUs tests scanning with multiple GPU profile files.
func TestScanMultipleGPUs(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create global config
	globalFile := filepath.Join(tmpDir, GlobalConfigFile)
	if err := os.WriteFile(globalFile, []byte("[Settings]"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create two hardware profiles (simulating dual GPU setup)
	gpu1File := filepath.Join(tmpDir, "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg")
	if err := os.WriteFile(gpu1File, []byte("[Startup]"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	gpu2File := filepath.Join(tmpDir, "VEN_10DE&DEV_2B86&SUBSYS_89ED1043&REV_A1&BUS_2&DEV_0&FN_0.cfg")
	if err := os.WriteFile(gpu2File, []byte("[Startup]"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := Scan(tmpDir)
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// Should have 2 hardware profiles
	if len(result.HardwareProfiles) != 2 {
		t.Errorf("Expected 2 hardware profiles, got %d", len(result.HardwareProfiles))
	}

	// Verify both GPUs were found
	found := make(map[string]bool)
	for _, hp := range result.HardwareProfiles {
		key := hp.VendorID + "_" + hp.DeviceID
		found[key] = true
	}

	if !found["10DE_2B85"] {
		t.Error("Expected to find GPU 1 (DEV_2B85)")
	}
	if !found["10DE_2B86"] {
		t.Error("Expected to find GPU 2 (DEV_2B86)")
	}
}

// TestHardwareProfileInfoGetFilename tests that GetFilename() correctly reconstructs the original filename.
// This verifies the round-trip: parse filename → extract fields → reconstruct filename.
func TestHardwareProfileInfoGetFilename(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create global config
	globalFile := filepath.Join(tmpDir, GlobalConfigFile)
	if err := os.WriteFile(globalFile, []byte("[Settings]"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test cases with different hardware profiles
	testCases := []struct {
		name           string
		filename       string
		expectedVendor string
		expectedDevice string
		expectedBus    int
	}{
		{
			name:           "NVIDIA RTX 5090",
			filename:       "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg",
			expectedVendor: "10DE",
			expectedDevice: "2B85",
			expectedBus:    1,
		},
		{
			name:           "AMD GPU",
			filename:       "VEN_1002&DEV_73BF&SUBSYS_375E1002&REV_C7&BUS_3&DEV_0&FN_0.cfg",
			expectedVendor: "1002",
			expectedDevice: "73BF",
			expectedBus:    3,
		},
		{
			name:           "High Bus Number",
			filename:       "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_255&DEV_0&FN_0.cfg",
			expectedVendor: "10DE",
			expectedDevice: "2B85",
			expectedBus:    255,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create test profile file
			testFile := filepath.Join(tmpDir, tc.filename)
			if err := os.WriteFile(testFile, []byte("[Startup]"), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Scan to get HardwareProfileInfo
			result, err := Scan(tmpDir)
			if err != nil {
				t.Fatalf("Scan failed: %v", err)
			}

			// Find the profile we're testing
			var profile *HardwareProfileInfo
			for _, hp := range result.HardwareProfiles {
				if hp.VendorID == tc.expectedVendor && hp.DeviceID == tc.expectedDevice && hp.BusNumber == tc.expectedBus {
					profile = &hp
					break
				}
			}

			if profile == nil {
				t.Fatalf("Could not find profile with VendorID=%s, DeviceID=%s, BusNumber=%d",
					tc.expectedVendor, tc.expectedDevice, tc.expectedBus)
			}

			// Test GetFilename() round-trip
			reconstructedFilename := profile.GetFilename()
			if reconstructedFilename != tc.filename {
				t.Errorf("GetFilename() round-trip failed:\n  Original:      %s\n  Reconstructed: %s", tc.filename, reconstructedFilename)
			}

			// Clean up for next test case
			os.Remove(testFile)
		})
	}
}
