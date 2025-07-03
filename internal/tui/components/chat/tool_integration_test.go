package chat

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

func TestToolRenderingIntegration(t *testing.T) {
	th := theme.GetTheme("default")

	t.Run("message with tool invocation", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add message with tool invocation
		testMessages := []Message{
			{
				ID:   "msg-1",
				Role: "assistant",
				Content: `I'll calculate that for you.

<tool name="calculator">
  <input>42 * 10</input>
</tool>

Let me process that calculation.`,
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Check that parts were parsed correctly
		if len(mc.messageParts) != 1 {
			t.Fatalf("Expected 1 message parts array, got %d", len(mc.messageParts))
		}
		
		parts := mc.messageParts[0]
		
		// Should have text, tool, text
		if len(parts) < 3 {
			t.Errorf("Expected at least 3 parts, got %d", len(parts))
		}
		
		// Find tool part
		var toolPart *MessagePart
		for i, part := range parts {
			if part.Type == PartTypeTool {
				toolPart = &parts[i]
				break
			}
		}
		
		if toolPart == nil {
			t.Fatal("No tool part found")
		}
		
		// Get rendered view
		view := msgs.ViewWithSize(100, 50)
		strippedView := stripANSI(view)
		
		// Should contain tool name
		if !strings.Contains(strippedView, "calculator") {
			t.Error("Expected tool name 'calculator' in view")
		}
		
		// Should show collapsed by default
		if !strings.Contains(strippedView, "▶") {
			t.Error("Expected collapsed indicator")
		}
	})

	t.Run("tool expansion in selection mode", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add message with tool
		testMessages := []Message{
			{
				ID:   "msg-2",
				Role: "assistant",
				Content: `<tool name="file_reader">
  <input>/path/to/file.txt</input>
</tool>`,
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Enter selection mode
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		
		// Navigate to tool part if needed
		part, ok := mc.GetSelectedPart()
		for i := 0; i < 10 && (!ok || part.Type != PartTypeTool); i++ {
			mc.selectNextPart()
			part, ok = mc.GetSelectedPart()
		}
		
		if !ok || part.Type != PartTypeTool {
			t.Fatal("Could not navigate to tool part")
		}
		
		// Get initial view
		view1 := msgs.ViewWithSize(100, 50)
		
		// Debug: print selected part info
		if testing.Verbose() {
			fmt.Printf("Selected part: type=%s, index=%d\n", part.Type, part.PartIndex)
			fmt.Printf("Initial view contains ▶: %v\n", strings.Contains(stripANSI(view1), "▶"))
		}
		
		// Toggle expansion with 'e'
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		
		// Get expanded view (ViewWithSize should trigger render if needed)
		view2 := msgs.ViewWithSize(100, 50)
		
		// Views should be different
		if view1 == view2 {
			t.Error("Expected view to change after expansion")
		}
		
		strippedView := stripANSI(view2)
		
		// Should show expanded indicator
		if !strings.Contains(strippedView, "▼") {
			t.Error("Expected expanded indicator")
		}
		
		// Should show parameter details
		if !strings.Contains(strippedView, "/path/to/file.txt") {
			t.Error("Expected parameter value in expanded view")
		}
	})

	t.Run("tool result rendering", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add message with tool result
		testMessages := []Message{
			{
				ID:   "msg-3",
				Role: "assistant",
				Content: `Here's the result:

<tool_result>
File contents:
Hello, World!
This is a test file.
</tool_result>

The file has been read successfully.`,
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Check parsing
		parts := mc.messageParts[0]
		
		// Find tool result part
		var resultPart *MessagePart
		for i, part := range parts {
			if part.Type == PartTypeToolResult {
				resultPart = &parts[i]
				break
			}
		}
		
		if resultPart == nil {
			t.Fatal("No tool result part found")
		}
		
		// Get view
		view := msgs.ViewWithSize(100, 50)
		strippedView := stripANSI(view)
		
		// Should show success status
		if !strings.Contains(strippedView, "Success") {
			t.Error("Expected 'Success' status for tool result")
		}
		
		// Should show preview in collapsed state
		if !strings.Contains(strippedView, "→") {
			t.Error("Expected arrow indicator for result preview")
		}
	})

	t.Run("expand and collapse all tools", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add multiple messages with tools
		testMessages := []Message{
			{
				ID:   "msg-4",
				Role: "assistant",
				Content: `First tool:
<tool name="tool1"><input>input1</input></tool>

Second tool:
<tool name="tool2"><input>input2</input></tool>`,
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Enter selection mode
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		
		// Get initial view (all collapsed)
		view1 := msgs.ViewWithSize(100, 50)
		
		// Expand all with 'x'
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		
		view2 := msgs.ViewWithSize(100, 50)
		
		// View should change
		if view1 == view2 {
			t.Error("Expected view to change after expanding all")
		}
		
		// Collapse all with 'c'
		mc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
		
		view3 := msgs.ViewWithSize(100, 50)
		
		// Should return to collapsed state
		if view3 == view2 {
			t.Error("Expected view to change after collapsing all")
		}
	})

	t.Run("tool with error result", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Add message with error result
		testMessages := []Message{
			{
				ID:   "msg-5",
				Role: "assistant",
				Content: `Tool execution failed:

<tool_result>{"id":"r1","tool_id":"t1","error":"Permission denied: cannot read file"}</tool_result>

I encountered an error while trying to read the file.`,
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Get view
		view := msgs.ViewWithSize(100, 50)
		strippedView := stripANSI(view)
		
		// Debug: print what we got
		if testing.Verbose() {
			fmt.Printf("Error result view:\n%s\n", strippedView[:min(500, len(strippedView))])
		}
		
		// Should show error status
		if !strings.Contains(strippedView, "Error") {
			t.Error("Expected 'Error' status for failed tool result")
		}
	})
}

func TestComplexToolMessage(t *testing.T) {
	th := theme.GetTheme("default")
	msgs := NewMessagesComponent(th)
	mc := msgs.(*messagesComponent)
	
	// Initialize
	mc.Update(tea.WindowSizeMsg{Width: 120, Height: 60})
	
	// Complex message with multiple tools and results
	testMessages := []Message{
		{
			ID:   "complex-1",
			Role: "assistant",
			Content: `I'll help you analyze these files. Let me start by listing the directory contents:

<tool name="ls">
  <input>{"path": "/project", "options": ["-la"]}</input>
</tool>

<tool_result>
total 24
drwxr-xr-x  4 user group  128 Jan  1 12:00 .
drwxr-xr-x 10 user group  320 Jan  1 11:00 ..
-rw-r--r--  1 user group 1024 Jan  1 12:00 README.md
-rw-r--r--  1 user group 2048 Jan  1 12:00 main.go
</tool_result>

Now let me read the README file:

<tool name="cat">
  <input>{"path": "/project/README.md"}</input>
</tool>

<tool_result>
# My Project

This is a sample project demonstrating tool usage.

## Features
- Fast performance
- Easy to use
- Well documented
</tool_result>

Based on the directory listing and README content, this appears to be a Go project with good documentation.`,
		},
	}
	
	msgs.SetMessages(testMessages)
	
	// Check parsing
	parts := mc.messageParts[0]
	
	// Count part types
	toolCount := 0
	resultCount := 0
	textCount := 0
	
	for _, part := range parts {
		switch part.Type {
		case PartTypeTool:
			toolCount++
		case PartTypeToolResult:
			resultCount++
		case PartTypeText:
			textCount++
		}
	}
	
	if toolCount != 2 {
		t.Errorf("Expected 2 tool parts, got %d", toolCount)
	}
	if resultCount != 2 {
		t.Errorf("Expected 2 tool result parts, got %d", resultCount)
	}
	if textCount < 3 {
		t.Errorf("Expected at least 3 text parts, got %d", textCount)
	}
	
	// Get view
	view := msgs.ViewWithSize(120, 60)
	strippedView := stripANSI(view)
	
	// Should contain both tool names
	if !strings.Contains(strippedView, "ls") {
		t.Error("Expected 'ls' tool in view")
	}
	if !strings.Contains(strippedView, "cat") {
		t.Error("Expected 'cat' tool in view")
	}
	
	// Should have success indicators for results
	successCount := strings.Count(strippedView, "Success")
	if successCount < 2 {
		t.Errorf("Expected at least 2 'Success' indicators, found %d", successCount)
	}
}