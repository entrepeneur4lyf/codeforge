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

// GitHubHandler implements the ApiHandler interface for GitHub Models
// Based on GitHub Models API documentation with OpenAI-compatible format
type GitHubHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
	orgMode bool
}

// GitHubRequest represents the request to GitHub Models API
type GitHubRequest struct {
	Model            string                    `json:"model"`
	Messages         []transform.OpenAIMessage `json:"messages"`
	MaxTokens        *int                      `json:"max_tokens,omitempty"`
	Temperature      *float64                  `json:"temperature,omitempty"`
	Stream           bool                      `json:"stream"`
	StreamOptions    *GitHubStreamOptions      `json:"stream_options,omitempty"`
	Tools            []GitHubTool              `json:"tools,omitempty"`
	ToolChoice       interface{}               `json:"tool_choice,omitempty"`
	FrequencyPenalty *float64                  `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64                  `json:"presence_penalty,omitempty"`
	TopP             *float64                  `json:"top_p,omitempty"`
	Stop             []string                  `json:"stop,omitempty"`
	Seed             *int                      `json:"seed,omitempty"`
}

// GitHubStreamOptions configures streaming behavior
type GitHubStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// GitHubTool represents a tool definition
type GitHubTool struct {
	Type     string            `json:"type"`
	Function GitHubFunctionDef `json:"function"`
}

// GitHubFunctionDef represents a function definition
type GitHubFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// GitHubStreamEvent represents a streaming event from GitHub Models
type GitHubStreamEvent struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []GitHubStreamChoice `json:"choices"`
	Usage   *GitHubUsage         `json:"usage,omitempty"`
}

// GitHubStreamChoice represents a choice in the stream
type GitHubStreamChoice struct {
	Index        int               `json:"index"`
	Delta        GitHubStreamDelta `json:"delta"`
	FinishReason *string           `json:"finish_reason"`
}

// GitHubStreamDelta represents delta content
type GitHubStreamDelta struct {
	Role      string                     `json:"role,omitempty"`
	Content   string                     `json:"content,omitempty"`
	ToolCalls []transform.OpenAIToolCall `json:"tool_calls,omitempty"`
}

// GitHubUsage represents token usage
type GitHubUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// NewGitHubHandler creates a new GitHub Models handler
func NewGitHubHandler(options llm.ApiHandlerOptions) *GitHubHandler {
	baseURL := "https://models.github.ai"
	orgMode := options.GitHubOrg != ""

	// Configure timeout
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &GitHubHandler{
		options: options,
		client:  &http.Client{Timeout: timeout},
		baseURL: baseURL,
		orgMode: orgMode,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *GitHubHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format (GitHub Models uses OpenAI-compatible format)
	openAIMessages, err := h.convertMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := GitHubRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &GitHubStreamOptions{
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
func (h *GitHubHandler) GetModel() llm.ModelResponse {
	// Try to get model from registry first
	registry := models.NewModelRegistry()
	if canonicalModel, exists := registry.GetModelByProvider(models.ProviderID("github"), h.options.ModelID); exists {
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
func (h *GitHubHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// GitHub Models includes usage in the stream, so this is not needed
	return nil, nil
}

// convertMessages converts LLM messages to OpenAI format for GitHub Models
func (h *GitHubHandler) convertMessages(systemPrompt string, messages []llm.Message) ([]transform.OpenAIMessage, error) {
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
			Content: convertContentBlocks(msg.Content),
		}
	}

	convertedMessages, err := transform.ConvertToOpenAIMessages(transformMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	openAIMessages = append(openAIMessages, convertedMessages...)
	return openAIMessages, nil
}

// convertContentBlocks converts llm.ContentBlock to transform.ContentBlock
func convertContentBlocks(blocks []llm.ContentBlock) []transform.ContentBlock {
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
func (h *GitHubHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default values
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       128000,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.0, // GitHub Models is free for developers
		OutputPrice:         0.0, // GitHub Models is free for developers
		Temperature:         &[]float64{1.0}[0],
	}

	// Model-specific configurations based on GitHub Models catalog
	switch {
	case strings.Contains(modelID, "gpt-4o"):
		info.SupportsImages = true
		info.ContextWindow = 128000
		info.MaxTokens = 4096

	case strings.Contains(modelID, "gpt-4"):
		info.SupportsImages = strings.Contains(modelID, "vision")
		info.ContextWindow = 128000
		info.MaxTokens = 4096

	case strings.Contains(modelID, "claude-3"):
		info.SupportsImages = true
		info.ContextWindow = 200000
		info.MaxTokens = 8192

	case strings.Contains(modelID, "llama"):
		info.ContextWindow = 128000
		info.MaxTokens = 4096

	case strings.Contains(modelID, "phi"):
		info.ContextWindow = 128000
		info.MaxTokens = 4096

	case strings.Contains(modelID, "mistral"):
		info.ContextWindow = 128000
		info.MaxTokens = 4096
	}

	return info
}

// convertToLLMModelInfo converts canonical model to LLM model info
func (h *GitHubHandler) convertToLLMModelInfo(canonicalModel *models.CanonicalModel) llm.ModelInfo {
	return llm.ModelInfo{
		MaxTokens:           canonicalModel.Limits.MaxTokens,
		ContextWindow:       canonicalModel.Limits.ContextWindow,
		SupportsImages:      canonicalModel.Capabilities.SupportsImages,
		SupportsPromptCache: canonicalModel.Capabilities.SupportsPromptCache,
		InputPrice:          0.0, // GitHub Models is free
		OutputPrice:         0.0, // GitHub Models is free
		CacheWritesPrice:    0.0,
		CacheReadsPrice:     0.0,
		Description:         fmt.Sprintf("%s - %s (GitHub Models)", canonicalModel.Name, canonicalModel.Family),
		Temperature:         &canonicalModel.Limits.DefaultTemperature,
	}
}

// streamRequest makes a streaming request to the GitHub Models API
func (h *GitHubHandler) streamRequest(ctx context.Context, request GitHubRequest) (llm.ApiStream, error) {
	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL
	var url string
	if h.orgMode {
		url = fmt.Sprintf("%s/orgs/%s/inference/chat/completions", h.baseURL, h.options.GitHubOrg)
	} else {
		url = fmt.Sprintf("%s/inference/chat/completions", h.baseURL)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+h.options.APIKey)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

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

// processStream processes the streaming response from GitHub Models
func (h *GitHubHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
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
		var streamEvent GitHubStreamEvent
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

		// Handle usage information
		if streamEvent.Usage != nil {
			usage := llm.ApiStreamUsageChunk{
				InputTokens:  streamEvent.Usage.PromptTokens,
				OutputTokens: streamEvent.Usage.CompletionTokens,
			}

			streamChan <- usage
		}
	}
}
