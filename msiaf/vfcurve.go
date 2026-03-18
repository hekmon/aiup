// Package msiaf provides tooling for working with MSI Afterburner profiles and configuration.
package msiaf

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
)

// ============================================================================
// V-F Curve Binary Format Specification
// ============================================================================
//
// Supported Version: 2.0 (0x00020000) only
//
// Binary Layout:
//   ┌─────────────────────────────────────────────────────────────────────┐
//   │ HEADER (12 bytes = 3 fields)                                        │
//   │ [version:uint32] [count:uint32] [reserved:float32=0.0]             │
//   ├─────────────────────────────────────────────────────────────────────┤
//   │ TRIPLETS (12 bytes each = 3 fields)                                 │
//   │ [voltage:float32] [oc_ref:float32] [offset:float32]  per point     │
//   └─────────────────────────────────────────────────────────────────────┘
//
// Header Fields:
//   - Bytes 0-3:  version (uint32 LE) - must be 0x00020000
//   - Bytes 4-7:  point_count (uint32 LE) - typically 127
//   - Bytes 8-11: reserved (float32 LE) - always 0.0 (unused)
//
// Triplet Fields (per control point):
//   - Bytes 0-3:  voltage (float32 LE) - voltage in mV (450-1240 mV range)
//   - Bytes 4-7:  oc_ref (float32 LE) - OC Scanner reference frequency
//   - Bytes 8-11: offset (float32 LE) - frequency offset in MHz
//
// Field Ranges:
//   - voltage:  450-1240 mV
//   - oc_ref:   675-3334 MHz (active points), 225.0 (inactive marker)
//   - offset:   -500 to +1100 MHz (user's delta from hardware boost)
//
// Special Values:
//   - oc_ref = 225.0: Inactive point (stock behavior, offset ignored)
//   - reserved = 0.0: Always zero (unused header field)
//   - All triplet fields = 0.0: Padding after declared point count
//
// Three Curves in VF Editor:
//   1. White dots (applied frequency): HardwareBoost(v) + offset
//   2. Grey diagonal line: oc_ref (visual reference only)
//   3. Hardware boost (implicit): AppliedFreq - offset (driver-private)
//
// Flat Curve Undervolt Example (target 2875 MHz):
//   Voltage   | Hardware Boost | Offset  | Applied Freq
//   800 mV    | ~1840 MHz      | +1035   | 1840+1035 = 2875 MHz
//   950 mV    | ~2575 MHz      | +300    | 2575+300 = 2875 MHz
//   1015 mV   | ~2875 MHz      | 0       | 2875+0 = 2875 MHz (crossover)
//   1240 mV   | ~3322 MHz      | -447    | 3322-447 = 2875 MHz
//
// DESIGN PRINCIPLE: Authoritative Data Only
//   - We ONLY expose data extracted from the binary blob
//   - We DO NOT estimate frequencies (requires driver-private hardware boost)
//   - Users needing actual frequencies must use runtime tools (nvidia-smi, NVML)
// ============================================================================

// VFControlCurveVersion2 is version 2.0 of the VF curve format (current, only supported version).
const VFControlCurveVersion2 = 0x00020000

// VFControlCurveMaxPoints is the maximum number of control points (127).
const VFControlCurveMaxPoints = 127

// VFControlCurveTripletSize is the size of each control point triplet in bytes (12).
// Each triplet contains [voltage:float32][oc_ref:float32][offset:float32].
const VFControlCurveTripletSize = 12

// VFControlCurveHeaderSize is the size of the header in bytes (12).
// Header structure: [version:uint32][count:uint32][reserved:float32].
const VFControlCurveHeaderSize = 12

// VFControlCurveInactiveMarker is the f2 value for inactive points.
// Points with this value have no user override (offset = 0, stock behavior).
const VFControlCurveInactiveMarker = 225.0

// VFControlCurveMinVoltageMV is the minimum voltage in the curve range.
const VFControlCurveMinVoltageMV = 450

// VFControlCurveMaxVoltageMV is the maximum voltage in the curve range.
const VFControlCurveMaxVoltageMV = 1240

// VFControlCurveMinOCScannerRef is the minimum OC Scanner reference frequency.
const VFControlCurveMinOCScannerRef = 675.0

// VFControlCurveMaxOCScannerRef is the maximum OC Scanner reference frequency.
const VFControlCurveMaxOCScannerRef = 3334.0

