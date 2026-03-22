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
//   │ [voltage:float32] [base:float32]   [offset:float32]  per point     │
//   └─────────────────────────────────────────────────────────────────────┘
//
// Header Fields:
//   - Bytes 0-3:  version (uint32 LE) - must be 0x00020000
//   - Bytes 4-7:  point_count (uint32 LE) - typically 127
//   - Bytes 8-11: reserved (float32 LE) - always 0.0 (unused)
//
// Triplet Fields (per control point):
//   - Bytes 0-3:  voltage (float32 LE) - voltage in mV (450-1240 mV range)
//   - Bytes 4-7:  base (float32 LE) - base frequency (f1 field)
//   - Bytes 8-11: offset (float32 LE) - frequency offset from base in MHz (f2 field)
//
// Field Ranges:
//   - voltage:  450-1240 mV
//   - base:     675-3334 MHz (active points), 225.0 (inactive marker)
//   - offset:   -500 to +1100 MHz (user's delta from base)
//
// Special Values:
//   - base = 225.0: Inactive point (stock behavior, offset ignored)
//   - reserved = 0.0: Always zero (unused header field)
//   - All triplet fields = 0.0: Padding after declared point count
//
// Three Curves in VF Editor:
//   1. White dots (applied frequency): base(v) + offset
//   2. Grey diagonal line: base frequency (f1 field, possibly from OC Scanner)
//   3. Stock boost curve: Queried live from driver (NOT stored in blob)
//
// Why store base frequency instead of stock curve?
//   - Stock boost is driver-private and can be queried anytime (no need to cache)
//   - Base frequency may come from OC Scanner results or manual curve editing
//   - Users tweak offsets relative to this base frequency
//   - The .cfg blob is self-contained: AppliedFreq = base + offset (no driver query needed)
//
// Flat Curve Undervolt Example (target 2875 MHz):
//   Voltage   | Base Freq      | Offset  | Applied Freq
//   800 mV    | ~1627 MHz*     | +1035   | 1627+1035 = 2662 MHz (*conservative at low voltage)
//   950 mV    | ~2575 MHz      | +300    | 2575+300 = 2875 MHz
//   1015 mV   | ~2875 MHz      | 0       | 2875+0 = 2875 MHz (crossover point)
//   1240 mV   | ~3322 MHz      | -447    | 3322-447 = 2875 MHz
//
// Note: Low voltage points may show conservative base frequencies if the GPU
// was unstable during OC Scanner at those voltages, or if the user manually
// adjusted those points.
//
// DESIGN PRINCIPLE: Authoritative Data Only
//   - We ONLY expose data extracted from the binary blob (voltage, base, offset)
//   - We DO NOT estimate applied frequencies (requires computing base + offset)
//   - The base frequency (f1) may originate from OC Scanner or manual editing
//   - Stock boost curve is queried live from the driver (NOT stored in the blob)
//   - Users needing actual runtime frequencies must use tools like nvidia-smi or NVML
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

// VFControlCurveMinBaseFreq is the minimum base frequency (f1 field).
// Base frequencies below this value (except the inactive marker 225.0) are considered invalid.
const VFControlCurveMinBaseFreq = 675.0

// VFControlCurveMaxBaseFreq is the maximum base frequency (f1 field).
// Base frequencies above this value are considered invalid.
const VFControlCurveMaxBaseFreq = 3334.0

