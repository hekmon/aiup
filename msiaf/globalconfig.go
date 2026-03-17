// Package msiaf provides tooling for working with MSI Afterburner profiles and configuration.
package msiaf

import (
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// GlobalConfig represents the fully parsed MSIAfterburner.cfg file.
type GlobalConfig struct {
	Settings  Settings  `ini:"Settings"`
	ATIADLHAL ATIADLHAL `ini:"ATIADLHAL"`
}

// Settings contains all settings from the [Settings] section.
type Settings struct {
	// UI & Language
	Language           string
	Views              string
	Skin               string
	ShowHints          bool
	ShowTooltips       bool
	Fahrenheit         bool
	Time24             bool
	SingleTrayIconMode bool
	LCDGraph           bool

	// Update & First Run
	LastUpdateCheck      time.Time
	UpdateCheckingPeriod int
	FirstRun             bool
	FirstUserDefineClick bool
	FirstServerRun       bool

	// Low-Level Access
	LowLevelInterface       bool
	MMIOUserMode            bool
	HAL                     bool
	Driver                  bool
	UnlockVoltageControl    bool
	UnlockVoltageMonitoring bool
	ForceConstantVoltage    bool
	OEM                     bool

	// Startup & Window
	StartPosition    int
	StartMinimized   bool
	RememberSettings bool
	WindowX          int
	WindowY          int
	ProfileContents  int
	Profile2D        int
	Profile3D        int
	LockProfiles     bool

	// Polling & Timing
	HwPollPeriod time.Duration
	LCDFont      string

	// GPU Selection
	CurrentGpu  int
	Sync        bool
	Link        bool
	LinkThermal bool
	FanSync     bool
	CurrentFan  int

	// OSD (On-Screen Display)
	ShowOSDTime          bool
	CaptureOSD           bool
	OSDToggleHotkey      uint32
	OSDOnHotkey          uint32
	OSDOffHotkey         uint32
	OSDServerBlockHotkey uint32
	OSDLayout            int

	// Profile Hotkeys
	Profile1Hotkey uint32
	Profile2Hotkey uint32
	Profile3Hotkey uint32
	Profile4Hotkey uint32
	Profile5Hotkey uint32

	// Limiter Hotkeys
	LimiterToggleHotkey uint32
	LimiterOnHotkey     uint32
	LimiterOffHotkey    uint32

	// Capture Hotkeys
	ScreenCaptureHotkey  uint32
	VideoCaptureHotkey   uint32
	VideoPrerecordHotkey uint32
	BeginRecordHotkey    uint32
	EndRecordHotkey      uint32

	// PTT (Push-to-Talk) Hotkeys
	PTTHotkey  uint32
	PTT2Hotkey uint32

	// Logging Hotkeys
	BeginLoggingHotkey uint32
	EndLoggingHotkey   uint32
	ClearHistoryHotkey uint32

	// Benchmark
	BenchmarkPath   string
	AppendBenchmark bool

	// Screen Capture
	ScreenCaptureFormat  string
	ScreenCaptureFolder  string
	ScreenCaptureQuality int

	// Video Capture
	VideoCaptureFolder    string
	VideoCaptureFormat    string
	VideoCaptureQuality   int
	VideoCaptureFramerate int
	VideoCaptureFramesize uint32
	VideoCaptureThreads   uint32
	VideoCaptureContainer string
	AudioCaptureFlags     uint32
	VideoCaptureFlagsEx   uint32
	AudioCaptureFlags2    uint32

	// Video Prerecord
	VideoPrerecordSizeLimit int
	VideoPrerecordTimeLimit time.Duration
	AutoPrerecord           bool

	// Software Auto Fan Control
	SwAutoFanControl       bool
	SwAutoFanControlFlags  uint32
	SwAutoFanControlPeriod time.Duration
	SwAutoFanControlCurve  []byte
	SwAutoFanControlCurve2 []byte

	// Power Management
	RestoreAfterSuspendedMode bool
	PauseMonitoring           bool

	// Performance Profiler
	ShowPerformanceProfilerStatus bool
	ShowPerformanceProfilerPanel  bool

	// Monitoring Window
	AttachMonitoringWindow bool
	HideMonitoring         bool
	MonitoringWindowOnTop  bool

	// Logging
	LogPath     string
	EnableLog   bool
	RecreateLog bool
	LogLimit    int

	// Voltage & Frequency Window
	VFWindowX     int
	VFWindowY     int
	VFWindowW     int
	VFWindowH     int
	VFWindowOnTop bool
}

// ATIADLHAL contains settings from the [ATIADLHAL] section.
type ATIADLHAL struct {
	UnofficialOverclockingMode     int
	UnofficialOverclockingDrvReset int
	UnifiedActivityMonitoring      int
	EraseStartupSettings           int
}

// ParseGlobalConfig reads and parses the MSIAfterburner.cfg file.
// Returns a strongly typed GlobalConfig struct.
//
// The config file uses INI-style format:
// - Sections are denoted by [SectionName]
// - Key-value pairs are separated by '='
// - Lines starting with ';' or '#' are comments
//
// Example:
//
//	[Settings]
//	Language=EN
//	StartWithWindows=1
func ParseGlobalConfig(filePath string) (*GlobalConfig, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read global config file: %w", err)
	}

	config := &GlobalConfig{
		Settings:  Settings{},
		ATIADLHAL: ATIADLHAL{},
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
			case "Settings":
				parseSettingsField(&config.Settings, key, value)
			case "ATIADLHAL":
				parseATIADLHALField(&config.ATIADLHAL, key, value)
			}
		}
	}

	return config, nil
}

