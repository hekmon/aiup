// Package msiaf provides tooling for working with MSI Afterburner profiles and configuration.
package msiaf

import (
	"fmt"
	"os"
	"strings"
)

// HardwareProfile represents a fully parsed hardware-specific .cfg file.
type HardwareProfile struct {
	// Startup contains the currently active settings
	Startup ProfileSection

	// Profiles contains the 5 user-defined profile slots (Profile1-5)
	Profiles [5]ProfileSlot

	// Defaults contains factory default values
	Defaults ProfileSection

	// PreSuspendedMode contains the state before system suspension
	PreSuspendedMode ProfileSection

	// Settings contains miscellaneous profile metadata
	Settings ProfileMiscSettings

	// filePath stores the source file path (for reference)
	filePath string
}

// ProfileSection contains overclocking and fan control settings.
// Used by Startup, Profiles, Defaults, and PreSuspendedMode sections.
//
// All fields are pointers to distinguish between:
//   - nil: field not present in file
//   - non-nil: field explicitly set (even if value is 0)
type ProfileSection struct {
	Format       *int   // Format version (e.g., 2)
	PowerLimit   *int   // Power limit percentage (50-200 typical)
	CoreClkBoost *int   // Core clock offset in kHz
	VFCurve      []byte // Voltage-frequency curve (raw hex blob)
	MemClkBoost  *int   // Memory clock offset in kHz
	FanMode      *int   // Fan mode: 0=auto, 1=manual
	FanSpeed     *int   // Manual fan speed percentage (0-100)
	FanMode2     *int   // Second GPU fan mode
	FanSpeed2    *int   // Second GPU fan speed
}

// ProfileSlot represents a user-defined profile slot (Profile1-5).
type ProfileSlot struct {
	ProfileSection
	SlotNumber int  // 1-5
	IsEmpty    bool // true if section is missing or all fields empty
}

// ProfileMiscSettings contains miscellaneous profile settings.
type ProfileMiscSettings struct {
	CaptureDefaults *int // Capture-related defaults
}

// ParseHardwareProfile reads and parses a hardware profile .cfg file.
// Returns a strongly typed HardwareProfile struct.
//
// The profile file uses INI-style format:
//   - Sections are denoted by [SectionName]
//   - Key-value pairs are separated by '='
//   - Lines starting with ';' or '#' are comments
//
// Example:
//
//	[Startup]
//	Format=2
//	PowerLimit=100
//	CoreClkBoost=1000000
//	VFCurve=000002007F0000...
func ParseHardwareProfile(filePath string) (*HardwareProfile, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read hardware profile file: %w", err)
	}

	profile := &HardwareProfile{
		filePath: filePath,
		Profiles: [5]ProfileSlot{
			{SlotNumber: 1, IsEmpty: true},
			{SlotNumber: 2, IsEmpty: true},
			{SlotNumber: 3, IsEmpty: true},
			{SlotNumber: 4, IsEmpty: true},
			{SlotNumber: 5, IsEmpty: true},
		},
	}

	currentSection := ""
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		// Trim whitespace and carriage returns
		line = strings.TrimSpace(strings.TrimRight(line, "\r"))

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for section header
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			continue
		}

		// Parse key=value pair
		if key, value, ok := strings.Cut(line, "="); ok {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)

			switch currentSection {
			case "Startup":
				parseProfileSection(&profile.Startup, key, value)
			case "Profile1":
				profile.Profiles[0].IsEmpty = false
				parseProfileSlot(&profile.Profiles[0], key, value)
			case "Profile2":
				profile.Profiles[1].IsEmpty = false
				parseProfileSlot(&profile.Profiles[1], key, value)
			case "Profile3":
				profile.Profiles[2].IsEmpty = false
				parseProfileSlot(&profile.Profiles[2], key, value)
			case "Profile4":
				profile.Profiles[3].IsEmpty = false
				parseProfileSlot(&profile.Profiles[3], key, value)
			case "Profile5":
				profile.Profiles[4].IsEmpty = false
				parseProfileSlot(&profile.Profiles[4], key, value)
			case "Defaults":
				parseProfileSection(&profile.Defaults, key, value)
			case "PreSuspendedMode":
				parseProfileSection(&profile.PreSuspendedMode, key, value)
			case "Settings":
				parseProfileMiscSettings(&profile.Settings, key, value)
			}
		}
	}

	return profile, nil
}

// parseProfileSection parses a single key=value pair into the ProfileSection struct.
func parseProfileSection(ps *ProfileSection, key, value string) {
	// Skip empty values (common in PreSuspendedMode)
	if value == "" {
		return
	}

	switch key {
	case "Format":
		v := parseInt(value)
		ps.Format = &v
	case "PowerLimit":
		v := parseInt(value)
		ps.PowerLimit = &v
	case "CoreClkBoost":
		v := parseInt(value)
		ps.CoreClkBoost = &v
	case "VFCurve":
		ps.VFCurve = parseHexBlob(value)
	case "MemClkBoost":
		v := parseInt(value)
		ps.MemClkBoost = &v
	case "FanMode":
		v := parseInt(value)
		ps.FanMode = &v
	case "FanSpeed":
		v := parseInt(value)
		ps.FanSpeed = &v
	case "FanMode2":
		v := parseInt(value)
		ps.FanMode2 = &v
	case "FanSpeed2":
		v := parseInt(value)
		ps.FanSpeed2 = &v
	}
}

// parseProfileSlot parses a single key=value pair into the ProfileSlot struct.
func parseProfileSlot(slot *ProfileSlot, key, value string) {
	parseProfileSection(&slot.ProfileSection, key, value)
}

