package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// MessagesComponent represents an enhanced viewport with part-based selection
type MessagesComponent interface {
	tea.Model
	View(width, height int) string
	SetWidth(width int) tea.Cmd
	PageUp() (tea.Model, tea.Cmd)
	PageDown() (tea.Model, tea.Cmd)
	HalfPageUp() (tea.Model, tea.Cmd)
	HalfPageDown() (tea.Model, tea.Cmd)
	First() (tea.Model, tea.Cmd)
	Last() (tea.Model, tea.Cmd)
	Previous() (tea.Model, tea.Cmd)
	Next() (tea.Model, tea.Cmd)
	Selected() string
	SetMessages(messages []Message)
	GetSelectedPart() (MessagePart, bool)
}

// MessagePart represents a selectable part of a message
type MessagePart struct {
	MessageID   string
	PartIndex   int
	Type        MessagePartType
	Content     string
	StartLine   int
	EndLine     int
	Metadata    map[string]interface{}
}

// MessagePartType defines the type of message part
type MessagePartType string

const (
	PartTypeText       MessagePartType = "text"
	PartTypeCode       MessagePartType = "code"
	PartTypeTool       MessagePartType = "tool"
	PartTypeToolResult MessagePartType = "tool_result"
	PartTypeDiff       MessagePartType = "diff"
	PartTypeFile       MessagePartType = "file"
	PartTypeError      MessagePartType = "error"
)

// messagesComponent implements the enhanced viewport
type messagesComponent struct {
	theme        theme.Theme
	messages     []Message
	messageParts [][]MessagePart // Parts for each message
	viewport     viewport.Model
	cache        *MessageCache
	renderer     *BlockRenderer
	markdown     *MarkdownRenderer
	parser       *MessageParser
	
	// Selection state
	selectedMessageIndex int
	selectedPartIndex    int
	selectionMode        bool
	
	// Layout
	width  int
	height int
	
	// Rendering state
	rendering bool
	tail      bool
}

// NewMessagesComponent creates a new enhanced messages viewport
func NewMessagesComponent(theme theme.Theme) MessagesComponent {
	vp := viewport.New(0, 0)
	md, _ := NewMarkdownRenderer(theme, 80)
	
	return &messagesComponent{
		theme:                theme,
		messages:             []Message{},
		messageParts:         [][]MessagePart{},
		viewport:             vp,
		cache:                NewMessageCache(200),
		renderer:             NewBlockRenderer(theme),
		markdown:             md,
		parser:               NewMessageParser(),
		selectedMessageIndex: -1,
		selectedPartIndex:    -1,
		selectionMode:        false,
		tail:                 true,
	}
}

func (m *messagesComponent) Init() tea.Cmd {
	return m.viewport.Init()
}

func (m *messagesComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		
		if m.markdown != nil && m.width > 4 {
			m.markdown.SetWidth(m.width - 4)
		}
		
		m.renderView()
		return m, nil
		
	case tea.KeyMsg:
		switch msg.String() {
		case "s", "S":
			// Toggle selection mode
			m.selectionMode = !m.selectionMode
			if m.selectionMode && m.selectedMessageIndex < 0 && len(m.messages) > 0 {
				// Start selection at first message
				m.selectedMessageIndex = 0
				m.selectedPartIndex = 0
			}
			m.renderView()
			
		case "n", "N":
			// Next part in selection mode
			if m.selectionMode {
				m.selectNextPart()
			}
			
		case "p", "P":
			// Previous part in selection mode
			if m.selectionMode {
				m.selectPreviousPart()
			}
			
		case "escape":
			// Exit selection mode
			if m.selectionMode {
				m.selectionMode = false
				m.renderView()
			}
		}
	}
	
	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)
	
	return m, tea.Batch(cmds...)
}

func (m *messagesComponent) View(width, height int) string {
	if width != m.width || height != m.height {
		m.width = width
		m.height = height
		m.viewport.Width = width
		m.viewport.Height = height
		m.renderView()
	}
	
	return m.viewport.View()
}

func (m *messagesComponent) renderView() {
	var content strings.Builder
	
	for i, msg := range m.messages {
		if i > 0 {
			content.WriteString("\n\n")
		}
		
		// Check if this message has parts
		var parts []MessagePart
		if i < len(m.messageParts) {
			parts = m.messageParts[i]
		}
		
		// Render message with part highlighting if in selection mode
		if len(parts) > 0 && m.selectionMode {
			rendered := m.renderMessageWithParts(msg, parts, i)
			content.WriteString(rendered)
		} else {
			// Regular rendering
			rendered := m.renderMessage(msg)
			content.WriteString(rendered)
		}
	}
	
	m.viewport.SetContent(content.String())
	
	if m.tail {
		m.viewport.GotoBottom()
	}
}

