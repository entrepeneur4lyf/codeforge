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
)

// AnthropicHandler implements the ApiHandler interface for Anthropic's Claude models
// Based on Cline's AnthropicHandler from anthropic.ts
type AnthropicHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// AnthropicMessage represents an Anthropic API message
type AnthropicMessage struct {
	Role    string                  `json:"role"`
	Content []AnthropicContentBlock `json:"content"`
}

// AnthropicContentBlock represents content in Anthropic format
type AnthropicContentBlock struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text,omitempty"`
	Source       *AnthropicImageSource  `json:"source,omitempty"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicImageSource represents image data in Anthropic format
type AnthropicImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// AnthropicCacheControl represents cache control settings
type AnthropicCacheControl struct {
	Type string `json:"type"`
}

// AnthropicRequest represents the request to Anthropic API
type AnthropicRequest struct {
	Model       string                 `json:"model"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float64                `json:"temperature,omitempty"`
	System      []AnthropicSystemBlock `json:"system,omitempty"`
	Messages    []AnthropicMessage     `json:"messages"`
	Stream      bool                   `json:"stream"`
	Thinking    *AnthropicThinking     `json:"thinking,omitempty"`
}

// AnthropicSystemBlock represents system prompt with cache control
type AnthropicSystemBlock struct {
	Type         string                 `json:"type"`
	Text         string                 `json:"text"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicThinking represents thinking configuration
type AnthropicThinking struct {
	Type         string `json:"type"`
	BudgetTokens int    `json:"budget_tokens"`
}

// AnthropicStreamEvent represents a streaming event from Anthropic
type AnthropicStreamEvent struct {
	Type         string                      `json:"type"`
	Message      *AnthropicMessageStart      `json:"message,omitempty"`
	Delta        *AnthropicDelta             `json:"delta,omitempty"`
	Usage        *AnthropicUsage             `json:"usage,omitempty"`
	ContentBlock *AnthropicContentBlockEvent `json:"content_block,omitempty"`
	Index        int                         `json:"index,omitempty"`
}

// AnthropicMessageStart represents the start of a message
type AnthropicMessageStart struct {
	Usage *AnthropicUsage `json:"usage,omitempty"`
}

// AnthropicDelta represents delta content
type AnthropicDelta struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
}

// AnthropicUsage represents token usage
type AnthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// AnthropicContentBlockEvent represents content block events
type AnthropicContentBlockEvent struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Thinking string `json:"thinking,omitempty"`
}

// NewAnthropicHandler creates a new Anthropic handler
func NewAnthropicHandler(options llm.ApiHandlerOptions) *AnthropicHandler {
	baseURL := options.AnthropicBaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}

	return &AnthropicHandler{
		options: options,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *AnthropicHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to Anthropic format
	anthropicMessages, err := h.convertMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := AnthropicRequest{
		Model:       model.ID,
		MaxTokens:   model.Info.MaxTokens,
		Temperature: 0,
		Messages:    anthropicMessages,
		Stream:      true,
	}

	// Add system prompt with cache control for supported models
	if systemPrompt != "" {
		systemBlock := AnthropicSystemBlock{
			Type: "text",
			Text: systemPrompt,
		}

		// Add cache control for supported models
		if model.Info.SupportsPromptCache {
			systemBlock.CacheControl = &AnthropicCacheControl{Type: "ephemeral"}
		}

		request.System = []AnthropicSystemBlock{systemBlock}
	}

	// Add thinking configuration if supported and requested
	if h.options.ThinkingBudgetTokens > 0 && h.supportsThinking(model.ID) {
		request.Thinking = &AnthropicThinking{
			Type:         "enabled",
			BudgetTokens: h.options.ThinkingBudgetTokens,
		}
		// Thinking mode doesn't support temperature
		request.Temperature = 0
	}

	// Add cache control to recent user messages for supported models
	if model.Info.SupportsPromptCache && len(request.Messages) > 0 {
		h.addCacheControlToMessages(request.Messages)
	}

	return h.streamRequest(ctx, request)
}

// GetModel implements the ApiHandler interface
func (h *AnthropicHandler) GetModel() llm.ModelResponse {
	// Try to get model from registry first
	registry := models.NewModelRegistry()
	if canonicalModel, exists := registry.GetModelByProvider(models.ProviderAnthropic, h.options.ModelID); exists {
		return llm.ModelResponse{
			ID:   h.options.ModelID,
			Info: h.convertToLLMModelInfo(canonicalModel),
		}
	}

	// Fallback to default model info with model-specific limits
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// getDefaultModelInfo returns default model information for Anthropic models
func (h *AnthropicHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for Anthropic models
	info := llm.ModelInfo{
		MaxTokens:           8192,
		ContextWindow:       200000,
		SupportsImages:      true,
		SupportsPromptCache: true,
		InputPrice:          3.0,
		OutputPrice:         15.0,
		CacheWritesPrice:    3.75,
		CacheReadsPrice:     0.3,
		Description:         fmt.Sprintf("Anthropic model: %s", modelID),
	}

	// Model-specific configurations
	modelLower := strings.ToLower(modelID)

	// Claude 3 Opus - has lower max tokens limit
	if strings.Contains(modelLower, "opus") {
		info.MaxTokens = 4096
		info.InputPrice = 15.0
		info.OutputPrice = 75.0
	}

	// Claude 3.5 Haiku - smaller model
	if strings.Contains(modelLower, "haiku") {
		info.MaxTokens = 4096
		info.InputPrice = 0.25
		info.OutputPrice = 1.25
	}

	// Claude 3.5 Sonnet - flagship model
	if strings.Contains(modelLower, "3.5") && strings.Contains(modelLower, "sonnet") {
		info.MaxTokens = 8192
		info.InputPrice = 3.0
		info.OutputPrice = 15.0
	}

	return info
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *AnthropicHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Anthropic includes usage in the stream, so this is not needed
	return nil, nil
}

// convertMessages converts LLM messages to Anthropic format
func (h *AnthropicHandler) convertMessages(messages []llm.Message) ([]AnthropicMessage, error) {
	var anthropicMessages []AnthropicMessage

	for _, msg := range messages {
		if msg.Role == "system" {
			// System messages are handled separately in Anthropic API
			continue
		}

		var content []AnthropicContentBlock
		for _, block := range msg.Content {
			switch b := block.(type) {
			case llm.TextBlock:
				content = append(content, AnthropicContentBlock{
					Type: "text",
					Text: b.Text,
				})
			case llm.ImageBlock:
				content = append(content, AnthropicContentBlock{
					Type: "image",
					Source: &AnthropicImageSource{
						Type:      b.Source.Type,
						MediaType: b.Source.MediaType,
						Data:      b.Source.Data,
					},
				})
			}
		}

		anthropicMessages = append(anthropicMessages, AnthropicMessage{
			Role:    msg.Role,
			Content: content,
		})
	}

	return anthropicMessages, nil
}

// supportsThinking checks if a model supports thinking
func (h *AnthropicHandler) supportsThinking(modelID string) bool {
	thinkingModels := []string{
		"claude-sonnet-4-20250514",
		"claude-opus-4-20250514",
		"claude-3-7-sonnet-20250219",
	}

	for _, model := range thinkingModels {
		if strings.Contains(modelID, model) {
			return true
		}
	}

	return false
}

// addCacheControlToMessages adds cache control to recent user messages
func (h *AnthropicHandler) addCacheControlToMessages(messages []AnthropicMessage) {
	// Find the last two user messages and add cache control
	userIndices := make([]int, 0)
	for i, msg := range messages {
		if msg.Role == "user" {
			userIndices = append(userIndices, i)
		}
	}

	// Add cache control to the last two user messages
	if len(userIndices) >= 1 {
		lastUserIndex := userIndices[len(userIndices)-1]
		h.addCacheControlToMessage(&messages[lastUserIndex])
	}

	if len(userIndices) >= 2 {
		secondLastUserIndex := userIndices[len(userIndices)-2]
		h.addCacheControlToMessage(&messages[secondLastUserIndex])
	}
}

// addCacheControlToMessage adds cache control to a specific message
func (h *AnthropicHandler) addCacheControlToMessage(message *AnthropicMessage) {
	if len(message.Content) > 0 {
		// Add cache control to the last content block
		lastIndex := len(message.Content) - 1
		message.Content[lastIndex].CacheControl = &AnthropicCacheControl{Type: "ephemeral"}
	}
}

// convertToLLMModelInfo converts canonical model to LLM model info
func (h *AnthropicHandler) convertToLLMModelInfo(canonicalModel *models.CanonicalModel) llm.ModelInfo {
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
	}
}

// streamRequest makes a streaming request to the Anthropic API
func (h *AnthropicHandler) streamRequest(ctx context.Context, request AnthropicRequest) (llm.ApiStream, error) {
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
	req.Header.Set("anthropic-beta", "prompt-caching-2024-07-31")

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

// processStream processes the streaming response from Anthropic
func (h *AnthropicHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
	scanner := NewSSEScanner(reader)

	for scanner.Scan() {
		event := scanner.Event()

		// Skip non-data events
		if event.Type != "data" {
			continue
		}

		// Parse the event data
		var streamEvent AnthropicStreamEvent
		if err := json.Unmarshal([]byte(event.Data), &streamEvent); err != nil {
			continue // Skip malformed events
		}

		// Process different event types
		switch streamEvent.Type {
		case "content_block_delta":
			if streamEvent.Delta != nil {
				if streamEvent.Delta.Text != "" {
					streamChan <- llm.ApiStreamTextChunk{Text: streamEvent.Delta.Text}
				}
				if streamEvent.Delta.Thinking != "" {
					streamChan <- llm.ApiStreamReasoningChunk{Reasoning: streamEvent.Delta.Thinking}
				}
			}
		case "message_delta":
			if streamEvent.Usage != nil {
				usage := llm.ApiStreamUsageChunk{
					InputTokens:  streamEvent.Usage.InputTokens,
					OutputTokens: streamEvent.Usage.OutputTokens,
				}

				// Add cache token information if available
				if streamEvent.Usage.CacheCreationInputTokens > 0 {
					cacheWrites := streamEvent.Usage.CacheCreationInputTokens
					usage.CacheWriteTokens = &cacheWrites
				}
				if streamEvent.Usage.CacheReadInputTokens > 0 {
					cacheReads := streamEvent.Usage.CacheReadInputTokens
					usage.CacheReadTokens = &cacheReads
				}

				streamChan <- usage
			}
		}
	}
}

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	Type string
	Data string
}

// SSEScanner scans Server-Sent Events from a reader
type SSEScanner struct {
	reader io.Reader
	buffer []byte
	events []SSEEvent
	index  int
}

// NewSSEScanner creates a new SSE scanner
func NewSSEScanner(reader io.Reader) *SSEScanner {
	return &SSEScanner{
		reader: reader,
		buffer: make([]byte, 4096),
		events: make([]SSEEvent, 0),
		index:  0,
	}
}

// Scan scans for the next SSE event
func (s *SSEScanner) Scan() bool {
	if s.index < len(s.events) {
		s.index++
		return true
	}

	// Read more data
	n, err := s.reader.Read(s.buffer)
	if err != nil {
		return false
	}

	// Parse SSE events from buffer
	data := string(s.buffer[:n])
	lines := strings.Split(data, "\n")

	var currentEvent SSEEvent
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "" {
			// Empty line indicates end of event
			if currentEvent.Type != "" || currentEvent.Data != "" {
				s.events = append(s.events, currentEvent)
				currentEvent = SSEEvent{}
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			currentEvent.Type = strings.TrimSpace(line[6:])
		} else if strings.HasPrefix(line, "data:") {
			currentEvent.Data = strings.TrimSpace(line[5:])
		}
	}

	// Add final event if exists
	if currentEvent.Type != "" || currentEvent.Data != "" {
		s.events = append(s.events, currentEvent)
	}

	return len(s.events) > s.index
}

// Event returns the current SSE event
func (s *SSEScanner) Event() SSEEvent {
	if s.index > 0 && s.index <= len(s.events) {
		return s.events[s.index-1]
	}
	return SSEEvent{}
}
