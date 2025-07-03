package chat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// DiagnosticLevel represents the severity of a diagnostic
type DiagnosticLevel string

const (
	DiagnosticError   DiagnosticLevel = "error"
	DiagnosticWarning DiagnosticLevel = "warning"
	DiagnosticInfo    DiagnosticLevel = "info"
	DiagnosticHint    DiagnosticLevel = "hint"
)

// Diagnostic represents a diagnostic message with context
type Diagnostic struct {
	Level    DiagnosticLevel
	Code     string // Optional error code (e.g., "E0001")
	Title    string
	Message  string
	File     string // Optional file path
	Line     int    // Optional line number (1-based)
	Column   int    // Optional column number (1-based)
	Context  []string // Optional lines of context
	Suggestions []string // Optional fix suggestions
}

// DiagnosticRenderer renders diagnostic messages with proper formatting
type DiagnosticRenderer struct {
	theme theme.Theme
	cache *MessageCache
	
	// Renderers for different parts
	errorRenderer   *BlockRenderer
	warningRenderer *BlockRenderer
	infoRenderer    *BlockRenderer
	hintRenderer    *BlockRenderer
	codeRenderer    *BlockRenderer
	contextRenderer *BlockRenderer
}

// NewDiagnosticRenderer creates a new diagnostic renderer
func NewDiagnosticRenderer(theme theme.Theme) *DiagnosticRenderer {
	return &DiagnosticRenderer{
		theme: theme,
		cache: NewMessageCache(100),
		
		errorRenderer: NewBlockRenderer(theme,
			WithTextColor(theme.Error()),
			WithBold(),
			WithBorder(lipgloss.ThickBorder(), theme.Error()),
			WithPadding(1),
		),
		
		warningRenderer: NewBlockRenderer(theme,
			WithTextColor(theme.Warning()),
			WithBold(),
			WithBorder(lipgloss.NormalBorder(), theme.Warning()),
			WithPadding(1),
		),
		
		infoRenderer: NewBlockRenderer(theme,
			WithTextColor(theme.Info()),
			WithBorder(lipgloss.RoundedBorder(), theme.Info()),
			WithPadding(1),
		),
		
		hintRenderer: NewBlockRenderer(theme,
			WithTextColor(theme.Success()),
			WithBorder(lipgloss.HiddenBorder(), theme.Success()),
			WithPadding(1),
		),
		
		codeRenderer: NewBlockRenderer(theme,
			WithBackgroundColor(theme.BackgroundSecondary()),
			WithTextColor(theme.Text()),
			WithPadding(1),
			WithNoBorder(),
		),
		
		contextRenderer: NewBlockRenderer(theme,
			WithTextColor(theme.TextMuted()),
			WithPadding(1),
			WithNoBorder(),
		),
	}
}

// RenderDiagnostic renders a complete diagnostic message
func (r *DiagnosticRenderer) RenderDiagnostic(diag Diagnostic) string {
	cacheKey := r.cache.GenerateKey(
		string(diag.Level),
		diag.Code,
		diag.Title,
		diag.Message,
		fmt.Sprintf("%s:%d:%d", diag.File, diag.Line, diag.Column),
	)
	
	if cached, found := r.cache.Get(cacheKey); found {
		return cached
	}
	
	var result strings.Builder
	
	// Header with level and optional code
	header := r.renderHeader(diag)
	result.WriteString(header)
	result.WriteString("\n\n")
	
	// Title
	if diag.Title != "" {
		titleStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(r.getLevelColor(diag.Level))
		result.WriteString(titleStyle.Render(diag.Title))
		result.WriteString("\n\n")
	}
	
	// Location info
	if diag.File != "" {
		location := r.renderLocation(diag)
		result.WriteString(location)
		result.WriteString("\n\n")
	}
	
	// Context with highlighting
	if len(diag.Context) > 0 && diag.Line > 0 {
		context := r.renderContext(diag)
		result.WriteString(context)
		result.WriteString("\n\n")
	}
	
	// Main message
	messageRenderer := r.getRendererForLevel(diag.Level)
	renderedMessage := messageRenderer.Render(diag.Message)
	result.WriteString(renderedMessage)
	
	// Suggestions
	if len(diag.Suggestions) > 0 {
		result.WriteString("\n\n")
		suggestions := r.renderSuggestions(diag.Suggestions)
		result.WriteString(suggestions)
	}
	
	rendered := result.String()
	r.cache.Set(cacheKey, rendered)
	return rendered
}

// RenderInline renders a compact inline diagnostic
func (r *DiagnosticRenderer) RenderInline(diag Diagnostic) string {
	icon := r.getLevelIcon(diag.Level)
	color := r.getLevelColor(diag.Level)
	
	style := lipgloss.NewStyle().
		Foreground(color).
		Bold(diag.Level == DiagnosticError || diag.Level == DiagnosticWarning)
	
	parts := []string{icon}
	
	if diag.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", diag.Code))
	}
	
	if diag.Title != "" {
		parts = append(parts, diag.Title)
	} else {
		parts = append(parts, diag.Message)
	}
	
	if diag.File != "" {
		location := fmt.Sprintf("(%s", diag.File)
		if diag.Line > 0 {
			location += fmt.Sprintf(":%d", diag.Line)
			if diag.Column > 0 {
				location += fmt.Sprintf(":%d", diag.Column)
			}
		}
		location += ")"
		parts = append(parts, location)
	}
	
	return style.Render(strings.Join(parts, " "))
}

