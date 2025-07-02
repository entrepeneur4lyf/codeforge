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
	"github.com/entrepeneur4lyf/codeforge/internal/models"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/transform"
)

// GroqHandler implements the ApiHandler interface for Groq's ultra-fast inference
// Based on Groq's OpenAI-compatible API with optimizations for speed
type GroqHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// GroqRequest represents the request to Groq API
type GroqRequest struct {
	Model            string                    `json:"model"`
	Messages         []transform.OpenAIMessage `json:"messages"`
	MaxTokens        *int                      `json:"max_tokens,omitempty"`
	Temperature      *float64                  `json:"temperature,omitempty"`
	Stream           bool                      `json:"stream"`
	StreamOptions    *GroqStreamOptions        `json:"stream_options,omitempty"`
	Tools            []GroqTool                `json:"tools,omitempty"`
	ToolChoice       interface{}               `json:"tool_choice,omitempty"`
	FrequencyPenalty *float64                  `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64                  `json:"presence_penalty,omitempty"`
	TopP             *float64                  `json:"top_p,omitempty"`
	Stop             []string                  `json:"stop,omitempty"`
	Seed             *int                      `json:"seed,omitempty"`
}

// GroqStreamOptions configures streaming behavior
type GroqStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// GroqTool represents a tool definition
type GroqTool struct {
	Type     string          `json:"type"`
	Function GroqFunctionDef `json:"function"`
}

// GroqFunctionDef represents a function definition
type GroqFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// GroqStreamEvent represents a streaming event from Groq
type GroqStreamEvent struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []GroqStreamChoice `json:"choices"`
	Usage   *GroqUsage         `json:"usage,omitempty"`
}

// GroqStreamChoice represents a choice in the stream
type GroqStreamChoice struct {
	Index        int             `json:"index"`
	Delta        GroqStreamDelta `json:"delta"`
	FinishReason *string         `json:"finish_reason"`
}

// GroqStreamDelta represents delta content
type GroqStreamDelta struct {
	Role      string                     `json:"role,omitempty"`
	Content   string                     `json:"content,omitempty"`
	ToolCalls []transform.OpenAIToolCall `json:"tool_calls,omitempty"`
}

// GroqUsage represents token usage
type GroqUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	PromptTime       float64 `json:"prompt_time,omitempty"`     // Groq-specific: time to process prompt
	CompletionTime   float64 `json:"completion_time,omitempty"` // Groq-specific: time to generate completion
	TotalTime        float64 `json:"total_time,omitempty"`      // Groq-specific: total request time
}

// NewGroqHandler creates a new Groq handler
func NewGroqHandler(options llm.ApiHandlerOptions) *GroqHandler {
	baseURL := "https://api.groq.com/openai/v1"

	// Configure timeout - Groq is ultra-fast, so shorter timeout is appropriate
	timeout := 30 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &GroqHandler{
		options: options,
		client:  &http.Client{Timeout: timeout},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *GroqHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format (Groq uses OpenAI-compatible format)
	openAIMessages, err := h.convertMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := GroqRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &GroqStreamOptions{
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
func (h *GroqHandler) GetModel() llm.ModelResponse {
	// Try to get model from registry first
	registry := models.NewModelRegistry()
	if canonicalModel, exists := registry.GetModelByProvider(models.ProviderID("groq"), h.options.ModelID); exists {
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
func (h *GroqHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Groq includes usage in the stream, so this is not needed
	return nil, nil
}

// convertMessages converts LLM messages to OpenAI format for Groq
func (h *GroqHandler) convertMessages(systemPrompt string, messages []llm.Message) ([]transform.OpenAIMessage, error) {
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
			Content: convertContentBlocksGroq(msg.Content),
		}
	}

	convertedMessages, err := transform.ConvertToOpenAIMessages(transformMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	openAIMessages = append(openAIMessages, convertedMessages...)
	return openAIMessages, nil
}

// convertContentBlocksGroq converts llm.ContentBlock to transform.ContentBlock
func convertContentBlocksGroq(blocks []llm.ContentBlock) []transform.ContentBlock {
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

// getDefaultModelInfo provides default model info based on model ID
func (h *GroqHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default values for Groq models
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       32768,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.1, // Groq pricing per million tokens
		OutputPrice:         0.1, // Groq pricing per million tokens
		Temperature:         &[]float64{1.0}[0],
		Description:         "Ultra-fast inference via Groq LPU",
	}

	// Model-specific configurations based on Groq's model catalog
	switch {
	case strings.Contains(modelID, "llama-3.1-405b"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.InputPrice = 0.59
		info.OutputPrice = 0.79

	case strings.Contains(modelID, "llama-3.1-70b"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.InputPrice = 0.59
		info.OutputPrice = 0.79

	case strings.Contains(modelID, "llama-3.1-8b"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.InputPrice = 0.05
		info.OutputPrice = 0.08

	case strings.Contains(modelID, "llama-3.2-90b"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.SupportsImages = true
		info.InputPrice = 0.59
		info.OutputPrice = 0.79

	case strings.Contains(modelID, "llama-3.2-11b"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.SupportsImages = true
		info.InputPrice = 0.18
		info.OutputPrice = 0.18

	case strings.Contains(modelID, "mixtral-8x7b"):
		info.ContextWindow = 32768
		info.MaxTokens = 4096
		info.InputPrice = 0.24
		info.OutputPrice = 0.24

	case strings.Contains(modelID, "gemma2-9b"):
		info.ContextWindow = 8192
		info.MaxTokens = 4096
		info.InputPrice = 0.20
		info.OutputPrice = 0.20

	case strings.Contains(modelID, "gemma-7b"):
		info.ContextWindow = 8192
		info.MaxTokens = 4096
		info.InputPrice = 0.10
		info.OutputPrice = 0.10
	}

	return info
}

// convertToLLMModelInfo converts canonical model to LLM model info
func (h *GroqHandler) convertToLLMModelInfo(canonicalModel *models.CanonicalModel) llm.ModelInfo {
	return llm.ModelInfo{
		MaxTokens:           canonicalModel.Limits.MaxTokens,
		ContextWindow:       canonicalModel.Limits.ContextWindow,
		SupportsImages:      canonicalModel.Capabilities.SupportsImages,
		SupportsPromptCache: canonicalModel.Capabilities.SupportsPromptCache,
		InputPrice:          canonicalModel.Pricing.InputPrice,
		OutputPrice:         canonicalModel.Pricing.OutputPrice,
		CacheWritesPrice:    canonicalModel.Pricing.CacheWritesPrice,
		CacheReadsPrice:     canonicalModel.Pricing.CacheReadsPrice,
		Description:         fmt.Sprintf("%s - %s (Groq LPU)", canonicalModel.Name, canonicalModel.Family),
		Temperature:         &canonicalModel.Limits.DefaultTemperature,
	}
}

// streamRequest makes a streaming request to the Groq API
func (h *GroqHandler) streamRequest(ctx context.Context, request GroqRequest) (llm.ApiStream, error) {
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

// processStream processes the streaming response from Groq
func (h *GroqHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
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
		var streamEvent GroqStreamEvent
		if err := json.Unmarshal([]byte(event.Data), &streamEvent); err != nil {
			continue // Skip malformed events
		}

		// Process choices
		for _, choice := range streamEvent.Choices {
			// Handle content delta
			if choice.Delta.Content != "" {
				streamChan <- llm.ApiStreamTextChunk{Text: choice.Delta.Content}
			}
		}

		// Handle usage information with Groq-specific timing data
		if streamEvent.Usage != nil {
			usage := llm.ApiStreamUsageChunk{
				InputTokens:  streamEvent.Usage.PromptTokens,
				OutputTokens: streamEvent.Usage.CompletionTokens,
			}

			// Calculate cost based on Groq pricing
			if h.options.ModelInfo != nil {
				inputCost := float64(streamEvent.Usage.PromptTokens) * h.options.ModelInfo.InputPrice / 1000000
				outputCost := float64(streamEvent.Usage.CompletionTokens) * h.options.ModelInfo.OutputPrice / 1000000
				totalCost := inputCost + outputCost
				usage.TotalCost = &totalCost
			}

			streamChan <- usage
		}
	}
}

// calculateGroqCost calculates the cost for a Groq API call
func (h *GroqHandler) calculateGroqCost(info llm.ModelInfo, inputTokens, outputTokens int) float64 {
	inputCost := float64(inputTokens) * info.InputPrice / 1000000
	outputCost := float64(outputTokens) * info.OutputPrice / 1000000
	return inputCost + outputCost
}
