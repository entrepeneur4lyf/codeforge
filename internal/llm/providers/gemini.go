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

// GeminiHandler implements the ApiHandler interface for Google's Gemini models
// Based on Cline's GeminiHandler with full feature parity
type GeminiHandler struct {
	options  llm.ApiHandlerOptions
	client   *http.Client
	baseURL  string
	isVertex bool
}

// GeminiRequest represents the request to Gemini API
type GeminiRequest struct {
	Contents          []GeminiContent         `json:"contents"`
	SystemInstruction *GeminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  *GeminiGenerationConfig `json:"generationConfig,omitempty"`
	ThinkingConfig    *GeminiThinkingConfig   `json:"thinkingConfig,omitempty"`
}

// GeminiContent represents content in Gemini format
type GeminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []GeminiPart `json:"parts"`
}

// GeminiPart represents different types of content parts
type GeminiPart struct {
	Text       string             `json:"text,omitempty"`
	InlineData *GeminiInlineData  `json:"inlineData,omitempty"`
	Thought    *GeminiThoughtPart `json:"thought,omitempty"`
}

// GeminiInlineData represents inline image data
type GeminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// GeminiThoughtPart represents thinking content
type GeminiThoughtPart struct {
	Text string `json:"text"`
}

// GeminiGenerationConfig represents generation configuration
type GeminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
}

// GeminiThinkingConfig represents thinking configuration
type GeminiThinkingConfig struct {
	ThinkingBudget  int  `json:"thinkingBudget"`
	IncludeThoughts bool `json:"includeThoughts"`
}

// GeminiStreamResponse represents a streaming response chunk
type GeminiStreamResponse struct {
	Candidates    []GeminiCandidate    `json:"candidates,omitempty"`
	UsageMetadata *GeminiUsageMetadata `json:"usageMetadata,omitempty"`
}

// GeminiCandidate represents a response candidate
type GeminiCandidate struct {
	Content      *GeminiContent `json:"content,omitempty"`
	FinishReason string         `json:"finishReason,omitempty"`
}

// GeminiUsageMetadata represents token usage information
type GeminiUsageMetadata struct {
	PromptTokenCount        int `json:"promptTokenCount"`
	CandidatesTokenCount    int `json:"candidatesTokenCount"`
	TotalTokenCount         int `json:"totalTokenCount"`
	CachedContentTokenCount int `json:"cachedContentTokenCount,omitempty"`
	ThoughtsTokenCount      int `json:"thoughtsTokenCount,omitempty"`
}

// NewGeminiHandler creates a new Gemini handler
func NewGeminiHandler(options llm.ApiHandlerOptions) *GeminiHandler {
	baseURL := options.GeminiBaseURL
	isVertex := options.VertexProjectID != ""

	if baseURL == "" {
		if isVertex {
			region := options.VertexRegion
			if region == "" {
				region = "us-central1"
			}
			baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google",
				region, options.VertexProjectID, region)
		} else {
			baseURL = "https://generativelanguage.googleapis.com/v1beta"
		}
	}

	// Configure timeout
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &GeminiHandler{
		options:  options,
		client:   &http.Client{Timeout: timeout},
		baseURL:  baseURL,
		isVertex: isVertex,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *GeminiHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to Gemini format
	geminiContents, err := h.convertMessages(messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := GeminiRequest{
		Contents: geminiContents,
		GenerationConfig: &GeminiGenerationConfig{
			Temperature: &[]float64{0.0}[0],
		},
	}

	// Add system instruction if provided
	if systemPrompt != "" {
		request.SystemInstruction = &GeminiContent{
			Parts: []GeminiPart{{Text: systemPrompt}},
		}
	}

	// Set max output tokens if specified
	if model.Info.MaxTokens > 0 {
		request.GenerationConfig.MaxOutputTokens = &model.Info.MaxTokens
	}

	// Add thinking configuration if supported and requested
	if h.options.ThinkingBudgetTokens > 0 && h.supportsThinking(model.ID) {
		request.ThinkingConfig = &GeminiThinkingConfig{
			ThinkingBudget:  h.options.ThinkingBudgetTokens,
			IncludeThoughts: true,
		}
	}

	return h.streamRequest(ctx, request, model.ID)
}

// GetModel implements the ApiHandler interface
func (h *GeminiHandler) GetModel() llm.ModelResponse {
	// Try to get model from registry first
	registry := models.NewModelRegistry()
	providerID := models.ProviderGemini
	if h.isVertex {
		providerID = models.ProviderVertex
	}

	if canonicalModel, exists := registry.GetModelByProvider(providerID, h.options.ModelID); exists {
		return llm.ModelResponse{
			ID:   h.options.ModelID,
			Info: h.convertToLLMModelInfo(canonicalModel),
		}
	}

	// Fallback to default model info
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *GeminiHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Gemini includes usage in the stream, so this is not needed
	return nil, nil
}

// convertMessages converts LLM messages to Gemini format
func (h *GeminiHandler) convertMessages(messages []llm.Message) ([]GeminiContent, error) {
	var geminiContents []GeminiContent

	for _, msg := range messages {
		if msg.Role == "system" {
			// System messages are handled separately in Gemini API
			continue
		}

		// Convert role
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}

		// Convert content blocks
		var parts []GeminiPart
		for _, block := range msg.Content {
			switch b := block.(type) {
			case llm.TextBlock:
				parts = append(parts, GeminiPart{Text: b.Text})
			case llm.ImageBlock:
				parts = append(parts, GeminiPart{
					InlineData: &GeminiInlineData{
						MimeType: b.Source.MediaType,
						Data:     b.Source.Data,
					},
				})
			}
		}

		geminiContents = append(geminiContents, GeminiContent{
			Role:  role,
			Parts: parts,
		})
	}

	return geminiContents, nil
}

