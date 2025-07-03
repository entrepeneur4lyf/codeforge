package chat

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// RenderStatus represents the status of an async rendering operation
type RenderStatus string

const (
	RenderStatusPending   RenderStatus = "pending"
	RenderStatusRendering RenderStatus = "rendering"
	RenderStatusComplete  RenderStatus = "complete"
	RenderStatusError     RenderStatus = "error"
)

// RenderTask represents an async rendering task
type RenderTask struct {
	ID        string
	MessageID string
	Content   string
	Status    RenderStatus
	Progress  float64 // 0.0 to 1.0
	Error     error
	Result    string
	StartTime time.Time
	EndTime   time.Time
}

// RenderCompleteMsg is sent when a render task completes
type RenderCompleteMsg struct {
	TaskID    string
	MessageID string
	Result    string
	Error     error
}

// RenderProgressMsg is sent to update render progress
type RenderProgressMsg struct {
	TaskID   string
	Progress float64
}

// AsyncRenderer handles background rendering of messages
type AsyncRenderer struct {
	theme      theme.Theme
	tasks      map[string]*RenderTask
	tasksMutex sync.RWMutex
	
	// Renderers for different content types
	markdown *MarkdownRenderer
	code     *BlockRenderer
	
	// Configuration
	maxConcurrent int
	timeout       time.Duration
	
	// Worker pool
	workerCh chan *RenderTask
	ctx      context.Context
	cancel   context.CancelFunc
}

// NewAsyncRenderer creates a new async renderer
func NewAsyncRenderer(theme theme.Theme) *AsyncRenderer {
	ctx, cancel := context.WithCancel(context.Background())
	md, _ := NewMarkdownRenderer(theme, 80)
	
	ar := &AsyncRenderer{
		theme:         theme,
		tasks:         make(map[string]*RenderTask),
		markdown:      md,
		code:          CodeBlockRenderer(theme),
		maxConcurrent: 3,
		timeout:       30 * time.Second,
		workerCh:      make(chan *RenderTask, 10),
		ctx:           ctx,
		cancel:        cancel,
	}
	
	// Start worker pool
	for i := 0; i < ar.maxConcurrent; i++ {
		go ar.worker()
	}
	
	return ar
}

// QueueRender queues a message for async rendering
func (ar *AsyncRenderer) QueueRender(messageID, content string) (string, tea.Cmd) {
	taskID := generateTaskID(messageID)
	
	task := &RenderTask{
		ID:        taskID,
		MessageID: messageID,
		Content:   content,
		Status:    RenderStatusPending,
		Progress:  0.0,
		StartTime: time.Now(),
	}
	
	ar.tasksMutex.Lock()
	ar.tasks[taskID] = task
	ar.tasksMutex.Unlock()
	
	// Send task to worker pool
	select {
	case ar.workerCh <- task:
		// Task queued successfully
	default:
		// Queue is full, mark as error
		task.Status = RenderStatusError
		task.Error = ErrQueueFull
	}
	
	// Return placeholder and command to check status
	placeholder := ar.renderPlaceholder(task)
	cmd := ar.checkRenderStatus(taskID)
	
	return placeholder, cmd
}

// GetTask returns the current status of a render task
func (ar *AsyncRenderer) GetTask(taskID string) (*RenderTask, bool) {
	ar.tasksMutex.RLock()
	defer ar.tasksMutex.RUnlock()
	
	task, ok := ar.tasks[taskID]
	return task, ok
}

// GetTaskByMessageID returns the render task for a message
func (ar *AsyncRenderer) GetTaskByMessageID(messageID string) (*RenderTask, bool) {
	ar.tasksMutex.RLock()
	defer ar.tasksMutex.RUnlock()
	
	for _, task := range ar.tasks {
		if task.MessageID == messageID {
			return task, true
		}
	}
	return nil, false
}

// RenderStatus renders a status indicator for async operations
func (ar *AsyncRenderer) RenderStatus(task *RenderTask) string {
	if task == nil {
		return ""
	}
	
	var statusIcon string
	var statusColor lipgloss.AdaptiveColor
	
	switch task.Status {
	case RenderStatusPending:
		statusIcon = "⏳"
		statusColor = ar.theme.TextMuted()
	case RenderStatusRendering:
		statusIcon = ar.getSpinner(task.StartTime)
		statusColor = ar.theme.Info()
	case RenderStatusComplete:
		statusIcon = "✓"
		statusColor = ar.theme.Success()
	case RenderStatusError:
		statusIcon = "✗"
		statusColor = ar.theme.Error()
	}
	
	style := lipgloss.NewStyle().Foreground(statusColor)
	
	// Build status line
	var status strings.Builder
	status.WriteString(style.Render(statusIcon))
	status.WriteString(" ")
	
	switch task.Status {
	case RenderStatusPending:
		status.WriteString(style.Render("Queued for rendering..."))
	case RenderStatusRendering:
		progress := int(task.Progress * 100)
		status.WriteString(style.Render(fmt.Sprintf("Rendering... %d%%", progress)))
		
		// Add progress bar
		if task.Progress > 0 {
			bar := ar.renderProgressBar(task.Progress, 20)
			status.WriteString(" ")
			status.WriteString(bar)
		}
	case RenderStatusComplete:
		duration := task.EndTime.Sub(task.StartTime)
		status.WriteString(style.Render(fmt.Sprintf("Rendered in %s", duration.Round(time.Millisecond))))
	case RenderStatusError:
		status.WriteString(style.Render("Render failed"))
		if task.Error != nil {
			status.WriteString(": ")
			status.WriteString(task.Error.Error())
		}
	}
	
	return status.String()
}

