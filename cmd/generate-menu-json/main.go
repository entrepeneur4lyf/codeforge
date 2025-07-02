package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/providers"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

// MenuStructure represents the complete menu hierarchy
type MenuStructure struct {
	Providers []ProviderMenu `json:"providers"`
	Generated string         `json:"generated"`
	Version   string         `json:"version"`
}

// ProviderMenu represents a provider and its models
type ProviderMenu struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	HasAPIKey   bool        `json:"hasApiKey"`
	Models      []ModelInfo `json:"models"`
	Filters     []Filter    `json:"filters,omitempty"` // For OpenRouter
}

// ModelInfo represents a model in the menu
type ModelInfo struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Provider      string   `json:"provider"`
	ContextLength int      `json:"contextLength"`
	InputPrice    float64  `json:"inputPrice"`
	OutputPrice   float64  `json:"outputPrice"`
	Capabilities  []string `json:"capabilities"`
	IsFavorite    bool     `json:"isFavorite"`
}

// Filter represents OpenRouter provider filters
type Filter struct {
	Name        string `json:"name"`
	ProviderKey string `json:"providerKey"`
	Description string `json:"description"`
}

// parsePrice converts a price string to float64, returns 0.0 if parsing fails
func parsePrice(priceStr string) float64 {
	if priceStr == "" {
		return 0.0
	}
	// Remove any currency symbols or extra characters
	cleanPrice := strings.TrimSpace(priceStr)
	if price, err := strconv.ParseFloat(cleanPrice, 64); err == nil {
		return price
	}
	return 0.0
}

