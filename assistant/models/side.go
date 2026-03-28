package models

import (
	"fmt"
	"strings"

	"github.com/hekmon/aiup/assistant/commands"
	"github.com/hekmon/aiup/overclocking"

	tea "charm.land/bubbletea/v2"
)

type sidePanel struct {
	width  int
	height int
	ready  bool

	gpuInfos *overclocking.GPUInfo
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
	case commands.GPUItem:
		lp.gpuInfos = &msg.GPUInfo
	}
	return lp, nil
}

func (lp sidePanel) View() (v tea.View) {
	if !lp.ready {
		v.SetContent(panelStyle.Render("Info panel loading..."))
		return
	}
	// Panel dynamic size
	sidePanelStyle := panelStyle.Width(lp.width).Height(lp.height)
	// Build panel content
	lines := []string{"📋 Info Panel"}
	lines = append(lines, "")
	if lp.gpuInfos != nil {
		// GPU name
		lines = append(lines, fmt.Sprintf("\t💻 GPU:          %s", lp.gpuInfos.Name))
		// Manufacturer
		lines = append(lines, fmt.Sprintf("\t🏭 Manufacturer: %s", lp.gpuInfos.Manufacturer))
		// PCIe address
		lines = append(lines, fmt.Sprintf("\t🔌 PCIe:         %d:%d:%d",
			lp.gpuInfos.BusNumber, lp.gpuInfos.DeviceNumber, lp.gpuInfos.FunctionNumber),
		)
	} else {
		lines = append(lines, "No GPU selected.")
	}
	// Render panel
	v.SetContent(sidePanelStyle.Render(strings.Join(lines, "\n")))
	return
}
