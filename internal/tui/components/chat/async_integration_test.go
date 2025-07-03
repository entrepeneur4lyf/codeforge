package chat

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

func TestAsyncRenderingIntegration(t *testing.T) {
	th := theme.GetTheme("default")
	
	t.Run("async rendering for large message", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Create a large message that should trigger async rendering
		largeContent := strings.Repeat("This is a very long message. ", 200) // ~6000 chars
		testMessages := []Message{
			{
				ID:      "large-msg-1",
				Role:    "assistant",
				Content: largeContent,
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Initial view should show placeholder
		view := msgs.ViewWithSize(100, 50)
		strippedView := stripANSI(view)
		
		if !strings.Contains(strippedView, "Rendering message") {
			t.Error("Expected rendering placeholder for large message")
		}
		
		// Wait for async rendering to complete
		var completed bool
		for i := 0; i < 50; i++ { // 5 seconds max
			time.Sleep(100 * time.Millisecond)
			
			// Check if task completed
			if task, ok := mc.asyncRenderer.GetTaskByMessageID("large-msg-1"); ok {
				if task.Status == RenderStatusComplete {
					// Simulate receiving the complete message
					mc.Update(RenderCompleteMsg{
						TaskID:    task.ID,
						MessageID: task.MessageID,
						Result:    task.Result,
					})
					completed = true
					break
				}
			}
		}
		
		if !completed {
			t.Fatal("Async rendering did not complete in time")
		}
		
		// View should now show the rendered content
		view = msgs.ViewWithSize(100, 50)
		strippedView = stripANSI(view)
		
		if strings.Contains(strippedView, "Rendering message") {
			t.Error("Should not show placeholder after completion")
		}
		
		if !strings.Contains(strippedView, "This is a very long message") {
			t.Error("Should show the actual content after completion")
		}
	})
	
	t.Run("async rendering status updates", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Create a message that triggers async
		testMessages := []Message{
			{
				ID:      "async-msg-2",
				Role:    "assistant",
				Content: strings.Repeat("Content ", 1000), // ~7000 chars
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Get initial view
		_ = msgs.ViewWithSize(100, 50)
		
		// Should have a task in progress
		task, ok := mc.asyncRenderer.GetTaskByMessageID("async-msg-2")
		if !ok {
			t.Fatal("Expected async task to be created")
		}
		
		// Should be able to see status
		status := mc.asyncRenderer.RenderStatus(task)
		if status == "" {
			t.Error("Expected non-empty status")
		}
		
		// Check that we can handle progress updates
		if task.ID != "" {
			_, cmd := mc.Update(RenderProgressMsg{
				TaskID:   task.ID,
				Progress: 0.5,
			})
			
			// Should return a command to continue checking
			if cmd == nil {
				t.Error("Expected command to continue checking status")
			}
		}
	})
	
	t.Run("multiple async messages", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Create multiple large messages
		var testMessages []Message
		for i := 0; i < 3; i++ {
			testMessages = append(testMessages, Message{
				ID:      fmt.Sprintf("multi-msg-%d", i),
				Role:    "assistant",
				Content: strings.Repeat(fmt.Sprintf("Message %d. ", i), 1000),
			})
		}
		
		msgs.SetMessages(testMessages)
		
		// Should have multiple async tasks
		taskCount := 0
		for _, msg := range testMessages {
			if _, ok := mc.asyncRenderer.GetTaskByMessageID(msg.ID); ok {
				taskCount++
			}
		}
		
		if taskCount != 3 {
			t.Errorf("Expected 3 async tasks, got %d", taskCount)
		}
		
		// Clean up
		mc.asyncRenderer.Close()
	})
	
	t.Run("small message uses sync rendering", func(t *testing.T) {
		msgs := NewMessagesComponent(th)
		mc := msgs.(*messagesComponent)
		
		// Initialize
		mc.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
		
		// Create a small message that should NOT trigger async
		testMessages := []Message{
			{
				ID:      "small-msg-1",
				Role:    "assistant",
				Content: "This is a small message.",
			},
		}
		
		msgs.SetMessages(testMessages)
		
		// Should not have an async task
		_, ok := mc.asyncRenderer.GetTaskByMessageID("small-msg-1")
		if ok {
			t.Error("Small message should not trigger async rendering")
		}
		
		// View should show content immediately
		view := msgs.ViewWithSize(100, 50)
		strippedView := stripANSI(view)
		
		if strings.Contains(strippedView, "Rendering message") {
			t.Error("Should not show placeholder for small message")
		}
		
		if !strings.Contains(strippedView, "This is a small message") {
			t.Error("Should show content immediately for small message")
		}
	})
}