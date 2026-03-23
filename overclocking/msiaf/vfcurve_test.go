// Package msiaf tests the VF Curve parser implementation.
package msiaf

import (
	"testing"
)

// TestVFCurveParsing tests parsing the VF curve from actual RTX 5090 profile data.
// Data extracted from: LocalProfiles/VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg
func TestVFCurveParsing(t *testing.T) {
	hexData := "000002007F000000000000000000E14300006143000000000000E64300006143000000000080E84300006143000000000000EB4300006143000000000080ED4300006143000000000080F24300006143000000000000F54300006143000000000080F74300006143000000000000FA4300006143000000000000FF43000061430000000000C00044000061430000000000000244000061430000000000400344000061430000000000C00544000061430000000000000744000061430000000000400844000061430000000000800944000061430000000000000C44000061430000000000400D44000061430000000000800E44000061430000000000C00F44000061430000000000401244000061430000000000801344000061430000000000C01444000061430000000000001644000061430000000000801844000061430000000000C01944000061430000000000001B44000061430000000000401C44000061430000000000C01E44000061430000000000002044000061430000000000402144000061430000000000802244000061430000000000002544000061430000000000402644000061430000000000802744000061430000000000C02844000061430000000000402B44000061430000000000802C44000061430000000000C02D44000061430000000000002F44000061430000000000803144000061430000000000C03244000061430000000000003444000061430000000000403544000061430000000000C03744000061430000000000003944000061430000000000403A44000061430000000000803B44000061430000000000003E44000061430000000000403F44000061430000000000804044000077430000000000C041440080B743000000000040444400403D44000000000080454400608F440000000000C046440000FB43006081440000484400C0284400006E4400804A4400403D4400406C4400C04B4400C0534400406C4400004D440080684400006E4400404E4400007F4400006E4400C0504400C0894400406C44000052440000954400406C440040534400609F4400006E440080544400A0AA4400006E440000574400E0B44400406C44004058440020C04400406C44008059440080CA4400006E4400C05A4400C0D44400406C4400405D440000E04400406C4400805E440060EA4400406C4400C05F440080F544000063440000614400A0FF4400C04E4400806344007005450040384400C0644400200B4500802144000066440040104500000D440040674400E015450000ED4300C0694400101B450080C34300006B4400B020450080964300406C4400F023450000794300806D4400602545000062430000704400C0264500004C430040714400802845000030430080724400D0294500001B4300C0734400602B45000002430040764400D02C450000D6420080774400302E450000AA4200C0784400202F4500008C4200007A4400D0304500002C4200807C44001033450000E04000C07D4400703445000070C100007F4400E03545000018C20020804400D03645000054C20060814400403745000070C20000824400303845000096C200A08244002039450000B4C20040834400C038450000A8C20080844400303D4500001BC30020854400203E4500002AC300C0854400903E45000031C30060864400803F45000040C300A087440070404500004FC30040884400F04045000057C300E0884400E04145000066C3008089440050424500006DC300C08A440040434500007CC300608B4400C04345000082C300008C4400B04445008089C300A08C440020454500008DC300E08D4400104645008094C300808E4400904645008098C300208F44008047450000A0C300C08F4400F047450080A3C300009144007048450080A7C300A09144006049450000AFC30040924400D049450080B2C300E0924400504A450080B6C30020944400404B450000BEC300C0944400B04B450080C1C30060954400304C450080C5C30000964400204D450000CDC30040974400904D450080D0C300E0974400104E450080D4C30080984400804E450000D8C30020994400004F450000DCC300609A4400704F450080DFC300009B44006050450000E7C30000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"

	// Parse the VF curve
	curve, err := UnmarshalVFControlCurve(hexData)
	if err != nil {
		t.Fatalf("Failed to parse VF curve: %v", err)
	}

	// Verify basic properties
	if curve.Version != 0x00020000 {
		t.Errorf("Expected version 0x00020000, got 0x%08X", curve.Version)
	}

	if curve.PointCount != 127 {
		t.Errorf("Expected 127 points, got %d", curve.PointCount)
	}

	if len(curve.Points) != 127 {
		t.Errorf("Expected 127 parsed points, got %d", len(curve.Points))
	}

	// Debug: Print first 10 raw triplets to verify field interpretation
	// f0=voltage_mV, f1=base_MHz (baseline frequency), f2=offset_MHz
	t.Log("Raw triplets (first 10 points - mostly inactive):")
	for i := 0; i < 10 && i < len(curve.RawTriplets); i++ {
		t.Logf("  Point %d: [f0=%.2f, f1=%.2f, f2=%.2f]",
			i, curve.RawTriplets[i][0], curve.RawTriplets[i][1], curve.RawTriplets[i][2])
	}

	// Debug: Find and print active points in the middle voltage range (800-1000 mV)
	// f1 = OC Scanner reference (cached from last scan), f2 = user offset from that reference
	t.Log("Active points in 800-1000 mV range:")
	for i, triplet := range curve.RawTriplets {
		voltage := triplet[0]
		if voltage >= 800 && voltage <= 1000 && triplet[1] != 225.0 {
			t.Logf("  Point %d: voltage=%.0f mV, base=%.0f MHz, Offset=%.0f MHz, Applied≈%.0f MHz",
				i, voltage, triplet[1], triplet[2], triplet[1]+triplet[2])
		}
	}

	// Debug: Show points at specific voltages to understand the pattern
	t.Log("Sample points across voltage range:")
	sampleVoltages := []float32{500, 700, 900, 1000, 1100, 1200}
	for _, sampleV := range sampleVoltages {
		point := curve.GetPointByVoltage(sampleV)
		if point != nil {
			t.Logf("  %.0f mV: base=%.0f MHz, Offset=%+.0f MHz, Active=%v",
				point.VoltageMV, point.BaseFreqMHz, point.OffsetMHz, point.IsActive)
		}
	}

	// Count active vs inactive points
	activeCount := 0
	inactiveCount := 0
	for _, p := range curve.Points {
		if p.IsActive {
			activeCount++
		} else {
			inactiveCount++
		}
	}

	t.Logf("Active points: %d, Inactive points: %d", activeCount, inactiveCount)

	// Verify voltage range
	minVolt := curve.Points[0].VoltageMV
	maxVolt := curve.Points[0].VoltageMV
	for _, p := range curve.Points {
		if p.VoltageMV < minVolt {
			minVolt = p.VoltageMV
		}
		if p.VoltageMV > maxVolt {
			maxVolt = p.VoltageMV
		}
	}

	t.Logf("Voltage range: %.0f-%.0f mV", minVolt, maxVolt)

	// Verify offset range
	minOffset := curve.Points[0].OffsetMHz
	maxOffset := curve.Points[0].OffsetMHz
	for _, p := range curve.Points {
		if p.OffsetMHz < minOffset {
			minOffset = p.OffsetMHz
		}
		if p.OffsetMHz > maxOffset {
			maxOffset = p.OffsetMHz
		}
	}

	t.Logf("Offset range: %+.0f ... %+.0f MHz", minOffset, maxOffset)

	// Validate the curve
	if err := curve.Validate(); err != nil {
		t.Errorf("Curve validation failed: %v", err)
	}

	// Test GetPointByVoltage
	point := curve.GetPointByVoltage(800)
	if point == nil {
		t.Error("GetPointByVoltage(800) returned nil")
	} else {
		t.Logf("Point at ~800 mV: offset=%+.0f MHz, base=%.0f MHz, active=%v",
			point.OffsetMHz, point.BaseFreqMHz, point.IsActive)
	}

	// Test GetBaseFreqAt
	baseFreq := curve.GetBaseFreqAt(800)
	if baseFreq == 0 {
		t.Error("GetBaseFreqAt(800) returned 0")
	} else {
		t.Logf("Base frequency at ~800 mV: %.0f MHz", baseFreq)
	}

	// Test GetOffsetAt
	offset := curve.GetOffsetAt(800)
	if offset == 0 {
		t.Error("GetOffsetAt(800) returned 0")
	} else {
		t.Logf("Offset at ~800 mV: %+.0f MHz", offset)
	}
}

