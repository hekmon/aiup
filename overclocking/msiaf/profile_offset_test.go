package msiaf

import (
	"encoding/hex"
	"testing"
)

// TestGetOffsetMode_FixedOffset tests detection of fixed offset mode (slider mode)
func TestGetOffsetMode_FixedOffset(t *testing.T) {
	// Create a V-F curve with uniform +123 MHz offset (like Startup profile)
	// This is a minimal valid curve with a few points all at +123 MHz
	curve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 3,
		Points: []VFPoint{
			{VoltageMV: 800.0, BaseFreqMHz: 700.0, OffsetMHz: 123.0, IsActive: true},
			{VoltageMV: 900.0, BaseFreqMHz: 800.0, OffsetMHz: 123.0, IsActive: true},
			{VoltageMV: 1000.0, BaseFreqMHz: 900.0, OffsetMHz: 123.0, IsActive: true},
		},
	}

	// Marshal the curve to get hex data
	hexData, err := curve.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal curve: %v", err)
	}

	vfCurveBytes, err := hex.DecodeString(hexData)
	if err != nil {
		t.Fatalf("Failed to decode hex: %v", err)
	}

	// Create profile section with CoreClkBoost = 123000 kHz (= 123 MHz)
	coreBoost := 123000
	ps := &ProfileSection{
		CoreClkBoost: &coreBoost,
		VFCurve:      vfCurveBytes,
	}

	mode := ps.GetOffsetMode()
	if mode != OffsetModeFixedOffset {
		t.Errorf("Expected OffsetModeFixedOffset, got %v (%d)", mode, mode)
	}

	// Verify GetFixedOffset returns the correct value
	offset, ok := ps.GetFixedOffset()
	if !ok {
		t.Error("GetFixedOffset returned false for fixed offset mode")
	}
	if offset != 123 {
		t.Errorf("Expected fixed offset 123 MHz, got %d MHz", offset)
	}
}

// TestGetOffsetMode_CustomCurve tests detection of custom curve mode (curve editor)
func TestGetOffsetMode_CustomCurve(t *testing.T) {
	// Create a V-F curve with varying offsets (like Profile1)
	curve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 3,
		Points: []VFPoint{
			{VoltageMV: 800.0, BaseFreqMHz: 700.0, OffsetMHz: 952.0, IsActive: true},
			{VoltageMV: 900.0, BaseFreqMHz: 800.0, OffsetMHz: 500.0, IsActive: true},
			{VoltageMV: 1000.0, BaseFreqMHz: 900.0, OffsetMHz: -100.0, IsActive: true},
		},
	}

	hexData, err := curve.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal curve: %v", err)
	}

	vfCurveBytes, err := hex.DecodeString(hexData)
	if err != nil {
		t.Fatalf("Failed to decode hex: %v", err)
	}

	// Create profile section with CoreClkBoost = 1000000 kHz (custom curve marker)
	coreBoost := coreClkBoostCustomCurveValue
	ps := &ProfileSection{
		CoreClkBoost: &coreBoost,
		VFCurve:      vfCurveBytes,
	}

	mode := ps.GetOffsetMode()
	if mode != OffsetModeCustomCurve {
		t.Errorf("Expected OffsetModeCustomCurve, got %v (%d)", mode, mode)
	}

	// Verify GetFixedOffset returns false for custom curve
	_, ok := ps.GetFixedOffset()
	if ok {
		t.Error("GetFixedOffset returned true for custom curve mode")
	}
}

// TestGetOffsetMode_FixedOffset_1000MHz tests the critical edge case where
// a user sets a fixed offset of +1000 MHz via the slider. This produces
// CoreClkBoost = 1000000 kHz (same as custom curve marker), but the V-F curve
// has uniform offsets, so it should be detected as FixedOffset mode.
func TestGetOffsetMode_FixedOffset_1000MHz(t *testing.T) {
	// Create a V-F curve with uniform +1000 MHz offset (slider at maximum)
	curve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 3,
		Points: []VFPoint{
			{VoltageMV: 800.0, BaseFreqMHz: 1675.0, OffsetMHz: 1000.0, IsActive: true},
			{VoltageMV: 900.0, BaseFreqMHz: 1775.0, OffsetMHz: 1000.0, IsActive: true},
			{VoltageMV: 1000.0, BaseFreqMHz: 1875.0, OffsetMHz: 1000.0, IsActive: true},
		},
	}

	hexData, err := curve.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal curve: %v", err)
	}

	vfCurveBytes, err := hex.DecodeString(hexData)
	if err != nil {
		t.Fatalf("Failed to decode hex: %v", err)
	}

	// CoreClkBoost = 1000000 kHz (= +1000 MHz, same as custom curve marker)
	coreBoost := coreClkBoostCustomCurveValue
	ps := &ProfileSection{
		CoreClkBoost: &coreBoost,
		VFCurve:      vfCurveBytes,
	}

	// This is the critical test: should detect FixedOffset, NOT CustomCurve
	mode := ps.GetOffsetMode()
	if mode != OffsetModeFixedOffset {
		t.Errorf("Expected OffsetModeFixedOffset for +1000 MHz uniform offset, got %v (%d)", mode, mode)
	}

	// Verify GetFixedOffset returns the correct value
	offset, ok := ps.GetFixedOffset()
	if !ok {
		t.Error("GetFixedOffset returned false for +1000 MHz fixed offset")
	}
	if offset != 1000 {
		t.Errorf("Expected fixed offset 1000 MHz, got %d MHz", offset)
	}
}

