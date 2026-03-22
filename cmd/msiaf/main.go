// Command example demonstrates how to use the msiaf package to scan MSI Afterburner profiles,
// parse the global configuration file, load hardware-specific profile settings, and parse VF curves.
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

			// Display fan control curve if available
			curve := config.Settings.GetFanControlCurve()
			if curve != nil && len(curve.Points) > 0 {
				fmt.Println()
				fmt.Println("  Software Auto Fan Control Curve:")
				fmt.Printf("    Version: %s, Points: %d\n", curve.VersionString(), len(curve.Points))
				for i, point := range curve.Points {
					fmt.Printf("    Point %d: %5.1f°C → %5.1f%% fan\n", i+1, point.Temperature, point.FanSpeed)
				}
			}
		}
		fmt.Println()
	} else {
		fmt.Println("Global Configuration File: Not found")
		fmt.Println()
	}

	// Display hardware profiles
	fmt.Printf("Hardware Profiles (%d found):\n", len(result.HardwareProfiles))
	fmt.Println()

	for i, profileInfo := range result.HardwareProfiles {
		fmt.Printf("[%d] %s\n", i+1, filepath.Base(profileInfo.FilePath))

		// Display human-readable GPU summary using value-added method
		fmt.Printf("    GPU: %s\n", profileInfo.GetGPUDescription())

		// Display resolved device and manufacturer info using profile methods
		gpuInfo := profileInfo.GetGPUInfo()
		manufacturer := profileInfo.GetManufacturer()
		fmt.Printf("    Vendor ID:  %s (%s)\n", profileInfo.VendorID, gpuInfo.VendorName)
		fmt.Printf("    Device ID:  %s (%s)\n", profileInfo.DeviceID, gpuInfo.GPUName)
		fmt.Printf("    Subsystem:  %s (%s)\n", profileInfo.SubsystemID, manufacturer)

		fmt.Printf("    Revision:   %s\n", profileInfo.Revision)
		fmt.Printf("    Location:   Bus %d, Device %d, Function %d\n",
			profileInfo.BusNumber, profileInfo.DeviceNumber, profileInfo.FunctionNumber)

		// Load and display hardware profile settings
		hwProfile, err := profileInfo.LoadProfile()
		if err != nil {
			fmt.Printf("    Error loading profile: %v\n", err)
		} else {
			fmt.Println()
			fmt.Println("    Hardware Profile Settings:")

			// Display Startup section (currently active settings)
			startup := hwProfile.GetCurrentSettings()
			fmt.Println("    Startup (Active Settings):")
			if startup.Format != nil {
				fmt.Printf("      Format:        %d\n", *startup.Format)
			}
			if startup.PowerLimit != nil {
				fmt.Printf("      PowerLimit:    %d%%\n", *startup.PowerLimit)
			}
			if startup.CoreClkBoost != nil {
				fmt.Printf("      CoreClkBoost:  +%d MHz\n", startup.GetCoreClkBoostMHz())
			}
			if startup.MemClkBoost != nil {
				fmt.Printf("      MemClkBoost:   +%d MHz\n", startup.GetMemClkBoostMHz())
			}
			if startup.FanMode != nil {
				mode := "Auto"
				if *startup.FanMode == 1 {
					mode = "Manual"
				}
				fmt.Printf("      FanMode:       %s\n", mode)
			}
			if startup.FanSpeed != nil {
				fmt.Printf("      FanSpeed:      %d%%\n", *startup.FanSpeed)
			}
			if startup.VFCurve != nil {
				fmt.Printf("      VFCurve:       %d bytes (binary format)\n", len(startup.VFCurve))
			}

			// Display Defaults section
			defaults := hwProfile.GetDefaults()
			if defaults.HasSettings() {
				fmt.Println()
				fmt.Println("    Defaults (Factory Settings):")
				if defaults.PowerLimit != nil {
					fmt.Printf("      PowerLimit:    %d%%\n", *defaults.PowerLimit)
				}
				if defaults.CoreClkBoost != nil {
					fmt.Printf("      CoreClkBoost:  +%d MHz\n", defaults.GetCoreClkBoostMHz())
				}
				if defaults.MemClkBoost != nil {
					fmt.Printf("      MemClkBoost:   +%d MHz\n", defaults.GetMemClkBoostMHz())
				}
			}

			// Display user-defined profile slots (Profile1-5)
			fmt.Println()
			fmt.Println("    User Profile Slots:")
			for slotNum := 1; slotNum <= 5; slotNum++ {
				slot := hwProfile.GetProfile(slotNum)
				if slot != nil && !slot.IsEmpty {
					fmt.Printf("      Profile%d:\n", slotNum)
					if slot.PowerLimit != nil {
						fmt.Printf("        PowerLimit:  %d%%\n", *slot.PowerLimit)
					}
					if slot.CoreClkBoost != nil {
						fmt.Printf("        CoreClk:     +%d MHz\n", slot.GetCoreClkBoostMHz())
					}
					if slot.MemClkBoost != nil {
						fmt.Printf("        MemClk:      +%d MHz\n", slot.GetMemClkBoostMHz())
					}
					if slot.FanSpeed != nil {
						fmt.Printf("        FanSpeed:    %d%%\n", *slot.FanSpeed)
					}
				} else if slot != nil && slot.IsEmpty {
					fmt.Printf("      Profile%d: (empty)\n", slotNum)
				}
			}

			// Display PreSuspendedMode if it has settings
			if hwProfile.PreSuspendedMode.HasSettings() {
				fmt.Println()
				fmt.Println("    PreSuspendedMode:")
				if hwProfile.PreSuspendedMode.Format != nil {
					fmt.Printf("      Format: %d\n", *hwProfile.PreSuspendedMode.Format)
				}
				// Show which fields are populated (non-nil)
				hasValues := false
				if hwProfile.PreSuspendedMode.PowerLimit != nil {
					fmt.Printf("      PowerLimit: %d%%\n", *hwProfile.PreSuspendedMode.PowerLimit)
					hasValues = true
				}
				if !hasValues {
					fmt.Println("      (sparse - only Format field set)")
				}
			}

			// Display VF Curve if available
			if len(startup.VFCurve) > 0 {
				fmt.Println()
				fmt.Println("    VF Curve (Voltage-Frequency) - [Startup] section:")
				fmt.Printf("      Raw data: %d bytes\n", len(startup.VFCurve))

				// Parse the VF curve from hex
				hexData := fmt.Sprintf("%x", startup.VFCurve)
				curve, err := msiaf.UnmarshalVFControlCurve(hexData)
				if err != nil {
					fmt.Printf("      Error parsing: %v\n", err)
				} else {
					fmt.Printf("      Version: %s\n", curve.VersionString())
					fmt.Printf("      Points: %d (%d active, %d inactive)\n",
						len(curve.Points), len(curve.GetActivePoints()), len(curve.GetInactivePoints()))

					// Show offset range
					if len(curve.Points) > 0 {
						minOffset := curve.GetMinOffset()
						maxOffset := curve.GetMaxOffset()
						fmt.Printf("      Offset range: %+.0f ... %+.0f MHz\n", minOffset, maxOffset)
					}

					// Display sample points in the 800-1000 mV range
					fmt.Println()
					fmt.Println("      Sample points (voltage → offset | base freq):")
					// Show all voltage points from 800-1000 mV at 25 mV intervals
					sampleVoltages := []float32{800, 825, 850, 875, 900, 925, 950, 975, 1000}
					for _, voltage := range sampleVoltages {
						point := curve.GetPointByVoltage(voltage)
						if point != nil {
							// Display actual voltage of the point found
							fmt.Printf("        %4.0f mV → %+.0f MHz | base: %.0f MHz\n",
								point.VoltageMV, point.OffsetMHz, point.BaseFreqMHz)
						}
					}
					fmt.Println("      Note: Actual frequency = hardware boost + offset.")
					fmt.Println("      Hardware boost is GPU-specific (not stored). Base freq is the reference for offsets.")
				}
			}

			// Display VF Curves from user-defined profile slots (Profile1-5)
			for slotNum := 1; slotNum <= 5; slotNum++ {
				slot := hwProfile.GetProfile(slotNum)
				if slot != nil && !slot.IsEmpty && len(slot.VFCurve) > 0 {
					fmt.Println()
					fmt.Printf("    VF Curve - [Profile%d] section:\n", slotNum)
					fmt.Printf("      Raw data: %d bytes\n", len(slot.VFCurve))

					// Parse the VF curve from hex
					hexData := fmt.Sprintf("%x", slot.VFCurve)
					curve, err := msiaf.UnmarshalVFControlCurve(hexData)
					if err != nil {
						fmt.Printf("      Error parsing: %v\n", err)
					} else {
						fmt.Printf("      Version: %s\n", curve.VersionString())
						fmt.Printf("      Points: %d (%d active, %d inactive)\n",
							len(curve.Points), len(curve.GetActivePoints()), len(curve.GetInactivePoints()))

						// Show offset range
						if len(curve.Points) > 0 {
							minOffset := curve.GetMinOffset()
							maxOffset := curve.GetMaxOffset()
							fmt.Printf("      Offset range: %+.0f ... %+.0f MHz\n", minOffset, maxOffset)
						}

						// Show sample points in the 800-1000 mV range
						fmt.Println()
						fmt.Println("      Sample points (voltage → offset | base freq):")
						sampleVoltages := []float32{800, 825, 850, 875, 900, 925, 950, 975, 1000}
						for _, voltage := range sampleVoltages {
							point := curve.GetPointByVoltage(voltage)
							if point != nil {
								fmt.Printf("        %4.0f mV → %+.0f MHz | base: %.0f MHz\n",
									point.VoltageMV, point.OffsetMHz, point.BaseFreqMHz)
							}
						}
					}
				}
			}
		}

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