// TestVFCurveInactiveMarker tests that inactive points have 225.0 OC Scanner reference.
// Inactive points indicate voltages where OC Scanner has no cached data (GPU unstable during scan).
func TestVFCurveInactiveMarker(t *testing.T) {
	// Create a test curve with known inactive points
	// Structure: [voltage_mV, oc_ref_MHz (OC Scanner cached ref), offset_MHz]
	triplets := [][3]float32{
		{500.0, 225.0, 0.0},   // inactive (no OC Scanner data at this voltage)
		{600.0, 750.0, 100.0}, // active (OC Scanner ref=750 MHz, user offset=+100 MHz)
		{700.0, 225.0, 0.0},   // inactive (no OC Scanner data at this voltage)
		{800.0, 850.0, -50.0}, // active (OC Scanner ref=850 MHz, user offset=-50 MHz)
	}

	points := buildVFPointsFromTriplets(triplets)

	if len(points) != 4 {
		t.Fatalf("Expected 4 points, got %d", len(points))
	}

	// Verify inactive points
	if points[0].IsActive {
		t.Error("Point 0 should be inactive (BaseFreqMHz=225.0)")
	}
	if points[0].OffsetMHz != 0.0 {
		t.Errorf("Inactive point should have offset 0, got %.0f", points[0].OffsetMHz)
	}

	if points[2].IsActive {
		t.Error("Point 2 should be inactive (BaseFreqMHz=225.0)")
	}

	// Verify active points
	if !points[1].IsActive {
		t.Error("Point 1 should be active (BaseFreqMHz=750.0)")
	}
	if !points[3].IsActive {
		t.Error("Point 3 should be active (BaseFreqMHz=850.0)")
	}
}