// parseSettingsField parses a single key=value pair into the Settings struct.
func parseSettingsField(s *Settings, key, value string) {
	switch key {
	case "Language":
		s.Language = value
	case "Views":
		s.Views = value
	case "Skin":
		s.Skin = value
	case "ShowHints":
		s.ShowHints = parseBool(value)
	case "ShowTooltips":
		s.ShowTooltips = parseBool(value)
	case "Fahrenheit":
		s.Fahrenheit = parseBool(value)
	case "Time24":
		s.Time24 = parseBool(value)
	case "SingleTrayIconMode":
		s.SingleTrayIconMode = parseBool(value)
	case "LCDGraph":
		s.LCDGraph = parseBool(value)

	case "LastUpdateCheck":
		s.LastUpdateCheck = parseHexTimestamp(value)
	case "UpdateCheckingPeriod":
		s.UpdateCheckingPeriod = parseInt(value)
	case "FirstRun":
		s.FirstRun = parseBool(value)
	case "FirstUserDefineClick":
		s.FirstUserDefineClick = parseBool(value)
	case "FirstServerRun":
		s.FirstServerRun = parseBool(value)

	case "LowLevelInterface":
		s.LowLevelInterface = parseBool(value)
	case "MMIOUserMode":
		s.MMIOUserMode = parseBool(value)
	case "HAL":
		s.HAL = parseBool(value)
	case "Driver":
		s.Driver = parseBool(value)
	case "UnlockVoltageControl":
		s.UnlockVoltageControl = parseBool(value)
	case "UnlockVoltageMonitoring":
		s.UnlockVoltageMonitoring = parseBool(value)
	case "ForceConstantVoltage":
		s.ForceConstantVoltage = parseBool(value)
	case "OEM":
		s.OEM = parseBool(value)

	case "StartWithWindows":
		s.StartPosition = 1 // Map to StartPosition for API consistency
		s.StartMinimized = parseBool(value)
	case "StartMinimized":
		s.StartMinimized = parseBool(value)
	case "RememberSettings":
		s.RememberSettings = parseBool(value)
	case "WindowX":
		s.WindowX = parseInt(value)
	case "WindowY":
		s.WindowY = parseInt(value)
	case "ProfileContents":
		s.ProfileContents = parseInt(value)
	case "Profile2D":
		s.Profile2D = parseInt(value)
	case "Profile3D":
		s.Profile3D = parseInt(value)
	case "LockProfiles":
		s.LockProfiles = parseBool(value)

	case "HwPollPeriod":
		s.HwPollPeriod = parseDuration(value)
	case "LCDFont":
		s.LCDFont = value

	case "CurrentGpu":
		s.CurrentGpu = parseInt(value)
	case "Sync":
		s.Sync = parseBool(value)
	case "Link":
		s.Link = parseBool(value)
	case "LinkThermal":
		s.LinkThermal = parseBool(value)
	case "FanSync":
		s.FanSync = parseBool(value)
	case "CurrentFan":
		s.CurrentFan = parseInt(value)

	case "ShowOSDTime":
		s.ShowOSDTime = parseBool(value)
	case "CaptureOSD":
		s.CaptureOSD = parseBool(value)
	case "OSDToggleHotkey":
		s.OSDToggleHotkey = parseHexUint32(value)
	case "OSDOnHotkey":
		s.OSDOnHotkey = parseHexUint32(value)
	case "OSDOffHotkey":
		s.OSDOffHotkey = parseHexUint32(value)
	case "OSDServerBlockHotkey":
		s.OSDServerBlockHotkey = parseHexUint32(value)
	case "OSDLayout":
		s.OSDLayout = parseInt(value)

	case "Profile1Hotkey":
		s.Profile1Hotkey = parseHexUint32(value)
	case "Profile2Hotkey":
		s.Profile2Hotkey = parseHexUint32(value)
	case "Profile3Hotkey":
		s.Profile3Hotkey = parseHexUint32(value)
	case "Profile4Hotkey":
		s.Profile4Hotkey = parseHexUint32(value)
	case "Profile5Hotkey":
		s.Profile5Hotkey = parseHexUint32(value)

	case "LimiterToggleHotkey":
		s.LimiterToggleHotkey = parseHexUint32(value)
	case "LimiterOnHotkey":
		s.LimiterOnHotkey = parseHexUint32(value)
	case "LimiterOffHotkey":
		s.LimiterOffHotkey = parseHexUint32(value)

	case "ScreenCaptureHotkey":
		s.ScreenCaptureHotkey = parseHexUint32(value)
	case "VideoCaptureHotkey":
		s.VideoCaptureHotkey = parseHexUint32(value)
	case "VideoPrerecordHotkey":
		s.VideoPrerecordHotkey = parseHexUint32(value)
	case "BeginRecordHotkey":
		s.BeginRecordHotkey = parseHexUint32(value)
	case "EndRecordHotkey":
		s.EndRecordHotkey = parseHexUint32(value)

	case "PTTHotkey":
		s.PTTHotkey = parseHexUint32(value)
	case "PTT2Hotkey":
		s.PTT2Hotkey = parseHexUint32(value)

	case "BeginLoggingHotkey":
		s.BeginLoggingHotkey = parseHexUint32(value)
	case "EndLoggingHotkey":
		s.EndLoggingHotkey = parseHexUint32(value)
	case "ClearHistoryHotkey":
		s.ClearHistoryHotkey = parseHexUint32(value)

	case "BenchmarkPath":
		s.BenchmarkPath = value
	case "AppendBenchmark":
		s.AppendBenchmark = parseBool(value)

	case "ScreenCaptureFormat":
		s.ScreenCaptureFormat = value
	case "ScreenCaptureFolder":
		s.ScreenCaptureFolder = value
	case "ScreenCaptureQuality":
		s.ScreenCaptureQuality = parseInt(value)

	case "VideoCaptureFolder":
		s.VideoCaptureFolder = value
	case "VideoCaptureFormat":
		s.VideoCaptureFormat = value
	case "VideoCaptureQuality":
		s.VideoCaptureQuality = parseInt(value)
	case "VideoCaptureFramerate":
		s.VideoCaptureFramerate = parseInt(value)
	case "VideoCaptureFramesize":
		s.VideoCaptureFramesize = parseHexUint32(value)
	case "VideoCaptureThreads":
		s.VideoCaptureThreads = parseHexUint32(value)
	case "VideoCaptureContainer":
		s.VideoCaptureContainer = value
	case "AudioCaptureFlags":
		s.AudioCaptureFlags = parseHexUint32(value)
	case "VideoCaptureFlagsEx":
		s.VideoCaptureFlagsEx = parseHexUint32(value)
	case "AudioCaptureFlags2":
		s.AudioCaptureFlags2 = parseHexUint32(value)

	case "VideoPrerecordSizeLimit":
		s.VideoPrerecordSizeLimit = parseInt(value)
	case "VideoPrerecordTimeLimit":
		s.VideoPrerecordTimeLimit = time.Duration(parseInt(value)) * time.Second
	case "AutoPrerecord":
		s.AutoPrerecord = parseBool(value)

	case "SwAutoFanControl":
		s.SwAutoFanControl = parseBool(value)
	case "SwAutoFanControlFlags":
		s.SwAutoFanControlFlags = parseHexUint32(value)
	case "SwAutoFanControlPeriod":
		s.SwAutoFanControlPeriod = parseDuration(value)
	case "SwAutoFanControlCurve":
		s.SwAutoFanControlCurve = parseHexBlob(value)
	case "SwAutoFanControlCurve2":
		s.SwAutoFanControlCurve2 = parseHexBlob(value)

	case "RestoreAfterSuspendedMode":
		s.RestoreAfterSuspendedMode = parseBool(value)
	case "PauseMonitoring":
		s.PauseMonitoring = parseBool(value)

	case "ShowPerformanceProfilerStatus":
		s.ShowPerformanceProfilerStatus = parseBool(value)
	case "ShowPerformanceProfilerPanel":
		s.ShowPerformanceProfilerPanel = parseBool(value)

	case "AttachMonitoringWindow":
		s.AttachMonitoringWindow = parseBool(value)
	case "HideMonitoring":
		s.HideMonitoring = parseBool(value)
	case "MonitoringWindowOnTop":
		s.MonitoringWindowOnTop = parseBool(value)

	case "LogPath":
		s.LogPath = value
	case "EnableLog":
		s.EnableLog = parseBool(value)
	case "RecreateLog":
		s.RecreateLog = parseBool(value)
	case "LogLimit":
		s.LogLimit = parseInt(value)

	case "VFWindowX":
		s.VFWindowX = parseInt(value)
	case "VFWindowY":
		s.VFWindowY = parseInt(value)
	case "VFWindowW":
		s.VFWindowW = parseInt(value)
	case "VFWindowH":
		s.VFWindowH = parseInt(value)
	case "VFWindowOnTop":
		s.VFWindowOnTop = parseBool(value)
	}
}

