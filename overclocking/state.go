// Package overclocking provides high-level GPU overclocking orchestration.
package overclocking

import (
	"fmt"

	"github.com/hekmon/aiup/overclocking/msiaf"
	"github.com/hekmon/aiup/overclocking/nvvf"
)

// CurrentStateResult contains the complete current GPU overclocking state.
// This includes the V-F curve, memory overclock, power limit, and fan settings.
// All fields are JSON-serializable for MCP/API compatibility.
type CurrentStateResult struct {
	// V-F Curve (core overclock)
	Points []VFPoint `json:"points"` // All voltage points with offsets

	// Memory overclock
	MemClkBoostMHz int `json:"mem_clk_boost_mhz"` // Memory clock offset in MHz (e.g., 3000)

	// Power and thermal limits
	PowerLimitPercent int `json:"power_limit_percent"` // Power limit percentage (e.g., 100)

	// Fan settings
	FanMode         string `json:"fan_mode"`          // "auto" or "manual"
	FanSpeedPercent *int   `json:"fan_speed_percent"` // Manual fan speed (0-100) if in manual mode

	// Profile matching info
	LiveMatchesStartup bool              `json:"live_matches_startup"` // true if live curve matches Startup profile
	Profile            *SavedProfileInfo `json:"profile"`              // Which saved profile slot matches (null if none)
}

// VFPoint represents a single voltage-frequency point with all components explicit.
// This structure is designed for AI agent consumption - all values are in MHz except voltage.
//
// Key insight: OffsetMHz is the CORE overclocking value that gets set/modified.
// EffectiveFreqMHz = BaseFreqMHz + OffsetMHz
type VFPoint struct {
	VoltageMV   float64 `json:"voltage_mv"`    // Voltage point (e.g., 850.0 mV)
	BaseFreqMHz float64 `json:"base_freq_mhz"` // Hardware base frequency at this voltage
	OffsetMHz   float64 `json:"offset_mhz"`    // Overclock offset applied (THE CORE VALUE)
}

func (vfp VFPoint) EffectiveFreqMHz() float64 {
	return vfp.BaseFreqMHz + vfp.OffsetMHz
}

// SavedProfileInfo indicates which profile slot matches the current curve.
// This is informational - helps the user decide where to save modifications.
// All fields are JSON-serializable for MCP/API compatibility.
type SavedProfileInfo struct {
	SlotNumber int     `json:"slot_number"` // 1-5 (Profile1-5)
	SlotName   string  `json:"slot_name"`   // "Profile1", "Profile2", etc.
	Confidence float64 `json:"confidence"`  // Match confidence (0.0-1.0)
}

