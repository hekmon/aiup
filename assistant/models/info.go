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
	infoTitleIconPrefix = "ℹ️  "
	infoTitleDefault    = "Information"

	warningColor       = lipgloss.Color("214") // Orange
	warningWidgetStyle = infoBaseStyle.BorderForeground(warningColor)
	warningTitleStyle  = lipgloss.NewStyle().
				Foreground(warningColor).
				Bold(true)
	warningTitleIconPrefix = "⚠  "
	warningTitleDefault    = "Warning"

	errorColor       = lipgloss.Color("196") // Red
	errorWidgetStyle = infoBaseStyle.BorderForeground(errorColor)
	errorTitleStyle  = lipgloss.NewStyle().
				Foreground(errorColor).
				Bold(true)
	errorTitleIconPrefix = "❌  "
	errorTitleDefault    = "Error"
)

type info struct {
	// config
	PanelType infoType
	Title     string
	Message   string
	// state
	width int
}

func (i *info) Init() tea.Cmd {
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
		i.width = msg.Width
	}
	return i, nil
}

func (i *info) View() (v tea.View) {
	// Select style
	var panelStyle, titleStyle lipgloss.Style
	switch i.PanelType {
	case infoTypeError:
		panelStyle = errorWidgetStyle
		titleStyle = errorTitleStyle
	case infoTypeWarning:
		panelStyle = warningWidgetStyle
		titleStyle = warningTitleStyle
	case infoTypeInfo:
		fallthrough
	default:
		panelStyle = infoWidgetStyle
		titleStyle = infoTitleStyle
	}
	// Prepare title
	var title, styledTitle string
	switch i.PanelType {
	case infoTypeError:
		title = errorTitleIconPrefix
	case infoTypeWarning:
		title = warningTitleIconPrefix
	case infoTypeInfo:
		fallthrough
	default:
		title = infoTitleIconPrefix
	}
	if i.Title != "" {
		title += i.Title
	} else {
		switch i.PanelType {
		case infoTypeError:
			title += errorTitleDefault
		case infoTypeWarning:
			title += warningTitleDefault
		case infoTypeInfo:
			fallthrough
		default:
			title += infoTitleDefault
		}
	}
	styledTitle = titleStyle.Render(title)
	// Adapt wraping and set message
	maxWidth := i.width - panelStyle.GetHorizontalFrameSize()
	var message string
	if i.width > 0 {
		title = wordwrap.String(styledTitle, maxWidth)
		message = wordwrap.String(i.Message, maxWidth)
	} else {
		title = styledTitle
		message = i.Message
	}
	// Render
	v.SetContent(
		panelStyle.Render(
			title + "\n\n" + message,
		),
	)
	return
}
