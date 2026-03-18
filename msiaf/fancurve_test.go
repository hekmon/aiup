// Package msiaf provides tooling for working with MSI Afterburner profiles and configuration.
package msiaf

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"
)

// TestUnmarshalSwAutoFanControlCurve tests parsing of a valid fan curve from real config data.
func TestUnmarshalSwAutoFanControlCurve(t *testing.T) {
	// Actual curve data from MSIAfterburner.cfg
	hexStr := "0000010004000000000000000000F0410000204200004842000048420000A0420000A0420000B4420000C8420000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	data, err := hex.DecodeString(hexStr)
	if err != nil {
		t.Fatalf("Failed to decode hex: %v", err)
	}

	curve, err := UnmarshalSwAutoFanControlCurve(data)
	if err != nil {
		t.Fatalf("UnmarshalSwAutoFanControlCurve failed: %v", err)
	}
	if curve == nil {
		t.Fatal("UnmarshalSwAutoFanControlCurve returned nil")
	}

	if curve.Version != FanCurveBinaryFormatVersion {
		t.Errorf("Expected version %d, got %d", FanCurveBinaryFormatVersion, curve.Version)
	}

	if len(curve.Points) != 4 {
		t.Errorf("Expected 4 points, got %d", len(curve.Points))
	}

	expectedPoints := []FanCurvePoint{
		{Temperature: 30.0, FanSpeed: 40.0},
		{Temperature: 50.0, FanSpeed: 50.0},
		{Temperature: 80.0, FanSpeed: 80.0},
		{Temperature: 90.0, FanSpeed: 100.0},
	}

	for i, expected := range expectedPoints {
		if i >= len(curve.Points) {
			break
		}
		if curve.Points[i].Temperature != expected.Temperature {
			t.Errorf("Point %d: expected temperature %.2f, got %.2f", i, expected.Temperature, curve.Points[i].Temperature)
		}
		if curve.Points[i].FanSpeed != expected.FanSpeed {
			t.Errorf("Point %d: expected fan speed %.2f, got %.2f", i, expected.FanSpeed, curve.Points[i].FanSpeed)
		}
	}
}

// TestSwAutoFanControlCurveInfo_Marshal tests serialization of a fan curve.
func TestSwAutoFanControlCurveInfo_Marshal(t *testing.T) {
	curve := &SwAutoFanControlCurveInfo{
		Version: FanCurveBinaryFormatVersion,
		Points: []FanCurvePoint{
			{Temperature: 30.0, FanSpeed: 40.0},
			{Temperature: 50.0, FanSpeed: 50.0},
			{Temperature: 80.0, FanSpeed: 80.0},
			{Temperature: 90.0, FanSpeed: 100.0},
		},
	}

	// Validate before marshaling
	if err := curve.Validate(); err != nil {
		t.Fatalf("Curve validation failed: %v", err)
	}

	data := curve.Marshal()

	if len(data) != FanCurveBlockSize {
		t.Errorf("Expected marshaled data to be %d bytes, got %d", FanCurveBlockSize, len(data))
	}

	// Verify we can unmarshal back to the same values
	curve2, err := UnmarshalSwAutoFanControlCurve(data)
	if err != nil {
		t.Fatalf("Unmarshal after marshal failed: %v", err)
	}
	if curve2 == nil {
		t.Fatal("Unmarshal after marshal returned nil")
	}

	if curve2.Version != curve.Version {
		t.Errorf("Version mismatch: expected %d, got %d", curve.Version, curve2.Version)
	}

	if len(curve2.Points) != len(curve.Points) {
		t.Errorf("Points count mismatch: expected %d, got %d", len(curve.Points), len(curve2.Points))
	}

	for i, expected := range curve.Points {
		if curve2.Points[i].Temperature != expected.Temperature {
			t.Errorf("Point %d: temperature mismatch: expected %.2f, got %.2f", i, expected.Temperature, curve2.Points[i].Temperature)
		}
		if curve2.Points[i].FanSpeed != expected.FanSpeed {
			t.Errorf("Point %d: fan speed mismatch: expected %.2f, got %.2f", i, expected.FanSpeed, curve2.Points[i].FanSpeed)
		}
	}
}

