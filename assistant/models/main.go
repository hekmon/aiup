package models

import (
	"github.com/hekmon/aiup/assistant/commands"
	"github.com/hekmon/aiup/overclocking"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	appTitle = "GPU OC AI Assistant"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2).Border(lipgloss.BlockBorder())

type main struct {
	// Layout state
	ready bool
	// Config
	profilesDir string
	selectedGPU *overclocking.GPUInfo
	// Sub-panels
	gpusPanel list.Model
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
		commands.GPUDiscovery(m.profilesDir),
		m.gpusPanel.StartSpinner(),
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
	case tea.BackgroundColorMsg:
		// TODO
	case tea.KeyPressMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		// Update preselection
		h, v := docStyle.GetFrameSize()
		m.gpusPanel.SetSize(msg.Width-h, msg.Height-v)
		// m.gpusPanel.SetSize(msg.Width, msg.Height)
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
	case commands.DiscoveryResult:
		m.gpusPanel.Title = "Select a GPU"
		m.gpusPanel.SetStatusBarItemName("GPU", "GPUs")
		cmd = m.gpusPanel.SetItems(msg.GPUs)
		cmds = append(cmds, cmd)
		// m.gpusPanel.StopSpinner()
		return m, tea.Batch(cmds...)
	}
	// Route non specific messages to sub-panels
	//// GPU selection
	m.gpusPanel, cmd = m.gpusPanel.Update(msg)
	cmds = append(cmds, cmd)
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
	if m.selectedGPU == nil {
		v.SetContent(docStyle.Render(m.gpusPanel.View()))
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
