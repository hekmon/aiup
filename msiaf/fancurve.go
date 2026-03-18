// Package msiaf provides tooling for working with MSI Afterburner profiles and configuration.
package msiaf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// ============================================================================
// Fan Curve Binary Format Specification
// ============================================================================
//
// MSI Afterburner stores software auto fan control curves as a 256-byte binary
// blob in the configuration file. The format uses little-endian byte ordering
// (standard for x86/x64 Windows systems).
//
// Binary Layout (256 bytes total):
// ┌──────────────┬──────────────┬──────────────┬─────────────────────────────┐
// │  Bytes 0-3   │  Bytes 4-7   │  Bytes 8-11  │       Bytes 12-255          │
// │   Version    │ Point Count  │   Reserved   │    Curve Points + Padding   │
// │  (uint32)    │  (uint32)    │  (uint32)    │   (float32 pairs + zeros)   │
// └──────────────┴──────────────┴──────────────┴─────────────────────────────┘
//
// Curve Points (starting at byte 12):
// Each point is 8 bytes: [Temperature (float32)] [FanSpeed (float32)]
// - Temperature: Target temperature in Celsius (°C)
// - FanSpeed: Fan speed percentage (0-100%)
//
// Example (4 points = 32 bytes of point data + 12 bytes header = 44 bytes used):
// Points: [(30°C, 40%), (50°C, 50%), (80°C, 80%), (90°C, 100%)]
// Remaining 212 bytes are zero-padded to reach 256 bytes total.
//
// Points MUST be sorted by temperature in ascending order for correct
// linear interpolation between consecutive points.
//
// ============================================================================

// FanCurveBinaryFormatVersion is the version number used in the binary format.
// Value 0x00010000 (65536) represents version 1.0 in a packed format.
// Stored as little-endian uint32: bytes 00 00 01 00 in the binary blob.
const FanCurveBinaryFormatVersion = 0x00010000

// FanCurveBlockSize is the fixed size of the serialized fan curve data.
// This is always 256 bytes regardless of the number of curve points.
const FanCurveBlockSize = 256

// FanCurveHeaderSize is the size of the header portion of the binary format.
// Contains version (4 bytes), point count (4 bytes), and reserved (4 bytes).
const FanCurveHeaderSize = 12

// FanCurvePointOffset is the byte offset where curve point data begins.
// Points start immediately after the header, so this equals FanCurveHeaderSize.
const FanCurvePointOffset = FanCurveHeaderSize

// FanCurveMaxPoints is the maximum number of points that can be stored.
// Calculated from available space after header: (256 - 12) / 8 = 30 points.
// In practice, MSI Afterburner typically uses 4-6 points.
const FanCurveMaxPoints = (FanCurveBlockSize - FanCurveHeaderSize) / 8

// FanCurvePointSize is the size of each curve point in bytes.
// Each point contains two float32 values: temperature (4 bytes) + fan speed (4 bytes).
const FanCurvePointSize = 8

// FanCurveValidationRange defines the arbitrary but reasonable validation bounds
// for fan curve values. These are NOT hardware limits - they are sanity checks
// to detect corrupted or incorrectly decoded data.
//
// Temperature range: -50°C to 150°C
//   - Below -50°C: Impossible for GPU operation
//   - Above 150°C: Beyond any safe GPU temperature (thermal throttling occurs ~85-95°C)
//
// Fan speed range: 0% to 100%
//   - Physical limits of fan speed control
const (
	FanCurveMinTemperature = -50.0
	FanCurveMaxTemperature = 150.0
	FanCurveMinFanSpeed    = 0.0
	FanCurveMaxFanSpeed    = 100.0
)

// Fan curve parsing errors.
var (
	ErrFanCurveDataTooShort       = errors.New("fan curve data too short for header")
	ErrFanCurveInvalidSize        = errors.New("fan curve data must be exactly 256 bytes")
	ErrFanCurveTooManyPoints      = errors.New("fan curve point count exceeds maximum")
	ErrFanCurveDataTruncated      = errors.New("fan curve data truncated before all points read")
	ErrFanCurveInvalidVersion     = errors.New("fan curve has unrecognized version")
	ErrFanCurveInvalidTemperature = errors.New("fan curve temperature value is invalid (NaN, Inf, or out of range)")
	ErrFanCurveInvalidFanSpeed    = errors.New("fan curve fan speed value is invalid (NaN, Inf, or out of range)")
	ErrFanCurvePointsNotSorted    = errors.New("fan curve points must be sorted by temperature in ascending order")
	ErrFanCurveNoPoints           = errors.New("fan curve has no points")
)

