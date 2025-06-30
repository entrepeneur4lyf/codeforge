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

// ClaudeCodeHandler implements the ApiHandler interface for Claude Code
// Claude Code is Anthropic's specialized coding model
type ClaudeCodeHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// ClaudeCodeRequest represents a request to Claude Code's API (OpenAI-compatible format)
type ClaudeCodeRequest struct {
	Model         string                    `json:"model"`
	Messages      []transform.OpenAIMessage `json:"messages"`
	MaxTokens     *int                      `json:"max_tokens,omitempty"`
	Temperature   *float64                  `json:"temperature,omitempty"`
	Stream        bool                      `json:"stream"`
	StreamOptions *ClaudeCodeStreamOptions  `json:"stream_options,omitempty"`
	User          string                    `json:"user,omitempty"`
}

// ClaudeCodeStreamOptions configures streaming behavior
type ClaudeCodeStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// ClaudeCodeStreamEvent represents a streaming event from Claude Code
type ClaudeCodeStreamEvent struct {
	Type  string                  `json:"type"`
	Index int                     `json:"index,omitempty"`
	Delta *ClaudeCodeContentDelta `json:"delta,omitempty"`
	Usage *ClaudeCodeUsage        `json:"usage,omitempty"`
}

// ClaudeCodeContentDelta represents incremental content in streaming
type ClaudeCodeContentDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ClaudeCodeUsage represents token usage information
type ClaudeCodeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewClaudeCodeHandler creates a new Claude Code handler
func NewClaudeCodeHandler(options llm.ApiHandlerOptions) *ClaudeCodeHandler {
	baseURL := options.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	// Configure timeout based on request timeout option
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &ClaudeCodeHandler{
		options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *ClaudeCodeHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := ClaudeCodeRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &ClaudeCodeStreamOptions{
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
func (h *ClaudeCodeHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *ClaudeCodeHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Claude Code provides usage in the final stream event
	return nil, nil
}

// getDefaultModelInfo returns default model information for Claude Code models
func (h *ClaudeCodeHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for Claude Code models
	info := llm.ModelInfo{
		MaxTokens:           8192,
		ContextWindow:       200000,
		SupportsImages:      false,
		SupportsPromptCache: true,
		InputPrice:          15.0, // Anthropic pricing
		OutputPrice:         75.0, // Anthropic pricing
		Description:         fmt.Sprintf("Claude Code specialized coding model: %s", modelID),
	}

	// Model-specific configurations
	modelLower := strings.ToLower(modelID)

	// Claude 3.5 Sonnet Code
	if strings.Contains(modelLower, "3.5") && strings.Contains(modelLower, "sonnet") {
		info.MaxTokens = 8192
		info.ContextWindow = 200000
		info.InputPrice = 3.0
		info.OutputPrice = 15.0
	}

	// Claude 3 Opus Code
	if strings.Contains(modelLower, "opus") {
		info.MaxTokens = 4096
		info.ContextWindow = 200000
		info.InputPrice = 15.0
		info.OutputPrice = 75.0
	}

	// Claude 3 Haiku Code
	if strings.Contains(modelLower, "haiku") {
		info.MaxTokens = 4096
		info.ContextWindow = 200000
		info.InputPrice = 0.25
		info.OutputPrice = 1.25
	}

	return info
}

// streamRequest makes a streaming request to the Claude Code API
func (h *ClaudeCodeHandler) streamRequest(ctx context.Context, request ClaudeCodeRequest) (llm.ApiStream, error) {
	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", h.baseURL+"/v1/messages", bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", h.options.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", "messages-2023-12-15")

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

// processStream processes the streaming response from Claude Code
func (h *ClaudeCodeHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
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
		var streamEvent ClaudeCodeStreamEvent
		if err := json.Unmarshal([]byte(event.Data), &streamEvent); err != nil {
			continue // Skip malformed events
		}

		// Process different event types
		switch streamEvent.Type {
		case "content_block_delta":
			if streamEvent.Delta != nil && streamEvent.Delta.Text != "" {
				streamChan <- llm.ApiStreamTextChunk{
					Text: streamEvent.Delta.Text,
				}
			}
		case "message_delta":
			if streamEvent.Usage != nil {
				streamChan <- llm.ApiStreamUsageChunk{
					InputTokens:  streamEvent.Usage.InputTokens,
					OutputTokens: streamEvent.Usage.OutputTokens,
				}
			}
		}
	}
}