// TestGetOffsetMode_VaryingOffsets tests detection of custom curve mode
// even when CoreClkBoost suggests a fixed offset value.
// This verifies that the V-F curve is the source of truth.
func TestGetOffsetMode_VaryingOffsets(t *testing.T) {
	// Create a V-F curve with varying offsets (custom curve)
	curve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 3,
		Points: []VFPoint{
			{VoltageMV: 800.0, BaseFreqMHz: 700.0, OffsetMHz: 500.0, IsActive: true},
			{VoltageMV: 900.0, BaseFreqMHz: 800.0, OffsetMHz: 200.0, IsActive: true},
			{VoltageMV: 1000.0, BaseFreqMHz: 900.0, OffsetMHz: -50.0, IsActive: true},
		},
	}

	hexData, err := curve.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal curve: %v", err)
	}

	vfCurveBytes, err := hex.DecodeString(hexData)
	if err != nil {
		t.Fatalf("Failed to decode hex: %v", err)
	}

	// Create profile section with CoreClkBoost suggesting fixed offset (123 MHz)
	// but V-F curve has varying offsets - V-F curve wins, it's CustomCurve
	coreBoost := 123000
	ps := &ProfileSection{
		CoreClkBoost: &coreBoost,
		VFCurve:      vfCurveBytes,
	}

	mode := ps.GetOffsetMode()
	if mode != OffsetModeCustomCurve {
		t.Errorf("Expected OffsetModeCustomCurve (V-F curve is source of truth), got %v (%d)", mode, mode)
	}
}

// TestGetOffsetMode_Invalid_ParseError tests detection of invalid state when
// V-F curve cannot be parsed (corrupt or invalid binary data)
func TestGetOffsetMode_Invalid_ParseError(t *testing.T) {
	coreBoost := 123000
	ps := &ProfileSection{
		CoreClkBoost: &coreBoost,
		VFCurve:      []byte{0xFF, 0xFF, 0xFF, 0xFF}, // Invalid/corrupt V-F curve data
	}

	mode := ps.GetOffsetMode()
	if mode != OffsetModeInvalid {
		t.Errorf("Expected OffsetModeInvalid for unparseable V-F curve, got %v (%d)", mode, mode)
	}
}

// TestGetOffsetMode_Unknown_MissingCoreClkBoost tests unknown mode when CoreClkBoost is nil
func TestGetOffsetMode_Unknown_MissingCoreClkBoost(t *testing.T) {
	// Profile without CoreClkBoost field
	ps := &ProfileSection{
		VFCurve: []byte{0x00, 0x01}, // Some dummy data
	}

	mode := ps.GetOffsetMode()
	if mode != OffsetModeUnknown {
		t.Errorf("Expected OffsetModeUnknown, got %v (%d)", mode, mode)
	}
}

// TestGetOffsetMode_Unknown_MissingVFCurve tests unknown mode when VFCurve is empty
func TestGetOffsetMode_Unknown_MissingVFCurve(t *testing.T) {
	coreBoost := 123000
	ps := &ProfileSection{
		CoreClkBoost: &coreBoost,
		VFCurve:      nil,
	}

	mode := ps.GetOffsetMode()
	if mode != OffsetModeUnknown {
		t.Errorf("Expected OffsetModeUnknown, got %v (%d)", mode, mode)
	}
}

