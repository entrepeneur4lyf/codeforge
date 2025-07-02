package themes

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme defines the interface for TUI themes
type Theme interface {
	// Base styles
	Base() lipgloss.Style
	Background() lipgloss.Style
	
	// Text styles
	PrimaryText() lipgloss.Style
	SecondaryText() lipgloss.Style
	MutedText() lipgloss.Style
	ErrorText() lipgloss.Style
	SuccessText() lipgloss.Style
	WarningText() lipgloss.Style
	
	// UI element styles
	Border() lipgloss.Style
	BorderActive() lipgloss.Style
	Button() lipgloss.Style
	ButtonActive() lipgloss.Style
	Input() lipgloss.Style
	InputActive() lipgloss.Style
	
	// Dialog styles
	DialogStyle() lipgloss.Style
	DialogTitleStyle() lipgloss.Style
	
	// List styles
	ListItem() lipgloss.Style
	ListItemActive() lipgloss.Style
	ListItemSelected() lipgloss.Style
	
	// Status styles
	StatusBar() lipgloss.Style
	StatusKey() lipgloss.Style
	StatusValue() lipgloss.Style
	
	// Special styles
	ErrorStyle() lipgloss.Style
	SuccessStyle() lipgloss.Style
	WarningStyle() lipgloss.Style
	InfoStyle() lipgloss.Style
	
	// Code styles
	CodeBlock() lipgloss.Style
	CodeInline() lipgloss.Style
	
	// Colors
	Primary() lipgloss.Color
	Secondary() lipgloss.Color
	BackgroundColor() lipgloss.Color
	Foreground() lipgloss.Color
	Error() lipgloss.Color
	Success() lipgloss.Color
	Warning() lipgloss.Color
	Info() lipgloss.Color
}