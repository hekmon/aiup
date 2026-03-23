//go:build linux

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

// GetGPUName retrieves the marketing name of the specified NVIDIA GPU.
//
// This function queries NvAPI for the human-readable GPU name, such as:
//   - "NVIDIA GeForce RTX 5090"
//   - "NVIDIA GeForce RTX 4090"
//   - "NVIDIA RTX A6000"
//
// This is useful for displaying GPU information to users or logging purposes.
//
// Requires libnvidia-api.so.1 and an NVIDIA GPU. Returns an error if NvAPI fails.
//
// ⚠️ UNTESTED - This implementation is based on the Windows version and NvAPI
// documentation, but has not been tested on native Linux hardware. The NvAPI
// function ID and calling convention should be identical, but behavior may vary.
func GetGPUName(gpuIndex int) (string, error) {
	nvapi, err := loadNvAPILinux()
	if err != nil {
		return "", fmt.Errorf("libnvidia-api.so.1 not found (NVIDIA driver required): %w", err)
	}
	defer nvapi.close()

	// Resolve function pointers
	initFn := nvapi.resolve(fnInitialize)
	enumFn := nvapi.resolve(fnEnumGPUs)
	// NvAPI_GPU_GetFullName = 0xCEEE8E9F
	getFullNameFn := nvapi.resolve(0xCEEE8E9F)

	if initFn == nil || enumFn == nil || getFullNameFn == nil {
		return "", fmt.Errorf("NvAPI function not available")
	}

	// Initialize NvAPI
	if s := nvapi.call0(initFn); s != 0 {
		return "", fmt.Errorf("NvAPI_Initialize: %d", s)
	}

	// Enumerate GPUs
	var handles [64]unsafe.Pointer
	var count uint32
	if s := nvapi.call2(enumFn,
		unsafe.Pointer(&handles),
		unsafe.Pointer(&count),
	); s != 0 {
		return "", fmt.Errorf("NvAPI_EnumPhysicalGPUs: %d", s)
	}
	if gpuIndex >= int(count) {
		return "", fmt.Errorf("GPU index %d out of range (found %d GPUs)", gpuIndex, count)
	}
	gpu := handles[gpuIndex]

	// Get GPU name
	var nameBuf [256]byte
	ret := nvapi.call2(getFullNameFn, gpu, unsafe.Pointer(&nameBuf[0]))
	if ret != 0 {
		return "", fmt.Errorf("NvAPI_GPU_GetFullName: %d", ret)
	}

	// Convert to string (null-terminated)
	name := string(nameBuf[:])
	if idx := indexOfByte(nameBuf[:], 0); idx >= 0 {
		name = string(nameBuf[:idx])
	}
	return name, nil
}