// VFPoint represents a single control point on the VF curve.
//
// Each point defines a frequency offset at a specific voltage.
// The actual frequency is: StockBoostCurve(voltage) + OffsetMHz
type VFPoint struct {
	// Index is the point position in the curve (0-based).
	Index int

	// VoltageMV is the voltage in millivolts for this point.
	// Range: 450-1240 mV
	VoltageMV float32

	// OffsetMHz is the frequency offset in MHz from the stock boost curve.
	// Positive = overclock, negative = undervolt, 0 = stock
	OffsetMHz float32

	// OCScannerRefMHz is the OC Scanner reference frequency (f2 field).
	// This is the grey diagonal background line in the UI.
	// Range: 675-3334 MHz. Does not affect actual frequencies.
	OCScannerRefMHz float32

	// IsActive indicates whether this point has a user-defined offset.
	// Inactive points (OCScannerRef = 225.0) use stock behavior (offset = 0).
	// Active points have custom offsets applied.
	IsActive bool
}

// VFControlCurveInfo contains the parsed VF curve data.
type VFControlCurveInfo struct {
	// Version is the binary format version (e.g., 0x00020000 for v2.0).
	Version uint32

	// PointCount is the number of control points declared in the header.
	PointCount uint32

	// Points contains all parsed VF curve points.
	// Length will be min(PointCount, VFControlCurveMaxPoints).
	Points []VFPoint

	// RawTriplets contains the raw parsed triplet values for debugging.
	// Each triplet is [voltage_mV, oc_ref_MHz, offset_MHz] for the current point.
	RawTriplets [][3]float32
}

// VFControlCurveError represents an error during VF curve parsing.
type VFControlCurveError struct {
	Field   string
	Message string
}

func (e *VFControlCurveError) Error() string {
	return fmt.Sprintf("VF curve error in %s: %s", e.Field, e.Message)
}

// UnmarshalVFControlCurve parses a hex-encoded VF curve blob.
//
// The hexData should be the raw hex string from the VFCurve= line
// in an MSI Afterburner .cfg profile file (no "0x" prefix).
//
// See the full binary format specification in the package-level comment above.
//
// Returns VFControlCurveInfo with parsed data, or an error if parsing fails.
func UnmarshalVFControlCurve(hexData string) (*VFControlCurveInfo, error) {
	// Decode hex to binary
	binaryData, err := hex.DecodeString(hexData)
	if err != nil {
		return nil, &VFControlCurveError{
			Field:   "hexData",
			Message: fmt.Sprintf("failed to decode hex: %v", err),
		}
	}

	if len(binaryData) < 8 {
		return nil, &VFControlCurveError{
			Field:   "binaryData",
			Message: fmt.Sprintf("data too short: %d bytes, need at least 8 for header", len(binaryData)),
		}
	}

	// Parse 8-byte header
	version := binary.LittleEndian.Uint32(binaryData[0:4])
	pointCount := binary.LittleEndian.Uint32(binaryData[4:8])

	result := &VFControlCurveInfo{
		Version:     version,
		PointCount:  pointCount,
		RawTriplets: make([][3]float32, 0),
	}

	// Skip 12-byte header (version + count + reserved), then read triplets
	dataOffset := VFControlCurveHeaderSize // skip header

	maxPoints := min(int(pointCount), VFControlCurveMaxPoints)

	for i := 0; i < maxPoints && (dataOffset+VFControlCurveTripletSize) <= len(binaryData); i++ {
		// Read triplet: [voltage, oc_ref, offset] for current point
		voltage := math.Float32frombits(binary.LittleEndian.Uint32(binaryData[dataOffset : dataOffset+4]))
		ocScannerRef := math.Float32frombits(binary.LittleEndian.Uint32(binaryData[dataOffset+4 : dataOffset+8]))
		offset := math.Float32frombits(binary.LittleEndian.Uint32(binaryData[dataOffset+8 : dataOffset+12]))

		result.RawTriplets = append(result.RawTriplets, [3]float32{voltage, ocScannerRef, offset})

		dataOffset += VFControlCurveTripletSize
	}

	// Build VF points from triplets
	result.Points = buildVFPointsFromTriplets(result.RawTriplets)

	return result, nil
}

