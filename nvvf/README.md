# nvvf - NVIDIA V-F Curve Reader

This package provides tools for reading NVIDIA GPU voltage-frequency (V-F) curves directly from the graphics driver using undocumented NvAPI calls.

## Purpose

The `nvvf` package reads **live V-F curve data from the NVIDIA driver** on Windows and Linux systems. This provides:

- **Exact hardware base frequencies** from the driver's pstate table
- **User-applied frequency offsets** from driver control settings
- **Precise effective frequencies** (base + offset)

This data was previously only accessible through proprietary tools like MSI Afterburner.

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
    BaseFreqMHz  float64 // Hardware base frequency from driver
    OffsetMHz    float64 // User frequency offset
    EffectiveMHz float64 // Base + Offset (exact applied frequency)
}
```

## Requirements

### Windows
- **Windows x64** operating system
- **NVIDIA display driver** installed (provides `nvapi64.dll`)
- **NVIDIA GPU** present in the system

### Linux
- **Linux x64** operating system (native, not WSL)
- **NVIDIA display driver** installed (provides `libnvidia-api.so.1`)
- **NVIDIA GPU** present in the system

**WSL Users:** Run the Windows build (`nvvf.exe`) directly from WSL. Example:
```bash
# From WSL terminal
GOOS=windows go build -o nvvf.exe ./cmd/nvvf    # Build Windows binary
./nvvf.exe -list                    # Works via WSL interop
```

## Command-Line Tool

The `cmd/nvvf` tool provides quick access to V-F curve data:

```bash
# Build
cd aiup
go build -o nvvf.exe ./cmd/nvvf    # Windows
go build -o nvvf ./cmd/nvvf        # Linux

# Usage
nvvf              # Read all GPUs
nvvf -gpu 0       # Read GPU 0 only
nvvf -v           # Verbose output
nvvf -json        # JSON output
nvvf -list        # List available GPUs
nvvf -h           # Show help
```

### Example Output

```
=== NVIDIA V-F Curve (from NvAPI) ===

GPU 0:
  Voltage (mV) | Base (MHz) | Offset (MHz) | Effective (MHz)
  -------------|------------|--------------|----------------
           450 |        225 |            0 |            225
           770 |        247 |            0 |            247
           890 |       1875 |            0 |           1875
  ...
           1240 |       2872 |            0 |           2872

=== Summary ===

Hardware Base Frequency Range: 225 - 2872 MHz
Voltage Range: 450 - 1240 mV
Active V/F Points: 128 / 128
```

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
    fmt.Printf("Point %d: live offset=%.0f MHz, saved offset=%.0f MHz\n",
        i, nvapiPoints[i].OffsetMHz, cfgPoints[i].OffsetMHz)
}
```

**Package responsibilities:**
- `nvvf` → NvAPI driver access (Windows + Linux)
- `msiaf` → MSI Afterburner .cfg file parsing (cross-platform)

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