// Package nvvf provides tools for working with NVIDIA GPU voltage-frequency (V-F) curves and clock domains.
//
// # NvAPI V-F Curve Access (Windows and Linux)
//
// The ReadNvAPIVF functions read V-F curves directly from the NVIDIA driver using
// undocumented NvAPI calls. This enables precise monitoring of GPU frequency/voltage
// behavior that was previously only accessible through proprietary tools.
//
// # NvAPI GPU Information Access (Windows)
//
// The GetGPUName function retrieves the marketing name of an NVIDIA GPU (e.g., "NVIDIA GeForce RTX 5090").
// This is useful for displaying GPU information to users or for logging purposes.
//
// # NvAPI Clock Domain Information (Windows)
//
// The ReadNvAPIClkDomains function queries clock domain information, including allowed min/max frequency
// offset ranges for each domain (GPU core, memory, etc.). This is useful for validating overclocking
// parameters before applying them.
//
// # Platform Support
//
//   - Windows: Uses nvapi64.dll (included with NVIDIA display drivers)
//   - Linux: Uses libnvidia-api.so.1 (included with NVIDIA display drivers)
//
// Both platforms use identical NvAPI function IDs and struct layouts.
//
// # IMPORTANT - OC Scanner and Hardware Profile Behavior
//
// When OC Scanner or MSI Afterburner hardware profiles are applied, the NVIDIA driver
// modifies its internal boost tables directly at a level below NvAPI. As a result:
//
//   - BaseFreqMHz returned by NvAPI is the ALREADY-MODIFIED curve (includes OC Scanner)
//   - OffsetMHz is ONLY for additional offsets applied via NvAPI SetControl (typically 0)
//   - OC Scanner profiles do NOT appear as offsets in NvAPI readings
//
// This means you CANNOT detect OC Scanner offsets by reading NvAPI alone. The driver
// has already applied OC Scanner's voltage-frequency math internally, so NvAPI sees
// the final result as the "base" hardware curve.
//
// To analyze OC Scanner profiles:
//   1. Read .cfg files using the msiaf package to extract V-F curve data
//   2. Parse the VFCurve blob to see oc_ref (f2) and offset (f0) values
//   3. Compare with NvAPI readings to understand applied modifications
//
// Example at 850 mV with OC Scanner applied:
//
//	.cfg file:     oc_ref=1365 MHz, offset=+952 MHz → effective=2317 MHz
//	NvAPI reads:   BaseFreqMHz=2317 MHz, OffsetMHz=0 MHz, EffectiveMHz=2317 MHz
//
// Both show 2317 MHz effective frequency, confirming OC Scanner is applied correctly.
// The +952 MHz offset is visible in the .cfg file but not in NvAPI's OffsetMHz field.
//
// For complete V-F curve analysis, combine nvvf (NvAPI readings) with msiaf (.cfg parsing).
//
// # Credit & Attribution
//
// The Blackwell (RTX 50xx) struct sizes and function definitions were discovered
// through community reverse-engineering efforts, specifically:
//
//   - LACT Project: https://github.com/ilya-zlobintsev/LACT
//   - Issue #936: https://github.com/ilya-zlobintsev/LACT/issues/936
//   - Researcher: Loong0x00 (GitHub)
//
// Their work on reverse-engineering ASUS GPU Tweak III and documenting the NvAPI
// V-F curve functions made this implementation possible. Special thanks to the
// open-source GPU tools community for sharing knowledge.
//
// # Technical Details
//
// Function IDs used:
//   - 0x21537AD4: ClockClientClkVfPointsGetStatus (read hardware base curve)
//   - 0x23F1B133: ClockClientClkVfPointsGetControl (read user offsets)
//
// Struct sizes differ by GPU generation:
//   - RTX 30/40xx (Pascal/Ampere/Ada): 1076 bytes
//   - RTX 50xx (Blackwell): 7208 bytes (status), 9248 bytes (control)
//
// The implementation provides both generation-specific functions and auto-detection.
//
// # Safety
//
// This package only READs data from the driver - it does not modify GPU settings.
// All functions are non-destructive and safe to use.

package nvvf

import (
	"fmt"
	"math"
)

// ---------------------------------------------------------------------------
// Shared types
// ---------------------------------------------------------------------------

// VFPoint is a fully resolved voltage/frequency operating point read from the NVIDIA driver.
//
// IMPORTANT - OC Scanner and Hardware Profile Behavior:
//
// When OC Scanner or MSI Afterburner hardware profiles are applied, the NVIDIA driver
// modifies its internal boost tables directly. NvAPI reads back these MODIFIED tables,
// NOT the stock hardware curve. This means:
//
//   - BaseFreqMHz  = Current driver state (includes OC Scanner/hardware profile modifications)
//   - OffsetMHz    = ONLY additional offsets applied via NvAPI SetControl (usually 0)
//   - EffectiveMHz = Actual GPU frequency (BaseFreqMHz + OffsetMHz)
//
// OC Scanner profiles do NOT appear in OffsetMHz because they work at a lower level
// than NvAPI SetControl. To see OC Scanner offsets, parse the .cfg file directly
// using the msiaf package.
//
// Example scenario (OC Scanner applied at 850 mV):
//
//	.cfg file:     oc_ref=1365 MHz, offset=+952 MHz → effective=2317 MHz
//	NvAPI reads:   BaseFreqMHz=2317 MHz, OffsetMHz=0, EffectiveMHz=2317 MHz
//
// The driver has already applied OC Scanner's math internally, so NvAPI sees the
// final result as the "base" curve with no additional offsets.
type VFPoint struct {
	Index        int     // Point index (0-127)
	VoltageMV    float64 // Voltage in millivolts
	BaseFreqMHz  float64 // Current driver frequency at this voltage (includes OC Scanner/hardware profile modifications)
	OffsetMHz    float64 // Additional offsets applied via NvAPI SetControl only (0 if using OC Scanner or .cfg profiles)
	EffectiveMHz float64 // Actual GPU frequency = BaseFreqMHz + OffsetMHz
	OCScanMHz    float64 // OC Scanner reference frequency (only from .cfg parsing, 0 from NvAPI)
}

