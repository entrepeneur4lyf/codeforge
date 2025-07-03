package chat

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

func TestBlockRenderer(t *testing.T) {
	// Use default theme for testing
	th := theme.GetTheme("default")
	
	t.Run("basic rendering", func(t *testing.T) {
		renderer := NewBlockRenderer(th)
		result := renderer.Render("Hello, World!")
		
		if !strings.Contains(result, "Hello, World!") {
			t.Errorf("Expected content to be rendered, got: %s", result)
		}
	})
	
	t.Run("with options", func(t *testing.T) {
		renderer := NewBlockRenderer(th,
			WithTextColor(th.Primary()),
			WithBackgroundColor(th.BackgroundSecondary()),
			WithPadding(2),
			WithBold(),
			WithItalic(),
		)
		
		result := renderer.Render("Styled text")
		
		// Should contain the content
		if !strings.Contains(result, "Styled text") {
			t.Errorf("Expected content to be rendered, got: %s", result)
		}
		
		// Should have styling applied (check ANSI codes exist)
		if len(result) <= len("Styled text") {
			t.Error("Expected styling to be applied")
		}
	})
	
	t.Run("no border option", func(t *testing.T) {
		renderer := NewBlockRenderer(th, WithNoBorder())
		result := renderer.Render("No border")
		
		// Should not contain border characters
		borderChars := []string{"─", "│", "┌", "┐", "└", "┘", "╭", "╮", "╰", "╯"}
		for _, char := range borderChars {
			if strings.Contains(result, char) {
				t.Errorf("Expected no border, but found border character: %s", char)
			}
		}
	})
	
	t.Run("custom border", func(t *testing.T) {
		renderer := NewBlockRenderer(th,
			WithBorder(lipgloss.DoubleBorder(), th.Error()),
		)
		result := renderer.Render("Custom border")
		
		// Should contain double border characters
		if !strings.Contains(result, "═") || !strings.Contains(result, "║") {
			t.Error("Expected double border characters")
		}
	})
	
	t.Run("margin and padding", func(t *testing.T) {
		renderer := NewBlockRenderer(th,
			WithNoBorder(),
			WithPadding(1),
			WithMargin(1),
		)
		result := renderer.Render("Spaced")
		
		lines := strings.Split(result, "\n")
		// Should have extra lines from margin
		if len(lines) < 3 { // 1 margin top + content + 1 margin bottom
			t.Errorf("Expected margins to add lines, got %d lines", len(lines))
		}
	})
	
	t.Run("width constraints", func(t *testing.T) {
		renderer := NewBlockRenderer(th,
			WithNoBorder(),
			WithWidth(20), // Set fixed width to force wrapping
		)
		
		longText := "This is a very long text that should wrap when rendered"
		result := renderer.Render(longText)
		
		// Check that content is preserved
		if !strings.Contains(result, "This is a very long") {
			t.Error("Expected content to be preserved with width constraint")
		}
		
		// Check width
		width := lipgloss.Width(result)
		if width != 20 {
			t.Errorf("Expected width to be 20, got %d", width)
		}
	})
}

func TestPresetRenderers(t *testing.T) {
	th := theme.GetTheme("default")
	
	t.Run("UserMessageRenderer", func(t *testing.T) {
		renderer := UserMessageRenderer(th)
		result := renderer.Render("User message")
		
		if !strings.Contains(result, "User message") {
			t.Error("Expected user message content")
		}
		
		// Should have border
		if !strings.Contains(result, "─") {
			t.Error("Expected user message to have border")
		}
	})
	
	t.Run("AssistantMessageRenderer", func(t *testing.T) {
		renderer := AssistantMessageRenderer(th)
		result := renderer.Render("Assistant message")
		
		if !strings.Contains(result, "Assistant message") {
			t.Error("Expected assistant message content")
		}
		
		// Should have border
		if !strings.Contains(result, "─") {
			t.Error("Expected assistant message to have border")
		}
	})
	
	t.Run("SystemMessageRenderer", func(t *testing.T) {
		renderer := SystemMessageRenderer(th)
		result := renderer.Render("System message")
		
		if !strings.Contains(result, "System message") {
			t.Error("Expected system message content")
		}
		
		// Should NOT have border
		borderChars := []string{"─", "│", "┌", "┐", "└", "┘"}
		for _, char := range borderChars {
			if strings.Contains(result, char) {
				t.Error("Expected system message to have no border")
			}
		}
	})
	
	t.Run("ErrorMessageRenderer", func(t *testing.T) {
		renderer := ErrorMessageRenderer(th)
		result := renderer.Render("Error occurred")
		
		if !strings.Contains(result, "Error occurred") {
			t.Error("Expected error message content")
		}
		
		// Should have double border
		if !strings.Contains(result, "═") {
			t.Error("Expected error message to have double border")
		}
	})
	
	t.Run("CodeBlockRenderer", func(t *testing.T) {
		renderer := CodeBlockRenderer(th)
		result := renderer.Render("func main() {}")
		
		if !strings.Contains(result, "func main() {}") {
			t.Error("Expected code block content")
		}
		
		// Should have normal border
		if !strings.Contains(result, "─") || !strings.Contains(result, "│") {
			t.Error("Expected code block to have normal border")
		}
	})
}

func TestRenderOptions(t *testing.T) {
	th := theme.GetTheme("default")
	
	t.Run("text decorations", func(t *testing.T) {
		decorations := []struct {
			name   string
			option RenderOption
		}{
			{"bold", WithBold()},
			{"italic", WithItalic()},
			{"underline", WithUnderline()},
			{"strikethrough", WithStrikethrough()},
		}
		
		for _, dec := range decorations {
			t.Run(dec.name, func(t *testing.T) {
				renderer := NewBlockRenderer(th, WithNoBorder(), dec.option)
				result := renderer.Render("Decorated")
				
				// Should contain the content at minimum
				if !strings.Contains(result, "Decorated") {
					t.Errorf("Expected %s decoration to preserve content", dec.name)
				}
			})
		}
	})
	
	t.Run("alignment", func(t *testing.T) {
		alignments := []struct {
			name  string
			align lipgloss.Position
		}{
			{"left", lipgloss.Left},
			{"center", lipgloss.Center},
			{"right", lipgloss.Right},
		}
		
		for _, a := range alignments {
			t.Run(a.name, func(t *testing.T) {
				renderer := NewBlockRenderer(th,
					WithNoBorder(),
					WithAlign(a.align),
					WithWidth(20),
				)
				result := renderer.Render("Aligned")
				
				// Basic check that width was applied
				if lipgloss.Width(result) != 20 {
					t.Errorf("Expected width 20 for %s alignment, got %d", a.name, lipgloss.Width(result))
				}
			})
		}
	})
	
	t.Run("directional padding", func(t *testing.T) {
		renderer := NewBlockRenderer(th,
			WithNoBorder(),
			WithPaddingX(3),
			WithPaddingY(1),
		)
		
		result := renderer.Render("XY")
		lines := strings.Split(result, "\n")
		
		// Should have vertical padding (3 lines: 1 top + content + 1 bottom)
		if len(lines) != 3 {
			t.Errorf("Expected 3 lines with Y padding, got %d", len(lines))
		}
		
		// Content line should have horizontal padding
		contentLine := lines[1]
		if !strings.HasPrefix(contentLine, "   ") || !strings.HasSuffix(contentLine, "   ") {
			t.Error("Expected horizontal padding of 3 spaces")
		}
	})
}