package markdown

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
)

// RendererConfig holds configuration for markdown rendering
type RendererConfig struct {
	Width           int
	Theme           string
	WordWrap        bool
	PreserveIndent  bool
	ColorProfile    string
	BackgroundColor string
}

// DefaultConfig returns a default renderer configuration
func DefaultConfig() *RendererConfig {
	return &RendererConfig{
		Width:           80,
		Theme:           "dark",
		WordWrap:        true,
		PreserveIndent:  true,
		ColorProfile:    "TrueColor",
		BackgroundColor: "#1a1a1a",
	}
}

// WebConfig returns a configuration optimized for web display
func WebConfig() *RendererConfig {
	return &RendererConfig{
		Width:           120,
		Theme:           "auto",
		WordWrap:        true,
		PreserveIndent:  true,
		ColorProfile:    "TrueColor",
		BackgroundColor: "#ffffff",
	}
}

// ChatConfig returns a configuration optimized for chat messages
func ChatConfig() *RendererConfig {
	return &RendererConfig{
		Width:           100,
		Theme:           "dark",
		WordWrap:        true,
		PreserveIndent:  false,
		ColorProfile:    "TrueColor",
		BackgroundColor: "#0d1117",
	}
}

// Renderer wraps glamour with CodeForge-specific configuration
type Renderer struct {
	glamourRenderer *glamour.TermRenderer
	config          *RendererConfig
}

// NewRenderer creates a new markdown renderer with the given configuration
func NewRenderer(config *RendererConfig) (*Renderer, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Create glamour renderer with built-in theme
	glamourRenderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(config.Width),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create glamour renderer: %w", err)
	}

	return &Renderer{
		glamourRenderer: glamourRenderer,
		config:          config,
	}, nil
}

// NewChatRenderer creates a renderer optimized for chat messages
func NewChatRenderer() (*Renderer, error) {
	return NewRenderer(ChatConfig())
}

// NewWebRenderer creates a renderer optimized for web display
func NewWebRenderer() (*Renderer, error) {
	return NewRenderer(WebConfig())
}

// Render renders markdown content to styled terminal output
func (r *Renderer) Render(markdown string) (string, error) {
	if markdown == "" {
		return "", nil
	}

	// Pre-process markdown for better chat display
	processed := r.preprocessMarkdown(markdown)

	// Render with glamour
	rendered, err := r.glamourRenderer.Render(processed)
	if err != nil {
		return "", fmt.Errorf("failed to render markdown: %w", err)
	}

	// Post-process for chat optimization
	return r.postprocessOutput(rendered), nil
}

// RenderToHTML renders markdown content to HTML (for web API responses)
func (r *Renderer) RenderToHTML(markdown string) (string, error) {
	// For HTML output, we'll use a simple approach
	// In a production system, you might want to use a dedicated HTML renderer
	rendered, err := r.Render(markdown)
	if err != nil {
		return "", err
	}

	// Convert ANSI codes to HTML (basic implementation)
	html := r.ansiToHTML(rendered)
	return html, nil
}

// preprocessMarkdown optimizes markdown for chat display
func (r *Renderer) preprocessMarkdown(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var processed []string

	for _, line := range lines {
		// Trim excessive whitespace but preserve code block formatting
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			processed = append(processed, line)
		} else if strings.TrimSpace(line) == "" {
			processed = append(processed, "")
		} else {
			processed = append(processed, strings.TrimRight(line, " \t"))
		}
	}

	return strings.Join(processed, "\n")
}

// postprocessOutput optimizes rendered output for chat display
func (r *Renderer) postprocessOutput(rendered string) string {
	// Remove excessive blank lines
	lines := strings.Split(rendered, "\n")
	var result []string
	blankCount := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount <= 1 { // Allow max 1 consecutive blank line
				result = append(result, line)
			}
		} else {
			blankCount = 0
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// ansiToHTML converts ANSI escape codes to basic HTML
func (r *Renderer) ansiToHTML(ansiText string) string {
	// Basic ANSI to HTML conversion
	// This is a simplified implementation - for production use, consider a more robust solution
	html := ansiText

	// Remove ANSI escape codes for now (basic implementation)
	// In production, you'd want to convert them to proper HTML/CSS
	html = strings.ReplaceAll(html, "\x1b[", "")
	html = strings.ReplaceAll(html, "\033[", "")

	// Wrap in basic HTML structure
	return fmt.Sprintf(`<div class="markdown-content">%s</div>`, html)
}

// Helper functions for style configuration (kept for potential future use)
func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }
func uintPtr(u uint) *uint       { return &u }
