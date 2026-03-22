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

// ---------------------------------------------------------------------------
// Legacy structs (RTX 30/40xx - Pascal, Ampere, Ada Lovelace)
// ---------------------------------------------------------------------------

// Legacy V/F entry: 8 bytes
type nvVfPointsStatusLegacy struct {
	Version  uint32               // 0x00: version = (1 << 16) | 0x0434
	Mask     [4]uint32            // 0x04: 128-bit mask
	Reserved [8]uint32            // 0x14: reserved
	Entries  [128]nvVfEntryLegacy // 0x34: 128 V/F points × 8 bytes each
}

// Legacy V/F control entry: 8 bytes
type nvVfEntryCtrlLegacy struct {
	FreqDeltaKHz int32  // +0x00: frequency offset in kHz (signed)
	VoltageUV    uint32 // +0x04: voltage in µV (unused for offsets)
}

// Legacy V/F control struct: 1076 bytes (0x0434) → version = 0x00010434
type nvVfPointsCtrlLegacy struct {
	Version  uint32                   // 0x00: version = (1 << 16) | 0x0434
	Mask     [4]uint32                // 0x04: 128-bit mask
	Reserved [8]uint32                // 0x14: reserved
	Entries  [128]nvVfEntryCtrlLegacy // 0x34: 128 control points × 8 bytes each
}

// Legacy V/F entry: 8 bytes
type nvVfEntryLegacy struct {
	FreqKHz   uint32 // +0x00: frequency in kHz
	VoltageUV uint32 // +0x04: voltage in µV
}

// Legacy V/F status struct: 1076 bytes (0x0434) → version = 0x00010434
// sizeof = 4 + 16 + 32 + 128*8 = 1076

// ---------------------------------------------------------------------------
// Blackwell structs (RTX 50xx - Blackwell architecture)
// ---------------------------------------------------------------------------

// Blackwell V/F entry: 28 bytes (0x1C stride)
type nvVfEntryBlackwell struct {
	FreqKHz   uint32   // +0x00: frequency in kHz
	VoltageUV uint32   // +0x04: voltage in µV
	Reserved  [20]byte // +0x08: padding to 28 bytes
}

// Blackwell V/F control entry: 72 bytes (0x48 stride)
type nvVfEntryCtrlBlackwell struct {
	FreqDeltaKHz int32    // +0x00: frequency offset in kHz (signed)
	Reserved     [68]byte // +0x04: padding to 72 bytes
}

// Blackwell V/F status struct: 7208 bytes (0x1C28) → version = 0x00011C28
// Layout:
//
//	0x00: version (4 bytes)
//	0x04: mask (16 bytes — 128 bits, set all 0xFF to request all points)
//	0x14: numClocks (4 bytes — set to 15 for GPU core)
//	0x18: reserved (48 bytes padding)
//	0x48: entries (128 × 28 bytes = 3584 bytes)
//	0xE48: trailing reserved (3552 bytes to reach 7208 total)
type nvVfPointsStatusBlackwell struct {
	Version          uint32                  // 0x00: version = (1 << 16) | 0x1C28
	Mask             [4]uint32               // 0x04: 128-bit mask (set all bits to get all 128 points)
	NumClocks        uint32                  // 0x14: number of clock domains (15 for GPU core)
	Reserved         [48]byte                // 0x18: padding to offset 0x48
	Entries          [128]nvVfEntryBlackwell // 0x48: 128 V/F points × 28 bytes each
	TrailingReserved [3552]byte              // 0xE48: padding to reach 7208 bytes total
}

