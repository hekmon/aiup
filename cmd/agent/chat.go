package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	chatStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2)
)

type chatPanel struct {
	width  int
	height int
	ready  bool

	// Chat-specific state
	messages []string
}

func (cp chatPanel) Init() tea.Cmd {
	return nil
}

func (cp chatPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cp.width = msg.Width
		cp.height = msg.Height
		cp.ready = true
	}
	return cp, nil
}

func (cp chatPanel) View() (v tea.View) {
	if !cp.ready {
		v.SetContent("Chat panel loading...")
		return
	}

	// Chat panel border and styling
	chatStyle := chatStyle.Width(cp.width).Height(cp.height)

	// Build chat content
	lines := []string{"💬 Chat Panel"}
	if len(cp.messages) > 0 {
		lines = append(lines, "")
		lines = append(lines, cp.messages...)
	} else {
		lines = append(lines, "")
		lines = append(lines, "No messages yet...")
	}

	v.SetContent(chatStyle.Render(strings.Join(lines, "\n")))
	return
}
