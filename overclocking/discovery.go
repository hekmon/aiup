// Package overclocking provides high-level GPU overclocking orchestration.
package overclocking

import (
	"fmt"
	"sort"
	"strings"

	"github.com/hekmon/aiup/msiaf"
	"github.com/hekmon/aiup/nvvf"
)

// GPUInfo contains complete information about a detected NVIDIA GPU with its MSI Afterburner profile.
// All fields are JSON-serializable for MCP/API compatibility.
type GPUInfo struct {
	Index           int    `json:"index"`            // NvAPI GPU index (0, 1, 2, ...)
	Name            string `json:"name"`             // Marketing name from NvAPI (e.g., "NVIDIA GeForce RTX 5090")
	VendorID        string `json:"vendor_id"`        // PCI vendor ID from profile (e.g., "10DE")
	DeviceID        string `json:"device_id"`        // PCI device ID from profile (e.g., "2B85")
	SubsystemID     string `json:"subsystem_id"`     // PCI subsystem ID from profile (e.g., "89EC1043")
	BusNumber       int    `json:"bus_number"`       // PCI bus number from profile
	DeviceNumber    int    `json:"device_number"`    // PCI device number from profile
	FunctionNumber  int    `json:"function_number"`  // PCI function number from profile
	ProfilePath     string `json:"profile_path"`     // Path to the hardware profile .cfg file
	Manufacturer    string `json:"manufacturer"`     // Card manufacturer from SubsystemID (e.g., "ASUS")
	FullDescription string `json:"full_description"` // Complete description (e.g., "ASUS NVIDIA GeForce RTX 5090")
}

// DiscoveryResult contains the result of GPU and profile discovery.
// All fields are JSON-serializable for MCP/API compatibility.
type DiscoveryResult struct {
	ProfilesDir      string    `json:"profiles_dir"`       // Path to MSI Afterburner Profiles directory
	GlobalConfigPath string    `json:"global_config_path"` // Path to MSIAfterburner.cfg
	GPUs             []GPUInfo `json:"gpus"`               // All detected GPUs with profiles
	Errors           []string  `json:"errors,omitempty"`   // Errors encountered (omitempty for clean output)
}

