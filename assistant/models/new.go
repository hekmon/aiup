package models

import (
	tea "charm.land/bubbletea/v2"
)

func NewMainModel(msiProfilesDir string) tea.Model {
	return main{
		gpusPanel: &gpuSelect{
			profilesDir: msiProfilesDir,
		},
		chatPanel: chatPanel{},
		sidePanel: sidePanel{},
	}
}
