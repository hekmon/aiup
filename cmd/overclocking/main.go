// Command overclocking is an example program demonstrating the overclocking package.
// It supports two modes:
//
//  1. GPU Discovery: Scans MSI Afterburner profiles and correlates them with NvAPI-detected GPUs
//  2. Current Curve: Reads the current V-F curve from a specific GPU and compares it against Startup
//
// Both modes output results as pretty-printed JSON.
//
// Usage:
//
//	overclocking.exe [flags]
//
// Examples:
//
//	# GPU Discovery mode (default)
//	overclocking.exe
//
//	# Current Curve mode (get curve for GPU 0)
//	overclocking.exe -gpu 0
//
// Flags:
//
//	-profiles string
//		Path to MSI Afterburner Profiles directory (default: C:\Program Files (x86)\MSI Afterburner\Profiles)
//	-gpu int
//		Get current V-F curve for specified GPU index (0, 1, 2, ...)
//	-h, --help
//		Show help message
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/hekmon/aiup/overclocking"
)

func main() {
	// Default MSI Afterburner Profiles directory (Windows standard installation path)
	defaultProfilesDir := `C:\Program Files (x86)\MSI Afterburner\Profiles`

	profilesDir := flag.String("profiles", defaultProfilesDir, "Path to MSI Afterburner Profiles directory")
	gpuIndex := flag.Int("gpu", -1, "Get current V-F curve for specified GPU index (0, 1, 2, ...)")
	flag.Parse()

	// Pretty-print helper
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	// Mode selection: GPU discovery (default) or current curve
	if *gpuIndex < 0 {
		// GPU Discovery mode
		result, err := overclocking.ScanGPUs(*profilesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Current Curve mode
		// Step 1: Discover GPUs to get profile path
		discovery, err := overclocking.ScanGPUs(*profilesDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// Step 2: Find the GPU by index
		var profilePath string
		for _, gpu := range discovery.GPUs {
			if gpu.Index == *gpuIndex {
				profilePath = gpu.ProfilePath
				break
			}
		}
		if profilePath == "" {
			fmt.Fprintf(os.Stderr, "Error: GPU index %d not found\n", *gpuIndex)
			os.Exit(1)
		}

		// Step 3: Get current V-F curve using profile path
		result, err := overclocking.GetCurrentCurve(*gpuIndex, profilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := encoder.Encode(result); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
			os.Exit(1)
		}
	}
}
