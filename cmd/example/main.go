// Command example demonstrates how to use the msiaf package to scan MSI Afterburner profiles
// and parse the global configuration file.
//
// Usage:
//
//	cd aiup
//	go run cmd/example/main.go
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

	// Parse and display global config file info
	if result.GlobalConfigPath != "" {
		fmt.Println("Global Configuration File:")
		fmt.Printf("  Path: %s\n", result.GlobalConfigPath)

		config, err := msiaf.ParseGlobalConfig(result.GlobalConfigPath)
		if err != nil {
			fmt.Printf("  Error parsing config: %v\n", err)
		} else {
			fmt.Println()
			fmt.Println("  Key Settings:")
			fmt.Printf("    Language:              %s\n", config.Settings.Language)
			fmt.Printf("    Fahrenheit:            %v\n", config.Settings.Fahrenheit)
			fmt.Printf("    HwPollPeriod:          %v\n", config.Settings.HwPollPeriod)
			fmt.Printf("    StartWithWindows:      %v\n", config.Settings.StartMinimized)
			fmt.Printf("    UnlockVoltageControl:  %v\n", config.Settings.UnlockVoltageControl)
			fmt.Printf("    LowLevelInterface:     %v\n", config.Settings.LowLevelInterface)
			fmt.Printf("    SwAutoFanControl:      %v\n", config.Settings.SwAutoFanControl)
			fmt.Printf("    CurrentGpu:            %d\n", config.Settings.CurrentGpu)
			fmt.Printf("    Sync (GPU linking):    %v\n", config.Settings.Sync)
			fmt.Printf("    LinkThermal:           %v\n", config.Settings.LinkThermal)
			fmt.Printf("    FanSync:               %v\n", config.Settings.FanSync)
		}
		fmt.Println()
	} else {
		fmt.Println("Global Configuration File: Not found")
		fmt.Println()
	}

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