// buildVFPointsFromTriplets converts raw triplets to VFPoint slice.
// Each triplet contains [voltage_mV, oc_ref_MHz, offset_MHz] for the current point.
func buildVFPointsFromTriplets(triplets [][3]float32) []VFPoint {
	points := make([]VFPoint, 0, len(triplets))

	for i, t := range triplets {
		voltage := t[0]
		ocScannerRef := t[1]
		offset := t[2]

		// Determine if point is active based on OC Scanner reference
		// 225.0 = inactive (stock), 675-3334 = active (custom offset)
		isActive := isPointActive(ocScannerRef)

		point := VFPoint{
			Index:           i,
			VoltageMV:       voltage,
			OffsetMHz:       offset,
			OCScannerRefMHz: ocScannerRef,
			IsActive:        isActive,
		}

		points = append(points, point)
	}

	return points
}

// isPointActive determines if a point has a user-defined override.
// Inactive points (OCScannerRef = 225.0) use stock behavior (offset = 0).
// Active points (OCScannerRef = 675-3334) have custom offsets.
func isPointActive(ocScannerRef float32) bool {
	// Check for inactive marker (with small tolerance for float precision)
	diff := ocScannerRef - VFControlCurveInactiveMarker
	if diff < 0 {
		diff = -diff
	}
	if diff < 1.0 {
		return false
	}

	// Active points have OC Scanner reference in 675-3334 range
	return ocScannerRef >= VFControlCurveMinOCScannerRef &&
		ocScannerRef <= VFControlCurveMaxOCScannerRef
}

// GetPoint returns the VF point at a specific index.
// Returns nil if index is out of range.
func (c *VFControlCurveInfo) GetPoint(index int) *VFPoint {
	if index < 0 || index >= len(c.Points) {
		return nil
	}
	return &c.Points[index]
}

// GetPointByVoltage returns the VF point closest to the specified voltage.
// Returns nil if no points exist.
func (c *VFControlCurveInfo) GetPointByVoltage(voltageMV float32) *VFPoint {
	if len(c.Points) == 0 {
		return nil
	}

	bestIdx := 0
	bestDiff := absFloat(c.Points[0].VoltageMV - voltageMV)

	for i := 1; i < len(c.Points); i++ {
		diff := absFloat(c.Points[i].VoltageMV - voltageMV)
		if diff < bestDiff {
			bestDiff = diff
			bestIdx = i
		}
	}

	return &c.Points[bestIdx]
}

// SetOffset sets the frequency offset for a point at the specified voltage.
// This automatically activates the point if it was inactive.
// Returns an error if no point exists near the specified voltage.
func (c *VFControlCurveInfo) SetOffset(voltageMV float32, offsetMHz float32) error {
	point := c.GetPointByVoltage(voltageMV)
	if point == nil {
		return &VFControlCurveError{
			Field:   "voltageMV",
			Message: fmt.Sprintf("no point found near %.1f mV", voltageMV),
		}
	}

	point.OffsetMHz = offsetMHz
	point.IsActive = true
	point.OCScannerRefMHz = VFControlCurveMinOCScannerRef // Mark as active

	return nil
}

// GetActivePoints returns all points with user-defined overrides.
func (c *VFControlCurveInfo) GetActivePoints() []VFPoint {
	active := make([]VFPoint, 0)
	for _, p := range c.Points {
		if p.IsActive {
			active = append(active, p)
		}
	}
	return active
}

// GetInactivePoints returns all points using stock behavior.
func (c *VFControlCurveInfo) GetInactivePoints() []VFPoint {
	inactive := make([]VFPoint, 0)
	for _, p := range c.Points {
		if !p.IsActive {
			inactive = append(inactive, p)
		}
	}
	return inactive
}

// GetMaxOffset returns the maximum frequency offset in the curve.
func (c *VFControlCurveInfo) GetMaxOffset() float32 {
	if len(c.Points) == 0 {
		return 0.0
	}

	maxOffset := c.Points[0].OffsetMHz
	for _, p := range c.Points {
		if p.OffsetMHz > maxOffset {
			maxOffset = p.OffsetMHz
		}
	}
	return maxOffset
}

// GetMinOffset returns the minimum frequency offset in the curve.
func (c *VFControlCurveInfo) GetMinOffset() float32 {
	if len(c.Points) == 0 {
		return 0.0
	}

	minOffset := c.Points[0].OffsetMHz
	for _, p := range c.Points {
		if p.OffsetMHz < minOffset {
			minOffset = p.OffsetMHz
		}
	}
	return minOffset
}

