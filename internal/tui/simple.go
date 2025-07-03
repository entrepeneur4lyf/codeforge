package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/app"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// SimpleModel is a minimal working TUI implementation
type SimpleModel struct {
	app     *app.App
	theme   theme.Theme
	width   int
	height  int
	
	// Components
	textarea textarea.Model
	messages []string
	
	// State
	inputFocused bool
}

// NewSimple creates a minimal working TUI
func NewSimple(application *app.App) *SimpleModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Ctrl+Enter to send)"
	ta.CharLimit = 2000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.Focus()

	return &SimpleModel{
		app:          application,
		theme:        theme.NewDefaultTheme(),
		textarea:     ta,
		messages:     []string{},
		inputFocused: true,
	}
}

func (m *SimpleModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m *SimpleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Update textarea width
		m.textarea.SetWidth(m.width - 4)
		
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			return m, tea.Quit
			
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+enter"))):
			// Send message
			content := strings.TrimSpace(m.textarea.Value())
			if content != "" {
				m.messages = append(m.messages, fmt.Sprintf("You: %s", content))
				m.messages = append(m.messages, fmt.Sprintf("Assistant: Echo - %s", content))
				m.textarea.Reset()
			}
			return m, nil
		}
	}

	// Update textarea
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *SimpleModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var sections []string

	// Header
	header := lipgloss.NewStyle().
		Background(m.theme.Primary()).
		Foreground(m.theme.Background()).
		Width(m.width).
		Padding(0, 2).
		Render("CodeForge Chat - Simple Mode")
	sections = append(sections, header)

	// Messages area
	chatHeight := m.height - 6 // Reserve space for header, input, status
	if chatHeight < 1 {
		chatHeight = 1
	}

	// Show recent messages
	var messageLines []string
	for i := len(m.messages) - 1; i >= 0 && len(messageLines) < chatHeight; i-- {
		messageLines = append([]string{m.messages[i]}, messageLines...)
	}
	
	// Pad with empty lines if needed
	for len(messageLines) < chatHeight {
		messageLines = append([]string{""}, messageLines...)
	}

	chatContent := lipgloss.NewStyle().
		Width(m.width).
		Height(chatHeight).
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.theme.BorderNormal()).
		Padding(1).
		Render(strings.Join(messageLines, "\n"))
	sections = append(sections, chatContent)

	// Input area
	inputContent := lipgloss.NewStyle().
		Width(m.width).
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.theme.BorderFocused()).
		Padding(0, 1).
		Render(m.textarea.View())
	sections = append(sections, inputContent)

	// Status bar
	status := lipgloss.NewStyle().
		Background(m.theme.BackgroundSecondary()).
		Foreground(m.theme.TextMuted()).
		Width(m.width).
		Padding(0, 2).
		Render("Ctrl+Enter: Send • Ctrl+C: Quit")
	sections = append(sections, status)

	return strings.Join(sections, "\n")
}