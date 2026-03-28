package models

import (
	"strings"

	"github.com/hekmon/aiup/assistant/commands"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	warningBottomPadding = 1
)

type gpuSelect struct {
	// Config
	profilesDir string
	// State
	ready         bool
	width, height int
	// Widgets
	warningPanel *info
	errorPanel   *info
	gpusPanel    list.Model
}

func (g *gpuSelect) Init() tea.Cmd {
	var (
		// cmd  tea.Cmd
		cmds []tea.Cmd
	)
	g.gpusPanel = list.New(nil, list.NewDefaultDelegate(), 0, 0)
	g.gpusPanel.Title = "Select the GPU you want to work with"
	g.gpusPanel.SetStatusBarItemName("GPU", "GPUs")
	cmds = append(cmds,
		g.gpusPanel.StartSpinner(),
		commands.GPUDiscovery(g.profilesDir),
	)
	return tea.Batch(cmds...)
}

func (g *gpuSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		model tea.Model
		cmd   tea.Cmd
		cmds  []tea.Cmd
	)
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		// TODO
	case tea.WindowSizeMsg:
		g.width, g.height = msg.Width, msg.Height
		horizontalFrameSize, verticalFrameSize := panelStyle.GetFrameSize()
		innerWidth, innerHeight := g.width-horizontalFrameSize, g.height-verticalFrameSize
		if g.warningPanel != nil {
			// Send the new inner size to the warning panel
			model, cmd = g.warningPanel.Update(tea.WindowSizeMsg{
				Width:  innerWidth,
				Height: innerHeight, // will be ignored, but let's be consistent
			})
			g.warningPanel = model.(*info)
			cmds = append(cmds, cmd)
			// Update the GPU panel size given the current warning panel height
			g.gpusPanel.SetSize(
				innerWidth,
				innerHeight-lipgloss.Height(g.warningPanel.View().Content)-warningBottomPadding,
			)
		} else {
			g.gpusPanel.SetSize(innerWidth, innerHeight)
		}
		if !g.ready {
			g.ready = true
		}
		return g, tea.Batch(cmds...)
	case commands.DiscoveryResult:
		g.gpusPanel.StopSpinner()
		// If only one GPU and no warning, and no error, go
		if len(msg.GPUs) == 1 && len(msg.Warnings) == 0 && msg.Error == nil {
			return g, func() tea.Msg {
				return msg.GPUs[0].(commands.GPUItem)
			}
		}
		// Set the GPU list
		if len(msg.GPUs) > 0 {
			cmds = append(cmds, g.gpusPanel.SetItems(msg.GPUs))
		}
		// Prepare for widget size changing
		horizontalFrameSize, verticalFrameSize := panelStyle.GetFrameSize()
		innerWidth, innerHeight := g.width-horizontalFrameSize, g.height-verticalFrameSize
		// Warnings
		if len(msg.Warnings) > 0 {
			// Create the warning panel
			g.warningPanel = &info{
				PanelType: infoTypeWarning,
				Message:   formatListDotted(msg.Warnings, 0),
			}
			// Update its size
			model, cmd = g.warningPanel.Update(tea.WindowSizeMsg{
				Width:  innerWidth,
				Height: innerHeight, // will be ignored, but let's be consistent
			})
			g.warningPanel = model.(*info)
			cmds = append(cmds, cmd)
			// Update the GPU panel size given the current warning panel height
			g.gpusPanel.SetSize(
				innerWidth,
				innerHeight-lipgloss.Height(g.warningPanel.View().Content)-warningBottomPadding,
			)
		}
		// Errors
		if msg.Error != nil {
			g.errorPanel = &info{
				PanelType: infoTypeError,
				Title:     "GPU discovery failed",
				Message:   msg.Error.Error(),
			}
			// Update its size
			model, cmd = g.errorPanel.Update(tea.WindowSizeMsg{
				Width:  innerWidth,
				Height: innerHeight, // will be ignored, but let's be consistent
			})
			g.errorPanel = model.(*info)
			cmds = append(cmds, cmd)
		}
		return g, tea.Batch(cmds...)
	case tea.KeyPressMsg:
		if msg.String() == "enter" {
			if g.ready && g.errorPanel != nil && len(g.gpusPanel.Items()) > 0 {
				return g, func() tea.Msg {
					return g.gpusPanel.SelectedItem().(commands.GPUItem)
				}
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
	viewStyle := panelStyle.Width(g.width).Height(g.height)
	// Main view
	var content strings.Builder
	if g.warningPanel != nil {
		content.WriteString(g.warningPanel.View().Content)
		content.WriteString("\n") // add a newline after the warning
		content.WriteString(strings.Repeat("\n", warningBottomPadding))
	}
	content.WriteString(g.gpusPanel.View())
	mainView := viewStyle.Render(content.String())
	if g.errorPanel == nil {
		v.SetContent(mainView)
		return
	}
	// Handle error popup
	popupView := g.errorPanel.View().Content
	popupX := (g.width - lipgloss.Width(popupView)) / 2
	popupY := (g.height - lipgloss.Height(popupView)) / 2
	v.SetContent(lipgloss.NewCompositor(
		lipgloss.NewLayer(mainView),
		lipgloss.NewLayer(popupView).X(popupX).Y(popupY),
	).Render())
	return
}