// FanCurveError provides detailed error information for fan curve parsing failures.
//
// This error type helps diagnose issues with corrupted or malformed fan curve data
// by identifying exactly which field failed validation and why.
//
// Example:
//
//	curve, err := UnmarshalSwAutoFanControlCurve(data)
//	if err != nil {
//	    var fanErr *FanCurveError
//	    if errors.As(err, &fanErr) {
//	        fmt.Printf("Operation: %s\n", fanErr.Op)
//	        fmt.Printf("Field: %s\n", fanErr.Field)
//	        fmt.Printf("Value: %v\n", fanErr.Value)
//	        fmt.Printf("Problem: %v\n", fanErr.Err)
//	    }
//	}
type FanCurveError struct {
	// Op is the operation that failed (e.g., "unmarshal", "validate").
	Op string

	// Field identifies which field or data element failed validation.
	// Examples: "data", "version", "pointCount", "point[2].temperature"
	Field string

	// Value is the problematic value that caused the failure.
	Value any

	// Err is the underlying error describing what went wrong.
	Err error
}

// Error implements the error interface for FanCurveError.
func (e *FanCurveError) Error() string {
	return fmt.Sprintf("fan curve %s: %s = %v: %v", e.Op, e.Field, e.Value, e.Err)
}

// Unwrap implements the errors.Unwrap interface for FanCurveError.
// This allows using errors.Is() and errors.As() with wrapped errors.
func (e *FanCurveError) Unwrap() error {
	return e.Err
}

// FanCurvePoint defines a single point on the fan control curve.
//
// The fan control system uses linear interpolation between consecutive points
// to determine the target fan speed for temperatures between defined points.
// Points MUST be sorted by temperature in ascending order.
//
// Example:
//
//	point := FanCurvePoint{
//	    Temperature: 50.0,  // 50°C
//	    FanSpeed:    75.0,  // 75% fan speed
//	}
type FanCurvePoint struct {
	// Temperature is the GPU temperature threshold in Celsius.
	// Valid range: FanCurveMinTemperature to FanCurveMaxTemperature (-50 to 150°C).
	// Typical values range from 30°C (idle) to 90°C (maximum safe temperature).
	Temperature float32

	// FanSpeed is the target fan speed as a percentage.
	// Valid range: FanCurveMinFanSpeed to FanCurveMaxFanSpeed (0-100%).
	// Values: 0.0 (fans off) to 100.0 (maximum speed).
	FanSpeed float32
}

// SwAutoFanControlCurveInfo represents a complete software auto fan control curve.
//
// This structure holds the deserialized representation of the 256-byte binary
// blob stored in MSI Afterburner's configuration file. It can be used to:
//   - Read existing fan curves from configuration files
//   - Modify curve points programmatically
//   - Create custom fan curves
//   - Serialize back to binary format for saving
//
// Version Format:
// The Version field uses a packed 32-bit format:
//   - High 16 bits: Major version
//   - Low 16 bits: Minor version
//
// For example, 0x00010000 (65536) represents version 1.0.
// Use VersionString() to get a human-readable version like "1.0".
//
// Validation:
//   - Points must be sorted by temperature in ascending order
//   - Temperatures must be in range [-50, 150] °C
//   - Fan speeds must be in range [0, 100] %
//   - At least one point is required
//
// Example - Creating a custom aggressive cooling curve:
//
//	curve := &SwAutoFanControlCurveInfo{
//	    Version: FanCurveBinaryFormatVersion,
//	    Points: []FanCurvePoint{
//	        {Temperature: 30, FanSpeed: 40},  // 40% at idle
//	        {Temperature: 50, FanSpeed: 50},  // 50% at moderate load
//	        {Temperature: 70, FanSpeed: 90},  // 90% at high load
//	        {Temperature: 80, FanSpeed: 100}, // 100% at critical temp
//	    },
//	}
//	err := curve.Validate() // Check if curve is valid
//	if err != nil {
//	    // Handle validation error
//	}
type SwAutoFanControlCurveInfo struct {
	// Version is the binary format version number.
	// Must be FanCurveBinaryFormatVersion (0x00010000) for compatibility.
	// The version uses a packed format: high 16 bits = major, low 16 bits = minor.
	// For example, 0x00010000 represents version 1.0.
	Version uint32

	// Points is the ordered list of temperature-to-fan-speed mapping points.
	// Points MUST be sorted by temperature in ascending order.
	// MSI Afterburner uses linear interpolation between consecutive points.
	// Minimum 1 point required, maximum FanCurveMaxPoints (30).
	Points []FanCurvePoint
}