// VFPoint represents a single control point on the VF curve.
//
// Each point defines a frequency offset at a specific voltage.
// The applied frequency is: BaseFreqMHz + OffsetMHz
//
// IMPORTANT: BaseFreqMHz is NOT the stock hardware boost curve.
// The stock curve is queried live from the driver and is NOT stored in the blob.
// BaseFreqMHz is the baseline frequency (f1 field) that the offset is applied to.
// This base frequency may originate from OC Scanner results or manual curve editing.
type VFPoint struct {
	// Index is the point position in the curve (0-based).
	Index int

	// VoltageMV is the voltage in millivolts for this point.
	// Range: 450-1240 mV
	VoltageMV float32

	// OffsetMHz is the frequency offset in MHz from the base frequency (f2 field).
	// Positive = overclock relative to base
	// Negative = undervolt relative to base
	// Zero = use base frequency (no user adjustment)
	OffsetMHz float32

	// BaseFreqMHz is the base frequency (f1 field).
	// This is the grey diagonal background line shown in the VF curve editor UI.
	// It represents the baseline frequency that the offset is applied to.
	// This base frequency may come from OC Scanner results or manual curve editing.
	// Range: 675-3334 MHz for active points, 225.0 for inactive points (stock behavior).
	//
	// Why this is stored: The base frequency curve would be lost if not saved.
	// The stock boost curve is driver-private and can be queried anytime.
	//
	// Applied frequency at this point: BaseFreqMHz + OffsetMHz
	BaseFreqMHz float32

	// IsActive indicates whether this point has been customized.
	// Inactive points (BaseFreqMHz = 225.0) use stock behavior (offset = 0).
	// Active points have a base frequency set and may have user-defined offsets applied.
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
	// Each triplet is [voltage_mV, base_MHz, offset_MHz] for the current point.
	// Applied frequency at each point = base + offset (when base != 225.0)
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
		baseFreq := math.Float32frombits(binary.LittleEndian.Uint32(binaryData[dataOffset+4 : dataOffset+8]))
		offset := math.Float32frombits(binary.LittleEndian.Uint32(binaryData[dataOffset+8 : dataOffset+12]))

		result.RawTriplets = append(result.RawTriplets, [3]float32{voltage, baseFreq, offset})

		dataOffset += VFControlCurveTripletSize
	}

	// Build VF points from triplets
	result.Points = buildVFPointsFromTriplets(result.RawTriplets)

	return result, nil
}

// buildVFPointsFromTriplets converts raw triplets to VFPoint slice.
// Each triplet contains [voltage_mV, base_MHz, offset_MHz] for the current point.
// - voltage_mV: The voltage point (450-1240 mV range)
// - base_MHz: Base frequency (675-3334 MHz for active points, or 225.0 if inactive)
// - offset_MHz: User's frequency offset from base
func buildVFPointsFromTriplets(triplets [][3]float32) []VFPoint {
	points := make([]VFPoint, 0, len(triplets))

	for i, t := range triplets {
		voltage := t[0]
		baseFreq := t[1]
		offset := t[2]

		// Determine if point is active based on base frequency
		// 225.0 = inactive (stock), 675-3334 = active (custom)
		isActive := isPointActive(baseFreq)

		point := VFPoint{
			Index:       i,
			VoltageMV:   voltage,
			OffsetMHz:   offset,
			BaseFreqMHz: baseFreq,
			IsActive:    isActive,
		}

		points = append(points, point)
	}

	return points
}

