package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	panelBorderColor = lipgloss.Color("69")
	panelStyle       = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(panelBorderColor).
				Padding(1, 3)
)

func main() {
	p := tea.NewProgram(chatPanel{})
	if _, err := p.Run(); err != nil {
		fmt.Println("could not run program:", err)
		os.Exit(1)
	}
}
