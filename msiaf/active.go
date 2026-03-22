// Package msiaf provides tooling for working with MSI Afterburner profiles and configuration.
package msiaf

import (
	"fmt"
	"math"
)

// ProfileMatchResult contains the result of matching a live V-F curve against a profile curve.
type ProfileMatchResult struct {
	// Slot is the profile slot that was tested (0=Startup, 1-5=Profile1-5).
	Slot int

	// SlotName is a human-readable name for the slot (e.g., "Startup", "Profile1").
	SlotName string

	// MatchedPoints is the number of voltage points that matched within tolerance.
	MatchedPoints int

	// TotalPoints is the total number of points that were compared.
	TotalPoints int

	// MatchConfidence is a value from 0.0 to 1.0 indicating confidence in the match.
	// 1.0 = all points matched within tolerance
	// 0.0 = no points matched
	MatchConfidence float64

	// AvgDeviationMHz is the average absolute deviation across all compared points in MHz.
	// Lower values indicate better match.
	AvgDeviationMHz float64

	// MaxDeviationMHz is the maximum deviation found across all points in MHz.
	// Useful for detecting outlier points.
	MaxDeviationMHz float64
}

// IsMatch returns true if the profile can be considered a match based on confidence threshold.
func (r *ProfileMatchResult) IsMatch(confidenceThreshold float64) bool {
	return r.MatchConfidence >= confidenceThreshold
}

// MatchVFCurve compares a live V-F curve against a profile curve from a .cfg file.
//
// liveFreqs maps voltage (mV) to the effective frequency (MHz) read from the live GPU.
// For nvvf users: build this map from nvvf.VFPoint slices using:
//
//	liveFreqs := make(map[float32]float64)
//	for _, pt := range nvvfPoints {
//	    liveFreqs[float32(pt.VoltageMV)] = pt.BaseFreqMHz
//	}
//
// profileCurve is the parsed V-F curve from a profile's VFCurve field.
//
// toleranceMHz is the acceptable frequency deviation in MHz for a point to be considered matching.
// Typical values: 5-10 MHz (accounts for floating point precision and minor driver variations).
//
// The function computes the effective frequency for each profile point as:
//
//	effectiveFreq = BaseFreqMHz + OffsetMHz (for active points)
//
// Inactive points (BaseFreqMHz = 225.0) are skipped from comparison.
//
// Returns ProfileMatchResult with match statistics, or an error if inputs are invalid.
func MatchVFCurve(liveFreqs map[float32]float64, profileCurve *VFControlCurveInfo, toleranceMHz float64) (*ProfileMatchResult, error) {
	if len(liveFreqs) == 0 {
		return nil, fmt.Errorf("liveFreqs is empty")
	}

	if profileCurve == nil || len(profileCurve.Points) == 0 {
		return nil, fmt.Errorf("profileCurve is nil or empty")
	}

	if toleranceMHz < 0 {
		return nil, fmt.Errorf("toleranceMHz cannot be negative")
	}

	result := &ProfileMatchResult{
		TotalPoints:     0,
		MatchedPoints:   0,
		AvgDeviationMHz: 0.0,
		MaxDeviationMHz: 0.0,
		MatchConfidence: 0.0,
	}

	totalDeviation := 0.0

	// Compare each active point in the profile against live data
	for _, profilePoint := range profileCurve.Points {
		// Skip inactive points (stock behavior, no OC Scanner data)
		if !profilePoint.IsActive {
			continue
		}

		// Compute expected effective frequency from profile
		// Applied frequency = base frequency + user offset
		expectedFreq := float64(profilePoint.BaseFreqMHz + profilePoint.OffsetMHz)

		// Find closest live frequency reading
		liveFreq, found := liveFreqs[profilePoint.VoltageMV]
		if !found {
			// Live data doesn't have this voltage point - skip it
			continue
		}

		result.TotalPoints++

		// Compute deviation
		deviation := math.Abs(expectedFreq - liveFreq)
		totalDeviation += deviation

		if deviation > result.MaxDeviationMHz {
			result.MaxDeviationMHz = deviation
		}

		// Check if within tolerance
		if deviation <= toleranceMHz {
			result.MatchedPoints++
		}
	}

	// Calculate confidence and average deviation
	if result.TotalPoints > 0 {
		result.AvgDeviationMHz = totalDeviation / float64(result.TotalPoints)
		result.MatchConfidence = float64(result.MatchedPoints) / float64(result.TotalPoints)
	} else {
		// No points could be compared
		result.MatchConfidence = 0.0
	}

	return result, nil
}

