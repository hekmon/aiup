package models

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	AppTitle = "GPU Overclocking AI Assistant"
)

var (
	headerStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
	headerTitleStyle = func() lipgloss.Style {
		b := lipgloss.RoundedBorder()
		b.BottomRight = "┴"
		b.BottomLeft = "┴"
		return lipgloss.NewStyle().BorderForeground(lipgloss.Color("69")).BorderStyle(b).Bold(true).Padding(0, 1)
	}()
	titleHeader = headerTitleStyle.Render(AppTitle)
)

type headerModel struct {
	// Layout state
	ready                   bool
	headerLeft, headerRight int
}

func (h headerModel) Init() tea.Cmd {
	return nil
}

func (h headerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Computer header realated sizes
		headerFiller := msg.Width - lipgloss.Width(titleHeader)
		h.headerLeft = headerFiller / 2
		h.headerRight = headerFiller - h.headerLeft
		h.ready = true
		cmds = append(cmds, func() tea.Msg {
			return remainingSizeAfterHeader{
				Width:  msg.Width,
				Height: msg.Height - lipgloss.Height(titleHeader),
			}
		})
	}
	return h, tea.Batch(cmds...)
}

func (h headerModel) View() (v tea.View) {
	if !h.ready {
		v.SetContent(AppTitle)
		return
	}
	var left, right strings.Builder
	// build left
	left.Grow(h.headerLeft)
	left.WriteRune('╭')
	left.WriteString(strings.Repeat("─", h.headerLeft-1))
	// build right
	right.Grow(h.headerRight)
	right.WriteString(strings.Repeat("─", h.headerRight-1))
	right.WriteRune('╮')
	// join
	v.SetContent(lipgloss.JoinHorizontal(
		lipgloss.Bottom,
		headerStyle.Render(left.String()),
		titleHeader,
		headerStyle.Render(right.String()),
	))
	return
}

type remainingSizeAfterHeader struct {
	Width, Height int
}
