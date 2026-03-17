# AI Agent Guidelines

This document provides guidelines and important information for AI agents working on this project.

## Package Structure

```
msiaf/
├── scan.go                  # Scanning functionality + value-added methods
├── scan_test.go
└── catalog/                 # GPU catalog subpackage
    ├── catalog.go           # Lookup functions (hand-written)
    ├── catalog_generated.go # Auto-generated data (DO NOT EDIT)
    └── catalog_test.go      # Tests
```

The `msiaf` package is organized with:
- **Root package** (`msiaf/`): Scanning functionality + value-added methods on types
- **Catalog subpackage** (`msiaf/catalog/`): Pure GPU lookup functions (importable by anyone)

## API Design Philosophy

**"Don't hide anything, but pack convenience helpers where users will naturally find them."**

This principle guides the package design:

1. **Transparency**: All raw data (VendorID, DeviceID, SubsystemID, etc.) is directly accessible on structs
2. **Discoverability**: Helper methods are on the objects users already manipulate (`HardwareProfileInfo`)
3. **Flexibility**: Users can import `catalog` directly if they want raw lookups without scanning
4. **No unnecessary wrappers**: Avoid thin wrapper functions that just re-export from subpackages

### Good Design Example

```go
// ✅ Good: Value-added method on the type users already have
profile := result.HardwareProfiles[0]
fmt.Println(profile.GetGPUDescription()) // "ASUS NVIDIA GeForce RTX 5090"

// ✅ Also Good: Direct catalog import for standalone lookups
import "github.com/hekmon/aiup/msiaf/catalog"
gpuInfo := catalog.LookupGPU("10DE", "2B85")

// ❌ Bad: Unnecessary thin wrapper in root package
// (Don't create msiaf.LookupGPU() that just calls catalog.LookupGPU())
```

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

## GPU Catalog System

### Auto-Generated Catalog

The GPU catalog (`msiaf/catalog/catalog_generated.go`) is **auto-generated** and should **NEVER** be edited manually. It is generated from the [pci-ids database](https://pci-ids.ucw.cz/v2.2/pci.ids).

**To regenerate the catalog:**
```bash
cd aiup
go generate ./msiaf/...
```

This will:
1. Fetch the latest pci-ids database
2. Filter for NVIDIA, AMD, and Intel GPUs (MSI Afterburner supported vendors)
3. Clean up device names (remove chip codenames)
4. Generate `msiaf/catalog/catalog_generated.go` with ~2,200+ GPU entries

### Key Files

- `cmd/gencatalog/main.go` - Generator tool
- `msiaf/scan.go` - Profile scanning logic + `HardwareProfileInfo` methods
- `msiaf/catalog/catalog.go` - Lookup functions (hand-written)
- `msiaf/catalog/catalog_generated.go` - Auto-generated data (DO NOT EDIT)
- `msiaf/catalog/catalog_test.go` - Tests using actual pci-ids device IDs

### Important Conventions

1. **Device IDs are lowercase**: The pci-ids database uses lowercase hex (e.g., `10de_2b85`). The lookup functions automatically normalize input to lowercase.

2. **Device IDs change**: Device IDs in pci-ids may differ from other databases (TechPowerUp, etc.). Always verify against the generated catalog or re-run the generator.

3. **Name formatting**: The generator removes chip codenames (AD104, Navi 21, DG2, etc.) for cleaner user-facing names.

4. **Filtering**: Only actual GPU/display devices are included. Non-GPU devices (audio, USB, network, PCIe switches) are filtered out.

### Testing

When updating tests:
- Use device IDs from the **generated** catalog, not hardcoded values
- Run `go generate` before running tests
- Test expectations should match pci-ids data, not external databases

Example:
```go
// Good: Uses actual pci-ids device ID
{"2B85", "GeForce RTX 5090"}

// Bad: May not match pci-ids
{"2786", "GeForce RTX 4090"} // Wrong ID in pci-ids
```

### Common Pitfalls

1. **Editing generated files**: Never edit `catalog_generated.go` manually - changes will be lost on next generation.

2. **Case sensitivity**: Always use `strings.ToLower()` when looking up device IDs.

3. **Stale catalog**: If tests fail with "unknown GPU", run `go generate` to update the catalog.

4. **Vendor IDs**: 
   - NVIDIA: `10de`
   - AMD: `1002`
   - Intel: `8086`

## Code Quality Guidelines

### Use fmt.Fprintf for String Building

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

## Workflow for AI Agents

### Mandatory Pre-Commit Checklist

Before considering a task complete, **always** run:

```bash
# 1. Regenerate auto-generated files
go generate ./...

# 2. Build everything
go build ./...

# 3. Run all tests
go test ./...

# 4. (Optional) Run linter if available
go vet ./...
```

**All four steps must pass** before committing changes.

### When Working on GPU-Related Features

1. **Check if catalog needs updating**: Run `go generate ./msiaf/...` to ensure you have the latest data.

2. **Verify device IDs**: Look up actual device IDs in `catalog_generated.go`, don't assume them.

3. **Test thoroughly**: Run `go test ./msiaf/...` after any changes.

4. **Commit generated file**: Unlike typical generated files, `catalog_generated.go` should be committed to ensure consistent behavior across environments.

### When Moving or Restructuring Files

1. Update `package` declarations in moved files
2. Update `//go:generate` paths to account for new directory depth
3. Update generator (`cmd/gencatalog/main.go`) if it hardcodes package names
4. Regenerate: `go generate ./...`
5. Verify build: `go build ./...`
6. Verify tests: `go test ./...`

### Adding New Features

If extending the catalog system:

1. Modify `cmd/gencatalog/main.go` to add new logic
2. Re-run generator: `go generate ./msiaf/...`
3. Update tests in `msiaf/catalog/catalog_test.go`
4. Verify with `go build ./...` and `go test ./...`

If extending the scanning functionality:

1. Add functions/methods to `msiaf/scan.go`
2. Use catalog subpackage for GPU lookups (don't duplicate logic)
3. Add methods to `HardwareProfileInfo` for value-added helpers
4. Update tests in `msiaf/scan_test.go`
5. Verify with `go build ./...` and `go test ./...`

## API Usage Examples

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

## Troubleshooting

### "unknown GPU" errors
- Run `go generate ./msiaf/...` to update the catalog

### Build fails with "package not found"
- Check that all moved files have correct `package` declarations
- Verify import paths are updated

### go generate fails
- Check `//go:generate` path is correct for the file's directory depth
- Verify `cmd/gencatalog/main.go` exists and is accessible

### Tests fail after moving files
- Ensure generator outputs correct package name
- Re-run `go generate ./...` to regenerate files with correct package

### Import errors with catalog subpackage
- Import as: `github.com/hekmon/aiup/msiaf/catalog`
- Not: `github.com/hekmon/aiup/msiaf` (root package doesn't re-export catalog)