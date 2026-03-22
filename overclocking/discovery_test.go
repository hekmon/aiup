// Package overclocking provides high-level GPU overclocking orchestration.
package overclocking

import (
	"encoding/json"
	"testing"
)

// TestMatchGPUName tests the GPU name matching logic.
func TestMatchGPUName(t *testing.T) {
	tests := []struct {
		name        string
		profileDesc string
		nvvName     string
		wantMatch   bool
	}{
		{
			name:        "exact_match_with_manufacturer",
			profileDesc: "ASUS NVIDIA GeForce RTX 5090",
			nvvName:     "NVIDIA GeForce RTX 5090",
			wantMatch:   true,
		},
		{
			name:        "exact_match_no_manufacturer",
			profileDesc: "NVIDIA GeForce RTX 4090",
			nvvName:     "NVIDIA GeForce RTX 4090",
			wantMatch:   true,
		},
		{
			name:        "different_manufacturer_same_gpu",
			profileDesc: "MSI NVIDIA GeForce RTX 4080",
			nvvName:     "NVIDIA GeForce RTX 4080",
			wantMatch:   true,
		},
		{
			name:        "no_match_different_gpu",
			profileDesc: "ASUS NVIDIA GeForce RTX 4090",
			nvvName:     "NVIDIA GeForce RTX 3080",
			wantMatch:   false,
		},
		{
			name:        "empty_profile_desc",
			profileDesc: "",
			nvvName:     "NVIDIA GeForce RTX 4090",
			wantMatch:   false,
		},
		{
			name:        "empty_nvv_name",
			profileDesc: "ASUS NVIDIA GeForce RTX 4090",
			nvvName:     "",
			wantMatch:   false,
		},
		{
			name:        "both_empty",
			profileDesc: "",
			nvvName:     "",
			wantMatch:   false,
		},
		{
			name:        "case_insensitive_match",
			profileDesc: "asus nvidia geforce rtx 4090",
			nvvName:     "NVIDIA GeForce RTX 4090",
			wantMatch:   true,
		},
		{
			name:        "partial_gpu_name_no_match",
			profileDesc: "ASUS NVIDIA GeForce RTX 4090",
			nvvName:     "RTX 4090",
			wantMatch:   true, // Contains "RTX 4090"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchGPUName(tt.profileDesc, tt.nvvName)
			if got != tt.wantMatch {
				t.Errorf("matchGPUName(%q, %q) = %v, want %v", tt.profileDesc, tt.nvvName, got, tt.wantMatch)
			}
		})
	}
}

// TestDiscoveryResultString tests the String method for various scenarios.
func TestDiscoveryResultString(t *testing.T) {
	tests := []struct {
		name     string
		result   *DiscoveryResult
		wantNull bool
	}{
		{
			name:     "nil_result",
			result:   nil,
			wantNull: true,
		},
		{
			name: "empty_gpus",
			result: &DiscoveryResult{
				GPUs: []GPUInfo{},
			},
			wantNull: true,
		},
		{
			name: "single_gpu_no_errors",
			result: &DiscoveryResult{
				ProfilesDir:      "/path/to/Profiles",
				GlobalConfigPath: "/path/to/Profiles/MSIAfterburner.cfg",
				GPUs: []GPUInfo{
					{
						Index:           0,
						Name:            "NVIDIA GeForce RTX 5090",
						FullDescription: "ASUS NVIDIA GeForce RTX 5090",
						ProfilePath:     "/path/to/Profiles/VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg",
					},
				},
				Errors: []string{},
			},
			wantNull: false,
		},
		{
			name: "multiple_gpus_with_errors",
			result: &DiscoveryResult{
				ProfilesDir: "/path/to/Profiles",
				GPUs: []GPUInfo{
					{Index: 0, Name: "NVIDIA GeForce RTX 5090", FullDescription: "ASUS NVIDIA GeForce RTX 5090"},
					{Index: 1, Name: "NVIDIA GeForce RTX 4090", FullDescription: "MSI NVIDIA GeForce RTX 4090"},
				},
				Errors: []string{"GPU 2 detected but has no profile"},
			},
			wantNull: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.String()
			if tt.wantNull && got != "No GPUs discovered" {
				t.Errorf("String() = %q, want %q", got, "No GPUs discovered")
			}
			if !tt.wantNull && got == "No GPUs discovered" {
				t.Errorf("String() returned null message for non-empty result")
			}
		})
	}
}

// TestGPUInfoJSONSerialization tests that GPUInfo serializes correctly to JSON.
func TestGPUInfoJSONSerialization(t *testing.T) {
	gpu := GPUInfo{
		Index:           0,
		Name:            "NVIDIA GeForce RTX 5090",
		VendorID:        "10DE",
		DeviceID:        "2B85",
		SubsystemID:     "89EC1043",
		BusNumber:       1,
		DeviceNumber:    0,
		FunctionNumber:  0,
		ProfilePath:     "C:\\Profiles\\test.cfg",
		Manufacturer:    "ASUS",
		FullDescription: "ASUS NVIDIA GeForce RTX 5090",
	}

	data, err := json.Marshal(gpu)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Verify expected JSON keys are present
	jsonStr := string(data)
	expectedKeys := []string{
		"index",
		"name",
		"vendor_id",
		"device_id",
		"subsystem_id",
		"bus_number",
		"device_number",
		"function_number",
		"profile_path",
		"manufacturer",
		"full_description",
	}

	for _, key := range expectedKeys {
		expected := `"` + key + `":`
		if !contains(jsonStr, expected) {
			t.Errorf("JSON missing expected key %q: %s", key, jsonStr)
		}
	}
}

// TestDiscoveryResultJSONSerialization tests that DiscoveryResult serializes correctly to JSON.
func TestDiscoveryResultJSONSerialization(t *testing.T) {
	result := &DiscoveryResult{
		ProfilesDir:      "C:\\Profiles",
		GlobalConfigPath: "C:\\Profiles\\MSIAfterburner.cfg",
		GPUs: []GPUInfo{
			{
				Index:    0,
				Name:     "NVIDIA GeForce RTX 5090",
				VendorID: "10DE",
				DeviceID: "2B85",
			},
		},
		Errors: []string{"Test error message"},
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	// Verify expected JSON keys are present
	jsonStr := string(data)
	expectedKeys := []string{
		"profiles_dir",
		"global_config_path",
		"gpus",
		"errors",
	}

	for _, key := range expectedKeys {
		expected := `"` + key + `":`
		if !contains(jsonStr, expected) {
			t.Errorf("JSON missing expected key %q: %s", key, jsonStr)
		}
	}

	// Verify errors field is present (not omitted since it has values)
	if !contains(jsonStr, `"errors"`) {
		t.Errorf("JSON should include errors field when non-empty: %s", jsonStr)
	}
}

// TestDiscoveryResultJSONSerializationNoErrors tests that empty errors field is omitted.
func TestDiscoveryResultJSONSerializationNoErrors(t *testing.T) {
	result := &DiscoveryResult{
		ProfilesDir:      "C:\\Profiles",
		GlobalConfigPath: "C:\\Profiles\\MSIAfterburner.cfg",
		GPUs: []GPUInfo{
			{
				Index: 0,
				Name:  "NVIDIA GeForce RTX 5090",
			},
		},
		Errors: []string{}, // Empty slice should be omitted
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	jsonStr := string(data)

	// Verify errors field is omitted when empty
	if contains(jsonStr, `"errors"`) {
		t.Errorf("JSON should omit errors field when empty: %s", jsonStr)
	}
}

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