// TestSwAutoFanControlCurveInfo_Marshal_RoundTrip tests round-trip serialization.
func TestSwAutoFanControlCurveInfo_Marshal_RoundTrip(t *testing.T) {
	// Start with raw hex data
	hexStr := "0000010004000000000000000000F0410000204200004842000048420000A0420000A0420000B4420000C8420000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	originalData, _ := hex.DecodeString(hexStr)

	// Unmarshal -> Marshal -> Unmarshal
	curve1, err := UnmarshalSwAutoFanControlCurve(originalData)
	if err != nil {
		t.Fatalf("First unmarshal failed: %v", err)
	}

	marshaled := curve1.Marshal()

	curve2, err := UnmarshalSwAutoFanControlCurve(marshaled)
	if err != nil {
		t.Fatalf("Second unmarshal failed: %v", err)
	}

	// Compare the two curves
	if curve1.Version != curve2.Version {
		t.Errorf("Version mismatch after round trip: %d vs %d", curve1.Version, curve2.Version)
	}

	if len(curve1.Points) != len(curve2.Points) {
		t.Errorf("Points count mismatch after round trip: %d vs %d", len(curve1.Points), len(curve2.Points))
	}

	for i := range curve1.Points {
		if curve1.Points[i].Temperature != curve2.Points[i].Temperature {
			t.Errorf("Point %d: temperature mismatch: %.2f vs %.2f", i, curve1.Points[i].Temperature, curve2.Points[i].Temperature)
		}
		if curve1.Points[i].FanSpeed != curve2.Points[i].FanSpeed {
			t.Errorf("Point %d: fan speed mismatch: %.2f vs %.2f", i, curve1.Points[i].FanSpeed, curve2.Points[i].FanSpeed)
		}
	}

	// Verify the marshaled data matches the original (case-insensitive hex comparison)
	marshaledHex := hex.EncodeToString(marshaled)
	if !strings.EqualFold(marshaledHex, hexStr) {
		t.Errorf("Round-trip hex mismatch")
	}
}

// TestUnmarshalSwAutoFanControlCurve_InvalidData tests error handling for invalid inputs.
func TestUnmarshalSwAutoFanControlCurve_InvalidData(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectError string
	}{
		{
			name:        "nil data",
			data:        nil,
			expectError: "must be exactly 256 bytes",
		},
		{
			name:        "too short data",
			data:        []byte{0x00, 0x01, 0x02},
			expectError: "must be exactly 256 bytes",
		},
		{
			name:        "wrong size data",
			data:        make([]byte, 100),
			expectError: "must be exactly 256 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			curve, err := UnmarshalSwAutoFanControlCurve(tt.data)
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if curve != nil {
				t.Error("Expected nil curve on error, got non-nil")
			}
			if tt.expectError != "" && !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, got %v", tt.expectError, err)
			}
		})
	}
}

// TestUnmarshalSwAutoFanControlCurve_InvalidVersion tests version validation.
func TestUnmarshalSwAutoFanControlCurve_InvalidVersion(t *testing.T) {
	data := make([]byte, FanCurveBlockSize)
	// Set wrong version (0x00020000 instead of 0x00010000)
	data[0] = 0x00
	data[1] = 0x02
	data[2] = 0x00
	data[3] = 0x00

	curve, err := UnmarshalSwAutoFanControlCurve(data)
	if err == nil {
		t.Fatal("Expected error for invalid version, got nil")
	}
	if curve != nil {
		t.Error("Expected nil curve on invalid version")
	}
	if !strings.Contains(err.Error(), "unrecognized version") {
		t.Errorf("Expected version error, got: %v", err)
	}
	var fanErr *FanCurveError
	if !errors.As(err, &fanErr) {
		t.Error("Expected FanCurveError type")
	}
}

// TestUnmarshalSwAutoFanControlCurve_InvalidTemperature tests temperature validation.
func TestUnmarshalSwAutoFanControlCurve_InvalidTemperature(t *testing.T) {
	// Create a curve with temperature outside valid range (> 150°C)
	curve := &SwAutoFanControlCurveInfo{
		Version: FanCurveBinaryFormatVersion,
		Points: []FanCurvePoint{
			{Temperature: 200.0, FanSpeed: 50.0}, // Invalid: > 150°C
		},
	}

	data := curve.Marshal()
	_, err := UnmarshalSwAutoFanControlCurve(data)
	if err == nil {
		t.Fatal("Expected error for invalid temperature, got nil")
	}
	if !strings.Contains(err.Error(), "temperature") {
		t.Errorf("Expected temperature error, got: %v", err)
	}
}

// TestUnmarshalSwAutoFanControlCurve_InvalidFanSpeed tests fan speed validation.
func TestUnmarshalSwAutoFanControlCurve_InvalidFanSpeed(t *testing.T) {
	// Create a curve with fan speed outside valid range (> 100%)
	curve := &SwAutoFanControlCurveInfo{
		Version: FanCurveBinaryFormatVersion,
		Points: []FanCurvePoint{
			{Temperature: 50.0, FanSpeed: 150.0}, // Invalid: > 100%
		},
	}

	data := curve.Marshal()
	_, err := UnmarshalSwAutoFanControlCurve(data)
	if err == nil {
		t.Fatal("Expected error for invalid fan speed, got nil")
	}
	if !strings.Contains(err.Error(), "fan speed") {
		t.Errorf("Expected fan speed error, got: %v", err)
	}
}

