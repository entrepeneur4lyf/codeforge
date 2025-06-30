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
	"github.com/entrepeneur4lyf/codeforge/internal/llm/models"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/transform"
)

// OpenAIHandler implements the ApiHandler interface for OpenAI's GPT models
// Based on Cline's OpenAiHandler with full feature parity
type OpenAIHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// OpenAIRequest represents the request to OpenAI API
type OpenAIRequest struct {
	Model           string                    `json:"model"`
	Messages        []transform.OpenAIMessage `json:"messages"`
	MaxTokens       *int                      `json:"max_tokens,omitempty"`
	Temperature     *float64                  `json:"temperature,omitempty"`
	Stream          bool                      `json:"stream"`
	Tools           []OpenAITool              `json:"tools,omitempty"`
	ToolChoice      interface{}               `json:"tool_choice,omitempty"`
	ReasoningEffort string                    `json:"reasoning_effort,omitempty"`
	StreamOptions   *OpenAIStreamOptions      `json:"stream_options,omitempty"`
}

// OpenAITool represents a tool definition
type OpenAITool struct {
	Type     string            `json:"type"`
	Function OpenAIFunctionDef `json:"function"`
}

// OpenAIFunctionDef represents a function definition
type OpenAIFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// OpenAIStreamOptions configures streaming behavior
type OpenAIStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// OpenAIStreamEvent represents a streaming event from OpenAI
type OpenAIStreamEvent struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []OpenAIStreamChoice `json:"choices"`
	Usage   *OpenAIUsage         `json:"usage,omitempty"`
}

// OpenAIStreamChoice represents a choice in the stream
type OpenAIStreamChoice struct {
	Index        int               `json:"index"`
	Delta        OpenAIStreamDelta `json:"delta"`
	FinishReason *string           `json:"finish_reason"`
}

// OpenAIStreamDelta represents delta content
type OpenAIStreamDelta struct {
	Role      string                     `json:"role,omitempty"`
	Content   string                     `json:"content,omitempty"`
	ToolCalls []transform.OpenAIToolCall `json:"tool_calls,omitempty"`
	Reasoning string                     `json:"reasoning,omitempty"`
}

