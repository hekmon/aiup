//go:build windows

package nvvf

import (
	"fmt"
	"syscall"
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
// Requires nvapi64.dll and an NVIDIA GPU. Returns an error if NvAPI fails.
func GetGPUName(gpuIndex int) (string, error) {
	dll, err := syscall.LoadDLL("nvapi64.dll")
	if err != nil {
		return "", fmt.Errorf("nvapi64.dll not found (NVIDIA driver required): %w", err)
	}
	defer dll.Release()

	qi, err := dll.FindProc("nvapi_QueryInterface")
	if err != nil {
		return "", fmt.Errorf("nvapi_QueryInterface not exported: %w", err)
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
		return "", fmt.Errorf("NvAPI_Initialize: 0x%08X", s)
	}

	// Enumerate GPUs
	var handles [64]uintptr
	var count uint32
	if s := call(resolve(fnEnumGPUs),
		uintptr(unsafe.Pointer(&handles)),
		uintptr(unsafe.Pointer(&count)),
	); s != 0 {
		return "", fmt.Errorf("NvAPI_EnumPhysicalGPUs: 0x%08X", s)
	}
	if gpuIndex >= int(count) {
		return "", fmt.Errorf("GPU index %d out of range (found %d GPUs)", gpuIndex, count)
	}
	gpu := handles[gpuIndex]

	// Get GPU name
	// NvAPI_GPU_GetFullName = 0xCEEE8E9F
	fnGetFullName := resolve(0xCEEE8E9F)
	if fnGetFullName == 0 {
		return "", fmt.Errorf("NvAPI_GPU_GetFullName not available")
	}

	var nameBuf [256]uint8
	ret := call(fnGetFullName, gpu, uintptr(unsafe.Pointer(&nameBuf[0])))
	if ret != 0 {
		return "", fmt.Errorf("NvAPI_GPU_GetFullName: 0x%08X", ret)
	}

	// Convert to string (null-terminated)
	name := string(nameBuf[:])
	if idx := indexOfByte(nameBuf[:], 0); idx >= 0 {
		name = string(nameBuf[:idx])
	}
	return name, nil
}
