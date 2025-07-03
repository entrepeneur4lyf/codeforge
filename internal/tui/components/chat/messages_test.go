package chat

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

func TestMessagesComponent(t *testing.T) {
	th := theme.GetTheme("default")

	t.Run("creation and initialization", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		
		// Test initial state
		mc := msgs.(*messagesComponent)
		if mc.theme == nil {
			t.Error("Expected theme to be set")
		}
		if mc.cache == nil {
			t.Error("Expected cache to be initialized")
		}
		if mc.renderer == nil {
			t.Error("Expected renderer to be initialized")
		}
		if mc.markdown == nil {
			t.Error("Expected markdown renderer to be initialized")
		}
		if mc.parser == nil {
			t.Error("Expected parser to be initialized")
		}
		if mc.selectionMode {
			t.Error("Expected selection mode to be false initially")
		}
		if mc.selectedMessageIndex != -1 || mc.selectedPartIndex != -1 {
			t.Error("Expected no selection initially")
		}
		if !mc.tail {
			t.Error("Expected tail mode to be true initially")
		}
	})

	t.Run("basic message rendering", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize with window size
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add test messages
		testMessages := []Message{
			{ID: "1", Role: "user", Content: "Hello, assistant!"},
			{ID: "2", Role: "assistant", Content: "Hello! How can I help you today?"},
		}
		
		msgs.SetMessages(testMessages)
		
		// Get rendered view
		view := msgs.ViewWithSize(100, 50)
		
		// Strip ANSI codes for comparison
		strippedView := stripANSI(view)
		
		// Should contain both messages
		if !strings.Contains(strippedView, "Hello, assistant!") {
			t.Error("User message not found in view")
		}
		if !strings.Contains(strippedView, "How can I help you today?") {
			t.Error("Assistant message not found in view")
		}
	})

	t.Run("message parsing", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Add message with code block
		testMessages := []Message{
			{
				ID:   "1",
				Role: "assistant",
				Content: `Here's a function:

` + "```go" + `
func hello() {
    fmt.Println("Hello")
}
` + "```" + `

That's the code.`,
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Check that parts were parsed
		if len(mc.messageParts) != 1 {
			t.Fatalf("Expected 1 message parts array, got %d", len(mc.messageParts))
		}
		
		parts := mc.messageParts[0]
		if len(parts) < 3 {
			t.Errorf("Expected at least 3 parts (text, code, text), got %d", len(parts))
		}
		
		// Find code part
		hasCodePart := false
		for _, part := range parts {
			if part.Type == PartTypeCode {
				hasCodePart = true
				if !strings.Contains(part.Content, "func hello()") {
					t.Error("Code part missing function content")
				}
			}
		}
		
		if !hasCodePart {
			t.Error("No code part found in parsed message")
		}
	})

	t.Run("selection mode", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add messages
		testMessages := []Message{
			{ID: "1", Role: "user", Content: "Test message 1"},
			{ID: "2", Role: "assistant", Content: "Test response 1"},
		}
		msgs.SetMessages(testMessages)
		
		// Initially not in selection mode
		if mc.selectionMode {
			t.Error("Should not be in selection mode initially")
		}
		
		// Toggle selection mode with 's' key
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		
		if !mc.selectionMode {
			t.Error("Should be in selection mode after pressing 's'")
		}
		
		// Should select first message
		if mc.selectedMessageIndex != 0 || mc.selectedPartIndex != 0 {
			t.Errorf("Expected first message/part selected, got msg:%d part:%d",
				mc.selectedMessageIndex, mc.selectedPartIndex)
		}
		
		// Exit selection mode with escape
		mc.Update(tea.KeyMsg{Type: tea.KeyEsc})
		
		if mc.selectionMode {
			t.Error("Should exit selection mode after escape")
		}
	})

	t.Run("navigation in selection mode", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize and add messages with multiple parts
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		testMessages := []Message{
			{
				ID:   "1",
				Role: "assistant",
				Content: `Part 1

` + "```code" + `
Code block
` + "```" + `

Part 2`,
			},
		}
		msgs.SetMessages(testMessages)
		
		// Enter selection mode
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		
		// Navigate to next part
		initialPart := mc.selectedPartIndex
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
		
		if mc.selectedPartIndex <= initialPart && len(mc.messageParts[0]) > 1 {
			t.Error("Should move to next part with 'n'")
		}
		
		// Navigate to previous part
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
		
		if mc.selectedPartIndex != initialPart {
			t.Error("Should move to previous part with 'p'")
		}
	})

	t.Run("get selected part", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// No selection initially
		part, ok := msgs.GetSelectedPart()
		if ok {
			t.Error("Should not have selected part when not in selection mode")
		}
		
		// Add message and enter selection mode
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		testMessages := []Message{
			{ID: "1", Role: "user", Content: "Selected content"},
		}
		msgs.SetMessages(testMessages)
		
		// Enter selection mode
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		
		// Get selected part
		part, ok = msgs.GetSelectedPart()
		if !ok {
			t.Error("Should have selected part in selection mode")
		}
		
		if part.MessageID != "1" {
			t.Errorf("Expected message ID '1', got '%s'", part.MessageID)
		}
		
		if !strings.Contains(part.Content, "Selected content") {
			t.Error("Selected part missing expected content")
		}
	})

	t.Run("navigation methods", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Test page navigation
		msgs.PageUp()
		if mc.tail {
			t.Error("Should disable tail mode on PageUp")
		}
		
		msgs.PageDown()
		// Note: tail mode only re-enabled if at bottom
		
		msgs.Last()
		if !mc.tail {
			t.Error("Should enable tail mode on Last")
		}
		
		msgs.First()
		if mc.tail {
			t.Error("Should disable tail mode on First")
		}
		
		// Test half-page navigation
		msgs.HalfPageUp()
		if mc.tail {
			t.Error("Should disable tail mode on HalfPageUp")
		}
		
		msgs.HalfPageDown()
		// Note: tail mode only re-enabled if at bottom
	})

	t.Run("width adjustment", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Set initial width
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add messages
		testMessages := []Message{
			{ID: "1", Role: "user", Content: "Test message"},
		}
		msgs.SetMessages(testMessages)
		
		// Change width
		msgs.SetWidth(120)
		
		if mc.width != 120 {
			t.Errorf("Expected width 120, got %d", mc.width)
		}
		
		if mc.viewport.Width != 120 {
			t.Errorf("Expected viewport width 120, got %d", mc.viewport.Width)
		}
		
		// Markdown renderer should also be updated
		if mc.markdown != nil && mc.markdown.width != 116 { // 120 - 4 for padding
			t.Errorf("Expected markdown width 116, got %d", mc.markdown.width)
		}
	})

	t.Run("selected content", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize and add messages
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		testMessages := []Message{
			{ID: "1", Role: "user", Content: "This is the selected content"},
		}
		msgs.SetMessages(testMessages)
		
		// Not in selection mode
		selected := msgs.Selected()
		if selected != "" {
			t.Error("Should return empty string when not in selection mode")
		}
		
		// Enter selection mode
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		
		// Get selected content
		selected = msgs.Selected()
		if !strings.Contains(selected, "This is the selected content") {
			t.Errorf("Expected selected content, got: %s", selected)
		}
	})

	t.Run("part highlighting in selection mode", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add message with multiple parts
		testMessages := []Message{
			{
				ID:   "1",
				Role: "assistant",
				Content: `Text part

` + "```go" + `
func code() {}
` + "```" + `

More text`,
			},
		}
		msgs.SetMessages(testMessages)
		
		// Enter selection mode
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		
		// Render should include part highlighting
		view := msgs.ViewWithSize(100, 50)
		
		// Should have content (basic check - exact rendering depends on styles)
		if view == "" {
			t.Error("Expected non-empty view in selection mode")
		}
	})

	t.Run("message with error", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add message with error
		testMessages := []Message{
			{
				ID:      "1",
				Role:    "assistant",
				Content: "Failed to process",
				Error:   &MockError{msg: "Network timeout"},
			},
		}
		msgs.SetMessages(testMessages)
		
		// Should parse error as error part
		if len(mc.messageParts) > 0 {
			// Error messages might be rendered differently
			// Just check that message was processed
			if len(mc.messageParts[0]) == 0 {
				t.Error("Expected at least one part for error message")
			}
		}
	})
}

