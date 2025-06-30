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

// TogetherHandler implements the ApiHandler interface for Together AI's API
// Together AI is OpenAI-compatible with some specific features
type TogetherHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// TogetherRequest represents a request to Together AI's API (OpenAI-compatible)
type TogetherRequest struct {
	Model         string                    `json:"model"`
	Messages      []transform.OpenAIMessage `json:"messages"`
	MaxTokens     *int                      `json:"max_tokens,omitempty"`
	Temperature   *float64                  `json:"temperature,omitempty"`
	Stream        bool                      `json:"stream"`
	StreamOptions *TogetherStreamOptions    `json:"stream_options,omitempty"`
	User          string                    `json:"user,omitempty"`
}

// TogetherStreamOptions configures streaming behavior
type TogetherStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// TogetherStreamEvent represents a streaming event from Together AI
type TogetherStreamEvent struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []TogetherChoice `json:"choices"`
	Usage   *TogetherUsage   `json:"usage,omitempty"`
}

// TogetherChoice represents a choice in the response
type TogetherChoice struct {
	Index        int               `json:"index"`
	Delta        *TogetherDelta    `json:"delta,omitempty"`
	Message      *TogetherMessage  `json:"message,omitempty"`
	FinishReason *string           `json:"finish_reason,omitempty"`
	Logprobs     *TogetherLogprobs `json:"logprobs,omitempty"`
}

// TogetherDelta represents incremental content in streaming
type TogetherDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// TogetherMessage represents a complete message
type TogetherMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// TogetherLogprobs represents log probabilities
type TogetherLogprobs struct {
	Content []TogetherTokenLogprob `json:"content,omitempty"`
}

// TogetherTokenLogprob represents token-level log probabilities
type TogetherTokenLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
}

// TogetherUsage represents token usage information
type TogetherUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewTogetherHandler creates a new Together AI handler
func NewTogetherHandler(options llm.ApiHandlerOptions) *TogetherHandler {
	baseURL := options.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "https://api.together.xyz/v1"
	}

	// Configure timeout based on request timeout option
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &TogetherHandler{
		options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *TogetherHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := TogetherRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &TogetherStreamOptions{
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
func (h *TogetherHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *TogetherHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Together AI provides usage in the final stream event
	return nil, nil
}

// getDefaultModelInfo returns default model information for Together AI models
func (h *TogetherHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for Together AI models
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       32768,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.2, // $0.20 per 1M tokens (typical)
		OutputPrice:         0.2, // $0.20 per 1M tokens (typical)
		Description:         fmt.Sprintf("Together AI model: %s", modelID),
	}

	// Model-specific configurations
	modelLower := strings.ToLower(modelID)

	// Llama models
	if strings.Contains(modelLower, "llama") {
		if strings.Contains(modelLower, "70b") || strings.Contains(modelLower, "405b") {
			info.ContextWindow = 128000
			info.MaxTokens = 8192
			info.InputPrice = 0.9
			info.OutputPrice = 0.9
		} else if strings.Contains(modelLower, "8b") || strings.Contains(modelLower, "7b") {
			info.ContextWindow = 128000
			info.MaxTokens = 8192
			info.InputPrice = 0.2
			info.OutputPrice = 0.2
		}
	}

	// Code models
	if strings.Contains(modelLower, "code") {
		info.MaxTokens = 8192
		info.ContextWindow = 32768
		info.Description = fmt.Sprintf("Together AI code model: %s", modelID)
	}

	// Vision models
	if strings.Contains(modelLower, "vision") || strings.Contains(modelLower, "llava") {
		info.SupportsImages = true
		info.Description = fmt.Sprintf("Together AI vision model: %s", modelID)
	}

	return info
}

// streamRequest makes a streaming request to the Together AI API
func (h *TogetherHandler) streamRequest(ctx context.Context, request TogetherRequest) (llm.ApiStream, error) {
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

// processStream processes the streaming response from Together AI
func (h *TogetherHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
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
		var streamEvent TogetherStreamEvent
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