// ScanGPUs discovers NVIDIA GPUs by scanning MSI Afterburner profiles and correlating with NvAPI.
//
// This function:
//  1. Scans the profiles directory for hardware profile .cfg files
//  2. Queries NvAPI for all detected NVIDIA GPUs
//  3. Correlates profiles with GPUs by matching marketing names
//  4. Returns an error if any profile cannot be matched to a physical GPU
//
// The profilesDir parameter must be provided (MCP layer should use default).
// Typical paths:
//   - Windows: C:\Program Files (x86)\MSI Afterburner\Profiles\
//   - Linux: ~/.msi-afterburner/Profiles/ (if using Proton/Wine)
//
// Returns DiscoveryResult with all matched GPUs and their profiles.
// Returns an error if:
//   - Profiles directory cannot be read
//   - Global config file is missing
//   - No hardware profiles found
//   - Any profile cannot be matched to a physical GPU
func ScanGPUs(profilesDir string) (*DiscoveryResult, error) {
	result := &DiscoveryResult{
		ProfilesDir: profilesDir,
		GPUs:        make([]GPUInfo, 0),
		Errors:      make([]string, 0),
	}

	// Step 1: Scan MSI Afterburner profiles directory
	scanResult, err := msiaf.Scan(profilesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to scan profiles directory: %w", err)
	}

	result.GlobalConfigPath = scanResult.GlobalConfigPath

	// Step 2: Query NvAPI for all detected GPUs
	// We don't know how many GPUs exist, so we enumerate until GetGPUName fails
	nvvGPUs := make(map[int]string) // index -> GPU name
	for i := 0; i < 64; i++ {       // NvAPI supports up to 64 GPUs
		name, err := nvvf.GetGPUName(i)
		if err != nil {
			// Stop enumeration when we hit an invalid index
			break
		}
		nvvGPUs[i] = name
	}

	// If no GPUs detected by NvAPI, that's an error (profiles exist but no hardware)
	if len(nvvGPUs) == 0 {
		return nil, fmt.Errorf("no NVIDIA GPUs detected by NvAPI (profiles exist but no matching hardware)")
	}

	// Step 3: Correlate profiles with NvAPI GPUs
	// Track which NvAPI GPUs have been matched to avoid duplicates
	matchedNvvIndices := make(map[int]bool)

	for _, profile := range scanResult.HardwareProfiles {
		// Get the full GPU description from the profile's PCI IDs
		// This gives us something like "ASUS NVIDIA GeForce RTX 5090"
		profileDesc := profile.GetGPUDescription()

		// Try to find a matching NvAPI GPU
		// Match by checking if NvAPI name is contained in the profile description
		var matchedIndex *int
		var matchedName string

		for idx, nvvName := range nvvGPUs {
			if matchGPUName(profileDesc, nvvName) {
				// Check if this NvAPI GPU is already matched to another profile
				if matchedNvvIndices[idx] {
					result.Errors = append(result.Errors, fmt.Sprintf("GPU %d (%s) matched by multiple profiles", idx, nvvName))
					continue
				}
				matchedIndex = &idx
				matchedName = nvvName
				break
			}
		}

		if matchedIndex == nil {
			// Profile exists but no matching GPU detected - this is an error
			return nil, fmt.Errorf("profile %s (%s) has no matching physical GPU detected by NvAPI", profile.FilePath, profileDesc)
		}

		// Mark this NvAPI GPU as matched
		matchedNvvIndices[*matchedIndex] = true

		// Build GPUInfo from profile data and NvAPI name
		gpuInfo := GPUInfo{
			Index:           *matchedIndex,
			Name:            matchedName,
			VendorID:        profile.VendorID,
			DeviceID:        profile.DeviceID,
			SubsystemID:     profile.SubsystemID,
			BusNumber:       profile.BusNumber,
			DeviceNumber:    profile.DeviceNumber,
			FunctionNumber:  profile.FunctionNumber,
			ProfilePath:     profile.FilePath,
			Manufacturer:    profile.GetManufacturer(),
			FullDescription: profileDesc,
		}

		result.GPUs = append(result.GPUs, gpuInfo)
	}

	// Step 4: Check for NvAPI GPUs without profiles
	// These are errors because profiles are required for overclocking
	for idx, nvvName := range nvvGPUs {
		if !matchedNvvIndices[idx] {
			result.Errors = append(result.Errors, fmt.Sprintf("GPU %d (%s) detected but has no MSI Afterburner profile (user must create profile first)", idx, nvvName))
		}
	}

	// Step 5: Sort GPUs by index for consistent output
	sort.Slice(result.GPUs, func(i, j int) bool {
		return result.GPUs[i].Index < result.GPUs[j].Index
	})

	return result, nil
}

// matchGPUName checks if an NvAPI GPU name matches a profile's catalog description.
//
// Example:
//
//	profileDesc = "ASUS NVIDIA GeForce RTX 5090"
//	nvvName = "NVIDIA GeForce RTX 5090"
//	match = true (nvvName is contained in profileDesc)
//
// This handles the case where catalog includes manufacturer prefix but NvAPI doesn't.
func matchGPUName(profileDesc string, nvvName string) bool {
	if nvvName == "" || profileDesc == "" {
		return false
	}

	// Direct containment check (case-sensitive, both should be properly capitalized)
	if strings.Contains(profileDesc, nvvName) {
		return true
	}

	// Fallback: case-insensitive check
	profileLower := strings.ToLower(profileDesc)
	nvvLower := strings.ToLower(nvvName)

	if strings.Contains(profileLower, nvvLower) {
		return true
	}

	return false
}

// String returns a human-readable summary of discovered GPUs.
func (r *DiscoveryResult) String() string {
	if r == nil || len(r.GPUs) == 0 {
		return "No GPUs discovered"
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Discovered %d GPU(s):\n", len(r.GPUs))

	for _, gpu := range r.GPUs {
		fmt.Fprintf(&sb, "  [%d] %s (%s)\n", gpu.Index, gpu.FullDescription, gpu.ProfilePath)
	}

	if len(r.Errors) > 0 {
		fmt.Fprintf(&sb, "\nErrors (%d):\n", len(r.Errors))
		for _, err := range r.Errors {
			fmt.Fprintf(&sb, "  - %s\n", err)
		}
	}

	return sb.String()
}
