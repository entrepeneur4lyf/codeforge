package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// ChatModel represents the main chat interface
type ChatModel struct {
	theme        theme.Theme
	messages     []Message
	viewport     viewport.Model
	width        int
	height       int
	sessionID    string
	isStreaming  bool
	streamBuffer strings.Builder
	ready        bool
	cache        *MessageCache
	markdown     *MarkdownRenderer
	enableMarkdown bool
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

// MessageSubmitMsg is sent when user submits a message
type MessageSubmitMsg struct {
	Content     string
	Attachments []string
}

// NewChatModel creates a new chat component
func NewChatModel(th theme.Theme, sessionID string) *ChatModel {
	vp := viewport.New(0, 0)
	
	// Create markdown renderer (ignore error for now, will handle in SetSize)
	md, _ := NewMarkdownRenderer(th, 80) // Default width
	
	return &ChatModel{
		theme:     th,
		messages:  []Message{},
		sessionID: sessionID,
		viewport:  vp,
		cache:     NewMessageCache(100), // Cache up to 100 rendered messages
		markdown:   md,
		enableMarkdown: true, // Enable by default
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
			m.ready = true
			
			// Initialize markdown renderer with proper width
			if m.markdown == nil {
				m.markdown, _ = NewMarkdownRenderer(m.theme, m.width-4)
			}
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = m.height
			
			// Update markdown width
			if m.markdown != nil && m.width > 4 {
				m.markdown.SetWidth(m.width - 4)
			}
		}

		// Update content
		m.updateViewport()

	case MessageSentMsg:
		if msg.SessionID == m.sessionID {
			m.addMessage(msg.Message)
		}

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

// StartStreaming prepares for streaming a new assistant message
func (m *ChatModel) StartStreaming() {
	assistantMsg := Message{
		ID:      fmt.Sprintf("stream-%d", len(m.messages)),
		Role:    "assistant",
		Content: "",
	}
	m.addMessage(assistantMsg)
	m.isStreaming = true
	m.streamBuffer.Reset()
}

// updateViewport updates the viewport content with styled messages
func (m *ChatModel) updateViewport() {
	var content strings.Builder

	for i, msg := range m.messages {
		if i > 0 {
			content.WriteString("\n\n")
		}

		// Generate cache key based on message properties and current state
		cacheKey := m.cache.GenerateKey(
			msg.ID,
			msg.Role,
			msg.Content,
			m.width,
			// Include streaming state if this is the last message
			m.isStreaming && i == len(m.messages)-1,
		)

		// Check cache first
		if rendered, found := m.cache.Get(cacheKey); found {
			content.WriteString(rendered)
			continue
		}

		// Render message based on role
		var rendered string
		
		// Check for error first
		if msg.Error != nil {
			rendered = m.renderErrorMessage(msg)
		} else {
			switch msg.Role {
			case "user":
				rendered = m.renderUserMessage(msg)
			case "assistant":
				rendered = m.renderAssistantMessage(msg)
			case "system":
				rendered = m.renderSystemMessage(msg)
			default:
				// Fallback for unknown roles
				rendered = m.renderSystemMessage(msg)
			}
		}

		// Cache the rendered message (except for actively streaming messages)
		if !(m.isStreaming && i == len(m.messages)-1) {
			m.cache.Set(cacheKey, rendered)
		}

		content.WriteString(rendered)
	}

	m.viewport.SetContent(content.String())
}

func (m *ChatModel) renderUserMessage(msg Message) string {
	header := NewBlockRenderer(m.theme,
		WithTextColor(m.theme.Primary()),
		WithBold(),
		WithNoBorder(),
		WithMarginBottom(0),
	).Render("You:")

	content := UserMessageRenderer(m.theme).Render(msg.Content)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
	)
}

func (m *ChatModel) renderAssistantMessage(msg Message) string {
	header := NewBlockRenderer(m.theme,
		WithTextColor(m.theme.Secondary()),
		WithBold(),
		WithNoBorder(),
		WithMarginBottom(0),
	).Render("Assistant:")

	// Process content
	content := msg.Content
	
	// Add cursor for streaming messages
	isStreamingMsg := m.isStreaming && len(m.messages) > 0 && m.messages[len(m.messages)-1].ID == msg.ID
	if isStreamingMsg {
		content = msg.Content + "▋"
	}

	// Render with markdown if enabled and not streaming
	var body string
	if m.enableMarkdown && m.markdown != nil && !isStreamingMsg {
		if rendered, err := m.markdown.Render(content); err == nil && rendered != "" {
			// Wrap markdown output in assistant styling
			body = AssistantMessageRenderer(m.theme).Render(rendered)
		} else {
			// Fallback to plain rendering
			body = AssistantMessageRenderer(m.theme).Render(content)
		}
	} else {
		body = AssistantMessageRenderer(m.theme).Render(content)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
	)
}

func (m *ChatModel) renderSystemMessage(msg Message) string {
	return SystemMessageRenderer(m.theme).Render(fmt.Sprintf("System: %s", msg.Content))
}

func (m *ChatModel) renderErrorMessage(msg Message) string {
	header := NewBlockRenderer(m.theme,
		WithTextColor(m.theme.Error()),
		WithBold(),
		WithNoBorder(),
		WithMarginBottom(0),
	).Render("Error:")

	errorContent := fmt.Sprintf("%s\n\nError: %v", msg.Content, msg.Error)
	body := ErrorMessageRenderer(m.theme).Render(errorContent)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		body,
	)
}

// SetSize sets the chat view size
func (m *ChatModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	if m.ready {
		m.viewport.Width = width
		m.viewport.Height = height
		
		// Update markdown renderer width
		if m.markdown != nil {
			// Account for padding and borders
			contentWidth := width - 4
			if contentWidth > 0 {
				m.markdown.SetWidth(contentWidth)
			}
		}
		
		// Clear cache and re-render with new width
		m.cache.Clear()
		m.updateViewport()
	}
}

// ClearMessages clears all messages
func (m *ChatModel) ClearMessages() {
	m.messages = []Message{}
	m.updateViewport()
	// Clear cache when clearing messages
	m.cache.Clear()
}

// ScrollUp scrolls the viewport up
func (m *ChatModel) ScrollUp() {
	// Scroll up by 3 lines
	for i := 0; i < 3; i++ {
		m.viewport.ViewUp()
	}
}

// ScrollDown scrolls the viewport down  
func (m *ChatModel) ScrollDown() {
	// Scroll down by 3 lines
	for i := 0; i < 3; i++ {
		m.viewport.ViewDown()
	}
}

// SetSessionID sets the session ID
func (m *ChatModel) SetSessionID(sessionID string) {
	m.sessionID = sessionID
	// Clear cache when switching sessions
	m.cache.Clear()
	if m.markdown != nil {
		m.markdown.Clear()
	}
}

// EnableMarkdown enables or disables markdown rendering
func (m *ChatModel) EnableMarkdown(enable bool) {
	if m.enableMarkdown != enable {
		m.enableMarkdown = enable
		// Clear cache and re-render
		m.cache.Clear()
		m.updateViewport()
	}
}

// IsMarkdownEnabled returns whether markdown rendering is enabled
func (m *ChatModel) IsMarkdownEnabled() bool {
	return m.enableMarkdown
}

// LoadMessages loads messages for display
func (m *ChatModel) LoadMessages(messages []Message) {
	m.messages = messages
	// Clear cache when loading new messages
	m.cache.Clear()
	if m.markdown != nil {
		m.markdown.Clear()
	}
	m.updateViewport()
	m.viewport.GotoBottom()
}