// OpenAIUsage represents token usage
type OpenAIUsage struct {
	PromptTokens            int                            `json:"prompt_tokens"`
	CompletionTokens        int                            `json:"completion_tokens"`
	TotalTokens             int                            `json:"total_tokens"`
	PromptTokensDetails     *OpenAIPromptTokensDetails     `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *OpenAICompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

// OpenAIPromptTokensDetails provides detailed prompt token breakdown
type OpenAIPromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

// OpenAICompletionTokensDetails provides detailed completion token breakdown
type OpenAICompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

// NewOpenAIHandler creates a new OpenAI handler
func NewOpenAIHandler(options llm.ApiHandlerOptions) *OpenAIHandler {
	baseURL := options.OpenAIBaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	// Configure timeout based on request timeout option
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &OpenAIHandler{
		options: options,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *OpenAIHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := h.convertMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := OpenAIRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &OpenAIStreamOptions{
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

	// Add reasoning effort for o1 models
	if h.isReasoningModel(model.ID) && h.options.ReasoningEffort != "" {
		request.ReasoningEffort = h.options.ReasoningEffort
		// o1 models don't support temperature or max_tokens
		request.Temperature = nil
		request.MaxTokens = nil
	}

	return h.streamRequest(ctx, request)
}

// GetModel implements the ApiHandler interface
func (h *OpenAIHandler) GetModel() llm.ModelResponse {
	// Try to get model from registry first
	registry := models.NewModelRegistry()
	if canonicalModel, exists := registry.GetModelByProvider(models.ProviderOpenAI, h.options.ModelID); exists {
		return llm.ModelResponse{
			ID:   h.options.ModelID,
			Info: h.convertToLLMModelInfo(canonicalModel),
		}
	}

	// Fallback to default model info based on model type
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *OpenAIHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// OpenAI includes usage in the stream, so this is not needed
	return nil, nil
}

// convertMessages converts LLM messages to OpenAI format
func (h *OpenAIHandler) convertMessages(systemPrompt string, messages []llm.Message) ([]transform.OpenAIMessage, error) {
	var openAIMessages []transform.OpenAIMessage

	// Add system message if provided
	if systemPrompt != "" {
		openAIMessages = append(openAIMessages, transform.CreateSystemMessage(systemPrompt))
	}

	// Convert messages using transform layer
	transformMessages := make([]transform.Message, len(messages))
	for i, msg := range messages {
		transformMessages[i] = transform.Message{
			Role:    msg.Role,
			Content: convertContentBlocksOpenAI(msg.Content),
		}
	}

	convertedMessages, err := transform.ConvertToOpenAIMessages(transformMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	openAIMessages = append(openAIMessages, convertedMessages...)
	return openAIMessages, nil
}

// convertContentBlocksOpenAI converts llm.ContentBlock to transform.ContentBlock
func convertContentBlocksOpenAI(blocks []llm.ContentBlock) []transform.ContentBlock {
	result := make([]transform.ContentBlock, len(blocks))
	for i, block := range blocks {
		switch b := block.(type) {
		case llm.TextBlock:
			result[i] = transform.TextBlock{Text: b.Text}
		case llm.ImageBlock:
			result[i] = transform.ImageBlock{
				Source: transform.ImageSource{
					Type:      b.Source.Type,
					MediaType: b.Source.MediaType,
					Data:      b.Source.Data,
				},
			}
		case llm.ToolUseBlock:
			result[i] = transform.ToolUseBlock{
				ID:    b.ID,
				Name:  b.Name,
				Input: b.Input,
			}
		case llm.ToolResultBlock:
			// Convert ToolResultBlock content to string
			var content string
			for _, contentBlock := range b.Content {
				if textBlock, ok := contentBlock.(llm.TextBlock); ok {
					content += textBlock.Text
				}
			}
			result[i] = transform.ToolResultBlock{
				ToolUseID: b.ToolUseID,
				Content:   content,
				IsError:   b.IsError,
			}
		default:
			// Fallback to text block
			result[i] = transform.TextBlock{Text: fmt.Sprintf("%v", block)}
		}
	}
	return result
}

// isReasoningModel checks if a model is a reasoning model (o1, o3)
func (h *OpenAIHandler) isReasoningModel(modelID string) bool {
	reasoningPrefixes := []string{"o1", "o3"}

	for _, prefix := range reasoningPrefixes {
		if strings.HasPrefix(modelID, prefix) {
			return true
		}
	}

	return false
}

// getDefaultModelInfo provides default model info based on model ID
func (h *OpenAIHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default values
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       128000,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          2.5,
		OutputPrice:         10.0,
		Temperature:         &[]float64{1.0}[0],
	}

	// Model-specific configurations
	switch {
	case strings.HasPrefix(modelID, "gpt-4o"):
		info.SupportsImages = true
		info.SupportsPromptCache = true
		info.CacheWritesPrice = 1.25
		info.CacheReadsPrice = 1.25

	case strings.HasPrefix(modelID, "gpt-4"):
		info.SupportsImages = strings.Contains(modelID, "vision") || strings.Contains(modelID, "4o")
		info.InputPrice = 30.0
		info.OutputPrice = 60.0

	case strings.HasPrefix(modelID, "o1"):
		info.IsR1FormatRequired = true
		info.SupportsImages = false
		info.Temperature = nil // o1 models don't support temperature
		info.InputPrice = 15.0
		info.OutputPrice = 60.0
		if strings.Contains(modelID, "mini") {
			info.InputPrice = 3.0
			info.OutputPrice = 12.0
		}

	case strings.HasPrefix(modelID, "o3"):
		info.IsR1FormatRequired = true
		info.SupportsImages = false
		info.Temperature = nil // o3 models don't support temperature
		info.InputPrice = 60.0
		info.OutputPrice = 240.0
		if strings.Contains(modelID, "mini") {
			info.InputPrice = 1.1
			info.OutputPrice = 4.4
		}

	case strings.HasPrefix(modelID, "gpt-3.5"):
		info.MaxTokens = 4096
		info.ContextWindow = 16385
		info.InputPrice = 0.5
		info.OutputPrice = 1.5
	}

	return info
}

// convertToLLMModelInfo converts canonical model to LLM model info
func (h *OpenAIHandler) convertToLLMModelInfo(canonicalModel *models.CanonicalModel) llm.ModelInfo {
	return llm.ModelInfo{
		MaxTokens:           canonicalModel.Limits.MaxTokens,
		ContextWindow:       canonicalModel.Limits.ContextWindow,
		SupportsImages:      canonicalModel.Capabilities.SupportsImages,
		SupportsPromptCache: canonicalModel.Capabilities.SupportsPromptCache,
		InputPrice:          canonicalModel.Pricing.InputPrice,
		OutputPrice:         canonicalModel.Pricing.OutputPrice,
		CacheWritesPrice:    canonicalModel.Pricing.CacheWritesPrice,
		CacheReadsPrice:     canonicalModel.Pricing.CacheReadsPrice,
		Description:         fmt.Sprintf("%s - %s", canonicalModel.Name, canonicalModel.Family),
		Temperature:         &canonicalModel.Limits.DefaultTemperature,
	}
}

// streamRequest makes a streaming request to the OpenAI API
func (h *OpenAIHandler) streamRequest(ctx context.Context, request OpenAIRequest) (llm.ApiStream, error) {
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

// processStream processes the streaming response from OpenAI
func (h *OpenAIHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
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
		var streamEvent OpenAIStreamEvent
		if err := json.Unmarshal([]byte(event.Data), &streamEvent); err != nil {
			continue // Skip malformed events
		}

		// Process choices
		for _, choice := range streamEvent.Choices {
			// Handle content delta
			if choice.Delta.Content != "" {
				streamChan <- llm.ApiStreamTextChunk{Text: choice.Delta.Content}
			}

			// Handle reasoning delta (for o1/o3 models)
			if choice.Delta.Reasoning != "" {
				streamChan <- llm.ApiStreamReasoningChunk{Reasoning: choice.Delta.Reasoning}
			}
		}

		// Handle usage information
		if streamEvent.Usage != nil {
			usage := llm.ApiStreamUsageChunk{
				InputTokens:  streamEvent.Usage.PromptTokens,
				OutputTokens: streamEvent.Usage.CompletionTokens,
			}

			// Add cache token information if available
			if streamEvent.Usage.PromptTokensDetails != nil && streamEvent.Usage.PromptTokensDetails.CachedTokens > 0 {
				cacheReads := streamEvent.Usage.PromptTokensDetails.CachedTokens
				usage.CacheReadTokens = &cacheReads
			}

			// Add reasoning token information if available
			if streamEvent.Usage.CompletionTokensDetails != nil && streamEvent.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
				reasoningTokens := streamEvent.Usage.CompletionTokensDetails.ReasoningTokens
				usage.ThoughtsTokenCount = &reasoningTokens
			}

			streamChan <- usage
		}
	}
}