// Validate checks if the fan curve is valid and well-formed.
//
// Validation checks:
//   - Version matches FanCurveBinaryFormatVersion
//   - At least one point exists
//   - Point count does not exceed FanCurveMaxPoints
//   - All temperatures are in valid range (FanCurveMinTemperature to FanCurveMaxTemperature)
//   - All fan speeds are in valid range (FanCurveMinFanSpeed to FanCurveMaxFanSpeed)
//   - No NaN or Inf values
//   - Points are sorted by temperature in ascending order
//
// Returns:
//
//	nil if the curve is valid, or a descriptive error explaining the validation failure.
//
// Example:
//
//	curve := &SwAutoFanControlCurveInfo{
//	    Version: FanCurveBinaryFormatVersion,
//	    Points: []FanCurvePoint{
//	        {Temperature: 30, FanSpeed: 40},
//	        {Temperature: 50, FanSpeed: 50},
//	    },
//	}
//	if err := curve.Validate(); err != nil {
//	    fmt.Printf("Invalid curve: %v\n", err)
//	}
func (c *SwAutoFanControlCurveInfo) Validate() error {
	// Validate version
	if c.Version != FanCurveBinaryFormatVersion {
		return &FanCurveError{
			Op:    "validate",
			Field: "version",
			Value: c.Version,
			Err:   ErrFanCurveInvalidVersion,
		}
	}

	// Validate point count
	numPoints := len(c.Points)
	if numPoints == 0 {
		return &FanCurveError{
			Op:    "validate",
			Field: "points",
			Value: numPoints,
			Err:   ErrFanCurveNoPoints,
		}
	}
	if numPoints > FanCurveMaxPoints {
		return &FanCurveError{
			Op:    "validate",
			Field: "pointCount",
			Value: numPoints,
			Err:   ErrFanCurveTooManyPoints,
		}
	}

	// Validate each point's values
	for i, point := range c.Points {
		// Validate temperature
		if math.IsNaN(float64(point.Temperature)) || math.IsInf(float64(point.Temperature), 0) {
			return &FanCurveError{
				Op:    "validate",
				Field: fmt.Sprintf("point[%d].temperature", i),
				Value: point.Temperature,
				Err:   fmt.Errorf("%w: value is NaN or Inf", ErrFanCurveInvalidTemperature),
			}
		}
		if point.Temperature < FanCurveMinTemperature || point.Temperature > FanCurveMaxTemperature {
			return &FanCurveError{
				Op:    "validate",
				Field: fmt.Sprintf("point[%d].temperature", i),
				Value: point.Temperature,
				Err: fmt.Errorf("%w: %.2f°C outside valid range [%.0f, %.0f]",
					ErrFanCurveInvalidTemperature, point.Temperature, FanCurveMinTemperature, FanCurveMaxTemperature),
			}
		}

		// Validate fan speed
		if math.IsNaN(float64(point.FanSpeed)) || math.IsInf(float64(point.FanSpeed), 0) {
			return &FanCurveError{
				Op:    "validate",
				Field: fmt.Sprintf("point[%d].fanSpeed", i),
				Value: point.FanSpeed,
				Err:   fmt.Errorf("%w: value is NaN or Inf", ErrFanCurveInvalidFanSpeed),
			}
		}
		if point.FanSpeed < FanCurveMinFanSpeed || point.FanSpeed > FanCurveMaxFanSpeed {
			return &FanCurveError{
				Op:    "validate",
				Field: fmt.Sprintf("point[%d].fanSpeed", i),
				Value: point.FanSpeed,
				Err: fmt.Errorf("%w: %.2f%% outside valid range [%.0f, %.0f]",
					ErrFanCurveInvalidFanSpeed, point.FanSpeed, FanCurveMinFanSpeed, FanCurveMaxFanSpeed),
			}
		}
	}

	// Validate points are sorted by temperature in ascending order
	for i := 1; i < numPoints; i++ {
		if c.Points[i].Temperature <= c.Points[i-1].Temperature {
			return &FanCurveError{
				Op:    "validate",
				Field: fmt.Sprintf("points[%d].temperature", i),
				Value: fmt.Sprintf("%.2f°C (previous: %.2f°C)", c.Points[i].Temperature, c.Points[i-1].Temperature),
				Err:   ErrFanCurvePointsNotSorted,
			}
		}
	}

	return nil
}

