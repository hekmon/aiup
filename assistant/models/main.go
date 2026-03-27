package models

import (
	"github.com/hekmon/aiup/overclocking"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	appTitle = "GPU OC AI Assistant"
)

var (
	panelStyle = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2)
)

type main struct {
	// Layout state
	ready bool
	// Config
	selectedGPU *overclocking.GPUInfo
	// Sub-panels
	gpusPanel tea.Model
	chatPanel tea.Model
	infoPanel tea.Model
}

func (m main) Init() tea.Cmd {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	cmds = append(cmds,
		tea.RequestBackgroundColor,
	)
	if cmd = m.gpusPanel.Init(); cmd != nil {
		cmds = append(cmds, cmd)
	}
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
	case tea.BackgroundColorMsg:
		// TODO
	case tea.KeyPressMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		// Update wizard view
		if m.gpusPanel != nil {
			m.gpusPanel, cmd = m.gpusPanel.Update(msg)
			cmds = append(cmds, cmd)
		}
		// Update main viewports
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
	//// GPU selection
	if m.gpusPanel != nil {
		m.gpusPanel, cmd = m.gpusPanel.Update(msg)
		cmds = append(cmds, cmd)
	}
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
	v.WindowTitle = appTitle
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	if m.gpusPanel != nil {
		v.SetContent(m.gpusPanel.View().Content)
	} else {
		v.SetContent(
			lipgloss.JoinHorizontal(
				lipgloss.Top,
				m.chatPanel.View().Content,
				m.infoPanel.View().Content,
			),
		)
	}
	return
}
