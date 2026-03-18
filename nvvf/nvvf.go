//go:build windows

// Package nvvf provides tools for working with NVIDIA GPU voltage-frequency (V-F) curves.
//
// # NvAPI V-F Curve Access (Windows Only)
//
// The ReadNvAPIVF functions read V-F curves directly from the NVIDIA driver using
// undocumented NvAPI calls. This enables precise monitoring of GPU frequency/voltage
// behavior that was previously only accessible through proprietary tools.
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
	"syscall"
	"unsafe"
)

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
// High-level auto-detect function
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
//   - EffectiveMHz : base + offset = exact applied frequency
//
// Requires nvapi64.dll (installed with any NVIDIA display driver).
// Windows x64 only.
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
// Generation-specific functions (exported, completely separate implementations)
// ---------------------------------------------------------------------------

// ReadNvAPIVFBlackwell reads the V-F curve for RTX 50xx (Blackwell) GPUs.
//
// Each returned VFPoint contains:
//   - VoltageMV    : voltage step in mV
//   - BaseFreqMHz  : hardware base clock at that voltage
//   - OffsetMHz    : user offset in MHz
//   - EffectiveMHz : base + offset (exact frequency)
//
// Requires nvapi64.dll and a Blackwell GPU (RTX 50xx series).
// Returns an error on non-Blackwell GPUs or if NvAPI fails.
//
// Credit: Blackwell struct sizes discovered by LACT community reverse-engineering.
// See: https://github.com/ilya-zlobintsev/LACT/issues/936
func ReadNvAPIVFBlackwell(gpuIndex int) ([]VFPoint, error) {
	dll, err := syscall.LoadDLL("nvapi64.dll")
	if err != nil {
		return nil, fmt.Errorf("nvapi64.dll not found (NVIDIA driver required): %w", err)
	}
	defer dll.Release()

	qi, err := dll.FindProc("nvapi_QueryInterface")
	if err != nil {
		return nil, fmt.Errorf("nvapi_QueryInterface not exported: %w", err)
	}

	resolve := func(id uint32) uintptr {
		addr, _, _ := qi.Call(uintptr(id))
		return addr
	}
	call := func(fn uintptr, args ...uintptr) uint32 {
		r, _, _ := syscall.SyscallN(fn, args...)
		return uint32(r)
	}

	// Initialize NvAPI
	if s := call(resolve(fnInitialize)); s != 0 {
		return nil, fmt.Errorf("NvAPI_Initialize: 0x%08X", s)
	}

	// Enumerate GPUs
	var handles [64]uintptr
	var count uint32
	if s := call(resolve(fnEnumGPUs),
		uintptr(unsafe.Pointer(&handles)),
		uintptr(unsafe.Pointer(&count)),
	); s != 0 {
		return nil, fmt.Errorf("NvAPI_EnumPhysicalGPUs: 0x%08X", s)
	}
	if gpuIndex >= int(count) {
		return nil, fmt.Errorf("GPU index %d out of range (found %d GPUs)", gpuIndex, count)
	}
	gpu := handles[gpuIndex]

	// Blackwell structs (RTX 50xx)
	var status nvVfPointsStatusBlackwell
	var ctrl nvVfPointsCtrlBlackwell

	// Initialize Blackwell status struct
	status.Version = 0x00011C28 // (1 << 16) | 0x1C28
	for i := range status.Mask {
		status.Mask[i] = 0xFFFFFFFF // Request all 128 points
	}
	status.NumClocks = 15 // GPU core clock domain

	// GetStatus
	if s := call(resolve(fnVfGetStatus), gpu, uintptr(unsafe.Pointer(&status))); s != 0 {
		return nil, fmt.Errorf("ClockClientClkVfPointsGetStatus: 0x%08X (Blackwell)", s)
	}

	// GetControl
	ctrl.Version = 0x00012420 // (1 << 16) | 0x2420
	for i := range ctrl.Mask {
		ctrl.Mask[i] = 0xFFFFFFFF // All bits for GetControl
	}
	if s := call(resolve(fnVfGetControl), gpu, uintptr(unsafe.Pointer(&ctrl))); s != 0 {
		return nil, fmt.Errorf("ClockClientClkVfPointsGetControl: 0x%08X (Blackwell)", s)
	}

	return parseBlackwellVFPoints(status, ctrl), nil
}

// ReadNvAPIVFLegacy reads the V-F curve for RTX 30/40xx (Pascal/Ampere/Ada) GPUs.
//
// Each returned VFPoint contains:
//   - VoltageMV    : voltage step in mV
//   - BaseFreqMHz  : hardware base clock at that voltage
//   - OffsetMHz    : user offset in MHz
//   - EffectiveMHz : base + offset (exact frequency)
//
// Requires nvapi64.dll and a legacy GPU (RTX 30/40xx series).
// Returns an error on Blackwell GPUs or if NvAPI fails.
func ReadNvAPIVFLegacy(gpuIndex int) ([]VFPoint, error) {
	dll, err := syscall.LoadDLL("nvapi64.dll")
	if err != nil {
		return nil, fmt.Errorf("nvapi64.dll not found (NVIDIA driver required): %w", err)
	}
	defer dll.Release()

	qi, err := dll.FindProc("nvapi_QueryInterface")
	if err != nil {
		return nil, fmt.Errorf("nvapi_QueryInterface not exported: %w", err)
	}

	resolve := func(id uint32) uintptr {
		addr, _, _ := qi.Call(uintptr(id))
		return addr
	}
	call := func(fn uintptr, args ...uintptr) uint32 {
		r, _, _ := syscall.SyscallN(fn, args...)
		return uint32(r)
	}

	// Initialize NvAPI
	if s := call(resolve(fnInitialize)); s != 0 {
		return nil, fmt.Errorf("NvAPI_Initialize: 0x%08X", s)
	}

	// Enumerate GPUs
	var handles [64]uintptr
	var count uint32
	if s := call(resolve(fnEnumGPUs),
		uintptr(unsafe.Pointer(&handles)),
		uintptr(unsafe.Pointer(&count)),
	); s != 0 {
		return nil, fmt.Errorf("NvAPI_EnumPhysicalGPUs: 0x%08X", s)
	}
	if gpuIndex >= int(count) {
		return nil, fmt.Errorf("GPU index %d out of range (found %d GPUs)", gpuIndex, count)
	}
	gpu := handles[gpuIndex]

	// Legacy structs (RTX 30/40xx)
	var status nvVfPointsStatusLegacy
	var ctrl nvVfPointsCtrlLegacy

	// Initialize legacy status struct
	status.Version = 0x00010434 // (1 << 16) | 0x0434
	for i := range status.Mask {
		status.Mask[i] = 0xFFFFFFFF
	}

	// GetStatus
	if s := call(resolve(fnVfGetStatus), gpu, uintptr(unsafe.Pointer(&status))); s != 0 {
		return nil, fmt.Errorf("ClockClientClkVfPointsGetStatus: 0x%08X (legacy)", s)
	}

	// GetControl
	ctrl.Version = 0x00010434
	for i := range ctrl.Mask {
		ctrl.Mask[i] = 0xFFFFFFFF
	}
	if s := call(resolve(fnVfGetControl), gpu, uintptr(unsafe.Pointer(&ctrl))); s != 0 {
		return nil, fmt.Errorf("ClockClientClkVfPointsGetControl: 0x%08X (legacy)", s)
	}

	return parseLegacyVFPoints(status, ctrl), nil
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
// Helpers
// ---------------------------------------------------------------------------

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
