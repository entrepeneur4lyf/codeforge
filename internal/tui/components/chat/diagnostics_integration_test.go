package chat

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

func TestDiagnosticIntegration(t *testing.T) {
	th := theme.GetTheme("default")

	t.Run("message with diagnostic", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add message with diagnostic
		testMessages := []Message{
			{
				ID:   "msg-1",
				Role: "assistant",
				Content: `I found an error in your code:

<diagnostic level="error" code="E0001">Undefined variable 'foo'</diagnostic>

You need to define 'foo' before using it.`,
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Check parsing
		if len(mc.messageParts) != 1 {
			t.Fatalf("Expected 1 message parts array, got %d", len(mc.messageParts))
		}
		
		parts := mc.messageParts[0]
		
		// Should have text, diagnostic, text
		if len(parts) < 3 {
			t.Errorf("Expected at least 3 parts, got %d", len(parts))
		}
		
		// Find diagnostic part
		var diagPart *MessagePart
		for i, part := range parts {
			if part.Type == PartTypeDiagnostic {
				diagPart = &parts[i]
				break
			}
		}
		
		if diagPart == nil {
			t.Fatal("No diagnostic part found")
		}
		
		// Get rendered view
		view := msgs.ViewWithSize(100, 50)
		strippedView := stripANSI(view)
		
		// Should contain error indicator
		if !strings.Contains(strippedView, "ERROR") {
			t.Error("Expected ERROR in view")
		}
		
		// Should contain error code
		if !strings.Contains(strippedView, "[E0001]") {
			t.Error("Expected error code in view")
		}
		
		// Should contain message
		if !strings.Contains(strippedView, "Undefined variable") {
			t.Error("Expected error message in view")
		}
	})

	t.Run("JSON diagnostic", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
		
		// Add message with JSON diagnostic
		diagJSON := `{"level":"error","code":"E0042","title":"Type mismatch","message":"Cannot assign string to int variable","file":"main.go","line":42,"column":15,"context":["var x int = 10","x = \"hello\"","fmt.Println(x)"],"suggestions":["Convert the string to int using strconv.Atoi()","Change the variable type to string"]}`
		
		testMessages := []Message{
			{
				ID:   "msg-2",
				Role: "assistant",
				Content: fmt.Sprintf(`Found a type error:

<diagnostic>%s</diagnostic>

Please fix this type mismatch.`, diagJSON),
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Get rendered view
		view := msgs.ViewWithSize(120, 50)
		strippedView := stripANSI(view)
		
		// Debug: print the full view
		if testing.Verbose() {
			fmt.Printf("JSON diagnostic view:\n%s\n", strippedView)
			fmt.Printf("View length: %d\n", len(strippedView))
		}
		
		// Should contain all diagnostic components
		if !strings.Contains(strippedView, "Type mismatch") {
			t.Error("Expected diagnostic title")
		}
		
		if !strings.Contains(strippedView, "main.go:42:15") {
			t.Error("Expected file location")
		}
		
		if !strings.Contains(strippedView, "x = \"hello\"") {
			t.Error("Expected error line in context")
		}
		
		if !strings.Contains(strippedView, "Convert the string") {
			t.Error("Expected suggestion")
		}
	})

	t.Run("multiple diagnostics", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add message with multiple diagnostics
		testMessages := []Message{
			{
				ID:   "msg-3",
				Role: "assistant",
				Content: `I found several issues:

<diagnostic level="error" code="E001">Syntax error: missing semicolon</diagnostic>

<diagnostic level="warning" code="W001">Unused variable 'temp'</diagnostic>

<diagnostic level="info">Consider using const for immutable values</diagnostic>

Please review these issues.`,
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Count diagnostic parts
		diagCount := 0
		for _, part := range mc.messageParts[0] {
			if part.Type == PartTypeDiagnostic {
				diagCount++
			}
		}
		
		if diagCount != 3 {
			t.Errorf("Expected 3 diagnostic parts, got %d", diagCount)
		}
		
		// Get rendered view
		view := msgs.ViewWithSize(100, 50)
		strippedView := stripANSI(view)
		
		// Should contain all levels
		if !strings.Contains(strippedView, "ERROR") {
			t.Error("Expected ERROR level")
		}
		if !strings.Contains(strippedView, "WARNING") {
			t.Error("Expected WARNING level")
		}
		if !strings.Contains(strippedView, "INFO") {
			t.Error("Expected INFO level")
		}
	})

	t.Run("selection mode with diagnostics", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add message with diagnostic
		testMessages := []Message{
			{
				ID:   "msg-4",
				Role: "assistant",
				Content: `Error found:

<diagnostic level="error">Test error message</diagnostic>

Fix needed.`,
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Enter selection mode
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		
		// Navigate to diagnostic part
		var diagPart *MessagePart
		for i := 0; i < 10; i++ {
			part, ok := mc.GetSelectedPart()
			if ok && part.Type == PartTypeDiagnostic {
				diagPart = &part
				break
			}
			mc.selectNextPart()
		}
		
		if diagPart == nil {
			t.Error("Could not navigate to diagnostic part")
			// Debug: show all parts
			if len(mc.messageParts) > 0 {
				for msgIdx, parts := range mc.messageParts {
					fmt.Printf("Message %d parts:\n", msgIdx)
					for partIdx, part := range parts {
						fmt.Printf("  Part %d: Type=%s, Content=%q\n", partIdx, part.Type, part.Content)
					}
				}
			}
			return
		}
		
		// Check that selection indicator is shown
		view := msgs.ViewWithSize(100, 50)
		strippedView := stripANSI(view)
		
		if testing.Verbose() {
			fmt.Printf("Selection mode view:\n%s\n", strippedView)
			fmt.Printf("Selected part: %+v\n", diagPart)
			fmt.Printf("Selected indices: msg=%d, part=%d\n", mc.selectedMessageIndex, mc.selectedPartIndex)
		}
		
		if !strings.Contains(strippedView, "▸ Selected") {
			t.Error("Expected selection indicator for diagnostic")
		}
	})
}

func TestDiagnosticParsing(t *testing.T) {
	t.Run("parse XML diagnostic", func(t *testing.T) {
		content := `<diagnostic level="warning" code="W123">Variable 'x' is declared but never used</diagnostic>`
		
		diag, err := ParseDiagnosticContent(content)
		if err != nil {
			t.Fatalf("Failed to parse diagnostic: %v", err)
		}
		
		if diag.Level != DiagnosticWarning {
			t.Errorf("Expected warning level, got %s", diag.Level)
		}
		
		if diag.Code != "W123" {
			t.Errorf("Expected code W123, got %s", diag.Code)
		}
		
		if !strings.Contains(diag.Message, "never used") {
			t.Error("Expected message about unused variable")
		}
	})

	t.Run("parse JSON diagnostic", func(t *testing.T) {
		content := `{"level":"error","code":"E500","message":"Internal server error","title":"Server Error"}`
		
		diag, err := ParseDiagnosticContent(content)
		if err != nil {
			t.Fatalf("Failed to parse JSON diagnostic: %v", err)
		}
		
		if diag.Level != DiagnosticError {
			t.Errorf("Expected error level, got %s", diag.Level)
		}
		
		if diag.Title != "Server Error" {
			t.Errorf("Expected title 'Server Error', got %s", diag.Title)
		}
	})

	t.Run("fallback to error", func(t *testing.T) {
		content := `This is just a plain error message`
		
		diag, err := ParseDiagnosticContent(content)
		if err != nil {
			t.Fatalf("Failed to parse fallback: %v", err)
		}
		
		if diag.Level != DiagnosticError {
			t.Errorf("Expected error level for fallback, got %s", diag.Level)
		}
		
		if diag.Message != content {
			t.Errorf("Expected message to be %q, got %q", content, diag.Message)
		}
	})
}