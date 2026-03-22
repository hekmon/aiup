// nvvf - NVIDIA V-F Curve Command-Line Tool
//
// This is the CLI tool for reading NVIDIA GPU voltage-frequency (V-F) curves.
// For the Go package API, see github.com/hekmon/aiup/nvvf.
//
// # PLATFORM SUPPORT
//
// Supports both Windows (nvapi64.dll) and Linux (libnvidia-api.so.1).
// Both platforms use identical NvAPI function IDs and struct layouts.
//
// WSL Users: Use the Windows build (nvvf.exe) directly from WSL. The Linux
// build cannot be used on WSL because libnvidia-api.so.1 is not available.
//
// USAGE
//
//	nvvf                          # Read all GPUs, human-readable output
//	nvvf -gpu 0                   # Read specific GPU
//	nvvf -json                    # Output as JSON
//	nvvf -v                       # Verbose (show diagnostic info)
//	nvvf -list                    # List available GPUs
//	nvvf -diag                    # Detailed diagnostics (base + offsets)
//	nvvf -debug                   # Raw struct debug info
//	nvvf -h                       # Show help
//
// EXAMPLES
//
//	# Quick check of GPU 0
//	nvvf -gpu 0
//
//	# Full diagnostic with verbose output
//	nvvf -v
//
//	# Export for analysis
//	nvvf -json > vfcurve.json
//
//	# List all available GPUs
//	nvvf -list
//
// FLAGS
//
//	-gpu int
//	    GPU index to query (default -1 = all GPUs)
//	-json
//	    Output as JSON instead of human-readable format
//	-v    Verbose output (show diagnostic info)
//	-list
//	    List available NVIDIA GPUs and exit
//	-diag
//	    Show detailed diagnostics (base curve + offsets separately)
//	-debug
//	    Show raw NvAPI struct bytes (for troubleshooting)
//	-h    Show help message
//
// # OUTPUT
//
// Human-readable table or JSON with voltage/frequency data from NvAPI.
//
//	Voltage (mV)   : Voltage step
//	Base (MHz)     : Hardware base frequency (from driver)
//	Offset (MHz)   : User frequency offset (via NvAPI SetControl)
//	Effective (MHz): Base + Offset (actual applied frequency)
//
// # ATTRIBUTION
//
// The Blackwell (RTX 50xx) NvAPI V-F curve struct sizes and function
// definitions were discovered through community reverse-engineering:
//
//   - LACT Project: https://github.com/ilya-zlobintsev/LACT
//   - Issue #936: https://github.com/ilya-zlobintsev/LACT/issues/936
//   - Researcher: Loong0x00 (GitHub)
//
// # RELATED
//
// For MSI Afterburner .cfg file parsing, see the msiaf package.
// For Go package API documentation, see https://github.com/hekmon/aiup/nvvf
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/hekmon/aiup/nvvf"
)

