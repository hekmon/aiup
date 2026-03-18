# AI Agent Guidelines

This document provides guidelines for AI agents working on this project. Read this at the start of each session.

---

## 🚀 GOLDEN RULES

### 1. Always Discuss Before Implementing

**Never start implementing without first presenting a plan and getting user validation.**

Before writing any code:
1. Analyze the current state of the codebase
2. Present a clear plan with proposed changes
3. Wait for explicit user validation before proceeding

### 2. Pre-Commit Checklist

**Before considering a task complete and presenting results to the user, always run:**

```bash
go generate ./...
go build ./...
go test ./...
go vet ./...
```

**And also check diagnostics:**
- Use the `diagnostics` tool to check for gopls linting suggestions
- Address any simplifications or improvements suggested (e.g., `strings.Cut`, deprecated APIs, style issues)

**All checks must pass** before considering work complete.

**When to run:** You don't need to run these after every single edit. Run them when you're ready to present final results to the user.

**Reporting:** Do not report successful verification checks to the user. Only mention issues if something fails and needs fixing.

### 3. Proactive AGENT.md Updates

At the end of successful sessions (everything builds, tests pass, stable conclusions reached):

1. Present a draft synthesis of what should be added to AGENT.md
2. Ask for approval before making changes

---

## 1. Project Overview

### Package Structure

```
msiaf/
├── scan.go                  # Directory/file discovery (scanning)
├── scan_test.go
├── globalconfig.go          # Global config parsing (MSIAfterburner.cfg)
├── globalconfig_test.go
└── catalog/                 # GPU catalog subpackage
    ├── catalog.go           # Lookup functions (hand-written)
    ├── catalog_generated.go # Auto-generated data (DO NOT EDIT)
    └── catalog_test.go      # Tests
```

- **Root package** (`msiaf/`): Scanning, config parsing, value-added methods on types
- **Catalog subpackage** (`msiaf/catalog/`): Pure GPU lookup functions (importable by anyone)

### File Organization Principle

**"Separate concerns by scope - new scopes deserve new files."**

| Scope | File | Purpose |
|-------|------|---------|
| Directory/File discovery | `scan.go` | Finding and listing files, parsing filenames |
| File content parsing | `globalconfig.go` | Parsing MSIAfterburner.cfg contents |
| File content parsing | `profile.go` | Parsing hardware profile .cfg contents (future) |
| GPU data lookups | `catalog/` | Auto-generated GPU database lookups |

```go
// ✅ Good: scan.go handles discovery only
result, err := msiaf.Scan(dir) // Finds files, parses filenames

// ✅ Good: globalconfig.go handles config file parsing
config, err := msiaf.ParseGlobalConfig(path) // Parses file contents

// ❌ Bad: Mixing concerns
// Don't put config parsing logic in scan.go
// Don't put file discovery logic in globalconfig.go
```

---

## 2. API Design Philosophy

### Core Principles

**"Don't hide anything, but pack convenience helpers where users will naturally find them."**

1. **Transparency**: All raw data (VendorID, DeviceID, SubsystemID, etc.) is directly accessible on structs
2. **Discoverability**: Helper methods are on the objects users already manipulate (`HardwareProfileInfo`)
3. **Flexibility**: Users can import `catalog` directly if they want raw lookups without scanning
4. **No unnecessary wrappers**: Avoid thin wrapper functions that just re-export from subpackages

### Strong Typing Philosophy

**"Parse everything during parsing - no lazy helper methods."**

#### Type Conversion Guidelines

| Config Format | Go Type | Example |
|--------------|---------|---------|
| `0`/`1` | `bool` | `StartWithWindows=1` → `true` |
| Decimal int | `int` | `LogLimit=10` → `10` |
| Hex int | `uint32` | `VideoCaptureFramesize=00000002h` → `2` |
| Hex blob | `[]byte` | `SwAutoFanControlCurve=...h` → decoded bytes |
| Time (ms) | `time.Duration` | `HwPollPeriod=1000` → `1 * time.Second` |
| Time (sec) | `time.Duration` | `VideoPrerecordTimeLimit=600` → `10 * time.Minute` |
| Hex timestamp | `time.Time` | `LastUpdateCheck=69B52594h` → `time.Time` |
| Path | `string` | `BenchmarkPath=%ABDir%\Benchmark.txt` (keep variables as-is) |
| Enum/Code | `string` | `Language=FR`, `VideoCaptureFormat=MJPG` |

