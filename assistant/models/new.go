package models

import (
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

func NewMainModel(msiProfilesDir string) tea.Model {
	return main{
		profilesDir: msiProfilesDir,
		gpusPanel:   list.New(nil, list.NewDefaultDelegate(), 0, 0),
		chatPanel:   chatPanel{},
		infoPanel:   infoPanel{},
	}
}
