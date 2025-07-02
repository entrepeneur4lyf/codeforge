package page

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/app"
	contextmgmt "github.com/entrepeneur4lyf/codeforge/internal/context"
	"github.com/entrepeneur4lyf/codeforge/internal/events"
	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/permissions"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/components/chat"
	dialog "github.com/entrepeneur4lyf/codeforge/internal/tui/components/dialogs"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// ChatPage represents the main chat interface
type ChatPage struct {
	app         *app.App
	theme       theme.Theme
	
	// Layout
	width       int
	height      int
	splitRatio  float64 // Sidebar width ratio
	showSidebar bool
	
	// Components
	chatView    *chat.ChatModel
	editor      *chat.EditorModel
	sessions    *chat.SessionsModel
	
	// State
	focusIndex  int // 0: chat, 1: editor, 2: sessions
	currentSessionID string
	currentModel     string
	
	// Processing state
	isProcessing bool
	lastError    error
	
	// Context tracking
	currentContextTokens int
	maxContextTokens     int
	lastProcessedContext *contextmgmt.ProcessedContext
}

// NewChatPage creates a new chat page
func NewChatPage(app *app.App, theme theme.Theme) *ChatPage {
	// Create initial session
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	
	// Get current model
	provider, model := app.GetCurrentModel()
	currentModel := fmt.Sprintf("%s/%s", provider, model)
	
	// Get model info for context window
	defaultModel := llm.GetDefaultModel()
	maxContext := defaultModel.Info.ContextWindow
	
	return &ChatPage{
		app:              app,
		theme:            theme,
		splitRatio:       0.25,
		showSidebar:      true,
		chatView:         chat.NewChatModel(theme, sessionID),
		editor:           chat.NewEditorModel(theme),
		sessions:         chat.NewSessionsModel(theme),
		focusIndex:       1, // Start with editor focused
		currentSessionID: sessionID,
		currentModel:     currentModel,
		maxContextTokens: maxContext,
		currentContextTokens: 0,
	}
}

func (p *ChatPage) Init() tea.Cmd {
	// Initialize components
	cmds := []tea.Cmd{
		p.chatView.Init(),
		p.editor.Init(),
		p.sessions.Init(),
	}
	
	// Create initial session
	initialSession := chat.Session{
		ID:           p.currentSessionID,
		Title:        "New Chat",
		UpdatedAt:    time.Now(),
		MessageCount: 0,
	}
	
	cmds = append(cmds, func() tea.Msg {
		return chat.SessionCreatedMsg{Session: initialSession}
	})
	
	// Focus editor
	cmds = append(cmds, p.editor.Focus())
	
	return tea.Batch(cmds...)
}