// Blackwell V/F control struct: 9248 bytes (0x2420) → version = 0x00012420
// Layout:
//
//	0x00: version (4 bytes)
//	0x04: mask (16 bytes — set ONLY ONE BIT per call for SetControl)
//	0x14: reserved (12 bytes padding to offset 0x20)
//	0x20: entries (128 × 72 bytes = 9216 bytes)
type nvVfPointsCtrlBlackwell struct {
	Version  uint32                      // 0x00: version = (1 << 16) | 0x2420
	Mask     [4]uint32                   // 0x04: 128-bit mask (single bit for SetControl, all bits for GetControl)
	Reserved [12]byte                    // 0x14: padding to offset 0x20
	Entries  [128]nvVfEntryCtrlBlackwell // 0x20: 128 control points × 72 bytes each
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
// Parser helper functions (internal)
// ---------------------------------------------------------------------------

// parseBlackwellVFPoints extracts VFPoint data from Blackwell-format structs
func parseBlackwellVFPoints(status nvVfPointsStatusBlackwell, ctrl nvVfPointsCtrlBlackwell) []VFPoint {
	var points []VFPoint
	for i := 0; i < 128; i++ {
		base := status.Entries[i]
		off := ctrl.Entries[i]
		// Skip inactive/padding slots
		if base.FreqKHz == 0 && base.VoltageUV == 0 {
			continue
		}
		// Convert from kHz/µV to MHz/mV
		baseMHz := float64(base.FreqKHz) / 1000.0
		deltaMHz := float64(off.FreqDeltaKHz) / 1000.0
		voltageMV := float64(base.VoltageUV) / 1000.0
		points = append(points, VFPoint{
			Index:        i,
			VoltageMV:    round2(voltageMV),
			BaseFreqMHz:  round2(baseMHz),
			OffsetMHz:    round2(deltaMHz),
			EffectiveMHz: round2(baseMHz + deltaMHz),
		})
	}
	return points
}

// parseLegacyVFPoints extracts VFPoint data from legacy-format structs (RTX 30/40xx)
func parseLegacyVFPoints(status nvVfPointsStatusLegacy, ctrl nvVfPointsCtrlLegacy) []VFPoint {
	var points []VFPoint
	for i := 0; i < 128; i++ {
		base := status.Entries[i]
		off := ctrl.Entries[i]
		// Skip inactive/padding slots
		if base.FreqKHz == 0 && base.VoltageUV == 0 {
			continue
		}
		// Convert from kHz/µV to MHz/mV
		baseMHz := float64(base.FreqKHz) / 1000.0
		deltaMHz := float64(off.FreqDeltaKHz) / 1000.0
		voltageMV := float64(base.VoltageUV) / 1000.0
		points = append(points, VFPoint{
			Index:        i,
			VoltageMV:    round2(voltageMV),
			BaseFreqMHz:  round2(baseMHz),
			OffsetMHz:    round2(deltaMHz),
			EffectiveMHz: round2(baseMHz + deltaMHz),
		})
	}
	return points
}

// ---------------------------------------------------------------------------
// Clock Domain types (for ClkDomainsGetInfo)
// ---------------------------------------------------------------------------

// ClkDomainInfo represents information about a GPU clock domain.
//
// Clock domains are different clock regions on the GPU that can be
// independently controlled. This includes:
//   - Graphics clock (GPU core)
//   - Memory clock (VRAM)
//   - Processor clock
//   - Video clock
//
// The MinOffsetKHz and MaxOffsetKHz values indicate the safe operating
// range for frequency offsets in each domain.
type ClkDomainInfo struct {
	DomainID     uint32 // Domain identifier
	Flags        uint32 // Domain flags
	MinOffsetKHz int32  // Minimum allowed offset in kHz
	MaxOffsetKHz int32  // Maximum allowed offset in kHz
}

// nvClkDomainInfoHeader is the header for NvAPI clock domain queries.
//
// Binary layout (16 bytes header + variable entries):
//
//	0x00: version (4 bytes) - must be (1 << 16) | entrySize
//	0x04: size (4 bytes) - total struct size in bytes
//	0x08: numDomains (4 bytes) - output: number of clock domains
//	0x0C: reserved (4 bytes)
//	0x10: entries[] - array of ClkDomainEntry structures
type nvClkDomainInfoHeader struct {
	Version    uint32 // 0x00: version number
	Size       uint32 // 0x04: total struct size
	NumDomains uint32 // 0x08: number of clock domains (output)
	Reserved   uint32 // 0x0C: reserved/padding
}

// nvClkDomainEntry is the binary format for a single clock domain entry (16 bytes).
//
// Binary layout:
//
//	0x00: domainID (4 bytes) - clock domain identifier
//	0x04: flags (4 bytes) - domain flags
//	0x08: minOffsetKHz (4 bytes, signed) - minimum allowed offset
//	0x0C: maxOffsetKHz (4 bytes, signed) - maximum allowed offset
type nvClkDomainEntry struct {
	DomainID     uint32 // 0x00: domain identifier
	Flags        uint32 // 0x04: flags
	MinOffsetKHz int32  // 0x08: minimum allowed offset in kHz
	MaxOffsetKHz int32  // 0x0C: maximum allowed offset in kHz
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
