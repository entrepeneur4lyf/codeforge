package themes

import (
	"github.com/charmbracelet/lipgloss"
)

// SimpleTheme implements a minimal theme with no background colors
type SimpleTheme struct {
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

// NewSimpleTheme creates a new simple theme with no backgrounds
func NewSimpleTheme() Theme {
	return &SimpleTheme{
		primary:    lipgloss.Color("6"),  // Cyan
		secondary:  lipgloss.Color("5"),  // Magenta
		background: lipgloss.Color(""),   // No background
		foreground: lipgloss.Color("7"),  // White
		error:      lipgloss.Color("1"),  // Red
		success:    lipgloss.Color("2"),  // Green
		warning:    lipgloss.Color("3"),  // Yellow
		info:       lipgloss.Color("4"),  // Blue
		muted:      lipgloss.Color("8"),  // Gray
	}
}

func (t *SimpleTheme) Base() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.foreground)
}

func (t *SimpleTheme) Background() lipgloss.Style {
	return lipgloss.NewStyle()
}

func (t *SimpleTheme) PrimaryText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.primary)
}

func (t *SimpleTheme) SecondaryText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.secondary)
}

func (t *SimpleTheme) MutedText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.muted)
}

func (t *SimpleTheme) ErrorText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.error)
}

func (t *SimpleTheme) SuccessText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.success)
}

func (t *SimpleTheme) WarningText() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.warning)
}

func (t *SimpleTheme) Border() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.muted)
}

func (t *SimpleTheme) BorderActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.primary)
}

func (t *SimpleTheme) Button() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.foreground).
		Padding(0, 2).
		MarginRight(1)
}

func (t *SimpleTheme) ButtonActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.primary).
		Bold(true).
		Padding(0, 2).
		MarginRight(1)
}

func (t *SimpleTheme) Input() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.muted).
		Padding(0, 1)
}

func (t *SimpleTheme) InputActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.primary).
		Padding(0, 1)
}

func (t *SimpleTheme) DialogStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.primary).
		Padding(1, 2)
}

func (t *SimpleTheme) DialogTitleStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.primary).
		Bold(true).
		MarginBottom(1)
}

func (t *SimpleTheme) ListItem() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.foreground).
		PaddingLeft(2)
}

func (t *SimpleTheme) ListItemActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.primary).
		PaddingLeft(2)
}

func (t *SimpleTheme) ListItemSelected() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.primary).
		Bold(true).
		PaddingLeft(2)
}

func (t *SimpleTheme) StatusBar() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.foreground).
		Padding(0, 1).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(t.muted)
}

func (t *SimpleTheme) StatusKey() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.muted).
		MarginRight(1)
}

func (t *SimpleTheme) StatusValue() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.primary).
		MarginRight(2)
}

func (t *SimpleTheme) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.error).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.error).
		Padding(1, 2)
}

func (t *SimpleTheme) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.success).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.success).
		Padding(1, 2)
}

func (t *SimpleTheme) WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.warning).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.warning).
		Padding(1, 2)
}

func (t *SimpleTheme) InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.info).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.info).
		Padding(1, 2)
}

func (t *SimpleTheme) CodeBlock() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.foreground).
		Padding(1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.muted)
}

func (t *SimpleTheme) CodeInline() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.info).
		Padding(0, 1)
}

// Color getters
func (t *SimpleTheme) Primary() lipgloss.Color    { return t.primary }
func (t *SimpleTheme) Secondary() lipgloss.Color  { return t.secondary }
func (t *SimpleTheme) BackgroundColor() lipgloss.Color { return t.background }
func (t *SimpleTheme) Foreground() lipgloss.Color { return t.foreground }
func (t *SimpleTheme) Error() lipgloss.Color      { return t.error }
func (t *SimpleTheme) Success() lipgloss.Color    { return t.success }
func (t *SimpleTheme) Warning() lipgloss.Color    { return t.warning }
func (t *SimpleTheme) Info() lipgloss.Color       { return t.info }