func (p *ChatPage) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.width = msg.Width
		p.height = msg.Height
		p.updateLayout()
		
	case tea.KeyMsg:
		// Global keys
		switch {
		case key.Matches(msg, chatKeys.ToggleSidebar):
			p.showSidebar = !p.showSidebar
			p.updateLayout()
			return p, nil
			
		case key.Matches(msg, chatKeys.FocusNext):
			p.cycleFocus(1)
			return p, p.updateFocus()
			
		case key.Matches(msg, chatKeys.FocusPrev):
			p.cycleFocus(-1)
			return p, p.updateFocus()
			
		case key.Matches(msg, chatKeys.NewSession):
			// Create new session
			newSession := chat.Session{
				ID:           fmt.Sprintf("session-%d", time.Now().Unix()),
				Title:        "New Chat",
				UpdatedAt:    time.Now(),
				MessageCount: 0,
			}
			return p, func() tea.Msg {
				return chat.SessionCreatedMsg{Session: newSession}
			}
			
		case key.Matches(msg, chatKeys.ClearChat):
			p.chatView.ClearMessages()
			return p, nil
		}
		
		// Route to focused component
		switch p.focusIndex {
		case 0: // Chat view
			switch {
			case key.Matches(msg, chatKeys.ScrollUp):
				p.chatView.ScrollUp()
			case key.Matches(msg, chatKeys.ScrollDown):
				p.chatView.ScrollDown()
			}
			
		case 1: // Editor
			// Editor handles its own keys
			
		case 2: // Sessions
			// Sessions handles its own keys
		}
		
	case chat.MessageSubmitMsg:
		// Handle message submission
		if !p.isProcessing {
			p.isProcessing = true
			
			// Add user message
			userMsg := chat.Message{
				ID:      fmt.Sprintf("msg-%d", time.Now().Unix()),
				Role:    "user",
				Content: msg.Content,
			}
			
			cmds = append(cmds, func() tea.Msg {
				return chat.MessageSentMsg{
					SessionID: p.currentSessionID,
					Message:   userMsg,
				}
			})
			
			// Add assistant message placeholder
			assistantMsg := chat.Message{
				ID:   fmt.Sprintf("msg-%d", time.Now().Unix()+1),
				Role: "assistant",
				Content: "",
			}
			
			cmds = append(cmds, func() tea.Msg {
				return chat.MessageSentMsg{
					SessionID: p.currentSessionID,
					Message:   assistantMsg,
				}
			})
			
			// Process with LLM
			cmds = append(cmds, p.processMessage(msg.Content, msg.Attachments))
			
			// Update session info
			p.sessions.UpdateSession(p.currentSessionID, func(s *chat.Session) {
				s.LastMessage = msg.Content
				s.UpdatedAt = time.Now()
				s.MessageCount++
				
				// Update title if it's the first message
				if s.MessageCount == 1 {
					// Use first few words as title
					words := strings.Fields(msg.Content)
					if len(words) > 5 {
						s.Title = strings.Join(words[:5], " ") + "..."
					} else {
						s.Title = msg.Content
					}
				}
			})
		}
		
	case chat.SessionSelectedMsg:
		// Switch session
		p.currentSessionID = msg.SessionID
		p.chatView.SetSessionID(msg.SessionID)
		// TODO: Load session history from storage
		
	case chat.SessionCreatedMsg:
		// Switch to new session
		p.currentSessionID = msg.Session.ID
		p.chatView.SetSessionID(msg.Session.ID)
		
	case streamCompleteMsg:
		p.isProcessing = false
		if msg.error != nil {
			p.lastError = msg.error
			// Update last message with error
			cmds = append(cmds, func() tea.Msg {
				return chat.StreamUpdateMsg{
					SessionID: p.currentSessionID,
					Content:   fmt.Sprintf("\n\nError: %v", msg.error),
					Done:      true,
				}
			})
		}
		
	case dialog.FileSelectedMsg:
		// Add attachments
		for _, path := range msg.Paths {
			p.editor.AddAttachment(path)
		}
		
	case dialog.ModelSelectedMsg:
		// Update current model
		p.currentModel = fmt.Sprintf("%s/%s", msg.Provider, msg.Model)
		p.app.SetCurrentModel(msg.Provider, msg.Model)
		
		// Update context window for new model
		availableModels := p.app.GetAvailableModels()
		for _, model := range availableModels {
			if model.Provider == msg.Provider && model.Name == msg.Model {
				p.maxContextTokens = model.Info.ContextWindow
				break
			}
		}
	}
	
	// Update components
	var cmd tea.Cmd
	
	// Update chat view
	newChat, cmd := p.chatView.Update(msg)
	if c, ok := newChat.(*chat.ChatModel); ok {
		p.chatView = c
	}
	cmds = append(cmds, cmd)
	
	// Update editor
	newEditor, cmd := p.editor.Update(msg)
	if e, ok := newEditor.(*chat.EditorModel); ok {
		p.editor = e
	}
	cmds = append(cmds, cmd)
	
	// Update sessions
	newSessions, cmd := p.sessions.Update(msg)
	if s, ok := newSessions.(*chat.SessionsModel); ok {
		p.sessions = s
	}
	cmds = append(cmds, cmd)
	
	return p, tea.Batch(cmds...)
}

func (p *ChatPage) View() string {
	// Calculate dimensions
	sidebarWidth := 0
	if p.showSidebar {
		sidebarWidth = int(float64(p.width) * p.splitRatio)
		if sidebarWidth < 30 {
			sidebarWidth = 30
		}
	}
	
	mainWidth := p.width - sidebarWidth
	chatHeight := p.height - 8 // Reserve space for editor and status
	
	// Build layout
	var sections []string
	
	// Header
	header := p.renderHeader()
	sections = append(sections, header)
	
	// Main content area
	var mainContent string
	
	// Chat view
	p.chatView.SetSize(mainWidth, chatHeight - 2)
	chatContent := p.chatView.View()
	
	// Editor
	p.editor.SetWidth(mainWidth)
	editorContent := p.editor.View()
	
	// Combine chat and editor
	mainContent = lipgloss.JoinVertical(
		lipgloss.Top,
		chatContent,
		lipgloss.NewStyle().Height(1).Render(""), // Spacer
		editorContent,
	)
	
	// Add sidebar if visible
	if p.showSidebar {
		p.sessions.SetSize(sidebarWidth - 1, p.height - 3)
		sidebar := p.sessions.View()
		
		// Join horizontally
		mainContent = lipgloss.JoinHorizontal(
			lipgloss.Left,
			sidebar,
			lipgloss.NewStyle().Width(1).Render(""), // Spacer
			mainContent,
		)
	}
	
	sections = append(sections, mainContent)
	
	// Status bar
	status := p.renderStatusBar()
	sections = append(sections, status)
	
	return strings.Join(sections, "\n")
}

