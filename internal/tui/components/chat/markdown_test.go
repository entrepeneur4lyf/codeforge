package chat

import (
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRegex.ReplaceAllString(s, "")
}

func TestMarkdownRenderer(t *testing.T) {
	th := theme.GetTheme("default")
	
	t.Run("creation and initialization", func(t *testing.T) {
		renderer, err := NewMarkdownRenderer(th, 80)
		if err != nil {
			t.Fatalf("Failed to create renderer: %v", err)
		}
		
		if renderer.width != 80 {
			t.Errorf("Expected width 80, got %d", renderer.width)
		}
		
		if renderer.cache == nil {
			t.Error("Expected cache to be initialized")
		}
		
		if renderer.renderer == nil {
			t.Error("Expected glamour renderer to be initialized")
		}
	})
	
	t.Run("basic rendering", func(t *testing.T) {
		renderer, err := NewMarkdownRenderer(th, 80)
		if err != nil {
			t.Fatalf("Failed to create renderer: %v", err)
		}
		
		tests := []struct {
			name     string
			input    string
			contains []string
		}{
			{
				name:     "plain text",
				input:    "Hello, world!",
				contains: []string{"Hello, world!"},
			},
			{
				name:     "heading",
				input:    "# Main Title",
				contains: []string{"Main Title"},
			},
			{
				name:     "bold text",
				input:    "This is **bold** text",
				contains: []string{"bold"},
			},
			{
				name:     "italic text",
				input:    "This is *italic* text",
				contains: []string{"italic"},
			},
			{
				name:     "code inline",
				input:    "Use `fmt.Println()` to print",
				contains: []string{"fmt.Println()"},
			},
			{
				name:     "list items",
				input:    "- Item 1\n- Item 2\n- Item 3",
				contains: []string{"Item 1", "Item 2", "Item 3"},
			},
			{
				name:     "code block",
				input:    "```go\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```",
				contains: []string{"func main()", "fmt.Println"},
			},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := renderer.Render(tt.input)
				if err != nil {
					t.Errorf("Render failed: %v", err)
				}
				
				for _, expected := range tt.contains {
					// Strip ANSI codes for comparison
					strippedResult := stripANSI(result)
					if !strings.Contains(strippedResult, expected) {
						t.Errorf("Expected result to contain %q, got: %s", expected, strippedResult)
					}
				}
			})
		}
	})
	
	t.Run("width adjustment", func(t *testing.T) {
		renderer, err := NewMarkdownRenderer(th, 80)
		if err != nil {
			t.Fatalf("Failed to create renderer: %v", err)
		}
		
		// Cache something
		content := "# Test Header"
		result1, _ := renderer.Render(content)
		
		// Change width
		err = renderer.SetWidth(100)
		if err != nil {
			t.Errorf("Failed to set width: %v", err)
		}
		
		if renderer.width != 100 {
			t.Errorf("Expected width 100, got %d", renderer.width)
		}
		
		// Render again - cache should be cleared
		result2, _ := renderer.Render(content)
		
		// Results might differ due to width change
		_ = result1
		_ = result2
		
		// Test invalid width
		err = renderer.SetWidth(0)
		if err == nil {
			t.Error("Expected error for zero width")
		}
		
		err = renderer.SetWidth(-10)
		if err == nil {
			t.Error("Expected error for negative width")
		}
	})
	
	t.Run("empty content", func(t *testing.T) {
		renderer, err := NewMarkdownRenderer(th, 80)
		if err != nil {
			t.Fatalf("Failed to create renderer: %v", err)
		}
		
		result, err := renderer.Render("")
		if err != nil {
			t.Errorf("Expected no error for empty content, got: %v", err)
		}
		
		if result != "" {
			t.Errorf("Expected empty result for empty content, got: %s", result)
		}
	})
	
	t.Run("cache functionality", func(t *testing.T) {
		renderer, err := NewMarkdownRenderer(th, 80)
		if err != nil {
			t.Fatalf("Failed to create renderer: %v", err)
		}
		
		content := "# Cached Content\n\nThis should be cached."
		
		// First render
		result1, err := renderer.Render(content)
		if err != nil {
			t.Errorf("First render failed: %v", err)
		}
		
		// Second render (should hit cache)
		result2, err := renderer.Render(content)
		if err != nil {
			t.Errorf("Second render failed: %v", err)
		}
		
		// Results should be identical
		if result1 != result2 {
			t.Error("Expected cached result to be identical")
		}
		
		// Clear cache
		renderer.Clear()
		
		// Third render (cache miss)
		result3, err := renderer.Render(content)
		if err != nil {
			t.Errorf("Third render failed: %v", err)
		}
		
		// Content should still be the same
		if result1 != result3 {
			t.Error("Expected consistent rendering after cache clear")
		}
	})
	
	t.Run("code block rendering", func(t *testing.T) {
		renderer, err := NewMarkdownRenderer(th, 80)
		if err != nil {
			t.Fatalf("Failed to create renderer: %v", err)
		}
		
		code := `func main() {
    fmt.Println("Hello, World!")
}`
		
		result, err := renderer.RenderCodeBlock(code, "go")
		if err != nil {
			t.Errorf("RenderCodeBlock failed: %v", err)
		}
		
		// Should contain the code (strip ANSI for comparison)
		strippedResult := stripANSI(result)
		if !strings.Contains(strippedResult, "func main()") {
			t.Errorf("Expected code block to contain function, got: %s", strippedResult)
		}
		
		if !strings.Contains(strippedResult, "fmt.Println") {
			t.Errorf("Expected code block to contain print statement, got: %s", strippedResult)
		}
	})
	
	t.Run("fallback rendering", func(t *testing.T) {
		renderer, err := NewMarkdownRenderer(th, 80)
		if err != nil {
			t.Fatalf("Failed to create renderer: %v", err)
		}
		
		// Test the fallback directly
		content := `# Header
		
- List item 1
- List item 2

Regular paragraph text.

1. Numbered item
2. Another numbered item

` + "```" + `
Code block
` + "```"
		
		result := renderer.renderFallback(content)
		
		// Check basic elements are preserved
		if !strings.Contains(result, "Header") {
			t.Error("Expected header in fallback")
		}
		
		if !strings.Contains(result, "• List item 1") {
			t.Error("Expected bullet list in fallback")
		}
		
		if !strings.Contains(result, "Regular paragraph") {
			t.Error("Expected paragraph in fallback")
		}
	})
	
	t.Run("inline markdown", func(t *testing.T) {
		renderer, err := NewMarkdownRenderer(th, 80)
		if err != nil {
			t.Fatalf("Failed to create renderer: %v", err)
		}
		
		tests := []struct {
			name  string
			input string
		}{
			{"bold", "**bold text**"},
			{"italic", "*italic text*"},
			{"code", "`inline code`"},
			{"strikethrough", "~~struck through~~"},
			{"mixed", "This has **bold** and *italic* and `code`"},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := renderer.RenderInline(tt.input)
				if err != nil {
					t.Errorf("RenderInline failed: %v", err)
				}
				
				// For now, just check it doesn't error
				// and returns something
				if result == "" {
					t.Error("Expected non-empty result")
				}
			})
		}
	})
}

