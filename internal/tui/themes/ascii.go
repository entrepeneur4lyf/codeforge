package themes

import (
	"github.com/charmbracelet/lipgloss"
)

// ASCIITheme implements a simple ASCII-only theme for better terminal compatibility
type ASCIITheme struct {
	*DefaultTheme
}

// NewASCIITheme creates a new ASCII-only theme
func NewASCIITheme() Theme {
	return &ASCIITheme{
		DefaultTheme: NewDefaultTheme().(*DefaultTheme),
	}
}

// Override border styles to use ASCII characters
func (t *ASCIITheme) Border() lipgloss.Style {
	return t.Base().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.muted)
}

func (t *ASCIITheme) BorderActive() lipgloss.Style {
	return t.Base().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.primary)
}

func (t *ASCIITheme) Input() lipgloss.Style {
	return t.Base().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.muted).
		Padding(0, 1)
}

func (t *ASCIITheme) InputActive() lipgloss.Style {
	return t.Base().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.primary).
		Padding(0, 1)
}

func (t *ASCIITheme) DialogStyle() lipgloss.Style {
	return t.Base().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.primary).
		Background(t.background).
		Padding(1, 2)
}

func (t *ASCIITheme) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.error).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.error).
		Padding(1, 2)
}

func (t *ASCIITheme) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.success).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.success).
		Padding(1, 2)
}

func (t *ASCIITheme) WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.warning).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.warning).
		Padding(1, 2)
}

func (t *ASCIITheme) InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.info).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.info).
		Padding(1, 2)
}

func (t *ASCIITheme) CodeBlock() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(t.foreground).
		Padding(1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.muted)
}