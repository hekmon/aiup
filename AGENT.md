# AI Agent Guidelines

**Purpose:** Read this at the start of each session. This document tells you how to work on this project.

**Last Updated:** Current session - Full restructure for AI agent optimization

---

## ⚡ QUICK REFERENCE

### Common Tasks

| Task | Command | Location |
|------|---------|----------|
| Regenerate GPU catalog | `go generate ./msiaf/...` | Project root |
| Run all pre-commit checks | `go generate ./... && go build ./... && go test ./... && go vet ./...` | Project root |
| Create temp experiment | `mkdir -p tmp/<name>` | Project root |
| Clean up experiments | `rm -rf tmp/` | After implementation |
| Lookup device ID | Check `msiaf/catalog/catalog_generated.go` | Or run `go generate` |

### Decision Tree: Where Does My Code Go?

```
What are you building?
│
├── Finding/listing files?          → msiaf/scan.go
├── Parsing MSIAfterburner.cfg?     → msiaf/globalconfig.go
├── Parsing hardware profile .cfg?  → msiaf/profile.go
├── Fan curve binary format?        → msiaf/fancurve.go
├── V-F curve binary format?        → msiaf/vfcurve.go
├── Profile matching/detection?     → msiaf/active.go
├── Writing profiles?               → msiaf/profile.go (Save methods)
├── New binary format?              → msiaf/<format>.go (new file)
├── GPU manufacturer lookups?       → msiaf/catalog/ (hand-written)
├── GPU data table?                 → msiaf/catalog_generated.go (auto-generated)
├── Combining multiple sources?     → Method on existing type in root package
├── Thin wrapper around subpackage? → DON'T DO IT
└── Utility/helper function?        → Method on the type users already have
```

### Heuristics: When in Doubt...

| Situation | Default Action |
|-----------|----------------|
| Unknown binary format | Create temp tool in `tmp/`, document findings, then implement |
| Unsure about struct fields | Parse everything during parsing, no lazy methods |
| Adding a helper function | Make it a method on the type users already manipulate |
| Found bug in generated code | Fix the generator (`cmd/gencatalog/`), not the output |
| Tests fail with "unknown GPU" | Run `go generate ./msiaf/...` first |
| Unclear file organization | Follow "new scopes deserve new files" principle |
| Need to move files | Update `//go:generate` paths, regenerate, verify |
| Adding new config field | Add to struct, parse during initial parse, use appropriate Go type |

---

## 🚀 GOLDEN RULES

### 1. Always Discuss Before Implementing

**Never start implementing without first presenting a plan and getting user validation.**

Before writing any code:
1. Analyze the current state of the codebase
2. Present a clear plan with proposed changes
3. Wait for explicit user validation before proceeding

### 2. Pre-Commit Checklist

**Before considering a task complete, always run:**

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

## 📁 PROJECT MAP

### Directory Structure

```
/home/doudou/perso/github.com/hekmon/aiup/
│
├── AGENT.md                    # This file - AI agent guidelines
├── README.md                   # Project documentation
├── go.mod, go.sum              # Go module definition
│
├── msiaf/                      # Main package - MSI Afterburner parsing
│   ├── scan.go                 # HardwareProfileInfo, Scan(), file discovery
│   ├── scan_test.go
│   ├── active.go               # Profile matching (detect active profile)
│   ├── active_test.go          # Profile matching tests
│   ├── globalconfig.go         # Settings struct, ParseGlobalConfig()
│   ├── globalconfig_test.go
│   ├── profile.go              # HardwareProfile struct, parsing + writing
│   ├── profile_test.go
│   ├── fancurve.go             # Fan curve binary deserialization
│   ├── fancurve_test.go
│   ├── vfcurve.go              # V-F curve binary deserialization + marshaling
│   ├── vfcurve_test.go
│   └── catalog/                # GPU lookup subpackage
│       ├── catalog.go          # LookupGPU(), LookupManufacturer() (hand-written)
│       ├── catalog_generated.go # GPU data table (DO NOT EDIT - auto-generated)
│       └── catalog_test.go
│
├── cmd/
│   ├── active/                 # Windows-only: Match profiles against live V-F curve
│   │   └── main.go             # Combines msiaf + nvvf for profile analysis
│   ├── gencatalog/             # GPU catalog generator tool
│   │   └── main.go             # Fetches pci-ids, generates catalog_generated.go
│   ├── msiaf/                  # Example: Pure msiaf package usage
│   │   └── main.go             # Scan profiles, parse configs (no hardware required)
│   └── nvvf/                   # Example: Live V-F curve reading
│       └── main.go             # Read NVIDIA GPU V-F data via NvAPI (requires GPU)
│
├── LocalProfiles/              # Test data (gitignored)
│   └── *.cfg                   # Hardware profile files
│
└── tmp/                        # Temporary experiment tools (gitignored)
    └── <experiment_name>/      # Remove after implementation complete
```

