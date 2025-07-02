package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrepeneur4lyf/codeforge/internal/app"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/components/dialogs"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/layout"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/page"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/themes"
)

type sessionState int

const (
	stateChat sessionState = iota
	stateModelSelect
	stateHelp
	stateFileSelect
)

type Model struct {
	app           *app.App
	chatModel     tea.Model
	state         sessionState
	width         int
	height        int
	theme         themes.Theme
	
	// Dialogs
	modelDialog   tea.Model
	helpDialog    tea.Model
	fileDialog    tea.Model
	
	// Dialog states
	showModelDialog bool
	showHelpDialog  bool
	showFileDialog  bool
	
	// Current session
	currentSessionID string
	
	// Error state
	err           error
}

func New(application *app.App) *Model {
	theme := themes.NewDefaultTheme()
	
	return &Model{
		app:    application,
		state:  stateChat,
		theme:  theme,
	}
}

func (m *Model) Init() tea.Cmd {
	// Initialize components
	m.chatModel = m.createChatModel()
	m.modelDialog = dialogs.NewModelDialog(m.app, m.theme)
	m.helpDialog = dialogs.NewHelpDialog(m.theme)
	m.fileDialog = dialogs.NewFileDialog(m.theme)
	
	// Initialize chat component
	return m.chatModel.Init()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	
	// Handle global keys first
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		
		// Update all components with new size
		if m.chatModel != nil {
			newModel, cmd := m.chatModel.Update(msg)
			m.chatModel = newModel
			cmds = append(cmds, cmd)
		}
		
		return m, tea.Batch(cmds...)
		
	case tea.KeyMsg:
		// Check if any dialog is open
		if m.showModelDialog || m.showHelpDialog || m.showFileDialog {
			return m.updateDialog(msg)
		}
		
		// Global keybindings
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
			
		case key.Matches(msg, keys.Help):
			m.showHelpDialog = true
			return m, nil
			
		case key.Matches(msg, keys.ModelSelect):
			m.showModelDialog = true
			return m, nil
			
		case key.Matches(msg, keys.FileSelect):
			m.showFileDialog = true
			return m, nil
		}
		
	case error:
		m.err = msg
		return m, nil
		
	case dialogs.ModelSelectedMsg:
		// Handle model selection
		m.showModelDialog = false
		// Update current model in app
		if err := m.app.SetCurrentModel(msg.Provider, msg.Model); err != nil {
			m.err = err
		}
		return m, nil
		
	case dialogs.DialogCloseMsg:
		// Close any open dialog
		m.showModelDialog = false
		m.showHelpDialog = false
		m.showFileDialog = false
		return m, nil
	}
	
	// Route to current view
	switch m.state {
	case stateChat:
		newModel, cmd := m.chatModel.Update(msg)
		m.chatModel = newModel
		return m, cmd
	}
	
	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}
	
	// Main content
	var content string
	switch m.state {
	case stateChat:
		content = m.chatModel.View()
	}
	
	// Apply theme styling
	styledContent := m.theme.Base().
		Width(m.width).
		Height(m.height).
		Render(content)
	
	// Overlay dialogs
	if m.showModelDialog {
		return layout.PlaceOverlay(
			m.width, m.height,
			m.modelDialog.View(),
			styledContent,
			layout.Center,
		)
	}
	
	if m.showHelpDialog {
		return layout.PlaceOverlay(
			m.width, m.height,
			m.helpDialog.View(),
			styledContent,
			layout.Center,
		)
	}
	
	if m.showFileDialog {
		return layout.PlaceOverlay(
			m.width, m.height,
			m.fileDialog.View(),
			styledContent,
			layout.Center,
		)
	}
	
	// Error overlay
	if m.err != nil {
		errorView := m.renderError()
		return layout.PlaceOverlay(
			m.width, m.height,
			errorView,
			styledContent,
			layout.Bottom,
		)
	}
	
	return styledContent
}

func (m *Model) updateDialog(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// ESC closes any dialog
	if key.Matches(msg, keys.Cancel) {
		m.showModelDialog = false
		m.showHelpDialog = false
		m.showFileDialog = false
		return m, nil
	}
	
	// Route to active dialog
	if m.showModelDialog {
		newModel, cmd := m.modelDialog.Update(msg)
		m.modelDialog = newModel
		return m, cmd
	}
	
	if m.showHelpDialog {
		newModel, cmd := m.helpDialog.Update(msg)
		m.helpDialog = newModel
		return m, cmd
	}
	
	if m.showFileDialog {
		newModel, cmd := m.fileDialog.Update(msg)
		m.fileDialog = newModel
		return m, cmd
	}
	
	return m, nil
}

func (m *Model) createChatModel() tea.Model {
	// Import the page package
	return page.NewChatPage(m.app, m.theme)
}

func (m *Model) renderError() string {
	if m.err == nil {
		return ""
	}
	
	style := m.theme.ErrorStyle().
		Width(m.width - 4).
		Padding(1, 2)
		
	return style.Render(fmt.Sprintf("Error: %v", m.err))
}

// Run starts the TUI application
func Run(app *app.App) error {
	model := New(app)
	
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}
	
	return nil
}

// Key bindings
type keyMap struct {
	Quit        key.Binding
	Help        key.Binding
	ModelSelect key.Binding
	FileSelect  key.Binding
	Cancel      key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "ctrl+q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?", "ctrl+h"),
		key.WithHelp("?", "help"),
	),
	ModelSelect: key.NewBinding(
		key.WithKeys("ctrl+m"),
		key.WithHelp("ctrl+m", "select model"),
	),
	FileSelect: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "attach file"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "cancel"),
	),
}