// parseATIADLHALField parses a single key=value pair into the ATIADLHAL struct.
func parseATIADLHALField(a *ATIADLHAL, key, value string) {
	switch key {
	case "UnofficialOverclockingMode":
		a.UnofficialOverclockingMode = parseInt(value)
	case "UnofficialOverclockingDrvReset":
		a.UnofficialOverclockingDrvReset = parseInt(value)
	case "UnifiedActivityMonitoring":
		a.UnifiedActivityMonitoring = parseInt(value)
	case "EraseStartupSettings":
		a.EraseStartupSettings = parseInt(value)
	}
}

// parseBool converts "1" to true, "0" to false.
func parseBool(value string) bool {
	return value == "1"
}

// parseInt parses a decimal integer, returns 0 on error.
func parseInt(value string) int {
	if value == "" {
		return 0
	}
	// Check if it's a hex value (ends with 'h')
	if strings.HasSuffix(value, "h") {
		hexStr := strings.TrimSuffix(value, "h")
		result, err := strconv.ParseInt(hexStr, 16, 64)
		if err != nil {
			return 0
		}
		return int(result)
	}
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return result
}

// parseHexUint32 parses a hex value (with or without 'h' suffix) to uint32.
func parseHexUint32(value string) uint32 {
	if value == "" {
		return 0
	}
	hexStr := strings.TrimSuffix(value, "h")
	result, err := strconv.ParseUint(hexStr, 16, 32)
	if err != nil {
		return 0
	}
	return uint32(result)
}

// parseDuration parses a millisecond value to time.Duration.
func parseDuration(value string) time.Duration {
	ms := parseInt(value)
	return time.Duration(ms) * time.Millisecond
}

// parseHexTimestamp parses a hex timestamp to time.Time.
// The timestamp appears to be Unix epoch in hex format.
func parseHexTimestamp(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	hexStr := strings.TrimSuffix(value, "h")
	ts, err := strconv.ParseInt(hexStr, 16, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

// parseHexBlob parses a hex string (with 'h' suffix) to []byte.
func parseHexBlob(value string) []byte {
	if value == "" {
		return nil
	}
	hexStr := strings.TrimSuffix(value, "h")
	// Hex blob should have even length
	if len(hexStr)%2 != 0 {
		return nil
	}
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil
	}
	return data
}