### tmp/ Directory Rules

**ALL temporary/test programs MUST be placed under `tmp/`.**

| Rule | Requirement |
|------|-------------|
| **Location** | All temporary code goes in `tmp/<experiment_name>/` |
| **Git status** | `tmp/` is in `.gitignore` - never commit temporary code |
| **Cleanup** | Remove `tmp/<experiment_name>/` after implementation complete |
| **Purpose** | Quick experiments, validation tests, one-off tools |
| **Structure** | Create subdirectory per experiment: `tmp/validate_set/`, `tmp/test_parser/` |
| **Duration** | Temporary by definition - clean up at end of session/task |

**Why tmp/?**
- Keeps experiments isolated from production code
- Prevents accidental commits of test code
- Easy to clean up: `rm -rf tmp/`
- Follows Go project conventions

### Package Responsibilities

| Package | Purpose | Import Path |
|---------|---------|-------------|
| `msiaf` (root) | Scanning, config parsing, value-added methods on types | `github.com/hekmon/aiup/msiaf` |
| `msiaf/catalog` | Pure GPU/manufacturer lookup functions | `github.com/hekmon/aiup/msiaf/catalog` |
| `cmd/gencatalog` | Generator tool (not importable) | N/A |

---

## 1. API DESIGN PHILOSOPHY

### Core Principles

**"Don't hide anything, but pack convenience helpers where users will naturally find them."**

| Principle | Meaning |
|-----------|---------|
| **Transparency** | All raw data (VendorID, DeviceID, SubsystemID, etc.) directly accessible on structs |
| **Discoverability** | Helper methods on objects users already manipulate (`HardwareProfileInfo`) |
| **Flexibility** | Users can import `catalog` directly for raw lookups without scanning |
| **No unnecessary wrappers** | Avoid thin wrapper functions that just re-export from subpackages |

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

## 2. GPU CATALOG SYSTEM

### Overview