// supportsThinking checks if a model supports thinking
func (h *GeminiHandler) supportsThinking(modelID string) bool {
	thinkingModels := []string{
		"gemini-2.5-flash-thinking",
		"gemini-2.0-flash-thinking",
	}

	for _, model := range thinkingModels {
		if strings.Contains(modelID, model) {
			return true
		}
	}

	return false
}

// getDefaultModelInfo provides default model info based on model ID
func (h *GeminiHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default values
	info := llm.ModelInfo{
		MaxTokens:           8192,
		ContextWindow:       1000000,
		SupportsImages:      true,
		SupportsPromptCache: true,
		InputPrice:          0.075,
		OutputPrice:         0.3,
		CacheWritesPrice:    0.09375,
		CacheReadsPrice:     0.01875,
		Temperature:         &[]float64{0.0}[0],
	}

	// Model-specific configurations
	switch {
	case strings.Contains(modelID, "2.5-flash"):
		info.InputPrice = 0.075
		info.OutputPrice = 0.3
		info.CacheWritesPrice = 0.09375
		info.CacheReadsPrice = 0.01875

	case strings.Contains(modelID, "2.0-flash"):
		info.InputPrice = 0.075
		info.OutputPrice = 0.3
		info.CacheWritesPrice = 0.09375
		info.CacheReadsPrice = 0.01875

	case strings.Contains(modelID, "1.5-pro"):
		info.InputPrice = 1.25
		info.OutputPrice = 5.0
		info.CacheWritesPrice = 1.5625
		info.CacheReadsPrice = 0.3125
		info.ContextWindow = 2000000

	case strings.Contains(modelID, "1.5-flash"):
		info.InputPrice = 0.075
		info.OutputPrice = 0.3
		info.CacheWritesPrice = 0.09375
		info.CacheReadsPrice = 0.01875
	}

	return info
}

// convertToLLMModelInfo converts canonical model to LLM model info
func (h *GeminiHandler) convertToLLMModelInfo(canonicalModel *models.CanonicalModel) llm.ModelInfo {
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

// streamRequest makes a streaming request to the Gemini API
func (h *GeminiHandler) streamRequest(ctx context.Context, request GeminiRequest, modelID string) (llm.ApiStream, error) {
	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL
	var url string
	if h.isVertex {
		url = fmt.Sprintf("%s/models/%s:streamGenerateContent", h.baseURL, modelID)
	} else {
		url = fmt.Sprintf("%s/models/%s:streamGenerateContent", h.baseURL, modelID)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	if h.isVertex {
		// For Vertex AI, use OAuth token (simplified - in production would use proper OAuth)
		req.Header.Set("Authorization", "Bearer "+h.options.APIKey)
	} else {
		// For AI Studio, use API key
		req.URL.RawQuery = "key=" + h.options.APIKey
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

// processStream processes the streaming response from Gemini
func (h *GeminiHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
	decoder := json.NewDecoder(reader)

	for {
		var response GeminiStreamResponse
		if err := decoder.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			continue // Skip malformed responses
		}

		// Process candidates
		for _, candidate := range response.Candidates {
			if candidate.Content != nil {
				for _, part := range candidate.Content.Parts {
					// Handle text content
					if part.Text != "" {
						streamChan <- llm.ApiStreamTextChunk{Text: part.Text}
					}

					// Handle thinking content
					if part.Thought != nil && part.Thought.Text != "" {
						streamChan <- llm.ApiStreamReasoningChunk{Reasoning: part.Thought.Text}
					}
				}
			}
		}

		// Handle usage information
		if response.UsageMetadata != nil {
			usage := llm.ApiStreamUsageChunk{
				InputTokens:  response.UsageMetadata.PromptTokenCount,
				OutputTokens: response.UsageMetadata.CandidatesTokenCount,
			}

			// Add cache token information if available
			if response.UsageMetadata.CachedContentTokenCount > 0 {
				cacheReads := response.UsageMetadata.CachedContentTokenCount
				usage.CacheReadTokens = &cacheReads
			}

			// Add thinking token information if available
			if response.UsageMetadata.ThoughtsTokenCount > 0 {
				thoughtsTokens := response.UsageMetadata.ThoughtsTokenCount
				usage.ThoughtsTokenCount = &thoughtsTokens
			}

			streamChan <- usage
		}
	}
}
