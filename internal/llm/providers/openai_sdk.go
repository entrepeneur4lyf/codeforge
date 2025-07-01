package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// OpenAISDKHandler implements the ApiHandler interface using the official OpenAI Go SDK
type OpenAISDKHandler struct {
	options llm.ApiHandlerOptions
	client  *openai.Client
}

// NewOpenAISDKHandler creates a new OpenAI handler using the official SDK
func NewOpenAISDKHandler(options llm.ApiHandlerOptions) *OpenAISDKHandler {
	// Create client with API key
	client := openai.NewClient(
		option.WithAPIKey(options.APIKey),
	)

	return &OpenAISDKHandler{
		options: options,
		client:  &client,
	}
}

func (h *OpenAISDKHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

func (h *OpenAISDKHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// OpenAI provides usage in the final stream event
	return nil, nil
}

// CreateMessage sends a message to OpenAI and returns a streaming response
func (h *OpenAISDKHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	// Convert messages to OpenAI format
	openaiMessages := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)

	// Add system prompt if provided
	if systemPrompt != "" {
		openaiMessages = append(openaiMessages, openai.SystemMessage(systemPrompt))
	}

	// Convert our messages to OpenAI format
	for _, msg := range messages {
		// Extract text content from message
		var textContent string
		for _, content := range msg.Content {
			if textBlock, ok := content.(llm.TextBlock); ok {
				textContent += textBlock.Text
			}
		}

		switch strings.ToLower(msg.Role) {
		case "user":
			openaiMessages = append(openaiMessages, openai.UserMessage(textContent))
		case "assistant":
			openaiMessages = append(openaiMessages, openai.AssistantMessage(textContent))
		case "system":
			openaiMessages = append(openaiMessages, openai.SystemMessage(textContent))
		default:
			openaiMessages = append(openaiMessages, openai.UserMessage(textContent))
		}
	}

	// Convert model ID to OpenAI model constant
	model := convertModelToOpenAI(h.options.ModelID)

	// Create streaming completion
	stream := h.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
		Messages: openaiMessages,
		Model:    model,
	})

	// Create output channel
	outputChan := make(chan llm.ApiStreamChunk, 100)

	// Start goroutine to process stream
	go func() {
		defer close(outputChan)

		for stream.Next() {
			evt := stream.Current()
			if len(evt.Choices) > 0 {
				content := evt.Choices[0].Delta.Content
				if content != "" {
					outputChan <- llm.ApiStreamTextChunk{
						Text: content,
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			// Log error but don't send error chunk as it's not defined in our interface
			fmt.Printf("OpenAI stream error: %v\n", err)
		}
	}()

	return outputChan, nil
}

// convertModelToOpenAI converts model ID to OpenAI model constant
func convertModelToOpenAI(modelID string) openai.ChatModel {
	// Remove provider prefix if present
	if strings.Contains(modelID, "/") {
		parts := strings.Split(modelID, "/")
		if len(parts) == 2 {
			modelID = parts[1]
		}
	}

	switch modelID {
	case "gpt-4o":
		return openai.ChatModelGPT4o
	case "gpt-4o-mini":
		return openai.ChatModelGPT4oMini
	case "gpt-4-turbo":
		return openai.ChatModelGPT4Turbo
	case "gpt-4":
		return openai.ChatModelGPT4
	case "gpt-3.5-turbo":
		return openai.ChatModelGPT3_5Turbo
	case "o1":
		return openai.ChatModelO1
	case "o1-mini":
		return openai.ChatModelO1Mini
	case "o1-preview":
		return openai.ChatModelO1Preview
	default:
		// Default to GPT-4o for unknown models
		return openai.ChatModelGPT4o
	}
}

// getDefaultModelInfo returns default model information for OpenAI models
func (h *OpenAISDKHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default values for OpenAI models
	return llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       128000, // Most OpenAI models support at least 128k context
		SupportsImages:      true,   // Most OpenAI models support images
		SupportsPromptCache: false,  // OpenAI doesn't support prompt caching
		InputPrice:          2.5,    // Default pricing per million tokens
		OutputPrice:         10.0,   // Default pricing per million tokens
		Description:         "OpenAI model",
	}
}

// OpenAIModelInfo represents model information from OpenAI API
type OpenAIModelInfo struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// OpenAIModelsResponse represents the response from OpenAI models API
type OpenAIModelsResponse struct {
	Object string            `json:"object"`
	Data   []OpenAIModelInfo `json:"data"`
}

// GetOpenAIModels fetches available models from OpenAI API and caches them
func GetOpenAIModels(ctx context.Context, apiKey string) ([]OpenAIModelInfo, error) {
	// Check cache first
	cacheDir := filepath.Join(os.Getenv("HOME"), ".codeforge", "cache")
	cacheFile := filepath.Join(cacheDir, "openai_models.json")

	// Check if cache exists and is recent (24 hours)
	if info, err := os.Stat(cacheFile); err == nil {
		if time.Since(info.ModTime()) < 24*time.Hour {
			// Load from cache
			data, err := os.ReadFile(cacheFile)
			if err == nil {
				var models []OpenAIModelInfo
				if json.Unmarshal(data, &models) == nil {
					return models, nil
				}
			}
		}
	}

	// Create OpenAI client
	client := openai.NewClient(option.WithAPIKey(apiKey))

	// Fetch models from API
	modelsList, err := client.Models.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OpenAI models: %w", err)
	}

	var models []OpenAIModelInfo
	for _, model := range modelsList.Data {
		// Filter for chat completion models only
		if strings.Contains(model.ID, "gpt") || strings.Contains(model.ID, "o1") {
			models = append(models, OpenAIModelInfo{
				ID:      model.ID,
				Object:  string(model.Object),
				Created: model.Created,
				OwnedBy: model.OwnedBy,
			})
		}
	}

	// Cache the results
	if err := os.MkdirAll(cacheDir, 0755); err == nil {
		if data, err := json.Marshal(models); err == nil {
			os.WriteFile(cacheFile, data, 0644)
		}
	}

	return models, nil
}