The GPU catalog (`msiaf/catalog/catalog_generated.go`) is **auto-generated** and should **NEVER** be edited manually. Generated from the [pci-ids database](https://pci-ids.ucw.cz/v2.2/pci.ids).

### Key Files

| File | Purpose | Edit? |
|------|---------|-------|
| `cmd/gencatalog/main.go` | Generator tool | Yes |
| `msiaf/catalog/catalog.go` | Lookup functions (hand-written) | Yes |
| `msiaf/catalog/catalog_generated.go` | Auto-generated data | **NO** |
| `msiaf/catalog/catalog_test.go` | Tests | Yes |

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

### Common Pitfalls

| Issue | Solution |
|-------|----------|
| Editing generated files | Never edit `catalog_generated.go` manually |
| Case sensitivity | Always use `strings.ToLower()` when looking up device IDs |
| Stale catalog | If tests fail with "unknown GPU", run `go generate` |
| Import errors | Import as `github.com/hekmon/aiup/msiaf/catalog`, not root package |

---

## 3. BINARY FORMAT PARSERS

### Overview

Binary formats in MSI Afterburner config files use hex-encoded blobs. Each format has its own parser with strict validation.

| Format | File | Purpose |
|--------|------|---------|
| Fan Curve | `msiaf/fancurve.go` | Auto fan control (256-byte format) |
| V-F Curve | `msiaf/vfcurve.go` | Voltage-frequency curve (variable length) |

### General Pattern for New Binary Parsers

When implementing a new binary format parser:

1. **Create new file** → `msiaf/<format>.go`
2. **Document format inline** → Package-level comments with complete binary specification
3. **Define error types** → Use structured errors (e.g., `FormatError`) with field-level details
4. **Validate strictly** → Version checks, range checks, ordering checks
5. **Return errors** → No silent failures - critical for hardware safety
6. **Write unit tests** → `msiaf/<format>_test.go` with known-good and known-bad inputs
7. **Add integration tests** → Parse actual config/profile files

### Validation Checklist

```
├── Version validation?           → Must match expected format version
├── Length validation?            → Check expected byte count
├── Range validation?             → Sanity check numeric ranges
├── Ordering validation?          → Points sorted correctly (if applicable)
├── Reserved field validation?    → Check reserved fields are as expected
└── Error reporting?              → Field-level error information
```

### Temp Experiments

When investigating unknown formats:

1. **Create temp tool** → `tmp/<experiment_name>/main.go`
2. **Store large constants** → Separate `.go` files for hex data
3. **Test hypotheses** → Systematically with clear labeled output
4. **Document findings** → In AGENT.md before implementing final parser
5. **Follow reference pattern** → Use `fancurve.go` or `vfcurve.go` as template
6. **Clean up** → `rm -rf tmp/<experiment_name>/` after implementation

**See also:** [tmp/ Directory Rules](#tmp-directory-rules) in Project Map

---

## 4. CROSS-PLATFORM NVAPI ACCESS

### Overview

The `nvvf` package provides cross-platform access to NVIDIA GPU V-F (voltage-frequency) curve data using undocumented NvAPI functions. The same NvAPI function IDs work on both Windows and Linux, but the library loading and calling conventions differ.

| Platform | Library | Build Tag | Loading Method |
|----------|---------|-----------|----------------|
| **Windows** | `nvapi64.dll` | `//go:build windows` | `syscall.LoadDLL()` + `syscall.SyscallN()` |
| **Linux** | `libnvidia-api.so.1` | `//go:build linux` | cgo + `dlopen()`/`dlsym()` |

### File Organization

```
nvvf/
├── nvvf.go           # Shared: types, structs, constants, parsers, ReadNvAPIVF()
├── nvvf_windows.go   # Windows: LoadNvAPI, ReadNvAPIVFBlackwell/Legacy
└── nvvf_linux.go     # Linux: loadNvAPILinux (cgo), ReadNvAPIVFBlackwell/Legacy
```

### Shared Code (`nvvf.go`)

**Contains:**
- `VFPoint` type and all NvAPI struct definitions (legacy + Blackwell)
- Function ID constants (`fnInitialize`, `fnEnumGPUs`, `fnVfGetStatus`, `fnVfGetControl`)
- `ReadNvAPIVF()` auto-detect function (identical on both platforms)
- Parser functions (`parseBlackwellVFPoints`, `parseLegacyVFPoints`, `round2`)

**Does NOT contain:**
- Platform-specific library loading code
- NvAPI call implementations (different calling conventions)

### Platform-Specific Code

**Windows (`nvvf_windows.go`):**
```go
//go:build windows

func ReadNvAPIVFBlackwell(gpuIndex int) ([]VFPoint, error) {
    dll, err := syscall.LoadDLL("nvapi64.dll")
    // ... use syscall.SyscallN() for calls
    // Error codes: 0x00000000 = success, 0x8000xxxx = error
}
```

**Linux (`nvvf_linux.go`):**
```go
//go:build linux

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
// ... cgo helpers for dlopen/dlsym
*/
import "C"

func ReadNvAPIVFBlackwell(gpuIndex int) ([]VFPoint, error) {
    nvapi, err := loadNvAPILinux()
    // ... use cgo call wrappers (call0, call2)
    // Error codes: 0 = success, negative integers = error
}
```

### Key Design Decisions

**1. Separate Call Conventions**
- Windows uses `syscall.SyscallN(fn, args...)` with `uintptr` arguments
- Linux uses cgo with `unsafe.Pointer` arguments
- **Do NOT** try to abstract this with a generic variadic function - it triggers `go vet` warnings about unsafe.Pointer conversions

**2. Shared Parsers, Separate Calls**
- Struct layouts and parsing logic are 100% identical → keep in `nvvf.go`
- Library loading and calling conventions differ → keep in platform files
- `ReadNvAPIVF()` auto-detect is identical → keep in `nvvf.go`

**3. Error Code Differences**
| Platform | Success | Error Format | Example |
|----------|---------|--------------|---------|
| Windows | `0x00000000` | `0x%08X` | `0x80001003` |
| Linux | `0` | `%d` (signed) | `-9` (INCOMPATIBLE_STRUCT_VERSION) |

**4. Type-Safe Call Wrappers (Linux)**
To avoid `go vet` warnings about `uintptr` → `unsafe.Pointer` conversions, Linux uses type-specific wrappers:
```go
func (n *nvapiLinux) call0(fn unsafe.Pointer) uint32  // 0 args
func (n *nvapiLinux) call2(fn unsafe.Pointer, arg1, arg2 unsafe.Pointer) uint32  // 2 args
```

### When to Use This Pattern

Use this cross-platform pattern when:
- ✅ Accessing platform-specific libraries with identical APIs (like NvAPI)
- ✅ Calling conventions differ fundamentally (syscall vs cgo)
- ✅ Data structures and logic are identical across platforms

**Do NOT use this pattern for:**
- ❌ Simple platform detection (use build tags only)
- ❌ Different APIs per platform (keep everything separate)
- ❌ When the abstraction becomes more complex than the duplication

### Testing Considerations

| Platform | Status | Notes |
|----------|--------|-------|
| Windows (native) | ✅ Tested | Windows binary runs on Windows with nvapi64.dll |
| Windows binary on WSL | ✅ Tested | WSL interop allows Windows .exe to access Windows nvapi64.dll |
| Linux (native) | ❌ Untested | Requires native Linux with libnvidia-api.so.1 from NVIDIA driver |
| Linux binary on WSL | ❌ Cannot test | libnvidia-api.so.1 not available in WSL's Linux environment |

**Important:** WSL can run Windows binaries (which access Windows drivers), but the Linux build cannot be tested on WSL because `libnvidia-api.so.1` is only available on native Linux systems with NVIDIA drivers.

**Pre-commit checks:**
```bash
go build ./...    # Builds current platform only
go vet ./...      # Ensure no unsafe.Pointer warnings
go test ./...     # Skip if no NVIDIA hardware
```

### OC Scanner and Hardware Profile Behavior

**Critical:** OC Scanner and MSI Afterburner hardware profiles modify the driver's internal boost tables **directly at a level below NvAPI**. This affects how you interpret `VFPoint` data:

| Field | What It Shows | OC Scanner Scenario |
|-------|---------------|---------------------|
| **BaseFreqMHz** | Current driver state | ✅ **Includes OC Scanner** (e.g., 2317 MHz) |
| **OffsetMHz** | NvAPI SetControl only | ❌ Always 0 (OC Scanner doesn't use SetControl) |
| **EffectiveMHz** | Actual GPU frequency | ✅ Matches applied curve (e.g., 2317 MHz) |

**Example at 850 mV with OC Scanner applied:**

|.cfg file (msiaf)|NvAPI (nvvf)|
|-----------------|------------|
|Voltage: 850 mV|Voltage: 850 mV|
|OC Ref: 1365 MHz (f2)|BaseFreqMHz: 2317 MHz|
|Offset: +952 MHz (f0)|OffsetMHz: 0 MHz|
|Effective: 2317 MHz|EffectiveMHz: 2317 MHz|

**Key takeaways:**
1. OC Scanner profiles ARE applied correctly - NvAPI shows the final result
2. OffsetMHz = 0 is expected - OC Scanner doesn't use NvAPI SetControl
3. To see OC Scanner offsets, parse .cfg files with the `msiaf` package
4. To verify OC Scanner is working, compare EffectiveMHz from both sources

**You cannot detect OC Scanner offsets from NvAPI alone.** The driver has already applied OC Scanner's math internally, so NvAPI reads back the modified curve as the "base" frequency.

**To analyze OC Scanner configurations**, parse the hardware profile `.cfg` files using the `msiaf` package, which stores both `BaseFreqMHz` (baseline frequency, possibly from OC Scanner) and `OffsetMHz` separately.

---

## 5. FAN CURVE SERIALIZATION

**Location:** `msiaf/fancurve.go`

The software auto fan control curve (`SwAutoFanControlCurve`) uses a 256-byte binary format stored in the MSI Afterburner configuration file.

### Binary Format Documentation

**See package-level comments in `msiaf/fancurve.go`** for complete specification.

### Key Characteristics

| Characteristic | Detail |
|----------------|--------|
| **Version** | Must match `FanCurveBinaryFormatVersion` (0x00010000 = v1.0) |
| **Size** | Fixed 256 bytes |
| **Temperature range** | -50 to 150°C (sanity checks, not hardware limits) |
| **Fan speed range** | 0-100% (physical limits) |
| **Point ordering** | MUST be sorted by temperature ascending for correct interpolation |

### Design Principle

**Strict validation with detailed error reporting** - Uses `FanCurveError` type to provide field-level error information. All parsing functions return errors - no silent failures! This is critical for hardware safety.

### Testing

| Test Type | Location |
|-----------|----------|
| Unit tests | `msiaf/fancurve_test.go` |
| Integration tests | `msiaf/globalconfig_test.go` |

---

## 6. V-F CURVE BINARY FORMAT

**Location:** `msiaf/vfcurve.go` (full specification in package-level comments)

The voltage-frequency curve (`VFCurve`) uses a binary format stored as a hex blob in hardware profile `.cfg` files.

### Quick Reference

| Component | Size | Format |
|-----------|------|--------|
| **Header** | 12 bytes | `[version:uint32][count:uint32][reserved:float32=0.0]` |
| **Per point** | 12 bytes | `[voltage:float32][oc_ref:float32][offset:float32]` |
| **Inactive marker** | - | `oc_ref = 225.0` (stock behavior) |
| **Applied frequency** | - | `BaseFreq(v) + offset` (computed from blob data) |

### Key Design Principle

The `.cfg` blob is **self-contained** and provides all data needed to compute applied frequencies. The formula is:

```
AppliedFreq(v) = BaseFreqMHz(v) + OffsetMHz
```

Where:
- `BaseFreqMHz` = Baseline frequency (f1 field, may originate from OC Scanner or manual editing)
- `OffsetMHz` = User's frequency offset relative to the baseline

The `vfcurve.go` implementation exposes all three values (voltage, base, offset) directly on `VFPoint`. Computing applied frequencies is straightforward: `point.BaseFreqMHz + point.OffsetMHz`.

**Important distinction: Configured vs. Actual**

| Source | What It Shows | Use Case |
|--------|---------------|----------|
| **`.cfg` file (msiaf)** | Configured values (what user set up) | Analyze profile settings, replicate configs |
| **NvAPI runtime (nvvf)** | Actual values (what GPU is running) | Verify profile is applied, debug issues |

The `.cfg` file tells you what the user *intended* to apply. To verify what's *actually running* on the GPU, you must read the live V-F curve via NvAPI (`nvvf` package). Discrepancies can occur if:
- The profile hasn't been loaded yet
- MSI Afterburner isn't running
- Driver reset or system sleep occurred
- Another tool modified the curve

**Best practice:** Compare both sources to verify profiles are working correctly (see Section 4, "OC Scanner and Hardware Profile Behavior").

### Marshaling (Writing V-F Curves)

```go
// Serialize V-F curve to hex string (no "0x" prefix or "h" suffix)
hexData, err := curve.Marshal()

// Apply flat offset to all active points
curve.ApplyFlatOffset(100)  // +100 MHz overclock
curve.ApplyFlatOffset(-50)  // -50 MHz undervolt
```

### Relationship to nvvf Package

| Capability | `msiaf` (cfg parsing) | `nvvf` (NvAPI runtime) |
|------------|----------------------|------------------------|
| Read baseline frequency (f1) | ✅ Yes (from blob) | ❌ No (already combined into BaseFreq) |
| Read user's frequency offset | ✅ Yes (from blob) | ⚠️ Only NvAPI SetControl offsets |
| Compute applied frequency | ✅ `BaseFreq + Offset` | ⚠️ `EffectiveMHz` only (no breakdown) |
| Detect OC Scanner usage | ⚠️ Possible (check f1 pattern) | ❌ No |
| Show **configured** values | ✅ Yes | N/A |
| Show **actual runtime** values | ❌ No | ✅ Yes |

See `nvvf/README.md` for details on NvAPI's limitations with OC Scanner detection.

### Testing

| Test Type | Location |
|-----------|----------|
| Unit tests | `msiaf/vfcurve_test.go` |
| Integration tests | Use actual profile files from `LocalProfiles/` |
| Verification | Parsed values match MSI Afterburner UI (e.g., 1000 mV → +43 MHz offset) |

---

## 7. PROFILE MATCHING

**Location:** `msiaf/active.go`

The profile matching system detects which hardware profile slot (Startup or Profile1-5) is currently active by comparing the live GPU V-F curve against saved profiles.

### How It Works

1. Read live V-F curve from NVIDIA driver using `nvvf.ReadNvAPIVF()`
2. Build voltage→frequency map from live data
3. Compare against each profile slot's V-F curve
4. Return confidence scores for each slot

### Key Types

| Type | Purpose |
|------|---------|
| `ProfileMatchResult` | Contains match statistics (confidence, matched points, deviations) |
| `ProfileMatchResult.Slot` | Slot number (0=Startup, 1-5=Profile1-5) |
| `ProfileMatchResult.SlotName` | Human-readable name ("Startup", "Profile1", etc.) |
| `ProfileMatchResult.MatchConfidence` | 0.0 to 1.0 confidence score |
| `ProfileMatchResult.IsMatch(threshold)` | Returns true if confidence meets threshold |

### Key Functions

| Function | Purpose |
|----------|---------|
| `MatchVFCurve(liveFreqs, profileCurve, toleranceMHz)` | Compare single V-F curve against live data |
| `MatchProfileAgainstLive(liveFreqs, hwProfile, toleranceMHz)` | Compare all profile slots, returns []ProfileMatchResult |
| `FindBestMatch(results, threshold)` | Find slot with highest confidence |

### Usage Pattern

```go
// Read live state from nvvf
nvvfPoints, err := nvvf.ReadNvAPIVF(0)
if err != nil {
    // handle error
}

// Build voltage→frequency map
liveFreqs := make(map[float32]float64)
for _, pt := range nvvfPoints {
    liveFreqs[float32(pt.VoltageMV)] = pt.BaseFreqMHz
}

// Load hardware profile
hwProfile, err := profileInfo.LoadProfile()
if err != nil {
    // handle error
}

// Match against all slots (tolerance: 10 MHz typical)
results, err := msiaf.MatchProfileAgainstLive(liveFreqs, hwProfile, 10.0)
if err != nil {
    // handle error
}

// Find active profile (80% confidence threshold)
best, isMatch := msiaf.FindBestMatch(results, 0.8)
if isMatch {
    fmt.Printf("Active: %s (%.0f%% confidence)\n", best.SlotName, best.MatchConfidence*100)
}
```

### Matching Algorithm

| Aspect | Detail |
|--------|--------|
| **Comparison** | `liveFreq` vs `BaseFreqMHz + OffsetMHz` |
| **Active points only** | Inactive points (BaseFreqMHz=225.0) are skipped |
| **Tolerance** | Accounts for floating-point precision (typical: 5-10 MHz) |
| **Confidence** | Ratio of matched points to total comparable points |
| **Best match** | Highest confidence wins (first slot breaks ties) |

### Testing Guidelines

- Use realistic tolerance (5-10 MHz)
- Account for incomplete live data (some voltages may be missing)
- Test with both perfect matches and partial matches
- Verify confidence scores reflect actual match quality

---

## 8. PROFILE WRITING

**Location:** `msiaf/profile.go`, `msiaf/vfcurve.go`

Profiles can be modified and saved back to disk with proper serialization.

### V-F Curve Marshaling

```go
// Serialize V-F curve to hex string
hexData, err := curve.Marshal()
if err != nil {
    return err
}

// Apply flat offset to all active points
curve.ApplyFlatOffset(100) // +100 MHz overclock
```

### Profile Saving

| Method | Purpose |
|--------|---------|
| `hwProfile.Save()` | Save to original path (auto-backup as `.bak`) |
| `hwProfile.SaveAs(path)` | Save to custom path |
| `section.SetVFCurveFromCurve(curve)` | Update section with new V-F curve |
| `section.SetVFCurve(hexData)` | Update section with raw hex string |

### Serialization Format

| Aspect | Detail |
|--------|--------|
| **All keys written** | Empty values for nil fields: `Key=` |
| **Hex case** | Uppercase (`%X` format) |
| **VFCurve suffix** | `h` (e.g., `VFCurve=ABCD1234h`) |
| **Compatibility** | Matches MSI Afterburner's expected format |

### Safety Features

- `Save()` creates automatic backup before overwriting (`.bak` extension)
- Backup restored if write fails
- Backup removed on success
- Use `SaveAs()` for creating new files without backup

### Example: Update and Save Profile

```go
// Load profile
hwProfile, err := profileInfo.LoadProfile()
if err != nil {
    return err
}

// Get current V-F curve
hexData := fmt.Sprintf("%x", hwProfile.Startup.VFCurve)
curve, err := msiaf.UnmarshalVFControlCurve(hexData)
if err != nil {
    return err
}

// Modify curve
curve.ApplyFlatOffset(100) // +100 MHz overclock

// Update profile
err = hwProfile.Startup.SetVFCurveFromCurve(curve)
if err != nil {
    return err
}

// Save (creates backup automatically)
err = hwProfile.Save()
if err != nil {
    return err
}
```

### Testing Guidelines

- Always test round-trip: parse → modify → serialize → parse again
- Verify VFCurve hex is identical after round-trip
- Test backup creation and restoration on failure
- Clean up test files after verification

---

---

## 7. HARDWARE PROFILE PARSING

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

| Value | Meaning |
|-------|---------|
| `nil` | Field not present in file |
| `&value` | Field explicitly set (even if 0) |

**Benefits:**
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

---

## 8. WORKFLOW GUIDELINES

### When Working on GPU-Related Features

1. **Check if catalog needs updating** → Run `go generate ./msiaf/...`
2. **Verify device IDs** → Look up in `catalog_generated.go`
3. **Test thoroughly** → Run `go test ./msiaf/...` after changes
4. **Commit generated file** → `catalog_generated.go` should be committed

### When Moving or Restructuring Files

1. Update `package` declarations in moved files
2. Update `//go:generate` paths to account for new directory depth
3. Update generator (`cmd/gencatalog/main.go`) if it hardcodes package names
4. Regenerate → `go generate ./...`
5. Verify build → `go build ./...`
6. Verify tests → `go test ./...`

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
2. Re-run generator → `go generate ./msiaf/...`
3. Update tests in `msiaf/catalog/catalog_test.go`
4. Verify → `go build ./...` and `go test ./...`

**Extending the scanning functionality:**
1. Add functions/methods to `msiaf/scan.go`
2. Use catalog subpackage for GPU lookups (don't duplicate logic)
3. Add methods to `HardwareProfileInfo` for value-added helpers
4. Update tests in `msiaf/scan_test.go`
5. Verify → `go build ./...` and `go test ./...`

---

## 9. CODE QUALITY GUIDELINES

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

| Location | Path | Levels Up |
|----------|------|-----------|
| From `msiaf/catalog/catalog.go` | `../../cmd/gencatalog/main.go` | 2 levels |
| From `msiaf/scan.go` | `../cmd/gencatalog/main.go` | 1 level |

**Note:** The generator writes to the **current directory**, so run it from the correct location.

### Package Consistency

When moving files to subpackages:
1. Update the `package` declaration in all moved files
2. Update the generator to output the correct package name
3. Update `go:generate` paths to reflect new directory depth
4. Verify imports in dependent files

---

## 10. API USAGE EXAMPLES

### Command-Line Examples

The project includes three example programs in the `cmd/` directory:

| Example | Purpose | Requirements |
|---------|---------|--------------|
| `cmd/msiaf/main.go` | Scan MSI Afterburner profiles and configs | None (offline parsing) |
| `cmd/nvvf/main.go` | Read live V-F curves from NVIDIA GPU | NVIDIA GPU with NvAPI |
| `cmd/active/main.go` | Match profiles against live V-F curve | Windows + NVIDIA GPU |

**Build and run:**
```bash
# Windows-only example (profile matching)
GOOS=windows GOARCH=amd64 go build -o active.exe ./cmd/active/
.\active.exe  # Scans C:\Program Files (x86)\MSI Afterburner\Profiles by default

# Cross-platform examples
go run cmd/msiaf/main.go  # Profile scanning (Linux/Windows)
go run cmd/nvvf/main.go   # Live V-F reading (requires GPU)
```

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

### Profile Matching (Windows + NVIDIA GPU)

Match MSI Afterburner profiles against live V-F curve data:

```go
package main

import (
    "fmt"
    "github.com/hekmon/aiup/msiaf"
    "github.com/hekmon/aiup/nvvf"
)

func main() {
    // Step 1: Scan profiles
    result, err := msiaf.Scan("LocalProfiles")
    
    // Step 2: Read live V-F curve from GPU 0
    livePoints, err := nvvf.ReadNvAPIVF(0)
    
    // Build liveFreqs map: voltage (mV) -> frequency (MHz)
    liveFreqs := make(map[float32]float64)
    for _, pt := range livePoints {
        liveFreqs[float32(pt.VoltageMV)] = pt.BaseFreqMHz
    }
    
    // Step 3: Load hardware profile
    hwProfile, err := result.HardwareProfiles[0].LoadProfile()
    
    // Step 4: Match all slots against live data (10 MHz tolerance)
    results, err := msiaf.MatchProfileAgainstLive(liveFreqs, hwProfile, 10.0)
    
    // Step 5: Find best match (50% confidence threshold)
    bestResult, isMatch := msiaf.FindBestMatch(results, 0.5)
    
    // Display results
    for _, r := range results {
        marker := "  "
        if r.Slot == bestResult.Slot && isMatch {
            marker = "← "
        }
        fmt.Printf("  %s%s: %s\n", marker, r.SlotName, r.String())
    }
}
```

**Key Functions:**
- `msiaf.MatchProfileAgainstLive()` - Compare all profile slots against live data
- `msiaf.MatchVFCurve()` - Compare single curve against live data
- `msiaf.FindBestMatch()` - Find slot with highest confidence
- `ProfileMatchResult.IsMatch(threshold)` - Check if match meets confidence threshold

**Output Example:**
```
  Startup:     100% confidence (73/73 points matched, avg deviation: 0.0 MHz)
  Profile1:    100% confidence (73/73 points matched, avg deviation: 0.0 MHz)
  Profile2:    0% confidence (0/0 points matched, avg deviation: 0.0 MHz)
  
✓ Status: Normal - Startup profile is applied and matches a saved profile
```

---

## 🐛 AGENT PITFALLS

**Common mistakes AI agents make on this project:**

| Mistake | Why It Happens | Fix |
|---------|----------------|-----|
| Editing `catalog_generated.go` | Looks like normal Go code | **NEVER** - regenerate via `go generate ./msiaf/...` |
| Guessing device IDs | Using IDs from external databases | Look up in `catalog_generated.go` or regenerate |
| Wrong `//go:generate` path | Forgetting to update path depth | Count levels: `../../cmd/...` from subpackage |
| Creating wrapper functions | Habit to centralize access | Add method to existing type instead |
| Skipping pre-commit checks | Wanting to save time | Run ALL four: generate, build, test, vet |
| Not running diagnostics | Forgetting gopls suggestions | Use `diagnostics` tool before finalizing |
| Mixing concerns in files | Unclear file boundaries | Follow "new scopes deserve new files" |
| Lazy struct fields | Using `Raw map[string]string` | Parse everything during parsing |
| Silent failures in parsers | Not wanting verbose errors | Return errors - hardware safety depends on it |
| Not cleaning up `tmp/` | Forgetting experiments | Remove temp tools after implementation |

---

## 🔧 TROUBLESHOOTING

| Issue | Solution |
|-------|----------|
| "unknown GPU" errors | Run `go generate ./msiaf/...` to update the catalog |
| Build fails with "package not found" | Check `package` declarations and import paths |
| `go generate` fails | Check `//go:generate` path for correct directory depth |
| Tests fail after moving files | Ensure generator outputs correct package name, re-run `go generate ./...` |
| Import errors with catalog | Import as `github.com/hekmon/aiup/msiaf/catalog` (root package doesn't re-export catalog) |
| Diagnostics show `strings.Cut` suggestion | Use `strings.Cut()` instead of `strings.SplitN()` for 2-part splits |
| Deprecated API warnings | Check Go version and update to modern equivalents |
| Fan curve parsing fails | Verify version byte matches `FanCurveBinaryFormatVersion` (0x00010000) |
| V-F curve offset seems wrong | Remember: offset is added to hardware boost (driver-private), not absolute frequency |
| Profile matching shows low confidence | Check if live data has enough voltage points; adjust tolerance (5-10 MHz typical) |
| Save() fails with "no file path" | Profile was created programmatically; use `SaveAs()` instead |
| Serialized profile has extra empty keys | Expected behavior - all keys written with empty values for nil fields |

---

## 📋 SESSION CHECKLIST

**Start of Session:**
- [ ] Read this AGENT.md file
- [ ] Understand the task requirements
- [ ] Present plan to user before implementing

**During Development:**
- [ ] Temporary code goes in `tmp/<experiment_name>/` (not in root or packages)

**End of Successful Session:**
- [ ] Clean up `tmp/` directory (remove all experiment folders)

**During Development:**
- [ ] Follow file organization principle (new scopes = new files)
- [ ] Use strong typing (parse during parsing)
- [ ] Add methods to existing types, not wrappers
- [ ] Write tests alongside implementation

**Before Presenting Results:**
- [ ] Run `go generate ./...`
- [ ] Run `go build ./...`
- [ ] Run `go test ./...`
- [ ] Run `go vet ./...`
- [ ] Check diagnostics tool
- [ ] Address all linting suggestions

**End of Successful Session:**
- [ ] Propose AGENT.md updates if new patterns discovered
- [ ] Clean up any `tmp/` experiment directories
- [ ] Ensure generated files are committed