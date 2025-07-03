package chat

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

type EditorModel struct {
	width    int
	height   int
	textarea textarea.Model
	theme    theme.Theme
	focused  bool
}

type EditorKeyMaps struct {
	Send    key.Binding
	NewLine key.Binding
}

var editorKeys = EditorKeyMaps{
	Send: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "send message"),
	),
	NewLine: key.NewBinding(
		key.WithKeys("shift+enter"),
		key.WithHelp("shift+enter", "new line"),
	),
}

func NewEditorModel(th theme.Theme) *EditorModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Enter to send, Shift+Enter for new line)"
	ta.CharLimit = 10000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.Focus()

	// Apply theme
	ta.FocusedStyle.Base = lipgloss.NewStyle().
		Background(th.Background()).
		Foreground(th.Text())
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle().
		Background(th.BackgroundSecondary())
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().
		Background(th.Background()).
		Foreground(th.TextMuted())

	ta.BlurredStyle.Base = lipgloss.NewStyle().
		Background(th.Background()).
		Foreground(th.TextMuted())

	return &EditorModel{
		textarea: ta,
		theme:    th,
		focused:  true,
	}
}

func (m *EditorModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m *EditorModel) send() tea.Cmd {
	value := strings.TrimSpace(m.textarea.Value())
	if value == "" {
		return nil
	}
	
	m.textarea.Reset()
	
	return func() tea.Msg {
		return MessageSubmitMsg{
			Content:     value,
			Attachments: []string{}, // TODO: Add attachment support
		}
	}
}

func (m *EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.textarea.Focused() {
			// Handle Shift+Enter for new line
			if key.Matches(msg, editorKeys.NewLine) {
				// Add a newline to the current value
				value := m.textarea.Value()
				m.textarea.SetValue(value + "\n")
				// Move cursor to end
				m.textarea.CursorEnd()
				return m, nil
			}
			
			// Handle Enter to send
			if key.Matches(msg, editorKeys.Send) {
				// Send the message
				return m, m.send()
			}
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m *EditorModel) View() string {
	// Style the prompt with theme colors
	promptStyle := lipgloss.NewStyle().
		Padding(0, 0, 0, 1).
		Bold(true).
		Foreground(m.theme.Primary())

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		promptStyle.Render(">"),
		m.textarea.View(),
	)
}

func (m *EditorModel) SetWidth(width int) {
	m.width = width
	m.textarea.SetWidth(width - 3) // Account for prompt and padding
}

func (m *EditorModel) SetHeight(height int) {
	m.height = height
	m.textarea.SetHeight(height)
}

func (m *EditorModel) Focus() tea.Cmd {
	m.focused = true
	return m.textarea.Focus()
}

func (m *EditorModel) Blur() {
	m.focused = false
	m.textarea.Blur()
}

func (m *EditorModel) Focused() bool {
	return m.focused
}

// AddAttachment adds a file attachment (placeholder for now)
func (m *EditorModel) AddAttachment(path string) {
	// TODO: Implement attachment functionality
}