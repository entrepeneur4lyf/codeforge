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

// OllamaHandler implements the ApiHandler interface for Ollama's local API
type OllamaHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
}

// OllamaRequest represents a request to Ollama's API (OpenAI-compatible)
type OllamaRequest struct {
	Model    string                    `json:"model"`
	Messages []transform.OpenAIMessage `json:"messages"`
	Stream   bool                      `json:"stream"`
	Options  *OllamaOptions            `json:"options,omitempty"`
}

// OllamaOptions represents Ollama-specific options
type OllamaOptions struct {
	Temperature *float64 `json:"temperature,omitempty"`
	NumPredict  *int     `json:"num_predict,omitempty"` // max_tokens equivalent
	TopK        *int     `json:"top_k,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
}

// OllamaStreamEvent represents a streaming event from Ollama
type OllamaStreamEvent struct {
	Model     string         `json:"model"`
	CreatedAt string         `json:"created_at"`
	Message   *OllamaMessage `json:"message,omitempty"`
	Done      bool           `json:"done"`

	// Usage information (when done=true)
	TotalDuration      int64 `json:"total_duration,omitempty"`
	LoadDuration       int64 `json:"load_duration,omitempty"`
	PromptEvalCount    int   `json:"prompt_eval_count,omitempty"`
	PromptEvalDuration int64 `json:"prompt_eval_duration,omitempty"`
	EvalCount          int   `json:"eval_count,omitempty"`
	EvalDuration       int64 `json:"eval_duration,omitempty"`
}

// OllamaMessage represents a message in Ollama format
type OllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// NewOllamaHandler creates a new Ollama handler
func NewOllamaHandler(options llm.ApiHandlerOptions) *OllamaHandler {
	baseURL := "http://localhost:11434"
	// For now, use default localhost. In the future, this could be configurable

	// Configure timeout (Ollama can be slower for large models)
	timeout := 120 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &OllamaHandler{
		options: options,
		client:  &http.Client{Timeout: timeout},
		baseURL: baseURL,
	}
}

// CreateMessage implements the ApiHandler interface
func (h *OllamaHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format
	openAIMessages, err := convertToOpenAIMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare Ollama options
	options := &OllamaOptions{}

	// Set temperature if specified
	if model.Info.Temperature != nil {
		options.Temperature = model.Info.Temperature
	}

	// Set max tokens if specified
	if model.Info.MaxTokens > 0 {
		options.NumPredict = &model.Info.MaxTokens
	}

	// Prepare request
	request := OllamaRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		Options:  options,
	}

	return h.streamRequest(ctx, request)
}

// GetModel implements the ApiHandler interface
func (h *OllamaHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *OllamaHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Ollama provides usage in the final stream event
	return nil, nil
}

// streamRequest handles the streaming request to Ollama
func (h *OllamaHandler) streamRequest(ctx context.Context, request OllamaRequest) (llm.ApiStream, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.baseURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

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
func (h *OllamaHandler) processStreamResponse(resp *http.Response, streamChan chan<- llm.ApiStreamChunk) {
	defer resp.Body.Close()
	defer close(streamChan)

	decoder := json.NewDecoder(resp.Body)

	for {
		var streamEvent OllamaStreamEvent
		if err := decoder.Decode(&streamEvent); err != nil {
			if err == io.EOF {
				break
			}
			continue // Skip malformed events
		}

		// Process message content
		if streamEvent.Message != nil && streamEvent.Message.Content != "" {
			streamChan <- llm.ApiStreamTextChunk{Text: streamEvent.Message.Content}
		}

		// Process completion
		if streamEvent.Done {
			// Send usage information if available
			if streamEvent.PromptEvalCount > 0 || streamEvent.EvalCount > 0 {
				streamChan <- llm.ApiStreamUsageChunk{
					InputTokens:  streamEvent.PromptEvalCount,
					OutputTokens: streamEvent.EvalCount,
				}
			}
			break
		}
	}
}

// getDefaultModelInfo provides default model info based on model ID
func (h *OllamaHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       4096,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.0, // Local models are free
		OutputPrice:         0.0, // Local models are free
		Temperature:         &[]float64{0.8}[0],
		Description:         "Local model via Ollama",
	}

	// Model-specific configurations based on common Ollama models
	switch {
	case strings.Contains(modelID, "llama3"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192
		info.Description = "Llama 3 - Meta's latest language model"
	case strings.Contains(modelID, "llama2"):
		info.ContextWindow = 4096
		info.MaxTokens = 4096
		info.Description = "Llama 2 - Meta's language model"
	case strings.Contains(modelID, "codellama"):
		info.ContextWindow = 16384
		info.MaxTokens = 8192
		info.Description = "Code Llama - Specialized for code generation"
	case strings.Contains(modelID, "mistral"):
		info.ContextWindow = 32768
		info.MaxTokens = 4096
		info.Description = "Mistral - Efficient language model"
	case strings.Contains(modelID, "mixtral"):
		info.ContextWindow = 32768
		info.MaxTokens = 4096
		info.Description = "Mixtral - Mixture of experts model"
	case strings.Contains(modelID, "gemma"):
		info.ContextWindow = 8192
		info.MaxTokens = 4096
		info.Description = "Gemma - Google's open model"
	case strings.Contains(modelID, "qwen"):
		info.ContextWindow = 32768
		info.MaxTokens = 8192
		info.Description = "Qwen - Alibaba's language model"
	case strings.Contains(modelID, "deepseek"):
		info.ContextWindow = 32768
		info.MaxTokens = 8192
		info.Description = "DeepSeek - Excellent for coding tasks"
	case strings.Contains(modelID, "phi"):
		info.ContextWindow = 131072
		info.MaxTokens = 4096
		info.Description = "Phi - Microsoft's small language model"
	case strings.Contains(modelID, "llava"):
		info.ContextWindow = 4096
		info.MaxTokens = 4096
		info.SupportsImages = true
		info.Description = "LLaVA - Large Language and Vision Assistant"
	}

	return info
}
