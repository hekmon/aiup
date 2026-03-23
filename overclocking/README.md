# Overclocking Package

**Package:** `github.com/hekmon/aiup/overclocking`

**Purpose:** High-level GPU overclocking orchestration layer that combines MSI Afterburner profile management (`msiaf`) with live NVIDIA GPU telemetry (`nvvf`).

**Status:** ✅ Production-ready (GPU Discovery implemented and tested)

**Target Consumers:**
1. **Terminal Agent** - Interactive CLI guiding users through overclocking workflows
2. **MCP Server** - Machine-readable API for custom agent development

---

## 🎯 DESIGN PHILOSOPHY

### Core Principles

| Principle | Description |
|-----------|-------------|
| **Safety First** | All operations validated against hardware limits before execution |
| **Transparent State** | Always expose raw data - no hidden magic |
| **Idempotent Operations** | Safe to retry failed operations without side effects |
| **Fail Fast** | Validate early, fail with clear error messages |
| **Session-Aware** | Track state across multiple operations |
| **JSON-First** | All function results must be JSON-marshallable for MCP compatibility |

### What This Package Does

The overclocking package serves as the orchestration layer between:

- **`msiaf`** - MSI Afterburner configuration and hardware profile parsing
- **`nvvf`** - Live NVIDIA GPU V-F curve reading via NvAPI

It provides high-level operations that combine these lower-level capabilities:

- **GPU Discovery** - Scan MSI Afterburner profiles, correlate with NvAPI GPUs, validate prerequisites
- **OC Scanner Integration** - Execute OC Scanner workflows, parse results, compare before/after curves
- **Profile Comparison** - Diff two hardware profiles, identify changes
- **Safety Validation** - Check voltage, temperature, and power limits against architecture-specific thresholds
- **Session Management** - Track overclocking state across operations

### What This Package Does NOT Do

- **Direct hardware access** - Delegates to `msiaf` (file I/O) and `nvvf` (NvAPI)
- **GPU architecture detection** - Uses data from `msiaf/catalog`
- **User interaction** - Pure library, no CLI or prompts
- **Automated tuning decisions** - Provides tools, not policy (reserved for agents)
- **Profile application** - User applies profiles via MSI Afterburner UI (this package only prepares them)

---

## 📦 ARCHITECTURE

### Package Structure

```
overclocking/
├── README.md              # This file - package documentation and guidelines
├── discovery.go           # GPU discovery (scan profiles, correlate with NvAPI) ✅ IMPLEMENTED
├── status.go              # Current curve detection (validate Startup, return V-F curve) ✅ IMPLEMENTED
├── session.go             # Session management (baseline, current, history)
├── scanner.go             # OC Scanner integration and analysis
├── profile.go             # Profile comparison and diffing
├── safety.go              # Safety limits and validation
└── discovery_test.go      # Unit tests for discovery ✅ IMPLEMENTED
```

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    OVERCLOCK PACKAGE                         │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────┐ │
│  │ OC Scanner      │  │ Profile         │  │ Safety      │ │
│  │ Integration     │  │ Management      │  │ Validation  │ │
│  └─────────────────┘  └─────────────────┘  └─────────────┘ │
│         ↓                    ↓                   ↓          │
│  • Run scanner        • Load profiles    • Voltage limits  │
│  • Parse results      • Compare states   • Temperature     │
│  • Measure gains      • Apply changes    • Power limits    │
│  • Rollback support   • Backup/restore   • Stability check │
│  └─────────────────────────────────────────────────────────┘ │
│                            ↓                                  │
│              ┌─────────────────────────┐                     │
│              │     Session State       │                     │
│              │  (Baseline, Current)    │                     │
│              └─────────────────────────┘                     │
└─────────────────────────────────────────────────────────────┘
         ↓                           ↓