// TestVFCurveHelperMethods tests the GetBaseFreqAt and GetOffsetAt helper methods.
// Base frequency serves as the baseline for offset calculations.
func TestVFCurveHelperMethods(t *testing.T) {
	// Create a test curve with known values
	// Structure: [voltage_mV, base_MHz (baseline frequency), offset_MHz (user delta)]
	triplets := [][3]float32{
		{500.0, 225.0, 0.0},   // inactive: stock behavior, offset=0
		{600.0, 750.0, 100.0}, // active: base=750 MHz, applied≈850 MHz
		{800.0, 850.0, -50.0}, // active: base=850 MHz, applied≈800 MHz
	}

	points := buildVFPointsFromTriplets(triplets)
	curve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: uint32(len(points)),
		Points:     points,
	}

	// Test GetBaseFreqAt
	t.Run("GetBaseFreqAt", func(t *testing.T) {
		// Test inactive point
		baseFreq := curve.GetBaseFreqAt(500)
		if baseFreq != 225.0 {
			t.Errorf("GetBaseFreqAt(500) = %.1f, want 225.0", baseFreq)
		}

		// Test active point
		baseFreq = curve.GetBaseFreqAt(600)
		if baseFreq != 750.0 {
			t.Errorf("GetBaseFreqAt(600) = %.1f, want 750.0", baseFreq)
		}

		// Test another active point
		baseFreq = curve.GetBaseFreqAt(800)
		if baseFreq != 850.0 {
			t.Errorf("GetBaseFreqAt(800) = %.1f, want 850.0", baseFreq)
		}

		// Test voltage with no exact match (should return closest)
		baseFreq = curve.GetBaseFreqAt(510)
		if baseFreq == 0 {
			t.Error("GetBaseFreqAt(510) returned 0 for close voltage")
		}
	})

	// Test GetOffsetAt
	t.Run("GetOffsetAt", func(t *testing.T) {
		// Test inactive point (offset should be 0)
		offset := curve.GetOffsetAt(500)
		if offset != 0.0 {
			t.Errorf("GetOffsetAt(500) = %.1f, want 0.0", offset)
		}

		// Test active point with positive offset
		offset = curve.GetOffsetAt(600)
		if offset != 100.0 {
			t.Errorf("GetOffsetAt(600) = %.1f, want 100.0", offset)
		}

		// Test active point with negative offset
		offset = curve.GetOffsetAt(800)
		if offset != -50.0 {
			t.Errorf("GetOffsetAt(800) = %.1f, want -50.0", offset)
		}
	})
}

