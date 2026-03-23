//go:build windows

// Command active demonstrates how to match MSI Afterburner V-F curve profiles
// against live NVIDIA GPU V-F curve data using both msiaf and nvvf packages.
//
// This example shows the end-to-end workflow:
//  1. Scan MSI Afterburner profiles (msiaf package)
//  2. Read live V-F curve from NVIDIA GPU (nvvf package)
//  3. Read GPU marketing name for easy identification (nvvf.GetGPUName)
//  4. Read clock domain ranges for OC validation (nvvf.ReadNvAPIClkDomains)
//  5. Match profile slots against live data (msiaf.MatchProfileAgainstLive)
//  6. Identify which profile slot is currently active
//
// Usage:
//
//	cd aiup
//	go run cmd/active/main.go
//
// Flags:
//
//	-dir    Path to MSI Afterburner Profiles directory (default: "C:\Program Files (x86)\MSI Afterburner\Profiles")
//
// Requirements:
//   - Windows with NVIDIA GPU and NvAPI available (nvapi64.dll from NVIDIA drivers)
//   - MSI Afterburner installed (default profiles location: C:\Program Files (x86)\MSI Afterburner\Profiles)
//
// Note: This example requires actual NVIDIA hardware to demonstrate the matching
// functionality. Without a GPU, it will display an informative message and exit.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hekmon/aiup/overclocking/msiaf"
	"github.com/hekmon/aiup/overclocking/nvvf"
)

