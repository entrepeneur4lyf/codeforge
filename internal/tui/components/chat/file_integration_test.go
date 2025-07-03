package chat

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

func TestFileRenderingIntegration(t *testing.T) {
	th := theme.GetTheme("default")
	
	t.Run("message with file block", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
		
		// Add message with file content
		testMessages := []Message{
			{
				ID:   "msg-1",
				Role: "assistant",
				Content: `Here's the implementation:

` + "```file:main.go:10-15\n" + `func Hello() string {
	return "Hello, World!"
}

func main() {
	fmt.Println(Hello())
}` + "\n```\n" + `

This function returns a greeting.`,
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Force width for file renderer
		mc.fileRenderer.SetMaxWidth(100)
		
		// Check parsing
		if len(mc.messageParts) != 1 {
			t.Fatalf("Expected 1 message, got %d", len(mc.messageParts))
		}
		
		parts := mc.messageParts[0]
		
		// Should have text, file, text
		if len(parts) < 3 {
			t.Errorf("Expected at least 3 parts, got %d", len(parts))
		}
		
		// Debug: print all parts
		for i, part := range parts {
			contentPreview := part.Content
			if len(contentPreview) > 50 {
				contentPreview = contentPreview[:50] + "..."
			}
			t.Logf("Part %d: Type=%s, Content=%q, Metadata=%v", i, part.Type, contentPreview, part.Metadata)
		}
		
		// Find file part
		var filePart *MessagePart
		for i, part := range parts {
			if part.Type == PartTypeFile {
				filePart = &parts[i]
				break
			}
		}
		
		if filePart == nil {
			t.Fatal("No file part found")
		}
		
		// Check metadata
		if path, ok := filePart.Metadata["path"].(string); !ok || path != "main.go:10-15" {
			t.Errorf("Expected path metadata 'main.go:10-15', got %v", filePart.Metadata["path"])
		}
		
		// Get rendered view
		view := msgs.ViewWithSize(120, 50)
		strippedView := stripANSI(view)
		
		// Should contain file path
		if !strings.Contains(strippedView, "main.go") {
			t.Error("Expected file path in view")
		}
		
		// Should contain line range
		if !strings.Contains(strippedView, "(lines 10-15)") {
			t.Error("Expected line range in view")
		}
		
		// Should contain file content
		if !strings.Contains(strippedView, "func Hello()") {
			viewPreview := strippedView
			if len(viewPreview) > 500 {
				viewPreview = viewPreview[:500] + "..."
			}
			t.Errorf("Expected function declaration in view. View preview: %s", viewPreview)
		}
		
		// Should have line numbers (since the content is shorter than the requested range,
		// it should show the actual line numbers 1-7)
		if !strings.Contains(strippedView, "1 │") {
			t.Error("Expected line number 1")
		}
		
		if !strings.Contains(strippedView, "6 │") {
			t.Error("Expected line number 6")
		}
	})
	
	t.Run("file with single line reference", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
		
		// Add message with single line file reference
		testMessages := []Message{
			{
				ID:   "msg-2",
				Role: "assistant",
				Content: "The error is here:\n\n```file:error.go:42\nreturn fmt.Errorf(\"invalid input: %v\", input)\n```",
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Get rendered view
		view := msgs.ViewWithSize(100, 30)
		strippedView := stripANSI(view)
		
		// Should show single line (since the content is only 1 line, it should show line 1)
		if !strings.Contains(strippedView, "1 │") {
			t.Error("Expected line 1")
		}
	})
	
	t.Run("file without line numbers", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Disable line numbers
		mc.fileRenderer.SetShowLineNumbers(false)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
		
		// Add message with file
		testMessages := []Message{
			{
				ID:   "msg-3",
				Role: "assistant",
				Content: "```file:config.json\n{\n  \"name\": \"test\",\n  \"version\": \"1.0\"\n}\n```",
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Get rendered view
		view := msgs.ViewWithSize(100, 30)
		strippedView := stripANSI(view)
		
		// Should contain content but no line numbers
		if !strings.Contains(strippedView, "config.json") {
			t.Error("Expected file name")
		}
		
		if !strings.Contains(strippedView, "\"name\": \"test\"") {
			t.Error("Expected JSON content")
		}
		
		// Should not have line numbers
		if strings.Contains(strippedView, " 1 │ ") {
			t.Error("Should not have line numbers when disabled")
		}
	})
	
	t.Run("selection mode with file", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
		
		// Add message with file
		testMessages := []Message{
			{
				ID:   "msg-4",
				Role: "assistant",
				Content: "Check this:\n\n```file:main.py\nprint('Hello')\n```\n\nDone.",
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Enter selection mode
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		
		// Navigate to file part
		var foundFile bool
		for i := 0; i < 10; i++ {
			part, ok := mc.GetSelectedPart()
			if ok && part.Type == PartTypeFile {
				foundFile = true
				break
			}
			mc.selectNextPart()
		}
		
		if !foundFile {
			t.Error("Could not navigate to file part")
		}
		
		// Check that selection indicator is shown
		view := msgs.ViewWithSize(100, 30)
		strippedView := stripANSI(view)
		
		if !strings.Contains(strippedView, "▸ Selected") {
			t.Error("Expected selection indicator for file")
		}
	})
	
	t.Run("multiple file blocks", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
		
		// Add message with multiple files
		testMessages := []Message{
			{
				ID:   "msg-5",
				Role: "assistant",
				Content: `Here are the files:

` + "```file:server.go:1-5\n" + `package main

import (
	"fmt"
	"net/http"
)` + "\n```\n\n" + "```file:handler.go:10-12\n" + `func HandleRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello!")
}` + "\n```",
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Count file parts
		fileCount := 0
		for _, part := range mc.messageParts[0] {
			if part.Type == PartTypeFile {
				fileCount++
			}
		}
		
		if fileCount != 2 {
			t.Errorf("Expected 2 file parts, got %d", fileCount)
		}
		
		// Get rendered view
		view := msgs.ViewWithSize(120, 50)
		strippedView := stripANSI(view)
		
		// Should contain both files
		if !strings.Contains(strippedView, "server.go") {
			t.Error("Expected server.go")
		}
		
		if !strings.Contains(strippedView, "handler.go") {
			t.Error("Expected handler.go")
		}
	})
}