// renderHeader renders the diagnostic header
func (r *DiagnosticRenderer) renderHeader(diag Diagnostic) string {
	icon := r.getLevelIcon(diag.Level)
	levelText := strings.ToUpper(string(diag.Level))
	
	parts := []string{icon, levelText}
	if diag.Code != "" {
		parts = append(parts, fmt.Sprintf("[%s]", diag.Code))
	}
	
	headerStyle := lipgloss.NewStyle().
		Foreground(r.getLevelColor(diag.Level)).
		Bold(true).
		Border(lipgloss.Border{
			Bottom: "─",
		}, false, false, true, false).
		BorderForeground(r.getLevelColor(diag.Level)).
		Width(60)
	
	return headerStyle.Render(strings.Join(parts, " "))
}

// renderLocation renders file location information
func (r *DiagnosticRenderer) renderLocation(diag Diagnostic) string {
	location := fmt.Sprintf("📍 %s", diag.File)
	if diag.Line > 0 {
		location += fmt.Sprintf(":%d", diag.Line)
		if diag.Column > 0 {
			location += fmt.Sprintf(":%d", diag.Column)
		}
	}
	
	style := lipgloss.NewStyle().
		Foreground(r.theme.TextMuted()).
		Italic(true)
	
	return style.Render(location)
}

// renderContext renders the code context with line numbers and highlighting
func (r *DiagnosticRenderer) renderContext(diag Diagnostic) string {
	if len(diag.Context) == 0 || diag.Line <= 0 {
		return ""
	}
	
	var result strings.Builder
	
	// Calculate line number width
	startLine := diag.Line - len(diag.Context)/2
	if startLine < 1 {
		startLine = 1
	}
	endLine := startLine + len(diag.Context) - 1
	lineNumWidth := len(fmt.Sprintf("%d", endLine))
	
	// Render each context line
	for i, line := range diag.Context {
		lineNum := startLine + i
		isErrorLine := lineNum == diag.Line
		
		// Line number
		lineNumStyle := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted()).
			Width(lineNumWidth).
			Align(lipgloss.Right)
		
		if isErrorLine {
			lineNumStyle = lineNumStyle.
				Foreground(r.getLevelColor(diag.Level)).
				Bold(true)
		}
		
		result.WriteString(lineNumStyle.Render(fmt.Sprintf("%d", lineNum)))
		result.WriteString(" │ ")
		
		// Line content
		if isErrorLine {
			// Highlight the error line
			lineStyle := lipgloss.NewStyle().
				Foreground(r.getLevelColor(diag.Level))
			result.WriteString(lineStyle.Render(line))
			
			// Add column indicator if specified
			if diag.Column > 0 && diag.Column <= len(line) {
				result.WriteString("\n")
				result.WriteString(strings.Repeat(" ", lineNumWidth+3+diag.Column-1))
				
				indicatorStyle := lipgloss.NewStyle().
					Foreground(r.getLevelColor(diag.Level)).
					Bold(true)
				result.WriteString(indicatorStyle.Render("^"))
			}
		} else {
			result.WriteString(line)
		}
		
		if i < len(diag.Context)-1 {
			result.WriteString("\n")
		}
	}
	
	// Wrap in a subtle border
	contextStyle := lipgloss.NewStyle().
		Border(lipgloss.Border{
			Left: "│",
		}, false, false, false, true).
		BorderForeground(r.theme.TextMuted()).
		PaddingLeft(1)
	
	return contextStyle.Render(result.String())
}