func main() {
	// Command-line flags
	gpuIndex := flag.Int("gpu", -1, "GPU index to query (default: -1 = all GPUs)")
	jsonOutput := flag.Bool("json", false, "Output as JSON instead of human-readable format")
	verbose := flag.Bool("v", false, "Verbose output (show diagnostic info)")
	listGPUs := flag.Bool("list", false, "List available NVIDIA GPUs and exit")
	showHelp := flag.Bool("h", false, "Show help message")
	showDiag := flag.Bool("diag", false, "Show detailed diagnostics (base curve + offsets separately)")
	showDebug := flag.Bool("debug", false, "Show raw NvAPI struct bytes for troubleshooting")
	flag.Parse()

	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	// List GPUs mode
	if *listGPUs {
		if err := listAvailableGPUs(*verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *showDiag {
		if err := showDiagnostics(*gpuIndex); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *showDebug {
		if err := showDebugRaw(*gpuIndex); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	// Read V-F curves from NvAPI
	if *verbose {
		fmt.Fprintf(os.Stderr, "=== Querying NVIDIA GPUs via NvAPI ===\n\n")
	}

	nvapiPoints, gpuCount, err := readGPUsViaNvAPI(*gpuIndex, *verbose)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NvAPI error: %v\n", err)
		os.Exit(1)
	}

	if gpuCount == 0 {
		fmt.Fprintln(os.Stderr, "No NVIDIA GPUs detected or NvAPI unavailable")
		os.Exit(1)
	}

	// Output results
	if *jsonOutput {
		outputJSON(nvapiPoints)
	} else {
		outputHuman(nvapiPoints, *verbose)
	}

	// Show summary in verbose mode
	if *verbose && !*jsonOutput {
		showSummary(nvapiPoints)
	}
}

// printUsage displays the help message
func printUsage() {
	fmt.Println(`NVIDIA V-F Curve Tool - Test nvvf package functionality

Usage:
  nvvf [flags]

Flags:
  -gpu int
        GPU index to query (default -1 = all GPUs)
  -json
        Output as JSON instead of human-readable format
  -v    Verbose output (show diagnostic info)
  -list
        List available NVIDIA GPUs and exit
  -diag
        Show detailed diagnostics (base curve + offsets separately)
  -debug
        Show raw NvAPI struct bytes (for troubleshooting)
  -h    Show this help message

Examples:
  nvvf                    # Read all GPUs
  nvvf -gpu 0             # Read GPU 0 only
  nvvf -v                 # Verbose mode
  nvvf -list              # List available GPUs
  nvvf -json > out.json   # Export as JSON

Output:
  Human-readable table or JSON with voltage/frequency data from NvAPI.

NvAPI Data:
  Voltage (mV)   : Voltage step
  Base (MHz)     : Hardware base frequency (from driver)
  Offset (MHz)   : User frequency offset
  Effective (MHz): Base + Offset (exact frequency)

Credits & Attribution:
  The Blackwell (RTX 50xx) NvAPI V-F curve struct sizes and function definitions
  were discovered through community reverse-engineering efforts:

  • LACT Project: https://github.com/ilya-zlobintsev/LACT
  • Issue #936: https://github.com/ilya-zlobintsev/LACT/issues/936
  • Researcher: Loong0x00 (GitHub)

  Their work on reverse-engineering ASUS GPU Tweak III and documenting the NvAPI
  V-F curve functions made this implementation possible. Special thanks to the
  open-source GPU tools community for sharing knowledge.

For MSI Afterburner .cfg parsing, see the msiaf package.
For Go package API documentation, see: https://github.com/hekmon/aiup/nvvf
For package README with technical details, see: nvvf/README.md`)
}

// listAvailableGPUs enumerates and lists all NVIDIA GPUs
func listAvailableGPUs(verbose bool) error {
	// Try to read from each GPU index
	found := 0
	for i := 0; i < 4; i++ {
		points, err := nvvf.ReadNvAPIVF(i)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "GPU %d: %v\n", i, err)
			}
			continue
		}
		found++
		pointCount := len(points)
		if pointCount > 0 {
			// Get voltage/frequency range
			minVolt, maxVolt := points[0].VoltageMV, points[0].VoltageMV
			minFreq, maxFreq := points[0].BaseFreqMHz, points[0].BaseFreqMHz
			for _, p := range points {
				if p.VoltageMV < minVolt && p.VoltageMV > 0 {
					minVolt = p.VoltageMV
				}
				if p.VoltageMV > maxVolt {
					maxVolt = p.VoltageMV
				}
				if p.BaseFreqMHz < minFreq && p.BaseFreqMHz > 0 {
					minFreq = p.BaseFreqMHz
				}
				if p.BaseFreqMHz > maxFreq {
					maxFreq = p.BaseFreqMHz
				}
			}
			fmt.Printf("GPU %d: %d points, %.0f-%.0f mV, %.0f-%.0f MHz\n",
				i, pointCount, minVolt, maxVolt, minFreq, maxFreq)
		} else {
			fmt.Printf("GPU %d: No V/F points returned\n", i)
		}
	}

	if found == 0 {
		return fmt.Errorf("no NVIDIA GPUs detected (tried indices 0-3)")
	}

	fmt.Printf("\nTotal: %d NVIDIA GPU(s) found\n", found)
	return nil
}

