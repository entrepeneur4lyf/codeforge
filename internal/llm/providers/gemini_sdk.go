package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"google.golang.org/genai"
)

// GeminiSDKHandler implements the ApiHandler interface using the official Google Generative AI SDK
type GeminiSDKHandler struct {
	options llm.ApiHandlerOptions
	client  *genai.Client
}

// NewGeminiSDKHandler creates a new Gemini handler using the official Google SDK
func NewGeminiSDKHandler(options llm.ApiHandlerOptions) *GeminiSDKHandler {
	return &GeminiSDKHandler{
		options: options,
		// Client will be created when needed with API key
	}
}

func (h *GeminiSDKHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

func (h *GeminiSDKHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Gemini provides usage in the response
	return nil, nil
}

// CreateMessage sends a message to Gemini and returns a streaming response
func (h *GeminiSDKHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	// Create client if not already created
	if h.client == nil {
		// Create client config with API key
		config := &genai.ClientConfig{
			APIKey:  h.options.APIKey,
			Backend: genai.BackendGeminiAPI,
		}
		client, err := genai.NewClient(ctx, config)
		if err != nil {
			return nil, fmt.Errorf("failed to create Gemini client: %w", err)
		}
		h.client = client
	}

	// Convert messages to Gemini format
	var contents []*genai.Content
	for _, msg := range messages {
		var parts []*genai.Part
		for _, block := range msg.Content {
			if textBlock, ok := block.(llm.TextBlock); ok {
				parts = append(parts, &genai.Part{Text: textBlock.Text})
			}
		}

		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}

		contents = append(contents, &genai.Content{
			Parts: parts,
			Role:  role,
		})
	}

	// Create generation config
	config := &genai.GenerateContentConfig{
		Temperature:     genai.Ptr(float32(0.7)),
		MaxOutputTokens: 4096,
	}

	// Set system instruction if provided
	if systemPrompt != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{{Text: systemPrompt}},
		}
	}

	// Use streaming generation
	iter := h.client.Models.GenerateContentStream(ctx, h.options.ModelID, contents, config)

	// Create response channel
	responseChan := make(chan llm.ApiStreamChunk, 100)

	// Start goroutine to handle streaming
	go func() {
		defer close(responseChan)

		for result, err := range iter {
			if err != nil {
				// Log error but don't send error chunk as it's not defined in our interface
				fmt.Printf("Gemini stream error: %v\n", err)
				return
			}

			// Extract text from response
			if len(result.Candidates) > 0 && result.Candidates[0].Content != nil {
				for _, part := range result.Candidates[0].Content.Parts {
					if part.Text != "" {
						responseChan <- llm.ApiStreamTextChunk{Text: part.Text}
					}
				}
			}
		}

		// Send final usage information
		responseChan <- llm.ApiStreamUsageChunk{
			InputTokens:  0, // Gemini doesn't provide detailed token counts in streaming
			OutputTokens: 0,
			TotalCost:    nil,
		}
	}()

	return responseChan, nil
}

// getDefaultModelInfo returns default model information for Gemini models
func (h *GeminiSDKHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default values for Gemini models
	return llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       128000, // Most Gemini models support large context
		SupportsImages:      true,   // Most Gemini models support images
		SupportsPromptCache: false,  // Google doesn't support prompt caching yet
		InputPrice:          1.0,    // Default pricing per million tokens
		OutputPrice:         3.0,    // Default pricing per million tokens
		Description:         "Google Gemini model",
	}
}

// GeminiModelInfo represents model information from Google
type GeminiModelInfo struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"display_name"`
	Version     string  `json:"version"`
	Description string  `json:"description"`
	InputPrice  float64 `json:"input_price"`
	OutputPrice float64 `json:"output_price"`
}

// GetGeminiModels returns available Gemini models (hardcoded since Google doesn't have a public models API)
func GetGeminiModels(ctx context.Context, apiKey string) ([]GeminiModelInfo, error) {
	return getCachedGeminiModels(ctx, apiKey, false)
}

// getCachedGeminiModels returns cached models if available and fresh
func getCachedGeminiModels(ctx context.Context, apiKey string, forceRefresh bool) ([]GeminiModelInfo, error) {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".codeforge", "cache")
	cacheFile := filepath.Join(cacheDir, "gemini_models.json")

	// Check cache first (unless force refresh)
	if !forceRefresh {
		if info, err := os.Stat(cacheFile); err == nil {
			if time.Since(info.ModTime()) < 24*time.Hour {
				// Load from cache
				data, err := os.ReadFile(cacheFile)
				if err == nil {
					var models []GeminiModelInfo
					if json.Unmarshal(data, &models) == nil {
						return models, nil
					}
				}
			}
		}
	}

	// Google doesn't have a public models API, so we use hardcoded models
	models := []GeminiModelInfo{
		{
			ID: "gemini-2.0-flash-exp", DisplayName: "Gemini 2.0 Flash (Experimental)", Version: "2.0",
			Description: "Latest experimental Gemini model", InputPrice: 0.075, OutputPrice: 0.3,
		},
		{
			ID: "gemini-1.5-pro", DisplayName: "Gemini 1.5 Pro", Version: "1.5",
			Description: "Most capable Gemini model", InputPrice: 1.25, OutputPrice: 5.0,
		},
		{
			ID: "gemini-1.5-flash", DisplayName: "Gemini 1.5 Flash", Version: "1.5",
			Description: "Fast and efficient Gemini model", InputPrice: 0.075, OutputPrice: 0.3,
		},
		{
			ID: "gemini-1.5-flash-8b", DisplayName: "Gemini 1.5 Flash 8B", Version: "1.5",
			Description: "Smaller, faster Gemini model", InputPrice: 0.0375, OutputPrice: 0.15,
		},
		{
			ID: "gemini-pro", DisplayName: "Gemini Pro", Version: "1.0",
			Description: "Original Gemini Pro model", InputPrice: 0.5, OutputPrice: 1.5,
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

// RefreshGeminiModelsAsync refreshes Gemini models in the background
func RefreshGeminiModelsAsync(apiKey string) {
	if apiKey == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := getCachedGeminiModels(ctx, apiKey, true)
		if err != nil {
			// Silent failure for background refresh
			fmt.Printf("Background Gemini model refresh failed: %v\n", err)
		}
	}()
}