┌─────────────────┐         ┌─────────────────┐
│   msiaf pkg     │         │   nvvf pkg      │
│  (Profiles)     │         │ (Live Telemetry)│
└─────────────────┘         └─────────────────┘
```

---

## 🔑 CORE RESPONSIBILITIES

### GPU Discovery ✅ IMPLEMENTED

Discovers NVIDIA GPUs and validates MSI Afterburner prerequisites:

- Scans MSI Afterburner Profiles directory for hardware profile .cfg files
- Queries NvAPI for all detected NVIDIA GPUs
- Correlates profiles with GPUs by matching marketing names
- Returns structured GPU information with PCI identifiers
- Validates that every profile has a matching physical GPU
- Reports errors for missing profiles (required for overclocking)

**Key Types:**

```go
type GPUInfo struct {
    Index           int    `json:"index"`            // NvAPI GPU index (0, 1, 2, ...)
    Name            string `json:"name"`             // Marketing name from NvAPI
    VendorID        string `json:"vendor_id"`        // PCI vendor ID
    DeviceID        string `json:"device_id"`        // PCI device ID
    SubsystemID     string `json:"subsystem_id"`     // PCI subsystem ID
    BusNumber       int    `json:"bus_number"`       // PCI bus number
    DeviceNumber    int    `json:"device_number"`    // PCI device number
    FunctionNumber  int    `json:"function_number"`  // PCI function number
    ProfilePath     string `json:"profile_path"`     // Path to .cfg file
    Manufacturer    string `json:"manufacturer"`     // Card manufacturer
    FullDescription string `json:"full_description"` // Complete description
}

type DiscoveryResult struct {
    ProfilesDir      string    `json:"profiles_dir"`
    GlobalConfigPath string    `json:"global_config_path"`
    GPUs             []GPUInfo `json:"gpus"`
    Errors           []string  `json:"errors,omitempty"`
}
```

**Key Functions:**

```go
// ScanGPUs discovers NVIDIA GPUs by scanning MSI Afterburner profiles
func ScanGPUs(profilesDir string) (*DiscoveryResult, error)
```

**Example Usage:**

```go
import "github.com/hekmon/aiup/overclocking"

result, err := overclocking.ScanGPUs(`C:\Program Files (x86)\MSI Afterburner\Profiles`)
if err != nil {
    panic(err)
}

for _, gpu := range result.GPUs {
    fmt.Printf("GPU %d: %s (%s)\n", gpu.Index, gpu.Name, gpu.Manufacturer)
}
```

**CLI Example:**

```bash
# Use default path
overclocking.exe

# Specify custom path
overclocking.exe -profiles "D:\Custom\Profiles"

# Show help
overclocking.exe -h
```

### Current Curve Detection ✅ IMPLEMENTED

Reads the current V-F curve from a GPU and compares it against the Startup profile.

**Purpose:** This function provides AI agents with the currently applied V-F curve, enabling them to:
- Read current overclock offsets at each voltage point
- Check if the live GPU state matches the Startup profile (non-breaking validation)
- Identify which profile slot (if any) the current curve originated from

**Validation:** The function compares the live V-F curve against the Startup profile and reports match status via `LiveMatchesStartup` field. It does **not** error on mismatch - the caller decides how to handle discrepancies (unsaved changes, corruption, wrong profile).

**Profile Match Hint:** Additionally checks if the Startup curve matches any Profile1-5 slot. This is informational - it helps the user decide where to save modifications (e.g., "save back to Profile2" if that's where the current curve originated).

**Key Types:**

```go
// VFPoint represents a single voltage-frequency point with all components explicit.
// OffsetMHz is the CORE overclocking value that gets set/modified.
type VFPoint struct {
    VoltageMV        float64 `json:"voltage_mv"`         // e.g., 850.0 mV
    BaseFreqMHz      float64 `json:"base_freq_mhz"`      // Hardware base frequency
    OffsetMHz        float64 `json:"offset_mhz"`         // ⭐ CORE OC VALUE
    EffectiveFreqMHz float64 `json:"effective_freq_mhz"` // BaseFreqMHz + OffsetMHz
}

// SavedProfileInfo indicates which profile slot matches the current curve.
// This is informational - helps the user decide where to save modifications.
type SavedProfileInfo struct {
    SlotNumber int     `json:"slot_number"` // 1-5 (Profile1-5)
    SlotName   string  `json:"slot_name"`   // "Profile1", "Profile2", etc.
    Confidence float64 `json:"confidence"`  // Match confidence (0.0-1.0)
}

