package models

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

type infoPanel struct {
	width  int
	height int
	ready  bool

	items []string
}

func (lp infoPanel) Init() tea.Cmd {
	return nil
}

func (lp infoPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		lp.width = msg.Width
		lp.height = msg.Height
		if !lp.ready {
			lp.ready = true
		}
	}
	return lp, nil
}

func (lp infoPanel) View() (v tea.View) {
	if !lp.ready {
		v.SetContent(panelStyle.Render("Info panel loading..."))
		return
	}
	// Panel dynamic size
	infoPanelStyle := panelStyle.Width(lp.width).Height(lp.height)
	// Build panel content
	lines := []string{"📋 Info Panel"}
	lines = append(lines, "")
	lines = append(lines, "No items to display")
	// Render panel
	v.SetContent(infoPanelStyle.Render(strings.Join(lines, "\n")))
	return
}