func (m *messagesComponent) renderMessage(msg Message) string {
	// Use cache for regular rendering
	cacheKey := m.cache.GenerateKey(msg.ID, msg.Role, msg.Content, m.width, "regular")
	
	if cached, found := m.cache.Get(cacheKey); found {
		return cached
	}
	
	var rendered string
	
	// Render based on role
	switch msg.Role {
	case "user":
		header := NewBlockRenderer(m.theme,
			WithTextColor(m.theme.Primary()),
			WithBold(),
			WithNoBorder(),
			WithMarginBottom(0),
		).Render("You:")
		
		body := UserMessageRenderer(m.theme).Render(msg.Content)
		
		rendered = lipgloss.JoinVertical(lipgloss.Left, header, body)
		
	case "assistant":
		header := NewBlockRenderer(m.theme,
			WithTextColor(m.theme.Secondary()),
			WithBold(),
			WithNoBorder(),
			WithMarginBottom(0),
		).Render("Assistant:")
		
		// Try markdown rendering
		content := msg.Content
		if m.markdown != nil {
			if mdContent, err := m.markdown.Render(content); err == nil && mdContent != "" {
				content = mdContent
			}
		}
		
		body := AssistantMessageRenderer(m.theme).Render(content)
		
		rendered = lipgloss.JoinVertical(lipgloss.Left, header, body)
		
	case "system":
		rendered = SystemMessageRenderer(m.theme).Render(fmt.Sprintf("System: %s", msg.Content))
		
	default:
		rendered = msg.Content
	}
	
	m.cache.Set(cacheKey, rendered)
	return rendered
}