// MockError for testing error messages
type MockError struct {
	msg string
}

func (e *MockError) Error() string {
	return e.msg
}

func TestMessagePartNavigation(t *testing.T) {
	th := theme.GetTheme("default")
	msgs := NewMessagesComponent(th)
	mc := msgs.(*messagesComponent)
	
	// Initialize
	mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	
	// Add multiple messages with multiple parts each
	testMessages := []Message{
		{
			ID:   "msg1",
			Role: "user",
			Content: `Question part 1

` + "```code" + `
code in question
` + "```",
		},
		{
			ID:   "msg2", 
			Role: "assistant",
			Content: `Answer part 1

` + "```python" + `
def solution():
    pass
` + "```" + `

Answer part 2`,
		},
	}
	msgs.SetMessages(testMessages)
	
	// Enter selection mode
	mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	
	// Should start at first message, first part
	if mc.selectedMessageIndex != 0 || mc.selectedPartIndex != 0 {
		t.Errorf("Should start at (0,0), got (%d,%d)", 
			mc.selectedMessageIndex, mc.selectedPartIndex)
	}
	
	// Navigate through all parts
	positions := []struct {
		msgIdx  int
		partIdx int
	}{}
	
	// Record initial position
	positions = append(positions, struct {
		msgIdx  int
		partIdx int
	}{mc.selectedMessageIndex, mc.selectedPartIndex})
	
	// Navigate forward through all parts
	for i := 0; i < 10; i++ { // Arbitrary limit to prevent infinite loop
		prevMsg := mc.selectedMessageIndex
		prevPart := mc.selectedPartIndex
		
		mc.selectNextPart()
		
		// Check if we moved
		if mc.selectedMessageIndex == prevMsg && mc.selectedPartIndex == prevPart {
			// Reached the end
			break
		}
		
		positions = append(positions, struct {
			msgIdx  int
			partIdx int
		}{mc.selectedMessageIndex, mc.selectedPartIndex})
	}
	
	// Should have visited multiple positions
	if len(positions) < 3 {
		t.Errorf("Expected to navigate through multiple parts, only got %d positions", len(positions))
	}
	
	// Navigate backward
	for i := len(positions) - 2; i >= 0; i-- {
		mc.selectPreviousPart()
		
		if mc.selectedMessageIndex != positions[i].msgIdx || 
		   mc.selectedPartIndex != positions[i].partIdx {
			t.Errorf("Backward navigation mismatch at position %d: expected (%d,%d), got (%d,%d)",
				i, positions[i].msgIdx, positions[i].partIdx,
				mc.selectedMessageIndex, mc.selectedPartIndex)
		}
	}
}

