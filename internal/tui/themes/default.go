package themes

import (
	"github.com/charmbracelet/lipgloss"
)

// DefaultTheme implements the CodeForge default theme
type DefaultTheme struct {
	primary    lipgloss.Color
	secondary  lipgloss.Color
	background lipgloss.Color
	foreground lipgloss.Color
	error      lipgloss.Color
	success    lipgloss.Color
	warning    lipgloss.Color
	info       lipgloss.Color
	muted      lipgloss.Color
}

// NewDefaultTheme creates a new default theme
func NewDefaultTheme() Theme {
	return &DefaultTheme{
		primary:    lipgloss.Color("#00D9FF"), // Cyan
		secondary:  lipgloss.Color("#FF79C6"), // Pink
		background: lipgloss.Color("#282A36"), // Dark background
		foreground: lipgloss.Color("#F8F8F2"), // Light foreground
		error:      lipgloss.Color("#FF5555"), // Red
		success:    lipgloss.Color("#50FA7B"), // Green
		warning:    lipgloss.Color("#FFB86C"), // Orange
		info:       lipgloss.Color("#8BE9FD"), // Light cyan
		muted:      lipgloss.Color("#6272A4"), // Comment gray
	}
}

func (t *DefaultTheme) Base() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(t.background).
		Foreground(t.foreground)
}

func (t *DefaultTheme) Background() lipgloss.Style {
	return lipgloss.NewStyle().Background(t.background)
}

func (t *DefaultTheme) PrimaryText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.primary)
}

func (t *DefaultTheme) SecondaryText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.secondary)
}

func (t *DefaultTheme) MutedText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.muted)
}

func (t *DefaultTheme) ErrorText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.error)
}

func (t *DefaultTheme) SuccessText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.success)
}

func (t *DefaultTheme) WarningText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.warning)
}

func (t *DefaultTheme) Border() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.muted)
}

func (t *DefaultTheme) BorderActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.primary)
}

func (t *DefaultTheme) Button() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.foreground).
		Background(t.muted).
		Padding(0, 2).
		MarginRight(1)
}

func (t *DefaultTheme) ButtonActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.background).
		Background(t.primary).
		Padding(0, 2).
		MarginRight(1)
}

func (t *DefaultTheme) Input() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.muted).
		Padding(0, 1)
}

func (t *DefaultTheme) InputActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.primary).
		Padding(0, 1)
}

func (t *DefaultTheme) DialogStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.primary).
		Background(t.background).
		Padding(1, 2)
}

func (t *DefaultTheme) DialogTitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.primary).
		Bold(true).
		MarginBottom(1)
}

func (t *DefaultTheme) ListItem() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.foreground).
		PaddingLeft(2)
}

func (t *DefaultTheme) ListItemActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.primary).
		PaddingLeft(2)
}

func (t *DefaultTheme) ListItemSelected() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.background).
		Background(t.primary).
		PaddingLeft(2)
}

func (t *DefaultTheme) StatusBar() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#44475A")).
		Foreground(t.foreground).
		Padding(0, 1)
}

func (t *DefaultTheme) StatusKey() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.muted).
		MarginRight(1)
}

func (t *DefaultTheme) StatusValue() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.primary).
		MarginRight(2)
}

func (t *DefaultTheme) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.error).
		Background(lipgloss.Color("#3B1919")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.error).
		Padding(1, 2)
}

func (t *DefaultTheme) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.success).
		Background(lipgloss.Color("#1B3B1B")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.success).
		Padding(1, 2)
}

func (t *DefaultTheme) WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.warning).
		Background(lipgloss.Color("#3B2B1B")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.warning).
		Padding(1, 2)
}

func (t *DefaultTheme) InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.info).
		Background(lipgloss.Color("#1B2B3B")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.info).
		Padding(1, 2)
}

func (t *DefaultTheme) CodeBlock() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#1E1F29")).
		Foreground(t.foreground).
		Padding(1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.muted)
}

func (t *DefaultTheme) CodeInline() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color("#44475A")).
		Foreground(t.info).
		Padding(0, 1)
}

// Color getters
func (t *DefaultTheme) Primary() lipgloss.Color    { return t.primary }
func (t *DefaultTheme) Secondary() lipgloss.Color  { return t.secondary }
func (t *DefaultTheme) BackgroundColor() lipgloss.Color { return t.background }
func (t *DefaultTheme) Foreground() lipgloss.Color { return t.foreground }
func (t *DefaultTheme) Error() lipgloss.Color      { return t.error }
func (t *DefaultTheme) Success() lipgloss.Color    { return t.success }
func (t *DefaultTheme) Warning() lipgloss.Color    { return t.warning }
func (t *DefaultTheme) Info() lipgloss.Color       { return t.info }