func (p *ChatPage) renderHeader() string {
	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(p.theme.TextEmphasized()).
		Width(p.width / 3).
		Align(lipgloss.Left)
		
	title := titleStyle.Render("CodeForge Chat")
	
	// Model info
	modelStyle := lipgloss.NewStyle().
		Foreground(p.theme.TextMuted()).
		Width(p.width / 3).
		Align(lipgloss.Center)
		
	model := modelStyle.Render(fmt.Sprintf("Model: %s", p.currentModel))
	
	// Session info
	sessionStyle := lipgloss.NewStyle().
		Foreground(p.theme.TextMuted()).
		Width(p.width / 3).
		Align(lipgloss.Right)
		
	sessionCount := p.sessions.GetSessionCount()
	session := sessionStyle.Render(fmt.Sprintf("Sessions: %d", sessionCount))
	
	// Combine
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		title,
		model,
		session,
	)
	
	// Add border
	return lipgloss.NewStyle().
		Background(p.theme.Background()).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(p.theme.Primary()).
		Width(p.width).
		Render(header)
}

func (p *ChatPage) renderStatusBar() string {
	// Left side - Focus indicator
	focusText := ""
	switch p.focusIndex {
	case 0:
		focusText = "Chat View"
	case 1:
		focusText = "Message Input"
	case 2:
		focusText = "Sessions"
	}
	
	focusStyle := lipgloss.NewStyle().
		Foreground(p.theme.Accent())
	focus := focusStyle.Render(fmt.Sprintf("Focus: %s", focusText))
	
	// Processing indicator
	var status string
	if p.isProcessing {
		status = lipgloss.NewStyle().
			Foreground(p.theme.Warning()).
			Render("Processing...")
	} else if p.lastError != nil {
		status = lipgloss.NewStyle().
			Foreground(p.theme.Error()).
			Render("Error")
	} else {
		status = lipgloss.NewStyle().
			Foreground(p.theme.Success()).
			Render("Ready")
	}
	
	// Help
	helpStyle := lipgloss.NewStyle().
		Foreground(p.theme.TextMuted())
	help := helpStyle.Render("Tab: Switch Focus • Ctrl+B: Toggle Sidebar • ?: Help")
	
	// Left side combined
	leftSide := lipgloss.JoinHorizontal(
		lipgloss.Top,
		focus,
		lipgloss.NewStyle().Width(2).Render(""),
		status,
		lipgloss.NewStyle().Width(2).Render(""),
		help,
	)
	
	// Right side - Model info and context usage
	modelInfo := p.getModelInfo()
	contextUsage := p.getContextUsage()
	
	modelStyle := lipgloss.NewStyle().
		Foreground(p.theme.Text())
	contextStyle := lipgloss.NewStyle().
		Foreground(p.theme.TextMuted())
	
	rightContent := fmt.Sprintf("%s • %s", 
		modelStyle.Render(modelInfo),
		contextStyle.Render(contextUsage))
	
	rightSide := lipgloss.NewStyle().
		Background(p.theme.Background()).
		Width(lipgloss.Width(rightContent)).
		Align(lipgloss.Right).
		Render(rightContent)
	
	// Calculate spacing
	leftWidth := lipgloss.Width(leftSide)
	rightWidth := lipgloss.Width(rightSide)
	spacerWidth := p.width - leftWidth - rightWidth - 2
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	
	// Combine with spacer
	statusBar := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftSide,
		lipgloss.NewStyle().Width(spacerWidth).Render(""),
		rightSide,
	)
	
	return lipgloss.NewStyle().
		Background(p.theme.BackgroundSecondary()).
		Width(p.width).
		Render(statusBar)
}

func (p *ChatPage) updateLayout() {
	// This is called when window size changes
	// Components will be updated in View()
}

func (p *ChatPage) cycleFocus(direction int) {
	maxFocus := 1
	if p.showSidebar {
		maxFocus = 2
	}
	
	p.focusIndex = (p.focusIndex + direction + maxFocus + 1) % (maxFocus + 1)
}

