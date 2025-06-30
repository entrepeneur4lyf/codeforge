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
)

// VertexHandler implements the ApiHandler interface for Google Cloud Vertex AI
// Vertex AI uses Gemini models with Google Cloud authentication
type VertexHandler struct {
	options   llm.ApiHandlerOptions
	client    *http.Client
	baseURL   string
	projectID string
	location  string
}

// VertexRequest represents a request to Vertex AI's API
type VertexRequest struct {
	Contents         []VertexContent         `json:"contents"`
	GenerationConfig *VertexGenerationConfig `json:"generationConfig,omitempty"`
	SafetySettings   []VertexSafetySetting   `json:"safetySettings,omitempty"`
}

// VertexContent represents content in Vertex AI format
type VertexContent struct {
	Role  string       `json:"role"`
	Parts []VertexPart `json:"parts"`
}

// VertexPart represents a part of content
type VertexPart struct {
	Text string `json:"text"`
}

// VertexGenerationConfig represents generation configuration
type VertexGenerationConfig struct {
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	Temperature     *float64 `json:"temperature,omitempty"`
	TopP            *float64 `json:"topP,omitempty"`
	TopK            *int     `json:"topK,omitempty"`
}

// VertexSafetySetting represents safety settings
type VertexSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// VertexStreamResponse represents a streaming response from Vertex AI
type VertexStreamResponse struct {
	Candidates    []VertexCandidate    `json:"candidates,omitempty"`
	UsageMetadata *VertexUsageMetadata `json:"usageMetadata,omitempty"`
}

// VertexCandidate represents a candidate response
type VertexCandidate struct {
	Content      *VertexContent `json:"content,omitempty"`
	FinishReason string         `json:"finishReason,omitempty"`
	Index        int            `json:"index"`
}

// VertexUsageMetadata represents usage information
type VertexUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

// NewVertexHandler creates a new Vertex AI handler
func NewVertexHandler(options llm.ApiHandlerOptions) *VertexHandler {
	projectID := options.VertexProjectID
	if projectID == "" {
		projectID = "default-project" // Should be configured properly
	}

	location := options.VertexRegion
	if location == "" {
		location = "us-central1" // Default location
	}

	baseURL := options.GeminiBaseURL
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models",
			location, projectID, location)
	}

	// Configure timeout based on request timeout option
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &VertexHandler{
		options:   options,
		projectID: projectID,
		location:  location,
		client: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *VertexHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to Vertex format
	contents, err := h.convertMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := VertexRequest{
		Contents: contents,
		GenerationConfig: &VertexGenerationConfig{
			MaxOutputTokens: model.Info.MaxTokens,
		},
	}

	// Set temperature if specified
	if model.Info.Temperature != nil {
		request.GenerationConfig.Temperature = model.Info.Temperature
	}

	// Add safety settings (permissive for coding tasks)
	request.SafetySettings = []VertexSafetySetting{
		{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_ONLY_HIGH"},
		{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_ONLY_HIGH"},
	}

	return h.streamRequest(ctx, request, model.ID)
}

// GetModel implements the ApiHandler interface
func (h *VertexHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *VertexHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Vertex AI provides usage in the stream
	return nil, nil
}

// getDefaultModelInfo returns default model information for Vertex AI models
func (h *VertexHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration for Vertex AI models
	info := llm.ModelInfo{
		MaxTokens:           8192,
		ContextWindow:       1000000, // Gemini 2.5 Flash has 1M context
		SupportsImages:      true,
		SupportsPromptCache: false,
		InputPrice:          0.075, // $0.075 per 1M tokens for Gemini 2.5 Flash
		OutputPrice:         0.3,   // $0.30 per 1M tokens for Gemini 2.5 Flash
		Description:         fmt.Sprintf("Vertex AI model: %s", modelID),
	}

	// Model-specific configurations
	modelLower := strings.ToLower(modelID)

	// Gemini 2.5 Flash
	if strings.Contains(modelLower, "gemini-2.5-flash") {
		info.ContextWindow = 1000000
		info.MaxTokens = 8192
		info.InputPrice = 0.075
		info.OutputPrice = 0.3
	}

	// Gemini 1.5 Pro
	if strings.Contains(modelLower, "gemini-1.5-pro") {
		info.ContextWindow = 2000000
		info.MaxTokens = 8192
		info.InputPrice = 1.25
		info.OutputPrice = 5.0
	}

	// Gemini 1.5 Flash
	if strings.Contains(modelLower, "gemini-1.5-flash") {
		info.ContextWindow = 1000000
		info.MaxTokens = 8192
		info.InputPrice = 0.075
		info.OutputPrice = 0.3
	}

	return info
}

// convertMessages converts LLM messages to Vertex format
func (h *VertexHandler) convertMessages(systemPrompt string, messages []llm.Message) ([]VertexContent, error) {
	var contents []VertexContent

	// Add system prompt as first user message if provided
	if systemPrompt != "" {
		contents = append(contents, VertexContent{
			Role: "user",
			Parts: []VertexPart{
				{Text: systemPrompt},
			},
		})
	}

	// Convert messages
	for _, msg := range messages {
		role := msg.Role
		if role == "assistant" {
			role = "model" // Vertex uses "model" instead of "assistant"
		}

		// Extract text content
		var text string
		for _, content := range msg.Content {
			if textBlock, ok := content.(llm.TextBlock); ok {
				text += textBlock.Text
			}
		}

		if text != "" {
			contents = append(contents, VertexContent{
				Role: role,
				Parts: []VertexPart{
					{Text: text},
				},
			})
		}
	}

	return contents, nil
}

// streamRequest makes a streaming request to the Vertex AI API
func (h *VertexHandler) streamRequest(ctx context.Context, request VertexRequest, modelID string) (llm.ApiStream, error) {
	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/%s:streamGenerateContent", h.baseURL, modelID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Use Google Cloud authentication (should be configured via environment)
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

// processStream processes the streaming response from Vertex AI
func (h *VertexHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
	decoder := json.NewDecoder(reader)

	for {
		var response VertexStreamResponse
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
					if part.Text != "" {
						streamChan <- llm.ApiStreamTextChunk{Text: part.Text}
					}
				}
			}
		}

		// Process usage information
		if response.UsageMetadata != nil {
			streamChan <- llm.ApiStreamUsageChunk{
				InputTokens:  response.UsageMetadata.PromptTokenCount,
				OutputTokens: response.UsageMetadata.CandidatesTokenCount,
			}
		}
	}
}
