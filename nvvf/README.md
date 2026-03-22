# nvvf - NVIDIA GPU Information Package

This package provides tools for reading NVIDIA GPU information directly from the graphics driver using undocumented NvAPI calls.

## Purpose

The `nvvf` package provides comprehensive access to NVIDIA GPU data on Windows and Linux:

- **V-F Curve Data**: Live voltage-frequency curves from the driver's pstate table
- **GPU Marketing Name**: Human-readable GPU names (e.g., "NVIDIA GeForce RTX 5090")
- **Clock Domain Ranges**: Safe overclocking limits for each clock domain
- **VF Analysis Helpers**: Convenient functions for analyzing V-F curve data

⚠️ **Important:** OC Scanner and MSI Afterburner hardware profiles modify the driver's internal boost tables directly. NvAPI reads back these **already-modified** tables, not the stock hardware curve. See the "OC Scanner Behavior" section below.

## Attribution

**Critical:** The Blackwell (RTX 50xx) struct sizes and function definitions were discovered through community reverse-engineering:

- **LACT Project:** https://github.com/ilya-zlobintsev/LACT
- **Issue #936:** https://github.com/ilya-zlobintsev/LACT/issues/936
- **Researcher:** Loong0x00 (GitHub)

Their work reverse-engineering ASUS GPU Tweak III and documenting NvAPI V-F curve functions made this implementation possible.

## Platform Support

| Platform | Library | Status |
|----------|---------|--------|
| **Windows x64** | `nvapi64.dll` | ✅ Fully tested (including via WSL interop) |
| **Linux x64 (native)** | `libnvidia-api.so.1` | ✅ Implemented (requires native Linux with NVIDIA driver) |

**WSL Users:** Use the Windows build (`nvvf.exe`) directly from WSL. WSL's interop layer allows Windows binaries to access Windows NVIDIA drivers. The Linux build cannot be used on WSL because `libnvidia-api.so.1` is not available in WSL's Linux environment.

Both platforms use **identical NvAPI function IDs and struct layouts**.

## Package API

### Get GPU Marketing Name

Retrieve the human-readable marketing name of an NVIDIA GPU:

```go
import "github.com/hekmon/aiup/nvvf"

// Get GPU name (e.g., "NVIDIA GeForce RTX 5090")
name, err := nvvf.GetGPUName(0) // GPU index 0
if err != nil {
    panic(err)
}
fmt.Printf("GPU 0: %s\n", name)
```

### Read Clock Domain Ranges

Query safe overclocking ranges for each clock domain:

```go
import "github.com/hekmon/aiup/nvvf"

// Get clock domain information
domains, err := nvvf.ReadNvAPIClkDomains(0)
if err != nil {
    panic(err)
}

for _, d := range domains {
    // d.Domain automatically formats as string (e.g., "Graphics Clock (GPU Core)")
    minMHz := float64(d.MinOffsetKHz) / 1000.0
    maxMHz := float64(d.MaxOffsetKHz) / 1000.0
    fmt.Printf("%s: %+.0f MHz to %+.0f MHz\n", d.Domain, minMHz, maxMHz)
}
```

**ClkDomain Type**: The `Domain` field uses the `ClkDomain` type which implements `fmt.Stringer`:

```go
// Well-known domain constants
nvvf.DomainGraphics  // GPU core clock (ID: 0)
nvvf.DomainMemory    // VRAM clock (ID: 4)
nvvf.DomainProcessor // Processor clock (ID: 7)
nvvf.DomainVideo     // Video clock (ID: 8)

// Automatic string conversion
fmt.Println(nvvf.DomainGraphics) // "Graphics Clock (GPU Core)"
```

### V-F Curve Analysis Helpers

Convenience functions for analyzing V-F curve data:

```go
import "github.com/hekmon/aiup/nvvf"

points, err := nvvf.ReadNvAPIVF(0)
if err != nil {
    panic(err)
}

// Get frequency range
minFreq, maxFreq := nvvf.VFRange(points)
fmt.Printf("Frequency: %.0f - %.0f MHz\n", minFreq, maxFreq)

// Get voltage range
minVolt := nvvf.VFMinVoltage(points)
maxVolt := nvvf.VFMaxVoltage(points)
fmt.Printf("Voltage: %.0f - %.0f mV\n", minVolt, maxVolt)
```

### Auto-Detect GPU Generation

