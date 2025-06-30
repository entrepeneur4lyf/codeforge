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

// AskSageHandler implements the ApiHandler interface for AskSage AI platform
// AskSage is an AI assistant platform providing various LLM services
type AskSageHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// AskSageRequest represents a request to AskSage's API (OpenAI-compatible)
type AskSageRequest struct {
	Model         string                    `json:"model"`
	Messages      []transform.OpenAIMessage `json:"messages"`
	MaxTokens     *int                      `json:"max_tokens,omitempty"`
	Temperature   *float64                  `json:"temperature,omitempty"`
	Stream        bool                      `json:"stream"`
	StreamOptions *AskSageStreamOptions     `json:"stream_options,omitempty"`
	User          string                    `json:"user,omitempty"`
}

// AskSageStreamOptions configures streaming behavior
type AskSageStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// AskSageStreamEvent represents a streaming event from AskSage
type AskSageStreamEvent struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []AskSageChoice `json:"choices"`
	Usage   *AskSageUsage   `json:"usage,omitempty"`
}

// AskSageChoice represents a choice in the response
type AskSageChoice struct {
	Index        int             `json:"index"`
	Delta        *AskSageDelta   `json:"delta,omitempty"`
	Message      *AskSageMessage `json:"message,omitempty"`
	FinishReason *string         `json:"finish_reason,omitempty"`
}

// AskSageDelta represents incremental content in streaming
type AskSageDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// AskSageMessage represents a complete message
type AskSageMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AskSageUsage represents token usage information
type AskSageUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewAskSageHandler creates a new AskSage handler
func NewAskSageHandler(options llm.ApiHandlerOptions) *AskSageHandler {
	baseURL := options.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "https://api.asksage.ai/v1"
	}

	// Configure timeout based on request timeout option
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &AskSageHandler{
		options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *AskSageHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := AskSageRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &AskSageStreamOptions{
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
func (h *AskSageHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *AskSageHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// AskSage provides usage in the final stream event
	return nil, nil
}

// getDefaultModelInfo returns default model information for AskSage models
func (h *AskSageHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for AskSage models
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       32768,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.4, // Assistant platform pricing
		OutputPrice:         0.8, // Assistant platform pricing
		Description:         fmt.Sprintf("AskSage AI model: %s", modelID),
	}

	// Model-specific configurations
	modelLower := strings.ToLower(modelID)

	// Pro models
	if strings.Contains(modelLower, "pro") {
		info.ContextWindow = 128000
		info.MaxTokens = 8192
		info.InputPrice = 1.0
		info.OutputPrice = 2.0
	}

	// Lite models
	if strings.Contains(modelLower, "lite") {
		info.ContextWindow = 16384
		info.MaxTokens = 4096
		info.InputPrice = 0.2
		info.OutputPrice = 0.4
	}

	// Coding models
	if strings.Contains(modelLower, "code") {
		info.MaxTokens = 8192
		info.ContextWindow = 32768
		info.Description = fmt.Sprintf("AskSage coding model: %s", modelID)
	}

	return info
}

// streamRequest makes a streaming request to the AskSage API
func (h *AskSageHandler) streamRequest(ctx context.Context, request AskSageRequest) (llm.ApiStream, error) {
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

// processStream processes the streaming response from AskSage
func (h *AskSageHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
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
		var streamEvent AskSageStreamEvent
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