func (m *messagesComponent) renderMessageWithParts(msg Message, parts []MessagePart, msgIndex int) string {
	var result strings.Builder
	
	// Render header
	switch msg.Role {
	case "user":
		header := NewBlockRenderer(m.theme,
			WithTextColor(m.theme.Primary()),
			WithBold(),
			WithNoBorder(),
			WithMarginBottom(0),
		).Render("You:")
		result.WriteString(header + "\n")
		
	case "assistant":
		header := NewBlockRenderer(m.theme,
			WithTextColor(m.theme.Secondary()),
			WithBold(),
			WithNoBorder(),
			WithMarginBottom(0),
		).Render("Assistant:")
		result.WriteString(header + "\n")
	}
	
	// Render each part
	for partIndex, part := range parts {
		isSelected := msgIndex == m.selectedMessageIndex && partIndex == m.selectedPartIndex
		
		var partRenderer *BlockRenderer
		if isSelected {
			// Highlight selected part
			partRenderer = NewBlockRenderer(m.theme,
				WithBackgroundColor(m.theme.Primary()),
				WithTextColor(m.theme.Background()),
				WithPadding(1),
				WithMarginY(1),
			)
		} else {
			// Normal part rendering based on type
			switch part.Type {
			case PartTypeCode:
				partRenderer = CodeBlockRenderer(m.theme)
			case PartTypeTool, PartTypeToolResult:
				partRenderer = NewBlockRenderer(m.theme,
					WithBackgroundColor(m.theme.BackgroundSecondary()),
					WithBorder(lipgloss.NormalBorder(), m.theme.Info()),
					WithPadding(1),
				)
			case PartTypeError:
				partRenderer = ErrorMessageRenderer(m.theme)
			default:
				partRenderer = NewBlockRenderer(m.theme,
					WithPadding(1),
				)
			}
		}
		
		// Add part type indicator
		if part.Type != PartTypeText {
			typeIndicator := NewBlockRenderer(m.theme,
				WithTextColor(m.theme.TextMuted()),
				WithItalic(),
				WithNoBorder(),
			).Render(fmt.Sprintf("[%s]", part.Type))
			result.WriteString(typeIndicator + "\n")
		}
		
		rendered := partRenderer.Render(part.Content)
		result.WriteString(rendered)
		
		if partIndex < len(parts)-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// Navigation methods

func (m *messagesComponent) SetWidth(width int) tea.Cmd {
	m.width = width
	m.viewport.Width = width
	if m.markdown != nil && width > 4 {
		m.markdown.SetWidth(width - 4)
	}
	m.renderView()
	return nil
}

func (m *messagesComponent) PageUp() (tea.Model, tea.Cmd) {
	m.viewport.ViewUp()
	m.tail = false
	return m, nil
}

func (m *messagesComponent) PageDown() (tea.Model, tea.Cmd) {
	m.viewport.ViewDown()
	// Check if we're at the bottom
	if m.viewport.AtBottom() {
		m.tail = true
	}
	return m, nil
}

func (m *messagesComponent) HalfPageUp() (tea.Model, tea.Cmd) {
	halfPage := m.viewport.Height / 2
	for i := 0; i < halfPage; i++ {
		m.viewport.ViewUp()
	}
	m.tail = false
	return m, nil
}

func (m *messagesComponent) HalfPageDown() (tea.Model, tea.Cmd) {
	halfPage := m.viewport.Height / 2
	for i := 0; i < halfPage; i++ {
		m.viewport.ViewDown()
	}
	if m.viewport.AtBottom() {
		m.tail = true
	}
	return m, nil
}

func (m *messagesComponent) First() (tea.Model, tea.Cmd) {
	m.viewport.GotoTop()
	m.tail = false
	return m, nil
}

func (m *messagesComponent) Last() (tea.Model, tea.Cmd) {
	m.viewport.GotoBottom()
	m.tail = true
	return m, nil
}

func (m *messagesComponent) Previous() (tea.Model, tea.Cmd) {
	if m.selectionMode {
		m.selectPreviousPart()
	} else {
		m.viewport.ViewUp()
	}
	return m, nil
}

func (m *messagesComponent) Next() (tea.Model, tea.Cmd) {
	if m.selectionMode {
		m.selectNextPart()
	} else {
		m.viewport.ViewDown()
	}
	return m, nil
}

// Selection methods

func (m *messagesComponent) selectNextPart() {
	if m.selectedMessageIndex < 0 || len(m.messageParts) == 0 {
		return
	}
	
	// Try next part in current message
	if m.selectedMessageIndex < len(m.messageParts) {
		parts := m.messageParts[m.selectedMessageIndex]
		if m.selectedPartIndex < len(parts)-1 {
			m.selectedPartIndex++
			m.renderView()
			return
		}
	}
	
	// Try first part of next message
	if m.selectedMessageIndex < len(m.messages)-1 {
		m.selectedMessageIndex++
		m.selectedPartIndex = 0
		m.renderView()
	}
}

func (m *messagesComponent) selectPreviousPart() {
	if m.selectedMessageIndex < 0 {
		return
	}
	
	// Try previous part in current message
	if m.selectedPartIndex > 0 {
		m.selectedPartIndex--
		m.renderView()
		return
	}
	
	// Try last part of previous message
	if m.selectedMessageIndex > 0 {
		m.selectedMessageIndex--
		if m.selectedMessageIndex < len(m.messageParts) {
			parts := m.messageParts[m.selectedMessageIndex]
			m.selectedPartIndex = len(parts) - 1
		}
		m.renderView()
	}
}

func (m *messagesComponent) Selected() string {
	part, ok := m.GetSelectedPart()
	if !ok {
		return ""
	}
	return part.Content
}

func (m *messagesComponent) GetSelectedPart() (MessagePart, bool) {
	if !m.selectionMode || m.selectedMessageIndex < 0 || m.selectedPartIndex < 0 {
		return MessagePart{}, false
	}
	
	if m.selectedMessageIndex >= len(m.messageParts) {
		return MessagePart{}, false
	}
	
	parts := m.messageParts[m.selectedMessageIndex]
	if m.selectedPartIndex >= len(parts) {
		return MessagePart{}, false
	}
	
	return parts[m.selectedPartIndex], true
}

func (m *messagesComponent) SetMessages(messages []Message) {
	m.messages = messages
	m.messageParts = m.parseMessageParts(messages)
	m.cache.Clear()
	m.renderView()
}

// parseMessageParts extracts selectable parts from messages
func (m *messagesComponent) parseMessageParts(messages []Message) [][]MessagePart {
	result := make([][]MessagePart, len(messages))
	
	for i, msg := range messages {
		parts := m.parseMessageContent(msg)
		result[i] = parts
	}
	
	return result
}

// parseMessageContent extracts parts from a single message
func (m *messagesComponent) parseMessageContent(msg Message) []MessagePart {
	if m.parser == nil {
		return nil
	}
	
	return m.parser.ParseMessage(msg)
}