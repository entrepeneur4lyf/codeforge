package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/entrepeneur4lyf/codeforge/internal/llm"
)

// AnthropicSDKHandler implements the ApiHandler interface using the official Anthropic SDK
type AnthropicSDKHandler struct {
	options llm.ApiHandlerOptions
	client  *anthropic.Client
}

// NewAnthropicSDKHandler creates a new Anthropic handler using the official SDK
func NewAnthropicSDKHandler(options llm.ApiHandlerOptions) *AnthropicSDKHandler {
	// Create client with API key
	client := anthropic.NewClient(
		option.WithAPIKey(options.APIKey),
	)

	return &AnthropicSDKHandler{
		options: options,
		client:  &client,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CreateMessage sends a message to Anthropic and returns a streaming response
func (h *AnthropicSDKHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	// Convert our messages to Anthropic format
	anthropicMessages := make([]anthropic.MessageParam, 0, len(messages))

	for _, msg := range messages {
		// Extract text content from message
		var textContent string
		for _, content := range msg.Content {
			if textBlock, ok := content.(llm.TextBlock); ok {
				textContent += textBlock.Text
			}
		}

		switch msg.Role {
		case "user":
			anthropicMessages = append(anthropicMessages,
				anthropic.NewUserMessage(anthropic.NewTextBlock(textContent)))
		case "assistant":
			anthropicMessages = append(anthropicMessages,
				anthropic.NewAssistantMessage(anthropic.NewTextBlock(textContent)))
		case "system":
			// System messages are handled separately in Anthropic API
			// For now, we'll convert them to user messages with a prefix
			content := fmt.Sprintf("System: %s", textContent)
			anthropicMessages = append(anthropicMessages,
				anthropic.NewUserMessage(anthropic.NewTextBlock(content)))
		}
	}

	// Determine the model to use
	model := h.getAnthropicModel(h.options.ModelID)

	// Create streaming request
	stream := h.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		MaxTokens: 4096, // Default max tokens
		Messages:  anthropicMessages,
		Model:     model,
	})

	// Create output channel
	outputChan := make(chan llm.ApiStreamChunk, 100)

	// Start goroutine to process stream
	go func() {
		defer close(outputChan)
		defer stream.Close()

		for stream.Next() {
			event := stream.Current()

			switch eventVariant := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch deltaVariant := eventVariant.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					// Send text chunk
					outputChan <- llm.ApiStreamTextChunk{
						Text: deltaVariant.Text,
					}
				}
			case anthropic.MessageDeltaEvent:
				// Handle message-level events if needed
				if eventVariant.Delta.StopSequence != "" {
					// Message completed - just continue, no special chunk needed
				}
			case anthropic.MessageStopEvent:
				// Stream completed - just return, channel will be closed
				return
			}
		}

		// Check for errors
		if err := stream.Err(); err != nil {
			// Log error but don't send error chunk as it's not defined in our interface
			fmt.Printf("Anthropic stream error: %v\n", err)
		}
	}()

	return outputChan, nil
}

// getAnthropicModel converts our model ID to the appropriate Anthropic model constant
func (h *AnthropicSDKHandler) getAnthropicModel(modelID string) anthropic.Model {
	// Remove provider prefix if present
	if strings.Contains(modelID, "/") {
		parts := strings.Split(modelID, "/")
		if len(parts) == 2 {
			modelID = parts[1]
		}
	}

	// Map to Anthropic model constants
	switch modelID {
	case "claude-3.5-sonnet", "claude-3-5-sonnet-20241022":
		return anthropic.ModelClaude3_5Sonnet20241022
	case "claude-3.5-sonnet-20240620":
		return anthropic.ModelClaude_3_5_Sonnet_20240620
	case "claude-3.5-sonnet-latest":
		return anthropic.ModelClaude3_5SonnetLatest
	case "claude-3.5-haiku", "claude-3-5-haiku-20241022":
		return anthropic.ModelClaude3_5Haiku20241022
	case "claude-3.5-haiku-latest":
		return anthropic.ModelClaude3_5HaikuLatest
	case "claude-3-opus", "claude-3-opus-20240229":
		return anthropic.ModelClaude_3_Opus_20240229
	case "claude-3-opus-latest":
		return anthropic.ModelClaude3OpusLatest
	case "claude-3-sonnet", "claude-3-sonnet-20240229":
		return anthropic.ModelClaude_3_Sonnet_20240229
	case "claude-3-haiku", "claude-3-haiku-20240307":
		return anthropic.ModelClaude_3_Haiku_20240307
	case "claude-2.1":
		return anthropic.ModelClaude_2_1
	case "claude-2.0":
		return anthropic.ModelClaude_2_0
	default:
		// Default to latest Sonnet if unknown
		return anthropic.ModelClaude3_5SonnetLatest
	}
}