func TestChatModelMarkdownIntegration(t *testing.T) {
	th := theme.GetTheme("default")
	
	t.Run("markdown enabled by default", func(t *testing.T) {
		chat := NewChatModel(th, "test-session")
		
		if !chat.IsMarkdownEnabled() {
			t.Error("Expected markdown to be enabled by default")
		}
		
		if chat.markdown == nil {
			t.Error("Expected markdown renderer to be initialized")
		}
	})
	
	t.Run("enable/disable markdown", func(t *testing.T) {
		chat := NewChatModel(th, "test-session")
		
		// Disable
		chat.EnableMarkdown(false)
		if chat.IsMarkdownEnabled() {
			t.Error("Expected markdown to be disabled")
		}
		
		// Re-enable
		chat.EnableMarkdown(true)
		if !chat.IsMarkdownEnabled() {
			t.Error("Expected markdown to be enabled")
		}
	})
	
	t.Run("markdown rendering in messages", func(t *testing.T) {
		chat := NewChatModel(th, "test-session")
		// Initialize properly
		chat.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add assistant message with markdown
		msg := Message{
			ID:      "test-1",
			Role:    "assistant",
			Content: "# Hello\n\nThis is **bold** and this is *italic*.\n\n```go\nfmt.Println(\"Code\")\n```",
		}
		
		chat.messages = append(chat.messages, msg)
		chat.updateViewport()
		
		// Get viewport content
		content := chat.viewport.View()
		
		// Should contain formatted content (strip ANSI for comparison)
		strippedContent := stripANSI(content)
		if !strings.Contains(strippedContent, "Hello") {
			t.Errorf("Expected heading to be rendered, got: %s", strippedContent)
		}
		
		// Note: Exact formatting depends on glamour and terminal
	})
	
	t.Run("plain text when markdown disabled", func(t *testing.T) {
		chat := NewChatModel(th, "test-session")
		// Initialize properly
		chat.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		chat.EnableMarkdown(false)
		
		// Add assistant message with markdown
		msg := Message{
			ID:      "test-1",
			Role:    "assistant",
			Content: "# Hello\n\nThis is **bold** text.",
		}
		
		chat.messages = append(chat.messages, msg)
		chat.updateViewport()
		
		// Get viewport content
		content := chat.viewport.View()
		
		// Should contain raw markdown (strip ANSI for comparison)
		strippedContent := stripANSI(content)
		if !strings.Contains(strippedContent, "# Hello") {
			t.Errorf("Expected raw markdown when disabled, got: %s", strippedContent)
		}
	})
}