// TestUnmarshalSwAutoFanControlCurve_UnsortedPoints tests point ordering validation.
func TestUnmarshalSwAutoFanControlCurve_UnsortedPoints(t *testing.T) {
	// Create a curve with unsorted points
	curve := &SwAutoFanControlCurveInfo{
		Version: FanCurveBinaryFormatVersion,
		Points: []FanCurvePoint{
			{Temperature: 50.0, FanSpeed: 50.0},
			{Temperature: 30.0, FanSpeed: 40.0}, // Invalid: lower than previous
			{Temperature: 80.0, FanSpeed: 80.0},
		},
	}

	data := curve.Marshal()
	_, err := UnmarshalSwAutoFanControlCurve(data)
	if err == nil {
		t.Fatal("Expected error for unsorted points, got nil")
	}
	if !strings.Contains(err.Error(), "sorted") {
		t.Errorf("Expected sorting error, got: %v", err)
	}
}

// TestSwAutoFanControlCurveInfo_Validate tests the Validate method.
func TestSwAutoFanControlCurveInfo_Validate(t *testing.T) {
	tests := []struct {
		name          string
		curve         *SwAutoFanControlCurveInfo
		expectError   bool
		errorContains string
	}{
		{
			name: "valid curve",
			curve: &SwAutoFanControlCurveInfo{
				Version: FanCurveBinaryFormatVersion,
				Points: []FanCurvePoint{
					{Temperature: 30.0, FanSpeed: 40.0},
					{Temperature: 50.0, FanSpeed: 50.0},
				},
			},
			expectError: false,
		},
		{
			name: "invalid version",
			curve: &SwAutoFanControlCurveInfo{
				Version: 0x00020000,
				Points: []FanCurvePoint{
					{Temperature: 30.0, FanSpeed: 40.0},
				},
			},
			expectError:   true,
			errorContains: "version",
		},
		{
			name: "no points",
			curve: &SwAutoFanControlCurveInfo{
				Version: FanCurveBinaryFormatVersion,
				Points:  []FanCurvePoint{},
			},
			expectError:   true,
			errorContains: "no points",
		},
		{
			name: "too many points",
			curve: &SwAutoFanControlCurveInfo{
				Version: FanCurveBinaryFormatVersion,
				Points:  make([]FanCurvePoint, FanCurveMaxPoints+1),
			},
			expectError:   true,
			errorContains: "exceeds maximum",
		},
		{
			name: "unsorted points",
			curve: &SwAutoFanControlCurveInfo{
				Version: FanCurveBinaryFormatVersion,
				Points: []FanCurvePoint{
					{Temperature: 50.0, FanSpeed: 50.0},
					{Temperature: 30.0, FanSpeed: 40.0},
				},
			},
			expectError:   true,
			errorContains: "sorted",
		},
		{
			name: "temperature too low",
			curve: &SwAutoFanControlCurveInfo{
				Version: FanCurveBinaryFormatVersion,
				Points: []FanCurvePoint{
					{Temperature: -100.0, FanSpeed: 50.0},
				},
			},
			expectError:   true,
			errorContains: "temperature",
		},
		{
			name: "temperature too high",
			curve: &SwAutoFanControlCurveInfo{
				Version: FanCurveBinaryFormatVersion,
				Points: []FanCurvePoint{
					{Temperature: 200.0, FanSpeed: 50.0},
				},
			},
			expectError:   true,
			errorContains: "temperature",
		},
		{
			name: "fan speed negative",
			curve: &SwAutoFanControlCurveInfo{
				Version: FanCurveBinaryFormatVersion,
				Points: []FanCurvePoint{
					{Temperature: 50.0, FanSpeed: -10.0},
				},
			},
			expectError:   true,
			errorContains: "fan speed",
		},
		{
			name: "fan speed over 100",
			curve: &SwAutoFanControlCurveInfo{
				Version: FanCurveBinaryFormatVersion,
				Points: []FanCurvePoint{
					{Temperature: 50.0, FanSpeed: 150.0},
				},
			},
			expectError:   true,
			errorContains: "fan speed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.curve.Validate()
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestSettings_SetFanControlCurve tests setting a curve with validation.
func TestSettings_SetFanControlCurve(t *testing.T) {
	settings := &Settings{}

	// Valid curve
	curve := &SwAutoFanControlCurveInfo{
		Version: FanCurveBinaryFormatVersion,
		Points: []FanCurvePoint{
			{Temperature: 30.0, FanSpeed: 40.0},
			{Temperature: 50.0, FanSpeed: 50.0},
		},
	}

	err := settings.SetFanControlCurve(curve)
	if err != nil {
		t.Fatalf("SetFanControlCurve failed: %v", err)
	}

	if settings.SwAutoFanControlCurve == nil {
		t.Fatal("SwAutoFanControlCurve is nil after SetFanControlCurve")
	}

	retrieved := settings.GetFanControlCurve()
	if retrieved == nil {
		t.Fatal("GetFanControlCurve returned nil")
	}

	if len(retrieved.Points) != 2 {
		t.Errorf("Expected 2 points, got %d", len(retrieved.Points))
	}

	// Invalid curve (unsorted points)
	invalidCurve := &SwAutoFanControlCurveInfo{
		Version: FanCurveBinaryFormatVersion,
		Points: []FanCurvePoint{
			{Temperature: 50.0, FanSpeed: 50.0},
			{Temperature: 30.0, FanSpeed: 40.0}, // Unsorted
		},
	}

	err = settings.SetFanControlCurve(invalidCurve)
	if err == nil {
		t.Fatal("SetFanControlCurve should have failed for invalid curve")
	}
	if !strings.Contains(err.Error(), "sorted") {
		t.Errorf("Expected sorting error, got: %v", err)
	}
}

// TestSettings_SetFanControlCurve_Clear tests clearing a curve.
func TestSettings_SetFanControlCurve_Clear(t *testing.T) {
	settings := &Settings{}

	// Set a valid curve first
	curve := &SwAutoFanControlCurveInfo{
		Version: FanCurveBinaryFormatVersion,
		Points:  []FanCurvePoint{{Temperature: 50.0, FanSpeed: 50.0}},
	}
	settings.SetFanControlCurve(curve)

	// Clear the curve
	err := settings.SetFanControlCurve(nil)
	if err != nil {
		t.Fatalf("SetFanControlCurve(nil) failed: %v", err)
	}

	if settings.SwAutoFanControlCurve != nil {
		t.Error("SwAutoFanControlCurve should be nil after clearing")
	}
	if settings.GetFanControlCurve() != nil {
		t.Error("GetFanControlCurve should return nil after clearing")
	}
}

// TestFanCurveError tests the FanCurveError error type.
func TestFanCurveError(t *testing.T) {
	err := &FanCurveError{
		Op:    "validate",
		Field: "point[2].temperature",
		Value: 200.0,
		Err:   ErrFanCurveInvalidTemperature,
	}

	// Test Error() method
	errStr := err.Error()
	if !strings.Contains(errStr, "validate") {
		t.Error("Error string should contain operation")
	}
	if !strings.Contains(errStr, "point[2].temperature") {
		t.Error("Error string should contain field")
	}
	if !strings.Contains(errStr, "200") {
		t.Error("Error string should contain value")
	}

	// Test Unwrap() method
	unwrapped := err.Unwrap()
	if unwrapped != ErrFanCurveInvalidTemperature {
		t.Error("Unwrap should return underlying error")
	}

	// Test errors.As
	var fanErr *FanCurveError
	if !errors.As(err, &fanErr) {
		t.Error("errors.As should work with FanCurveError")
	}
}

// TestSwAutoFanControlCurveInfo_Validate_EdgeCases tests boundary values.
func TestSwAutoFanControlCurveInfo_Validate_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		temperature float32
		fanSpeed    float32
		expectValid bool
	}{
		{"min temperature", FanCurveMinTemperature, 50.0, true},
		{"max temperature", FanCurveMaxTemperature, 50.0, true},
		{"min fan speed", 50.0, FanCurveMinFanSpeed, true},
		{"max fan speed", 50.0, FanCurveMaxFanSpeed, true},
		{"below min temp", FanCurveMinTemperature - 1, 50.0, false},
		{"above max temp", FanCurveMaxTemperature + 1, 50.0, false},
		{"negative fan speed", 50.0, -1.0, false},
		{"fan speed over 100", 50.0, 101.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			curve := &SwAutoFanControlCurveInfo{
				Version: FanCurveBinaryFormatVersion,
				Points:  []FanCurvePoint{{Temperature: tt.temperature, FanSpeed: tt.fanSpeed}},
			}
			err := curve.Validate()
			if tt.expectValid && err != nil {
				t.Errorf("Expected valid, got error: %v", err)
			}
			if !tt.expectValid && err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}
