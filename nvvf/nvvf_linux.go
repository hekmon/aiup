//go:build linux

// Package nvvf provides tools for working with NVIDIA GPU voltage-frequency (V-F) curves.
package nvvf

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdint.h>
#include <stdlib.h>

typedef void* (*nvapi_query_t)(uint32_t);

static void* load_nvapi() {
	return dlopen("libnvidia-api.so.1", RTLD_LAZY);
}

static void* get_nvapi_func(void* handle, const char* name) {
	return dlsym(handle, name);
}

static void* call_nvapi_query(void* func, uint32_t id) {
	return ((nvapi_query_t)func)(id);
}

static int32_t call_nvapi_func_0(void* func) {
	return ((int32_t(*)(void))func)();
}

static int32_t call_nvapi_func_2(void* func, void* arg1, void* arg2) {
	return ((int32_t(*)(void*, void*))func)(arg1, arg2);
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// nvapiLinux holds the loaded library and query function.
type nvapiLinux struct {
	handle    unsafe.Pointer
	queryFunc unsafe.Pointer
}

// loadNvAPILinux loads the NvAPI library on Linux.
func loadNvAPILinux() (*nvapiLinux, error) {
	handle := C.load_nvapi()
	if handle == nil {
		return nil, fmt.Errorf("libnvidia-api.so.1 not found (NVIDIA driver required)")
	}

	queryFunc := C.get_nvapi_func(handle, C.CString("nvapi_QueryInterface"))
	if queryFunc == nil {
		C.dlclose(handle)
		return nil, fmt.Errorf("nvapi_QueryInterface not exported")
	}

	return &nvapiLinux{
		handle:    handle,
		queryFunc: queryFunc,
	}, nil
}

// close cleans up the loaded library.
func (n *nvapiLinux) close() {
	if n.handle != nil {
		C.dlclose(n.handle)
	}
}

// resolve returns the function pointer for the given NvAPI function ID.
func (n *nvapiLinux) resolve(id uint32) unsafe.Pointer {
	return C.call_nvapi_query(n.queryFunc, C.uint32_t(id))
}

// call0 calls an NvAPI function with 0 arguments.
// Returns int32 cast to uint32 (Linux uses negative error codes).
func (n *nvapiLinux) call0(fn unsafe.Pointer) uint32 {
	return uint32(int32(C.call_nvapi_func_0(fn)))
}

// call2 calls an NvAPI function with 2 arguments.
func (n *nvapiLinux) call2(fn unsafe.Pointer, arg1, arg2 unsafe.Pointer) uint32 {
	return uint32(int32(C.call_nvapi_func_2(fn, arg1, arg2)))
}

// ReadNvAPIVFBlackwell reads the V-F curve for RTX 50xx (Blackwell) GPUs.
//
// Each returned VFPoint contains:
//   - VoltageMV    : voltage step in mV
//   - BaseFreqMHz  : hardware base clock at that voltage
//   - OffsetMHz    : user offset in MHz
//   - EffectiveMHz : base + offset (exact frequency)
//
// Requires libnvidia-api.so.1 and a Blackwell GPU (RTX 50xx series).
// Returns an error on non-Blackwell GPUs or if NvAPI fails.
//
// Credit: Blackwell struct sizes discovered by LACT community reverse-engineering.
// See: https://github.com/ilya-zlobintsev/LACT/issues/936
func ReadNvAPIVFBlackwell(gpuIndex int) ([]VFPoint, error) {
	nvapi, err := loadNvAPILinux()
	if err != nil {
		return nil, fmt.Errorf("libnvidia-api.so.1 not found (NVIDIA driver required): %w", err)
	}
	defer nvapi.close()

	// Resolve function pointers
	initFn := nvapi.resolve(fnInitialize)
	enumFn := nvapi.resolve(fnEnumGPUs)
	statusFn := nvapi.resolve(fnVfGetStatus)
	controlFn := nvapi.resolve(fnVfGetControl)

	if initFn == nil || enumFn == nil || statusFn == nil || controlFn == nil {
		return nil, fmt.Errorf("NvAPI function not available")
	}

	// Initialize NvAPI
	if s := nvapi.call0(initFn); s != 0 {
		return nil, fmt.Errorf("NvAPI_Initialize: %d", s)
	}

	// Enumerate GPUs
	var handles [64]unsafe.Pointer
	var count uint32
	if s := nvapi.call2(enumFn,
		unsafe.Pointer(&handles),
		unsafe.Pointer(&count),
	); s != 0 {
		return nil, fmt.Errorf("NvAPI_EnumPhysicalGPUs: %d", s)
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
	if s := nvapi.call2(statusFn, gpu, unsafe.Pointer(&status)); s != 0 {
		return nil, fmt.Errorf("ClockClientClkVfPointsGetStatus: %d (Blackwell)", s)
	}

	// GetControl
	ctrl.Version = 0x00012420 // (1 << 16) | 0x2420
	for i := range ctrl.Mask {
		ctrl.Mask[i] = 0xFFFFFFFF // All bits for GetControl
	}
	if s := nvapi.call2(controlFn, gpu, unsafe.Pointer(&ctrl)); s != 0 {
		return nil, fmt.Errorf("ClockClientClkVfPointsGetControl: %d (Blackwell)", s)
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
// Requires libnvidia-api.so.1 and a legacy GPU (RTX 30/40xx series).
// Returns an error on Blackwell GPUs or if NvAPI fails.
func ReadNvAPIVFLegacy(gpuIndex int) ([]VFPoint, error) {
	nvapi, err := loadNvAPILinux()
	if err != nil {
		return nil, fmt.Errorf("libnvidia-api.so.1 not found (NVIDIA driver required): %w", err)
	}
	defer nvapi.close()

	// Resolve function pointers
	initFn := nvapi.resolve(fnInitialize)
	enumFn := nvapi.resolve(fnEnumGPUs)
	statusFn := nvapi.resolve(fnVfGetStatus)
	controlFn := nvapi.resolve(fnVfGetControl)

	if initFn == nil || enumFn == nil || statusFn == nil || controlFn == nil {
		return nil, fmt.Errorf("NvAPI function not available")
	}

	// Initialize NvAPI
	if s := nvapi.call0(initFn); s != 0 {
		return nil, fmt.Errorf("NvAPI_Initialize: %d", s)
	}

	// Enumerate GPUs
	var handles [64]unsafe.Pointer
	var count uint32
	if s := nvapi.call2(enumFn,
		unsafe.Pointer(&handles),
		unsafe.Pointer(&count),
	); s != 0 {
		return nil, fmt.Errorf("NvAPI_EnumPhysicalGPUs: %d", s)
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
	if s := nvapi.call2(statusFn, gpu, unsafe.Pointer(&status)); s != 0 {
		return nil, fmt.Errorf("ClockClientClkVfPointsGetStatus: %d (legacy)", s)
	}

	// GetControl
	ctrl.Version = 0x00010434
	for i := range ctrl.Mask {
		ctrl.Mask[i] = 0xFFFFFFFF
	}
	if s := nvapi.call2(controlFn, gpu, unsafe.Pointer(&ctrl)); s != 0 {
		return nil, fmt.Errorf("ClockClientClkVfPointsGetControl: %d (legacy)", s)
	}

	return parseLegacyVFPoints(status, ctrl), nil
}

// SetNvAPIVFPointBlackwell sets a frequency offset for a single V/F curve point on RTX 50xx GPUs.
//
// Parameters:
//   - gpuIndex: GPU index (0-63)
//   - pointIndex: V/F curve point index (0-127)
//   - offsetKHz: frequency offset in kHz (signed, e.g., +50000 for +50 MHz)
//
// The mask is set to a single bit corresponding to the point being modified.
// Setting multiple bits in one call will cause the function to fail.
//
// Returns an error if NvAPI fails or if the GPU is not Blackwell.
func SetNvAPIVFPointBlackwell(gpuIndex, pointIndex, offsetKHz int) error {
	if pointIndex < 0 || pointIndex > 127 {
		return fmt.Errorf("pointIndex %d out of range (0-127)", pointIndex)
	}
	if gpuIndex < 0 || gpuIndex > 63 {
		return fmt.Errorf("gpuIndex %d out of range (0-63)", gpuIndex)
	}

	nvapi, err := loadNvAPILinux()
	if err != nil {
		return fmt.Errorf("libnvidia-api.so.1 not found (NVIDIA driver required): %w", err)
	}
	defer nvapi.close()

	// Resolve function pointers
	initFn := nvapi.resolve(fnInitialize)
	enumFn := nvapi.resolve(fnEnumGPUs)
	setFn := nvapi.resolve(fnVfSetControl)

	if initFn == nil || enumFn == nil || setFn == nil {
		return fmt.Errorf("NvAPI function not available")
	}

	// Initialize NvAPI
	if s := nvapi.call0(initFn); s != 0 {
		return fmt.Errorf("NvAPI_Initialize: %d", s)
	}

	// Enumerate GPUs
	var handles [64]unsafe.Pointer
	var count uint32
	if s := nvapi.call2(enumFn,
		unsafe.Pointer(&handles),
		unsafe.Pointer(&count),
	); s != 0 {
		return fmt.Errorf("NvAPI_EnumPhysicalGPUs: %d", s)
	}
	if gpuIndex >= int(count) {
		return fmt.Errorf("GPU index %d out of range (found %d GPUs)", gpuIndex, count)
	}
	gpu := handles[gpuIndex]

	// Blackwell control struct (9248 bytes)
	var ctrl nvVfPointsCtrlBlackwell

	// Initialize control struct
	ctrl.Version = 0x00012420 // (1 << 16) | 0x2420

	// Set single-bit mask for the target point
	ctrl.Mask[pointIndex/32] = 1 << (pointIndex % 32)

	// Set frequency offset for the target point (in kHz)
	ctrl.Entries[pointIndex].FreqDeltaKHz = int32(offsetKHz)

	// SetControl
	if s := nvapi.call2(setFn, gpu, unsafe.Pointer(&ctrl)); s != 0 {
		return fmt.Errorf("ClockClientClkVfPointsSetControl: %d (Blackwell)", s)
	}

	return nil
}

// SetNvAPIVFLegacy sets a frequency offset for a single V/F curve point on RTX 30/40xx GPUs.
//
// Parameters:
//   - gpuIndex: GPU index (0-63)
//   - pointIndex: V/F curve point index (0-127)
//   - offsetKHz: frequency offset in kHz (signed, e.g., +50000 for +50 MHz)
//
// The mask is set to a single bit corresponding to the point being modified.
// Setting multiple bits in one call will cause the function to fail.
//
// Returns an error if NvAPI fails or if the GPU is not legacy (Pascal/Ampere/Ada).
func SetNvAPIVFLegacy(gpuIndex, pointIndex, offsetKHz int) error {
	if pointIndex < 0 || pointIndex > 127 {
		return fmt.Errorf("pointIndex %d out of range (0-127)", pointIndex)
	}
	if gpuIndex < 0 || gpuIndex > 63 {
		return fmt.Errorf("gpuIndex %d out of range (0-63)", gpuIndex)
	}

	nvapi, err := loadNvAPILinux()
	if err != nil {
		return fmt.Errorf("libnvidia-api.so.1 not found (NVIDIA driver required): %w", err)
	}
	defer nvapi.close()

	// Resolve function pointers
	initFn := nvapi.resolve(fnInitialize)
	enumFn := nvapi.resolve(fnEnumGPUs)
	setFn := nvapi.resolve(fnVfSetControl)

	if initFn == nil || enumFn == nil || setFn == nil {
		return fmt.Errorf("NvAPI function not available")
	}

	// Initialize NvAPI
	if s := nvapi.call0(initFn); s != 0 {
		return fmt.Errorf("NvAPI_Initialize: %d", s)
	}

	// Enumerate GPUs
	var handles [64]unsafe.Pointer
	var count uint32
	if s := nvapi.call2(enumFn,
		unsafe.Pointer(&handles),
		unsafe.Pointer(&count),
	); s != 0 {
		return fmt.Errorf("NvAPI_EnumPhysicalGPUs: %d", s)
	}
	if gpuIndex >= int(count) {
		return fmt.Errorf("GPU index %d out of range (found %d GPUs)", gpuIndex, count)
	}
	gpu := handles[gpuIndex]

	// Legacy control struct (1076 bytes)
	var ctrl nvVfPointsCtrlLegacy

	// Initialize control struct
	ctrl.Version = 0x00010434 // (1 << 16) | 0x0434

	// Set single-bit mask for the target point
	ctrl.Mask[pointIndex/32] = 1 << (pointIndex % 32)

	// Set frequency offset for the target point (in kHz)
	ctrl.Entries[pointIndex].FreqDeltaKHz = int32(offsetKHz)

	// SetControl
	if s := nvapi.call2(setFn, gpu, unsafe.Pointer(&ctrl)); s != 0 {
		return fmt.Errorf("ClockClientClkVfPointsSetControl: %d (legacy)", s)
	}

	return nil
}