// readGPUsViaNvAPI queries GPUs and returns points with GPU count
func readGPUsViaNvAPI(gpuIndex int, verbose bool) ([]nvvf.VFPoint, int, error) {
	if gpuIndex >= 0 {
		// Specific GPU requested
		if verbose {
			fmt.Fprintf(os.Stderr, "Querying GPU %d via NvAPI... ", gpuIndex)
		}
		points, err := nvvf.ReadNvAPIVF(gpuIndex)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "failed: %v\n", err)
			}
			return nil, 0, fmt.Errorf("GPU %d: %w", gpuIndex, err)
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "found %d points\n", len(points))
		}
		// Tag points with GPU index
		for i := range points {
			points[i].Index = gpuIndex*1000 + points[i].Index
		}
		return points, 1, nil
	}

	// Query all GPUs - try indices 0-3 (most common configurations)
	var allPoints []nvvf.VFPoint
	foundGPUs := 0
	for i := 0; i < 4; i++ {
		if verbose {
			fmt.Fprintf(os.Stderr, "Querying GPU %d via NvAPI... ", i)
		}
		points, err := nvvf.ReadNvAPIVF(i)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "not found (%v)\n", err)
			}
			continue
		}
		if verbose {
			fmt.Fprintf(os.Stderr, "found %d points\n", len(points))
		}
		foundGPUs++
		// Tag points with GPU index
		for j := range points {
			points[j].Index = i*1000 + points[j].Index
		}
		allPoints = append(allPoints, points...)
	}

	if foundGPUs == 0 {
		return nil, 0, fmt.Errorf("no NVIDIA GPUs detected (tried indices 0-3)")
	}

	return allPoints, foundGPUs, nil
}

// outputHuman prints results in a human-readable table format
func outputHuman(nvapiPoints []nvvf.VFPoint, verbose bool) {
	fmt.Println("=== NVIDIA V-F Curve (from NvAPI) ===")
	fmt.Println()

	currentGPU := -1
	for _, p := range nvapiPoints {
		gpuIdx := p.Index / 1000

		if gpuIdx != currentGPU {
			if currentGPU >= 0 {
				fmt.Println()
			}
			fmt.Printf("GPU %d:\n", gpuIdx)
			fmt.Println("  Voltage (mV) | Base (MHz) | Offset (MHz) | Effective (MHz)")
			fmt.Println("  -------------|------------|--------------|----------------")
			currentGPU = gpuIdx
		}

		fmt.Printf("  %12.0f | %10.0f | %12.0f | %15.0f\n",
			p.VoltageMV, p.BaseFreqMHz, p.OffsetMHz, p.EffectiveMHz)
	}
}

// outputJSON prints results as formatted JSON
func outputJSON(nvapiPoints []nvvf.VFPoint) {
	result := map[string]interface{}{
		"nvapi_points": nvapiPoints,
		"attribution": map[string]string{
			"project":      "LACT (Linux AMD/NVIDIA Control Tool)",
			"github":       "https://github.com/ilya-zlobintsev/LACT",
			"issue":        "https://github.com/ilya-zlobintsev/LACT/issues/936",
			"researcher":   "Loong0x00",
			"contribution": "Blackwell V-F curve struct sizes and NvAPI function discovery",
		},
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "JSON encoding error: %v\n", err)
		os.Exit(1)
	}
}

