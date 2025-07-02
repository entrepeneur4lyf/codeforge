package chat

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/themes"
)

// Session represents a chat session
type Session struct {
	ID          string
	Title       string
	LastMessage string
	UpdatedAt   time.Time
	MessageCount int
}

// SessionsModel represents the session list sidebar
type SessionsModel struct {
	theme        themes.Theme
	sessions     []Session
	list         list.Model
	width        int
	height       int
	focused      bool
	currentID    string
}

// SessionSelectedMsg is sent when a session is selected
type SessionSelectedMsg struct {
	SessionID string
}

// SessionCreatedMsg is sent when a new session is created
type SessionCreatedMsg struct {
	Session Session
}

// SessionDeletedMsg is sent when a session is deleted
type SessionDeletedMsg struct {
	SessionID string
}

// sessionItem implements list.Item interface
type sessionItem struct {
	session Session
	current bool
}

func (i sessionItem) FilterValue() string { return i.session.Title }
func (i sessionItem) Title() string       { return i.session.Title }
func (i sessionItem) Description() string { 
	return fmt.Sprintf("%d messages • %s", 
		i.session.MessageCount, 
		i.session.UpdatedAt.Format("Jan 2, 15:04"))
}

// sessionDelegate implements list.ItemDelegate
type sessionDelegate struct {
	theme     themes.Theme
	currentID string
}

func (d sessionDelegate) Height() int                               { return 3 }
func (d sessionDelegate) Spacing() int                              { return 1 }
func (d sessionDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }

func (d sessionDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(sessionItem)
	if !ok {
		return
	}
	
	// Base style
	style := d.theme.ListItem().
		Width(m.Width() - 4).
		Padding(0, 1)
	
	// Highlight current session
	if i.current {
		style = style.
			Background(d.theme.Primary()).
			Foreground(d.theme.BackgroundColor())
	}
	
	// Highlight selected item
	if index == m.Index() {
		style = style.BorderLeft(true).
			BorderForeground(d.theme.Primary())
	}
	
	// Title
	title := i.session.Title
	if title == "" {
		title = "New Chat"
	}
	
	// Description
	desc := d.theme.MutedText().Render(i.Description())
	
	// Render
	content := fmt.Sprintf("%s\n%s", title, desc)
	fmt.Fprint(w, style.Render(content))
}

// NewSessionsModel creates a new sessions sidebar
func NewSessionsModel(theme themes.Theme) *SessionsModel {
	// Create list
	items := []list.Item{}
	delegate := sessionDelegate{theme: theme}
	
	l := list.New(items, delegate, 0, 0)
	l.Title = "Sessions"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.Title = theme.DialogTitleStyle()
	l.Styles.FilterPrompt = theme.PrimaryText()
	l.Styles.FilterCursor = theme.Primary()
	
	return &SessionsModel{
		theme:    theme,
		sessions: []Session{},
		list:     l,
	}
}

func (m *SessionsModel) Init() tea.Cmd {
	return nil
}

func (m *SessionsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(m.width, m.height)
		
	case tea.KeyMsg:
		if m.focused {
			switch {
			case key.Matches(msg, sessionKeys.Select):
				if i, ok := m.list.SelectedItem().(sessionItem); ok {
					return m, func() tea.Msg {
						return SessionSelectedMsg{SessionID: i.session.ID}
					}
				}
				
			case key.Matches(msg, sessionKeys.New):
				// Create new session
				newSession := Session{
					ID:          fmt.Sprintf("session-%d", time.Now().Unix()),
					Title:       "New Chat",
					UpdatedAt:   time.Now(),
					MessageCount: 0,
				}
				return m, func() tea.Msg {
					return SessionCreatedMsg{Session: newSession}
				}
				
			case key.Matches(msg, sessionKeys.Delete):
				if i, ok := m.list.SelectedItem().(sessionItem); ok {
					return m, func() tea.Msg {
						return SessionDeletedMsg{SessionID: i.session.ID}
					}
				}
			}
		}
		
	case SessionCreatedMsg:
		m.addSession(msg.Session)
		m.currentID = msg.Session.ID
		m.updateList()
		
	case SessionDeletedMsg:
		m.removeSession(msg.SessionID)
		m.updateList()
		
	case SessionSelectedMsg:
		m.currentID = msg.SessionID
		m.updateList()
	}
	
	// Update list
	if m.focused {
		m.list, cmd = m.list.Update(msg)
	}
	
	return m, cmd
}

