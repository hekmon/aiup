package msiaf

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hekmon/aiup/overclocking/msiaf/catalog"
)

// GlobalConfigFile is the name of the global MSI Afterburner configuration file.
const GlobalConfigFile = "MSIAfterburner.cfg"

// HardwareProfilePattern matches hardware-specific profile filenames.
// Example: VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg (RTX 5090)
var HardwareProfilePattern = regexp.MustCompile(`^VEN_([0-9A-Fa-f]{4})&DEV_([0-9A-Fa-f]{4})&SUBSYS_([0-9A-Fa-f]{8})&REV_([0-9A-Fa-f]{2})&BUS_(\d+)&DEV_(\d+)&FN_(\d+)\.cfg$`)

// ScanResult contains the result of scanning a Profiles directory.
type ScanResult struct {
	// GlobalConfigPath is the path to MSIAfterburner.cfg, empty if not found.
	GlobalConfigPath string

	// HardwareProfiles is a list of hardware-specific profile files.
	// Each entry corresponds to one GPU.
	HardwareProfiles []HardwareProfileInfo

	// Errors contains any non-fatal errors encountered during scanning.
	Errors []string
}

// HardwareProfileInfo contains information about a hardware-specific profile file.
type HardwareProfileInfo struct {
	// FilePath is the full path to the .cfg file.
	FilePath string

	// VendorID is the hex vendor ID (e.g., "10DE" for NVIDIA).
	VendorID string

	// DeviceID is the hex device ID (e.g., "2B85" for RTX 5090).
	DeviceID string

	// SubsystemID is the hex subsystem ID (manufacturer-specific).
	SubsystemID string

	// Revision is the hex revision code.
	Revision string

	// BusNumber is the PCI bus number.
	BusNumber int

	// DeviceNumber is the PCI device number on the bus.
	DeviceNumber int

	// FunctionNumber is the PCI function number.
	FunctionNumber int
}

// GetGPUInfo returns GPU information from the catalog based on VendorID and DeviceID.
func (h HardwareProfileInfo) GetGPUInfo() catalog.GPUInfo {
	return catalog.LookupGPU(h.VendorID, h.DeviceID)
}

// GetManufacturer returns the manufacturer name from SubsystemID.
func (h HardwareProfileInfo) GetManufacturer() string {
	return catalog.LookupManufacturer(h.SubsystemID)
}

// GetGPUDescription returns a complete human-readable GPU description.
// Format: "Manufacturer Vendor GPUName" (e.g., "ASUS NVIDIA GeForce RTX 5090")
func (h HardwareProfileInfo) GetGPUDescription() string {
	return catalog.GetFullGPUDescription(h.VendorID, h.DeviceID, h.SubsystemID)
}

// String returns a formatted string with all resolved hardware information.
func (h HardwareProfileInfo) String() string {
	gpuInfo := h.GetGPUInfo()
	manufacturer := h.GetManufacturer()
	return fmt.Sprintf("%s %s (Bus %d, Device %d, Function %d)",
		manufacturer, gpuInfo.GPUName, h.BusNumber, h.DeviceNumber, h.FunctionNumber)
}

// GetFilename returns the expected filename for this hardware profile.
// Format: VEN_10DE&DEV_2B85&SUBSYS_89EC1043&REV_A1&BUS_1&DEV_0&FN_0.cfg
func (h HardwareProfileInfo) GetFilename() string {
	return fmt.Sprintf("VEN_%s&DEV_%s&SUBSYS_%s&REV_%s&BUS_%d&DEV_%d&FN_%d.cfg",
		h.VendorID, h.DeviceID, h.SubsystemID, h.Revision,
		h.BusNumber, h.DeviceNumber, h.FunctionNumber)
}

// Scan profiles directory and validate required files exist.
//
// Returns:
// - ScanResult with global config path and list of hardware profiles
// - error if directory cannot be read or global config is missing
func Scan(profilesDir string) (*ScanResult, error) {
	result := &ScanResult{}

	// Read directory entries
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read profiles directory: %w", err)
	}

	// Process each entry
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Check for global config file
		if name == GlobalConfigFile {
			result.GlobalConfigPath = filepath.Join(profilesDir, name)
			continue
		}

		// Check for hardware-specific profile files
		if matches := HardwareProfilePattern.FindStringSubmatch(name); matches != nil {
			info := HardwareProfileInfo{
				FilePath:    filepath.Join(profilesDir, name),
				VendorID:    strings.ToUpper(matches[1]),
				DeviceID:    strings.ToUpper(matches[2]),
				SubsystemID: strings.ToUpper(matches[3]),
				Revision:    strings.ToUpper(matches[4]),
			}

			// Parse numeric fields
			fmt.Sscanf(matches[5], "%d", &info.BusNumber)
			fmt.Sscanf(matches[6], "%d", &info.DeviceNumber)
			fmt.Sscanf(matches[7], "%d", &info.FunctionNumber)

			// Filter out placeholder/generic hardware (DEV_0000)
			if info.DeviceID != "0000" {
				result.HardwareProfiles = append(result.HardwareProfiles, info)
			} else {
				result.Errors = append(result.Errors, fmt.Sprintf("skipping placeholder profile: %s", name))
			}
		}
	}

	// Validate: global config must exist
	if result.GlobalConfigPath == "" {
		return nil, fmt.Errorf("global config file '%s' not found in %s", GlobalConfigFile, profilesDir)
	}

	// Validate: at least one hardware profile should exist
	if len(result.HardwareProfiles) == 0 {
		return nil, fmt.Errorf("no hardware-specific profile files (VEN_*.cfg) found in %s", profilesDir)
	}

	return result, nil
}
