package models

import (
	"github.com/hekmon/aiup/assistant/commands"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
)

type gpuSelect struct {
	// Config
	profilesDir string
	// Widgets
	ready         bool
	width, height int
	gpusPanel     list.Model
}

func (g *gpuSelect) Init() tea.Cmd {
	var (
		// cmd  tea.Cmd
		cmds []tea.Cmd
	)
	g.gpusPanel = list.New(nil, list.NewDefaultDelegate(), 0, 0)
	g.gpusPanel.Title = "Select the GPU you want to work with"
	g.gpusPanel.SetStatusBarItemName("GPU", "GPUs")
	// g.gpusPanel.Styles.
	cmds = append(cmds,
		g.gpusPanel.StartSpinner(),
		commands.GPUDiscovery(g.profilesDir),
	)
	return tea.Batch(cmds...)
}

func (g *gpuSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		// TODO
	case tea.WindowSizeMsg:
		// Update preselection
		g.height, g.width = msg.Height, msg.Width
		w, h := panelStyle.GetFrameSize()
		g.gpusPanel.SetSize(msg.Width-w, msg.Height-h)
		if !g.ready {
			g.ready = true
		}
		return g, nil
	case commands.DiscoveryResult:
		cmds = append(cmds, g.gpusPanel.SetItems(msg.GPUs))
		g.gpusPanel.StopSpinner()
		// TODO handle warning and error
		return g, tea.Batch(cmds...)
	case tea.KeyPressMsg:
		if g.ready && len(g.gpusPanel.Items()) > 0 && msg.String() == "enter" {
			return g, func() tea.Msg {
				return g.gpusPanel.SelectedItem().(commands.GPUItem)
			}
		}
	}
	// Standard update, let it flow
	g.gpusPanel, cmd = g.gpusPanel.Update(msg)
	cmds = append(cmds, cmd)
	return g, tea.Batch(cmds...)
}

func (g *gpuSelect) View() (v tea.View) {
	if !g.ready {
		v.SetContent("\n  Initializing...")
		return
	}
	gpuSelectStyle := panelStyle.Width(g.width).Height(g.height)
	v.SetContent(
		gpuSelectStyle.Render(
			g.gpusPanel.View(),
		),
	)
	return
}