// TestGetOffsetMode_ZeroOffset tests fixed offset mode with 0 MHz offset (stock)
func TestGetOffsetMode_ZeroOffset(t *testing.T) {
	// Create a V-F curve with 0 MHz offset (stock settings)
	curve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 2,
		Points: []VFPoint{
			{VoltageMV: 800.0, BaseFreqMHz: 700.0, OffsetMHz: 0.0, IsActive: true},
			{VoltageMV: 900.0, BaseFreqMHz: 800.0, OffsetMHz: 0.0, IsActive: true},
		},
	}

	hexData, err := curve.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal curve: %v", err)
	}

	vfCurveBytes, err := hex.DecodeString(hexData)
	if err != nil {
		t.Fatalf("Failed to decode hex: %v", err)
	}

	// CoreClkBoost = 0 kHz (= 0 MHz, stock)
	coreBoost := 0
	ps := &ProfileSection{
		CoreClkBoost: &coreBoost,
		VFCurve:      vfCurveBytes,
	}

	mode := ps.GetOffsetMode()
	if mode != OffsetModeFixedOffset {
		t.Errorf("Expected OffsetModeFixedOffset for zero offset, got %v (%d)", mode, mode)
	}

	offset, ok := ps.GetFixedOffset()
	if !ok {
		t.Error("GetFixedOffset returned false for zero offset")
	}
	if offset != 0 {
		t.Errorf("Expected zero offset, got %d MHz", offset)
	}
}

// TestGetOffsetMode_NegativeOffset tests fixed offset mode with negative offset (undervolt)
func TestGetOffsetMode_NegativeOffset(t *testing.T) {
	// Create a V-F curve with uniform -50 MHz offset
	curve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 2,
		Points: []VFPoint{
			{VoltageMV: 800.0, BaseFreqMHz: 700.0, OffsetMHz: -50.0, IsActive: true},
			{VoltageMV: 900.0, BaseFreqMHz: 800.0, OffsetMHz: -50.0, IsActive: true},
		},
	}

	hexData, err := curve.Marshal()
	if err != nil {
		t.Fatalf("Failed to marshal curve: %v", err)
	}

	vfCurveBytes, err := hex.DecodeString(hexData)
	if err != nil {
		t.Fatalf("Failed to decode hex: %v", err)
	}

	// CoreClkBoost = -50000 kHz (= -50 MHz)
	coreBoost := -50000
	ps := &ProfileSection{
		CoreClkBoost: &coreBoost,
		VFCurve:      vfCurveBytes,
	}

	mode := ps.GetOffsetMode()
	if mode != OffsetModeFixedOffset {
		t.Errorf("Expected OffsetModeFixedOffset for negative offset, got %v (%d)", mode, mode)
	}

	offset, ok := ps.GetFixedOffset()
	if !ok {
		t.Error("GetFixedOffset returned false for negative offset")
	}
	if offset != -50 {
		t.Errorf("Expected -50 MHz offset, got %d MHz", offset)
	}
}

// TestOffsetMode_String tests the String() method for OffsetMode
func TestOffsetMode_String(t *testing.T) {
	tests := []struct {
		mode     OffsetMode
		expected string
	}{
		{OffsetModeFixedOffset, "Fixed Offset (slider mode)"},
		{OffsetModeCustomCurve, "Custom Curve (curve editor)"},
		{OffsetModeInvalid, "Invalid (inconsistent state)"},
		{OffsetModeUnknown, "Unknown"},
		{OffsetMode(999), "Unknown"}, // Unknown value
	}

	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.mode.String()
			if result != test.expected {
				t.Errorf("OffsetMode(%d).String() = %q, want %q", test.mode, result, test.expected)
			}
		})
	}
}

// TestGetFixedOffset_NotFixedOffset tests GetFixedOffset returns false for non-fixed modes
func TestGetFixedOffset_NotFixedOffset(t *testing.T) {
	// Custom curve mode
	coreBoost := coreClkBoostCustomCurveValue
	ps := &ProfileSection{
		CoreClkBoost: &coreBoost,
	}

	offset, ok := ps.GetFixedOffset()
	if ok {
		t.Error("GetFixedOffset should return false for custom curve mode")
	}
	if offset != 0 {
		t.Errorf("Expected 0 offset, got %d", offset)
	}
}

// TestGetFixedOffset_NilCoreClkBoost tests GetFixedOffset with nil CoreClkBoost
func TestGetFixedOffset_NilCoreClkBoost(t *testing.T) {
	ps := &ProfileSection{
		CoreClkBoost: nil,
	}

	offset, ok := ps.GetFixedOffset()
	if ok {
		t.Error("GetFixedOffset should return false when CoreClkBoost is nil")
	}
	if offset != 0 {
		t.Errorf("Expected 0 offset, got %d", offset)
	}
}