// getOpenAIModelsFromDatabase queries OpenAI models from database
func getOpenAIModelsFromDatabase(ctx context.Context, db *vectordb.VectorDB) ([]ModelInfo, error) {
	query := `
		SELECT model_id, name, description, context_length, created_date, last_seen
		FROM openai_models
		WHERE last_seen > datetime('now', '-24 hours')
		ORDER BY name
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query OpenAI models: %w", err)
	}
	defer rows.Close()

	var models []ModelInfo
	for rows.Next() {
		var modelID, name, description, createdDate, lastSeen string
		var contextLength int

		err := rows.Scan(&modelID, &name, &description, &contextLength, &createdDate, &lastSeen)
		if err != nil {
			continue // Skip invalid rows
		}

		// Set pricing based on model type
		var inputPrice, outputPrice float64
		if strings.HasPrefix(modelID, "gpt-4o") {
			inputPrice = 2.5
			outputPrice = 10.0
		} else if strings.HasPrefix(modelID, "o1") {
			inputPrice = 15.0
			outputPrice = 60.0
			if strings.Contains(modelID, "mini") {
				inputPrice = 3.0
				outputPrice = 12.0
			}
		} else if strings.HasPrefix(modelID, "gpt-4") {
			inputPrice = 30.0
			outputPrice = 60.0
		} else if strings.HasPrefix(modelID, "gpt-3.5") {
			inputPrice = 0.5
			outputPrice = 1.5
		} else {
			inputPrice = 1.0
			outputPrice = 2.0
		}

		models = append(models, ModelInfo{
			ID:            modelID,
			Name:          name,
			Description:   description,
			Provider:      "openai",
			ContextLength: contextLength,
			InputPrice:    inputPrice,
			OutputPrice:   outputPrice,
			Capabilities:  []string{"text", "code"},
			IsFavorite:    false,
		})
	}

	return models, nil
}

func main() {
	// Initialize configuration
	wd, _ := os.Getwd()
	cfg, err := config.Load(wd, false)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize vector database
	if err := vectordb.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	ctx := context.Background()
	menu := MenuStructure{
		Generated: "2025-01-02T00:00:00Z",
		Version:   "1.0.0",
		Providers: []ProviderMenu{},
	}

	// Add Anthropic provider
	anthropicMenu := ProviderMenu{
		ID:          "anthropic",
		Name:        "Anthropic",
		Description: "Claude models for advanced reasoning",
		HasAPIKey:   os.Getenv("ANTHROPIC_API_KEY") != "",
		Models:      []ModelInfo{},
	}

	// Use latest Anthropic models (Claude 4 released May 2025)
	// Note: Anthropic doesn't have a public models API, so we use the latest known models
	log.Printf("Using latest Anthropic models (Claude 4 generation)...")
	{
		anthropicMenu.Models = []ModelInfo{
			{
				ID:            "claude-opus-4-20250514",
				Name:          "Claude Opus 4",
				Description:   "Most powerful model for complex challenges and coding",
				Provider:      "anthropic",
				ContextLength: 200000,
				InputPrice:    15.0,
				OutputPrice:   75.0,
				Capabilities:  []string{"text", "code", "vision", "reasoning", "analysis"},
				IsFavorite:    false,
			},
			{
				ID:            "claude-sonnet-4-20250514",
				Name:          "Claude Sonnet 4",
				Description:   "High-performance model with exceptional reasoning and efficiency",
				Provider:      "anthropic",
				ContextLength: 200000,
				InputPrice:    3.0,
				OutputPrice:   15.0,
				Capabilities:  []string{"text", "code", "vision", "reasoning"},
				IsFavorite:    false,
			},
			{
				ID:            "claude-3-5-sonnet-20241022",
				Name:          "Claude 3.5 Sonnet",
				Description:   "Previous generation intelligent model",
				Provider:      "anthropic",
				ContextLength: 200000,
				InputPrice:    3.0,
				OutputPrice:   15.0,
				Capabilities:  []string{"text", "code", "vision", "reasoning"},
				IsFavorite:    false,
			},
			{
				ID:            "claude-3-5-haiku-20241022",
				Name:          "Claude 3.5 Haiku",
				Description:   "Fast model for simple tasks",
				Provider:      "anthropic",
				ContextLength: 200000,
				InputPrice:    0.8,
				OutputPrice:   4.0,
				Capabilities:  []string{"text", "code", "speed"},
				IsFavorite:    false,
			},
		}
	}

	menu.Providers = append(menu.Providers, anthropicMenu)

	// Add OpenAI provider
	openaiMenu := ProviderMenu{
		ID:          "openai",
		Name:        "OpenAI",
		Description: "GPT models for general purpose tasks",
		HasAPIKey:   os.Getenv("OPENAI_API_KEY") != "",
		Models:      []ModelInfo{},
	}

	// Get OpenAI models from API (always fresh)
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		log.Printf("Fetching fresh OpenAI models from API...")
		if openaiModels, err := providers.GetOpenAIModels(ctx, apiKey); err == nil {
			log.Printf("Successfully fetched %d OpenAI models from API", len(openaiModels))

			// Log all models for debugging
			log.Printf("All OpenAI models:")
			for i, model := range openaiModels {
				log.Printf("  %d. %s (created: %d, owned_by: %s)", i+1, model.ID, model.Created, model.OwnedBy)
			}

			for _, model := range openaiModels {
				if isCodeGenerationModel(model.ID) {
					// Create user-friendly name from ID
					name := strings.ReplaceAll(model.ID, "-", " ")
					words := strings.Fields(name)
					for i, word := range words {
						if len(word) > 0 {
							words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
						}
					}
					displayName := strings.Join(words, " ")

					// Set default values based on model type
					contextLength := 128000
					inputPrice := 2.5
					outputPrice := 10.0

					if strings.HasPrefix(model.ID, "gpt-4o") {
						contextLength = 128000
						inputPrice = 2.5
						outputPrice = 10.0
					} else if strings.HasPrefix(model.ID, "o1") {
						contextLength = 200000
						inputPrice = 15.0
						outputPrice = 60.0
						if strings.Contains(model.ID, "mini") {
							inputPrice = 3.0
							outputPrice = 12.0
						}
					}

					openaiMenu.Models = append(openaiMenu.Models, ModelInfo{
						ID:            model.ID,
						Name:          displayName,
						Description:   fmt.Sprintf("OpenAI %s model", displayName),
						Provider:      "openai",
						ContextLength: contextLength,
						InputPrice:    inputPrice,
						OutputPrice:   outputPrice,
						Capabilities:  []string{"text", "code"},
						IsFavorite:    false,
					})
				}
			}
		} else {
			log.Printf("Failed to fetch OpenAI models from API: %v", err)
		}
	}

	// Add fallback models if no API key or API failed
	if len(openaiMenu.Models) == 0 {
		openaiMenu.Models = []ModelInfo{
			{
				ID:            "gpt-4o",
				Name:          "GPT-4o",
				Description:   "Most capable GPT-4 model",
				Provider:      "openai",
				ContextLength: 128000,
				InputPrice:    5.0,
				OutputPrice:   15.0,
				Capabilities:  []string{"text", "code", "vision"},
				IsFavorite:    false,
			},
			{
				ID:            "gpt-4o-mini",
				Name:          "GPT-4o Mini",
				Description:   "Faster and cheaper GPT-4 model",
				Provider:      "openai",
				ContextLength: 128000,
				InputPrice:    0.15,
				OutputPrice:   0.6,
				Capabilities:  []string{"text", "code"},
				IsFavorite:    false,
			},
		}
	}

	menu.Providers = append(menu.Providers, openaiMenu)

	// Add OpenRouter provider with filters
	openrouterMenu := ProviderMenu{
		ID:          "openrouter",
		Name:        "OpenRouter",
		Description: "Access to multiple providers through OpenRouter",
		HasAPIKey:   os.Getenv("OPENROUTER_API_KEY") != "",
		Models:      []ModelInfo{},
		Filters: []Filter{
			{Name: "All Providers", ProviderKey: "", Description: "Show models from all providers"},
			{Name: "Anthropic", ProviderKey: "anthropic", Description: "Claude models via OpenRouter"},
			{Name: "OpenAI", ProviderKey: "openai", Description: "GPT models via OpenRouter"},
			{Name: "Google", ProviderKey: "google", Description: "Gemini models via OpenRouter"},
			{Name: "Meta/Llama", ProviderKey: "meta-llama", Description: "Llama models via OpenRouter"},
			{Name: "Mistral", ProviderKey: "mistralai", Description: "Mistral models via OpenRouter"},
			{Name: "Qwen", ProviderKey: "qwen", Description: "Qwen models via OpenRouter"},
			{Name: "DeepSeek", ProviderKey: "deepseek", Description: "DeepSeek models via OpenRouter"},
		},
	}

	// Get OpenRouter models from database
	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		if openrouterModels, err := providers.GetOpenRouterModels(ctx, apiKey); err == nil {
			for _, model := range openrouterModels {
				if isCodeGenerationModel(model.ID) {
					openrouterMenu.Models = append(openrouterMenu.Models, ModelInfo{
						ID:            model.ID,
						Name:          model.Name,
						Description:   model.Description,
						Provider:      "openrouter",
						ContextLength: model.ContextLength,
						InputPrice:    parsePrice(model.Pricing.Prompt),
						OutputPrice:   parsePrice(model.Pricing.Completion),
						Capabilities:  []string{"text", "code"},
						IsFavorite:    false,
					})
				}
			}
		}
	}

	menu.Providers = append(menu.Providers, openrouterMenu)

	// Generate JSON file
	jsonData, err := json.MarshalIndent(menu, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Write to file
	outputPath := filepath.Join(".", "menu-structure.json")
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		log.Fatalf("Failed to write JSON file: %v", err)
	}

	fmt.Printf("Menu structure JSON generated successfully: %s\n", outputPath)
	fmt.Printf("Total providers: %d\n", len(menu.Providers))

	for _, provider := range menu.Providers {
		fmt.Printf("- %s: %d models\n", provider.Name, len(provider.Models))
	}
}

// isCodeGenerationModel filters out non-coding models
func isCodeGenerationModel(modelID string) bool {
	// Filter out audio, video, image, and embedding models
	excludePatterns := []string{
		"whisper", "tts", "dall-e", "embedding", "moderation",
		"vision", "audio", "video", "image", "speech",
	}

	modelLower := fmt.Sprintf("%s", modelID)
	for _, pattern := range excludePatterns {
		if contains(modelLower, pattern) {
			return false
		}
	}

	return true
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					containsMiddle(s, substr))))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