// VersionString returns a human-readable version string for the fan curve.
//
// The Version field uses a packed 32-bit format where:
//   - High 16 bits: Major version
//   - Low 16 bits: Minor version
//
// For example, Version = 65536 (0x00010000) returns "1.0".
//
// This method provides a convenient way to display the version in logs,
// error messages, or UI output without manually decoding the packed format.
//
// Example:
//
//	curve, _ := UnmarshalSwAutoFanControlCurve(data)
//	fmt.Printf("Fan curve version: %s\n", curve.VersionString())
//	// Output: "Fan curve version: 1.0"
func (c *SwAutoFanControlCurveInfo) VersionString() string {
	major := (c.Version >> 16) & 0xFFFF
	minor := c.Version & 0xFFFF
	return fmt.Sprintf("%d.%d", major, minor)
}

// Marshal serializes the fan control curve to the 256-byte binary format
// used by MSI Afterburner configuration files.
//
// IMPORTANT: Call Validate() before Marshal() to ensure the curve is valid.
// Marshal() does not perform validation - it assumes the curve is already valid.
//
// The serialization process:
//  1. Writes the version number (4 bytes, little-endian uint32)
//  2. Writes the number of points (4 bytes, little-endian uint32)
//  3. Writes reserved bytes (4 bytes of zeros)
//  4. Writes each point as two float32 values (8 bytes per point)
//  5. Zero-pads the remaining bytes to reach 256 bytes total
//
// Returns:
//
//	A 256-byte slice containing the serialized curve data.
//	This can be directly written to the config file as a hex string.
//
// Example:
//
//	curve := &SwAutoFanControlCurveInfo{
//	    Version: FanCurveBinaryFormatVersion,
//	    Points: []FanCurvePoint{
//	        {Temperature: 30, FanSpeed: 40},
//	        {Temperature: 50, FanSpeed: 50},
//	    },
//	}
//	if err := curve.Validate(); err != nil {
//	    return err
//	}
//	data := curve.Marshal() // Returns []byte of length 256
//	hexStr := hex.EncodeToString(data) + "h" // For config file
func (c *SwAutoFanControlCurveInfo) Marshal() []byte {
	// Allocate fixed 256-byte buffer, pre-zeroed
	data := make([]byte, FanCurveBlockSize)

	// Write header: version (bytes 0-3, little-endian uint32)
	binary.LittleEndian.PutUint32(data[0:4], c.Version)

	// Write header: point count (bytes 4-7, little-endian uint32)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(c.Points)))

	// Write header: reserved bytes (bytes 8-11) - already zero from allocation

	// Write curve points starting at byte 12
	// Each point is 8 bytes: [temperature float32][fanSpeed float32]
	offset := FanCurvePointOffset
	for _, point := range c.Points {
		// Convert float32 to IEEE 754 bit representation
		tempBits := math.Float32bits(point.Temperature)
		fanBits := math.Float32bits(point.FanSpeed)

		// Write as little-endian uint32
		binary.LittleEndian.PutUint32(data[offset:offset+4], tempBits)
		binary.LittleEndian.PutUint32(data[offset+4:offset+8], fanBits)

		offset += FanCurvePointSize
	}

	// Remaining bytes (offset to 256) are already zero-padded

	return data
}

