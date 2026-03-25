package models

import tea "charm.land/bubbletea/v2"

func NewMainModel() tea.Model {
	return main{
		chatPanel: chatPanel{},
		infoPanel: infoPanel{},
	}
}
