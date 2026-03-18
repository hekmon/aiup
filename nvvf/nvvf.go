// Package nvvf provides tools for working with NVIDIA GPU voltage-frequency (V-F) curves.
//
// # NvAPI V-F Curve Access (Windows and Linux)
//
// The ReadNvAPIVF functions read V-F curves directly from the NVIDIA driver using
// undocumented NvAPI calls. This enables precise monitoring of GPU frequency/voltage
// behavior that was previously only accessible through proprietary tools.
//
// # Platform Support
//
//   - Windows: Uses nvapi64.dll (included with NVIDIA display drivers)
//   - Linux: Uses libnvidia-api.so.1 (included with NVIDIA display drivers)
//
// Both platforms use identical NvAPI function IDs and struct layouts.
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

import "math"

// ---------------------------------------------------------------------------
// Shared types
// ---------------------------------------------------------------------------

// VFPoint is a fully resolved voltage/frequency operating point.
type VFPoint struct {
	Index        int
	VoltageMV    float64 // mV
	BaseFreqMHz  float64 // hardware base clock (only from NvAPI, 0 if from cfg only)
	OffsetMHz    float64 // user offset in MHz  (= f0 from cfg blob)
	EffectiveMHz float64 // base + offset       (only exact when base is known)
	OCScanMHz    float64 // OC Scanner ref      (= f2 from cfg blob, 0 if from NvAPI)
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
	fnInitialize   = 0x0150E828
	fnEnumGPUs     = 0xE5AC921F
	fnVfGetStatus  = 0x21537AD4 // base hardware VF curve (no user offsets)
	fnVfGetControl = 0x23F1B133 // user offsets (delta kHz per point)
)

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
// Helpers
// ---------------------------------------------------------------------------

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
