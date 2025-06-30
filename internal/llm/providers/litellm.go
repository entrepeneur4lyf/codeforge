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

// LiteLLMHandler implements the ApiHandler interface for LiteLLM proxy
// LiteLLM is a proxy that provides OpenAI-compatible API for 100+ LLM providers
type LiteLLMHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// LiteLLMRequest represents a request to LiteLLM's API (OpenAI-compatible)
type LiteLLMRequest struct {
	Model         string                    `json:"model"`
	Messages      []transform.OpenAIMessage `json:"messages"`
	MaxTokens     *int                      `json:"max_tokens,omitempty"`
	Temperature   *float64                  `json:"temperature,omitempty"`
	Stream        bool                      `json:"stream"`
	StreamOptions *LiteLLMStreamOptions     `json:"stream_options,omitempty"`
	User          string                    `json:"user,omitempty"`
}

// LiteLLMStreamOptions configures streaming behavior
type LiteLLMStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// LiteLLMStreamEvent represents a streaming event from LiteLLM
type LiteLLMStreamEvent struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []LiteLLMChoice `json:"choices"`
	Usage   *LiteLLMUsage   `json:"usage,omitempty"`
}

// LiteLLMChoice represents a choice in the response
type LiteLLMChoice struct {
	Index        int             `json:"index"`
	Delta        *LiteLLMDelta   `json:"delta,omitempty"`
	Message      *LiteLLMMessage `json:"message,omitempty"`
	FinishReason *string         `json:"finish_reason,omitempty"`
}

// LiteLLMDelta represents incremental content in streaming
type LiteLLMDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// LiteLLMMessage represents a complete message
type LiteLLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LiteLLMUsage represents token usage information
type LiteLLMUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewLiteLLMHandler creates a new LiteLLM handler
func NewLiteLLMHandler(options llm.ApiHandlerOptions) *LiteLLMHandler {
	baseURL := options.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:4000" // Default LiteLLM proxy server
	}

	// Configure timeout based on request timeout option
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &LiteLLMHandler{
		options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *LiteLLMHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := LiteLLMRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &LiteLLMStreamOptions{
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
func (h *LiteLLMHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *LiteLLMHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// LiteLLM provides usage in the final stream event
	return nil, nil
}

// getDefaultModelInfo returns default model information for LiteLLM models
func (h *LiteLLMHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for LiteLLM proxy models
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       32768,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.5, // Variable pricing through proxy
		OutputPrice:         1.5, // Variable pricing through proxy
		Description:         fmt.Sprintf("LiteLLM proxy model: %s", modelID),
	}

	// Model-specific configurations based on common patterns
	modelLower := strings.ToLower(modelID)

	// OpenAI models through LiteLLM
	if strings.Contains(modelLower, "gpt-4") {
		info.ContextWindow = 128000
		info.MaxTokens = 8192
		info.InputPrice = 10.0
		info.OutputPrice = 30.0
	} else if strings.Contains(modelLower, "gpt-3.5") {
		info.ContextWindow = 16384
		info.MaxTokens = 4096
		info.InputPrice = 0.5
		info.OutputPrice = 1.5
	}

	// Claude models through LiteLLM
	if strings.Contains(modelLower, "claude") {
		info.ContextWindow = 200000
		info.MaxTokens = 8192
		info.InputPrice = 3.0
		info.OutputPrice = 15.0
	}

	// Gemini models through LiteLLM
	if strings.Contains(modelLower, "gemini") {
		info.ContextWindow = 1000000
		info.MaxTokens = 8192
		info.InputPrice = 0.075
		info.OutputPrice = 0.3
		info.SupportsImages = true
	}

	// Llama models through LiteLLM
	if strings.Contains(modelLower, "llama") {
		info.ContextWindow = 128000
		info.MaxTokens = 8192
		info.InputPrice = 0.2
		info.OutputPrice = 0.2
	}

	return info
}

// streamRequest makes a streaming request to the LiteLLM API
func (h *LiteLLMHandler) streamRequest(ctx context.Context, request LiteLLMRequest) (llm.ApiStream, error) {
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

// processStream processes the streaming response from LiteLLM
func (h *LiteLLMHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
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
		var streamEvent LiteLLMStreamEvent
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