// parseProfileMiscSettings parses a single key=value pair into ProfileMiscSettings.
func parseProfileMiscSettings(s *ProfileMiscSettings, key, value string) {
	switch key {
	case "CaptureDefaults":
		v := parseInt(value)
		s.CaptureDefaults = &v
	}
}

// GetCaptureDefaults returns the capture defaults setting, or 0 if not set.
func (s *ProfileMiscSettings) GetCaptureDefaults() int {
	if s.CaptureDefaults == nil {
		return 0
	}
	return *s.CaptureDefaults
}

// SetCaptureDefaults sets the capture defaults setting.
func (s *ProfileMiscSettings) SetCaptureDefaults(v int) {
	s.CaptureDefaults = &v
}

// GetProfile returns the profile slot for the given slot number (1-5).
// Returns nil if the slot number is invalid.
func (h *HardwareProfile) GetProfile(slot int) *ProfileSlot {
	if slot < 1 || slot > 5 {
		return nil
	}
	return &h.Profiles[slot-1]
}

// GetCurrentSettings returns the currently active settings (Startup section).
func (h *HardwareProfile) GetCurrentSettings() *ProfileSection {
	return &h.Startup
}

// GetDefaults returns the factory default settings.
func (h *HardwareProfile) GetDefaults() *ProfileSection {
	return &h.Defaults
}

// FilePath returns the source file path of the hardware profile.
func (h *HardwareProfile) FilePath() string {
	return h.filePath
}

// HasSettings returns true if any meaningful settings are present.
// Useful for checking if a section is completely empty.
func (ps *ProfileSection) HasSettings() bool {
	return ps.Format != nil ||
		ps.PowerLimit != nil ||
		ps.CoreClkBoost != nil ||
		ps.VFCurve != nil ||
		ps.MemClkBoost != nil ||
		ps.FanMode != nil ||
		ps.FanSpeed != nil ||
		ps.FanMode2 != nil ||
		ps.FanSpeed2 != nil
}

// GetFormat returns the format version, or 0 if not set.
func (ps *ProfileSection) GetFormat() int {
	if ps.Format == nil {
		return 0
	}
	return *ps.Format
}

// GetPowerLimit returns the power limit percentage, or 0 if not set.
func (ps *ProfileSection) GetPowerLimit() int {
	if ps.PowerLimit == nil {
		return 0
	}
	return *ps.PowerLimit
}

// GetCoreClkBoost returns the core clock offset in kHz, or 0 if not set.
func (ps *ProfileSection) GetCoreClkBoost() int {
	if ps.CoreClkBoost == nil {
		return 0
	}
	return *ps.CoreClkBoost
}

// GetMemClkBoost returns the memory clock offset in kHz, or 0 if not set.
func (ps *ProfileSection) GetMemClkBoost() int {
	if ps.MemClkBoost == nil {
		return 0
	}
	return *ps.MemClkBoost
}

// GetCoreClkBoostMHz returns the core clock offset in MHz, or 0 if not set.
// This is a convenience method that converts from the raw kHz value.
func (ps *ProfileSection) GetCoreClkBoostMHz() int {
	return ps.GetCoreClkBoost() / 1000
}

// GetMemClkBoostMHz returns the memory clock offset in MHz, or 0 if not set.
// This is a convenience method that converts from the raw kHz value.
func (ps *ProfileSection) GetMemClkBoostMHz() int {
	return ps.GetMemClkBoost() / 1000
}

// GetFanMode returns the fan mode (0=auto, 1=manual), or 0 if not set.
func (ps *ProfileSection) GetFanMode() int {
	if ps.FanMode == nil {
		return 0
	}
	return *ps.FanMode
}

// GetFanSpeed returns the fan speed percentage, or 0 if not set.
func (ps *ProfileSection) GetFanSpeed() int {
	if ps.FanSpeed == nil {
		return 0
	}
	return *ps.FanSpeed
}

// GetFanMode2 returns the second GPU fan mode, or 0 if not set.
func (ps *ProfileSection) GetFanMode2() int {
	if ps.FanMode2 == nil {
		return 0
	}
	return *ps.FanMode2
}

// GetFanSpeed2 returns the second GPU fan speed, or 0 if not set.
func (ps *ProfileSection) GetFanSpeed2() int {
	if ps.FanSpeed2 == nil {
		return 0
	}
	return *ps.FanSpeed2
}

// SetFormat sets the format version.
func (ps *ProfileSection) SetFormat(v int) {
	ps.Format = &v
}

// SetPowerLimit sets the power limit percentage.
func (ps *ProfileSection) SetPowerLimit(v int) {
	ps.PowerLimit = &v
}

// SetCoreClkBoost sets the core clock offset in kHz.
func (ps *ProfileSection) SetCoreClkBoost(v int) {
	ps.CoreClkBoost = &v
}

// SetMemClkBoost sets the memory clock offset in kHz.
func (ps *ProfileSection) SetMemClkBoost(v int) {
	ps.MemClkBoost = &v
}

// SetFanMode sets the fan mode (0=auto, 1=manual).
func (ps *ProfileSection) SetFanMode(v int) {
	ps.FanMode = &v
}

// SetFanSpeed sets the fan speed percentage.
func (ps *ProfileSection) SetFanSpeed(v int) {
	ps.FanSpeed = &v
}

// SetFanMode2 sets the second GPU fan mode.
func (ps *ProfileSection) SetFanMode2(v int) {
	ps.FanMode2 = &v
}

// SetFanSpeed2 sets the second GPU fan speed.
func (ps *ProfileSection) SetFanSpeed2(v int) {
	ps.FanSpeed2 = &v
}

// LoadProfile reads and parses the hardware profile file for this hardware info.
func (h *HardwareProfileInfo) LoadProfile() (*HardwareProfile, error) {
	return ParseHardwareProfile(h.FilePath)
}
