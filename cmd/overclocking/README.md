# overclocking CLI

**Package:** `github.com/hekmon/aiup/cmd/overclocking`

**Purpose:** Command-line interface demonstrating the overclocking package capabilities.

**Platform:** Windows (requires NvAPI and MSI Afterburner)

---

## Overview

The `overclocking` CLI provides two modes of operation:

1. **GPU Discovery** (default) - Lists all NVIDIA GPUs with their MSI Afterburner profiles
2. **Current Curve** - Reads the current V-F curve from a specific GPU

Both modes output results as pretty-printed JSON for easy parsing by scripts or agents.

---

## Installation

Build from source:

```bash
cd aiup
GOOS=windows GOARCH=amd64 go build -o overclocking.exe ./cmd/overclocking/
```

The executable can be placed anywhere - it has no external dependencies beyond:
- NVIDIA GPU driver with NvAPI
- MSI Afterburner installed with profiles configured

---

## Usage

### Mode 1: GPU Discovery (Default)

Scans MSI Afterburner Profiles directory and correlates with NvAPI-detected GPUs:

```bash
overclocking.exe
```

**Output:**
```json
{
  "profiles_dir": "C:\\Program Files (x86)\\MSI Afterburner\\Profiles",
  "global_config_path": "C:\\Program Files (x86)\\MSI Afterburner\\Profiles\\MSIAfterburner.cfg",
  "gpus": [
    {
      "index": 0,
      "name": "NVIDIA GeForce RTX 5090",
      "vendor_id": "10DE",
      "device_id": "2B85",
      "subsystem_id": "89EC1043",
      "bus_number": 1,
      "device_number": 0,
      "function_number": 0,
      "profile_path": "C:\\Program Files (x86)\\MSI Afterburner\\Profiles\\VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg",
      "manufacturer": "ASUS",
      "full_description": "ASUS NVIDIA GeForce RTX 5090"
    }
  ]
}
```

**Use Cases:**
- Discover which GPUs have MSI Afterburner profiles
- Get profile paths for each GPU
- Identify manufacturer from SubsystemID

---

### Mode 2: Current Curve

Reads the current V-F curve from a specific GPU and validates it matches the Startup profile:

```bash
overclocking.exe -gpu 0
```

**Output:**
```json
{
  "points": [
    {
      "voltage_mv": 850.0,
      "base_freq_mhz": 1365.0,
      "offset_mhz": 952.0,
      "effective_freq_mhz": 2317.0
    },
    {
      "voltage_mv": 862.0,
      "base_freq_mhz": 1380.0,
      "offset_mhz": 937.0,
      "effective_freq_mhz": 2317.0
    }
  ],
  "profile": {
    "slot_number": 2,
    "slot_name": "Profile2",
    "confidence": 1.0
  }
}
```

**Or if no profile slot matches:**
```json
{
  "points": [...],
  "profile": null
}
```

**Use Cases:**
- Read current overclock offsets before modification
- Validate that Startup profile is properly applied
- Identify which profile slot the current curve originated from

---

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-profiles` | string | `C:\Program Files (x86)\MSI Afterburner\Profiles` | Path to MSI Afterburner Profiles directory |
| `-gpu` | int | `-1` | Get current V-F curve for specified GPU index (0, 1, 2, ...) |
| `-h, --help` | - | - | Show help message |

---

## Examples

### List all GPUs with profiles

```bash
overclocking.exe
```

### Use custom profiles directory

```bash
overclocking.exe -profiles "D:\Custom\Profiles"
```

### Get current V-F curve for GPU 0

```bash
overclocking.exe -gpu 0
```

### Get current V-F curve for GPU 1

```bash
overclocking.exe -gpu 1
```

### Show help

```bash
overclocking.exe -h
```

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (message printed to stderr) |

---

## JSON Schema

### Discovery Mode Output

```typescript
{
  profiles_dir: string;
  global_config_path: string;
  GPUs: Array<{
    index: number;           // NvAPI GPU index
    name: string;            // Marketing name
    vendor_id: string;       // PCI vendor ID (e.g., "10DE")
    device_id: string;       // PCI device ID (e.g., "2B85")
    subsystem_id: string;    // PCI subsystem ID (e.g., "89EC1043")
    bus_number: number;      // PCI bus number
    device_number: number;   // PCI device number
    function_number: number; // PCI function number
    profile_path: string;    // Path to hardware profile .cfg
    manufacturer: string;    // Card manufacturer (e.g., "ASUS")
    full_description: string; // Complete description
  }>;
  errors?: string[];
}
```

### Current Curve Mode Output

```typescript
{
  points: Array<{
    voltage_mv: number;         // Voltage point (e.g., 850.0)
    base_freq_mhz: number;      // Hardware base frequency
    offset_mhz: number;         // Overclock offset (CORE VALUE)
    effective_freq_mhz: number; // BaseFreqMHz + OffsetMHz
  }>;
  profile: {
    slot_number: number;    // 1-5 (Profile1-5)
    slot_name: string;      // "Profile1", "Profile2", etc.
    confidence: number;     // 0.0-1.0
  } | null;
}
```

---

## Error Handling

Errors are printed to stderr with a descriptive message:

```
Error: failed to scan profiles directory: directory not found
Error: GPU index 5 not found in discovery result
Error: live V-F curve does not match Startup profile (0% confidence) - this should not happen
```

The program exits with code `1` on any error.

---

## Notes

- **WSL Support:** Can be executed from WSL - Windows interop provides NvAPI access
- **GPU Index:** Uses NvAPI GPU indices (0-based, in order of detection)
- **Profile Validation:** Current curve mode validates that live = Startup (100% confidence required)
- **Profile Hint:** The `profile` field indicates which Profile1-5 slot matches (informational only)

---

## See Also

- **Package Documentation:** [`../../overclocking/README.md`](../../overclocking/README.md)
- **MSI Afterburner:** [Official Website](https://www.msi.com/Landing/afterburner/graphics-cards)
- **NvAPI:** [NVIDIA NvAPI Documentation](https://nvidia.github.io/open-gpu-kernel-modules/)