package models

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/muesli/reflow/wordwrap"
)

type infoType uint8

const (
	infoTypeInfo infoType = iota
	infoTypeWarning
	infoTypeError
)

var (
	infoBaseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			Padding(0, 1)

	infoColor       = lipgloss.Color("75") // Blue
	infoWidgetStyle = infoBaseStyle.BorderForeground(infoColor)
	infoTitleStyle  = lipgloss.NewStyle().
			Foreground(infoColor).
			Bold(true)
	infoTitle = infoTitleStyle.Render("ℹ️  Information")

	warningColor       = lipgloss.Color("214") // Orange
	warningWidgetStyle = infoBaseStyle.BorderForeground(warningColor)
	warningTitleStyle  = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)
	warningTitle = warningTitleStyle.Render("⚠  Warning")

	errorColor       = lipgloss.Color("196") // Red
	errorWidgetStyle = infoBaseStyle.BorderForeground(errorColor)
	errorTitleStyle  = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)
	errorTitle = errorTitleStyle.Render("❌  Error")
)

type info struct {
	// config
	PanelType infoType
	Message   string
	// state
	title string
	style lipgloss.Style
	width int
}

func (i *info) Init() tea.Cmd {
	switch i.PanelType {
	case infoTypeError:
		i.title = errorTitle
		i.style = errorWidgetStyle
	case infoTypeWarning:
		i.title = warningTitle
		i.style = warningWidgetStyle
	case infoTypeInfo:
		fallthrough
	default:
		i.title = infoTitle
		i.style = infoWidgetStyle
	}
	return nil
}

func (i *info) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// var (
	// 	cmd  tea.Cmd
	// 	cmds []tea.Cmd
	// )
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		// TODO
	case tea.WindowSizeMsg:
		frameSizeHorizontal := i.style.GetHorizontalFrameSize()
		i.width = msg.Width - frameSizeHorizontal
	}
	return i, nil
}

func (i *info) View() (v tea.View) {
	var title, message string
	if i.width > 0 {
		title = wordwrap.String(i.title, i.width)
		message = wordwrap.String(i.Message, i.width)
	} else {
		title = i.title
		message = i.Message
	}
	v.SetContent(
		i.style.Render(
			title + "\n\n" + message,
		),
	)
	return
}
