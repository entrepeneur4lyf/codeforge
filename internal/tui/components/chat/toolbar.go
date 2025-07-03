package chat

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// ToolbarButton represents a clickable button in the toolbar
type ToolbarButton struct {
	Icon    string
	Tooltip string
	Action  ToolbarAction
	X       int // Position for mouse detection
	Width   int
}

// ToolbarAction represents an action triggered by a button
type ToolbarAction int

const (
	ActionNewChat ToolbarAction = iota
	ActionClearChat
	ActionAttachFile
	ActionFilePicker
	ActionModelSelect
)

// ToolbarModel represents the toolbar component
type ToolbarModel struct {
	theme         theme.Theme
	buttons       []ToolbarButton
	width         int
	hoveredButton int // -1 means no button is hovered
	focused       bool
}

// NewToolbarModel creates a new toolbar
func NewToolbarModel(th theme.Theme) *ToolbarModel {
	return &ToolbarModel{
		theme: th,
		buttons: []ToolbarButton{
			{Icon: "🔄", Tooltip: "New Chat (Ctrl+N)", Action: ActionNewChat},
			{Icon: "🗑", Tooltip: "Clear Chat (Ctrl+L)", Action: ActionClearChat},
			{Icon: "🔗", Tooltip: "Attach File (Ctrl+F)", Action: ActionAttachFile},
			{Icon: "📂", Tooltip: "File Picker (Ctrl+P)", Action: ActionFilePicker},
			{Icon: "🤖", Tooltip: "Select Model (Ctrl+M)", Action: ActionModelSelect},
		},
		hoveredButton: -1,
	}
}

func (m *ToolbarModel) Init() tea.Cmd {
	return nil
}

func (m *ToolbarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.MouseMsg:
		// Handle mouse hover and clicks
		if msg.Y == 0 { // Toolbar is on the first line
			m.hoveredButton = m.getButtonAtX(msg.X)

			if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
				if m.hoveredButton >= 0 && m.hoveredButton < len(m.buttons) {
					// Return the action message
					return m, func() tea.Msg {
						return ToolbarClickMsg{Action: m.buttons[m.hoveredButton].Action}
					}
				}
			}
		} else {
			m.hoveredButton = -1
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.updateButtonPositions()
	}

	return m, nil
}

func (m *ToolbarModel) View() string {
	if m.width == 0 {
		return ""
	}

	// Build the toolbar
	var buttons []string

	for i, btn := range m.buttons {
		style := lipgloss.NewStyle().
			Padding(0, 1).
			Background(m.theme.BackgroundSecondary()).
			Foreground(m.theme.Text())

		// Highlight hovered button
		if i == m.hoveredButton {
			style = style.
				Background(m.theme.Primary()).
				Foreground(m.theme.Background()).
				Bold(true)
		}

		// Add button with icon
		buttons = append(buttons, style.Render(btn.Icon))
	}

	// Join buttons with spacing
	toolbar := lipgloss.JoinHorizontal(
		lipgloss.Left,
		buttons...,
	)

	// Add tooltip if hovering
	if m.hoveredButton >= 0 && m.hoveredButton < len(m.buttons) {
		tooltipStyle := lipgloss.NewStyle().
			Foreground(m.theme.TextMuted()).
			Italic(true).
			PaddingLeft(2)

		tooltip := tooltipStyle.Render(m.buttons[m.hoveredButton].Tooltip)
		toolbar = lipgloss.JoinHorizontal(lipgloss.Left, toolbar, tooltip)
	}

	// Center the toolbar
	return lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Left).
		Background(m.theme.Background()).
		Render(toolbar)
}

func (m *ToolbarModel) updateButtonPositions() {
	// Calculate button positions for mouse detection
	x := 0
	for i := range m.buttons {
		m.buttons[i].X = x
		// Each button is icon (3 chars) + padding (2) = 5 chars
		m.buttons[i].Width = 5
		x += m.buttons[i].Width
	}
}

func (m *ToolbarModel) getButtonAtX(x int) int {
	for i, btn := range m.buttons {
		if x >= btn.X && x < btn.X+btn.Width {
			return i
		}
	}
	return -1
}

func (m *ToolbarModel) SetWidth(width int) {
	m.width = width
	m.updateButtonPositions()
}

func (m *ToolbarModel) Height() int {
	return 1 // Toolbar is always 1 line high
}

func (m *ToolbarModel) Focus() {
	m.focused = true
}

func (m *ToolbarModel) Blur() {
	m.focused = false
}

// ToolbarClickMsg is sent when a toolbar button is clicked
type ToolbarClickMsg struct {
	Action ToolbarAction
}
