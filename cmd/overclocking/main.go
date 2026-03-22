// Command overclocking is an example program demonstrating the overclocking package GPU discovery.
// It scans MSI Afterburner profiles and correlates them with NvAPI-detected GPUs,
// then outputs the results as pretty-printed JSON.
//
// Usage:
//
//	overclocking.exe [flags]
//
// Flags:
//
//	-profiles string
//		Path to MSI Afterburner Profiles directory (default: C:\Program Files (x86)\MSI Afterburner\Profiles)
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
	flag.Parse()

	// Call the discovery function
	result, err := overclocking.ScanGPUs(*profilesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Pretty-print the JSON result
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding JSON: %v\n", err)
		os.Exit(1)
	}
}