func main() {
	// Parse command-line flags
	profilesDir := flag.String("dir", "C:\\Program Files (x86)\\MSI Afterburner\\Profiles", "Path to MSI Afterburner Profiles directory")
	flag.Parse()

	// Check if directory exists
	if _, err := os.Stat(*profilesDir); os.IsNotExist(err) {
		fmt.Printf("Error: LocalProfiles directory does not exist at %s\n", *profilesDir)
		fmt.Println("Please ensure MSI Afterburner profiles are exported to this directory.")
		fmt.Println("You can export profiles using MSI Afterburner's profile export feature.")
		os.Exit(1)
	}

	fmt.Println("=== MSI Afterburner Profile Matcher ===")
	fmt.Println()

	// Step 1: Scan profiles directory
	fmt.Println("Step 1: Scanning MSI Afterburner profiles...")
	result, err := msiaf.Scan(*profilesDir)
	if err != nil {
		fmt.Printf("Error scanning profiles: %v\n", err)
		os.Exit(1)
	}

	if len(result.HardwareProfiles) == 0 {
		fmt.Println("No hardware profiles found. Please export profiles from MSI Afterburner.")
		os.Exit(1)
	}

	fmt.Printf("Found %d hardware profile(s)\n", len(result.HardwareProfiles))
	fmt.Println()

	// Step 2: Read live V-F curve from first NVIDIA GPU
	fmt.Println("Step 2: Reading live GPU information...")

	// Get GPU marketing name for easy identification
	gpuName, err := nvvf.GetGPUName(0)
	if err != nil {
		fmt.Printf("Warning: Could not read GPU name: %v\n", err)
		gpuName = "Unknown GPU"
	}
	fmt.Printf("GPU 0: %s\n", gpuName)

	// Read clock domain ranges for OC validation
	fmt.Println("\nStep 3: Reading clock domain information...")
	clkDomains, err := nvvf.ReadNvAPIClkDomains(0)
	if err != nil {
		fmt.Printf("Warning: Could not read clock domains: %v\n", err)
		fmt.Println("  (OC validation information will be unavailable)")
	} else {
		fmt.Println("Safe Overclocking Ranges:")
		for _, domain := range clkDomains {
			minMHz := float64(domain.MinOffsetKHz) / 1000.0
			maxMHz := float64(domain.MaxOffsetKHz) / 1000.0
			fmt.Printf("  %s: %+.0f MHz to %+.0f MHz\n", domain.Domain, minMHz, maxMHz)
		}
	}

	// Read live V-F curve
	fmt.Println("\nStep 4: Reading live V-F curve from NVIDIA GPU...")
	livePoints, err := nvvf.ReadNvAPIVF(0)
	if err != nil {
		fmt.Printf("Error: Could not read live V-F curve: %v\n", err)
		fmt.Println()
		fmt.Println("This error typically occurs when:")
		fmt.Println("  - No NVIDIA GPU is present in the system")
		fmt.Println("  - NvAPI is not available (Windows: nvapi64.dll, Linux: libnvidia-api.so.1)")
		fmt.Println("  - NVIDIA driver is not installed or is outdated")
		fmt.Println()
		fmt.Println("On Windows, ensure you have NVIDIA drivers installed from nvidia.com")
		fmt.Println("On Linux, ensure you have the proprietary NVIDIA driver installed.")
		os.Exit(1)
	}

	fmt.Printf("Successfully read %d voltage points from GPU 0\n", len(livePoints))
	fmt.Println()

	// Build liveFreqs map: voltage (mV) -> effective frequency (MHz)
	// This is the format expected by msiaf.MatchProfileAgainstLive
	liveFreqs := make(map[float32]float64)
	for _, pt := range livePoints {
		liveFreqs[float32(pt.VoltageMV)] = pt.BaseFreqMHz
	}

	// Display live V-F curve summary
	fmt.Println("Live V-F Curve Summary:")
	fmt.Printf("  Voltage range: %.0f - %.0f mV\n", nvvf.VFMinVoltage(livePoints), nvvf.VFMaxVoltage(livePoints))

	// Find min/max frequencies for active points
	minFreq, maxFreq := nvvf.VFRange(livePoints)
	fmt.Printf("  Frequency range: %.0f - %.0f MHz\n", minFreq, maxFreq)
	fmt.Printf("  GPU: %s\n", gpuName)
	fmt.Println()

	// Step 5: Match each hardware profile against live data
	fmt.Println("Step 5: Matching profiles against live V-F curve...")
	fmt.Println("Using 10 MHz tolerance (accounts for minor driver variations)")
	fmt.Println()

	for i, profileInfo := range result.HardwareProfiles {
		fmt.Printf("--- Profile %d: %s ---\n", i+1, filepath.Base(profileInfo.FilePath))
		fmt.Printf("GPU: %s\n", profileInfo.GetGPUDescription())
		fmt.Printf("Matching against: %s\n", gpuName)
		fmt.Println()

		// Load hardware profile
		hwProfile, err := profileInfo.LoadProfile()
		if err != nil {
			fmt.Printf("Error loading profile: %v\n", err)
			fmt.Println()
			continue
		}

		// Match all profile slots (Startup + Profile1-5) against live data
		results, err := msiaf.MatchProfileAgainstLive(liveFreqs, hwProfile, 10.0)
		if err != nil {
			fmt.Printf("Error matching profiles: %v\n", err)
			fmt.Println()
			continue
		}

		// Separate Startup (currently applied) from saved profile slots
		startupResult := results[0] // Index 0 is always Startup
		profileResults := results[1:]

		// Display match results
		fmt.Println("V-F Curve Analysis:")
		fmt.Printf("  Startup (applied settings): %s\n", startupResult.StringWithoutSlotName())

		// Check if Startup matches live data (it SHOULD)
		startupMatches := startupResult.IsMatch(0.9) // 90% confidence threshold

		if !startupMatches {
			fmt.Println()
			fmt.Println("⚠ WARNING: Live GPU V-F curve does NOT match Startup profile!")
			fmt.Println("  This indicates:")
			fmt.Println("    - Settings were modified outside MSI Afterburner, OR")
			fmt.Println("    - MSI Afterburner did not apply settings correctly, OR")
			fmt.Println("    - Profile data is stale/outdated")
		}
		fmt.Println()

		// Check saved profile slots (Profile1-5)
		fmt.Println("Saved Profile Slots:")
		hasMatchingProfile := false
		for _, r := range profileResults {
			if r.TotalPoints > 0 {
				marker := "  "
				if r.IsMatch(0.9) {
					marker = "✓ "
					hasMatchingProfile = true
				}
				fmt.Printf("  %s%s\n", marker, r.StringWithoutSlotName())
			} else {
				fmt.Printf("  %s: (empty slot)\n", r.SlotName)
			}
		}

		// Display conclusion
		fmt.Println()
		if startupMatches && hasMatchingProfile {
			fmt.Println("✓ Status: Normal - Startup profile is applied and matches a saved profile")
		} else if startupMatches && !hasMatchingProfile {
			fmt.Println("✓ Status: Normal - Startup profile is applied")
			fmt.Println("  Note: No saved profile slots match (settings may be unsaved modifications)")
		} else if !startupMatches && hasMatchingProfile {
			fmt.Println("⚠ Status: Mismatch - A saved profile matches live GPU, but Startup does not")
			fmt.Println("  This is unusual - MSI Afterburner should have applied Startup settings")
		} else {
			fmt.Println("⚠ Status: Unknown - Neither Startup nor saved profiles match live GPU")
			fmt.Println("  Settings may have been applied via other tools (e.g., nvidia-smi, third-party OC)")
		}
		fmt.Println()
	}

	fmt.Println("Profile matching complete.")
}
