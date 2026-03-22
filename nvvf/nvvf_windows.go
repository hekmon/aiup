//go:build windows

package nvvf

import (
	"encoding/binary"
	"fmt"
	"syscall"
	"unsafe"
)

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
// Requires nvapi64.dll and an NVIDIA GPU. Returns an error if NvAPI fails.
//
// Note: Based on LACT project reverse-engineering (issue #936), the struct size
// for ClkDomainsGetInfo is 0x0928 (2344 bytes). Version format: (1 << 16) | structSize.
func ReadNvAPIClkDomains(gpuIndex int) ([]ClkDomainInfo, error) {
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

	// Buffer for clock domain info
	// Based on LACT issue #936: struct size = 0x0928 (2344 bytes)
	bufferSize := 2344
	data := make([]byte, bufferSize)

	// Version format: (1 << 16) | structSize
	// Struct size is 2344 bytes (0x0928)
	version := (1 << 16) | 2344
	binary.LittleEndian.PutUint32(data[0:4], uint32(version))

	// Call ClkDomainsGetInfo(gpuHandle, &data)
	fnPtr := resolve(fnClkDomainsGetInfo)
	if fnPtr == 0 {
		return nil, fmt.Errorf("ClkDomainsGetInfo not available (function pointer is nil)")
	}

	ret := call(fnPtr, gpu, uintptr(unsafe.Pointer(&data[0])))
	if ret != 0 {
		return nil, fmt.Errorf("ClkDomainsGetInfo: 0x%08X", ret)
	}

	// Parse entries
	//
	// NvAPI ClkDomainsGetInfo struct layout (documented by LACT project):
	// https://github.com/ilya-zlobintsev/LACT/issues/936
	//
	// Official layout:
	//   Header: 40 bytes (0x28)
	//     - 0x00: version = MAKE_NVAPI_VERSION(this_struct, 1)
	//     - 0x04: unknown
	//     - 0x08: numDomains (often 0 on Windows, scan entries instead)
	//     - 0x0C-0x27: padding
	//   Entries: 72 bytes each (0x48 stride)
	//     - 0x00: domainId (4 bytes) - NV_GPU_PUBLIC_CLOCK_ID
	//     - 0x04: flags (4 bytes) - bit0=isPresent, bit1=isEditable
	//     - 0x10: offsetMinKHz (4 bytes, signed)
	//     - 0x14: offsetMaxKHz (4 bytes, signed)
	//
	// WINDOWS-SPECIFIC deviations observed on RTX 5090 (Blackwell):
	//   1. Entries start at offset 0x50 (80 bytes), NOT 0x28 (40 bytes)
	//   2. Field order is REVERSED: offsetMaxKHz first, then offsetMinKHz
	//   3. domainId field contains unexpected values (not 0/4/7/8)
	//
	// This implementation uses the empirically verified Windows layout.

	entryOffset := 80 // 0x50 - observed entry start on Windows (RTX 5090)
	entryStride := 72 // 0x48 - stride between entries (matches documentation)
	entrySize := 12   // Parsing first 12 bytes (offsetMaxKHz + offsetMinKHz)

	var domains []ClkDomainInfo
	for offset := entryOffset; offset+entrySize <= len(data); offset += entryStride {
		// Read entry fields
		// Note: Field order is reversed from documentation on Windows
		offsetMaxKHz := int32(binary.LittleEndian.Uint32(data[offset : offset+4]))
		offsetMinKHz := int32(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))

		// Skip entries with all-zero offsets (padding/unused slots)
		if offsetMinKHz == 0 && offsetMaxKHz == 0 {
			continue
		}

		// Infer domain ID from offset ranges
		// The domainId field in the struct contains unexpected values,
		// so we identify domains by their characteristic offset ranges
		var domainID uint32
		switch {
		case offsetMinKHz == -1000000 && offsetMaxKHz == 1000000:
			// ±1000 MHz is typical for GPU core clock
			domainID = 0 // NVAPI_GPU_PUBLIC_CLOCK_GRAPHICS
		case offsetMinKHz == -1000000 && offsetMaxKHz == 3000000:
			// -1000 to +3000 MHz is typical for GDDR6X memory
			domainID = 4 // NVAPI_GPU_PUBLIC_CLOCK_MEMORY
		default:
			// Other domains (processor, video, etc.) have varying ranges
			// Use conservative default
			domainID = 7 // NVAPI_GPU_PUBLIC_CLOCK_PROCESSOR (fallback)
		}

		entry := ClkDomainInfo{
			DomainID:     domainID,
			Flags:        0, // flags field not reliably populated on Windows
			MinOffsetKHz: offsetMinKHz,
			MaxOffsetKHz: offsetMaxKHz,
		}
		domains = append(domains, entry)

		// Safety limit: don't parse more than 10 domains
		if len(domains) >= 10 {
			break
		}
	}

	return domains, nil
}

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

// indexOfByte finds the first occurrence of a byte in a slice.
func indexOfByte(buf []byte, b byte) int {
	for i, v := range buf {
		if v == b {
			return i
		}
	}
	return -1
}