// renderSuggestions renders fix suggestions
func (r *DiagnosticRenderer) renderSuggestions(suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}
	
	var result strings.Builder
	
	headerStyle := lipgloss.NewStyle().
		Foreground(r.theme.Success()).
		Bold(true)
	result.WriteString(headerStyle.Render("💡 Suggestions:"))
	result.WriteString("\n")
	
	for i, suggestion := range suggestions {
		bulletStyle := lipgloss.NewStyle().
			Foreground(r.theme.Success())
		result.WriteString(bulletStyle.Render("  • "))
		result.WriteString(suggestion)
		
		if i < len(suggestions)-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// Helper methods

func (r *DiagnosticRenderer) getLevelColor(level DiagnosticLevel) lipgloss.AdaptiveColor {
	switch level {
	case DiagnosticError:
		return r.theme.Error()
	case DiagnosticWarning:
		return r.theme.Warning()
	case DiagnosticInfo:
		return r.theme.Info()
	case DiagnosticHint:
		return r.theme.Success()
	default:
		return r.theme.Text()
	}
}

func (r *DiagnosticRenderer) getLevelIcon(level DiagnosticLevel) string {
	switch level {
	case DiagnosticError:
		return "❌"
	case DiagnosticWarning:
		return "⚠️"
	case DiagnosticInfo:
		return "ℹ️"
	case DiagnosticHint:
		return "💡"
	default:
		return "•"
	}
}

func (r *DiagnosticRenderer) getRendererForLevel(level DiagnosticLevel) *BlockRenderer {
	switch level {
	case DiagnosticError:
		return r.errorRenderer
	case DiagnosticWarning:
		return r.warningRenderer
	case DiagnosticInfo:
		return r.infoRenderer
	case DiagnosticHint:
		return r.hintRenderer
	default:
		return r.infoRenderer
	}
}

// DiagnosticGroup represents a collection of related diagnostics
type DiagnosticGroup struct {
	Title       string
	Diagnostics []Diagnostic
}

// RenderDiagnosticGroup renders a group of related diagnostics
func (r *DiagnosticRenderer) RenderDiagnosticGroup(group DiagnosticGroup) string {
	if len(group.Diagnostics) == 0 {
		return ""
	}
	
	var result strings.Builder
	
	// Group header
	if group.Title != "" {
		headerStyle := lipgloss.NewStyle().
			Bold(true).
			Underline(true).
			MarginBottom(1)
		result.WriteString(headerStyle.Render(group.Title))
		result.WriteString("\n\n")
	}
	
	// Render each diagnostic
	for i, diag := range group.Diagnostics {
		// Use inline rendering for multiple diagnostics
		if len(group.Diagnostics) > 3 {
			result.WriteString(r.RenderInline(diag))
		} else {
			result.WriteString(r.RenderDiagnostic(diag))
		}
		
		if i < len(group.Diagnostics)-1 {
			result.WriteString("\n")
			if len(group.Diagnostics) <= 3 {
				result.WriteString("\n") // Extra spacing for full diagnostics
			}
		}
	}
	
	return result.String()
}

// Summary generates a summary line for multiple diagnostics
func (r *DiagnosticRenderer) Summary(diagnostics []Diagnostic) string {
	counts := make(map[DiagnosticLevel]int)
	for _, d := range diagnostics {
		counts[d.Level]++
	}
	
	var parts []string
	
	if c := counts[DiagnosticError]; c > 0 {
		style := lipgloss.NewStyle().Foreground(r.theme.Error())
		parts = append(parts, style.Render(fmt.Sprintf("%d error(s)", c)))
	}
	
	if c := counts[DiagnosticWarning]; c > 0 {
		style := lipgloss.NewStyle().Foreground(r.theme.Warning())
		parts = append(parts, style.Render(fmt.Sprintf("%d warning(s)", c)))
	}
	
	if c := counts[DiagnosticInfo]; c > 0 {
		style := lipgloss.NewStyle().Foreground(r.theme.Info())
		parts = append(parts, style.Render(fmt.Sprintf("%d info", c)))
	}
	
	if c := counts[DiagnosticHint]; c > 0 {
		style := lipgloss.NewStyle().Foreground(r.theme.Success())
		parts = append(parts, style.Render(fmt.Sprintf("%d hint(s)", c)))
	}
	
	if len(parts) == 0 {
		return "No diagnostics"
	}
	
	return strings.Join(parts, ", ")
}

// ParseDiagnosticContent parses diagnostic content from various formats
func ParseDiagnosticContent(content string) (*Diagnostic, error) {
	var diag Diagnostic
	
	// Try JSON first
	if err := json.Unmarshal([]byte(content), &diag); err == nil {
		return &diag, nil
	}
	
	// Try XML-style format for simple diagnostics
	// <diagnostic level="error" code="E001">Message here</diagnostic>
	if strings.HasPrefix(content, "<diagnostic") && strings.HasSuffix(content, "</diagnostic>") {
		// Extract content first
		start := strings.Index(content, ">")
		end := strings.LastIndex(content, "</diagnostic>")
		if start != -1 && end != -1 && start < end {
			innerContent := strings.TrimSpace(content[start+1 : end])
			
			// Try to parse inner content as JSON first
			if err := json.Unmarshal([]byte(innerContent), &diag); err == nil {
				return &diag, nil
			}
			
			// Otherwise parse XML attributes
			levelMatch := regexp.MustCompile(`level="([^"]+)"`).FindStringSubmatch(content)
			if len(levelMatch) > 1 {
				diag.Level = DiagnosticLevel(levelMatch[1])
			}
			
			codeMatch := regexp.MustCompile(`code="([^"]+)"`).FindStringSubmatch(content)
			if len(codeMatch) > 1 {
				diag.Code = codeMatch[1]
			}
			
			// Use inner content as message
			diag.Message = innerContent
			
			if diag.Level != "" && diag.Message != "" {
				return &diag, nil
			}
		}
	}
	
	// Fallback: treat as error message
	diag.Level = DiagnosticError
	diag.Message = content
	return &diag, nil
}