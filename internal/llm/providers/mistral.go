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

// MistralHandler implements the ApiHandler interface for Mistral AI's API
type MistralHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// MistralRequest represents a request to Mistral's API (OpenAI-compatible)
type MistralRequest struct {
	Model       string                    `json:"model"`
	Messages    []transform.OpenAIMessage `json:"messages"`
	MaxTokens   *int                      `json:"max_tokens,omitempty"`
	Temperature *float64                  `json:"temperature,omitempty"`
	Stream      bool                      `json:"stream"`
	SafePrompt  bool                      `json:"safe_prompt,omitempty"`
}

// MistralStreamEvent represents a streaming event from Mistral
type MistralStreamEvent struct {
	ID      string          `json:"id"`
	Object  string          `json:"object"`
	Created int64           `json:"created"`
	Model   string          `json:"model"`
	Choices []MistralChoice `json:"choices"`
	Usage   *MistralUsage   `json:"usage,omitempty"`
}

// MistralChoice represents a choice in the response
type MistralChoice struct {
	Index        int             `json:"index"`
	Delta        *MistralDelta   `json:"delta,omitempty"`
	Message      *MistralMessage `json:"message,omitempty"`
	FinishReason *string         `json:"finish_reason,omitempty"`
}

// MistralDelta represents delta content
type MistralDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// MistralMessage represents a complete message
type MistralMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// MistralUsage represents token usage
type MistralUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewMistralHandler creates a new Mistral handler
func NewMistralHandler(options llm.ApiHandlerOptions) *MistralHandler {
	baseURL := "https://api.mistral.ai/v1"

	// Configure timeout
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &MistralHandler{
		options: options,
		client:  &http.Client{Timeout: timeout},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *MistralHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := MistralRequest{
		Model:      model.ID,
		Messages:   openAIMessages,
		Stream:     true,
		SafePrompt: false, // Disable content filtering for coding tasks
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
func (h *MistralHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *MistralHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Mistral provides usage in the final stream event
	return nil, nil
}

// streamRequest handles the streaming request to Mistral
func (h *MistralHandler) streamRequest(ctx context.Context, request MistralRequest) (llm.ApiStream, error) {
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
func (h *MistralHandler) processStreamResponse(resp *http.Response, streamChan chan<- llm.ApiStreamChunk) {
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

		var streamEvent MistralStreamEvent
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
func (h *MistralHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       32768,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          1.0, // Mistral pricing (example)
		OutputPrice:         3.0, // Mistral pricing (example)
		Temperature:         &[]float64{0.7}[0],
		Description:         "Mistral AI model",
	}

	// Model-specific configurations
	switch {
	case strings.Contains(modelID, "large"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.Description = "Mistral Large - Most capable model"
	case strings.Contains(modelID, "medium"):
		info.ContextWindow = 32768
		info.MaxTokens = 4096
		info.Description = "Mistral Medium - Balanced performance"
	case strings.Contains(modelID, "small"):
		info.ContextWindow = 32768
		info.MaxTokens = 4096
		info.Description = "Mistral Small - Fast and efficient"
	case strings.Contains(modelID, "codestral"):
		info.ContextWindow = 32768
		info.MaxTokens = 8192
		info.Description = "Codestral - Specialized for code generation"
	case strings.Contains(modelID, "mixtral"):
		info.ContextWindow = 32768
		info.MaxTokens = 4096
		info.Description = "Mixtral - Mixture of experts model"
	}

	return info
}