// CurrentStateResult contains the complete current GPU overclocking state.
type CurrentStateResult struct {
    // V-F Curve (core overclock)
    Points []VFPoint `json:"points"` // All voltage points with offsets

    // Memory overclock
    MemClkBoostMHz int `json:"mem_clk_boost_mhz"` // Memory clock offset in MHz (e.g., 3000)

    // Power and thermal limits
    PowerLimitPercent int `json:"power_limit_percent"` // Power limit percentage (e.g., 100)

    // Fan settings
    FanMode         string `json:"fan_mode"`          // "auto" or "manual"
    FanSpeedPercent *int   `json:"fan_speed_percent"` // Manual fan speed (0-100) if in manual mode

    // Profile matching info
    LiveMatchesStartup bool              `json:"live_matches_startup"` // true if live curve matches Startup profile
    Profile            *SavedProfileInfo `json:"profile"`              // Which saved profile slot matches (null if none)
}
```

**Key Functions:**

```go
// GetCurrentState reads the complete current GPU overclocking state and compares it against the Startup profile.
// The profilePath parameter comes from DiscoveryResult.GPUs[i].ProfilePath.
func GetCurrentState(gpuIndex int, profilePath string) (*CurrentStateResult, error)
```

**Example Usage:**

```go
import "github.com/hekmon/aiup/overclocking"

// Step 1: Discover GPUs and their profile paths
discovery, err := overclocking.ScanGPUs(profilesDir)
if err != nil {
    return fmt.Errorf("failed to scan GPUs: %w", err)
}

// Step 2: Get current GPU state for GPU 0
result, err := overclocking.GetCurrentState(0, discovery.GPUs[0].ProfilePath)
if err != nil {
    return fmt.Errorf("failed to get current state: %w", err)
}

// Check if live curve matches Startup (non-breaking validation)
if !result.LiveMatchesStartup {
    fmt.Println("Warning: Live curve differs from Startup profile (unsaved changes?)")
}

// Access complete current state
fmt.Printf("Memory overclock: +%d MHz\n", result.MemClkBoostMHz)
fmt.Printf("Power limit: %d%%\n", result.PowerLimitPercent)
fmt.Printf("Fan mode: %s", result.FanMode)
if result.FanSpeedPercent != nil {
    fmt.Printf(" (%d%%)\n", *result.FanSpeedPercent)
}

// Read current V-F curve offsets
for _, pt := range result.Points {
    fmt.Printf("At %v mV: %v MHz offset\n", pt.VoltageMV, pt.OffsetMHz)
}

// Modify offsets (AI agent workflow)
result.Points[0].OffsetMHz = 1000 // Set 1000 MHz offset at first voltage point

// Check if current curve matches a saved profile (hint for save location)
if result.Profile != nil {
    fmt.Printf("Current curve matches %s - save to same slot?\n", result.Profile.SlotName)
} else {
    fmt.Println("Current curve is unique - save to any slot (Profile1-5)")
}
```

### Session Management

Maintains overclocking session state for a specific GPU:

- Captures baseline V-F curve on session start
- Tracks current live V-F curve state
- Supports checkpoint/restore for safe experimentation
- Records session history for audit trails
- Exposes GPU metadata (name, architecture)

### OC Scanner Integration

Orchestrates complete OC Scanner workflows:

- Captures pre-scan V-F curve baseline
- Triggers OC Scanner execution via MSI Afterburner
- Monitors scan completion
- Captures post-scan V-F curve
- Calculates per-point frequency gains
- Exports results as MSI Afterburner hardware profiles
- Provides confidence scoring for scan quality

### Profile Comparison

Enables detailed profile analysis:

- Compares two hardware profiles
- Identifies clock offset changes (core, memory)
- Detects V-F curve modifications per voltage point
- Reports power limit differences
- Tracks fan mode changes
- Produces human-readable summaries

### Safety Validation

Enforces architecture-specific safety limits:

- Voltage thresholds per GPU architecture
- Temperature limits to prevent thermal throttling
- Power headroom validation
- Clock offset sanity checks
- Full profile validation before application
- Structured error types for programmatic handling

---

## 📝 DEVELOPER GUIDELINES

### Package Rules (CRITICAL)

These rules apply to all code in the overclocking package:

#### 0. MCP Function Design Rules

**All exported functions are MCP entry points and must follow these rules:**

| Rule | Requirement | Example |
|------|-------------|---------|
| Simple parameters | Only `int`, `string`, `bool`, slices | ✅ `func GetCurrentState(gpuIndex int, profilePath string)` |
| JSON-serializable return | Struct with json tags + error | ✅ `(*CurrentStateResult, error)` |
| No complex input types | Don't require caller to construct structs | ❌ `func GetCurrentState(gpu *GPUInfo)` |
| Non-breaking validation | Report issues in result, don't error | ✅ `LiveMatchesStartup: false` instead of `return nil, err` |

**Rationale:** MCP clients can only pass simple JSON values. Complex types must be constructed internally or returned, never required as input.

#### 1. JSON Serialization Required

**All exported structs must have JSON tags** for API/MCP compatibility:

```go
type GPUInfo struct {
    Index     int    `json:"index"`
    Name      string `json:"name"`
    VendorID  string `json:"vendor_id"`
    // ...
}
```

**Guidelines:**
- Use `snake_case` for JSON field names (Go convention for APIs)
- Use `omitempty` for optional/sparse fields (e.g., `Errors []string`)
- All fields should be exported (capitalized) for JSON marshaling
- Use standard Go types (int, float64, string, bool, time.Time, slices, maps)

#### 2. Hide Low-Level Complexity

**Users should never import `nvvf` or `msiaf` directly.**

The overclocking package is a **high-level orchestration layer** that:
- Imports `nvvf` and `msiaf` internally
- Converts all low-level types to overclocking package types
- Returns only JSON-serializable overclocking types

**Example:**

```go
// ✅ Good: User only imports overclocking
import "github.com/hekmon/aiup/overclocking"