// worker processes render tasks from the queue
func (ar *AsyncRenderer) worker() {
	for {
		select {
		case <-ar.ctx.Done():
			return
		case task := <-ar.workerCh:
			ar.processTask(task)
		}
	}
}

// processTask renders a single task
func (ar *AsyncRenderer) processTask(task *RenderTask) {
	// Update status
	task.Status = RenderStatusRendering
	task.Progress = 0.1
	
	// Create timeout context
	ctx, cancel := context.WithTimeout(ar.ctx, ar.timeout)
	defer cancel()
	
	// Simulate progressive rendering for large content
	contentLen := len(task.Content)
	if contentLen > 1000 {
		// For large content, update progress periodically
		go ar.updateProgress(ctx, task)
	}
	
	// Perform the actual rendering
	result, err := ar.renderContent(ctx, task.Content)
	
	// Update task with result
	task.EndTime = time.Now()
	if err != nil {
		task.Status = RenderStatusError
		task.Error = err
	} else {
		task.Status = RenderStatusComplete
		task.Result = result
		task.Progress = 1.0
	}
}

// renderContent performs the actual content rendering
func (ar *AsyncRenderer) renderContent(ctx context.Context, content string) (string, error) {
	// Check for cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}
	
	// Detect content type and render accordingly
	if looksLikeMarkdown(content) && ar.markdown != nil {
		rendered, err := ar.markdown.Render(content)
		if err != nil {
			return content, err
		}
		return rendered, nil
	}
	
	// Default rendering
	return content, nil
}

// updateProgress updates the progress of a long-running render task
func (ar *AsyncRenderer) updateProgress(ctx context.Context, task *RenderTask) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	startTime := time.Now()
	estimatedDuration := time.Duration(len(task.Content)/1000) * time.Millisecond
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if task.Status != RenderStatusRendering {
				return
			}
			
			elapsed := time.Since(startTime)
			progress := float64(elapsed) / float64(estimatedDuration)
			if progress > 0.9 {
				progress = 0.9 // Cap at 90% until actually complete
			}
			task.Progress = progress
		}
	}
}

// renderPlaceholder returns a placeholder while content is rendering
func (ar *AsyncRenderer) renderPlaceholder(task *RenderTask) string {
	style := lipgloss.NewStyle().
		Foreground(ar.theme.TextMuted()).
		Border(lipgloss.NormalBorder()).
		BorderForeground(ar.theme.TextMuted()).
		Padding(1)
	
	content := fmt.Sprintf("⏳ Rendering message...\nID: %s", task.MessageID)
	return style.Render(content)
}

// renderProgressBar renders a progress bar
func (ar *AsyncRenderer) renderProgressBar(progress float64, width int) string {
	filled := int(float64(width) * progress)
	empty := width - filled
	
	filledStyle := lipgloss.NewStyle().Foreground(ar.theme.Primary())
	emptyStyle := lipgloss.NewStyle().Foreground(ar.theme.TextMuted())
	
	bar := filledStyle.Render(strings.Repeat("█", filled))
	bar += emptyStyle.Render(strings.Repeat("░", empty))
	
	return fmt.Sprintf("[%s]", bar)
}

// getSpinner returns a spinner character based on time
func (ar *AsyncRenderer) getSpinner(startTime time.Time) string {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	elapsed := time.Since(startTime)
	index := int(elapsed.Milliseconds()/100) % len(spinners)
	return spinners[index]
}

// checkRenderStatus returns a command to check render status
func (ar *AsyncRenderer) checkRenderStatus(taskID string) tea.Cmd {
	return func() tea.Msg {
		// Check periodically until complete
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				ar.tasksMutex.RLock()
				task, ok := ar.tasks[taskID]
				ar.tasksMutex.RUnlock()
				
				if !ok {
					return nil
				}
				
				switch task.Status {
				case RenderStatusComplete:
					return RenderCompleteMsg{
						TaskID:    task.ID,
						MessageID: task.MessageID,
						Result:    task.Result,
					}
				case RenderStatusError:
					return RenderCompleteMsg{
						TaskID:    task.ID,
						MessageID: task.MessageID,
						Error:     task.Error,
					}
				case RenderStatusRendering:
					// Send progress update
					return RenderProgressMsg{
						TaskID:   task.ID,
						Progress: task.Progress,
					}
				}
			}
		}
	}
}

// Cleanup removes completed tasks older than the specified duration
func (ar *AsyncRenderer) Cleanup(olderThan time.Duration) {
	ar.tasksMutex.Lock()
	defer ar.tasksMutex.Unlock()
	
	now := time.Now()
	for id, task := range ar.tasks {
		if task.Status == RenderStatusComplete || task.Status == RenderStatusError {
			if now.Sub(task.EndTime) > olderThan {
				delete(ar.tasks, id)
			}
		}
	}
}

// Close shuts down the async renderer
func (ar *AsyncRenderer) Close() {
	ar.cancel()
}

// Helper functions

func generateTaskID(messageID string) string {
	return fmt.Sprintf("render-%s-%d", messageID, time.Now().UnixNano())
}

func looksLikeMarkdown(content string) bool {
	// Simple heuristic to detect markdown content
	markdownIndicators := []string{"#", "*", "-", "```", "[", "]", "**", "__"}
	for _, indicator := range markdownIndicators {
		if strings.Contains(content, indicator) {
			return true
		}
	}
	return false
}

// Errors
var (
	ErrQueueFull = fmt.Errorf("render queue is full")
)