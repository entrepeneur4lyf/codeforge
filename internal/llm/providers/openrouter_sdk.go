package providers

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	openrouter "github.com/revrost/go-openrouter"
)

// OpenRouterSDKHandler implements the ApiHandler interface using the official OpenRouter Go SDK
type OpenRouterSDKHandler struct {
	options llm.ApiHandlerOptions
	client  *openrouter.Client
}

// NewOpenRouterSDKHandler creates a new OpenRouter handler using the official SDK
func NewOpenRouterSDKHandler(options llm.ApiHandlerOptions) *OpenRouterSDKHandler {
	// Create client with API key
	client := openrouter.NewClient(options.APIKey)

	return &OpenRouterSDKHandler{
		options: options,
		client:  client,
	}
}

func (h *OpenRouterSDKHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

func (h *OpenRouterSDKHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// OpenRouter provides usage in the final stream event
	return nil, nil
}

// CreateMessage sends a message to OpenRouter and returns a streaming response
func (h *OpenRouterSDKHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	// Convert messages to OpenRouter format
	openrouterMessages := make([]openrouter.ChatCompletionMessage, 0, len(messages)+1)

	// Add system prompt if provided
	if systemPrompt != "" {
		openrouterMessages = append(openrouterMessages, openrouter.ChatCompletionMessage{
			Role:    openrouter.ChatMessageRoleSystem,
			Content: openrouter.Content{Text: systemPrompt},
		})
	}

	// Convert our messages to OpenRouter format
	for _, msg := range messages {
		// Extract text content from message
		var textContent string
		for _, content := range msg.Content {
			if textBlock, ok := content.(llm.TextBlock); ok {
				textContent += textBlock.Text
			}
		}

		openrouterMessages = append(openrouterMessages, openrouter.ChatCompletionMessage{
			Role:    convertRoleToOpenRouter(msg.Role),
			Content: openrouter.Content{Text: textContent},
		})
	}

	// Create request
	request := openrouter.ChatCompletionRequest{
		Model:    h.options.ModelID,
		Messages: openrouterMessages,
		Stream:   true,
	}

	// Create streaming completion
	stream, err := h.client.CreateChatCompletionStream(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to create chat completion stream: %w", err)
	}

	// Create output channel
	outputChan := make(chan llm.ApiStreamChunk, 100)

	// Start goroutine to process stream
	go func() {
		defer close(outputChan)
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				// Log error but don't send error chunk as it's not defined in our interface
				fmt.Printf("OpenRouter stream error: %v\n", err)
				break
			}

			// Extract content from response
			if len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
				content := response.Choices[0].Delta.Content
				if content != "" {
					outputChan <- llm.ApiStreamTextChunk{
						Text: content,
					}
				}
			}
		}
	}()

	return outputChan, nil
}

// convertRoleToOpenRouter converts our role format to OpenRouter format
func convertRoleToOpenRouter(role string) string {
	switch strings.ToLower(role) {
	case "user":
		return openrouter.ChatMessageRoleUser
	case "assistant":
		return openrouter.ChatMessageRoleAssistant
	case "system":
		return openrouter.ChatMessageRoleSystem
	default:
		return openrouter.ChatMessageRoleUser
	}
}

// getDefaultModelInfo returns default model information for OpenRouter models
func (h *OpenRouterSDKHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default values for OpenRouter models
	return llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       128000, // Most models support at least 128k context
		SupportsImages:      false,  // Varies by model
		SupportsPromptCache: false,  // OpenRouter doesn't support prompt caching
		InputPrice:          1.0,    // Default pricing per million tokens
		OutputPrice:         3.0,    // Default pricing per million tokens
		Description:         "OpenRouter model",
	}
}