// TestVFCurveValidation tests validation of VF curve data.
func TestVFCurveValidation(t *testing.T) {
	tests := []struct {
		name        string
		version     uint32
		points      []VFPoint
		shouldError bool
	}{
		{
			name:    "valid curve",
			version: 0x00020000,
			points: []VFPoint{
				{Index: 0, VoltageMV: 500, OffsetMHz: 0, BaseFreqMHz: 225.0, IsActive: false},
				{Index: 1, VoltageMV: 600, OffsetMHz: 100, BaseFreqMHz: 750.0, IsActive: true},
			},
			shouldError: false,
		},
		{
			name:    "voltage out of range",
			version: 0x00020000,
			points: []VFPoint{
				{Index: 0, VoltageMV: 300, OffsetMHz: 0, BaseFreqMHz: 225.0, IsActive: false},
			},
			shouldError: true,
		},
		{
			name:    "active point OC Scanner ref out of range",
			version: 0x00020000,
			points: []VFPoint{
				{Index: 0, VoltageMV: 800, OffsetMHz: 100, BaseFreqMHz: 500.0, IsActive: true},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			curve := &VFControlCurveInfo{
				Version:    tt.version,
				PointCount: uint32(len(tt.points)),
				Points:     tt.points,
			}

			err := curve.Validate()
			if tt.shouldError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

// TestVFControlCurveInfo_Marshal tests marshaling a VF curve to hex format.
func TestVFControlCurveInfo_Marshal(t *testing.T) {
	// Create a simple curve with known values
	curve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 3,
		Points: []VFPoint{
			{
				Index:       0,
				VoltageMV:   800,
				BaseFreqMHz: 2500,
				OffsetMHz:   100,
				IsActive:    true,
			},
			{
				Index:       1,
				VoltageMV:   900,
				BaseFreqMHz: 2700,
				OffsetMHz:   50,
				IsActive:    true,
			},
			{
				Index:       2,
				VoltageMV:   1000,
				BaseFreqMHz: 225.0, // Inactive
				OffsetMHz:   0,
				IsActive:    false,
			},
		},
	}

	hexData, err := curve.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Verify hex data is not empty
	if len(hexData) == 0 {
		t.Error("Marshal returned empty hex data")
	}

	// Verify hex data length: 12-byte header + 3 points * 12 bytes = 48 bytes = 96 hex chars
	expectedLen := 12 + (3 * 12)
	expectedHexLen := expectedLen * 2
	if len(hexData) != expectedHexLen {
		t.Errorf("Expected hex length %d, got %d", expectedHexLen, len(hexData))
	}

	// Verify no prefix or suffix
	if len(hexData) > 2 && (hexData[:2] == "0x" || hexData[len(hexData)-1:] == "h") {
		t.Error("Marshal should not include 0x prefix or h suffix")
	}
}

// TestVFControlCurveInfo_Marshal_RoundTrip tests that Marshal + Unmarshal produces identical data.
func TestVFControlCurveInfo_Marshal_RoundTrip(t *testing.T) {
	// Create original curve
	original := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 4,
		Points: []VFPoint{
			{
				Index:       0,
				VoltageMV:   800,
				BaseFreqMHz: 2500,
				OffsetMHz:   100,
				IsActive:    true,
			},
			{
				Index:       1,
				VoltageMV:   850,
				BaseFreqMHz: 2600,
				OffsetMHz:   75,
				IsActive:    true,
			},
			{
				Index:       2,
				VoltageMV:   900,
				BaseFreqMHz: 225.0,
				OffsetMHz:   0,
				IsActive:    false,
			},
			{
				Index:       3,
				VoltageMV:   950,
				BaseFreqMHz: 2750,
				OffsetMHz:   -50,
				IsActive:    true,
			},
		},
	}

	// Marshal to hex
	hexData, err := original.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal back
	roundTrip, err := UnmarshalVFControlCurve(hexData)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify round-trip data matches original
	if roundTrip.Version != original.Version {
		t.Errorf("Version mismatch: %d vs %d", roundTrip.Version, original.Version)
	}

	if len(roundTrip.Points) != len(original.Points) {
		t.Errorf("Point count mismatch: %d vs %d", len(roundTrip.Points), len(original.Points))
	}

	for i := range original.Points {
		orig := original.Points[i]
		rt := roundTrip.Points[i]

		if orig.VoltageMV != rt.VoltageMV {
			t.Errorf("Point %d voltage mismatch: %.1f vs %.1f", i, orig.VoltageMV, rt.VoltageMV)
		}

		if orig.OffsetMHz != rt.OffsetMHz {
			t.Errorf("Point %d offset mismatch: %.1f vs %.1f", i, orig.OffsetMHz, rt.OffsetMHz)
		}

		if orig.IsActive != rt.IsActive {
			t.Errorf("Point %d IsActive mismatch: %v vs %v", i, orig.IsActive, rt.IsActive)
		}

		// For active points, BaseFreqMHz should match
		if orig.IsActive && orig.BaseFreqMHz != rt.BaseFreqMHz {
			t.Errorf("Point %d BaseFreqMHz mismatch: %.1f vs %.1f", i, orig.BaseFreqMHz, rt.BaseFreqMHz)
		}

		// For inactive points, BaseFreqMHz should be 225.0
		if !orig.IsActive && rt.BaseFreqMHz != 225.0 {
			t.Errorf("Point %d inactive marker mismatch: %.1f vs 225.0", i, rt.BaseFreqMHz)
		}
	}
}

// TestVFControlCurveInfo_Marshal_Errors tests error cases for Marshal.
func TestVFControlCurveInfo_Marshal_Errors(t *testing.T) {
	// Test unsupported version
	badVersion := &VFControlCurveInfo{
		Version:    0x00010000,
		PointCount: 1,
		Points: []VFPoint{
			{Index: 0, VoltageMV: 800, BaseFreqMHz: 2500, OffsetMHz: 100, IsActive: true},
		},
	}
	_, err := badVersion.Marshal()
	if err == nil {
		t.Error("Expected error for unsupported version")
	}

	// Test no points
	noPoints := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 0,
		Points:     []VFPoint{},
	}
	_, err = noPoints.Marshal()
	if err == nil {
		t.Error("Expected error for curve with no points")
	}

	// Test too many points
	tooMany := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: VFControlCurveMaxPoints + 1,
		Points:     make([]VFPoint, VFControlCurveMaxPoints+1),
	}
	_, err = tooMany.Marshal()
	if err == nil {
		t.Error("Expected error for too many points")
	}
}

