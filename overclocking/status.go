// Package overclocking provides high-level GPU overclocking orchestration.
package overclocking

import (
	"fmt"

	"github.com/hekmon/aiup/overclocking/msiaf"
	"github.com/hekmon/aiup/overclocking/nvvf"
)

// VFPoint represents a single voltage-frequency point with all components explicit.
// This structure is designed for AI agent consumption - all values are in MHz except voltage.
//
// Key insight: OffsetMHz is the CORE overclocking value that gets set/modified.
// EffectiveFreqMHz = BaseFreqMHz + OffsetMHz
type VFPoint struct {
	VoltageMV        float64 `json:"voltage_mv"`         // Voltage point (e.g., 850.0 mV)
	BaseFreqMHz      float64 `json:"base_freq_mhz"`      // Hardware base frequency at this voltage
	OffsetMHz        float64 `json:"offset_mhz"`         // Overclock offset applied (THE CORE VALUE)
	EffectiveFreqMHz float64 `json:"effective_freq_mhz"` // Resulting frequency = BaseFreqMHz + OffsetMHz
}

// SavedProfileInfo indicates which profile slot matches the current curve.
// This is informational - helps the user decide where to save modifications.
// All fields are JSON-serializable for MCP/API compatibility.
type SavedProfileInfo struct {
	SlotNumber int     `json:"slot_number"` // 1-5 (Profile1-5)
	SlotName   string  `json:"slot_name"`   // "Profile1", "Profile2", etc.
	Confidence float64 `json:"confidence"`  // Match confidence (0.0-1.0)
}

// CurrentCurveResult contains the result of getting the current V-F curve.
// All fields are JSON-serializable for MCP/API compatibility.
type CurrentCurveResult struct {
	Points  []VFPoint         `json:"points"`  // All voltage points with offsets
	Profile *SavedProfileInfo `json:"profile"` // Which slot matches (null if no match)
}

// GetCurrentCurve reads the current V-F curve from the GPU and validates it matches Startup.
// Additionally checks if Startup matches any saved profile slot (informational for save operations).
//
// This function:
//  1. Reads the live V-F curve from the GPU via NvAPI
//  2. Loads the hardware profile from MSI Afterburner Profiles directory
//  3. Validates that live curve matches Startup profile (error if not - this should not happen)
//  4. Returns the Startup V-F curve (currently applied)
//  5. Additionally checks if Startup matches Profile1-5 (informational for save operations)
//
// Parameters:
//   - gpuIndex: NvAPI GPU index (0, 1, 2, ...)
//   - profilesDir: Path to MSI Afterburner Profiles directory
//
// Returns:
//   - result: CurrentCurveResult with V-F curve points and optional profile match info
//   - error: If live doesn't match Startup, or other errors
//
// Example usage:
//
//	result, err := GetCurrentCurve(0, `C:\Program Files (x86)\MSI Afterburner\Profiles`)
//	if err != nil {
//	    return fmt.Errorf("failed to get current curve: %w", err)
//	}
//
//	// Modify the curve
//	result.Points[0].OffsetMHz = 1000 // Set 1000 MHz offset at first voltage point
//
//	// Inform user about save location
//	if result.Profile != nil {
//	    fmt.Printf("Current curve matches %s - save to same slot?\n", result.Profile.SlotName)
//	} else {
//	    fmt.Println("Current curve is unique - save to any slot")
//	}
func GetCurrentCurve(gpuIndex int, profilesDir string) (*CurrentCurveResult, error) {
	// Step 1: Perform GPU discovery to get profile paths
	discovery, err := ScanGPUs(profilesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan profiles directory: %w", err)
	}

	// Step 2: Find the GPU by index in discovery result
	var gpuInfo *GPUInfo
	for i := range discovery.GPUs {
		if discovery.GPUs[i].Index == gpuIndex {
			gpuInfo = &discovery.GPUs[i]
			break
		}
	}
	if gpuInfo == nil {
		return nil, fmt.Errorf("GPU index %d not found in discovery result", gpuIndex)
	}

	// Step 3: Load hardware profile from disk
	hwProfile, err := msiaf.ParseHardwareProfile(gpuInfo.ProfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load hardware profile %s: %w", gpuInfo.ProfilePath, err)
	}

	// Step 4: Read live V-F curve from GPU via NvAPI
	livePoints, err := nvvf.ReadNvAPIVF(gpuIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to read live V-F curve from GPU %d: %w", gpuIndex, err)
	}

	// Step 5: Build liveFreqs map: voltage (mV) -> frequency (MHz)
	liveFreqs := make(map[float32]float64, len(livePoints))
	for _, pt := range livePoints {
		liveFreqs[float32(pt.VoltageMV)] = pt.BaseFreqMHz
	}

	// Step 6: Match all profile slots against live V-F curve (10 MHz tolerance)
	const toleranceMHz = 10.0
	matchResults, err := msiaf.MatchProfileAgainstLive(liveFreqs, hwProfile, toleranceMHz)
	if err != nil {
		return nil, fmt.Errorf("failed to match profiles against live curve: %w", err)
	}

	// Step 7: CRITICAL - Verify Startup (slot 0) matches live data
	// This should always be true - if not, something is fundamentally wrong
	startupResult := matchResults[0]
	if !startupResult.IsMatch(1.0) {
		return nil, fmt.Errorf("live V-F curve does not match Startup profile (%.0f%% confidence) - this should not happen, user may have unsaved changes", startupResult.MatchConfidence*100)
	}

	// Step 8: Extract Startup V-F curve (slot 0)
	vfCurveRaw, err := extractVFCurve(hwProfile, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to extract V-F curve from Startup slot: %w", err)
	}

	// Step 9: Convert to user-friendly VFPoint format
	points := convertToVFPoints(vfCurveRaw)

	// Step 10: Additionally check if Startup matches any Profile1-5 slot
	// This is informational - helps user decide where to save modifications
	var savedProfile *SavedProfileInfo
	for i := 1; i <= 5; i++ {
		slotResult := matchResults[i]
		// Check if this slot matches with high confidence (>=90%)
		if slotResult.IsMatch(0.9) {
			savedProfile = &SavedProfileInfo{
				SlotNumber: i,
				SlotName:   slotResult.SlotName,
				Confidence: slotResult.MatchConfidence,
			}
			break // Return first match (highest priority slot)
		}
	}

	return &CurrentCurveResult{
		Points:  points,
		Profile: savedProfile,
	}, nil
}

