package main

import (
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour/v2"
	"github.com/charmbracelet/glamour/v2/styles"
)

// keyMap defines the keybindings for the application
type keyMap struct {
	Send      key.Binding
	Quit      key.Binding
	Help      key.Binding
	Clear     key.Binding
	SelectAll key.Binding
	Cut       key.Binding
	Copy      key.Binding
	Paste     key.Binding
}

// ShortHelp returns keybindings for the minimal help view
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Send, k.Quit, k.Help}
}

// FullHelp returns keybindings for the expanded help view
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Send, k.Clear, k.SelectAll, k.Cut, k.Copy, k.Paste},
		{k.Quit, k.Help},
	}
}

var keys = keyMap{
	Send: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "send message"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "esc"),
		key.WithHelp("ctrl+c", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
	Clear: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "clear chat"),
	),
	SelectAll: key.NewBinding(
		key.WithKeys("ctrl+a"),
		key.WithHelp("ctrl+a", "select all"),
	),
	Cut: key.NewBinding(
		key.WithKeys("ctrl+x"),
		key.WithHelp("ctrl+x", "cut"),
	),
	Copy: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "copy"),
	),
	Paste: key.NewBinding(
		key.WithKeys("ctrl+v"),
		key.WithHelp("ctrl+v", "paste"),
	),
}

// Message represents a single chat message
type Message struct {
	Content  string
	IsUser   bool
	Rendered string // Pre-rendered markdown content
}

// model is the main application model
type model struct {
	viewport    viewport.Model
	messages    []Message
	textarea    textarea.Model
	help        help.Model
	renderer    *glamour.TermRenderer
	ready       bool
	showHelp    bool
	err         error
	senderStyle lipgloss.Style
	agentStyle  lipgloss.Style
	width       int
	height      int
}

// newModel initializes the application model
func newModel(isDark bool) model {
	// Initialize textarea for user input
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.SetVirtualCursor(false)
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 1000
	ta.SetWidth(60)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	// Remove cursor line styling
	s := ta.Styles()
	s.Focused.CursorLine = lipgloss.NewStyle()
	ta.SetStyles(s)

	// Initialize viewport for chat history
	vp := viewport.New()
	vp.SetWidth(60)
	vp.SetHeight(10)
	vp.KeyMap.Left.SetEnabled(false)
	vp.KeyMap.Right.SetEnabled(false)

	// Initialize markdown renderer
	style := styles.DarkStyleConfig
	if !isDark {
		style = styles.LightStyleConfig
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(56), // Account for viewport padding
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create markdown renderer: %v\n", err)
		os.Exit(1)
	}

	// Add initial welcome message
	welcomeMsg := Message{
		Content:  "## Welcome to Agent Chat!\n\nI'm your AI assistant. Ask me anything!\n\n**Commands:**\n- Press `Enter` to send a message\n- Press `?` to toggle help\n- Press `Ctrl+L` to clear chat",
		IsUser:   false,
		Rendered: "", // Will be rendered on first display
	}

	return model{
		viewport:    vp,
		messages:    []Message{welcomeMsg},
		textarea:    ta,
		help:        help.New(),
		renderer:    renderer,
		senderStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("5")).Bold(true),
		agentStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true),
	}
}

// Init initializes the application
func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, tea.RequestBackgroundColor)
}

