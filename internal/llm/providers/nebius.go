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

// NebiusHandler implements the ApiHandler interface for Nebius AI platform
// Nebius is a cloud AI platform providing various LLM services
type NebiusHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// NebiusRequest represents a request to Nebius's API (OpenAI-compatible)
type NebiusRequest struct {
	Model         string                    `json:"model"`
	Messages      []transform.OpenAIMessage `json:"messages"`
	MaxTokens     *int                      `json:"max_tokens,omitempty"`
	Temperature   *float64                  `json:"temperature,omitempty"`
	Stream        bool                      `json:"stream"`
	StreamOptions *NebiusStreamOptions      `json:"stream_options,omitempty"`
	User          string                    `json:"user,omitempty"`
}

// NebiusStreamOptions configures streaming behavior
type NebiusStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// NebiusStreamEvent represents a streaming event from Nebius
type NebiusStreamEvent struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []NebiusChoice `json:"choices"`
	Usage   *NebiusUsage   `json:"usage,omitempty"`
}

// NebiusChoice represents a choice in the response
type NebiusChoice struct {
	Index        int            `json:"index"`
	Delta        *NebiusDelta   `json:"delta,omitempty"`
	Message      *NebiusMessage `json:"message,omitempty"`
	FinishReason *string        `json:"finish_reason,omitempty"`
}

// NebiusDelta represents incremental content in streaming
type NebiusDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// NebiusMessage represents a complete message
type NebiusMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NebiusUsage represents token usage information
type NebiusUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewNebiusHandler creates a new Nebius handler
func NewNebiusHandler(options llm.ApiHandlerOptions) *NebiusHandler {
	baseURL := options.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "https://api.nebius.ai/v1"
	}

	// Configure timeout based on request timeout option
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &NebiusHandler{
		options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *NebiusHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := NebiusRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &NebiusStreamOptions{
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
func (h *NebiusHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *NebiusHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Nebius provides usage in the final stream event
	return nil, nil
}

// getDefaultModelInfo returns default model information for Nebius models
func (h *NebiusHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for Nebius models
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       32768,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.6, // Cloud platform pricing
		OutputPrice:         1.2, // Cloud platform pricing
		Description:         fmt.Sprintf("Nebius AI model: %s", modelID),
	}

	// Model-specific configurations
	modelLower := strings.ToLower(modelID)

	// Large models
	if strings.Contains(modelLower, "large") || strings.Contains(modelLower, "70b") {
		info.ContextWindow = 128000
		info.MaxTokens = 8192
		info.InputPrice = 1.5
		info.OutputPrice = 3.0
	}

	// Small/fast models
	if strings.Contains(modelLower, "small") || strings.Contains(modelLower, "fast") {
		info.ContextWindow = 16384
		info.MaxTokens = 4096
		info.InputPrice = 0.3
		info.OutputPrice = 0.6
	}

	// Vision models
	if strings.Contains(modelLower, "vision") {
		info.SupportsImages = true
		info.Description = fmt.Sprintf("Nebius AI vision model: %s", modelID)
	}

	return info
}

// streamRequest makes a streaming request to the Nebius API
func (h *NebiusHandler) streamRequest(ctx context.Context, request NebiusRequest) (llm.ApiStream, error) {
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

// processStream processes the streaming response from Nebius
func (h *NebiusHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
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
		var streamEvent NebiusStreamEvent
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