// UnmarshalSwAutoFanControlCurve deserializes a 256-byte binary fan curve
// into a SwAutoFanControlCurveInfo structure.
//
// This function performs STRICT validation:
//   - Data must be exactly 256 bytes
//   - Version must match FanCurveBinaryFormatVersion
//   - Point count must be <= FanCurveMaxPoints
//   - All temperatures must be in range [-50, 150] °C
//   - All fan speeds must be in range [0, 100] %
//   - Points must be sorted by temperature in ascending order
//   - No NaN or Inf values allowed
//
// Parameters:
//
//	data - The raw binary data from the config file (must be exactly 256 bytes).
//	       Typically obtained from Settings.SwAutoFanControlCurve.
//
// Returns:
//
//	*SwAutoFanControlCurveInfo and nil error if successful.
//	nil and a descriptive FanCurveError if validation fails.
//
// Example:
//
//	// After parsing config
//	config, _ := ParseGlobalConfig("MSIAfterburner.cfg")
//
//	// Deserialize the curve with strict validation
//	curve, err := UnmarshalSwAutoFanControlCurve(config.Settings.SwAutoFanControlCurve)
//	if err != nil {
//	    var fanErr *FanCurveError
//	    if errors.As(err, &fanErr) {
//	        fmt.Printf("Failed to parse curve at field: %s\n", fanErr.Field)
//	        fmt.Printf("Problematic value: %v\n", fanErr.Value)
//	        fmt.Printf("Error: %v\n", fanErr.Err)
//	    }
//	    return err
//	}
//
//	fmt.Printf("Curve has %d points\n", len(curve.Points))
//	for _, p := range curve.Points {
//	    fmt.Printf("  %.1f°C → %.1f%%\n", p.Temperature, p.FanSpeed)
//	}
func UnmarshalSwAutoFanControlCurve(data []byte) (*SwAutoFanControlCurveInfo, error) {
	// STRICT: Validate exact data size
	if len(data) != FanCurveBlockSize {
		return nil, &FanCurveError{
			Op:    "unmarshal",
			Field: "data",
			Value: fmt.Sprintf("%d bytes", len(data)),
			Err:   ErrFanCurveInvalidSize,
		}
	}

	// Read header: version number
	version := binary.LittleEndian.Uint32(data[0:4])

	// STRICT: Validate version
	if version != FanCurveBinaryFormatVersion {
		return nil, &FanCurveError{
			Op:    "unmarshal",
			Field: "version",
			Value: fmt.Sprintf("0x%08X (%d)", version, version),
			Err:   ErrFanCurveInvalidVersion,
		}
	}

	// Read header: number of points
	numPoints := binary.LittleEndian.Uint32(data[4:8])

	// Validate point count is reasonable
	if numPoints > FanCurveMaxPoints {
		return nil, &FanCurveError{
			Op:    "unmarshal",
			Field: "pointCount",
			Value: numPoints,
			Err:   ErrFanCurveTooManyPoints,
		}
	}

	// Validate we have at least one point
	if numPoints == 0 {
		return nil, &FanCurveError{
			Op:    "unmarshal",
			Field: "pointCount",
			Value: numPoints,
			Err:   ErrFanCurveNoPoints,
		}
	}

	// Validate we have enough data for all declared points
	expectedSize := FanCurvePointOffset + int(numPoints)*FanCurvePointSize
	if len(data) < expectedSize {
		return nil, &FanCurveError{
			Op:    "unmarshal",
			Field: "data",
			Value: fmt.Sprintf("%d bytes (need %d for %d points)", len(data), expectedSize, numPoints),
			Err:   ErrFanCurveDataTruncated,
		}
	}

	// Parse and validate each point
	points := make([]FanCurvePoint, 0, numPoints)
	offset := FanCurvePointOffset
	var prevTemp float32 = -9999 // Sentinel value lower than any valid temperature

	for i := uint32(0); i < numPoints; i++ {
		// Read temperature float32 (bytes offset to offset+4)
		tempBits := binary.LittleEndian.Uint32(data[offset : offset+4])
		temp := math.Float32frombits(tempBits)

		// Read fan speed float32 (bytes offset+4 to offset+8)
		fanBits := binary.LittleEndian.Uint32(data[offset+4 : offset+8])
		fan := math.Float32frombits(fanBits)

		// STRICT: Validate temperature is not NaN or Inf
		if math.IsNaN(float64(temp)) || math.IsInf(float64(temp), 0) {
			return nil, &FanCurveError{
				Op:    "validate",
				Field: fmt.Sprintf("point[%d].temperature", i),
				Value: temp,
				Err:   fmt.Errorf("%w: value is NaN or Inf", ErrFanCurveInvalidTemperature),
			}
		}

		// STRICT: Validate temperature is in reasonable range
		if temp < FanCurveMinTemperature || temp > FanCurveMaxTemperature {
			return nil, &FanCurveError{
				Op:    "validate",
				Field: fmt.Sprintf("point[%d].temperature", i),
				Value: temp,
				Err: fmt.Errorf("%w: %.2f°C outside valid range [%.0f, %.0f]",
					ErrFanCurveInvalidTemperature, temp, FanCurveMinTemperature, FanCurveMaxTemperature),
			}
		}

		// STRICT: Validate points are sorted by temperature (ascending)
		if temp <= prevTemp {
			return nil, &FanCurveError{
				Op:    "validate",
				Field: fmt.Sprintf("point[%d].temperature", i),
				Value: fmt.Sprintf("%.2f°C (previous: %.2f°C)", temp, prevTemp),
				Err:   ErrFanCurvePointsNotSorted,
			}
		}
		prevTemp = temp

		// STRICT: Validate fan speed is not NaN or Inf
		if math.IsNaN(float64(fan)) || math.IsInf(float64(fan), 0) {
			return nil, &FanCurveError{
				Op:    "validate",
				Field: fmt.Sprintf("point[%d].fanSpeed", i),
				Value: fan,
				Err:   fmt.Errorf("%w: value is NaN or Inf", ErrFanCurveInvalidFanSpeed),
			}
		}

		// STRICT: Validate fan speed is in valid range [0, 100]
		if fan < FanCurveMinFanSpeed || fan > FanCurveMaxFanSpeed {
			return nil, &FanCurveError{
				Op:    "validate",
				Field: fmt.Sprintf("point[%d].fanSpeed", i),
				Value: fan,
				Err: fmt.Errorf("%w: %.2f%% outside valid range [%.0f, %.0f]",
					ErrFanCurveInvalidFanSpeed, fan, FanCurveMinFanSpeed, FanCurveMaxFanSpeed),
			}
		}

		points = append(points, FanCurvePoint{
			Temperature: temp,
			FanSpeed:    fan,
		})

		offset += FanCurvePointSize
	}

	return &SwAutoFanControlCurveInfo{
		Version: version,
		Points:  points,
	}, nil
}

