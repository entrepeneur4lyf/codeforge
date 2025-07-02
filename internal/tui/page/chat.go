package page

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/app"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/components/chat"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/components/dialogs"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/themes"
)

// ChatPage represents the main chat interface
type ChatPage struct {
	app         *app.App
	theme       themes.Theme
	
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
}

// NewChatPage creates a new chat page
func NewChatPage(app *app.App, theme themes.Theme) *ChatPage {
	// Create initial session
	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	
	// Get current model
	provider, model := app.GetCurrentModel()
	currentModel := fmt.Sprintf("%s/%s", provider, model)
	
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
		
	case dialogs.FileSelectedMsg:
		// Add attachments
		for _, path := range msg.Paths {
			p.editor.AddAttachment(path)
		}
		
	case dialogs.ModelSelectedMsg:
		// Update current model
		p.currentModel = fmt.Sprintf("%s/%s", msg.Provider, msg.Model)
		p.app.SetCurrentModel(msg.Provider, msg.Model)
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
		p.theme.Base().Height(1).Render(""), // Spacer
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
			p.theme.Base().Width(1).Render(""), // Spacer
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
	titleStyle := p.theme.DialogTitleStyle().
		Width(p.width / 3).
		Align(lipgloss.Left)
		
	title := titleStyle.Render("CodeForge Chat")
	
	// Model info
	modelStyle := p.theme.SecondaryText().
		Width(p.width / 3).
		Align(lipgloss.Center)
		
	model := modelStyle.Render(fmt.Sprintf("Model: %s", p.currentModel))
	
	// Session info
	sessionStyle := p.theme.MutedText().
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
	return p.theme.Base().
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(p.theme.Primary()).
		Width(p.width).
		Render(header)
}

func (p *ChatPage) renderStatusBar() string {
	// Focus indicator
	focusText := ""
	switch p.focusIndex {
	case 0:
		focusText = "Chat"
	case 1:
		focusText = "Editor"
	case 2:
		focusText = "Sessions"
	}
	
	focusStyle := p.theme.StatusKey()
	focus := focusStyle.Render(fmt.Sprintf("Focus: %s", focusText))
	
	// Processing indicator
	var status string
	if p.isProcessing {
		status = p.theme.WarningText().Render("Processing...")
	} else if p.lastError != nil {
		status = p.theme.ErrorText().Render("Error")
	} else {
		status = p.theme.SuccessText().Render("Ready")
	}
	
	// Help
	helpStyle := p.theme.MutedText()
	help := helpStyle.Render("Tab: Switch Focus • Ctrl+B: Toggle Sidebar • ?: Help")
	
	// Combine
	statusBar := lipgloss.JoinHorizontal(
		lipgloss.Top,
		focus,
		p.theme.Base().Width(2).Render(""),
		status,
		p.theme.Base().Width(2).Render(""),
		help,
	)
	
	return p.theme.StatusBar().
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

// processMessage sends the message to the LLM and streams the response
func (p *ChatPage) processMessage(content string, attachments []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		
		// Add file contents if attachments exist
		fullContent := content
		if len(attachments) > 0 {
			fullContent += "\n\nAttached files:\n"
			for _, path := range attachments {
				// Read file content
				// TODO: Use app's file operation manager
				fullContent += fmt.Sprintf("\n--- %s ---\n", path)
				// Add file content here
			}
		}
		
		// Process with app
		response, err := p.app.ProcessChatMessage(ctx, p.currentSessionID, fullContent, p.currentModel)
		if err != nil {
			return streamCompleteMsg{error: err}
		}
		
		// For now, send complete response
		// TODO: Implement proper streaming
		return tea.Batch(
			func() tea.Msg {
				return chat.StreamUpdateMsg{
					SessionID: p.currentSessionID,
					Content:   response,
					Done:      false,
				}
			},
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
	}
}

// Messages
type streamCompleteMsg struct {
	error error
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