func (p *ChatPage) updateFocus() tea.Cmd {
	// Blur all
	p.editor.Blur()
	p.sessions.Blur()
	
	// Focus current
	switch p.focusIndex {
	case 1:
		return p.editor.Focus()
	case 2:
		p.sessions.Focus()
	}
	
	return nil
}

// debugLog writes debug information to a log file
func (p *ChatPage) debugLog(format string, args ...interface{}) {
	// Create log directory if it doesn't exist
	debugDir := filepath.Join(p.app.WorkspaceRoot, "log")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return
	}
	
	// Open or create log file
	logPath := filepath.Join(debugDir, "tui-debug.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()
	
	// Create logger
	logger := log.New(file, "", log.LstdFlags)
	logger.Printf(format, args...)
}

// processMessage sends the message to the LLM and streams the response
func (p *ChatPage) processMessage(content string, attachments []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Debug log the start of processing
		p.debugLog("=== NEW MESSAGE REQUEST ===")
		p.debugLog("Session ID: %s", p.currentSessionID)
		p.debugLog("Model: %s", p.currentModel)
		p.debugLog("Original Content: %s", content)
		p.debugLog("Attachments: %v", attachments)
		
		// Add file contents if attachments exist
		fullContent := content
		if len(attachments) > 0 {
			fullContent += "\n\nAttached files:\n"
			for _, path := range attachments {
				// Read file content using app's file operation manager
				fileOps := p.app.GetFileOperationManager()
				if fileOps != nil {
					fileContent, err := fileOps.ReadFile(ctx, &permissions.FileOperationRequest{
						SessionID: p.currentSessionID,
						Operation: "read",
						Path:      path,
					})
					if err != nil {
						fullContent += fmt.Sprintf("\n--- %s ---\nError reading file: %v\n", path, err)
					} else {
						fullContent += fmt.Sprintf("\n--- %s ---\n%s\n", path, fileContent.Content)
					}
				} else {
					// Fallback to direct file reading
					data, err := os.ReadFile(path)
					if err != nil {
						fullContent += fmt.Sprintf("\n--- %s ---\nError reading file: %v\n", path, err)
					} else {
						fullContent += fmt.Sprintf("\n--- %s ---\n%s\n", path, string(data))
					}
				}
			}
		}
		
		// Debug log the full content being sent
		p.debugLog("Full Content to Model:\n%s", fullContent)
		p.debugLog("Content Length: %d characters", len(fullContent))
		
		// Create a channel for streaming responses
		streamChan := make(chan string, 100)
		errorChan := make(chan error, 1)
		doneChan := make(chan bool, 1)
		
		// Start processing in background
		go func() {
			defer close(streamChan)
			defer close(doneChan)
			
			// Use the new context-aware streaming method
			p.debugLog("Using context-aware streaming API")
			processedCtx, respStreamChan, err := p.app.ProcessChatMessageWithStream(
				ctx,
				p.currentSessionID,
				fullContent,
				p.currentModel,
			)
			
			if err != nil {
				p.debugLog("Context-aware streaming API error: %v", err)
				errorChan <- err
				return
			}
			
			// Update context display with processed context
			if processedCtx != nil {
				p.lastProcessedContext = processedCtx
				p.debugLog("Context processed - Final tokens: %d, Compression ratio: %.2f%%",
					processedCtx.FinalTokens, (1.0-processedCtx.CompressionRatio)*100)
				
				// Update context token counts
				p.currentContextTokens = processedCtx.FinalTokens
			}
			
			p.debugLog("Streaming started, reading chunks...")
			totalChunks := 0
			totalLength := 0
			
			// Forward chunks from the response stream channel
			for chunk := range respStreamChan {
				totalChunks++
				totalLength += len(chunk)
				streamChan <- chunk
			}
			
			p.debugLog("Streaming completed - Total chunks: %d, Total characters: %d", totalChunks, totalLength)
			p.debugLog("=== END MESSAGE REQUEST ===\n")
			doneChan <- true
		}()
		
		// Return command that processes stream updates
		return func() tea.Msg {
			for {
				select {
				case chunk, ok := <-streamChan:
					if !ok {
						return streamCompleteMsg{}
					}
					if chunk != "" {
						// Send non-blocking update
						go func(c string) {
							p.app.EventManager.PublishSystem(events.SystemHealthCheck, events.SystemEventPayload{
								Component: "tui_chat",
								Status:    "streaming",
								Message:   "Stream chunk received",
								Metadata: map[string]interface{}{
									"session_id": p.currentSessionID,
									"chunk":      c,
								},
							})
						}(chunk)
						
						return chat.StreamUpdateMsg{
							SessionID: p.currentSessionID,
							Content:   chunk,
							Done:      false,
						}
					}
					
				case err := <-errorChan:
					return streamCompleteMsg{error: err}
					
				case <-doneChan:
					return tea.Batch(
						func() tea.Msg {
							return chat.StreamUpdateMsg{
								SessionID: p.currentSessionID,
								Content:   "",
								Done:      true,
							}
						},
						func() tea.Msg {
							return streamCompleteMsg{}
						},
					)
					
				case <-time.After(100 * time.Millisecond):
					// Continue checking
				}
			}
		}
	}
}