```go
import "github.com/hekmon/aiup/nvvf"

// Automatically detects GPU generation (RTX 30/40/50xx)
points, err := nvvf.ReadNvAPIVF(0) // GPU index 0
if err != nil {
    panic(err)
}

for _, p := range points {
    fmt.Printf("%4.0f mV: base %6.0f MHz + offset %+6.0f MHz = %6.0f MHz\n",
        p.VoltageMV, p.BaseFreqMHz, p.OffsetMHz, p.EffectiveMHz)
}
```

### Direct Generation-Specific Access

When you know the GPU generation (e.g., from device ID lookup):

```go
// For RTX 50xx (Blackwell)
points, err := nvvf.ReadNvAPIVFBlackwell(0)

// For RTX 30/40xx (Pascal/Ampere/Ada)
points, err := nvvf.ReadNvAPIVFLegacy(0)
```

### VFPoint Structure

```go
type VFPoint struct {
    Index        int     // Point index (0-127)
    VoltageMV    float64 // Voltage in millivolts
    BaseFreqMHz  float64 // Current driver frequency (includes OC Scanner/hardware profile modifications)
    OffsetMHz    float64 // Additional offsets via NvAPI SetControl only (0 if using OC Scanner)
    EffectiveMHz float64 // Actual GPU frequency = BaseFreqMHz + OffsetMHz
    OCScanMHz    float64 // OC Scanner reference (only from .cfg parsing, 0 from NvAPI)
}
```

⚠️ **Note:** `BaseFreqMHz` is NOT the stock hardware curve when OC Scanner or hardware profiles are applied - it's the **final applied frequency**. OC Scanner offsets do NOT appear in `OffsetMHz` because they're applied at a lower level than NvAPI SetControl.

## Requirements

### Windows
- **Windows x64** operating system
- **NVIDIA display driver** installed (provides `nvapi64.dll`)
- **NVIDIA GPU** present in the system

### Linux
- **Linux x64** operating system (native, not WSL)
- **NVIDIA display driver** installed (provides `libnvidia-api.so.1`)
- **NVIDIA GPU** present in the system

**WSL Users:** The Windows build (`nvvf.exe`) can be run from WSL via interop. The Linux build requires native Linux with `libnvidia-api.so.1`.

## Command-Line Tool

For quick command-line access to V-F curve data, see the [`cmd/nvvf`](../cmd/nvvf/) tool.

## Package Structure

The package is organized by feature with platform-specific implementations:

```
nvvf/
├── nvvf.go                      # Core: VFPoint, ClkDomain type, constants, helpers
├── vf.go                        # V-F curve structs (legacy + Blackwell) + parsers
├── vf_windows.go                # Windows V-F implementations
├── vf_linux.go                  # Linux V-F implementations
├── clkdomains.go                # ClkDomainInfo struct
├── clkdomains_windows.go        # Windows clock domain implementation
├── clkdomains_linux.go          # Linux clock domain implementation (⚠️ UNTESTED)
├── gpuname.go                   # indexOfByte() helper
├── gpuname_windows.go           # Windows GetGPUName() implementation
├── gpuname_linux.go             # Linux GetGPUName() implementation (⚠️ UNTESTED)
├── nvapi_linux.go               # Linux NvAPI loading helpers (cgo)
└── README.md                    # This documentation
```

### File Organization Principles

| Principle | Explanation |
|-----------|-------------|
| **Feature-based** | Each feature (V-F curves, clock domains, GPU name) has its own files |
| **Platform separation** | Windows and Linux implementations are in separate files with `//go:build` tags |
| **Shared types** | Public types (`VFPoint`, `ClkDomainInfo`) are in common files |
| **Shared helpers** | Internal helpers used across features are in feature-specific common files |
| **Linux cgo helpers** | `nvapi_linux.go` contains cgo loading code (Windows uses `syscall` directly) |

### Platform Support Notes

| Feature | Windows | Linux |
|---------|---------|-------|
| V-F Curve Reading | ✅ Tested | ✅ Supported (same NvAPI functions) |
| Clock Domains | ✅ Tested | ⚠️ UNTESTED (implementation based on Windows) |
| GPU Name | ✅ Tested | ⚠️ UNTESTED (implementation based on Windows) |

## Technical Details

### NvAPI Functions Used