// showSummary displays a summary of the V-F curve analysis
func showSummary(nvapiPoints []nvvf.VFPoint) {
	if len(nvapiPoints) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Println()

	// Find min/max from NvAPI data
	minVolt, maxVolt := nvapiPoints[0].VoltageMV, nvapiPoints[0].VoltageMV
	minFreq, maxFreq := nvapiPoints[0].BaseFreqMHz, nvapiPoints[0].BaseFreqMHz
	activePoints := 0

	for _, p := range nvapiPoints {
		if p.BaseFreqMHz > 0 {
			activePoints++
		}
		if p.VoltageMV > 0 && p.VoltageMV < minVolt {
			minVolt = p.VoltageMV
		}
		if p.VoltageMV > maxVolt {
			maxVolt = p.VoltageMV
		}
		if p.BaseFreqMHz > 0 && p.BaseFreqMHz < minFreq {
			minFreq = p.BaseFreqMHz
		}
		if p.BaseFreqMHz > maxFreq {
			maxFreq = p.BaseFreqMHz
		}
	}

	fmt.Printf("Hardware Base Frequency Range: %.0f - %.0f MHz\n", minFreq, maxFreq)
	fmt.Printf("Voltage Range: %.0f - %.0f mV\n", minVolt, maxVolt)
	fmt.Printf("Active V/F Points: %d / 128\n", activePoints)
}

// showDiagnostics displays base curve and offsets separately
func showDiagnostics(gpuIndex int) error {
	if gpuIndex < 0 {
		gpuIndex = 0
	}

	fmt.Println("=== NVIDIA V-F Curve Diagnostics ===")
	fmt.Println()
	fmt.Println("This shows the TWO NvAPI endpoints separately:")
	fmt.Println("  1. GetStatus (0x21537AD4)  → Hardware base curve")
	fmt.Println("  2. GetControl (0x23F1B133) → User offsets")
	fmt.Println()

	points, err := nvvf.ReadNvAPIVF(gpuIndex)
	if err != nil {
		return err
	}

	// Show base curve
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 1. BASE CURVE (from GetStatus - hardware pstate table)     │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Printf("%4s | %8s | %10s\n", "mV", "Base MHz", "State")
	fmt.Println("-----|----------|------------")
	for i, p := range points {
		state := ""
		if p.BaseFreqMHz <= 300 {
			state = "idle"
		} else if p.BaseFreqMHz >= 2800 {
			state = "boost"
		}
		fmt.Printf("%4d | %8d | %s\n", int(p.VoltageMV), int(p.BaseFreqMHz), state)
		if i == 50 {
			fmt.Println("  ... (showing key points only)")
		}
		if i > 100 {
			break
		}
	}
	fmt.Println()

	// Show offsets
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 2. OFFSETS (from GetControl - user applied offsets)        │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Printf("%4s | %10s | %12s\n", "mV", "Offset MHz", "Status")
	fmt.Println("-----|------------|--------------")
	hasOffsets := false
	for i, p := range points {
		status := "no change"
		if p.OffsetMHz != 0 {
			status = fmt.Sprintf("%+.0f MHz", p.OffsetMHz)
			hasOffsets = true
		}
		if i <= 50 || i > 100 || p.OffsetMHz != 0 {
			fmt.Printf("%4d | %10.0f | %s\n", int(p.VoltageMV), p.OffsetMHz, status)
		}
		if i == 50 {
			fmt.Println("  ... (showing key points only)")
		}
		if i > 100 {
			break
		}
	}
	fmt.Println()
	if !hasOffsets {
		fmt.Println("⚠️  NO OFFSETS DETECTED - Profile may be applied via .cfg file")
		fmt.Println("    (MSI Afterburner hardware profiles ≠ NvAPI SetControl)")
	}
	fmt.Println()

	// Show combined
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 3. EFFECTIVE (Base + Offset = actual applied frequency)    │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Printf("%4s | %10s | %10s | %10s\n", "mV", "Base", "Offset", "Effective")
	fmt.Println("-----|------------|------------|------------")
	for i, p := range points {
		if i <= 50 || i > 100 {
			fmt.Printf("%4d | %10d | %10.0f | %10d\n", int(p.VoltageMV), int(p.BaseFreqMHz), p.OffsetMHz, int(p.EffectiveMHz))
		}
		if i == 50 {
			fmt.Println("  ... (showing key points only)")
		}
		if i > 100 {
			break
		}
	}
	fmt.Println()

	// Summary
	fmt.Println("=== Key Takeaways ===")
	fmt.Println()
	maxBase := 0.0
	maxOffset := 0.0
	for _, p := range points {
		if p.BaseFreqMHz > maxBase {
			maxBase = p.BaseFreqMHz
		}
		if p.OffsetMHz > maxOffset {
			maxOffset = p.OffsetMHz
		}
		if -p.OffsetMHz > maxOffset {
			maxOffset = -p.OffsetMHz
		}
	}
	fmt.Printf("Max Base Frequency:    %.0f MHz (from driver pstate)\n", maxBase)
	fmt.Printf("Max Offset Applied:    %.0f MHz (from NvAPI GetControl)\n", maxOffset)
	fmt.Printf("Max Effective:         %.0f MHz (base + offset)\n", maxBase+maxOffset)
	fmt.Println()
	if maxOffset == 0 {
		fmt.Println("💡 Your profile is likely applied via MSI Afterburner .cfg files,")
		fmt.Println("   not through NvAPI's live SetControl API. Check LocalProfiles/ directory.")
	}

	return nil
}

