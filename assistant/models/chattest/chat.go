package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/glamour/v2/ansi"
	"charm.land/glamour/v2/styles"
	"charm.land/lipgloss/v2"
)

const assistant = `# Today’s Menu

## Appetizers

| Name        | Price | Notes                           |
| ---         | ---   | ---                             |
| Tsukemono   | $2    | Just an appetizer               |
| Tomato Soup | $4    | Made with San Marzano tomatoes  |
| Okonomiyaki | $4    | Takes a few minutes to make     |
| Curry       | $3    | We can add squash if you’d like |

## Seasonal Dishes

| Name                 | Price | Notes              |
| ---                  | ---   | ---                |
| Steamed bitter melon | $2    | Not so bitter      |
| Takoyaki             | $3    | Fun to eat         |
| Winter squash        | $3    | Today it's pumpkin |

## Desserts

| Name         | Price | Notes                 |
| ---          | ---   | ---                   |
| Dorayaki     | $4    | Looks good on rabbits |
| Banana Split | $5    | A classic             |
| Cream Puff   | $3    | Pretty creamy!        |

All our dishes are made in-house by Karen, our chef. Most of our ingredients are from our garden or the fish market down the street.

Some famous people that have eaten here lately:

* [x] René Redzepi
* [x] David Chang
* [ ] Jiro Ono (maybe some day)

Bon appétit!
`

type msgSource uint8

const (
	msgSourceEmpty msgSource = iota
	msgSourceUser
	msgSourceAssistant
	msgSourceThinking
	msgSourceTool
)

func (ms msgSource) String() string {
	switch ms {
	case msgSourceUser:
		return "👤 "
	case msgSourceAssistant:
		return "💬 "
	case msgSourceThinking:
		return "🧠 "
	case msgSourceTool:
		return "🔨 "
	default:
		return "   "
	}
}

type msg struct {
	Source  msgSource
	Content string
}

var conversation = []msg{
	{
		Source:  msgSourceUser,
		Content: "What is today's menu?",
	},
	{
		Source:  msgSourceThinking,
		Content: "Ok so the user want the menu, let's retreive it using the provided tool",
	},
	{
		Source:  msgSourceTool,
		Content: "Calling get_menu",
	},
	{
		Source:  msgSourceAssistant,
		Content: assistant,
	},
	{
		Source:  msgSourceUser,
		Content: "Thank you!\nHave a good day!",
	},
}

const (
	chatPanelTitle = "✨ Assistant Chat"
)

var (
	chatPanelViewPortStyle = lipgloss.NewStyle().Border(lipgloss.HiddenBorder())
)

type chatPanel struct {
	width  int
	height int
	ready  bool
	isDark bool

	viewport     viewport.Model
	glamourStyle ansi.StyleConfig

	messages []string
}

func (cp chatPanel) Init() tea.Cmd {
	return nil
}

func (cp chatPanel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		cp.isDark = msg.IsDark()
		if cp.isDark {
			cp.glamourStyle = styles.DarkStyleConfig
		} else {
			cp.glamourStyle = styles.LightStyleConfig
		}
		content, gutters, err := cp.renderConversation()
		if err != nil {
			panic(err)
		}
		// TODO handle error
		cp.viewport.LeftGutterFunc = func(info viewport.GutterContext) string {
			return gutters[info.Index]
		}
		cp.viewport.SetContent(content)
		cp.viewport.GotoBottom()
		return cp, nil
	case tea.KeyPressMsg:
		if k := msg.String(); k == "ctrl+c" || k == "q" {
			return cp, tea.Quit
		}
	case tea.WindowSizeMsg:
		cp.width = msg.Width
		cp.height = msg.Height
		innerWidth := cp.width - panelStyle.GetHorizontalFrameSize()
		innerHeight := cp.height - panelStyle.GetVerticalFrameSize()
		if !cp.ready {
			// Setup the viewport
			cp.viewport = viewport.New()
			cp.viewport.Style = chatPanelViewPortStyle
			cp.viewport.LeftGutterFunc = func(info viewport.GutterContext) string {
				return msgSourceEmpty.String()
			}
			// Done
			cp.ready = true
		}
		cp.viewport.SetWidth(innerWidth)
		cp.viewport.SetHeight(innerHeight - lipgloss.Height(chatPanelTitle))
		content, gutters, err := cp.renderConversation()
		if err != nil {
			panic(err)
		}
		// TODO handle error
		cp.viewport.LeftGutterFunc = func(info viewport.GutterContext) string {
			return gutters[info.Index]
		}
		cp.viewport.SetContent(content)
		cp.viewport.GotoBottom()
		return cp, nil
	}
	cp.viewport, cmd = cp.viewport.Update(msg)
	return cp, cmd
}

func (cp chatPanel) renderConversation() (full string, gutters []string, err error) {
	// Prepare the renderer
	innerWidth := cp.width - panelStyle.GetHorizontalFrameSize() - chatPanelViewPortStyle.GetHorizontalFrameSize() - (len(msgSourceEmpty.String()) + 1)
	var glamourStyle ansi.StyleConfig
	if cp.isDark {
		glamourStyle = styles.DarkStyleConfig
	} else {
		glamourStyle = styles.LightStyleConfig
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(glamourStyle),
		glamour.WithWordWrap(innerWidth),
	)
	if err != nil {
		err = fmt.Errorf("failed to create a glamour terminal rendered: %w", err)
		return
	}
	// Use it to render the conversation
	var (
		rendered string
		builder  strings.Builder
	)
	for index, message := range conversation {
		// Add spacing between messages
		if index > 0 {
			builder.WriteString("\n")
			newLength := nbLines(builder.String())
			for i := len(gutters); i < newLength; i++ {
				gutters = append(gutters, msgSourceEmpty.String()+" ")
			}
		}
		// Render the message content
		if rendered, err = renderer.Render(message.Content); err != nil {
			err = fmt.Errorf("failed to render conversation message #%d with glamour renderer: %w", index, err)
			return
		}
		// Trim newlines to have clean content
		rendered = strings.Trim(rendered, "\n")
		if rendered == "" {
			continue
		}
		// Record where this message starts
		iconLineIndex := len(gutters)
		// Add rendered content with a trailing newline
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(rendered)
		// Extend gutters to match the new line count
		newLineCount := nbLines(builder.String())
		for i := len(gutters); i < newLineCount; i++ {
			gutters = append(gutters, msgSourceEmpty.String()+"│")
		}
		// Set the icon at the first line of this message
		if iconLineIndex < len(gutters) {
			gutters[iconLineIndex] = message.Source.String() + "│"
		}
	}
	full = builder.String()
	return
}

func (cp chatPanel) View() (v tea.View) {
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	if !cp.ready {
		v.SetContent(panelStyle.Render("Chat panel loading..."))
		return
	}
	// Build scroll indicators as a corner border
	scrollIndicatorBorder, _, _, _, _ := cp.viewport.Style.GetBorder()
	if !cp.viewport.AtTop() {
		scrollIndicatorBorder.TopRight = "↑"
	}
	if !cp.viewport.AtBottom() {
		scrollIndicatorBorder.BottomRight = "↓"
	}
	cp.viewport.Style = cp.viewport.Style.Border(scrollIndicatorBorder)
	// Panel dynamic size
	chatStyle := panelStyle.Width(cp.width).Height(cp.height)
	// Render
	v.SetContent(chatStyle.Render(chatPanelTitle + "\n" + cp.viewport.View()))
	return
}

func nbLines(text string) (nbLines int) {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}