| Function ID | Name | Purpose |
|-------------|------|---------|
| `0x21537AD4` | `ClockClientClkVfPointsGetStatus` | Read hardware base V-F curve |
| `0x23F1B133` | `ClockClientClkVfPointsGetControl` | Read user frequency offsets |
| `0x64B43A6A` | `ClockClientClkDomainsGetInfo` | Query clock domain offset ranges |
| `0xCEEE8E9F` | `NvAPI_GPU_GetFullName` | Get GPU marketing name |

### Struct Sizes by Generation

| GPU Generation | Status Struct | Control Struct | Entry Stride |
|----------------|---------------|----------------|--------------|
| **RTX 30/40xx** (Pascal/Ampere/Ada) | 1076 bytes | 1076 bytes | 8 bytes |
| **RTX 50xx** (Blackwell) | 7208 bytes | 9248 bytes | 28/72 bytes |

The implementation auto-detects which struct sizes to use based on GPU compatibility.

---

## NvAPI Clock Domain Information (ClkDomainsGetInfo)

The `ReadNvAPIClkDomains` function queries clock domain information from the NVIDIA driver, including allowed min/max frequency offset ranges for each domain. This is useful for validating overclocking parameters before applying them.

### Function Signature

```c
NvAPI_Status NvAPI_GPU_ClkDomainsGetInfo(
    NvPhysicalGpuHandle                   hPhysicalGpu,  // [in]
    NV_GPU_CLOCK_CLIENT_CLK_DOMAINS_INFO *pInfo          // [out]
);
```

### Documented Struct Layout (Linux / Reference)

