package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/transform"
)

// LMStudioHandler implements the ApiHandler interface for LM Studio's local API
// LM Studio is OpenAI-compatible for local model serving
type LMStudioHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// LMStudioRequest represents a request to LM Studio's API (OpenAI-compatible)
type LMStudioRequest struct {
	Model         string                    `json:"model"`
	Messages      []transform.OpenAIMessage `json:"messages"`
	MaxTokens     *int                      `json:"max_tokens,omitempty"`
	Temperature   *float64                  `json:"temperature,omitempty"`
	Stream        bool                      `json:"stream"`
	StreamOptions *LMStudioStreamOptions    `json:"stream_options,omitempty"`
	User          string                    `json:"user,omitempty"`
}

// LMStudioStreamOptions configures streaming behavior
type LMStudioStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// LMStudioStreamEvent represents a streaming event from LM Studio
type LMStudioStreamEvent struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []LMStudioChoice `json:"choices"`
	Usage   *LMStudioUsage   `json:"usage,omitempty"`
}

// LMStudioChoice represents a choice in the response
type LMStudioChoice struct {
	Index        int              `json:"index"`
	Delta        *LMStudioDelta   `json:"delta,omitempty"`
	Message      *LMStudioMessage `json:"message,omitempty"`
	FinishReason *string          `json:"finish_reason,omitempty"`
}

// LMStudioDelta represents incremental content in streaming
type LMStudioDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// LMStudioMessage represents a complete message
type LMStudioMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LMStudioUsage represents token usage information
type LMStudioUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewLMStudioHandler creates a new LM Studio handler
func NewLMStudioHandler(options llm.ApiHandlerOptions) *LMStudioHandler {
	baseURL := options.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:1234/v1" // Default LM Studio local server
	}

	// Configure timeout based on request timeout option
	timeout := 120 * time.Second // Longer timeout for local models
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &LMStudioHandler{
		options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *LMStudioHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := LMStudioRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &LMStudioStreamOptions{
			IncludeUsage: true,
		},
	}

	// Set max tokens if specified
	if model.Info.MaxTokens > 0 {
		request.MaxTokens = &model.Info.MaxTokens
	}

	// Set temperature if specified
	if model.Info.Temperature != nil {
		request.Temperature = model.Info.Temperature
	}

	// Add user identifier if available
	if h.options.TaskID != "" {
		request.User = h.options.TaskID
	}

	return h.streamRequest(ctx, request)
}

// GetModel implements the ApiHandler interface
func (h *LMStudioHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *LMStudioHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// LM Studio provides usage in the final stream event
	return nil, nil
}

// getDefaultModelInfo returns default model information for LM Studio models
func (h *LMStudioHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for LM Studio local models
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       32768,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.0, // Local models are free
		OutputPrice:         0.0, // Local models are free
		Description:         fmt.Sprintf("LM Studio local model: %s", modelID),
	}

	// Model-specific configurations based on common local models
	modelLower := strings.ToLower(modelID)

	// Llama models
	if strings.Contains(modelLower, "llama") {
		if strings.Contains(modelLower, "70b") || strings.Contains(modelLower, "405b") {
			info.ContextWindow = 128000
			info.MaxTokens = 8192
		} else if strings.Contains(modelLower, "8b") || strings.Contains(modelLower, "7b") {
			info.ContextWindow = 128000
			info.MaxTokens = 8192
		}
	}

	// Code models
	if strings.Contains(modelLower, "code") || strings.Contains(modelLower, "coder") {
		info.MaxTokens = 8192
		info.ContextWindow = 32768
		info.Description = fmt.Sprintf("LM Studio code model: %s", modelID)
	}

	// Vision models
	if strings.Contains(modelLower, "vision") || strings.Contains(modelLower, "llava") {
		info.SupportsImages = true
		info.Description = fmt.Sprintf("LM Studio vision model: %s", modelID)
	}

	// Mistral models
	if strings.Contains(modelLower, "mistral") {
		info.ContextWindow = 32768
		info.MaxTokens = 8192
	}

	// Qwen models
	if strings.Contains(modelLower, "qwen") {
		info.ContextWindow = 32768
		info.MaxTokens = 8192
	}

	return info
}

// streamRequest makes a streaming request to the LM Studio API
func (h *LMStudioHandler) streamRequest(ctx context.Context, request LMStudioRequest) (llm.ApiStream, error) {
	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", h.baseURL+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// LM Studio typically doesn't require API key for local access
	if h.options.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.options.APIKey)
	}

	// Add custom headers if specified
	for key, value := range h.options.OpenAIHeaders {
		req.Header.Set(key, value)
	}

	// Make request
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, llm.WrapHTTPError(fmt.Errorf("request failed: %w", err), resp)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, llm.WrapHTTPError(fmt.Errorf("API error %d: %s", resp.StatusCode, string(body)), resp)
	}

	// Create stream channel
	streamChan := make(chan llm.ApiStreamChunk, 100)

	// Start streaming goroutine
	go func() {
		defer close(streamChan)
		defer resp.Body.Close()

		h.processStream(resp.Body, streamChan)
	}()

	return streamChan, nil
}

// processStream processes the streaming response from LM Studio
func (h *LMStudioHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
	scanner := NewSSEScanner(reader)

	for scanner.Scan() {
		event := scanner.Event()

		// Skip non-data events
		if event.Type != "data" {
			continue
		}

		// Handle [DONE] marker
		if strings.TrimSpace(event.Data) == "[DONE]" {
			break
		}

		// Parse the event data
		var streamEvent LMStudioStreamEvent
		if err := json.Unmarshal([]byte(event.Data), &streamEvent); err != nil {
			continue // Skip malformed events
		}

		// Process choices
		for _, choice := range streamEvent.Choices {
			if choice.Delta != nil && choice.Delta.Content != "" {
				streamChan <- llm.ApiStreamTextChunk{
					Text: choice.Delta.Content,
				}
			}

			// Handle finish reason and send usage if available
			if choice.FinishReason != nil && *choice.FinishReason != "" {
				if streamEvent.Usage != nil {
					streamChan <- llm.ApiStreamUsageChunk{
						InputTokens:  streamEvent.Usage.PromptTokens,
						OutputTokens: streamEvent.Usage.CompletionTokens,
					}
				}
			}
		}
	}
}
