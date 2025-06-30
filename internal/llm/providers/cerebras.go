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

// CerebrasHandler implements the ApiHandler interface for Cerebras's API
// Cerebras is OpenAI-compatible with ultra-fast inference
type CerebrasHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// CerebrasRequest represents a request to Cerebras's API (OpenAI-compatible)
type CerebrasRequest struct {
	Model         string                    `json:"model"`
	Messages      []transform.OpenAIMessage `json:"messages"`
	MaxTokens     *int                      `json:"max_tokens,omitempty"`
	Temperature   *float64                  `json:"temperature,omitempty"`
	Stream        bool                      `json:"stream"`
	StreamOptions *CerebrasStreamOptions    `json:"stream_options,omitempty"`
	User          string                    `json:"user,omitempty"`
}

// CerebrasStreamOptions configures streaming behavior
type CerebrasStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// CerebrasStreamEvent represents a streaming event from Cerebras
type CerebrasStreamEvent struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []CerebrasChoice `json:"choices"`
	Usage   *CerebrasUsage   `json:"usage,omitempty"`
}

// CerebrasChoice represents a choice in the response
type CerebrasChoice struct {
	Index        int              `json:"index"`
	Delta        *CerebrasDelta   `json:"delta,omitempty"`
	Message      *CerebrasMessage `json:"message,omitempty"`
	FinishReason *string          `json:"finish_reason,omitempty"`
}

// CerebrasDelta represents incremental content in streaming
type CerebrasDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// CerebrasMessage represents a complete message
type CerebrasMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CerebrasUsage represents token usage information
type CerebrasUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewCerebrasHandler creates a new Cerebras handler
func NewCerebrasHandler(options llm.ApiHandlerOptions) *CerebrasHandler {
	baseURL := options.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "https://api.cerebras.ai/v1"
	}

	// Configure timeout based on request timeout option
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &CerebrasHandler{
		options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *CerebrasHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := CerebrasRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &CerebrasStreamOptions{
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
func (h *CerebrasHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *CerebrasHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Cerebras provides usage in the final stream event
	return nil, nil
}

// getDefaultModelInfo returns default model information for Cerebras models
func (h *CerebrasHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for Cerebras models
	info := llm.ModelInfo{
		MaxTokens:           8192,
		ContextWindow:       128000,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.1, // $0.10 per 1M tokens (ultra-fast inference)
		OutputPrice:         0.1, // $0.10 per 1M tokens (ultra-fast inference)
		Description:         fmt.Sprintf("Cerebras ultra-fast model: %s", modelID),
	}

	// Model-specific configurations
	modelLower := strings.ToLower(modelID)

	// Llama models on Cerebras (ultra-fast)
	if strings.Contains(modelLower, "llama") {
		if strings.Contains(modelLower, "70b") {
			info.ContextWindow = 128000
			info.MaxTokens = 8192
			info.InputPrice = 0.6
			info.OutputPrice = 0.6
		} else if strings.Contains(modelLower, "8b") || strings.Contains(modelLower, "7b") {
			info.ContextWindow = 128000
			info.MaxTokens = 8192
			info.InputPrice = 0.1
			info.OutputPrice = 0.1
		}
	}

	// Cerebras-specific models
	if strings.Contains(modelLower, "cerebras") {
		info.MaxTokens = 8192
		info.ContextWindow = 128000
		info.Description = fmt.Sprintf("Cerebras native model: %s", modelID)
	}

	return info
}

// streamRequest makes a streaming request to the Cerebras API
func (h *CerebrasHandler) streamRequest(ctx context.Context, request CerebrasRequest) (llm.ApiStream, error) {
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

// processStream processes the streaming response from Cerebras
func (h *CerebrasHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
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
		var streamEvent CerebrasStreamEvent
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
