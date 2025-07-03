package chat

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/styles"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// MarkdownRenderer wraps glamour for rendering markdown in chat messages
type MarkdownRenderer struct {
	theme    theme.Theme
	renderer *glamour.TermRenderer
	width    int
	cache    *MessageCache
}

// NewMarkdownRenderer creates a new markdown renderer
func NewMarkdownRenderer(theme theme.Theme, width int) (*MarkdownRenderer, error) {
	// Use the existing markdown styles from the styles package
	renderer := styles.GetMarkdownRenderer(width)
	if renderer == nil {
		return nil, fmt.Errorf("failed to create markdown renderer")
	}

	return &MarkdownRenderer{
		theme:    theme,
		renderer: renderer,
		width:    width,
		cache:    NewMessageCache(50), // Cache rendered markdown
	}, nil
}

// SetWidth updates the renderer width
func (m *MarkdownRenderer) SetWidth(width int) error {
	if width <= 0 {
		return fmt.Errorf("width must be positive, got %d", width)
	}
	
	if width == m.width {
		return nil // No change needed
	}
	
	m.width = width
	
	// Recreate renderer with new width
	m.renderer = styles.GetMarkdownRenderer(width)
	if m.renderer == nil {
		return fmt.Errorf("failed to recreate renderer with width %d", width)
	}
	
	// Clear cache since width changed
	m.cache.Clear()
	
	return nil
}

// Render converts markdown to styled terminal output
func (m *MarkdownRenderer) Render(content string) (string, error) {
	if content == "" {
		return "", nil
	}
	
	// Generate cache key
	cacheKey := m.cache.GenerateKey(content, m.width, "markdown")
	
	// Check cache
	if rendered, found := m.cache.Get(cacheKey); found {
		return rendered, nil
	}
	
	// Render with glamour
	rendered, err := m.renderer.Render(content)
	if err != nil {
		// Fallback to plain text on error
		return m.renderFallback(content), nil
	}
	
	// Trim excessive newlines that glamour sometimes adds
	rendered = strings.TrimSpace(rendered)
	
	// Cache the result
	m.cache.Set(cacheKey, rendered)
	
	return rendered, nil
}

// RenderInline renders markdown without block-level formatting
func (m *MarkdownRenderer) RenderInline(content string) (string, error) {
	// For inline rendering, we'll strip certain block elements
	// but preserve inline formatting like bold, italic, code
	
	// Generate cache key
	cacheKey := m.cache.GenerateKey(content, m.width, "inline")
	
	// Check cache
	if rendered, found := m.cache.Get(cacheKey); found {
		return rendered, nil
	}
	
	// Process inline elements manually for better control
	rendered := m.processInlineMarkdown(content)
	
	// Cache the result
	m.cache.Set(cacheKey, rendered)
	
	return rendered, nil
}

// processInlineMarkdown handles basic inline markdown elements
func (m *MarkdownRenderer) processInlineMarkdown(content string) string {
	// This is a simplified inline processor
	// In production, you might want to use a proper markdown parser
	
	style := lipgloss.NewStyle()
	
	// Bold: **text** or __text__
	content = processInlinePattern(content, `\*\*([^*]+)\*\*`, func(match string) string {
		return style.Bold(true).Render(match)
	})
	content = processInlinePattern(content, `__([^_]+)__`, func(match string) string {
		return style.Bold(true).Render(match)
	})
	
	// Italic: *text* or _text_
	content = processInlinePattern(content, `\*([^*]+)\*`, func(match string) string {
		return style.Italic(true).Render(match)
	})
	content = processInlinePattern(content, `_([^_]+)_`, func(match string) string {
		return style.Italic(true).Render(match)
	})
	
	// Code: `text`
	content = processInlinePattern(content, "`([^`]+)`", func(match string) string {
		return style.
			Background(m.theme.BackgroundSecondary()).
			Foreground(m.theme.Text()).
			Render(match)
	})
	
	// Strikethrough: ~~text~~
	content = processInlinePattern(content, `~~([^~]+)~~`, func(match string) string {
		return style.Strikethrough(true).Render(match)
	})
	
	return content
}

// processInlinePattern is a helper to process markdown patterns
func processInlinePattern(content, pattern string, styler func(string) string) string {
	// This is a simplified implementation
	// In production, use a proper regex with backreferences
	// For now, we'll just return the content as-is
	// TODO: Implement proper pattern matching
	return content
}

// renderFallback provides a simple fallback rendering
func (m *MarkdownRenderer) renderFallback(content string) string {
	// Basic fallback that preserves some structure
	lines := strings.Split(content, "\n")
	var result []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Headers
		if strings.HasPrefix(trimmed, "#") {
			level := strings.Count(strings.Split(trimmed, " ")[0], "#")
			text := strings.TrimSpace(strings.TrimPrefix(trimmed, strings.Repeat("#", level)))
			styled := lipgloss.NewStyle().
				Bold(true).
				Foreground(m.theme.Primary()).
				Render(text)
			result = append(result, styled)
			continue
		}
		
		// Code blocks (simple detection)
		if strings.HasPrefix(trimmed, "```") {
			result = append(result, lipgloss.NewStyle().
				Foreground(m.theme.TextMuted()).
				Render(line))
			continue
		}
		
		// Lists
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			text := strings.TrimSpace(trimmed[2:])
			result = append(result, "• "+text)
			continue
		}
		
		// Numbered lists
		if len(trimmed) > 2 && trimmed[1] == '.' && trimmed[0] >= '0' && trimmed[0] <= '9' {
			result = append(result, line)
			continue
		}
		
		// Regular text
		result = append(result, line)
	}
	
	return strings.Join(result, "\n")
}

// RenderCodeBlock renders a code block with syntax highlighting
func (m *MarkdownRenderer) RenderCodeBlock(code, language string) (string, error) {
	// Wrap in markdown code block and render
	markdown := fmt.Sprintf("```%s\n%s\n```", language, code)
	return m.Render(markdown)
}

// Clear clears the markdown cache
func (m *MarkdownRenderer) Clear() {
	m.cache.Clear()
}