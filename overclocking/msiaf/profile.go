// Package msiaf provides tooling for working with MSI Afterburner profiles and configuration.
package msiaf

import (
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strings"
)

// OffsetMode represents how a V-F curve offset is configured in a profile.
type OffsetMode int

const (
	// OffsetModeUnknown indicates the offset mode could not be determined.
	OffsetModeUnknown OffsetMode = iota

	// OffsetModeFixedOffset indicates a uniform offset applied to all V-F points
	// (typically set via the Core (MHz) slider in MSI Afterburner UI).
	// CoreClkBoost field contains the offset value in kHz.
	OffsetModeFixedOffset

	// OffsetModeCustomCurve indicates individual offsets per V-F point
	// (typically set via the Curve Editor in MSI Afterburner UI).
	// CoreClkBoost field is set to 1000000 kHz (default/max value).
	OffsetModeCustomCurve

	// OffsetModeInvalid indicates an inconsistent state where CoreClkBoost
	// suggests fixed offset but V-F curve points have varying offsets.
	OffsetModeInvalid

	// coreClkBoostCustomCurveValue is the value MSI Afterburner sets when
	// using the curve editor (1000000 kHz = 1000 MHz).
	coreClkBoostCustomCurveValue = 1000000
)

// String returns a human-readable description of the offset mode.
func (m OffsetMode) String() string {
	switch m {
	case OffsetModeFixedOffset:
		return "Fixed Offset (slider mode)"
	case OffsetModeCustomCurve:
		return "Custom Curve (curve editor)"
	case OffsetModeInvalid:
		return "Invalid (inconsistent state)"
	default:
		return "Unknown"
	}
}

// FanMode represents how the GPU fan is configured in a profile.
type FanMode int

const (
	// FanModeAuto indicates automatic fan control (temperature-based curve).
	FanModeAuto FanMode = iota

	// FanModeManual indicates manual fan speed control (fixed percentage).
	FanModeManual
)

// String returns a human-readable description of the fan mode.
func (m FanMode) String() string {
	switch m {
	case FanModeAuto:
		return "auto"
	case FanModeManual:
		return "manual"
	default:
		return "unknown"
	}
}

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

// GetOffsetMode detects whether the profile uses fixed offset (slider mode)
// or custom curve (curve editor mode) by analyzing CoreClkBoost and V-F curve data.
//
// Detection logic:
//   - CoreClkBoost == 1000000 kHz → Custom Curve mode
//   - CoreClkBoost != 1000000 kHz → Fixed Offset mode (value = uniform offset)
//   - Mismatch between CoreClkBoost and V-F curve → Invalid state
//
// Returns:
//   - OffsetModeFixedOffset: All V-F points have uniform offset (any value, even +1000 MHz)
//   - OffsetModeCustomCurve: V-F points have varying offsets per voltage point
//   - OffsetModeInvalid: Unable to parse V-F curve or other errors
//   - OffsetModeUnknown: Missing CoreClkBoost or VFCurve data
//
// Note: CoreClkBoost = 1000000 kHz is used as a marker for custom curve mode,
// but the V-F curve is always checked to handle the edge case where a user
// sets a fixed offset of exactly +1000 MHz via the slider.
func (ps *ProfileSection) GetOffsetMode() OffsetMode {
	// Check if we have the necessary data
	if ps.CoreClkBoost == nil || len(ps.VFCurve) == 0 {
		return OffsetModeUnknown
	}

	// Parse V-F curve to analyze offset pattern
	hexStr := hex.EncodeToString(ps.VFCurve)
	curve, err := UnmarshalVFControlCurve(hexStr)
	if err != nil {
		return OffsetModeInvalid
	}

	// Collect offsets from all active points
	var offsets []float64
	for _, point := range curve.Points {
		if point.IsActive {
			offsets = append(offsets, float64(point.OffsetMHz))
		}
	}

	// Need at least one active point
	if len(offsets) == 0 {
		return OffsetModeUnknown
	}

	// Check if all offsets are uniform (within tolerance)
	const toleranceMHz = 1.0 // Allow 1 MHz tolerance for floating point precision
	firstOffset := offsets[0]
	isUniform := true

	for _, offset := range offsets[1:] {
		if math.Abs(offset-firstOffset) > toleranceMHz {
			isUniform = false
			break
		}
	}

	if isUniform {
		return OffsetModeFixedOffset
	}
	return OffsetModeCustomCurve
}

