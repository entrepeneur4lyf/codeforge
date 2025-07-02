package dialogs

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/themes"
)

// HelpDialog displays keyboard shortcuts and help information
type HelpDialog struct {
	theme  themes.Theme
	width  int
	height int
}

// NewHelpDialog creates a new help dialog
func NewHelpDialog(theme themes.Theme) tea.Model {
	return &HelpDialog{
		theme: theme,
	}
}

func (h *HelpDialog) Init() tea.Cmd {
	return nil
}

func (h *HelpDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h.width = msg.Width
		h.height = msg.Height
		
	case tea.KeyMsg:
		// Any key closes the help dialog
		return h, func() tea.Msg { return DialogCloseMsg{} }
	}
	
	return h, nil
}

func (h *HelpDialog) View() string {
	if h.width == 0 || h.height == 0 {
		return ""
	}
	
	// Calculate dialog dimensions
	dialogWidth := min(h.width-4, 60)
	dialogHeight := min(h.height-4, 25)
	
	// Build content
	var content strings.Builder
	
	// Title
	titleStyle := h.theme.DialogTitleStyle().Width(dialogWidth - 4).Align(lipgloss.Center)
	content.WriteString(titleStyle.Render("CodeForge TUI Help"))
	content.WriteString("\n\n")
	
	// Key bindings sections
	sections := []struct {
		title string
		keys  []struct{ key, desc string }
	}{
		{
			title: "General",
			keys: []struct{ key, desc string }{
				{"ctrl+c/q", "Quit"},
				{"?", "Show this help"},
				{"ctrl+m", "Select model"},
				{"ctrl+f", "Attach file"},
				{"ctrl+p", "Search files"},
				{"ctrl+shift+f", "Search in files"},
				{"ctrl+n", "New session"},
				{"ctrl+l", "Clear chat"},
			},
		},
		{
			title: "Navigation",
			keys: []struct{ key, desc string }{
				{"↑/↓ or j/k", "Scroll messages"},
				{"PgUp/PgDn", "Page up/down"},
				{"Home/End", "Jump to start/end"},
				{"tab", "Switch focus"},
				{"ctrl+b", "Toggle sidebar"},
			},
		},
		{
			title: "Editor",
			keys: []struct{ key, desc string }{
				{"enter", "Send message"},
				{"shift+enter", "New line"},
				{"ctrl+r", "Remove attachment"},
			},
		},
		{
			title: "Model Selection",
			keys: []struct{ key, desc string }{
				{"←/→", "Switch provider"},
				{"↑/↓", "Select model"},
				{"space", "Toggle favorite"},
				{"f", "Show only favorites"},
				{"enter", "Confirm selection"},
				{"esc", "Cancel"},
			},
		},
	}
	
	// Render sections
	for i, section := range sections {
		if i > 0 {
			content.WriteString("\n")
		}
		
		// Section title
		sectionStyle := h.theme.SecondaryText().Bold(true)
		content.WriteString(sectionStyle.Render(section.title))
		content.WriteString("\n")
		
		// Key bindings
		for _, binding := range section.keys {
			keyStyle := h.theme.PrimaryText().Width(15)
			descStyle := h.theme.Base()
			
			line := lipgloss.JoinHorizontal(
				lipgloss.Top,
				keyStyle.Render(binding.key),
				descStyle.Render(binding.desc),
			)
			content.WriteString("  " + line + "\n")
		}
	}
	
	// Footer
	content.WriteString("\n")
	footerStyle := h.theme.MutedText().Width(dialogWidth - 4).Align(lipgloss.Center)
	content.WriteString(footerStyle.Render("Press any key to close"))
	
	// Apply dialog style
	dialogStyle := h.theme.DialogStyle().
		Width(dialogWidth).
		Height(dialogHeight).
		MaxWidth(dialogWidth).
		MaxHeight(dialogHeight)
		
	return dialogStyle.Render(content.String())
}