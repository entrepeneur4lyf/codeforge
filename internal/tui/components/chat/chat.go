package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/themes"
)

// ChatModel represents the main chat interface
type ChatModel struct {
	theme        themes.Theme
	messages     []Message
	viewport     viewport.Model
	width        int
	height       int
	sessionID    string
	isStreaming  bool
	streamBuffer strings.Builder
	ready        bool
}

// Message represents a chat message
type Message struct {
	ID       string
	Role     string // "user", "assistant", "system"
	Content  string
	Error    error
	Metadata map[string]interface{}
}

// StreamUpdateMsg is sent when streaming content arrives
type StreamUpdateMsg struct {
	SessionID string
	Content   string
	Done      bool
}

// MessageSentMsg is sent when a new message is added
type MessageSentMsg struct {
	SessionID string
	Message   Message
}

// NewChatModel creates a new chat component
func NewChatModel(theme themes.Theme, sessionID string) *ChatModel {
	return &ChatModel{
		theme:     theme,
		messages:  []Message{},
		sessionID: sessionID,
		viewport:  viewport.New(0, 0),
	}
}

func (m *ChatModel) Init() tea.Cmd {
	return nil
}

func (m *ChatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		if !m.ready {
			// Initialize viewport
			m.viewport = viewport.New(m.width, m.height)
			m.viewport.YPosition = 0
			m.viewport.HighPerformanceRendering = false
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = m.height
		}
		
		// Update content
		m.updateViewport()
		
	case StreamUpdateMsg:
		if msg.SessionID == m.sessionID && m.isStreaming {
			if msg.Done {
				// Finalize the streamed message
				if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
					m.messages[len(m.messages)-1].Content = m.streamBuffer.String()
				}
				m.isStreaming = false
				m.streamBuffer.Reset()
			} else {
				// Append to stream buffer
				m.streamBuffer.WriteString(msg.Content)
				if len(m.messages) > 0 && m.messages[len(m.messages)-1].Role == "assistant" {
					m.messages[len(m.messages)-1].Content = m.streamBuffer.String()
				}
			}
			m.updateViewport()
			// Auto-scroll to bottom
			m.viewport.GotoBottom()
		}
		
	case MessageSentMsg:
		if msg.SessionID == m.sessionID {
			m.addMessage(msg.Message)
			
			// If it's an assistant message, prepare for streaming
			if msg.Message.Role == "assistant" {
				m.isStreaming = true
				m.streamBuffer.Reset()
			}
		}
	}
	
	// Update viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	
	return m, tea.Batch(cmds...)
}

func (m *ChatModel) View() string {
	if !m.ready {
		return "Initializing chat..."
	}
	
	return m.viewport.View()
}

// AddMessage adds a new message to the chat
func (m *ChatModel) addMessage(msg Message) {
	m.messages = append(m.messages, msg)
	m.updateViewport()
	// Auto-scroll to bottom for new messages
	m.viewport.GotoBottom()
}

// updateViewport updates the viewport content with styled messages
func (m *ChatModel) updateViewport() {
	var content strings.Builder
	
	for i, msg := range m.messages {
		if i > 0 {
			content.WriteString("\n\n")
		}
		
		// Render message based on role
		switch msg.Role {
		case "user":
			content.WriteString(m.renderUserMessage(msg))
		case "assistant":
			content.WriteString(m.renderAssistantMessage(msg))
		case "system":
			content.WriteString(m.renderSystemMessage(msg))
		}
	}
	
	m.viewport.SetContent(content.String())
}

func (m *ChatModel) renderUserMessage(msg Message) string {
	header := m.theme.PrimaryText().Bold(true).Render("You:")
	
	// Style the content
	contentStyle := m.theme.Base().
		PaddingLeft(2).
		Width(m.width - 4)
	
	content := contentStyle.Render(msg.Content)
	
	return fmt.Sprintf("%s\n%s", header, content)
}

func (m *ChatModel) renderAssistantMessage(msg Message) string {
	header := m.theme.SecondaryText().Bold(true).Render("Assistant:")
	
	// Render markdown if possible
	content := msg.Content
	if m.isStreaming && &msg == &m.messages[len(m.messages)-1] {
		// Add cursor for streaming
		content += "▋"
	}
	
	// Try to render as markdown
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(m.width-4),
	)
	if err == nil {
		rendered, err := renderer.Render(content)
		if err == nil {
			content = rendered
		}
	}
	
	// Apply padding
	contentStyle := m.theme.Base().
		PaddingLeft(2).
		Width(m.width - 4)
	
	styledContent := contentStyle.Render(strings.TrimSpace(content))
	
	// Add error if present
	if msg.Error != nil {
		errorStyle := m.theme.ErrorText().
			PaddingLeft(2).
			Width(m.width - 4)
		errorMsg := errorStyle.Render(fmt.Sprintf("Error: %v", msg.Error))
		styledContent = fmt.Sprintf("%s\n%s", styledContent, errorMsg)
	}
	
	return fmt.Sprintf("%s\n%s", header, styledContent)
}

func (m *ChatModel) renderSystemMessage(msg Message) string {
	style := m.theme.MutedText().
		Italic(true).
		PaddingLeft(2).
		Width(m.width - 4)
	
	return style.Render(fmt.Sprintf("System: %s", msg.Content))
}

// GetSessionID returns the current session ID
func (m *ChatModel) GetSessionID() string {
	return m.sessionID
}

// SetSessionID sets a new session ID and clears messages
func (m *ChatModel) SetSessionID(sessionID string) {
	m.sessionID = sessionID
	m.messages = []Message{}
	m.updateViewport()
}

// ClearMessages clears all messages
func (m *ChatModel) ClearMessages() {
	m.messages = []Message{}
	m.updateViewport()
	m.viewport.GotoTop()
}

// ScrollUp scrolls the viewport up
func (m *ChatModel) ScrollUp() {
	m.viewport.LineUp(3)
}

// ScrollDown scrolls the viewport down
func (m *ChatModel) ScrollDown() {
	m.viewport.LineDown(3)
}

// AtTop returns true if viewport is at the top
func (m *ChatModel) AtTop() bool {
	return m.viewport.AtTop()
}

// AtBottom returns true if viewport is at the bottom
func (m *ChatModel) AtBottom() bool {
	return m.viewport.AtBottom()
}

// SetSize sets the chat view size
func (m *ChatModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.ready {
		m.viewport.Width = width
		m.viewport.Height = height
		m.updateViewport()
	}
}