#### Design Rationale

```go
// ✅ Good: Field is already time.Duration, ready to use
config.Settings.HwPollPeriod // time.Duration (1s)
fmt.Println(config.Settings.HwPollPeriod) // "1s"

// ❌ Bad: Requires helper method to convert
config.Settings.HwPollPeriodMs // int (1000) - user must convert manually
```

```go
// ✅ Good: All fields are named, IDE discovers them
type Settings struct {
    Language              string
    StartWithWindows      bool
    HwPollPeriod          time.Duration
    SwAutoFanControlCurve []byte
    // ... ALL fields, no exceptions
}

// ❌ Bad: Lazy fallback map
type Settings struct {
    Language string
    // ... some fields
    Raw map[string]string // Don't do this!
}
```

### Good Design Examples

```go
// ✅ Good: Value-added method on the type users already have
profile := result.HardwareProfiles[0]
fmt.Println(profile.GetGPUDescription()) // "ASUS NVIDIA GeForce RTX 5090"

// ✅ Also Good: Direct catalog import for standalone lookups
import "github.com/hekmon/aiup/msiaf/catalog"
gpuInfo := catalog.LookupGPU("10DE", "2B85")

// ❌ Bad: Unnecessary thin wrapper in root package
// Don't create msiaf.LookupGPU() that just calls catalog.LookupGPU()
```

---

## 3. GPU Catalog System

### Overview