// GetFanControlCurve returns the deserialized software auto fan control curve.
//
// This method provides convenient access to the parsed fan curve data.
// The curve is automatically parsed when the config file is loaded via
// ParseGlobalConfig().
//
// IMPORTANT: If the curve failed to parse during config loading, this method
// returns nil. Use GetFanControlCurveError() to get detailed error information.
//
// Returns:
//
//	*SwAutoFanControlCurveInfo if a valid curve exists and was successfully parsed.
//	nil if the curve was not set, malformed, or failed validation.
//
// Example:
//
//	config, _ := ParseGlobalConfig("MSIAfterburner.cfg")
//	curve := config.Settings.GetFanControlCurve()
//	if curve != nil {
//	    for _, point := range curve.Points {
//	        fmt.Printf("%.0f°C → %.0f%% fan\n", point.Temperature, point.FanSpeed)
//	    }
//	} else {
//	    // Check if there was a parse error
//	    if err := config.Settings.GetFanControlCurveError(); err != nil {
//	        fmt.Printf("Curve parse error: %v\n", err)
//	    }
//	}
func (s *Settings) GetFanControlCurve() *SwAutoFanControlCurveInfo {
	return s.parsedCurve
}

// GetFanControlCurveError returns the error from parsing the fan control curve.
//
// This method provides access to detailed error information if curve parsing
// failed during config file loading. Returns nil if parsing was successful
// or if no curve was set.
//
// Returns:
//
//	error if curve parsing failed, nil if successful or no curve was set.
//
// Example:
//
//	config, _ := ParseGlobalConfig("MSIAfterburner.cfg")
//	if err := config.Settings.GetFanControlCurveError(); err != nil {
//	    var fanErr *FanCurveError
//	    if errors.As(err, &fanErr) {
//	        fmt.Printf("Failed at field: %s\n", fanErr.Field)
//	    }
//	}
func (s *Settings) GetFanControlCurveError() error {
	return s.parsedCurveErr
}

