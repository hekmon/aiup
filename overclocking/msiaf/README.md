# MSI Afterburner Package (`msiaf`)

Package `msiaf` provides tooling for working with MSI Afterburner profiles and configuration files.

## Features

- **Profile Scanning** - Discover and parse MSI Afterburner hardware profiles
- **Configuration Parsing** - Parse `MSIAfterburner.cfg` global settings
- **Hardware Profile Parsing** - Parse GPU-specific `.cfg` profile files
- **V-F Curve Analysis** - Unmarshal and analyze voltage-frequency curves
- **Fan Curve Analysis** - Unmarshal and analyze fan control curves
- **Offset Mode Detection** - Detect fixed offset (slider) vs custom curve (editor) modes
- **Profile Matching** - Match profiles against live GPU V-F data

## Offset Mode Detection

The package can automatically detect whether a profile uses **fixed offset mode** (simple slider) or **custom curve mode** (curve editor).

### Detection Logic

**The V-F curve is the source of truth.** Detection is based on analyzing the offset pattern in the V-F curve binary data:

| V-F Curve Pattern | Detected Mode | CoreClkBoost Value |
|-------------------|---------------|---------------------|
| All active points have uniform offset (±1 MHz tolerance) | Fixed Offset | Any value (even 1000000 kHz for +1000 MHz) |
| Active points have varying offsets | Custom Curve | Typically `1000000` kHz (marker value) |

**Critical Edge Case:** When you slide the Core slider to **+1000 MHz**, `CoreClkBoost = 1000000 kHz`, which is the same value used as a marker for custom curve mode. The detection logic handles this by **always checking the V-F curve**:
- Uniform +1000 MHz offsets → Fixed Offset mode (slider at max)
- Varying offsets → Custom Curve mode (curve editor)

### API Reference

#### `OffsetMode` Type

```go
type OffsetMode int

const (
    OffsetModeUnknown     OffsetMode = iota // Unable to determine
    OffsetModeFixedOffset                   // Uniform offset (slider mode)
    OffsetModeCustomCurve                   // Varying offsets (curve editor)
    OffsetModeInvalid                       // Inconsistent state
)
```

#### `GetOffsetMode()` Method

Detect the offset mode for a profile section:

```go
import "github.com/hekmon/aiup/overclocking/msiaf"

profile, err := msiaf.ParseHardwareProfile("path/to/profile.cfg")
if err != nil {
    panic(err)
}

mode := profile.Startup.GetOffsetMode()
fmt.Printf("Offset mode: %s\n", mode)
// Output: "Offset mode: Fixed Offset (slider mode)"
```

**Detection Process:**
1. Parse V-F curve binary data
2. Collect offset values from all active points
3. Check if all offsets are uniform (within ±1 MHz tolerance)
   - **Uniform** → Fixed Offset mode (regardless of CoreClkBoost value)
   - **Varying** → Custom Curve mode
4. If in Fixed Offset mode and CoreClkBoost = 1000000 kHz, extract offset from first active V-F point (handles +1000 MHz edge case)

#### `GetFixedOffset()` Method

Extract the fixed offset value when in fixed offset mode:

```go
offset, ok := profile.Startup.GetFixedOffset()
if ok {
    fmt.Printf("Fixed offset: +%d MHz\n", offset)
} else {
    fmt.Println("Not in fixed offset mode")
}
```

**Returns:**
- `(offsetMHz, true)` - When in fixed offset mode
- `(0, false)` - When in custom curve mode or unable to determine

### Usage Examples

#### Detect Profile Mode

```go
package main

import (
    "fmt"
    "github.com/hekmon/aiup/overclocking/msiaf"
)

func main() {
    profile, err := msiaf.ParseHardwareProfile(
        "VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg"
    )
    if err != nil {
        panic(err)
    }

    // Check Startup profile
    mode := profile.Startup.GetOffsetMode()
    fmt.Printf("Startup mode: %s\n", mode)

    if mode == msiaf.OffsetModeFixedOffset {
        offset, _ := profile.Startup.GetFixedOffset()
        fmt.Printf("  Fixed offset: +%d MHz\n", offset)
    }

    // Check all profile slots
    for i := 0; i < 5; i++ {
        slotMode := profile.Profiles[i].GetOffsetMode()
        fmt.Printf("Profile%d mode: %s\n", i+1, slotMode)
    }
}
```

#### Validate Profile Consistency

```go
// Detect invalid states where CoreClkBoost doesn't match V-F curve
mode := profile.Startup.GetOffsetMode()
if mode == msiaf.OffsetModeInvalid {
    fmt.Println("⚠️  Warning: Profile is in inconsistent state!")
    fmt.Println("   CoreClkBoost suggests fixed offset but V-F curve has varying offsets")
}
```

#### Compare Multiple Profiles

```go
startupMode := profile.Startup.GetOffsetMode()
profile1Mode := profile.Profiles[0].GetOffsetMode()

if startupMode != profile1Mode {
    fmt.Println("Profile modes differ:")
    fmt.Printf("  Startup:  %s\n", startupMode)
    fmt.Printf("  Profile1: %s\n", profile1Mode)
}
```

### Mode String Representations

```go
fmt.Println(msiaf.OffsetModeFixedOffset)   // "Fixed Offset (slider mode)"
fmt.Println(msiaf.OffsetModeCustomCurve)   // "Custom Curve (curve editor)"
fmt.Println(msiaf.OffsetModeInvalid)       // "Invalid (inconsistent state)"
fmt.Println(msiaf.OffsetModeUnknown)       // "Unknown"
```

## Related Files

| File | Purpose |
|------|---------|
| [`profile.go`](profile.go) | Hardware profile parsing, `GetOffsetMode()`, `GetFixedOffset()` |
| [`vfcurve.go`](vfcurve.go) | V-F curve binary format parsing and marshaling |
| [`fancurve.go`](fancurve.go) | Fan curve binary format parsing |
| [`globalconfig.go`](globalconfig.go) | Global config parsing (`MSIAfterburner.cfg`) |
| [`scan.go`](scan.go) | Profile file discovery and scanning |
| [`active.go`](active.go) | Profile matching against live V-F data |

### Technical Details

### CoreClkBoost Field Values

| Value | Typical Meaning | Actual Mode (from V-F curve) |
|-------|-----------------|------------------------------|
| `0` | Stock settings (0 MHz offset) | Fixed Offset (+0 MHz) |
| `123000` | Fixed +123 MHz offset | Fixed Offset (+123 MHz) |
| `-50000` | Fixed -50 MHz offset (undervolt) | Fixed Offset (-50 MHz) |
| `1000000` | Custom curve mode (marker) | **Could be either:** Custom Curve OR Fixed +1000 MHz |

**Important:** CoreClkBoost = `1000000` kHz is ambiguous. Always check the V-F curve to determine the actual mode.

### Tolerance for Fixed Offset Detection

The detection uses a **1.0 MHz tolerance** when comparing V-F curve offsets to `CoreClkBoost`. This accounts for floating-point precision in the binary format.

### Invalid State Detection

An `OffsetModeInvalid` result indicates:
- V-F curve binary data could not be parsed (corrupt or invalid format)
- V-F curve is empty or has no active points

**Note:** The previous implementation flagged mismatches between `CoreClkBoost` and V-F curve as "Invalid". The corrected logic recognizes that **CoreClkBoost is just a hint** - the V-F curve always contains the authoritative data. A mismatch simply means the V-F curve pattern determines the mode.