// showDebugRaw displays raw bytes from NvAPI structs
func showDebugRaw(gpuIndex int) error {
	if gpuIndex < 0 {
		gpuIndex = 0
	}

	fmt.Println("=== NvAPI Raw Struct Debug ===")
	fmt.Println()
	fmt.Println("WARNING: This shows internal NvAPI struct layout for debugging.")
	fmt.Println()

	// We can't access raw structs from the public API, so this is a placeholder
	// that explains what WOULD be shown if we exposed raw data
	fmt.Println("To debug control struct parsing, check:")
	fmt.Println("  1. nvVfPointsCtrlBlackwell struct size: 9248 bytes (0x2420)")
	fmt.Println("  2. Entry stride: 72 bytes (0x48)")
	fmt.Println("  3. Offset field: int32 at byte 0 of each entry")
	fmt.Println("  4. Expected: non-zero values if offsets applied")
	fmt.Println()
	fmt.Println("Current implementation returns parsed VFPoint data only.")
	fmt.Println("For raw byte access, would need to modify nvvf package to expose internals.")
	fmt.Println()

	// Show what we CAN see
	points, err := nvvf.ReadNvAPIVF(gpuIndex)
	if err != nil {
		return err
	}

	fmt.Println("Parsed control data (from GetControl):")
	fmt.Printf("  Total points: %d\n", len(points))

	maxOffset := 0.0
	nonZeroCount := 0
	for _, p := range points {
		if p.OffsetMHz != 0 {
			nonZeroCount++
			if p.OffsetMHz > maxOffset {
				maxOffset = p.OffsetMHz
			}
			if -p.OffsetMHz > maxOffset {
				maxOffset = -p.OffsetMHz
			}
		}
	}

	fmt.Printf("  Points with non-zero offset: %d / %d\n", nonZeroCount, len(points))
	fmt.Printf("  Max absolute offset: %.0f MHz\n", maxOffset)
	fmt.Println()

	if nonZeroCount == 0 {
		fmt.Println("⚠️  ALL OFFSETS ARE ZERO")
		fmt.Println()
		fmt.Println("Possible causes:")
		fmt.Println("  1. No profile applied via NvAPI SetControl")
		fmt.Println("  2. MSI Afterburner uses NVML/driver-level mechanism instead")
		fmt.Println("  3. Profile applied via .cfg file (hardware profile)")
		fmt.Println("  4. Struct parsing bug (check Blackwell entry stride = 72 bytes)")
	}

	return nil
}