// TestVFControlCurveInfo_ApplyFlatOffset tests applying flat offset to all active points.
func TestVFControlCurveInfo_ApplyFlatOffset(t *testing.T) {
	curve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 4,
		Points: []VFPoint{
			{
				Index:       0,
				VoltageMV:   800,
				BaseFreqMHz: 2500,
				OffsetMHz:   100,
				IsActive:    true,
			},
			{
				Index:       1,
				VoltageMV:   850,
				BaseFreqMHz: 225.0,
				OffsetMHz:   0,
				IsActive:    false,
			},
			{
				Index:       2,
				VoltageMV:   900,
				BaseFreqMHz: 2700,
				OffsetMHz:   50,
				IsActive:    true,
			},
			{
				Index:       3,
				VoltageMV:   950,
				BaseFreqMHz: 2800,
				OffsetMHz:   -25,
				IsActive:    true,
			},
		},
	}

	// Apply +100 MHz offset
	curve.ApplyFlatOffset(100)

	// Verify active points were modified
	if curve.Points[0].OffsetMHz != 200 {
		t.Errorf("Point 0 offset should be 200, got %.1f", curve.Points[0].OffsetMHz)
	}

	// Inactive point should not be modified
	if curve.Points[1].OffsetMHz != 0 {
		t.Errorf("Inactive point 1 offset should remain 0, got %.1f", curve.Points[1].OffsetMHz)
	}

	if curve.Points[2].OffsetMHz != 150 {
		t.Errorf("Point 2 offset should be 150, got %.1f", curve.Points[2].OffsetMHz)
	}

	if curve.Points[3].OffsetMHz != 75 {
		t.Errorf("Point 3 offset should be 75, got %.1f", curve.Points[3].OffsetMHz)
	}
}

// TestVFControlCurveInfo_ApplyFlatOffset_Negative tests applying negative offset (undervolt).
func TestVFControlCurveInfo_ApplyFlatOffset_Negative(t *testing.T) {
	curve := &VFControlCurveInfo{
		Version:    VFControlCurveVersion2,
		PointCount: 2,
		Points: []VFPoint{
			{
				Index:       0,
				VoltageMV:   800,
				BaseFreqMHz: 2500,
				OffsetMHz:   100,
				IsActive:    true,
			},
			{
				Index:       1,
				VoltageMV:   900,
				BaseFreqMHz: 2700,
				OffsetMHz:   50,
				IsActive:    true,
			},
		},
	}

	// Apply -50 MHz offset (undervolt)
	curve.ApplyFlatOffset(-50)

	if curve.Points[0].OffsetMHz != 50 {
		t.Errorf("Point 0 offset should be 50, got %.1f", curve.Points[0].OffsetMHz)
	}

	if curve.Points[1].OffsetMHz != 0 {
		t.Errorf("Point 1 offset should be 0, got %.1f", curve.Points[1].OffsetMHz)
	}
}
