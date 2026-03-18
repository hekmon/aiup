//go:build windows

package nvvf

import (
	"fmt"
	"syscall"
	"unsafe"
)

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