// Validate checks the VF curve for consistency and valid values.
// Returns nil if valid, or an error describing the issue.
// Only version 2.0 (0x00020000) is supported.
func (c *VFControlCurveInfo) Validate() error {
	if c.Version != VFControlCurveVersion2 {
		return &VFControlCurveError{
			Field:   "Version",
			Message: fmt.Sprintf("unsupported version 0x%08X, only 0x%08X is supported", c.Version, VFControlCurveVersion2),
		}
	}

	if c.PointCount == 0 {
		return &VFControlCurveError{
			Field:   "PointCount",
			Message: "point count cannot be zero",
		}
	}

	for i, p := range c.Points {
		if p.VoltageMV < VFControlCurveMinVoltageMV || p.VoltageMV > VFControlCurveMaxVoltageMV {
			return &VFControlCurveError{
				Field: fmt.Sprintf("Points[%d].VoltageMV", i),
				Message: fmt.Sprintf("voltage %.1f mV outside valid range (%d-%d mV)",
					p.VoltageMV, VFControlCurveMinVoltageMV, VFControlCurveMaxVoltageMV),
			}
		}

		if p.IsActive {
			if p.OCScannerRefMHz < VFControlCurveMinOCScannerRef || p.OCScannerRefMHz > VFControlCurveMaxOCScannerRef {
				return &VFControlCurveError{
					Field: fmt.Sprintf("Points[%d].OCScannerRefMHz", i),
					Message: fmt.Sprintf("active point OC Scanner ref %.1f MHz outside valid range (%.0f-%.0f)",
						p.OCScannerRefMHz, VFControlCurveMinOCScannerRef, VFControlCurveMaxOCScannerRef),
				}
			}
		}
	}

	return nil
}

// String returns a human-readable summary of the VF curve.
func (c *VFControlCurveInfo) String() string {
	if len(c.Points) == 0 {
		return "VFControlCurveInfo{empty}"
	}

	activeCount := 0
	inactiveCount := 0
	minOffset := c.Points[0].OffsetMHz
	maxOffset := c.Points[0].OffsetMHz

	for _, p := range c.Points {
		if p.IsActive {
			activeCount++
		} else {
			inactiveCount++
		}
		if p.OffsetMHz < minOffset {
			minOffset = p.OffsetMHz
		}
		if p.OffsetMHz > maxOffset {
			maxOffset = p.OffsetMHz
		}
	}

	return fmt.Sprintf("VFControlCurveInfo{Version:0x%08X, Points:%d, Active:%d, Inactive:%d, OffsetRange:%+.0f...%+.0f MHz}",
		c.Version, len(c.Points), activeCount, inactiveCount, minOffset, maxOffset)
}

// VersionString returns the version in human-readable format (e.g., "1.0", "2.0").
func (c *VFControlCurveInfo) VersionString() string {
	major := c.Version >> 16
	minor := c.Version & 0xFFFF
	return fmt.Sprintf("%d.%d", major, minor)
}

// GetOCScannerReferenceAt returns the OC Scanner reference frequency (f2) at a given voltage.
// This is the grey diagonal background line shown in the VF curve editor UI.
// It is a visual reference cached from a previous OC Scanner run, not a control value.
// Returns 0 if no point exists near the specified voltage.
func (c *VFControlCurveInfo) GetOCScannerReferenceAt(voltageMV float32) float32 {
	point := c.GetPointByVoltage(voltageMV)
	if point == nil {
		return 0.0
	}
	return point.OCScannerRefMHz
}

// GetOffsetAt returns the frequency offset (f0) at a given voltage.
// This is the user-defined offset applied on top of the GPU's natural hardware boost.
// Positive = overclock, negative = undervolt, 0 = stock behavior.
// Returns 0 if no point exists near the specified voltage.
func (c *VFControlCurveInfo) GetOffsetAt(voltageMV float32) float32 {
	point := c.GetPointByVoltage(voltageMV)
	if point == nil {
		return 0.0
	}
	return point.OffsetMHz
}

// absFloat returns the absolute value of a float32.
func absFloat(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}

// Ensure VFControlCurveError implements error interface.
var _ error = (*VFControlCurveError)(nil)
