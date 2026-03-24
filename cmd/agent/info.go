package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	infoPanelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(1, 2)
)

type infoPanel struct {
	width  int
	height int
	ready  bool

	// Left panel-specific state
	title    string
	items    []string
	selected int
}

func (lp infoPanel) Init() tea.Cmd {
	return nil
}

func (lp infoPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		lp.width = msg.Width
		lp.height = msg.Height
		lp.ready = true
	}
	return lp, nil
}

func (lp infoPanel) View() (v tea.View) {
	if !lp.ready {
		v.SetContent("Left panel loading...")
		return
	}

	// Left panel border and styling
	panelStyle := infoPanelStyle.Width(lp.width).Height(lp.height)

	// Build panel content
	lines := []string{"📋 Left Panel"}
	if len(lp.items) > 0 {
		lines = append(lines, "")
		for i, item := range lp.items {
			prefix := "  "
			if i == lp.selected {
				prefix = "> "
			}
			lines = append(lines, fmt.Sprintf("%s%s", prefix, item))
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, "No items to display")
	}

	v.SetContent(panelStyle.Render(strings.Join(lines, "\n")))
	return
}
