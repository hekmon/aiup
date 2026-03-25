package main

import (
	"fmt"
	"os"

	"github.com/hekmon/aiup/cmd/agent/models"

	tea "charm.land/bubbletea/v2"
)

func main() {
	p := tea.NewProgram(models.NewMainModel())
	if _, err := p.Run(); err != nil {
		fmt.Println("could not run program:", err)
		os.Exit(1)
	}
}
