package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hekmon/aiup/assistant/models"

	tea "charm.land/bubbletea/v2"
)

func main() {
	msiProfilesDir := flag.String(
		"profilesDir", `C:\Program Files (x86)\MSI Afterburner\Profiles`,
		"Path to MSI Afterburner profiles directory",
	)
	flag.Parse()
	p := tea.NewProgram(models.NewMainModel(*msiProfilesDir))
	if _, err := p.Run(); err != nil {
		fmt.Println("could not run program:", err)
		os.Exit(1)
	}
}
