# Overclocking Package

**Package:** `github.com/hekmon/aiup/overclocking`

**Purpose:** High-level GPU overclocking orchestration layer that combines MSI Afterburner profile management (`msiaf`) with live NVIDIA GPU telemetry (`nvvf`).

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

- **OC Scanner Integration** - Execute OC Scanner workflows, parse results, compare before/after curves
- **Profile Management** - Load, compare, diff, and validate hardware profiles
- **Safety Validation** - Check voltage, temperature, and power limits against architecture-specific thresholds
- **Session Tracking** - Maintain state across multiple operations with checkpoint/restore support

### What This Package Does NOT Do

- **Direct hardware access** - Delegates to `msiaf` (file I/O) and `nvvf` (NvAPI)
- **GPU architecture detection** - Uses data from `msiaf/catalog`
- **User interaction** - Pure library, no CLI or prompts
- **Automated tuning decisions** - Provides tools, not policy (reserved for agents)

---

## 📦 ARCHITECTURE

### Package Structure

```
overclock/
├── README.md              # This file - package documentation and guidelines
├── session.go             # Session management (baseline, current, history)
├── scanner.go             # OC Scanner integration and analysis
├── profile.go             # Profile comparison and diffing
├── safety.go              # Safety limits and validation
└── overclock_test.go      # Unit and integration tests
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

### JSON-Marshallable Results (CRITICAL)

**All exported functions must return JSON-marshallable data structures.**

This requirement ensures MCP Server compatibility - any function result should be serializable to JSON for transmission to MCP clients.

**Do:**
- Return structs with exported fields
- Use standard Go types (int, float64, string, bool, slices, maps)
- Use `time.Time` for timestamps (JSON-serializable)
- Use pointers for optional fields (nil = not present)
- Implement `MarshalJSON()` only when necessary for custom formatting

**Don't:**
- Return unexported types from public functions
- Include channels or functions in result structs
- Return raw errors without structured error types
- Use types that don't serialize cleanly to JSON

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
go test ./overclock/...
```

### Coverage Goals

| Component | Target |
|-----------|--------|
| Safety validation | 100% (critical path) |
| Session management | 90% |
| OC Scanner integration | 80% (requires mocks) |
| Profile comparison | 90% |

### Mocking Strategy

Use interfaces to mock external dependencies (`msiaf`, `nvvf`) for unit testing.

---

## 📝 VERSION HISTORY

| Version | Date | Changes |
|---------|------|---------|
| 0.1.0 | TBD | Initial package structure and API definition |

---

## ⚖️ LICENSE

Same as parent project - see [LICENSE](../LICENSE).