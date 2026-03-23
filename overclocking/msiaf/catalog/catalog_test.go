package catalog

import (
	"strings"
	"testing"
)

// TestLookupGPU_KnownNVIDIA tests GPU lookup for known NVIDIA GPUs.
// Device IDs are from pci.ids database (auto-generated).
func TestLookupGPU_KnownNVIDIA(t *testing.T) {
	tests := []struct {
		deviceID string
		expected string
	}{
		{"2B85", "GeForce RTX 5090"},
		{"2684", "GeForce RTX 4090"},
		{"2704", "GeForce RTX 4080"},
		{"2204", "GeForce RTX 3090"},
		{"2503", "GeForce RTX 3060"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			info := LookupGPU("10DE", tt.deviceID)
			if !info.IsKnown {
				t.Errorf("Expected GPU to be known, got IsKnown=false")
			}
			if info.VendorName != "NVIDIA" {
				t.Errorf("Expected VendorName='NVIDIA', got '%s'", info.VendorName)
			}
			if info.GPUName != tt.expected {
				t.Errorf("Expected GPUName='%s', got '%s'", tt.expected, info.GPUName)
			}
		})
	}
}

// TestLookupGPU_KnownAMD tests GPU lookup for known AMD GPUs.
// Device IDs are from pci.ids database (auto-generated).
func TestLookupGPU_KnownAMD(t *testing.T) {
	tests := []struct {
		deviceID string
		expected string
	}{
		{"73AF", "Radeon RX 6900 XT"},
		{"73AE", "Radeon Pro V620 Mx"},
		{"73BF", "Radeon RX 6800/6800 XT / 6900 XT"},
		{"73FF", "Radeon RX 6600/6600 XT/6600M"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			info := LookupGPU("1002", tt.deviceID)
			if !info.IsKnown {
				t.Errorf("Expected GPU to be known, got IsKnown=false")
			}
			if info.VendorName != "AMD" {
				t.Errorf("Expected VendorName='AMD', got '%s'", info.VendorName)
			}
			if info.GPUName != tt.expected {
				t.Errorf("Expected GPUName='%s', got '%s'", tt.expected, info.GPUName)
			}
		})
	}
}

// TestLookupGPU_KnownIntel tests GPU lookup for known Intel GPUs.
// Device IDs are from pci.ids database (auto-generated).
func TestLookupGPU_KnownIntel(t *testing.T) {
	tests := []struct {
		deviceID string
		expected string
	}{
		{"56A0", "Arc A770"},
		{"56A1", "Arc A750"},
		{"56A2", "Arc A580"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			info := LookupGPU("8086", tt.deviceID)
			if !info.IsKnown {
				t.Errorf("Expected GPU to be known, got IsKnown=false")
			}
			if info.VendorName != "Intel" {
				t.Errorf("Expected VendorName='Intel', got '%s'", info.VendorName)
			}
			if info.GPUName != tt.expected {
				t.Errorf("Expected GPUName='%s', got '%s'", tt.expected, info.GPUName)
			}
		})
	}
}

// TestLookupGPU_Unknown tests GPU lookup for unknown Device IDs.
func TestLookupGPU_Unknown(t *testing.T) {
	info := LookupGPU("10DE", "FFFF")
	if info.IsKnown {
		t.Errorf("Expected IsKnown=false for unknown GPU, got true")
	}
	if !strings.Contains(info.GPUName, "Unknown") {
		t.Errorf("Expected GPUName to contain 'Unknown', got '%s'", info.GPUName)
	}
	if !strings.Contains(info.GPUName, "DEV_FFFF") {
		t.Errorf("Expected GPUName to contain raw Device ID, got '%s'", info.GPUName)
	}
}

// TestLookupGPU_UnknownVendor tests GPU lookup for unknown Vendor IDs.
func TestLookupGPU_UnknownVendor(t *testing.T) {
	info := LookupGPU("9999", "1234")
	if info.IsKnown {
		t.Errorf("Expected IsKnown=false for unknown vendor, got true")
	}
	if !strings.Contains(info.VendorName, "Unknown Vendor") {
		t.Errorf("Expected VendorName to indicate unknown, got '%s'", info.VendorName)
	}
}

// TestLookupManufacturer_Known tests manufacturer lookup for known vendor codes.
func TestLookupManufacturer_Known(t *testing.T) {
	tests := []struct {
		subsystemID string
		expected    string
	}{
		{"89EC1043", "ASUS"},
		{"00001043", "ASUS"},
		{"00001462", "MSI"},
		{"000010DE", "NVIDIA (Founders Edition)"},
		{"00003842", "EVGA"},
		{"00001458", "Gigabyte"},
		{"0000174B", "Sapphire"},
		{"00001787", "PowerColor"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := LookupManufacturer(tt.subsystemID)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

// TestLookupManufacturer_Unknown tests manufacturer lookup for unknown vendor codes.
func TestLookupManufacturer_Unknown(t *testing.T) {
	result := LookupManufacturer("00009999")
	if !strings.Contains(result, "Unknown") {
		t.Errorf("Expected 'Unknown', got '%s'", result)
	}
	if !strings.Contains(result, "VID_9999") {
		t.Errorf("Expected raw vendor ID in result, got '%s'", result)
	}
}

// TestLookupManufacturer_ShortInput tests manufacturer lookup with short input.
func TestLookupManufacturer_ShortInput(t *testing.T) {
	result := LookupManufacturer("123")
	if result != "Unknown" {
		t.Errorf("Expected 'Unknown' for short input, got '%s'", result)
	}
}

// TestGetFullGPUDescription_Known tests the full GPU description for known GPUs.
func TestGetFullGPUDescription_Known(t *testing.T) {
	result := GetFullGPUDescription("10DE", "2B85", "1043")
	// Just verify it returns a known GPU with manufacturer
	if !strings.Contains(result, "GeForce RTX 5090") {
		t.Errorf("Expected result to contain 'GeForce RTX 5090', got '%s'", result)
	}
	if !strings.Contains(result, "ASUS") {
		t.Errorf("Expected result to contain 'ASUS', got '%s'", result)
	}
}

// TestGetFullGPUDescription_UnknownGPU tests full GPU description for unknown GPU.
func TestGetFullGPUDescription_UnknownGPU(t *testing.T) {
	result := GetFullGPUDescription("10DE", "FFFF", "89EC1043")
	if !strings.Contains(result, "Unknown GPU") {
		t.Errorf("Expected result to contain 'Unknown GPU', got '%s'", result)
	}
	if !strings.Contains(result, "ASUS") {
		t.Errorf("Expected result to contain manufacturer 'ASUS', got '%s'", result)
	}
}

// TestGetFullGPUDescription_UnknownManufacturer tests full GPU description for unknown manufacturer.
func TestGetFullGPUDescription_UnknownManufacturer(t *testing.T) {
	result := GetFullGPUDescription("10DE", "2B85", "00009999")
	if !strings.Contains(result, "RTX 5090") {
		t.Errorf("Expected result to contain GPU name, got '%s'", result)
	}
	if !strings.Contains(result, "Unknown") {
		t.Errorf("Expected result to contain 'Unknown' for manufacturer, got '%s'", result)
	}
}