// Update handles messages and updates the model
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.BackgroundColorMsg:
		// Update styling based on background color
		m.textarea.SetStyles(textarea.DefaultStyles(msg.IsDark()))

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate dimensions
		helpHeight := 0
		if m.showHelp {
			helpHeight = lipgloss.Height(m.helpView())
		}

		// Set textarea width
		textareaWidth := max(40, msg.Width-4)
		textareaHeight := 3
		m.textarea.SetWidth(textareaWidth)
		m.textarea.SetHeight(textareaHeight)

		// Set viewport dimensions
		viewportWidth := msg.Width - 4
		viewportHeight := msg.Height - textareaHeight - helpHeight - 4
		if viewportHeight < 5 {
			viewportHeight = 5
		}
		m.viewport.SetWidth(viewportWidth)
		m.viewport.SetHeight(viewportHeight)

		// Render all messages with new width (including initial welcome message)
		m.renderMessages()

		m.ready = true

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			// Always quit on ctrl+c or esc
			return m, tea.Quit

		case "enter":
			// Send message
			userText := strings.TrimSpace(m.textarea.Value())
			if userText != "" {
				// Add user message
				m.messages = append(m.messages, Message{
					Content:  userText,
					IsUser:   true,
					Rendered: "",
				})

				// Generate echo response (placeholder for future LLM integration)
				agentResponse := fmt.Sprintf("echo back: %s", userText)

				// Add agent response
				m.messages = append(m.messages, Message{
					Content:  agentResponse,
					IsUser:   false,
					Rendered: "",
				})

				// Clear textarea
				m.textarea.Reset()

				// Re-render messages
				m.renderMessages()

				// Scroll to bottom
				m.viewport.GotoBottom()
			}
			return m, nil

		case "?":
			// Toggle help
			m.showHelp = !m.showHelp
			// Recalculate viewport height
			helpHeight := 0
			if m.showHelp {
				helpHeight = lipgloss.Height(m.helpView())
			}
			viewportHeight := m.height - 3 - helpHeight - 4
			if viewportHeight < 5 {
				viewportHeight = 5
			}
			m.viewport.SetHeight(viewportHeight)
			return m, nil

		case "ctrl+l":
			// Clear chat history (keep welcome message)
			if len(m.messages) > 1 {
				m.messages = m.messages[:1]
				m.renderMessages()
			}
			return m, nil
		}

		// Handle help keybindings when help is shown
		if m.showHelp {
			if key.Matches(msg, keys.Quit) {
				return m, tea.Quit
			}
			if key.Matches(msg, keys.Help) {
				m.showHelp = !m.showHelp
				return m, nil
			}
		}
	}

	// Pass messages to textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// renderMessages renders all messages with markdown
func (m *model) renderMessages() {
	var sb strings.Builder

	for i, msg := range m.messages {
		// Render message content if not already rendered
		if msg.Rendered == "" {
			if msg.IsUser {
				// User messages are shown as plain text
				msg.Rendered = m.senderStyle.Render("You: ") + msg.Content
			} else {
				// Agent messages are rendered as markdown
				rendered, err := m.renderer.Render(msg.Content)
				if err != nil {
					msg.Rendered = m.agentStyle.Render("Agent: ") + msg.Content
				} else {
					// Glamour already includes newlines, so we just prepend the label
					msg.Rendered = m.agentStyle.Render("Agent:") + "\n" + rendered
				}
			}
			m.messages[i] = msg
		}

		sb.WriteString(msg.Rendered)
		if i < len(m.messages)-1 {
			sb.WriteString("\n\n")
		}
	}

	// Wrap content to viewport width
	content := lipgloss.NewStyle().Width(m.viewport.Width()).Render(sb.String())
	m.viewport.SetContent(content)
}

// View renders the application
func (m model) View() tea.View {
	if !m.ready {
		return tea.NewView("Initializing...")
	}

	var b strings.Builder

	// Header
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("╔═══ Agent Chat ═══╗") + "\n\n")

	// Viewport (chat history)
	b.WriteString(m.viewport.View() + "\n")

	// Textarea (user input)
	b.WriteString(m.textarea.View() + "\n")

	// Help panel (if shown)
	if m.showHelp {
		b.WriteString("\n" + m.helpView())
	}

	v := tea.NewView(b.String())

	// Set cursor position - calculate based on header + viewport + spacing
	if !m.textarea.VirtualCursor() {
		c := m.textarea.Cursor()
		if c != nil {
			// Header (3 lines: "╔═══ Agent Chat ═══╗" + 2 newlines) + viewport content height + 1 newline
			headerHeight := 3
			viewportContentHeight := lipgloss.Height(m.viewport.View())
			c.Y = headerHeight + viewportContentHeight + 1
		}
		v.Cursor = c
	}

	v.AltScreen = true
	return v
}

// helpView renders the help panel
func (m model) helpView() string {
	if !m.showHelp {
		return ""
	}
	return m.help.View(keys)
}

func main() {
	// Check for dark background
	hasDarkBg := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)

	model := newModel(hasDarkBg)

	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}