// GetModel returns the model ID and info for the current configuration
func (h *AnthropicSDKHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

// GetApiStreamUsage returns usage information if available
func (h *AnthropicSDKHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Anthropic provides usage in the final stream event
	return nil, nil
}

// getDefaultModelInfo returns default model information for Anthropic models
func (h *AnthropicSDKHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default values for Anthropic models
	return llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       200000, // Most Claude models support 200k context
		SupportsImages:      true,   // Most Claude models support images
		SupportsPromptCache: true,   // Anthropic supports prompt caching
		InputPrice:          3.0,    // Default pricing per million tokens
		OutputPrice:         15.0,   // Default pricing per million tokens
		Description:         "Anthropic Claude model",
	}
}

// AnthropicModelInfo represents model information from Anthropic
type AnthropicModelInfo struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Type        string  `json:"type"`
	CreatedAt   string  `json:"created_at"`
	MaxTokens   int     `json:"max_tokens"`
	InputPrice  float64 `json:"input_price"`
	OutputPrice float64 `json:"output_price"`
}

// GetAnthropicModels returns available Anthropic models (hardcoded since Anthropic doesn't have a public models API)
func GetAnthropicModels(ctx context.Context, apiKey string) ([]AnthropicModelInfo, error) {
	return getCachedAnthropicModels(ctx, apiKey, false)
}

// getCachedAnthropicModels returns cached models if available and fresh
func getCachedAnthropicModels(ctx context.Context, apiKey string, forceRefresh bool) ([]AnthropicModelInfo, error) {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".codeforge", "cache")
	cacheFile := filepath.Join(cacheDir, "anthropic_models.json")

	// Check cache first (unless force refresh)
	if !forceRefresh {
		if info, err := os.Stat(cacheFile); err == nil {
			if time.Since(info.ModTime()) < 24*time.Hour {
				// Load from cache
				data, err := os.ReadFile(cacheFile)
				if err == nil {
					var models []AnthropicModelInfo
					if json.Unmarshal(data, &models) == nil {
						return models, nil
					}
				}
			}
		}
	}

	// Anthropic doesn't have a public models API, so we use hardcoded models
	models := []AnthropicModelInfo{
		{
			ID: "claude-3-5-sonnet-20241022", DisplayName: "Claude 3.5 Sonnet", Type: "text",
			CreatedAt: "2024-10-22", MaxTokens: 8192, InputPrice: 3.0, OutputPrice: 15.0,
		},
		{
			ID: "claude-3-5-haiku-20241022", DisplayName: "Claude 3.5 Haiku", Type: "text",
			CreatedAt: "2024-10-22", MaxTokens: 8192, InputPrice: 0.8, OutputPrice: 4.0,
		},
		{
			ID: "claude-3-opus-20240229", DisplayName: "Claude 3 Opus", Type: "text",
			CreatedAt: "2024-02-29", MaxTokens: 4096, InputPrice: 15.0, OutputPrice: 75.0,
		},
		{
			ID: "claude-3-sonnet-20240229", DisplayName: "Claude 3 Sonnet", Type: "text",
			CreatedAt: "2024-02-29", MaxTokens: 4096, InputPrice: 3.0, OutputPrice: 15.0,
		},
		{
			ID: "claude-3-haiku-20240307", DisplayName: "Claude 3 Haiku", Type: "text",
			CreatedAt: "2024-03-07", MaxTokens: 4096, InputPrice: 0.25, OutputPrice: 1.25,
		},
	}

	// Cache the results
	if err := os.MkdirAll(cacheDir, 0755); err == nil {
		if data, err := json.Marshal(models); err == nil {
			os.WriteFile(cacheFile, data, 0644)
		}
	}

	return models, nil
}

// RefreshAnthropicModelsAsync refreshes Anthropic models in the background
func RefreshAnthropicModelsAsync(apiKey string) {
	if apiKey == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := getCachedAnthropicModels(ctx, apiKey, true)
		if err != nil {
			// Silent failure for background refresh
			fmt.Printf("Background Anthropic model refresh failed: %v\n", err)
		}
	}()
}