**Source:** LACT project reverse-engineering (https://github.com/ilya-zlobintsev/LACT/issues/936)

```c
// Per-domain entry — sizeof = 0x48 (72 bytes)
typedef struct _NV_GPU_CLOCK_CLIENT_CLK_DOMAIN_ENTRY {
    NvU32  domainId;      // +0x00  clock domain ID (see NV_GPU_PUBLIC_CLOCK_ID below)
    NvU32  flags;         // +0x04  bit0 = isPresent, bit1 = isEditable  
    NvU32  unk08;         // +0x08  unknown/reserved
    NvU32  unk0C;         // +0x0C  unknown/reserved
    NvS32  offsetMinKHz;  // +0x10  minimum allowed offset (kHz, usually negative)
    NvS32  offsetMaxKHz;  // +0x14  maximum allowed offset (kHz, usually positive)
    NvU8   pad[0x30];     // +0x18  padding to 72 bytes total
} NV_GPU_CLOCK_CLIENT_CLK_DOMAIN_ENTRY;

// Main output struct — total sizeof = 0x0928 (2344 bytes)
typedef struct _NV_GPU_CLOCK_CLIENT_CLK_DOMAINS_INFO_V1 {
    NvU32  version;       // +0x00  MAKE_NVAPI_VERSION(this_struct, 1) = (1 << 16) | 0x0928
    NvU32  unk04;         // +0x04  unknown
    NvU32  numDomains;    // +0x08  number of active/present domains
    NvU32  unk0C;         // +0x0C  unknown/reserved
    NvU8   pad[0x18];     // +0x10  header padding → header total = 0x28 (40 bytes)
    NV_GPU_CLOCK_CLIENT_CLK_DOMAIN_ENTRY domains[32]; // +0x28 → 32 × 72 = 2304 bytes
} NV_GPU_CLOCK_CLIENT_CLK_DOMAINS_INFO_V1;

// Layout verification: 40 (header) + 32 × 72 (entries) = 2344 = 0x0928 ✓
```

### Clock Domain IDs (NV_GPU_PUBLIC_CLOCK_ID)

| Value | Constant | Description | Typical Offset Range |
|-------|----------|-------------|---------------------|
| `0` | NVAPI_GPU_PUBLIC_CLOCK_GRAPHICS | GPU core clock | ±1000 MHz (±1000000 kHz) |
| `4` | NVAPI_GPU_PUBLIC_CLOCK_MEMORY | Memory clock (VRAM) | -1000 to +3000 MHz |
| `7` | NVAPI_GPU_PUBLIC_CLOCK_PROCESSOR | Processor clock | Varies |
| `8` | NVAPI_GPU_PUBLIC_CLOCK_VIDEO | Video encoder/decoder | Varies |

### ⚠️ Windows-Specific Deviations (RTX 5090 Blackwell)

**Tested on:** NVIDIA GeForce RTX 5090, Windows 11, WSL2, driver 572.xx+

The Windows implementation deviates from the documented Linux structure:

| Aspect | Documented (Linux) | Observed (Windows) |
|--------|-------------------|-------------------|
| **Entry start offset** | `0x28` (40 bytes) | `0x50` (80 bytes) ❌ |
| **Field order** | `offsetMinKHz` at +0x10, `offsetMaxKHz` at +0x14 | **REVERSED**: `offsetMaxKHz` first, then `offsetMinKHz` ❌ |
| **domainId field** | Contains 0, 4, 7, 8 | Contains unexpected values (32256, 33663) ❌ |
| **numDomains** | Correctly populated | Always 0 (must scan entries) ❌ |
| **flags field** | bit0=isPresent, bit1=isEditable | Not reliably populated |

**Working Windows Implementation:**

The Go implementation in `nvvf_windows.go` uses empirically verified layout:

```go
entryOffset := 80  // 0x50 - observed entry start on Windows (NOT 0x28!)
entryStride := 72  // 0x48 - matches documentation ✓
entrySize := 12    // Parsing only offset fields

for offset := entryOffset; offset+entrySize <= len(data); offset += entryStride {
    // Field order is REVERSED from documentation on Windows
    offsetMaxKHz := int32(binary.LittleEndian.Uint32(data[offset : offset+4]))
    offsetMinKHz := int32(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
    
    // Skip zero entries
    if offsetMinKHz == 0 && offsetMaxKHz == 0 {
        continue
    }
    
    // Domain ID must be inferred from offset ranges (domainId field unreliable)
    var domainID uint32
    if offsetMinKHz == -1000000 && offsetMaxKHz == 1000000 {
        domainID = 0 // Graphics
    } else if offsetMinKHz == -1000000 && offsetMaxKHz == 3000000 {
        domainID = 4 // Memory
    } else {
        domainID = 7 // Processor (fallback)
    }
}
```

### Linux Implementation Notes

For Linux implementations, use the documented struct layout:

- Entry start offset: `0x28` (40 bytes)
- Standard field order: `offsetMinKHz` then `offsetMaxKHz`
- `domainId` field should contain valid values (0, 4, 7, 8)
- `numDomains` may be correctly populated (unlike Windows)

### Older GPU Generation Considerations

**Pascal (GTX 10xx), Ampere (RTX 30xx), Ada (RTX 40xx):**

1. **Struct size may differ** — Test with multiple sizes:
   - Try 0x0928 (2344) first (Blackwell/RTX 50xx)
   - Try 0x0434 (1076) if that fails (legacy format)
   - Try 0x0200 (512) as fallback

2. **Entry count may vary** — Older GPUs may have fewer clock domains:
   - RTX 50xx: 2+ domains (graphics, memory)
   - RTX 30/40xx: May have 2-4 domains
   - GTX 10xx: May have only 1-2 domains

3. **Offset ranges differ by architecture:**
   - **RTX 50xx (Blackwell)**: Graphics ±1000 MHz, Memory -1000/+3000 MHz
   - **RTX 40xx (Ada)**: Graphics ±1000 MHz, Memory -500/+2000 MHz (typical)
   - **RTX 30xx (Ampere)**: Graphics ±500 MHz, Memory -500/+1500 MHz (typical)
   - **GTX 10xx (Pascal)**: Graphics ±200 MHz (more limited)

4. **Function availability:**
   - RTX 50xx: ✅ Confirmed working
   - RTX 30/40xx: ⚠️ Likely available, untested
   - GTX 10xx: ❌ May not be supported (function returns error)

### Error Handling

| Platform | Success | Error Format | Example |
|----------|---------|--------------|---------|
| **Windows** | `0x00000000` | `0x%08X` (unsigned hex) | `0x80001003` |
| **Linux** | `0` | `%d` (signed decimal) | `-9` |

**Common Error Codes:**

| Code (Windows) | Code (Linux) | Name | Meaning |
|----------------|--------------|------|---------|
| `0x00000000` | `0` | NVAPI_OK | Success |
| `0xFFFFFFF7` | `-9` | INCOMPATIBLE_STRUCT_VERSION | Wrong struct size/version |
| `0xFFFFFFFF` | `-1` | NVAPI_ERROR | Generic failure |
| `0xCD57D4D1` | `-3` | INVALID_ARGUMENT | Bad parameters |
| `0x341543F1` | `-15` | NVIDIA_DEVICE_NOT_FOUND | GPU not supported |

### Known Working Configurations

| GPU | Platform | Driver | Status | Notes |
|-----|----------|--------|--------|-------|
| RTX 5090 | Windows (WSL2) | 572.xx+ | ✅ Working | Entries at 0x50, reversed field order |
| RTX 5090 | Linux (native) | 590.48.01 | ✅ Working (LACT) | Standard documented layout |
| RTX 40xx | Untested | - | ❓ Unknown | Likely similar to 50xx |
| RTX 30xx | Untested | - | ❓ Unknown | May use legacy struct |
| GTX 10xx | Untested | - | ❓ Unknown | Function may not exist |

### References

- **LACT Project:** https://github.com/ilya-zlobintsev/LACT
- **LACT Issue #936:** https://github.com/ilya-zlobintsev/LACT/issues/936 (primary source)
- **NvAPI Function IDs:** nvapi-rs, nvapioc, vertminer-nvidia (community definitions)
- **Discovery Method:** Reverse engineering of ASUS GPU Tweak III `Vender.dll` using Ghidra

### Data Returned

Each V/F point contains:

| Field | Description | Accuracy |
|-------|-------------|----------|
| `VoltageMV` | Voltage step in millivolts | Exact |
| `BaseFreqMHz` | Hardware base frequency at that voltage | Exact (from driver pstate table) |
| `OffsetMHz` | User-applied frequency offset | Exact (matches f0 in .cfg files) |
| `EffectiveMHz` | Base + Offset | Exact (actual applied frequency) |

## Integration with msiaf Package

For complete GPU analysis, combine with the [`msiaf`](../msiaf/) package:

```go
// Read live data from driver (Windows/Linux)
nvapiPoints, _ := nvvf.ReadNvAPIVF(0)

// Read saved profile from .cfg file (cross-platform)
profile, _ := msiaf.Scan("LocalProfiles")
cfgBlob := profile.HardwareProfiles[0].VFCurve // Raw hex blob

// Decode .cfg blob (msiaf package)
cfgPoints, _ := msiaf.UnmarshalVFControlCurve(cfgBlob)

// Compare live vs saved
for i := range nvapiPoints {
    fmt.Printf("Point %d: live freq=%.0f MHz, saved freq=%.0f MHz\n",
        i, nvapiPoints[i].EffectiveMHz, cfgPoints[i].EffectiveMHz)
}
```

**Package responsibilities:**
- `nvvf` → NvAPI driver access (Windows + Linux) - reads **applied** curve
- `msiaf` → MSI Afterburner .cfg file parsing (cross-platform) - reads **configured** curve

## OC Scanner and Hardware Profile Behavior

### Why OffsetMHz Shows 0 When OC Scanner is Active

When you apply an OC Scanner profile or MSI Afterburner hardware profile, the NVIDIA driver modifies its internal boost tables **directly at a level below NvAPI**. This means:

| Field | What It Shows | OC Scanner Scenario |
|-------|---------------|---------------------|
| **BaseFreqMHz** | Current driver state | ✅ **Includes OC Scanner** (e.g., 2317 MHz) |
| **OffsetMHz** | NvAPI SetControl only | ❌ Always 0 (OC Scanner doesn't use SetControl) |
| **EffectiveMHz** | Actual GPU frequency | ✅ Matches applied curve (e.g., 2317 MHz) |

### Example: 850 mV with OC Scanner Applied

**From .cfg file (msiaf package):**
```
Voltage:   850 mV
Base:      1365 MHz (f1 - baseline frequency, possibly from OC Scanner)
Offset:    +952 MHz (f2 - user offset from base)
Effective: 2317 MHz (1365 + 952)
```

**From NvAPI (nvvf package):**
```
Voltage:      850 mV
BaseFreqMHz:  2317 MHz (driver already applied OC Scanner math)
OffsetMHz:    0 MHz (no additional SetControl offsets)
EffectiveMHz: 2317 MHz (matches .cfg!)
```

### Key Takeaways

1. **OC Scanner profiles ARE applied correctly** - NvAPI shows the final result
2. **OffsetMHz = 0 is expected** - OC Scanner doesn't use NvAPI SetControl
3. **To see OC Scanner offsets** - Parse .cfg files with the `msiaf` package
4. **To verify OC Scanner is working** - Compare EffectiveMHz from both sources

### Detecting OC Scanner vs Stock Curve

You **cannot** detect OC Scanner offsets from NvAPI alone. To analyze OC Scanner behavior:

```go
// Step 1: Read what the driver is applying (includes OC Scanner)
nvapiPoints, _ := nvvf.ReadNvAPIVF(0)

// Step 2: Read the .cfg configuration
profile, _ := msiaf.Scan("LocalProfiles")
cfgPoints, _ := msiaf.UnmarshalVFControlCurve(profile.HardwareProfiles[0].VFCurve)

// Step 3: Compare to understand what's applied
for i := range nvapiPoints {
    fmt.Printf("Voltage %.0f mV: NvAPI=%.0f MHz, .cfg=%.0f MHz\n",
        nvapiPoints[i].VoltageMV,
        nvapiPoints[i].EffectiveMHz,
        cfgPoints[i].EffectiveMHz)
}
```

If both match, OC Scanner/hardware profile is applied correctly. The .cfg file will show the individual `oc_ref` and `offset` components that NvAPI combines internally.

## Use Cases

### 1. Monitor Live GPU State

Read the current V-F curve directly from the driver to see actual hardware behavior.

### 2. Validate Overclocking Profiles

Compare what's saved in MSI Afterburner profiles against what the driver is actually applying.

### 3. Performance Analysis

Study the relationship between voltage and frequency to understand boost behavior.

### 4. Undervolt/Overclock Research

Analyze how user offsets affect the V-F curve at different voltage points.

## Safety

**This package is read-only and safe:**

- ✅ Only READs data from the driver
- ✅ Does NOT modify GPU settings
- ✅ Does NOT apply overclocks
- ✅ Non-destructive
- ✅ No risk to hardware

## Limitations

| Limitation | Reason |
|------------|--------|
| NVIDIA only | This is NVIDIA-specific via NvAPI |
| Undocumented functions | These NvAPI calls are not in the official SDK |
| Function IDs may change | NVIDIA could change them in future drivers (though unlikely based on historical stability) |
| No AMD/Intel support | This implementation is NVIDIA-specific |
| WSL support untested | Linux implementation uses libnvidia-api.so.1, WSL compatibility unknown |

## Quick Reference

### Exported Types

| Type | Purpose |
|------|---------|
| `VFPoint` | Voltage-frequency operating point from V-F curve |
| `ClkDomain` | Clock domain identifier with `String()` method |
| `ClkDomainInfo` | Clock domain information (domain, flags, min/max offsets) |

### ClkDomain Constants

| Constant | ID | Description |
|----------|-----|-------------|
| `DomainGraphics` | 0 | GPU core clock |
| `DomainMemory` | 4 | VRAM clock |
| `DomainProcessor` | 7 | Processor clock |
| `DomainVideo` | 8 | Video encoder/decoder clock |

### V-F Curve Functions

| Function | Description |
|----------|-------------|
| `ReadNvAPIVF(gpuIndex)` | Auto-detect GPU generation, read V-F curve |
| `ReadNvAPIVFBlackwell(gpuIndex)` | RTX 50xx (Blackwell) V-F curve |
| `ReadNvAPIVFLegacy(gpuIndex)` | RTX 30/40xx (Pascal/Ampere/Ada) V-F curve |
| `VFRange(points)` | Get min/max frequencies from VFPoints |
| `VFMinVoltage(points)` | Get minimum voltage from VFPoints |
| `VFMaxVoltage(points)` | Get maximum voltage from VFPoints |

### GPU Information Functions

| Function | Description |
|----------|-------------|
| `GetGPUName(gpuIndex)` | Get GPU marketing name (e.g., "NVIDIA GeForce RTX 5090") |
| `ReadNvAPIClkDomains(gpuIndex)` | Get clock domain ranges for OC validation |


## References

- **LACT Issue #936:** https://github.com/ilya-zlobintsev/LACT/issues/936 (Blackwell struct discovery, Linux confirmation)
- **LACT Project:** https://github.com/ilya-zlobintsev/LACT (Linux GPU control tool)
- **NvAPI SDK:** https://github.com/NVIDIA/nvapi (official SDK, does NOT document these functions)

## License

Same as the parent project (see [`LICENSE`](../LICENSE) in repository root).