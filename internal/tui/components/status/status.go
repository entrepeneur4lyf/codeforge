package status

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/app"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// StatusComponent is the interface for the status bar
type StatusComponent interface {
	tea.Model
	View() string
	SetWidth(width int)
}

// Model represents the status bar component
type Model struct {
	app    *app.App
	theme  theme.Theme
	width  int
	height int
	
	// Custom content to show in the status bar
	customLeft string
	
	// Session tracking
	sessionTokens float64
	sessionCost   float64
}

// NewStatusBar creates a new status bar component
func NewStatusBar(app *app.App, th theme.Theme) *Model {
	return &Model{
		app:    app,
		theme:  th,
		height: 2, // Status bar is 2 lines tall
	}
}

// Init initializes the status bar
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages for the status bar
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	}
	return m, nil
}

// SetWidth sets the width of the status bar
func (m *Model) SetWidth(width int) {
	m.width = width
}

// SetCustomLeft sets custom content for the left side of the status bar
func (m *Model) SetCustomLeft(content string) {
	m.customLeft = content
}

// UpdateSessionInfo updates the session token and cost information
func (m *Model) UpdateSessionInfo(tokens float64, cost float64) {
	m.sessionTokens = tokens
	m.sessionCost = cost
}

// logo renders the CodeForge logo/brand
func (m *Model) logo() string {
	baseStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted()).
		Background(m.theme.BackgroundSecondary())
	
	emphasisStyle := lipgloss.NewStyle().
		Foreground(m.theme.Text()).
		Background(m.theme.BackgroundSecondary()).
		Bold(true)
	
	code := baseStyle.Render("Code")
	forge := emphasisStyle.Render("Forge ")
	version := baseStyle.Render("v1.0") // TODO: Get version from app
	
	return lipgloss.NewStyle().
		Background(m.theme.BackgroundSecondary()).
		Padding(0, 1).
		Render(code + forge + version)
}

// formatTokensAndCost formats token usage and cost information
func formatTokensAndCost(tokens float64, contextWindow float64, cost float64) string {
	// Format tokens in human-readable format (e.g., 110K, 1.2M)
	var formattedTokens string
	switch {
	case tokens >= 1_000_000:
		formattedTokens = fmt.Sprintf("%.1fM", tokens/1_000_000)
	case tokens >= 1_000:
		formattedTokens = fmt.Sprintf("%.1fK", tokens/1_000)
	default:
		formattedTokens = fmt.Sprintf("%d", int(tokens))
	}

	// Remove .0 suffix if present
	if strings.HasSuffix(formattedTokens, ".0K") {
		formattedTokens = strings.Replace(formattedTokens, ".0K", "K", 1)
	}
	if strings.HasSuffix(formattedTokens, ".0M") {
		formattedTokens = strings.Replace(formattedTokens, ".0M", "M", 1)
	}

	// Format cost with $ symbol and 2 decimal places
	formattedCost := fmt.Sprintf("$%.2f", cost)
	
	// Calculate percentage
	percentage := 0
	if contextWindow > 0 {
		percentage = int((tokens / contextWindow) * 100)
	}

	return fmt.Sprintf("Context: %s (%d%%), Cost: %s", formattedTokens, percentage, formattedCost)
}

// getCurrentSessionInfo gets current session token usage and cost
func (m *Model) getCurrentSessionInfo() (tokens float64, cost float64, contextWindow float64) {
	// Default context window
	contextWindow = 128000 // Default to 128K
	
	// Use tracked session values
	tokens = m.sessionTokens
	cost = m.sessionCost
	
	// Get context window from current model if available
	if provider, model := m.app.GetCurrentModel(); provider != "" && model != "" {
		models := m.app.GetAvailableModels()
		for _, m := range models {
			if m.Provider == provider && m.Name == model {
				contextWindow = float64(m.Info.ContextWindow)
				break
			}
		}
	}
	
	return tokens, cost, contextWindow
}

// View renders the status bar
func (m *Model) View() string {
	if m.width == 0 {
		return ""
	}

	// First line - custom content or blank
	firstLine := ""
	if m.customLeft != "" {
		firstLine = m.customLeft
	} else {
		firstLine = lipgloss.NewStyle().
			Background(m.theme.Background()).
			Width(m.width).
			Render("")
	}
	
	// Second line - status bar
	// Get components
	logo := m.logo()
	
	// Working directory
	cwdStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted()).
		Background(m.theme.Background()).
		Padding(0, 1)
	
	workDir := m.app.WorkspaceRoot
	if workDir == "" {
		workDir = "~"
	}
	// Truncate if too long
	maxCwdWidth := m.width / 3
	if lipgloss.Width(workDir) > maxCwdWidth {
		workDir = "..." + workDir[len(workDir)-maxCwdWidth+3:]
	}
	cwd := cwdStyle.Render(workDir)
	
	// Session info (tokens and cost)
	sessionInfo := ""
	tokens, cost, contextWindow := m.getCurrentSessionInfo()
	if tokens > 0 || cost > 0 {
		sessionInfo = lipgloss.NewStyle().
			Foreground(m.theme.TextMuted()).
			Background(m.theme.BackgroundSecondary()).
			Padding(0, 1).
			Render(formatTokensAndCost(tokens, contextWindow, cost))
	}
	
	// Calculate spacer
	space := max(
		0,
		m.width-lipgloss.Width(logo)-lipgloss.Width(cwd)-lipgloss.Width(sessionInfo),
	)
	spacer := lipgloss.NewStyle().
		Background(m.theme.Background()).
		Width(space).
		Render("")
	
	// Combine status line
	status := logo + cwd + spacer + sessionInfo
	
	return firstLine + "\n" + status
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}