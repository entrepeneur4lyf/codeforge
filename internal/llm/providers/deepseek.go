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

// DeepSeekHandler implements the ApiHandler interface for DeepSeek's API
type DeepSeekHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// DeepSeekRequest represents a request to DeepSeek's API (OpenAI-compatible)
type DeepSeekRequest struct {
	Model       string                    `json:"model"`
	Messages    []transform.OpenAIMessage `json:"messages"`
	MaxTokens   *int                      `json:"max_tokens,omitempty"`
	Temperature *float64                  `json:"temperature,omitempty"`
	Stream      bool                      `json:"stream"`
	User        string                    `json:"user,omitempty"`
}

// DeepSeekStreamEvent represents a streaming event from DeepSeek
type DeepSeekStreamEvent struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []DeepSeekChoice `json:"choices"`
	Usage   *DeepSeekUsage   `json:"usage,omitempty"`
}

// DeepSeekChoice represents a choice in the response
type DeepSeekChoice struct {
	Index        int              `json:"index"`
	Delta        *DeepSeekDelta   `json:"delta,omitempty"`
	Message      *DeepSeekMessage `json:"message,omitempty"`
	FinishReason *string          `json:"finish_reason,omitempty"`
}

// DeepSeekDelta represents delta content
type DeepSeekDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// DeepSeekMessage represents a complete message
type DeepSeekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// DeepSeekUsage represents token usage
type DeepSeekUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewDeepSeekHandler creates a new DeepSeek handler
func NewDeepSeekHandler(options llm.ApiHandlerOptions) *DeepSeekHandler {
	baseURL := "https://api.deepseek.com/v1"

	// Configure timeout
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &DeepSeekHandler{
		options: options,
		client:  &http.Client{Timeout: timeout},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *DeepSeekHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := DeepSeekRequest{
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
func (h *DeepSeekHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *DeepSeekHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// DeepSeek provides usage in the final stream event
	return nil, nil
}

// streamRequest handles the streaming request to DeepSeek
func (h *DeepSeekHandler) streamRequest(ctx context.Context, request DeepSeekRequest) (llm.ApiStream, error) {
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
func (h *DeepSeekHandler) processStreamResponse(resp *http.Response, streamChan chan<- llm.ApiStreamChunk) {
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

		var streamEvent DeepSeekStreamEvent
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
func (h *DeepSeekHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       32768,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.14, // DeepSeek competitive pricing
		OutputPrice:         0.28, // DeepSeek competitive pricing
		Temperature:         &[]float64{1.0}[0],
		Description:         "DeepSeek model - Excellent for coding",
	}

	// Model-specific configurations
	switch {
	case strings.Contains(modelID, "coder"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.Description = "DeepSeek Coder - Specialized for programming tasks"
	case strings.Contains(modelID, "chat"):
		info.ContextWindow = 32768
		info.MaxTokens = 4096
		info.Description = "DeepSeek Chat - General conversation model"
	case strings.Contains(modelID, "v3"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.Description = "DeepSeek V3 - Latest and most capable model"
	case strings.Contains(modelID, "v2.5"):
		info.ContextWindow = 32768
		info.MaxTokens = 4096
		info.Description = "DeepSeek V2.5 - Improved reasoning model"
	}

	return info
}