func TestRenderMessageWithParts(t *testing.T) {
	th := theme.GetTheme("default")
	msgs := NewMessagesComponent(th)
	mc := msgs.(*messagesComponent)
	
	// Create a message with parts
	msg := Message{
		ID:      "test",
		Role:    "assistant",
		Content: "Full content here",
	}
	
	parts := []MessagePart{
		{
			MessageID: "test",
			PartIndex: 0,
			Type:      PartTypeText,
			Content:   "Text part",
		},
		{
			MessageID: "test",
			PartIndex: 1,
			Type:      PartTypeCode,
			Content:   "code part",
			Metadata:  map[string]interface{}{"language": "go"},
		},
		{
			MessageID: "test",
			PartIndex: 2,
			Type:      PartTypeError,
			Content:   "error part",
		},
	}
	
	// Not in selection mode
	rendered := mc.renderMessageWithParts(msg, parts, 0)
	
	// Should contain all parts
	if !strings.Contains(rendered, "Text part") {
		t.Error("Missing text part in rendered output")
	}
	if !strings.Contains(rendered, "code part") {
		t.Error("Missing code part in rendered output")
	}
	if !strings.Contains(rendered, "error part") {
		t.Error("Missing error part in rendered output")
	}
	
	// Should have type indicators for non-text parts
	if !strings.Contains(rendered, "[code]") {
		t.Error("Missing code type indicator")
	}
	if !strings.Contains(rendered, "[error]") {
		t.Error("Missing error type indicator")
	}
	
	// Now test with selection
	mc.selectionMode = true
	mc.selectedMessageIndex = 0
	mc.selectedPartIndex = 1 // Select the code part
	
	rendered = mc.renderMessageWithParts(msg, parts, 0)
	
	// Should still contain all parts
	if !strings.Contains(rendered, "Text part") {
		t.Error("Missing text part when selection active")
	}
	if !strings.Contains(rendered, "code part") {
		t.Error("Missing selected code part")
	}
}

func TestViewportBehavior(t *testing.T) {
	th := theme.GetTheme("default")
	msgs := NewMessagesComponent(th)
	mc := msgs.(*messagesComponent)
	
	// Initialize
	mc.Update(tea.WindowSizeMsg{Width: 100, Height: 10}) // Small height
	
	// Add many messages to exceed viewport
	var testMessages []Message
	for i := 0; i < 20; i++ {
		testMessages = append(testMessages, Message{
			ID:      string(rune('a' + i)),
			Role:    "user",
			Content: strings.Repeat("Line of text\n", 3),
		})
	}
	msgs.SetMessages(testMessages)
	
	// Should be at bottom (tail mode)
	if !mc.tail {
		t.Error("Should be in tail mode with new messages")
	}
	
	// Scroll up
	msgs.PageUp()
	if mc.tail {
		t.Error("Should exit tail mode when scrolling up")
	}
	
	// Go to bottom
	msgs.Last()
	if !mc.tail {
		t.Error("Should re-enable tail mode when going to last")
	}
}