// GetCurrentState reads the complete current GPU overclocking state from the GPU and profile.
//
// This function:
//  1. Reads the live V-F curve from the GPU via NvAPI
//  2. Loads the hardware profile from the specified profile file path
//  3. Compares live curve against Startup profile (reports match status, does not error on mismatch)
//  4. Extracts all current settings: V-F curve, memory overclock, power limit, fan settings
//  5. Additionally checks if Startup matches Profile1-5 (informational for save operations)
//
// Parameters:
//   - gpuIndex: NvAPI GPU index (0, 1, 2, ...)
//   - profilePath: Path to the hardware profile .cfg file (from DiscoveryResult.GPUs[i].ProfilePath)
//
// Returns:
//   - result: CurrentStateResult with complete GPU state
//   - error: Only for actual errors (file read failures, NvAPI errors, etc.)
//
// Example usage:
//
//	// First, discover GPUs and their profile paths
//	discovery, err := overclocking.ScanGPUs(profilesDir)
//	if err != nil {
//	    return err
//	}
//
//	// Get current state for GPU 0 using its profile path
//	result, err := overclocking.GetCurrentState(0, discovery.GPUs[0].ProfilePath)
//	if err != nil {
//	    return fmt.Errorf("failed to get current state: %w", err)
//	}
//
//	// Check if live curve matches Startup
//	if !result.LiveMatchesStartup {
//	    fmt.Println("Warning: Live curve differs from Startup profile (unsaved changes?)")
//	}
//
//	// Access current settings
//	fmt.Printf("Memory overclock: +%d MHz\n", result.MemClkBoostMHz)
//	fmt.Printf("Power limit: %d%%\n", result.PowerLimitPercent)
//	fmt.Printf("Fan mode: %s", result.FanMode)
//	if result.FanSpeedPercent != nil {
//	    fmt.Printf(" (%d%%)\n", *result.FanSpeedPercent)
//	}
//
//	// Modify the V-F curve
//	result.Points[0].OffsetMHz = 1000 // Set 1000 MHz offset at first voltage point
//
//	// Inform user about save location
//	if result.Profile != nil {
//	    fmt.Printf("Current curve matches %s - save to same slot?\n", result.Profile.SlotName)
//	} else {
//	    fmt.Println("Current curve is unique - save to any slot")
//	}
func GetCurrentState(gpuIndex int, profilePath string) (*CurrentStateResult, error) {
	// Step 1: Load hardware profile from disk
	hwProfile, err := msiaf.ParseHardwareProfile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load hardware profile %s: %w", profilePath, err)
	}

	// Step 2: Extract current settings from Startup section
	startup := &hwProfile.Startup

	// Extract memory overclock (convert from kHz to MHz)
	memClkBoostMHz := startup.GetMemClkBoostMHz()

	// Extract power limit
	powerLimitPercent := startup.GetPowerLimit()

	// Extract fan settings
	fanMode := startup.GetFanMode()
	fanModeStr := fanMode.String()
	var fanSpeedPercent *int
	if fanMode == msiaf.FanModeManual {
		fs := startup.GetFanSpeed()
		fanSpeedPercent = &fs
	}

	// Step 3: Read live V-F curve from GPU via NvAPI
	livePoints, err := nvvf.ReadNvAPIVF(gpuIndex)
	if err != nil {
		return nil, fmt.Errorf("failed to read live V-F curve from GPU %d: %w", gpuIndex, err)
	}

	// Step 4: Build liveFreqs map: voltage (mV) -> frequency (MHz)
	liveFreqs := make(map[float32]float64, len(livePoints))
	for _, pt := range livePoints {
		liveFreqs[float32(pt.VoltageMV)] = pt.BaseFreqMHz
	}

	// Step 5: Match all profile slots against live V-F curve (10 MHz tolerance)
	const toleranceMHz = 10.0
	matchResults, err := msiaf.MatchProfileAgainstLive(liveFreqs, hwProfile, toleranceMHz)
	if err != nil {
		return nil, fmt.Errorf("failed to match profiles against live curve: %w", err)
	}

	// Step 6: Check if Startup (slot 0) matches live data
	// This is informational - caller decides how to handle mismatch
	startupResult := matchResults[0]
	liveMatchesStartup := startupResult.IsMatch(1.0)

	// Step 7: Extract Startup V-F curve (slot 0)
	vfCurveRaw, err := extractVFCurve(hwProfile, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to extract V-F curve from Startup slot: %w", err)
	}

	// Step 8: Convert to user-friendly VFPoint format
	points := convertToVFPoints(vfCurveRaw)

	// Step 9: Additionally check if Startup matches any Profile1-5 slot
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

	return &CurrentStateResult{
		Points:             points,
		MemClkBoostMHz:     memClkBoostMHz,
		PowerLimitPercent:  powerLimitPercent,
		FanMode:            fanModeStr,
		FanSpeedPercent:    fanSpeedPercent,
		Profile:            savedProfile,
		LiveMatchesStartup: liveMatchesStartup,
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
	if curve == nil {
		return nil
	}
	if len(curve.Points) == 0 {
		return []VFPoint{}
	}

	points := make([]VFPoint, len(curve.Points))
	for i, pt := range curve.Points {
		points[i] = VFPoint{
			VoltageMV:   float64(pt.VoltageMV),
			BaseFreqMHz: float64(pt.BaseFreqMHz),
			OffsetMHz:   float64(pt.OffsetMHz),
		}
	}

	return points
}
