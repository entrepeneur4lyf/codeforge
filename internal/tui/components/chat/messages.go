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
	ViewWithSize(width, height int) string
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
	PartTypeDiagnostic MessagePartType = "diagnostic"
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
	toolRenderer *ToolRenderer
	diagRenderer *DiagnosticRenderer
	fileRenderer *FileRenderer
	asyncRenderer *AsyncRenderer
	
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
	
	// Async rendering state
	asyncTasks map[string]string // messageID -> taskID
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
		toolRenderer:         NewToolRenderer(theme),
		diagRenderer:         NewDiagnosticRenderer(theme),
		fileRenderer:         NewFileRenderer(theme),
		asyncRenderer:        NewAsyncRenderer(theme),
		selectedMessageIndex: -1,
		selectedPartIndex:    -1,
		selectionMode:        false,
		tail:                 true,
		asyncTasks:           make(map[string]string),
	}
}

func (m *messagesComponent) Init() tea.Cmd {
	return m.viewport.Init()
}

func (m *messagesComponent) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case RenderCompleteMsg:
		// Handle async render completion
		m.handleRenderComplete(msg)
		m.renderView()
		return m, nil
		
	case RenderProgressMsg:
		// Update render progress (could show in status line)
		// For now, just continue checking
		if taskID, ok := m.asyncTasks[msg.TaskID]; ok {
			cmd := m.asyncRenderer.checkRenderStatus(taskID)
			return m, cmd
		}
		return m, nil
		
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
		switch msg.Type {
		case tea.KeyRunes:
			switch string(msg.Runes) {
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
			
			case "e", "E":
				// Toggle expansion of current tool part
				if m.selectionMode {
					if part, ok := m.GetSelectedPart(); ok {
						if part.Type == PartTypeTool || part.Type == PartTypeToolResult {
							var toolID string
							if part.Type == PartTypeTool {
								toolID = fmt.Sprintf("%s-tool-%d", part.MessageID, part.PartIndex)
							} else {
								toolID = fmt.Sprintf("%s-result-%d", part.MessageID, part.PartIndex)
							}
							m.toolRenderer.ToggleExpanded(toolID)
							m.renderView()
						}
					}
				}
			
			case "x", "X":
				// Expand all tools
				if m.selectionMode {
					// Find all tool IDs
					for msgIdx, parts := range m.messageParts {
						if msgIdx >= len(m.messages) {
							continue
						}
						msg := m.messages[msgIdx]
						for _, part := range parts {
							if part.Type == PartTypeTool {
								toolID := fmt.Sprintf("%s-tool-%d", msg.ID, part.PartIndex)
								m.toolRenderer.SetExpanded(toolID, true)
							} else if part.Type == PartTypeToolResult {
								toolID := fmt.Sprintf("%s-result-%d", msg.ID, part.PartIndex)
								m.toolRenderer.SetExpanded(toolID, true)
							}
						}
					}
					m.renderView()
				}
			
			case "c", "C":
				// Collapse all tools
				if m.selectionMode {
					m.toolRenderer.CollapseAll()
					m.renderView()
				}
			}
			
		case tea.KeyEsc:
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

func (m *messagesComponent) View() string {
	return m.viewport.View()
}

func (m *messagesComponent) ViewWithSize(width, height int) string {
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
		
		// Use part-based rendering if we have parts with special types (tools, etc.)
		// or if we're in selection mode
		usePartRendering := m.selectionMode
		if !usePartRendering && len(parts) > 0 {
			// Check if any parts need special rendering
			for _, part := range parts {
				if part.Type != PartTypeText {
					usePartRendering = true
					break
				}
			}
		}
		
		if usePartRendering && len(parts) > 0 {
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
	// Check if async rendering is in progress
	if taskID, ok := m.asyncTasks[msg.ID]; ok {
		if task, found := m.asyncRenderer.GetTask(taskID); found {
			if task.Status == RenderStatusComplete {
				// Use the async rendered result
				m.cache.Set(m.cache.GenerateKey(msg.ID, "async-complete"), task.Result)
				delete(m.asyncTasks, msg.ID)
				return task.Result
			} else {
				// Show status while rendering
				return m.renderMessageWithStatus(msg, task)
			}
		}
	}
	
	// Check if we should use async rendering (for large messages)
	if len(msg.Content) > 5000 && msg.Role == "assistant" {
		// Start async rendering
		placeholder, _ := m.asyncRenderer.QueueRender(msg.ID, msg.Content)
		// Store the task ID
		if task, ok := m.asyncRenderer.GetTaskByMessageID(msg.ID); ok {
			m.asyncTasks[msg.ID] = task.ID
		}
		// Return placeholder for now
		return m.renderMessageWithPlaceholder(msg, placeholder)
	}
	
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
		if isSelected && part.Type != PartTypeTool && part.Type != PartTypeToolResult && part.Type != PartTypeDiagnostic && part.Type != PartTypeFile {
			// Highlight selected part (except for tools and diagnostics which have their own rendering)
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
			case PartTypeTool:
				// Use tool renderer for tool invocations
				if tool, err := ParseToolContent(part.Content); err == nil {
					toolID := fmt.Sprintf("%s-tool-%d", part.MessageID, part.PartIndex)
					expanded := m.toolRenderer.IsExpanded(toolID)
					rendered := m.toolRenderer.RenderToolInvocation(*tool, expanded)
					
					// Add selection indicator if selected
					if isSelected {
						selectionIndicator := NewBlockRenderer(m.theme,
							WithTextColor(m.theme.Primary()),
							WithBold(),
							WithNoBorder(),
						).Render("▸ Selected")
						rendered = selectionIndicator + "\n" + rendered
					}
					
					result.WriteString(rendered)
					if partIndex < len(parts)-1 {
						result.WriteString("\n")
					}
					continue
				}
				// Fallback to default rendering if parsing fails
				partRenderer = NewBlockRenderer(m.theme,
					WithBackgroundColor(m.theme.BackgroundSecondary()),
					WithBorder(lipgloss.NormalBorder(), m.theme.Info()),
					WithPadding(1),
				)
			case PartTypeToolResult:
				// Use tool renderer for tool results
				if toolResult, err := ParseToolResult(part.Content); err == nil {
					toolID := fmt.Sprintf("%s-result-%d", part.MessageID, part.PartIndex)
					expanded := m.toolRenderer.IsExpanded(toolID)
					rendered := m.toolRenderer.RenderToolResult(*toolResult, expanded)
					
					// Add selection indicator if selected
					if isSelected {
						selectionIndicator := NewBlockRenderer(m.theme,
							WithTextColor(m.theme.Primary()),
							WithBold(),
							WithNoBorder(),
						).Render("▸ Selected")
						rendered = selectionIndicator + "\n" + rendered
					}
					
					result.WriteString(rendered)
					if partIndex < len(parts)-1 {
						result.WriteString("\n")
					}
					continue
				}
				// Fallback to default rendering if parsing fails
				partRenderer = NewBlockRenderer(m.theme,
					WithBackgroundColor(m.theme.BackgroundSecondary()),
					WithBorder(lipgloss.NormalBorder(), m.theme.Info()),
					WithPadding(1),
				)
			case PartTypeError:
				partRenderer = ErrorMessageRenderer(m.theme)
			case PartTypeDiagnostic:
				// Use diagnostic renderer
				diag, err := ParseDiagnosticContent(part.Content)
				if err == nil {
					rendered := m.diagRenderer.RenderDiagnostic(*diag)
					
					// Add selection indicator if selected
					if isSelected {
						selectionIndicator := NewBlockRenderer(m.theme,
							WithTextColor(m.theme.Primary()),
							WithBold(),
							WithNoBorder(),
						).Render("▸ Selected")
						rendered = selectionIndicator + "\n" + rendered
					}
					
					result.WriteString(rendered)
					if partIndex < len(parts)-1 {
						result.WriteString("\n")
					}
					continue
				}
				// Fallback to error renderer if parsing fails
				partRenderer = ErrorMessageRenderer(m.theme)
			case PartTypeFile:
				// Use file renderer
				var fileContent FileContent
				fileContent.Content = part.Content
				
				// Extract metadata
				if path, ok := part.Metadata["path"].(string); ok {
					fileContent.Path = path
					
					// Parse line range from path (e.g., file.go:10-20)
					if parts := strings.Split(path, ":"); len(parts) > 1 {
						fileContent.Path = parts[0]
						// Parse line range
						if lineRange := parts[1]; strings.Contains(lineRange, "-") {
							var start, end int
							fmt.Sscanf(lineRange, "%d-%d", &start, &end)
							fileContent.StartLine = start
							fileContent.EndLine = end
						} else {
							// Single line
							var line int
							fmt.Sscanf(lineRange, "%d", &line)
							fileContent.StartLine = line
							fileContent.EndLine = line
						}
					}
				}
				
				rendered := m.fileRenderer.RenderFile(fileContent)
				
				// Add selection indicator if selected
				if isSelected {
					selectionIndicator := NewBlockRenderer(m.theme,
						WithTextColor(m.theme.Primary()),
						WithBold(),
						WithNoBorder(),
					).Render("▸ Selected")
					rendered = selectionIndicator + "\n" + rendered
				}
				
				result.WriteString(rendered)
				if partIndex < len(parts)-1 {
					result.WriteString("\n")
				}
				continue
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

// handleRenderComplete handles completion of async rendering
func (m *messagesComponent) handleRenderComplete(msg RenderCompleteMsg) {
	// Find the message that completed
	for msgID, taskID := range m.asyncTasks {
		if taskID == msg.TaskID && msgID == msg.MessageID {
			// Clear from cache to force re-render with new content
			m.cache.InvalidateMatching(func(key string) bool {
				return strings.Contains(key, msgID)
			})
			delete(m.asyncTasks, msgID)
			break
		}
	}
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

// renderMessageWithStatus renders a message with async rendering status
func (m *messagesComponent) renderMessageWithStatus(msg Message, task *RenderTask) string {
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
	
	// Render status
	status := m.asyncRenderer.RenderStatus(task)
	statusBlock := NewBlockRenderer(m.theme,
		WithTextColor(m.theme.Info()),
		WithBorder(lipgloss.NormalBorder(), m.theme.Info()),
		WithPadding(1),
		WithMarginY(1),
	).Render(status)
	
	result.WriteString(statusBlock)
	
	// If we have partial content, show it
	if task.Result != "" {
		result.WriteString("\n")
		result.WriteString(task.Result)
	}
	
	return result.String()
}

// renderMessageWithPlaceholder renders a message with a placeholder
func (m *messagesComponent) renderMessageWithPlaceholder(msg Message, placeholder string) string {
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
	
	// Add placeholder
	result.WriteString(placeholder)
	
	return result.String()
}