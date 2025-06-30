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

// QwenHandler implements the ApiHandler interface for Qwen models
// Qwen is OpenAI-compatible via Alibaba Cloud DashScope
type QwenHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// QwenRequest represents a request to Qwen's API (OpenAI-compatible)
type QwenRequest struct {
	Model         string                    `json:"model"`
	Messages      []transform.OpenAIMessage `json:"messages"`
	MaxTokens     *int                      `json:"max_tokens,omitempty"`
	Temperature   *float64                  `json:"temperature,omitempty"`
	Stream        bool                      `json:"stream"`
	StreamOptions *QwenStreamOptions        `json:"stream_options,omitempty"`
}

// QwenStreamOptions configures streaming behavior
type QwenStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// QwenStreamEvent represents a streaming event from Qwen
type QwenStreamEvent struct {
	ID      string       `json:"id"`
	Object  string       `json:"object"`
	Created int64        `json:"created"`
	Model   string       `json:"model"`
	Choices []QwenChoice `json:"choices"`
	Usage   *QwenUsage   `json:"usage,omitempty"`
}

// QwenChoice represents a choice in the response
type QwenChoice struct {
	Index        int          `json:"index"`
	Delta        *QwenDelta   `json:"delta,omitempty"`
	Message      *QwenMessage `json:"message,omitempty"`
	FinishReason *string      `json:"finish_reason,omitempty"`
}

// QwenDelta represents incremental content in streaming
type QwenDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// QwenMessage represents a complete message
type QwenMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// QwenUsage represents token usage information
type QwenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewQwenHandler creates a new Qwen handler
func NewQwenHandler(options llm.ApiHandlerOptions) *QwenHandler {
	baseURL := options.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	}

	// Configure timeout based on request timeout option
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &QwenHandler{
		options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *QwenHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := QwenRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &QwenStreamOptions{
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

	return h.streamRequest(ctx, request)
}

// GetModel implements the ApiHandler interface
func (h *QwenHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *QwenHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Qwen provides usage in the final stream event
	return nil, nil
}

// getDefaultModelInfo returns default model information for Qwen models
func (h *QwenHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for Qwen models
	info := llm.ModelInfo{
		MaxTokens:           8192,
		ContextWindow:       32768,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.5, // Approximate pricing
		OutputPrice:         1.5, // Approximate pricing
		Description:         fmt.Sprintf("Qwen model: %s", modelID),
	}

	// Model-specific configurations
	modelLower := strings.ToLower(modelID)

	// Qwen2.5 models
	if strings.Contains(modelLower, "qwen2.5") {
		if strings.Contains(modelLower, "72b") {
			info.ContextWindow = 128000
			info.MaxTokens = 8192
			info.InputPrice = 2.0
			info.OutputPrice = 6.0
		} else if strings.Contains(modelLower, "32b") {
			info.ContextWindow = 128000
			info.MaxTokens = 8192
			info.InputPrice = 1.0
			info.OutputPrice = 3.0
		} else if strings.Contains(modelLower, "14b") {
			info.ContextWindow = 128000
			info.MaxTokens = 8192
			info.InputPrice = 0.7
			info.OutputPrice = 2.0
		} else if strings.Contains(modelLower, "7b") {
			info.ContextWindow = 128000
			info.MaxTokens = 8192
			info.InputPrice = 0.3
			info.OutputPrice = 0.6
		}
	}

	// Qwen2 models
	if strings.Contains(modelLower, "qwen2") && !strings.Contains(modelLower, "qwen2.5") {
		info.ContextWindow = 32768
		info.MaxTokens = 8192
	}

	// Code models
	if strings.Contains(modelLower, "coder") {
		info.MaxTokens = 8192
		info.ContextWindow = 32768
		info.Description = fmt.Sprintf("Qwen code model: %s", modelID)
	}

	// Vision models
	if strings.Contains(modelLower, "vl") {
		info.SupportsImages = true
		info.Description = fmt.Sprintf("Qwen vision model: %s", modelID)
	}

	return info
}

// streamRequest makes a streaming request to the Qwen API
func (h *QwenHandler) streamRequest(ctx context.Context, request QwenRequest) (llm.ApiStream, error) {
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

// processStream processes the streaming response from Qwen
func (h *QwenHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
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
		var streamEvent QwenStreamEvent
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
