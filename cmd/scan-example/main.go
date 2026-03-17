// Command scan-example demonstrates how to use the msiaf package to scan MSI Afterburner profiles.
//
// Usage:
//
//	cd aiup
//	go run cmd/scan-example/main.go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hekmon/aiup/msiaf"
)

func main() {
	// Define the LocalProfiles directory path
	profilesDir := "LocalProfiles"

	// Check if directory exists
	if _, err := os.Stat(profilesDir); os.IsNotExist(err) {
		fmt.Printf("Error: LocalProfiles directory does not exist at %s\n", profilesDir)
		fmt.Println("Please ensure MSI Afterburner profiles are exported to this directory.")
		os.Exit(1)
	}

	fmt.Println("=== MSI Afterburner Profile Scanner ===")
	fmt.Println()

	// Scan the profiles directory
	result, err := msiaf.Scan(profilesDir)
	if err != nil {
		fmt.Printf("Error scanning profiles: %v\n", err)
		os.Exit(1)
	}

	// Display global config file info
	fmt.Println("Global Configuration File:")
	if result.GlobalConfigPath != "" {
		fmt.Printf("  Path: %s\n", result.GlobalConfigPath)
	} else {
		fmt.Println("  Not found")
	}
	fmt.Println()

	// Display hardware profiles
	fmt.Printf("Hardware Profiles (%d found):\n", len(result.HardwareProfiles))
	fmt.Println()

	for i, profile := range result.HardwareProfiles {
		fmt.Printf("[%d] %s\n", i+1, filepath.Base(profile.FilePath))

		// Display human-readable GPU summary using value-added method
		fmt.Printf("    GPU: %s\n", profile.GetGPUDescription())

		// Display resolved device and manufacturer info using profile methods
		gpuInfo := profile.GetGPUInfo()
		manufacturer := profile.GetManufacturer()
		fmt.Printf("    Vendor ID:  %s (%s)\n", profile.VendorID, gpuInfo.VendorName)
		fmt.Printf("    Device ID:  %s (%s)\n", profile.DeviceID, gpuInfo.GPUName)
		fmt.Printf("    Subsystem:  %s (%s)\n", profile.SubsystemID, manufacturer)

		fmt.Printf("    Revision:   %s\n", profile.Revision)
		fmt.Printf("    Location:   Bus %d, Device %d, Function %d\n",
			profile.BusNumber, profile.DeviceNumber, profile.FunctionNumber)

		fmt.Println()
	}

	// Display any warnings/errors encountered
	if len(result.Errors) > 0 {
		fmt.Println("Warnings:")
		for _, errMsg := range result.Errors {
			fmt.Printf("  - %s\n", errMsg)
		}
		fmt.Println()
	}

	fmt.Println("Scan complete.")
}