result, err := overclocking.ScanGPUs("/path/to/Profiles")
fmt.Println(result.GPUs[0].Name) // No nvvf/msiaf types exposed

// ❌ Bad: User shouldn't need to know about nvvf/msiaf
import (
    "github.com/hekmon/aiup/overclocking"
    "github.com/hekmon/aiup/nvvf"  // Don't do this!
)
```

#### 3. Profiles Are Mandatory

Overclocking requires a profile to apply offsets. If no profile exists:

- Return an error (do not proceed silently)
- Tell the user to create a profile in MSI Afterburner first
- Do not attempt to create profiles automatically

### JSON-Marshallable Results (CRITICAL)

**All exported functions must return JSON-marshallable data structures.**

This requirement ensures MCP Server compatibility - any function result should be serializable to JSON for transmission to MCP clients.

**Do:**
- Return structs with exported fields
- Use standard Go types (int, float64, string, bool, slices, maps)
- Use `snake_case` for JSON field names (e.g., `vendor_id`, `profile_path`)
- Use `omitempty` for optional/sparse fields (e.g., `Errors []string`)
- Use pointers for optional fields (nil = not present)
- Implement `MarshalJSON()` only when necessary for custom formatting

**Don't:**
- Return unexported types from public functions
- Include channels or functions in result structs
- Return raw errors without structured error types
- Use types that don't serialize cleanly to JSON (e.g., `time.Time` unless needed)

### Error Handling

**Use structured error types for safety violations:**

```
SafetyError {
    Parameter: string  // What was validated
    Value: int         // Actual value provided
    Limit: int         // Safe limit exceeded
    Message: string    // Human-readable explanation
}
```

This allows MCP clients and terminal agents to programmatically inspect errors and present meaningful feedback to users.

### Naming Conventions

| Type | Convention | Example |
|------|------------|---------|
| Discovery types | `Discovery<Thing>` | `DiscoveryResult`, `GPUInfo` |
| Session types | `<Thing>Session` | `OCScannerSession` |
| Result types | `<Thing>Result` | `ScanResult` |
| Diff types | `<Thing>Diff` | `ProfileDiff` |
| Error types | `<Thing>Error` | `SafetyError` |
| Limit types | `SafetyLimits` | `SafetyLimits` |

### Function Design

**Prefer focused, composable functions:**

| Pattern | Example |
|---------|---------|
| Single responsibility | `RunOCScanner()` does scanning only |
| Explicit inputs | Pass profile pointers, don't discover internally |
| Clear outputs | Return result structs, not multiple bare values |
| No side effects | Don't modify global state |

### Session Design

Sessions should be:

- **Explicit** - Created with `NewSession(gpuIndex)`
- **Self-contained** - Hold all state needed for operations
- **Disposable** - Clean shutdown with `Close()`
- **Inspectable** - All fields exported for debugging

---

## 🔒 SAFETY CONSIDERATIONS

### Validation Rules

| Parameter | Rule | Rationale |
|-----------|------|-----------|
| Voltage | Must not exceed architecture maximum | Prevents silicon degradation |
| Temperature | Must not exceed thermal throttle point | Prevents instability |
| Power Limit | Must not exceed PSU headroom | Prevents system crashes |
| Core Offset | Must not exceed tested stable range | Prevents application errors |
| Memory Offset | Must not exceed VRAM tolerance | Prevents visual artifacts |

### Best Practices

1. **Always validate before applying** - Call validation functions before saving or applying profiles
2. **Checkpoint before changes** - Use session checkpoints before modifications
3. **Monitor during stress testing** - Refresh session state periodically
4. **One change at a time** - Isolate variables when debugging stability
5. **Document workflow** - Use descriptive checkpoint names for audit trails

---

## 🔗 INTEGRATION NOTES

### For Terminal Agent Developers

The terminal agent imports this package to provide interactive guidance:

- Create sessions to track user progress
- Run OC Scanner as part of guided workflows
- Validate profiles before recommending application
- Use checkpoint/restore for safe experimentation
- Display structured error messages to users

### For MCP Server Developers

The MCP server exposes this package's capabilities as tools:

- Each tool maps to a package function
- Tool inputs are JSON objects
- Tool outputs are JSON-marshallable result structs
- Errors are structured for client-side handling
- Session state may persist across tool calls

### Platform Considerations

| Platform | Support | Notes |
|----------|---------|-------|
| Windows | Full | Native NvAPI and MSI Afterburner access |
| WSL | Via Windows interop | Build Windows binary, run through WSL |
| Native Linux | Partial | Requires `libnvidia-api.so.1`, no OC Scanner |

---

## 📋 FUTURE CONSIDERATIONS

### Planned Features

| Feature | Description | Priority |
|---------|-------------|----------|
| Session persistence | Save/load session state to disk | High |
| Automated stability testing | Stress test → measure → rollback loops | Medium |
| Stability scoring | Score based on error rates during testing | Medium |
| AMD/Intel support | Extend beyond NVIDIA GPUs | Low |

### Known Limitations

| Limitation | Workaround |
|------------|------------|
| OC Scanner requires MSI Afterburner | Document as prerequisite |
| No direct profile application | User applies via MSI Afterburner UI |
| Single GPU per session | Create multiple sessions for multi-GPU |
| Windows-only OC Scanner | Linux requires alternative approach |

---

## 📚 RELATED PACKAGES

| Package | Purpose | Import |
|---------|---------|--------|
| `msiaf` | MSI Afterburner profile parsing | `github.com/hekmon/aiup/msiaf` |
| `msiaf/catalog` | GPU and manufacturer lookup | `github.com/hekmon/aiup/msiaf/catalog` |
| `nvvf` | Live NVIDIA V-F curve reading | `github.com/hekmon/aiup/nvvf` |

---

## 🧪 TESTING

### Running Tests

```bash
go test ./overclocking/...
```

### Coverage Goals

| Component | Target | Status |
|-----------|--------|--------|
| GPU Discovery | 100% | ✅ Implemented |
| Safety validation | 100% (critical path) | Planned |
| Session management | 90% | Planned |
| OC Scanner integration | 80% (requires mocks) | Planned |
| Profile comparison | 90% | Planned |

### Mocking Strategy

Use interfaces to mock external dependencies (`msiaf`, `nvvf`) for unit testing.

### Test Results

```
=== RUN   TestMatchGPUName
=== RUN   TestMatchGPUName/exact_match_with_manufacturer
=== RUN   TestMatchGPUName/exact_match_no_manufacturer
# ... 9 test cases, all passing
--- PASS: TestMatchGPUName (0.00s)

=== RUN   TestGPUInfoJSONSerialization
--- PASS: TestGPUInfoJSONSerialization (0.00s)

=== RUN   TestDiscoveryResultJSONSerialization
--- PASS: TestDiscoveryResultJSONSerialization (0.00s)
```

---

## 📝 VERSION HISTORY

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | 2024 | Initial package structure and API definition |
| 0.1.1 | 2024 | GPU Discovery implemented and tested (`ScanGPUs`, `GPUInfo`, `DiscoveryResult`) |

---

## ⚖️ LICENSE

Same as parent project - see [LICENSE](../LICENSE).