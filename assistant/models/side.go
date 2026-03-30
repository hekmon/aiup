package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/hekmon/aiup/assistant/commands"
	"github.com/hekmon/aiup/overclocking"
	"github.com/hekmon/aiup/overclocking/msiaf/catalog"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	sidePanelTitleStyle = lipgloss.NewStyle().
				Foreground(panelBorderColor).
				Bold(true)
	sidePanelTitleIconPrefix = "📋 "
	sidePanelTitleDefault    = "GPU Informations"

	// Section styles
	sectionLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("249")). // Gray
				Bold(true)

	// Detail row styles
	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("144")) // Muted gray
	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("231")) // Light gray/white
)

type sidePanel struct {
	width  int
	height int
	ready  bool

	gpuInfos *overclocking.GPUInfo
	gpuState *overclocking.CurrentStateResult
}

func (lp sidePanel) Init() tea.Cmd {
	return nil
}

func (lp sidePanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		lp.width = msg.Width
		lp.height = msg.Height
		if !lp.ready {
			lp.ready = true
		}
		return lp, nil
	case commands.GPUItem:
		lp.gpuInfos = &msg.GPUInfo
		// Immediately get the current state
		return lp, commands.GetCurrentState(lp.gpuInfos.Index, lp.gpuInfos.ProfilePath)
	case commands.CurrentState:
		if msg.Error == nil {
			lp.gpuState = msg.State
		} else {
			lp.gpuState = nil
			// TODO, show error
		}
		// start/continue ticking for updates
		return lp, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return commands.GetCurrentStateRaw(lp.gpuInfos.Index, lp.gpuInfos.ProfilePath)
		})
	}
	return lp, nil
}

func (lp sidePanel) View() (v tea.View) {
	if !lp.ready {
		v.SetContent(panelStyle.Render("Loading..."))
		return
	}
	// Panel dynamic size
	sidePanelStyle := panelStyle.Width(lp.width).Height(lp.height)
	// Build panel content
	var content strings.Builder
	// Title
	var title string
	if lp.gpuInfos != nil && lp.gpuInfos.FullDescription != "" {
		title = lp.gpuInfos.FullDescription
	} else {
		title = sidePanelTitleDefault
	}
	content.WriteString(sidePanelTitleStyle.Render(sidePanelTitleIconPrefix + title))
	content.WriteString("\n\n")
	// Hardware Information Section
	if lp.gpuInfos != nil {
		// GPU Information Section (includes PCI IDs)
		content.WriteString(lp.renderGPUSection())
		// GPU State Section
		if lp.gpuState != nil {
			content.WriteString("\n")
			content.WriteString(lp.renderStateSection())
		}
	} else {
		content.WriteString(sectionLabelStyle.Render("No GPU selected."))
	}
	// Render panel
	v.SetContent(sidePanelStyle.Render(content.String()))
	return
}

func (lp sidePanel) renderGPUSection() string {
	width := lp.width - panelStyle.GetHorizontalFrameSize()
	var content strings.Builder
	content.WriteString(lp.formatDetail(
		"Vendor ID", lp.formatPCIVendor(lp.gpuInfos.VendorID), width,
	))
	content.WriteString(lp.formatDetail(
		"Device ID", lp.formatPCIDevice(lp.gpuInfos.VendorID, lp.gpuInfos.DeviceID), width,
	))
	content.WriteString(lp.formatDetail(
		"Subsystem ID", lp.formatPCISubsystem(lp.gpuInfos.SubsystemID), width,
	))
	content.WriteString(lp.formatDetail(
		"PCIe Address", fmt.Sprintf("%d:%d:%d",
			lp.gpuInfos.BusNumber,
			lp.gpuInfos.DeviceNumber,
			lp.gpuInfos.FunctionNumber,
		), width,
	))
	content.WriteString(lp.formatDetail(
		"NvAPI Index", fmt.Sprintf("%d", lp.gpuInfos.Index), width,
	))
	return content.String()
}

func (lp sidePanel) formatDetail(label, value string, width int) string {
	if value == "" {
		return ""
	}
	leftFillers := (width / 3) - lipgloss.Width(label)
	leftFillers = max(leftFillers, 0)
	var content strings.Builder
	content.Grow(width + 1)
	content.WriteString(strings.Repeat(" ", leftFillers))
	content.WriteString(detailLabelStyle.Render(label))
	content.WriteRune(nonBreakingSpace)
	content.WriteString(detailValueStyle.Render(value))
	content.WriteString("\n")
	return content.String()
}

func (lp sidePanel) formatPCIVendor(vendorID string) string {
	if vendorID == "" {
		return ""
	}
	// Lookup vendor name from catalog
	vendorName := catalog.LookupVendorName(vendorID)
	return fmt.Sprintf("%s (%s)", vendorID, vendorName)
}

func (lp sidePanel) formatPCIDevice(vendorID, deviceID string) string {
	if deviceID == "" {
		return ""
	}
	// Lookup GPU name from catalog
	gpuInfo := catalog.LookupGPU(vendorID, deviceID)
	if gpuInfo.IsKnown {
		return fmt.Sprintf("%s (%s)", deviceID, gpuInfo.GPUName)
	}
	return fmt.Sprintf("%s (Unknown)", deviceID)
}

func (lp sidePanel) formatPCISubsystem(subsystemID string) string {
	if subsystemID == "" {
		return ""
	}
	// Lookup manufacturer from subsystem ID
	manufacturer := catalog.LookupManufacturer(subsystemID)
	return fmt.Sprintf("%s (%s)", subsystemID, manufacturer)
}

func (lp sidePanel) renderStateSection() string {
	width := lp.width - panelStyle.GetHorizontalFrameSize()
	var content strings.Builder
	if lp.gpuState.LiveMatchesStartup {
		content.WriteString(lp.formatDetail(
			"MSI AF Sync", "✅", width,
		))
	} else {
		content.WriteString(lp.formatDetail(
			"MSI AF Sync", "❗", width,
		))
	}
	if lp.gpuState.Profile == nil {
		content.WriteString(lp.formatDetail(
			"Profile", "⚠️ startup profile only", width,
		))
	} else if lp.gpuState.Profile.Confidence == 1 {
		content.WriteString(lp.formatDetail(
			"Profile", fmt.Sprintf("#%d", lp.gpuState.Profile.SlotNumber), width,
		))
	} else {
		content.WriteString(lp.formatDetail(
			"Profile",
			fmt.Sprintf("%s (⚠️ %0.0f%% confidence)", lp.gpuState.Profile.SlotName, lp.gpuState.Profile.Confidence*100),
			width,
		))
	}
	content.WriteString(lp.formatDetail(
		"Power Limit", fmt.Sprintf("%d%%", lp.gpuState.PowerLimitPercent), width,
	))
	content.WriteString(
		lp.formatDetail(
			"Mem Clock Boost", fmt.Sprintf("+%d MHz", lp.gpuState.MemClkBoostMHz), width,
		),
	)
	return content.String()
}
