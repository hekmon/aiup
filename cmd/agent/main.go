package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// mainView is the root model that composes chatPanel and infoPanel
type mainView struct {
	// Layout state
	ready bool
	// Sub-panels
	chatPanel chatPanel
	infoPanel infoPanel
}

func (m mainView) Init() tea.Cmd {
	// Initialize sub-panels
	var cmds []tea.Cmd

	if cmd := m.chatPanel.Init(); cmd != nil {
		cmds = append(cmds, cmd)
	}
	if cmd := m.infoPanel.Init(); cmd != nil {
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

func (m mainView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		model tea.Model
		cmd   tea.Cmd
		cmds  []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" || k == "esc" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		infoPanelWidth := msg.Width / 3
		// Update chat panel
		model, cmd = m.chatPanel.Update(tea.WindowSizeMsg{
			Width:  msg.Width - infoPanelWidth,
			Height: msg.Height,
		})
		m.chatPanel = model.(chatPanel)
		cmds = append(cmds, cmd)
		// Update info panel
		model, cmd = m.infoPanel.Update(tea.WindowSizeMsg{
			Width:  infoPanelWidth,
			Height: msg.Height,
		})
		m.infoPanel = model.(infoPanel)
		cmds = append(cmds, cmd)
		// If we did not had size yet, we are now ready
		if !m.ready {
			m.ready = true
		}
		// For WindowSizeMsg we adapted ourself the message to the subpanels:
		// skip auto forwarding to subpanels
		return m, tea.Batch(cmds...)
	}

	// Route non specific messages to sub-panels
	// Each panel can act on the message and return a new model and a command

	model, cmd = m.chatPanel.Update(msg)
	if cp, ok := model.(chatPanel); ok {
		m.chatPanel = cp
	}
	cmds = append(cmds, cmd)

	model, cmd = m.infoPanel.Update(msg)
	if lp, ok := model.(infoPanel); ok {
		m.infoPanel = lp
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m mainView) View() (v tea.View) {
	if !m.ready {
		v.SetContent("\n  Initializing...")
		return
	}
	v.SetContent(
		lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.chatPanel.View().Content,
			m.infoPanel.View().Content,
		),
	)
	v.WindowTitle = "MSI Afterburner AI Assistant"
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return
}

func main() {
	p := tea.NewProgram(mainView{})
	if _, err := p.Run(); err != nil {
		fmt.Println("could not run program:", err)
		os.Exit(1)
	}
}
