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

// RequestyHandler implements the ApiHandler interface for Requesty API proxy
// Requesty is an API testing and proxy tool that can route to various LLM providers
type RequestyHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// RequestyRequest represents a request to Requesty's API (OpenAI-compatible)
type RequestyRequest struct {
	Model         string                    `json:"model"`
	Messages      []transform.OpenAIMessage `json:"messages"`
	MaxTokens     *int                      `json:"max_tokens,omitempty"`
	Temperature   *float64                  `json:"temperature,omitempty"`
	Stream        bool                      `json:"stream"`
	StreamOptions *RequestyStreamOptions    `json:"stream_options,omitempty"`
	User          string                    `json:"user,omitempty"`
}

// RequestyStreamOptions configures streaming behavior
type RequestyStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// RequestyStreamEvent represents a streaming event from Requesty
type RequestyStreamEvent struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []RequestyChoice `json:"choices"`
	Usage   *RequestyUsage   `json:"usage,omitempty"`
}

// RequestyChoice represents a choice in the response
type RequestyChoice struct {
	Index        int              `json:"index"`
	Delta        *RequestyDelta   `json:"delta,omitempty"`
	Message      *RequestyMessage `json:"message,omitempty"`
	FinishReason *string          `json:"finish_reason,omitempty"`
}

// RequestyDelta represents incremental content in streaming
type RequestyDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// RequestyMessage represents a complete message
type RequestyMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RequestyUsage represents token usage information
type RequestyUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewRequestyHandler creates a new Requesty handler
func NewRequestyHandler(options llm.ApiHandlerOptions) *RequestyHandler {
	baseURL := options.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "https://api.requesty.com/v1"
	}

	// Configure timeout based on request timeout option
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &RequestyHandler{
		options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *RequestyHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := RequestyRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &RequestyStreamOptions{
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
func (h *RequestyHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *RequestyHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Requesty provides usage in the final stream event
	return nil, nil
}

// getDefaultModelInfo returns default model information for Requesty proxied models
func (h *RequestyHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for Requesty proxied models
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       32768,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.5, // Proxy service pricing
		OutputPrice:         1.0, // Proxy service pricing
		Description:         fmt.Sprintf("Requesty proxied model: %s", modelID),
	}

	// Model-specific configurations based on common patterns
	modelLower := strings.ToLower(modelID)

	// GPT models
	if strings.Contains(modelLower, "gpt-4") {
		info.ContextWindow = 128000
		info.MaxTokens = 8192
		info.InputPrice = 30.0
		info.OutputPrice = 60.0
	} else if strings.Contains(modelLower, "gpt-3.5") {
		info.ContextWindow = 16384
		info.MaxTokens = 4096
		info.InputPrice = 1.5
		info.OutputPrice = 2.0
	}

	// Claude models
	if strings.Contains(modelLower, "claude") {
		info.ContextWindow = 200000
		info.MaxTokens = 8192
		info.InputPrice = 15.0
		info.OutputPrice = 75.0
	}

	// Vision models
	if strings.Contains(modelLower, "vision") {
		info.SupportsImages = true
		info.Description = fmt.Sprintf("Requesty proxied vision model: %s", modelID)
	}

	return info
}

// streamRequest makes a streaming request to the Requesty API
func (h *RequestyHandler) streamRequest(ctx context.Context, request RequestyRequest) (llm.ApiStream, error) {
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
	req.Header.Set("Authorization", "Bearer "+h.options.APIKey)

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

// processStream processes the streaming response from Requesty
func (h *RequestyHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
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
		var streamEvent RequestyStreamEvent
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