// Messages
type streamCompleteMsg struct {
	error error
}

// getModelInfo returns a formatted string with the current model
func (p *ChatPage) getModelInfo() string {
	// Extract just the model name from the full path
	parts := strings.Split(p.currentModel, "/")
	modelName := p.currentModel
	if len(parts) > 1 {
		modelName = parts[1]
	}
	
	// Shorten long model names
	if len(modelName) > 25 {
		modelName = modelName[:22] + "..."
	}
	
	return modelName
}

// getContextUsage returns a formatted string showing context usage
func (p *ChatPage) getContextUsage() string {
	// Use actual token count from last processed context if available
	totalTokens := 0
	
	if p.lastProcessedContext != nil {
		// Use the actual final token count from the processed context
		totalTokens = p.lastProcessedContext.FinalTokens
		p.currentContextTokens = totalTokens
		
		// Debug log the context details
		p.debugLog("Context Usage - Final Tokens: %d, Original: %d, Compression: %.2f%%", 
			p.lastProcessedContext.FinalTokens,
			p.lastProcessedContext.OriginalTokens,
			(1.0-p.lastProcessedContext.CompressionRatio)*100)
	} else {
		// Estimate if no processed context yet
		// The context includes:
		// - System prompt
		// - Project overview (AGENTS.md)  
		// - Repository map
		// - Relevant symbols
		// - Summarized conversation
		
		systemPromptTokens := 50        // "You are CodeForge, an AI coding assistant"
		projectOverviewTokens := 2000   // AGENTS.md content
		repoMapTokens := 1000          // Repository structure
		symbolsTokens := 500           // Relevant symbols
		conversationTokens := 0         // No conversation yet
		
		totalTokens = systemPromptTokens + projectOverviewTokens + repoMapTokens + symbolsTokens + conversationTokens
		p.currentContextTokens = totalTokens
	}
	
	// Format the display
	percentage := 0
	if p.maxContextTokens > 0 {
		percentage = (p.currentContextTokens * 100) / p.maxContextTokens
	}
	
	// Color code based on usage
	var percentageStr string
	if percentage > 90 {
		percentageStr = lipgloss.NewStyle().
			Foreground(p.theme.Error()).
			Render(fmt.Sprintf("%d%%", percentage))
	} else if percentage > 75 {
		percentageStr = lipgloss.NewStyle().
			Foreground(p.theme.Warning()).
			Render(fmt.Sprintf("%d%%", percentage))
	} else {
		percentageStr = fmt.Sprintf("%d%%", percentage)
	}
	
	// Use K notation for large numbers
	currentK := float64(p.currentContextTokens) / 1000.0
	maxK := float64(p.maxContextTokens) / 1000.0
	
	if p.maxContextTokens >= 1000 {
		return fmt.Sprintf("%.1fK/%.0fK (%s)", currentK, maxK, percentageStr)
	}
	
	return fmt.Sprintf("%d/%d (%s)", p.currentContextTokens, p.maxContextTokens, percentageStr)
}

// Key bindings
type chatKeyMap struct {
	ToggleSidebar key.Binding
	FocusNext     key.Binding
	FocusPrev     key.Binding
	NewSession    key.Binding
	ClearChat     key.Binding
	ScrollUp      key.Binding
	ScrollDown    key.Binding
}

var chatKeys = chatKeyMap{
	ToggleSidebar: key.NewBinding(
		key.WithKeys("ctrl+b"),
		key.WithHelp("ctrl+b", "toggle sidebar"),
	),
	FocusNext: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next focus"),
	),
	FocusPrev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "previous focus"),
	),
	NewSession: key.NewBinding(
		key.WithKeys("ctrl+n"),
		key.WithHelp("ctrl+n", "new session"),
	),
	ClearChat: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "clear chat"),
	),
	ScrollUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "scroll up"),
	),
	ScrollDown: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "scroll down"),
	),
}