// GetFixedOffset returns the fixed offset value in MHz if the profile uses
// fixed offset mode. Returns 0, false if not in fixed offset mode.
//
// This method extracts the offset from the V-F curve when CoreClkBoost is at
// the marker value (1000000 kHz), ensuring correct detection even for +1000 MHz
// fixed offsets set via the slider.
//
// Example:
//
//	offset, ok := profile.Startup.GetFixedOffset()
//	if ok {
//	    fmt.Printf("Fixed offset: +%d MHz\n", offset)
//	}
func (ps *ProfileSection) GetFixedOffset() (int, bool) {
	if ps.CoreClkBoost == nil || len(ps.VFCurve) == 0 {
		return 0, false
	}

	mode := ps.GetOffsetMode()
	if mode != OffsetModeFixedOffset {
		return 0, false
	}

	coreBoost := *ps.CoreClkBoost

	// Edge case: CoreClkBoost = 1000000 kHz could be either:
	// - Custom curve mode (marker value)
	// - Fixed +1000 MHz offset (slider at max)
	// Since GetOffsetMode() already confirmed fixed offset, extract from V-F curve
	if coreBoost == coreClkBoostCustomCurveValue {
		// Parse V-F curve to get the actual offset value
		hexStr := hex.EncodeToString(ps.VFCurve)
		curve, err := UnmarshalVFControlCurve(hexStr)
		if err != nil {
			return 0, false
		}

		// Get offset from first active point (all should be uniform)
		for _, point := range curve.Points {
			if point.IsActive {
				return int(point.OffsetMHz), true
			}
		}
		return 0, false
	}

	// Normal case: CoreClkBoost contains the actual offset value
	return coreBoost / 1000, true
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

// GetFanMode returns the fan mode, or FanModeAuto if not set.
func (ps *ProfileSection) GetFanMode() FanMode {
	if ps.FanMode == nil {
		return FanModeAuto
	}
	return FanMode(*ps.FanMode)
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

// SetVFCurve sets the V-F curve from a hex-encoded blob.
// The hexData should be the raw hex string (no "0x" prefix or "h" suffix).
func (ps *ProfileSection) SetVFCurve(hexData string) {
	ps.VFCurve = parseHexBlob(hexData)
}

// SetVFCurveFromCurve serializes a VFControlCurveInfo and sets it as the V-F curve.
// Returns an error if marshaling fails.
func (ps *ProfileSection) SetVFCurveFromCurve(curve *VFControlCurveInfo) error {
	hexData, err := curve.Marshal()
	if err != nil {
		return err
	}
	ps.VFCurve = parseHexBlob(hexData)
	return nil
}

// SaveAs writes the hardware profile to the specified file path.
// The profile is serialized in INI format compatible with MSI Afterburner.
//
// Example:
//
//	err := hwProfile.SaveAs("backup.cfg")
//	if err != nil {
//	    return err
//	}
func (h *HardwareProfile) SaveAs(filePath string) error {
	content, err := h.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize profile: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write profile file: %w", err)
	}

	return nil
}

// Save writes the hardware profile back to its original file path.
// A backup is automatically created with .bak extension before overwriting.
//
// Example:
//
//	err := hwProfile.Save()
//	if err != nil {
//	    return err
//	}
func (h *HardwareProfile) Save() error {
	if h.filePath == "" {
		return fmt.Errorf("cannot save: profile has no file path")
	}

	// Create backup before overwriting
	backupPath := h.filePath + ".bak"
	if err := os.Rename(h.filePath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Write new profile
	if err := h.SaveAs(h.filePath); err != nil {
		// Try to restore backup on failure
		os.Rename(backupPath, h.filePath)
		return err
	}

	// Remove backup on success
	os.Remove(backupPath)
	return nil
}

// Serialize converts the HardwareProfile to INI format.
// This method is exported for testing and debugging purposes.
func (h *HardwareProfile) Serialize() (string, error) {
	var sb strings.Builder

	// Write Startup section
	sb.WriteString("[Startup]\n")
	h.serializeProfileSection(&sb, &h.Startup)
	sb.WriteString("\n")

	// Write Profile1-5 sections
	for i := range h.Profiles {
		slot := &h.Profiles[i]
		if !slot.IsEmpty {
			sb.WriteString(fmt.Sprintf("[Profile%d]\n", i+1))
			h.serializeProfileSection(&sb, &slot.ProfileSection)
			sb.WriteString("\n")
		}
	}

	// Write Defaults section if it has settings
	if h.Defaults.HasSettings() {
		sb.WriteString("[Defaults]\n")
		h.serializeProfileSection(&sb, &h.Defaults)
		sb.WriteString("\n")
	}

	// Write PreSuspendedMode section if it has settings
	if h.PreSuspendedMode.HasSettings() {
		sb.WriteString("[PreSuspendedMode]\n")
		h.serializeProfileSection(&sb, &h.PreSuspendedMode)
		sb.WriteString("\n")
	}

	// Write Settings section if it has values
	if h.Settings.CaptureDefaults != nil {
		sb.WriteString("[Settings]\n")
		sb.WriteString(fmt.Sprintf("CaptureDefaults=%d\n", *h.Settings.CaptureDefaults))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// serializeProfileSection writes a ProfileSection to the string builder.
// All keys are written, with empty values for nil fields.
func (h *HardwareProfile) serializeProfileSection(sb *strings.Builder, ps *ProfileSection) {
	if ps.Format != nil {
		sb.WriteString(fmt.Sprintf("Format=%d\n", *ps.Format))
	} else {
		sb.WriteString("Format=\n")
	}
	if ps.PowerLimit != nil {
		sb.WriteString(fmt.Sprintf("PowerLimit=%d\n", *ps.PowerLimit))
	} else {
		sb.WriteString("PowerLimit=\n")
	}
	if ps.CoreClkBoost != nil {
		sb.WriteString(fmt.Sprintf("CoreClkBoost=%d\n", *ps.CoreClkBoost))
	} else {
		sb.WriteString("CoreClkBoost=\n")
	}
	if len(ps.VFCurve) > 0 {
		sb.WriteString(fmt.Sprintf("VFCurve=%Xh\n", ps.VFCurve))
	} else {
		sb.WriteString("VFCurve=\n")
	}
	if ps.MemClkBoost != nil {
		sb.WriteString(fmt.Sprintf("MemClkBoost=%d\n", *ps.MemClkBoost))
	} else {
		sb.WriteString("MemClkBoost=\n")
	}
	if ps.FanMode != nil {
		sb.WriteString(fmt.Sprintf("FanMode=%d\n", *ps.FanMode))
	} else {
		sb.WriteString("FanMode=\n")
	}
	if ps.FanSpeed != nil {
		sb.WriteString(fmt.Sprintf("FanSpeed=%d\n", *ps.FanSpeed))
	} else {
		sb.WriteString("FanSpeed=\n")
	}
	if ps.FanMode2 != nil {
		sb.WriteString(fmt.Sprintf("FanMode2=%d\n", *ps.FanMode2))
	} else {
		sb.WriteString("FanMode2=\n")
	}
	if ps.FanSpeed2 != nil {
		sb.WriteString(fmt.Sprintf("FanSpeed2=%d\n", *ps.FanSpeed2))
	} else {
		sb.WriteString("FanSpeed2=\n")
	}
}