// ClkDomain represents a GPU clock domain identifier.
//
// Clock domains are different clock regions on the GPU that can be
// independently controlled. This type provides type safety and
// human-readable string representation for domain IDs.
type ClkDomain uint32

// Well-known clock domain constants
const (
	DomainGraphics  ClkDomain = 0 // GPU core clock
	DomainMemory    ClkDomain = 4 // VRAM clock
	DomainProcessor ClkDomain = 7 // Processor clock
	DomainVideo     ClkDomain = 8 // Video encoder/decoder clock
)

// String implements fmt.Stringer, returning a human-readable name for the domain.
func (d ClkDomain) String() string {
	switch d {
	case DomainGraphics:
		return "Graphics Clock (GPU Core)"
	case DomainMemory:
		return "Memory Clock (VRAM)"
	case DomainProcessor:
		return "Processor Clock"
	case DomainVideo:
		return "Video Clock"
	default:
		return fmt.Sprintf("Unknown Domain (ID: %d)", d)
	}
}

// ---------------------------------------------------------------------------
// NvAPI function IDs
// ---------------------------------------------------------------------------

const (
	fnInitialize        = 0x0150E828
	fnEnumGPUs          = 0xE5AC921F
	fnVfGetStatus       = 0x21537AD4 // base hardware VF curve (no user offsets)
	fnVfGetControl      = 0x23F1B133 // user offsets (delta kHz per point)
	fnClkDomainsGetInfo = 0x64B43A6A // clock domain ranges (allowed offset min/max)
)

// ---------------------------------------------------------------------------
// High-level auto-detect function (shared)
// ---------------------------------------------------------------------------

// ReadNvAPIVF reads the complete, exact VF curve for GPU at index gpuIndex.
//
// This is the high-level convenience function that auto-detects the GPU
// generation and calls the appropriate low-level function:
//   - ReadNvAPIVFBlackwell() for RTX 50xx (Blackwell)
//   - ReadNvAPIVFLegacy() for RTX 30/40xx (Pascal/Ampere/Ada)
//
// Each returned VFPoint contains:
//   - VoltageMV    : voltage step in mV
//   - BaseFreqMHz  : hardware base clock at that voltage (driver pstate table)
//   - OffsetMHz    : user offset in MHz (same as f0 in the .cfg blob)
//   - EffectiveMHz : base + offset = actual applied frequency
//
// For direct generation-specific access, use:
//   - ReadNvAPIVFBlackwell(gpuIndex) for RTX 50xx
//   - ReadNvAPIVFLegacy(gpuIndex) for RTX 30/40xx
func ReadNvAPIVF(gpuIndex int) ([]VFPoint, error) {
	// Auto-detect: Try Blackwell first (newer GPUs), then legacy
	points, err := ReadNvAPIVFBlackwell(gpuIndex)
	if err == nil {
		return points, nil
	}

	// Blackwell failed, try legacy
	points, err = ReadNvAPIVFLegacy(gpuIndex)
	if err != nil {
		return nil, fmt.Errorf("NvAPI auto-detect failed (tried Blackwell and legacy): %w", err)
	}

	return points, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// ---------------------------------------------------------------------------
// VFPoint helper functions
// ---------------------------------------------------------------------------

// VFRange returns the minimum and maximum frequencies from a slice of VFPoints.
// Returns (0, 0) if the slice is empty.
func VFRange(points []VFPoint) (minFreq, maxFreq float64) {
	if len(points) == 0 {
		return 0, 0
	}

	minFreq = points[0].BaseFreqMHz
	maxFreq = points[0].BaseFreqMHz

	for _, pt := range points[1:] {
		if pt.BaseFreqMHz < minFreq {
			minFreq = pt.BaseFreqMHz
		}
		if pt.BaseFreqMHz > maxFreq {
			maxFreq = pt.BaseFreqMHz
		}
	}
	return minFreq, maxFreq
}

// VFMinVoltage returns the minimum voltage from a slice of VFPoints.
// Returns 0 if the slice is empty.
func VFMinVoltage(points []VFPoint) float64 {
	if len(points) == 0 {
		return 0
	}
	minVal := points[0].VoltageMV
	for _, pt := range points[1:] {
		if pt.VoltageMV < minVal {
			minVal = pt.VoltageMV
		}
	}
	return minVal
}

// VFMaxVoltage returns the maximum voltage from a slice of VFPoints.
// Returns 0 if the slice is empty.
func VFMaxVoltage(points []VFPoint) float64 {
	if len(points) == 0 {
		return 0
	}
	maxVal := points[0].VoltageMV
	for _, pt := range points[1:] {
		if pt.VoltageMV > maxVal {
			maxVal = pt.VoltageMV
		}
	}
	return maxVal
}
