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

// XAIHandler implements the ApiHandler interface for xAI's Grok API
type XAIHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// XAIRequest represents a request to xAI's API (OpenAI-compatible)
type XAIRequest struct {
	Model       string                    `json:"model"`
	Messages    []transform.OpenAIMessage `json:"messages"`
	MaxTokens   *int                      `json:"max_tokens,omitempty"`
	Temperature *float64                  `json:"temperature,omitempty"`
	Stream      bool                      `json:"stream"`
	User        string                    `json:"user,omitempty"`
}

// XAIStreamEvent represents a streaming event from xAI
type XAIStreamEvent struct {
	ID      string      `json:"id"`
	Object  string      `json:"object"`
	Created int64       `json:"created"`
	Model   string      `json:"model"`
	Choices []XAIChoice `json:"choices"`
	Usage   *XAIUsage   `json:"usage,omitempty"`
}

// XAIChoice represents a choice in the response
type XAIChoice struct {
	Index        int         `json:"index"`
	Delta        *XAIDelta   `json:"delta,omitempty"`
	Message      *XAIMessage `json:"message,omitempty"`
	FinishReason *string     `json:"finish_reason,omitempty"`
}

// XAIDelta represents delta content
type XAIDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// XAIMessage represents a complete message
type XAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// XAIUsage represents token usage
type XAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewXAIHandler creates a new xAI handler
func NewXAIHandler(options llm.ApiHandlerOptions) *XAIHandler {
	baseURL := "https://api.x.ai/v1"

	// Configure timeout
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &XAIHandler{
		options: options,
		client:  &http.Client{Timeout: timeout},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *XAIHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := XAIRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
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
func (h *XAIHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *XAIHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// xAI provides usage in the final stream event
	return nil, nil
}

// streamRequest handles the streaming request to xAI
func (h *XAIHandler) streamRequest(ctx context.Context, request XAIRequest) (llm.ApiStream, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.options.APIKey)

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

	// Start processing response in goroutine
	go h.processStreamResponse(resp, streamChan)

	return streamChan, nil
}

// processStreamResponse processes the streaming response
func (h *XAIHandler) processStreamResponse(resp *http.Response, streamChan chan<- llm.ApiStreamChunk) {
	defer resp.Body.Close()
	defer close(streamChan)

	decoder := json.NewDecoder(resp.Body)

	for {
		var line string
		if err := decoder.Decode(&line); err != nil {
			if err == io.EOF {
				break
			}
			continue // Skip malformed events
		}

		// Skip empty lines and data: prefix
		line = strings.TrimSpace(line)
		if line == "" || line == "data: [DONE]" {
			continue
		}

		line = strings.TrimPrefix(line, "data: ")

		var streamEvent XAIStreamEvent
		if err := json.Unmarshal([]byte(line), &streamEvent); err != nil {
			continue // Skip malformed events
		}

		// Process choices
		for _, choice := range streamEvent.Choices {
			if choice.Delta != nil && choice.Delta.Content != "" {
				streamChan <- llm.ApiStreamTextChunk{Text: choice.Delta.Content}
			}

			if choice.FinishReason != nil && *choice.FinishReason != "" {
				// Send usage information if available
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

// getDefaultModelInfo provides default model info based on model ID
func (h *XAIHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       131072,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          5.0,  // Grok pricing (example)
		OutputPrice:         15.0, // Grok pricing (example)
		Temperature:         &[]float64{1.0}[0],
		Description:         "Grok model by xAI",
	}

	// Model-specific configurations
	switch {
	case strings.Contains(modelID, "grok-3"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.Description = "Grok-3 by xAI - Latest reasoning model"
	case strings.Contains(modelID, "grok-2"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.Description = "Grok-2 by xAI - Advanced reasoning model"
	case strings.Contains(modelID, "grok-beta"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.Description = "Grok Beta by xAI - Experimental model"
	}

	return info
}
