# nvvf - NVIDIA V-F Curve Reader

This package provides tools for reading NVIDIA GPU voltage-frequency (V-F) curves directly from the graphics driver using undocumented NvAPI calls.

## Purpose

The `nvvf` package reads **live V-F curve data from the NVIDIA driver** on Windows and Linux systems. This provides:

- **Current driver frequencies** from the driver's pstate table (includes OC Scanner/hardware profile modifications)
- **Additional offsets** applied via NvAPI SetControl (typically 0 when using OC Scanner)
- **Precise effective frequencies** (base + offset)

This data was previously only accessible through proprietary tools like MSI Afterburner.

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

## Technical Details

### NvAPI Functions Used

| Function ID | Name | Purpose |
|-------------|------|---------|
| `0x21537AD4` | `ClockClientClkVfPointsGetStatus` | Read hardware base V-F curve |
| `0x23F1B133` | `ClockClientClkVfPointsGetControl` | Read user frequency offsets |

### Struct Sizes by Generation

| GPU Generation | Status Struct | Control Struct | Entry Stride |
|----------------|---------------|----------------|--------------|
| **RTX 30/40xx** (Pascal/Ampere/Ada) | 1076 bytes | 1076 bytes | 8 bytes |
| **RTX 50xx** (Blackwell) | 7208 bytes | 9248 bytes | 28/72 bytes |

The implementation auto-detects which struct sizes to use based on GPU compatibility.

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

## References

- **LACT Issue #936:** https://github.com/ilya-zlobintsev/LACT/issues/936 (Blackwell struct discovery, Linux confirmation)
- **LACT Project:** https://github.com/ilya-zlobintsev/LACT (Linux GPU control tool)
- **NvAPI SDK:** https://github.com/NVIDIA/nvapi (official SDK, does NOT document these functions)

## License

Same as the parent project (see [`LICENSE`](../LICENSE) in repository root).