The GPU catalog (`msiaf/catalog/catalog_generated.go`) is **auto-generated** and should **NEVER** be edited manually. It is generated from the [pci-ids database](https://pci-ids.ucw.cz/v2.2/pci.ids).

### Key Files

| File | Purpose |
|------|---------|
| `cmd/gencatalog/main.go` | Generator tool |
| `msiaf/scan.go` | Profile scanning logic + `HardwareProfileInfo` methods |
| `msiaf/catalog/catalog.go` | Lookup functions (hand-written) |
| `msiaf/catalog/catalog_generated.go` | Auto-generated data (DO NOT EDIT) |
| `msiaf/catalog/catalog_test.go` | Tests using actual pci-ids device IDs |

### Regenerating the Catalog

```bash
cd aiup
go generate ./msiaf/...
```

This will:
1. Fetch the latest pci-ids database
2. Filter for NVIDIA, AMD, and Intel GPUs (MSI Afterburner supported vendors)
3. Clean up device names (remove chip codenames)
4. Generate `msiaf/catalog/catalog_generated.go` with ~2,200+ GPU entries

### Important Conventions

| Convention | Detail |
|------------|--------|
| **Device IDs are lowercase** | pci-ids uses lowercase hex (e.g., `10de_2b85`). Lookup functions normalize input. |
| **Device IDs change** | IDs in pci-ids may differ from other databases. Verify against generated catalog. |
| **Name formatting** | Generator removes chip codenames (AD104, Navi 21, DG2, etc.) for cleaner names. |
| **Filtering** | Only actual GPU/display devices included. Non-GPU devices filtered out. |
| **Vendor IDs** | NVIDIA: `10de`, AMD: `1002`, Intel: `8086` |

### Testing Guidelines

- Use device IDs from the **generated** catalog, not hardcoded values
- Run `go generate` before running tests
- Test expectations should match pci-ids data, not external databases

```go
// Good: Uses actual pci-ids device ID
{"2B85", "GeForce RTX 5090"}

// Bad: May not match pci-ids
{"2786", "GeForce RTX 4090"} // Wrong ID in pci-ids
```

### Common Pitfalls

| Issue | Solution |
|-------|----------|
| Editing generated files | Never edit `catalog_generated.go` manually |
| Case sensitivity | Always use `strings.ToLower()` when looking up device IDs |
| Stale catalog | If tests fail with "unknown GPU", run `go generate` |
| Import errors | Import as `github.com/hekmon/aiup/msiaf/catalog`, not root package |

---

## 4. Fan Curve Serialization

The software auto fan control curve (`SwAutoFanControlCurve`) uses a 256-byte binary format stored in the MSI Afterburner configuration file.

**Location:** `msiaf/fancurve.go`

**Key characteristics:**
- **Strict validation with detailed error reporting** - Uses `FanCurveError` type to provide field-level error information
- **Version validation** - Must match `FanCurveBinaryFormatVersion` (0x00010000 = version 1.0)
- **Temperature range validation** - -50 to 150°C (sanity checks, not hardware limits)
- **Fan speed range validation** - 0-100% (physical limits)
- **Point ordering validation** - Points MUST be sorted by temperature in ascending order for correct interpolation
- **All parsing functions return errors** - No silent failures! This is critical for hardware safety - never apply corrupted fan curves

**Binary format documentation:** The complete binary format specification is documented inline in `fancurve.go` (see comments at the top of the file). Do not duplicate this documentation elsewhere - keep it with the code.

**Testing:** Unit tests are in `msiaf/fancurve_test.go`. Integration tests (config file parsing) are in `msiaf/globalconfig_test.go`.

---

## 5. Workflow Guidelines

### When Working on GPU-Related Features

1. **Check if catalog needs updating**: Run `go generate ./msiaf/...`
2. **Verify device IDs**: Look up actual device IDs in `catalog_generated.go`
3. **Test thoroughly**: Run `go test ./msiaf/...` after any changes
4. **Commit generated file**: Unlike typical generated files, `catalog_generated.go` should be committed

### When Moving or Restructuring Files

1. Update `package` declarations in moved files
2. Update `//go:generate` paths to account for new directory depth
3. Update generator (`cmd/gencatalog/main.go`) if it hardcodes package names
4. Regenerate: `go generate ./...`
5. Verify build: `go build ./...`
6. Verify tests: `go test ./...`

### When to Add Root Package Functions

Add functions to the root `msiaf` package when they:
- Combine data from multiple sources (e.g., scan results + catalog lookups)
- Provide value-added functionality (not just re-exports)
- Are methods on types that root package users already work with

### When to Use Catalog Subpackage

The `catalog` subpackage should:
- Contain pure lookup functions with no side effects
- Be importable directly by users who need raw lookups
- Stay focused on GPU/manufacturer data resolution

### Adding New Features

**Extending the catalog system:**
1. Modify `cmd/gencatalog/main.go` to add new logic
2. Re-run generator: `go generate ./msiaf/...`
3. Update tests in `msiaf/catalog/catalog_test.go`
4. Verify with `go build ./...` and `go test ./...`

**Extending the scanning functionality:**
1. Add functions/methods to `msiaf/scan.go`
2. Use catalog subpackage for GPU lookups (don't duplicate logic)
3. Add methods to `HardwareProfileInfo` for value-added helpers
4. Update tests in `msiaf/scan_test.go`
5. Verify with `go build ./...` and `go test ./...`

---

## 6. Code Quality Guidelines

### String Building

When building strings with `strings.Builder`, use `fmt.Fprintf()` instead of `WriteString(fmt.Sprintf())`:

```go
// ✅ Good: More efficient
fmt.Fprintf(&sb, "\t\"%s\": {Vendor: \"%s\", GPU: \"%s\"},\n", key, entry.Vendor, entry.GPU)

// ❌ Bad: Less efficient (creates intermediate string)
sb.WriteString(fmt.Sprintf("\t\"%s\": {Vendor: \"%s\", GPU: \"%s\"},\n", key, entry.Vendor, entry.GPU))
```

### go:generate Path Depth

When moving files with `//go:generate` directives, **update the path** to account for directory depth:

```go
//go:generate go run ../../cmd/gencatalog/main.go
```

- From `msiaf/catalog/catalog.go`: Use `../../cmd/gencatalog/main.go` (up 2 levels)
- The generator writes to the **current directory**, so run it from the correct location

### Package Consistency

When moving files to subpackages:
1. Update the `package` declaration in all moved files
2. Update the generator to output the correct package name
3. Update `go:generate` paths to reflect new directory depth
4. Verify imports in dependent files

---

## 7. API Usage Examples

### Basic Scanning

```go
package main

import (
    "fmt"
    "github.com/hekmon/aiup/msiaf"
)

func main() {
    result, err := msiaf.Scan("LocalProfiles")
    if err != nil {
        panic(err)
    }
    
    for _, profile := range result.HardwareProfiles {
        // Access raw IDs directly
        fmt.Printf("Vendor: %s\n", profile.VendorID)
        fmt.Printf("Device: %s\n", profile.DeviceID)
        
        // Use convenience methods for resolved info
        fmt.Printf("GPU: %s\n", profile.GetGPUDescription())
        fmt.Printf("Manufacturer: %s\n", profile.GetManufacturer())
    }
}
```

### Direct Catalog Lookup

```go
package main

import (
    "fmt"
    "github.com/hekmon/aiup/msiaf/catalog"
)

func main() {
    // Lookup GPU without scanning
    gpuInfo := catalog.LookupGPU("10DE", "2B85")
    if gpuInfo.IsKnown {
        fmt.Printf("GPU: %s %s\n", gpuInfo.VendorName, gpuInfo.GPUName)
    }
    
    // Lookup manufacturer from SubsystemID
    manufacturer := catalog.LookupManufacturer("89EC1043") // Returns "ASUS"
    
    // Get full description
    desc := catalog.GetFullGPUDescription("10DE", "2B85", "89EC1043")
    // Returns: "ASUS NVIDIA GeForce RTX 5090"
}
```

---

## 8. Troubleshooting

| Issue | Solution |
|-------|----------|
| "unknown GPU" errors | Run `go generate ./msiaf/...` to update the catalog |
| Build fails with "package not found" | Check `package` declarations and import paths |
| go generate fails | Check `//go:generate` path for correct directory depth |
| Tests fail after moving files | Ensure generator outputs correct package name, re-run `go generate ./...` |
| Import errors with catalog | Import as `github.com/hekmon/aiup/msiaf/catalog` (root package doesn't re-export catalog) |

---

## 9. Hardware Profile Parsing

**Location:** `msiaf/profile.go`

Hardware profile files (e.g., `VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg`) contain GPU-specific overclocking and fan settings.

### File Structure

Hardware profiles use INI-style format with the following sections:

| Section | Purpose |
|---------|---------|
| `[Startup]` | Currently active settings (applied on load) |
| `[Profile1]` - `[Profile5]` | User-defined overclocking slots |
| `[Defaults]` | Factory default baseline values |
| `[PreSuspendedMode]` | State before system suspension (for restoration) |
| `[Settings]` | Miscellaneous profile metadata |

### Key Fields

| Field | Type | Unit | Description |
|-------|------|------|-------------|
| `Format` | *int | - | Profile format version (e.g., 2) |
| `PowerLimit` | *int | % | Power limit percentage |
| `CoreClkBoost` | *int | kHz | Core clock offset |
| `MemClkBoost` | *int | kHz | Memory clock offset |
| `VFCurve` | []byte | - | Voltage-frequency curve (binary format) |
| `FanMode` | *int | - | 0=auto, 1=manual |
| `FanSpeed` | *int | % | Manual fan speed |

### Pointer Field Design

**All numeric fields use pointers** to distinguish between:
- **`nil`** = field not present in file
- **`&value`** = field explicitly set (even if 0)

This is more idiomatic Go and enables:
- Detecting sparse sections (e.g., PreSuspendedMode)
- Clean serialization (only write non-nil fields)
- Semantic clarity without auxiliary tracking

```go
// Good: Use helper methods for ergonomic access
startup := profile.GetCurrentSettings()
if startup.PowerLimit != nil {
    fmt.Printf("Power: %d%%\n", *startup.PowerLimit)
}

// Better: Use getters (handles nil automatically)
fmt.Printf("Power: %d%%\n", startup.GetPowerLimit())

// Best: Use unit-appropriate helpers
fmt.Printf("Core Clock: +%d MHz\n", startup.GetCoreClkBoostMHz())
```

### VFCurve Handling

The voltage-frequency curve (`VFCurve`) is currently stored as a raw `[]byte` (decoded hex blob). Binary format parsing is deferred to a future task, following the same pattern as `SwAutoFanControlCurve` before its deserializer was implemented.

### Usage Example

```go
// Scan for profiles
result, err := msiaf.Scan("LocalProfiles")

// Load hardware profile for a specific GPU
hwProfile, err := result.HardwareProfiles[0].LoadProfile()

// Access settings
startup := hwProfile.GetCurrentSettings()
fmt.Printf("Power Limit: %d%%\n", startup.GetPowerLimit())
fmt.Printf("Core Clock: +%d MHz\n", startup.GetCoreClkBoostMHz())
fmt.Printf("Memory: +%d MHz\n", startup.GetMemClkBoostMHz())

// Check if profile slot is populated
slot := hwProfile.GetProfile(1)
if slot != nil && !slot.IsEmpty {
    fmt.Println("Profile1 is configured")
}

// Check for sparse sections
if hwProfile.PreSuspendedMode.HasSettings() {
    // Process pre-suspend state
}
```

### Testing

- Unit tests: `msiaf/profile_test.go`
- Integration tests use actual profile files from `LocalProfiles/`
- Tests verify pointer field semantics (nil vs &0)