// extractVFCurve extracts the raw V-F curve from a specific profile slot.
// slotNum: 0=Startup, 1-5=Profile1-5
func extractVFCurve(hwProfile *msiaf.HardwareProfile, slotNum int) (*msiaf.VFControlCurveInfo, error) {
	var section *msiaf.ProfileSection

	switch slotNum {
	case 0:
		section = &hwProfile.Startup
	case 1:
		section = &hwProfile.Profiles[0].ProfileSection
	case 2:
		section = &hwProfile.Profiles[1].ProfileSection
	case 3:
		section = &hwProfile.Profiles[2].ProfileSection
	case 4:
		section = &hwProfile.Profiles[3].ProfileSection
	case 5:
		section = &hwProfile.Profiles[4].ProfileSection
	default:
		return nil, fmt.Errorf("invalid slot number: %d", slotNum)
	}

	if section == nil || section.VFCurve == nil || len(section.VFCurve) == 0 {
		return nil, fmt.Errorf("no VFCurve data in slot %d", slotNum)
	}

	// Convert []byte to hex string for parsing
	hexData := fmt.Sprintf("%x", section.VFCurve)

	// Parse using existing V-F curve unmarshaler
	return msiaf.UnmarshalVFControlCurve(hexData)
}

// convertToVFPoints converts raw VFControlCurveInfo to user-friendly VFPoint slice.
// Each VFPoint exposes: voltage, base frequency, offset, and effective frequency.
func convertToVFPoints(curve *msiaf.VFControlCurveInfo) []VFPoint {
	if curve == nil || len(curve.Points) == 0 {
		return []VFPoint{}
	}

	points := make([]VFPoint, 0, len(curve.Points))
	for _, pt := range curve.Points {
		points = append(points, VFPoint{
			VoltageMV:        float64(pt.VoltageMV),
			BaseFreqMHz:      float64(pt.BaseFreqMHz),
			OffsetMHz:        float64(pt.OffsetMHz),
			EffectiveFreqMHz: float64(pt.BaseFreqMHz + pt.OffsetMHz),
		})
	}

	return points
}
