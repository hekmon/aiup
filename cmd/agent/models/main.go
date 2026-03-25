package models

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	appTitle = "GPU Overclocking AI Assistant"
)

func NewMainModel() tea.Model {
	return main{
		headerPanel: headerModel{},
		chatPanel:   chatPanel{},
		infoPanel:   infoPanel{},
	}
}

// main is the root model that composes chatPanel and infoPanel
type main struct {
	// Layout state
	ready bool
	// Sub-panels
	headerPanel tea.Model
	chatPanel   tea.Model
	infoPanel   tea.Model
}

func (m main) Init() tea.Cmd {
	// Initialize sub-panels
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	if cmd = m.chatPanel.Init(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if cmd = m.infoPanel.Init(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

func (m main) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	// Catch specific messages we handle a certain way
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.headerPanel, cmd = m.headerPanel.Update(msg)
		return m, cmd
	case remainingSizeAfterHeader:
		// Update sub pannels sizes
		infoPanelWidth := msg.Width / 3
		m.chatPanel, cmd = m.chatPanel.Update(tea.WindowSizeMsg{
			Width:  msg.Width - infoPanelWidth,
			Height: msg.Height,
		})
		cmds = append(cmds, cmd)
		m.infoPanel, cmd = m.infoPanel.Update(tea.WindowSizeMsg{
			Width:  infoPanelWidth,
			Height: msg.Height,
		})
		cmds = append(cmds, cmd)
		// If we did not had size yet, we are now ready
		if !m.ready {
			m.ready = true
		}
		return m, tea.Batch(cmds...)
	}
	// Route non specific messages to sub-panels
	//// Chat panel
	m.chatPanel, cmd = m.chatPanel.Update(msg)
	cmds = append(cmds, cmd)
	//// Info panel
	m.infoPanel, cmd = m.infoPanel.Update(msg)
	cmds = append(cmds, cmd)
	// Return to the application updated model and commands to execute
	return m, tea.Batch(cmds...)
}

func (m main) View() (v tea.View) {
	if !m.ready {
		v.SetContent("\n  Initializing...")
		return
	}
	v.SetContent(
		lipgloss.JoinVertical(
			lipgloss.Center,
			m.headerPanel.View().Content,
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				m.chatPanel.View().Content,
				m.infoPanel.View().Content,
			),
		),
	)
	v.WindowTitle = appTitle
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return
}
