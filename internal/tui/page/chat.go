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
	"github.com/entrepeneur4lyf/codeforge/internal/tui/components/status"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/components/toast"
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
	toolbar     *chat.ToolbarModel
	toastMgr    *toast.ToastManager
	statusBar   *status.Model
	
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
	
	// Stream state
	streamChan  chan string
	errorChan   chan error
	doneChan    chan bool
	
	// Mouse state
	hoveredButton int // -1 means no button is hovered
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
		showSidebar:      false,  // Hide sidebar by default
		chatView:         chat.NewChatModel(theme, sessionID),
		editor:           chat.NewEditorModel(theme),
		sessions:         chat.NewSessionsModel(theme),
		toolbar:          chat.NewToolbarModel(theme),
		toastMgr:         toast.NewToastManager(theme),
		statusBar:        status.NewStatusBar(app, theme),
		focusIndex:       1, // Start with editor focused
		currentSessionID: sessionID,
		currentModel:     currentModel,
		maxContextTokens: maxContext,
		currentContextTokens: 0,
		hoveredButton: -1,
	}
}

func (p *ChatPage) Init() tea.Cmd {
	// Initialize components
	cmds := []tea.Cmd{
		p.chatView.Init(),
		p.editor.Init(),
		p.sessions.Init(),
		p.toolbar.Init(),
		p.toastMgr.Init(),
		p.statusBar.Init(),
	}
	
	// Load existing sessions from database and create initial session if none exist
	cmds = append(cmds, func() tea.Msg {
		if p.app != nil && p.app.ChatStore != nil {
			ctx := context.Background()
			sessions, err := p.app.GetChatSessions(ctx, "", 20, 0)
			if err != nil {
				log.Printf("Warning: Failed to load sessions: %v", err)
			} else if len(sessions) > 0 {
				// Convert storage sessions to TUI sessions
				tuiSessions := make([]chat.Session, len(sessions))
				for i, s := range sessions {
					tuiSessions[i] = chat.Session{
						ID:           s.ID,
						Title:        s.Title,
						LastMessage:  s.LastMessage,
						UpdatedAt:    s.UpdatedAt,
						MessageCount: s.MessageCount,
					}
				}
				
				// Use the most recent session as current
				p.currentSessionID = sessions[0].ID
				
				// Load sessions into the sessions model
				p.sessions.SetSessions(tuiSessions)
				
				log.Printf("Loaded %d existing sessions", len(sessions))
				return chat.SessionSelectedMsg{SessionID: sessions[0].ID}
			}
		}
		
		// No existing sessions, create initial session
		initialSession := chat.Session{
			ID:           p.currentSessionID,
			Title:        "New Chat",
			UpdatedAt:    time.Now(),
			MessageCount: 0,
		}
		
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
		
	case tea.MouseMsg:
		// Status bar is at the bottom, occupying the last 2 lines
		// The toolbar buttons are on the first line of the status bar
		statusBarStartY := p.height - 2
		
		// Check if mouse is on the toolbar line of status bar
		if msg.Y == statusBarStartY {
			// Calculate which button was clicked based on X position
			// Each button is emoji (2 cells) + padding (2) = 4 cells for emojis
			buttonWidth := 4
			buttonIndex := msg.X / buttonWidth
			
			if buttonIndex >= 0 && buttonIndex < 5 { // We have 5 buttons
				p.hoveredButton = buttonIndex
				
				if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
					// Handle button click
					switch buttonIndex {
					case 0: // New Chat
						newSession := chat.Session{
							ID:           fmt.Sprintf("session-%d", time.Now().Unix()),
							Title:        "New Chat",
							UpdatedAt:    time.Now(),
							MessageCount: 0,
						}
						cmds = append(cmds, func() tea.Msg {
							return chat.SessionCreatedMsg{Session: newSession}
						})
						// Show toast
						cmds = append(cmds, toast.NewInfoToast(
							"New chat created",
							p.theme,
							toast.WithDuration(2*time.Second),
						))
					case 1: // Clear Chat
						p.chatView.ClearMessages()
						cmds = append(cmds, toast.NewSuccessToast(
							"Chat cleared",
							p.theme,
							toast.WithDuration(2*time.Second),
						))
					case 2: // Attach File
						cmds = append(cmds, func() tea.Msg {
							return tea.KeyMsg{Type: tea.KeyCtrlF}
						})
					case 3: // File Picker
						cmds = append(cmds, func() tea.Msg {
							return tea.KeyMsg{Type: tea.KeyCtrlP}
						})
					case 4: // Select Model
						cmds = append(cmds, func() tea.Msg {
							return tea.KeyMsg{Type: tea.KeyCtrlM}
						})
					}
				}
			} else {
				p.hoveredButton = -1
			}
		} else {
			p.hoveredButton = -1
		}
		
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
			
			// Start streaming in chat view
			p.chatView.StartStreaming()
			
			// Process with LLM
			cmds = append(cmds, p.processMessage(msg.Content, msg.Attachments))
			
			// Show processing toast
			cmds = append(cmds, toast.NewInfoToast(
				"Processing message...",
				p.theme,
				toast.WithDuration(2*time.Second),
			))
			
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
		
		// Load session history from storage
		if p.app != nil {
			go func() {
				ctx := context.Background()
				messages, err := p.app.GetLatestChatMessages(ctx, msg.SessionID, 50)
				if err != nil {
					log.Printf("Warning: Failed to load session history: %v", err)
					return
				}
				
				// Convert storage messages to TUI messages
				tuiMessages := make([]chat.Message, len(messages))
				for i, storageMsg := range messages {
					tuiMessages[i] = chat.Message{
						ID:      storageMsg.ID,
						Role:    storageMsg.Role,
						Content: storageMsg.Content,
					}
				}
				
				// Load messages into chat view
				p.chatView.LoadMessages(tuiMessages)
				
				log.Printf("Loaded %d messages for session %s", len(messages), msg.SessionID)
			}()
		}
		
	case chat.SessionCreatedMsg:
		// Switch to new session
		p.currentSessionID = msg.Session.ID
		p.chatView.SetSessionID(msg.Session.ID)
		
		// Show info toast
		cmds = append(cmds, toast.NewInfoToast(
			"New chat session created",
			p.theme,
			toast.WithDuration(2*time.Second),
		))
		
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
			// Show error toast
			cmds = append(cmds, toast.NewErrorToast(
				fmt.Sprintf("Failed to process message: %v", msg.error),
				p.theme,
				toast.WithTitle("Error"),
				toast.WithDuration(5*time.Second),
			))
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
		
		// Show success toast
		cmds = append(cmds, toast.NewSuccessToast(
			fmt.Sprintf("Model changed to %s", msg.Model),
			p.theme,
			toast.WithTitle("Model Updated"),
			toast.WithDuration(3*time.Second),
		))
		
	case chat.ToolbarClickMsg:
		// Handle toolbar button clicks
		switch msg.Action {
		case chat.ActionNewChat:
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
			
		case chat.ActionClearChat:
			p.chatView.ClearMessages()
			return p, toast.NewSuccessToast("Chat cleared", p.theme, toast.WithDuration(3*time.Second))
			
		case chat.ActionAttachFile:
			// Open file dialog
			return p, func() tea.Msg {
				return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{}, Alt: false, Paste: false}
			}
			
		case chat.ActionFilePicker:
			// Open file picker
			return p, func() tea.Msg {
				return tea.KeyMsg{Type: tea.KeyCtrlP}
			}
			
		case chat.ActionModelSelect:
			// Open model selection
			return p, func() tea.Msg {
				return tea.KeyMsg{Type: tea.KeyCtrlM}
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
	
	// Update toolbar
	newToolbar, cmd := p.toolbar.Update(msg)
	if t, ok := newToolbar.(*chat.ToolbarModel); ok {
		p.toolbar = t
	}
	cmds = append(cmds, cmd)
	
	// Update toast manager
	_, cmd = p.toastMgr.Update(msg)
	cmds = append(cmds, cmd)
	
	// Update status bar
	newStatus, cmd := p.statusBar.Update(msg)
	if s, ok := newStatus.(*status.Model); ok {
		p.statusBar = s
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
	// Reserve space: header(2) + editor(3) + status(2) + spacers(1) = 8
	chatHeight := p.height - 8
	
	// Build layout
	var sections []string
	
	// Header
	header := p.renderHeader()
	sections = append(sections, header)
	
	// Main content area
	var mainContent string
	
	// Chat view
	p.chatView.SetSize(mainWidth, chatHeight)
	chatContent := p.chatView.View()
	
	// Editor
	p.editor.SetWidth(mainWidth - 2) // Account for borders
	p.editor.SetHeight(3)  // Fixed height for editor
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
	
	// Status bar with toolbar buttons
	p.statusBar.SetWidth(p.width)
	p.statusBar.SetCustomLeft(p.renderToolbarButtons())
	
	// Update session info if we have context data
	if p.lastProcessedContext != nil {
		tokens := float64(p.lastProcessedContext.FinalTokens)
		cost := p.calculateSessionCost(tokens)
		p.statusBar.UpdateSessionInfo(tokens, cost)
	}
	
	status := p.statusBar.View()
	sections = append(sections, status)
	
	// Join all sections
	result := strings.Join(sections, "\n")
	
	// Apply toast overlay if there are any toasts
	result = p.toastMgr.RenderOverlay(result)
	
	return result
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

// calculateSessionCost estimates the cost based on token usage
func (p *ChatPage) calculateSessionCost(tokens float64) float64 {
	// Get current model info from ModelInfo which contains pricing
	availableModels := p.app.GetAvailableModels()
	for _, model := range availableModels {
		provider, modelName := p.app.GetCurrentModel()
		if model.Provider == provider && model.Name == modelName {
			// Use pricing from ModelInfo
			// Estimate cost: assume 80% input, 20% output for mixed usage
			inputCost := (tokens * 0.8) * model.Info.InputPrice / 1_000_000
			outputCost := (tokens * 0.2) * model.Info.OutputPrice / 1_000_000
			return inputCost + outputCost
		}
	}
	// Default estimate if model not found
	return tokens * 0.000001 // $1 per million tokens as fallback
}

func (p *ChatPage) renderStatusBar() string {
	// Toolbar buttons first
	toolbarButtons := p.renderToolbarButtons()
	
	// Focus indicator
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
	help := helpStyle.Render("Tab: Switch Focus • Ctrl+B: Toggle Sidebar • ?: Help • Enter: Send")
	
	// Left side combined
	leftSide := lipgloss.JoinHorizontal(
		lipgloss.Top,
		toolbarButtons,
		lipgloss.NewStyle().Width(2).Render(""),
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
			
			// Publish typing start event
			if p.app.EventManager != nil {
				p.app.EventManager.PublishChat(events.ChatTypingStart, events.ChatEventPayload{
					SessionID: p.currentSessionID,
					Role:      "assistant",
					Model:     p.currentModel,
				}, events.WithSessionID(p.currentSessionID))
			}
			
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
			
			// Publish typing stop event
			if p.app.EventManager != nil {
				p.app.EventManager.PublishChat(events.ChatTypingStop, events.ChatEventPayload{
					SessionID: p.currentSessionID,
					Role:      "assistant",
					Model:     p.currentModel,
				}, events.WithSessionID(p.currentSessionID))
			}
			
			doneChan <- true
		}()
		
		// Store stream state
		p.streamChan = streamChan
		p.errorChan = errorChan
		p.doneChan = doneChan
		
		// Return a ticker that checks for updates
		return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
			select {
			case chunk, ok := <-streamChan:
				if !ok {
					return streamCompleteMsg{}
				}
				return chat.StreamUpdateMsg{
					SessionID: p.currentSessionID,
					Content:   chunk,
					Done:      false,
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
					return nil
				}
			})
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

// renderToolbarButtons renders clickable buttons in the status bar
func (p *ChatPage) renderToolbarButtons() string {
	buttons := []struct {
		icon    string
		tooltip string
		action  func() tea.Cmd
	}{
		{"➕", "New Chat", func() tea.Cmd {
			return func() tea.Msg {
				newSession := chat.Session{
					ID:           fmt.Sprintf("session-%d", time.Now().Unix()),
					Title:        "New Chat",
					UpdatedAt:    time.Now(),
					MessageCount: 0,
				}
				return chat.SessionCreatedMsg{Session: newSession}
			}
		}},
		{"🗑", "Clear Chat", func() tea.Cmd {
			p.chatView.ClearMessages()
			return nil
		}},
		{"📎", "Attach File", func() tea.Cmd {
			return func() tea.Msg {
				return tea.KeyMsg{Type: tea.KeyCtrlF}
			}
		}},
		{"📁", "File Picker", func() tea.Cmd {
			return func() tea.Msg {
				return tea.KeyMsg{Type: tea.KeyCtrlP}
			}
		}},
		{"🤖", "Select Model", func() tea.Cmd {
			return func() tea.Msg {
				return tea.KeyMsg{Type: tea.KeyCtrlM}
			}
		}},
	}
	
	var renderedButtons []string
	for i, btn := range buttons {
		style := lipgloss.NewStyle().
			Padding(0, 1).
			Background(p.theme.BackgroundSecondary()).
			Foreground(p.theme.Text())
		
		// Highlight hovered button
		if i == p.hoveredButton {
			style = style.
				Background(p.theme.Primary()).
				Foreground(p.theme.Background()).
				Bold(true)
		}
		
		renderedButtons = append(renderedButtons, style.Render(btn.icon))
	}
	
	// Add tooltip if hovering
	result := lipgloss.JoinHorizontal(lipgloss.Left, renderedButtons...)
	
	if p.hoveredButton >= 0 && p.hoveredButton < len(buttons) {
		tooltipStyle := lipgloss.NewStyle().
			Foreground(p.theme.TextMuted()).
			Italic(true).
			PaddingLeft(1)
		
		tooltip := tooltipStyle.Render(buttons[p.hoveredButton].tooltip)
		result = lipgloss.JoinHorizontal(lipgloss.Left, result, tooltip)
	}
	
	return result
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