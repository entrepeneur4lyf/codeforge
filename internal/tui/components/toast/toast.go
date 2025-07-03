package toast

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// ShowToastMsg is a message to display a toast notification
type ShowToastMsg struct {
	Message  string
	Title    *string
	Color    lipgloss.AdaptiveColor
	Duration time.Duration
}

// DismissToastMsg is a message to dismiss a specific toast
type DismissToastMsg struct {
	ID string
}

// Toast represents a single toast notification
type Toast struct {
	ID        string
	Message   string
	Title     *string
	Color     lipgloss.AdaptiveColor
	CreatedAt time.Time
	Duration  time.Duration
}

// ToastManager manages multiple toast notifications
type ToastManager struct {
	toasts []Toast
	theme  theme.Theme
}

// NewToastManager creates a new toast manager
func NewToastManager(th theme.Theme) *ToastManager {
	return &ToastManager{
		toasts: []Toast{},
		theme:  th,
	}
}

// Init initializes the toast manager
func (tm *ToastManager) Init() tea.Cmd {
	return nil
}

// Update handles messages for the toast manager
func (tm *ToastManager) Update(msg tea.Msg) (*ToastManager, tea.Cmd) {
	switch msg := msg.(type) {
	case ShowToastMsg:
		toast := Toast{
			ID:        fmt.Sprintf("toast-%d", time.Now().UnixNano()),
			Title:     msg.Title,
			Message:   msg.Message,
			Color:     msg.Color,
			CreatedAt: time.Now(),
			Duration:  msg.Duration,
		}

		tm.toasts = append(tm.toasts, toast)

		// Return command to dismiss after duration
		return tm, tea.Tick(toast.Duration, func(t time.Time) tea.Msg {
			return DismissToastMsg{ID: toast.ID}
		})

	case DismissToastMsg:
		var newToasts []Toast
		for _, t := range tm.toasts {
			if t.ID != msg.ID {
				newToasts = append(newToasts, t)
			}
		}
		tm.toasts = newToasts
	}

	return tm, nil
}

// renderSingleToast renders a single toast notification
func (tm *ToastManager) renderSingleToast(toast Toast, width int) string {
	baseStyle := lipgloss.NewStyle().
		Foreground(tm.theme.Text()).
		Background(tm.theme.BackgroundSecondary()).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(toast.Color)

	maxWidth := max(40, width/3)
	contentMaxWidth := max(maxWidth-6, 20)

	// Build content with wrapping
	var content strings.Builder
	if toast.Title != nil {
		titleStyle := lipgloss.NewStyle().
			Foreground(toast.Color).
			Bold(true)
		content.WriteString(titleStyle.Render(*toast.Title))
		content.WriteString("\n")
	}

	// Wrap message text
	messageStyle := lipgloss.NewStyle()
	contentWidth := lipgloss.Width(toast.Message)
	if contentWidth > contentMaxWidth {
		messageStyle = messageStyle.Width(contentMaxWidth)
	}
	content.WriteString(messageStyle.Render(toast.Message))

	// Render toast with max width
	return baseStyle.MaxWidth(maxWidth).Render(content.String())
}

// View renders all active toasts
func (tm *ToastManager) View(width int) string {
	if len(tm.toasts) == 0 {
		return ""
	}

	var toastViews []string
	for _, toast := range tm.toasts {
		toastView := tm.renderSingleToast(toast, width)
		toastViews = append(toastViews, toastView)
	}

	return strings.Join(toastViews, "\n")
}

// RenderOverlay renders the toasts as an overlay on the given background
func (tm *ToastManager) RenderOverlay(background string) string {
	if len(tm.toasts) == 0 {
		return background
	}

	bgWidth := lipgloss.Width(background)
	bgHeight := lipgloss.Height(background)
	
	// Get toast views
	toastView := tm.View(bgWidth)
	if toastView == "" {
		return background
	}
	
	// Calculate position (top-right with padding)
	toastWidth := lipgloss.Width(toastView)
	toastHeight := lipgloss.Height(toastView)
	
	x := max(bgWidth-toastWidth-2, 0)
	y := 2
	
	// Check if toast fits
	if y+toastHeight > bgHeight-2 {
		// No room for toasts
		return background
	}
	
	// Create overlay
	return placeOverlay(x, y, toastView, background)
}

// placeOverlay places content over background at specified position
func placeOverlay(x, y int, overlay, background string) string {
	bgLines := strings.Split(background, "\n")
	overlayLines := strings.Split(overlay, "\n")
	
	for i, overlayLine := range overlayLines {
		bgLineIdx := y + i
		if bgLineIdx >= 0 && bgLineIdx < len(bgLines) {
			line := bgLines[bgLineIdx]
			lineRunes := []rune(line)
			overlayRunes := []rune(overlayLine)
			
			// Replace characters in the background line
			for j, r := range overlayRunes {
				if x+j < len(lineRunes) {
					lineRunes[x+j] = r
				}
			}
			
			bgLines[bgLineIdx] = string(lineRunes)
		}
	}
	
	return strings.Join(bgLines, "\n")
}

// Helper functions for creating toasts with different styles

type ToastOptions struct {
	Title    string
	Duration time.Duration
}

type toastOptions struct {
	title    *string
	duration *time.Duration
	color    *lipgloss.AdaptiveColor
}

type ToastOption func(*toastOptions)

func WithTitle(title string) ToastOption {
	return func(t *toastOptions) {
		t.title = &title
	}
}

func WithDuration(duration time.Duration) ToastOption {
	return func(t *toastOptions) {
		t.duration = &duration
	}
}

func WithColor(color lipgloss.AdaptiveColor) ToastOption {
	return func(t *toastOptions) {
		t.color = &color
	}
}

func NewToast(message string, th theme.Theme, options ...ToastOption) tea.Cmd {
	duration := 5 * time.Second
	color := th.Primary()

	opts := toastOptions{
		duration: &duration,
		color:    &color,
	}
	for _, option := range options {
		option(&opts)
	}

	return func() tea.Msg {
		return ShowToastMsg{
			Message:  message,
			Title:    opts.title,
			Duration: *opts.duration,
			Color:    *opts.color,
		}
	}
}

func NewInfoToast(message string, th theme.Theme, options ...ToastOption) tea.Cmd {
	options = append(options, WithColor(th.Info()))
	return NewToast(message, th, options...)
}

func NewSuccessToast(message string, th theme.Theme, options ...ToastOption) tea.Cmd {
	options = append(options, WithColor(th.Success()))
	return NewToast(message, th, options...)
}

func NewWarningToast(message string, th theme.Theme, options ...ToastOption) tea.Cmd {
	options = append(options, WithColor(th.Warning()))
	return NewToast(message, th, options...)
}

func NewErrorToast(message string, th theme.Theme, options ...ToastOption) tea.Cmd {
	options = append(options, WithColor(th.Error()))
	return NewToast(message, th, options...)
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}