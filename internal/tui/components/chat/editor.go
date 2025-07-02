package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/themes"
)

// EditorModel represents the message input editor
type EditorModel struct {
	theme        themes.Theme
	textarea     textarea.Model
	attachments  []string
	width        int
	height       int
	focused      bool
	maxHeight    int
}

// MessageSubmitMsg is sent when user submits a message
type MessageSubmitMsg struct {
	Content     string
	Attachments []string
}

// AttachmentAddedMsg is sent when a file is attached
type AttachmentAddedMsg struct {
	Path string
}

// AttachmentRemovedMsg is sent when attachment is removed
type AttachmentRemovedMsg struct {
	Index int
}

// NewEditorModel creates a new editor component
func NewEditorModel(theme themes.Theme) *EditorModel {
	ta := textarea.New()
	ta.Placeholder = "Type your message... (Ctrl+Enter to send)"
	ta.CharLimit = 10000
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	
	// Style the textarea
	ta.FocusedStyle.Base = theme.InputActive()
	ta.BlurredStyle.Base = theme.Input()
	
	return &EditorModel{
		theme:       theme,
		textarea:    ta,
		attachments: []string{},
		maxHeight:   10,
	}
}

func (m *EditorModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m *EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(m.width - 4)
		
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, editorKeys.Submit):
			// Submit message
			content := strings.TrimSpace(m.textarea.Value())
			if content != "" {
				// Clear editor
				m.textarea.Reset()
				attachments := make([]string, len(m.attachments))
				copy(attachments, m.attachments)
				m.attachments = []string{}
				
				return m, func() tea.Msg {
					return MessageSubmitMsg{
						Content:     content,
						Attachments: attachments,
					}
				}
			}
			
		case key.Matches(msg, editorKeys.NewLine):
			// Insert newline
			m.textarea.InsertString("\n")
			
		case key.Matches(msg, editorKeys.RemoveAttachment) && len(m.attachments) > 0:
			// Remove last attachment
			m.attachments = m.attachments[:len(m.attachments)-1]
			
		default:
			// Pass to textarea
			m.textarea, cmd = m.textarea.Update(msg)
			cmds = append(cmds, cmd)
		}
		
	case AttachmentAddedMsg:
		// Add attachment
		m.attachments = append(m.attachments, msg.Path)
		
	case AttachmentRemovedMsg:
		// Remove specific attachment
		if msg.Index >= 0 && msg.Index < len(m.attachments) {
			m.attachments = append(m.attachments[:msg.Index], m.attachments[msg.Index+1:]...)
		}
		
	default:
		// Pass to textarea
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}
	
	// Auto-resize based on content
	lines := strings.Count(m.textarea.Value(), "\n") + 1
	height := min(lines, m.maxHeight)
	if height < 3 {
		height = 3
	}
	m.textarea.SetHeight(height)
	
	return m, tea.Batch(cmds...)
}

func (m *EditorModel) View() string {
	var sections []string
	
	// Attachments section
	if len(m.attachments) > 0 {
		attachmentView := m.renderAttachments()
		sections = append(sections, attachmentView)
	}
	
	// Editor section
	editorStyle := m.theme.Base().
		Width(m.width).
		Padding(0, 1)
		
	editor := editorStyle.Render(m.textarea.View())
	sections = append(sections, editor)
	
	// Help text
	helpStyle := m.theme.MutedText().
		Width(m.width).
		Align(lipgloss.Center)
		
	help := helpStyle.Render("Ctrl+Enter: Send • Enter: New Line • Ctrl+R: Remove Attachment")
	sections = append(sections, help)
	
	return strings.Join(sections, "\n")
}

func (m *EditorModel) renderAttachments() string {
	var attachments []string
	
	iconStyle := m.theme.SecondaryText()
	pathStyle := m.theme.MutedText()
	
	for _, path := range m.attachments {
		icon := iconStyle.Render("📎")
		name := pathStyle.Render(truncatePath(path, 40))
		attachment := fmt.Sprintf("%s %s", icon, name)
		attachments = append(attachments, attachment)
	}
	
	containerStyle := m.theme.Base().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Primary()).
		Padding(0, 1).
		Width(m.width - 2)
		
	content := strings.Join(attachments, " • ")
	return containerStyle.Render(content)
}

// Focus sets focus on the editor
func (m *EditorModel) Focus() tea.Cmd {
	m.focused = true
	return m.textarea.Focus()
}

// Blur removes focus from the editor
func (m *EditorModel) Blur() {
	m.focused = false
	m.textarea.Blur()
}

// SetWidth sets the editor width
func (m *EditorModel) SetWidth(width int) {
	m.width = width
	m.textarea.SetWidth(width - 4)
}

// SetHeight sets the editor height
func (m *EditorModel) SetHeight(height int) {
	m.height = height
}

// AddAttachment adds a file attachment
func (m *EditorModel) AddAttachment(path string) {
	m.attachments = append(m.attachments, path)
}

// ClearAttachments removes all attachments
func (m *EditorModel) ClearAttachments() {
	m.attachments = []string{}
}

// truncatePath truncates a file path to fit within maxLen
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	
	// Show the end of the path
	return "..." + path[len(path)-maxLen+3:]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Key bindings
type editorKeyMap struct {
	Submit           key.Binding
	NewLine          key.Binding
	RemoveAttachment key.Binding
}

var editorKeys = editorKeyMap{
	Submit: key.NewBinding(
		key.WithKeys("ctrl+enter"),
		key.WithHelp("ctrl+enter", "send message"),
	),
	NewLine: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "new line"),
	),
	RemoveAttachment: key.NewBinding(
		key.WithKeys("ctrl+r"),
		key.WithHelp("ctrl+r", "remove attachment"),
	),
}