// GetFanControlCurve2 returns the deserialized backup fan control curve.
//
// MSI Afterburner stores a secondary curve (SwAutoFanControlCurve2) which
// appears to be a backup or alternative profile. The format is identical
// to the primary curve.
//
// IMPORTANT: If the curve failed to parse during config loading, this method
// returns nil. Use GetFanControlCurve2Error() to get detailed error information.
//
// Returns:
//
//	*SwAutoFanControlCurveInfo if a valid backup curve exists and was successfully parsed.
//	nil if the curve was not set, malformed, or failed validation.
func (s *Settings) GetFanControlCurve2() *SwAutoFanControlCurveInfo {
	return s.parsedCurve2
}

// GetFanControlCurve2Error returns the error from parsing the backup fan control curve.
//
// Returns:
//
//	error if curve parsing failed, nil if successful or no curve was set.
func (s *Settings) GetFanControlCurve2Error() error {
	return s.parsedCurve2Err
}

// SetFanControlCurve sets the software auto fan control curve.
//
// This method validates the curve before serializing it. If validation fails,
// an error is returned and the curve is NOT updated.
//
// Parameters:
//
//	curve - The fan curve to set. If nil, clears the curve data.
//
// Returns:
//
//	error if the curve fails validation, nil if successful or curve is nil.
//
// Example:
//
//	config, _ := ParseGlobalConfig("MSIAfterburner.cfg")
//
//	// Create a silent cooling profile
//	silentCurve := &SwAutoFanControlCurveInfo{
//	    Version: FanCurveBinaryFormatVersion,
//	    Points: []FanCurvePoint{
//	        {Temperature: 40, FanSpeed: 30},  // Silent at low temps
//	        {Temperature: 60, FanSpeed: 50},  // Moderate at medium temps
//	        {Temperature: 80, FanSpeed: 70},  // Aggressive at high temps
//	    },
//	}
//
//	if err := config.Settings.SetFanControlCurve(silentCurve); err != nil {
//	    fmt.Printf("Invalid curve: %v\n", err)
//	    return err
//	}
//
//	// Note: You must write the config file back to persist changes
//	// (config file writing is not yet implemented in this package)
func (s *Settings) SetFanControlCurve(curve *SwAutoFanControlCurveInfo) error {
	if curve == nil {
		s.SwAutoFanControlCurve = nil
		s.parsedCurve = nil
		s.parsedCurveErr = nil
		return nil
	}

	// STRICT: Validate before setting
	if err := curve.Validate(); err != nil {
		return err
	}

	s.SwAutoFanControlCurve = curve.Marshal()
	s.parsedCurve = curve
	s.parsedCurveErr = nil
	return nil
}

// SetFanControlCurve2 sets the backup software auto fan control curve.
//
// This method validates the curve before serializing it. If validation fails,
// an error is returned and the curve is NOT updated.
//
// Parameters:
//
//	curve - The fan curve to set as backup. If nil, clears the curve data.
//
// Returns:
//
//	error if the curve fails validation, nil if successful or curve is nil.
//
// See SetFanControlCurve for usage example.
func (s *Settings) SetFanControlCurve2(curve *SwAutoFanControlCurveInfo) error {
	if curve == nil {
		s.SwAutoFanControlCurve2 = nil
		s.parsedCurve2 = nil
		s.parsedCurve2Err = nil
		return nil
	}

	// STRICT: Validate before setting
	if err := curve.Validate(); err != nil {
		return err
	}

	s.SwAutoFanControlCurve2 = curve.Marshal()
	s.parsedCurve2 = curve
	s.parsedCurve2Err = nil
	return nil
}
