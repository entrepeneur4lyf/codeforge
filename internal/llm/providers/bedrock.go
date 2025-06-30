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

// BedrockHandler implements the ApiHandler interface for AWS Bedrock
// Bedrock provides access to foundation models from various providers
type BedrockHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
	region  string
}

// BedrockRequest represents a request to Bedrock's API
type BedrockRequest struct {
	ModelID          string                    `json:"modelId"`
	Messages         []transform.OpenAIMessage `json:"messages"`
	MaxTokens        *int                      `json:"maxTokens,omitempty"`
	Temperature      *float64                  `json:"temperature,omitempty"`
	TopP             *float64                  `json:"topP,omitempty"`
	StopSequences    []string                  `json:"stopSequences,omitempty"`
	AnthropicVersion string                    `json:"anthropic_version,omitempty"`
	Stream           bool                      `json:"stream,omitempty"`
}

// BedrockStreamEvent represents a streaming event from Bedrock
type BedrockStreamEvent struct {
	Type    string          `json:"type"`
	Message *BedrockMessage `json:"message,omitempty"`
	Delta   *BedrockDelta   `json:"delta,omitempty"`
	Usage   *BedrockUsage   `json:"usage,omitempty"`
}

// BedrockMessage represents a complete message
type BedrockMessage struct {
	ID      string           `json:"id"`
	Type    string           `json:"type"`
	Role    string           `json:"role"`
	Content []BedrockContent `json:"content"`
	Model   string           `json:"model"`
	Usage   *BedrockUsage    `json:"usage,omitempty"`
}

// BedrockContent represents content in a message
type BedrockContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// BedrockDelta represents incremental content in streaming
type BedrockDelta struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// BedrockUsage represents token usage information
type BedrockUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewBedrockHandler creates a new Bedrock handler
func NewBedrockHandler(options llm.ApiHandlerOptions) *BedrockHandler {
	region := "us-east-1" // Default region
	// Could be configured via environment variable or options in the future

	baseURL := options.OpenAIBaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", region)
	}

	// Configure timeout based on request timeout option
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &BedrockHandler{
		options: options,
		region:  region,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *BedrockHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to Bedrock format
	bedrockMessages, err := h.convertMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request based on model type
	if strings.Contains(strings.ToLower(model.ID), "claude") {
		return h.streamClaudeRequest(ctx, model, bedrockMessages)
	} else if strings.Contains(strings.ToLower(model.ID), "llama") {
		return h.streamLlamaRequest(ctx, model, bedrockMessages)
	} else {
		// Default to Claude format
		return h.streamClaudeRequest(ctx, model, bedrockMessages)
	}
}

// GetModel implements the ApiHandler interface
func (h *BedrockHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *BedrockHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Bedrock provides usage in the stream
	return nil, nil
}

// getDefaultModelInfo returns default model information for Bedrock models
func (h *BedrockHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for Bedrock models
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       200000,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          3.0,  // $3.00 per 1M tokens (Claude 3.5 Sonnet)
		OutputPrice:         15.0, // $15.00 per 1M tokens (Claude 3.5 Sonnet)
		Description:         fmt.Sprintf("AWS Bedrock model: %s", modelID),
	}

	// Model-specific configurations
	modelLower := strings.ToLower(modelID)

	// Claude models on Bedrock
	if strings.Contains(modelLower, "claude-3-5-sonnet") {
		info.ContextWindow = 200000
		info.MaxTokens = 8192
		info.InputPrice = 3.0
		info.OutputPrice = 15.0
		info.SupportsImages = true
	} else if strings.Contains(modelLower, "claude-3-haiku") {
		info.ContextWindow = 200000
		info.MaxTokens = 4096
		info.InputPrice = 0.25
		info.OutputPrice = 1.25
		info.SupportsImages = true
	} else if strings.Contains(modelLower, "claude-3-opus") {
		info.ContextWindow = 200000
		info.MaxTokens = 4096
		info.InputPrice = 15.0
		info.OutputPrice = 75.0
		info.SupportsImages = true
	}

	// Llama models on Bedrock
	if strings.Contains(modelLower, "llama") {
		info.ContextWindow = 128000
		info.MaxTokens = 8192
		info.InputPrice = 0.65
		info.OutputPrice = 0.65
		info.SupportsImages = false
	}

	// Titan models
	if strings.Contains(modelLower, "titan") {
		info.ContextWindow = 32000
		info.MaxTokens = 4096
		info.InputPrice = 0.5
		info.OutputPrice = 1.5
	}

	return info
}

// convertMessages converts LLM messages to Bedrock format
func (h *BedrockHandler) convertMessages(systemPrompt string, messages []llm.Message) ([]transform.OpenAIMessage, error) {
	return convertToOpenAIMessages(systemPrompt, messages)
}

// streamClaudeRequest makes a streaming request for Claude models on Bedrock
func (h *BedrockHandler) streamClaudeRequest(ctx context.Context, model llm.ModelResponse, messages []transform.OpenAIMessage) (llm.ApiStream, error) {
	// Prepare Claude-specific request
	request := map[string]interface{}{
		"anthropic_version": "bedrock-2023-05-31",
		"max_tokens":        model.Info.MaxTokens,
		"messages":          messages,
		"stream":            true,
	}

	// Set temperature if specified
	if model.Info.Temperature != nil {
		request["temperature"] = *model.Info.Temperature
	}

	return h.streamRequest(ctx, request, model.ID)
}

// streamLlamaRequest makes a streaming request for Llama models on Bedrock
func (h *BedrockHandler) streamLlamaRequest(ctx context.Context, model llm.ModelResponse, messages []transform.OpenAIMessage) (llm.ApiStream, error) {
	// Convert messages to Llama format (single prompt)
	var prompt strings.Builder
	for _, msg := range messages {
		prompt.WriteString(fmt.Sprintf("%s: %s\n", msg.Role, msg.Content))
	}

	request := map[string]interface{}{
		"prompt":      prompt.String(),
		"max_gen_len": model.Info.MaxTokens,
		"stream":      true,
	}

	// Set temperature if specified
	if model.Info.Temperature != nil {
		request["temperature"] = *model.Info.Temperature
	}

	return h.streamRequest(ctx, request, model.ID)
}

// streamRequest makes a streaming request to the Bedrock API
func (h *BedrockHandler) streamRequest(ctx context.Context, request map[string]interface{}, modelID string) (llm.ApiStream, error) {
	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/model/%s/invoke-with-response-stream", h.baseURL, modelID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.amazon.eventstream")

	// AWS authentication would be handled here (AWS SDK or manual signing)
	// For now, use API key if provided
	if h.options.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.options.APIKey)
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

// processStream processes the streaming response from Bedrock
func (h *BedrockHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
	// Bedrock uses AWS event stream format, but for simplicity we'll handle JSON lines
	decoder := json.NewDecoder(reader)

	for {
		var event BedrockStreamEvent
		if err := decoder.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			continue // Skip malformed events
		}

		// Process different event types
		switch event.Type {
		case "content_block_delta":
			if event.Delta != nil && event.Delta.Text != "" {
				streamChan <- llm.ApiStreamTextChunk{
					Text: event.Delta.Text,
				}
			}
		case "message_stop":
			if event.Usage != nil {
				streamChan <- llm.ApiStreamUsageChunk{
					InputTokens:  event.Usage.InputTokens,
					OutputTokens: event.Usage.OutputTokens,
				}
			}
		}
	}
}