// isPointActive determines if a point is active (customized) or inactive (stock).
// Inactive points (BaseFreqMHz = 225.0) use stock behavior (offset = 0).
// This typically occurs at voltages the GPU couldn't stabilize during OC Scanner,
// or if the user hasn't customized those points.
// Active points (BaseFreqMHz = 675-3334) have a base frequency set and may have
// user-defined offsets applied.
func isPointActive(baseFreq float32) bool {
	// Check for inactive marker (with small tolerance for float precision)
	diff := baseFreq - VFControlCurveInactiveMarker
	if diff < 0 {
		diff = -diff
	}
	if diff < 1.0 {
		return false
	}

	// Active points have base frequency in 675-3334 range
	return baseFreq >= VFControlCurveMinBaseFreq &&
		baseFreq <= VFControlCurveMaxBaseFreq
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
	point.BaseFreqMHz = VFControlCurveMinBaseFreq // Mark as active

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
			if p.BaseFreqMHz < VFControlCurveMinBaseFreq || p.BaseFreqMHz > VFControlCurveMaxBaseFreq {
				return &VFControlCurveError{
					Field: fmt.Sprintf("Points[%d].BaseFreqMHz", i),
					Message: fmt.Sprintf("active point base freq %.1f MHz outside valid range (%.0f-%.0f)",
						p.BaseFreqMHz, VFControlCurveMinBaseFreq, VFControlCurveMaxBaseFreq),
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

// GetBaseFreqAt returns the base frequency (f1) at a given voltage.
// This is the grey diagonal background line shown in the VF curve editor UI.
// It represents the baseline frequency that the offset is applied to.
// This base frequency may come from OC Scanner results or manual curve editing.
// This is NOT the stock hardware boost curve (which is queried live from the driver).
// Returns 0 if no point exists near the specified voltage.
func (c *VFControlCurveInfo) GetBaseFreqAt(voltageMV float32) float32 {
	point := c.GetPointByVoltage(voltageMV)
	if point == nil {
		return 0.0
	}
	return point.BaseFreqMHz
}

// GetOffsetAt returns the frequency offset (f2) at a given voltage.
// This is the user-defined offset applied on top of the base frequency.
// Applied frequency = BaseFreq(voltage) + Offset
// Positive = overclock relative to base
// Negative = undervolt relative to base
// Zero = use base frequency (no adjustment)
// Returns 0 if no point exists near the specified voltage.
func (c *VFControlCurveInfo) GetOffsetAt(voltageMV float32) float32 {
	point := c.GetPointByVoltage(voltageMV)
	if point == nil {
		return 0.0
	}
	return point.OffsetMHz
}

// Marshal serializes the V-F curve to the hex blob format used in MSI Afterburner .cfg files.
//
// The serialization follows the binary format specification:
//   - Header (12 bytes): [version:uint32][count:uint32][reserved:float32]
//   - Triplets (12 bytes each): [voltage:float32][base:float32][offset:float32]
//
// The returned hex string can be directly written to a .cfg file as the VFCurve= value.
// Do NOT include "0x" prefix or "h" suffix - those are added by the profile writer.
//
// Example:
//
//	hexData, err := curve.Marshal()
//	if err != nil {
//	    return err
//	}
//	// Write to .cfg file: VFCurve=%shexData
//
// This is the inverse of UnmarshalVFControlCurve.
func (c *VFControlCurveInfo) Marshal() (string, error) {
	if c.Version != VFControlCurveVersion2 {
		return "", &VFControlCurveError{
			Field:   "Version",
			Message: fmt.Sprintf("unsupported version 0x%08X, only 0x%08X is supported", c.Version, VFControlCurveVersion2),
		}
	}

	pointCount := len(c.Points)
	if pointCount == 0 {
		return "", &VFControlCurveError{
			Field:   "Points",
			Message: "cannot marshal curve with no points",
		}
	}

	if pointCount > VFControlCurveMaxPoints {
		return "", &VFControlCurveError{
			Field:   "Points",
			Message: fmt.Sprintf("too many points: %d, maximum is %d", pointCount, VFControlCurveMaxPoints),
		}
	}

	// Calculate buffer size: 12-byte header + 12 bytes per point
	headerSize := VFControlCurveHeaderSize
	dataSize := headerSize + (pointCount * VFControlCurveTripletSize)
	binaryData := make([]byte, dataSize)

	// Write header
	// Bytes 0-3: version (little-endian uint32)
	binary.LittleEndian.PutUint32(binaryData[0:4], c.Version)

	// Bytes 4-7: point count (little-endian uint32)
	binary.LittleEndian.PutUint32(binaryData[4:8], uint32(pointCount))

	// Bytes 8-11: reserved (float32, always 0.0)
	binary.LittleEndian.PutUint32(binaryData[8:12], 0)

	// Write triplets for each point
	offset := headerSize
	for i, point := range c.Points {
		if i >= pointCount {
			break
		}

		// Bytes 0-3: voltage (float32)
		binary.LittleEndian.PutUint32(binaryData[offset:offset+4], math.Float32bits(point.VoltageMV))

		// Bytes 4-7: Base frequency (float32)
		// For inactive points, use the inactive marker (225.0)
		baseFreq := point.BaseFreqMHz
		if !point.IsActive {
			baseFreq = VFControlCurveInactiveMarker
		}
		binary.LittleEndian.PutUint32(binaryData[offset+4:offset+8], math.Float32bits(baseFreq))

		// Bytes 8-11: offset (float32)
		binary.LittleEndian.PutUint32(binaryData[offset+8:offset+12], math.Float32bits(point.OffsetMHz))

		offset += VFControlCurveTripletSize
	}

	// Return hex-encoded string (no prefix or suffix)
	return fmt.Sprintf("%x", binaryData), nil
}

// ApplyFlatOffset adds a frequency offset to all active points in the curve.
// This is useful for applying a uniform overclock or undervolt across the entire curve.
//
// Positive offsetMHz increases frequencies (overclock).
// Negative offsetMHz decreases frequencies (undervolt).
//
// Inactive points remain unchanged (they use stock behavior).
//
// Example:
//
//	// Apply +100 MHz overclock to all active points
//	curve.ApplyFlatOffset(100)
//
//	// Apply -50 MHz undervolt to all active points
//	curve.ApplyFlatOffset(-50)
func (c *VFControlCurveInfo) ApplyFlatOffset(offsetMHz float32) {
	for i := range c.Points {
		if c.Points[i].IsActive {
			c.Points[i].OffsetMHz += offsetMHz
		}
	}
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