// MatchProfileAgainstLive compares live GPU state against all profile slots.
//
// liveFreqs maps voltage (mV) to effective frequency (MHz) from live GPU readings.
// Build this from nvvf.VFPoint slices using point.VoltageMV → point.BaseFreqMHz.
//
// hwProfile is the parsed hardware profile containing Startup and Profile1-5 slots.
//
// toleranceMHz is the acceptable frequency deviation in MHz (typically 5-10 MHz).
//
// Returns a slice of ProfileMatchResult, one for each slot:
//   - Index 0: Startup section (currently active in MSI Afterburner)
//   - Index 1-5: Profile1 through Profile5 user slots
//
// The caller can determine which profile is active by finding the result with
// the highest MatchConfidence.
func MatchProfileAgainstLive(liveFreqs map[float32]float64, hwProfile *HardwareProfile, toleranceMHz float64) ([]ProfileMatchResult, error) {
	if hwProfile == nil {
		return nil, fmt.Errorf("hwProfile is nil")
	}

	results := make([]ProfileMatchResult, 6) // Startup + 5 profile slots

	// Slot 0: Startup section
	startupCurve, err := parseProfileVFCurve(&hwProfile.Startup)
	if err != nil {
		results[0] = ProfileMatchResult{
			Slot:            0,
			SlotName:        "Startup",
			MatchConfidence: 0.0,
		}
	} else {
		result, err := MatchVFCurve(liveFreqs, startupCurve, toleranceMHz)
		if err != nil {
			results[0] = ProfileMatchResult{
				Slot:            0,
				SlotName:        "Startup",
				MatchConfidence: 0.0,
			}
		} else {
			results[0] = *result
			results[0].Slot = 0
			results[0].SlotName = "Startup"
		}
	}

	// Slots 1-5: Profile1 through Profile5
	for i := 1; i <= 5; i++ {
		slot := &hwProfile.Profiles[i-1]
		profileCurve, err := parseProfileVFCurve(&slot.ProfileSection)
		if err != nil || slot.IsEmpty {
			results[i] = ProfileMatchResult{
				Slot:            i,
				SlotName:        fmt.Sprintf("Profile%d", i),
				MatchConfidence: 0.0,
			}
			continue
		}

		result, err := MatchVFCurve(liveFreqs, profileCurve, toleranceMHz)
		if err != nil {
			results[i] = ProfileMatchResult{
				Slot:            i,
				SlotName:        fmt.Sprintf("Profile%d", i),
				MatchConfidence: 0.0,
			}
		} else {
			results[i] = *result
			results[i].Slot = i
			results[i].SlotName = fmt.Sprintf("Profile%d", i)
		}
	}

	return results, nil
}

// parseProfileVFCurve extracts and parses the V-F curve from a ProfileSection.
// Returns nil if no VFCurve data is present.
func parseProfileVFCurve(section *ProfileSection) (*VFControlCurveInfo, error) {
	if section == nil || section.VFCurve == nil || len(section.VFCurve) == 0 {
		return nil, fmt.Errorf("no VFCurve data")
	}

	// Convert []byte to hex string
	hexData := fmt.Sprintf("%x", section.VFCurve)

	// Parse using existing V-F curve unmarshaler
	return UnmarshalVFControlCurve(hexData)
}

// FindBestMatch finds the profile slot with the highest match confidence.
// Returns the best result and whether it meets the confidence threshold.
// If multiple slots have the same confidence, the first one (lowest slot number) is returned.
func FindBestMatch(results []ProfileMatchResult, confidenceThreshold float64) (best ProfileMatchResult, isMatch bool) {
	if len(results) == 0 {
		return ProfileMatchResult{}, false
	}

	best = results[0]
	for _, r := range results[1:] {
		if r.MatchConfidence > best.MatchConfidence {
			best = r
		}
	}

	isMatch = best.MatchConfidence >= confidenceThreshold
	return best, isMatch
}

// String returns a human-readable summary of the match result.
func (r *ProfileMatchResult) String() string {
	return fmt.Sprintf("%s: %.0f%% confidence (%d/%d points matched, avg deviation: %.1f MHz, max: %.1f MHz)",
		r.SlotName,
		r.MatchConfidence*100,
		r.MatchedPoints,
		r.TotalPoints,
		r.AvgDeviationMHz,
		r.MaxDeviationMHz)
}

// StringWithoutSlotName returns the match result string without the slot name prefix.
// Useful when the slot name is already displayed separately.
func (r *ProfileMatchResult) StringWithoutSlotName() string {
	return fmt.Sprintf("%.0f%% confidence (%d/%d points matched, avg deviation: %.1f MHz, max: %.1f MHz)",
		r.MatchConfidence*100,
		r.MatchedPoints,
		r.TotalPoints,
		r.AvgDeviationMHz,
		r.MaxDeviationMHz)
}
