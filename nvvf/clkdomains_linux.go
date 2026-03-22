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

// ReadNvAPIClkDomains reads clock domain information for the specified GPU.
//
// This function queries the NVIDIA driver for clock domain ranges, including
// the minimum and maximum allowed frequency offsets for each domain.
//
// Clock domains typically include:
//   - Graphics clock (GPU core, domain ID ~0)
//   - Memory clock (VRAM, domain ID ~1)
//   - Processor clock
//   - Video encoder/decoder clocks
//
// The returned ClkDomainInfo structs contain:
//   - DomainID: Identifier for the clock domain
//   - Flags: Domain-specific flags
//   - MinOffsetKHz: Minimum allowed frequency offset in kHz
//   - MaxOffsetKHz: Maximum allowed frequency offset in kHz
//
// This is useful for validating overclocking parameters before applying them,
// or for displaying safe operating ranges to users.
//
// Requires libnvidia-api.so.1 and an NVIDIA GPU. Returns an error if NvAPI fails.
//
// ⚠️ UNTESTED - This implementation is based on the Windows version and NvAPI
// documentation, but has not been tested on native Linux hardware. The NvAPI
// function ID and struct layout should be identical, but behavior may vary.
func ReadNvAPIClkDomains(gpuIndex int) ([]ClkDomainInfo, error) {
	nvapi, err := loadNvAPILinux()
	if err != nil {
		return nil, fmt.Errorf("libnvidia-api.so.1 not found (NVIDIA driver required): %w", err)
	}
	defer nvapi.close()

	// Resolve function pointers
	initFn := nvapi.resolve(fnInitialize)
	enumFn := nvapi.resolve(fnEnumGPUs)
	clkInfoFn := nvapi.resolve(fnClkDomainsGetInfo)

	if initFn == nil || enumFn == nil || clkInfoFn == nil {
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

	// Buffer for clock domain info
	// Based on LACT issue #936: struct size = 0x0928 (2344 bytes)
	bufferSize := 2344
	data := make([]byte, bufferSize)

	// Version format: (1 << 16) | structSize
	// Struct size is 2344 bytes (0x0928)
	version := (1 << 16) | 2344
	*(*uint32)(unsafe.Pointer(&data[0])) = uint32(version)

	// Call ClkDomainsGetInfo(gpuHandle, &data)
	ret := nvapi.call2(clkInfoFn, gpu, unsafe.Pointer(&data[0]))
	if ret != 0 {
		return nil, fmt.Errorf("ClkDomainsGetInfo: %d", ret)
	}

	// Parse entries - using same layout as Windows
	// See clkdomains_windows.go for detailed documentation of the Windows-specific deviations
	entryOffset := 80 // 0x50 - observed entry start on Windows (RTX 5090)
	entryStride := 72 // 0x48 - stride between entries
	entrySize := 12   // Parsing first 12 bytes

	var domains []ClkDomainInfo
	for offset := entryOffset; offset+entrySize <= len(data); offset += entryStride {
		offsetMaxKHz := int32(*(*uint32)(unsafe.Pointer(&data[offset])))
		offsetMinKHz := int32(*(*uint32)(unsafe.Pointer(&data[offset+4])))

		// Skip entries with all-zero offsets
		if offsetMinKHz == 0 && offsetMaxKHz == 0 {
			continue
		}

		// Infer domain ID from offset ranges (same as Windows)
		var domain ClkDomain
		switch {
		case offsetMinKHz == -1000000 && offsetMaxKHz == 1000000:
			domain = DomainGraphics
		case offsetMinKHz == -1000000 && offsetMaxKHz == 3000000:
			domain = DomainMemory
		default:
			domain = DomainProcessor
		}

		entry := ClkDomainInfo{
			Domain:       domain,
			Flags:        0,
			MinOffsetKHz: offsetMinKHz,
			MaxOffsetKHz: offsetMaxKHz,
		}
		domains = append(domains, entry)

		if len(domains) >= 10 {
			break
		}
	}

	return domains, nil
}