func (m *SessionsModel) View() string {
	// Container style
	containerStyle := m.theme.Border().
		Width(m.width).
		Height(m.height).
		Padding(1)
		
	// Add title if not focused
	if !m.focused {
		titleStyle := m.theme.DialogTitleStyle().
			Width(m.width - 4).
			MarginBottom(1)
		title := titleStyle.Render("Sessions")
		
		// Session list
		var sessions []string
		for i, session := range m.sessions {
			sessionView := m.renderSession(session, i == m.list.Index())
			sessions = append(sessions, sessionView)
		}
		
		content := strings.Join(sessions, "\n")
		return containerStyle.Render(fmt.Sprintf("%s\n%s", title, content))
	}
	
	// Focused view with list
	return containerStyle.Render(m.list.View())
}

func (m *SessionsModel) renderSession(session Session, selected bool) string {
	// Base style
	style := m.theme.ListItem().
		Width(m.width - 6).
		Padding(0, 1)
		
	// Highlight current
	if session.ID == m.currentID {
		style = style.
			Background(m.theme.Primary()).
			Foreground(m.theme.BackgroundColor())
	}
	
	// Highlight selected
	if selected {
		style = style.BorderLeft(true).
			BorderForeground(m.theme.Primary())
	}
	
	// Title
	title := session.Title
	if title == "" {
		title = "New Chat"
	}
	
	// Info
	info := m.theme.MutedText().Render(
		fmt.Sprintf("%d msgs • %s", 
			session.MessageCount,
			session.UpdatedAt.Format("15:04")))
	
	return style.Render(fmt.Sprintf("%s\n%s", title, info))
}

// Focus sets focus on the sessions list
func (m *SessionsModel) Focus() {
	m.focused = true
}

// Blur removes focus from the sessions list
func (m *SessionsModel) Blur() {
	m.focused = false
}

// SetSessions sets the session list
func (m *SessionsModel) SetSessions(sessions []Session) {
	m.sessions = sessions
	m.updateList()
}

// AddSession adds a new session
func (m *SessionsModel) addSession(session Session) {
	m.sessions = append([]Session{session}, m.sessions...)
	m.updateList()
}

// RemoveSession removes a session
func (m *SessionsModel) removeSession(sessionID string) {
	var filtered []Session
	for _, s := range m.sessions {
		if s.ID != sessionID {
			filtered = append(filtered, s)
		}
	}
	m.sessions = filtered
	
	// If current session was deleted, select first
	if m.currentID == sessionID && len(m.sessions) > 0 {
		m.currentID = m.sessions[0].ID
	}
}

// UpdateSession updates a session's info
func (m *SessionsModel) UpdateSession(sessionID string, updater func(*Session)) {
	for i := range m.sessions {
		if m.sessions[i].ID == sessionID {
			updater(&m.sessions[i])
			m.updateList()
			break
		}
	}
}

// updateList updates the list items
func (m *SessionsModel) updateList() {
	items := make([]list.Item, len(m.sessions))
	for i, session := range m.sessions {
		items[i] = sessionItem{
			session: session,
			current: session.ID == m.currentID,
		}
	}
	
	// Update delegate with current ID
	delegate := sessionDelegate{
		theme:     m.theme,
		currentID: m.currentID,
	}
	m.list.SetDelegate(delegate)
	m.list.SetItems(items)
}

// GetCurrentSessionID returns the current session ID
func (m *SessionsModel) GetCurrentSessionID() string {
	return m.currentID
}

// Key bindings
type sessionKeyMap struct {
	Select key.Binding
	New    key.Binding
	Delete key.Binding
}

var sessionKeys = sessionKeyMap{
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select session"),
	),
	New: key.NewBinding(
		key.WithKeys("n", "ctrl+n"),
		key.WithHelp("n", "new session"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d", "delete"),
		key.WithHelp("d", "delete session"),
	),
}