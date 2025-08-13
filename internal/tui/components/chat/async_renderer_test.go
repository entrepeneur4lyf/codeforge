package chat

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

func TestAsyncRenderer(t *testing.T) {
	th := theme.GetTheme("default")
	
	t.Run("create and close", func(t *testing.T) {
		ar := NewAsyncRenderer(th)
		defer ar.Close()
		
		if ar.maxConcurrent != 3 {
			t.Errorf("Expected 3 concurrent workers, got %d", ar.maxConcurrent)
		}
	})
	
	t.Run("queue render task", func(t *testing.T) {
		ar := NewAsyncRenderer(th)
		defer ar.Close()
		
		placeholder, cmd := ar.QueueRender("msg-1", "Hello, world!")
		
		if placeholder == "" {
			t.Error("Expected placeholder content")
		}
		
		if !strings.Contains(placeholder, "Rendering message") {
			t.Error("Placeholder should indicate rendering")
		}
		
		if cmd == nil {
			t.Error("Expected command to check status")
		}
		
		// Check task was created
		task, ok := ar.GetTaskByMessageID("msg-1")
		if !ok {
			t.Error("Task should be created")
		}
		
        // Rendering can complete very fast on small inputs; accept complete as valid
        if task.Status != RenderStatusPending && task.Status != RenderStatusRendering && task.Status != RenderStatusComplete {
            t.Errorf("Expected pending, rendering, or complete status, got %s", task.Status)
        }
	})
	
	t.Run("render completion", func(t *testing.T) {
		ar := NewAsyncRenderer(th)
		defer ar.Close()
		
		content := "# Hello\n\nThis is a test."
		_, _ = ar.QueueRender("msg-2", content)
		
		// Wait for rendering to complete
		var task *RenderTask
		for i := 0; i < 50; i++ { // 5 seconds max
			time.Sleep(100 * time.Millisecond)
			if t, ok := ar.GetTaskByMessageID("msg-2"); ok {
				task = t
				if task.Status == RenderStatusComplete || task.Status == RenderStatusError {
					break
				}
			}
		}
		
		if task == nil {
			t.Fatal("Task not found")
		}
		
		if task.Status != RenderStatusComplete {
			t.Errorf("Expected complete status, got %s", task.Status)
			if task.Error != nil {
				t.Errorf("Error: %v", task.Error)
			}
		}
		
		if task.Result == "" {
			t.Error("Expected rendered result")
		}
	})
	
	t.Run("render status display", func(t *testing.T) {
		ar := NewAsyncRenderer(th)
		defer ar.Close()
		
		testCases := []struct {
			name   string
			task   *RenderTask
			expect string
		}{
			{
				name: "pending",
				task: &RenderTask{
					Status: RenderStatusPending,
				},
				expect: "Queued for rendering",
			},
			{
				name: "rendering",
				task: &RenderTask{
					Status:   RenderStatusRendering,
					Progress: 0.5,
				},
				expect: "Rendering... 50%",
			},
			{
				name: "complete",
				task: &RenderTask{
					Status:    RenderStatusComplete,
					StartTime: time.Now(),
					EndTime:   time.Now().Add(123 * time.Millisecond),
				},
				expect: "Rendered in",
			},
			{
				name: "error",
				task: &RenderTask{
					Status: RenderStatusError,
					Error:  ErrQueueFull,
				},
				expect: "Render failed",
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				status := ar.RenderStatus(tc.task)
				if !strings.Contains(status, tc.expect) {
					t.Errorf("Expected status to contain %q, got %q", tc.expect, status)
				}
			})
		}
	})
	
	t.Run("progress bar", func(t *testing.T) {
		ar := NewAsyncRenderer(th)
		defer ar.Close()
		
		bar := ar.renderProgressBar(0.5, 10)
		
		// Should have 5 filled and 5 empty
		if !strings.Contains(bar, "█████") {
			t.Error("Expected 5 filled blocks")
		}
		
		if !strings.Contains(bar, "░░░░░") {
			t.Error("Expected 5 empty blocks")
		}
	})
	
	t.Run("cleanup old tasks", func(t *testing.T) {
		ar := NewAsyncRenderer(th)
		defer ar.Close()
		
		// Create a completed task
		task := &RenderTask{
			ID:        "test-1",
			MessageID: "msg-3",
			Status:    RenderStatusComplete,
			EndTime:   time.Now().Add(-2 * time.Hour), // 2 hours ago
		}
		
		ar.tasksMutex.Lock()
		ar.tasks[task.ID] = task
		ar.tasksMutex.Unlock()
		
		// Cleanup tasks older than 1 hour
		ar.Cleanup(1 * time.Hour)
		
		// Task should be removed
		_, ok := ar.GetTask("test-1")
		if ok {
			t.Error("Old task should be cleaned up")
		}
	})
	
	t.Run("concurrent rendering", func(t *testing.T) {
		ar := NewAsyncRenderer(th)
		defer ar.Close()
		
		// Queue multiple tasks
		for i := 0; i < 5; i++ {
			msgID := fmt.Sprintf("msg-%d", i)
			content := fmt.Sprintf("Content %d", i)
			ar.QueueRender(msgID, content)
		}
		
		// Wait for all to complete
		completed := 0
		timeout := time.After(5 * time.Second)
		
		for completed < 5 {
			select {
			case <-timeout:
				t.Fatalf("Timeout waiting for tasks to complete. Completed: %d/5", completed)
			default:
				completed = 0
				ar.tasksMutex.RLock()
				for _, task := range ar.tasks {
					if task.Status == RenderStatusComplete {
						completed++
					}
				}
				ar.tasksMutex.RUnlock()
				time.Sleep(100 * time.Millisecond)
			}
		}
	})
	
	t.Run("markdown detection", func(t *testing.T) {
		testCases := []struct {
			content  string
			expected bool
		}{
			{"# Hello", true},
			{"**bold**", true},
			{"```code```", true},
			{"[link](url)", true},
			{"plain text", false},
			{"just some words", false},
		}
		
		for _, tc := range testCases {
			result := looksLikeMarkdown(tc.content)
			if result != tc.expected {
				t.Errorf("looksLikeMarkdown(%q) = %v, want %v", tc.content, result, tc.expected)
			}
		}
	})
	
	t.Run("spinner animation", func(t *testing.T) {
		ar := NewAsyncRenderer(th)
		defer ar.Close()
		
		startTime := time.Now()
		
		// Get spinner at different times
		spinner1 := ar.getSpinner(startTime)
		time.Sleep(100 * time.Millisecond)
		spinner2 := ar.getSpinner(startTime)
		
		// Spinners should be different (unless we hit the same index)
		// This is a weak test but better than nothing
		if spinner1 == "" || spinner2 == "" {
			t.Error("Spinner should not be empty")
		}
	})
	
	t.Run("render with timeout", func(t *testing.T) {
		ar := NewAsyncRenderer(th)
		ar.timeout = 100 * time.Millisecond // Very short timeout
		defer ar.Close()
		
		// Create a context that will be cancelled
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		
		result, err := ar.renderContent(ctx, "test content")
		
		if err == nil {
			// Either it completed fast enough or timeout didn't trigger
			if result == "" {
				t.Error("Expected non-empty result")
			}
		} else {
			// Should be context error
			if err != context.DeadlineExceeded && err != context.Canceled {
				t.Errorf("Expected context error, got %v", err)
			}
		}
	})
}

func TestRenderMessages(t *testing.T) {
	t.Run("render complete message", func(t *testing.T) {
		msg := RenderCompleteMsg{
			TaskID:    "task-1",
			MessageID: "msg-1",
			Result:    "Rendered content",
			Error:     nil,
		}
		
		if msg.TaskID != "task-1" {
			t.Errorf("Expected task-1, got %s", msg.TaskID)
		}
		
		if msg.Error != nil {
			t.Error("Expected no error")
		}
	})
	
	t.Run("render progress message", func(t *testing.T) {
		msg := RenderProgressMsg{
			TaskID:   "task-2",
			Progress: 0.75,
		}
		
		if msg.Progress != 0.75 {
			t.Errorf("Expected 0.75 progress, got %f", msg.Progress)
		}
	})
}

func TestTaskID(t *testing.T) {
	id1 := generateTaskID("msg-1")
	id2 := generateTaskID("msg-1")
	
	if id1 == id2 {
		t.Error("Task IDs should be unique even for same message")
	}
	
	if !strings.HasPrefix(id1, "render-msg-1-") {
		t.Error("Task ID should have expected prefix")
	}
}