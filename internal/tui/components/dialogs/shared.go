package dialog

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

// DialogCloseMsg is sent when a dialog should close
type DialogCloseMsg struct{}

// Helper functions used by multiple dialogs

// formatContextWindow formats token counts for display
func formatContextWindow(tokens int) string {
	if tokens >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(tokens)/1000000)
	}
	if tokens >= 1000 {
		return fmt.Sprintf("%dk", tokens/1000)
	}
	return fmt.Sprintf("%d", tokens)
}

// truncate truncates a string to fit within maxWidth while preserving runes
func truncate(s string, maxWidth int) string {
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		truncated := string(runes[:i])
		if lipgloss.Width(truncated) <= maxWidth {
			return truncated
		}
	}
	return ""
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}