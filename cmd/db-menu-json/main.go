package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
	_ "github.com/tursodatabase/go-libsql"
)

// MenuStructure represents the complete menu hierarchy from database
type MenuStructure struct {
	Providers []ProviderMenu `json:"providers"`
	Generated string         `json:"generated"`
	Version   string         `json:"version"`
	Source    string         `json:"source"`
}

// ProviderMenu represents a provider and its models
type ProviderMenu struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	ModelCount  int         `json:"modelCount"`
	Models      []ModelInfo `json:"models"`
	Filters     []Filter    `json:"filters,omitempty"` // For OpenRouter
}

// ModelInfo represents a model from database
type ModelInfo struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Provider      string    `json:"provider"`
	ContextLength int       `json:"contextLength"`
	CreatedDate   int64     `json:"createdDate"`
	LastSeen      time.Time `json:"lastSeen"`
	IsFiltered    bool      `json:"isFiltered"`
}

// Filter represents OpenRouter provider filters
type Filter struct {
	Name        string `json:"name"`
	ProviderKey string `json:"providerKey"`
	Description string `json:"description"`
	ModelCount  int    `json:"modelCount"`
}

func main() {
	// Initialize configuration
	workingDir, _ := os.Getwd()
	cfg, err := config.Load(workingDir, false)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize vector database
	if err := vectordb.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	db := vectordb.GetInstance()
	ctx := context.Background()

	menu := MenuStructure{
		Generated: time.Now().Format(time.RFC3339),
		Version:   "1.0.0",
		Source:    "database",
		Providers: []ProviderMenu{},
	}

	// Add hardcoded providers (Anthropic, OpenAI)
	menu.Providers = append(menu.Providers, getAnthropicProvider())
	menu.Providers = append(menu.Providers, getOpenAIProvider())

	// Add OpenRouter provider with database models
	openrouterProvider, err := getOpenRouterProvider(ctx, db)
	if err != nil {
		log.Printf("Warning: Failed to get OpenRouter models from database: %v", err)
		// Add empty OpenRouter provider
		openrouterProvider = ProviderMenu{
			ID:          "openrouter",
			Name:        "OpenRouter",
			Description: "Access to multiple providers through OpenRouter",
			ModelCount:  0,
			Models:      []ModelInfo{},
			Filters:     getOpenRouterFilters(),
		}
	}
	menu.Providers = append(menu.Providers, openrouterProvider)

	// Generate JSON file
	jsonData, err := json.MarshalIndent(menu, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Write to file
	outputPath := filepath.Join(".", "menu-structure-db.json")
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		log.Fatalf("Failed to write JSON file: %v", err)
	}

	fmt.Printf("Database menu structure JSON generated successfully: %s\n", outputPath)
	fmt.Printf("Total providers: %d\n", len(menu.Providers))

	for _, provider := range menu.Providers {
		fmt.Printf("- %s: %d models", provider.Name, provider.ModelCount)
		if len(provider.Filters) > 0 {
			fmt.Printf(" (%d filters)", len(provider.Filters))
		}
		fmt.Println()
	}
}

// getAnthropicProvider returns hardcoded Anthropic provider
func getAnthropicProvider() ProviderMenu {
	models := []ModelInfo{
		{
			ID:            "claude-opus-4-20250514",
			Name:          "Claude Opus 4",
			Description:   "Most powerful model for complex challenges and coding",
			Provider:      "anthropic",
			ContextLength: 200000,
			CreatedDate:   1747526400, // 2025-05-14
			LastSeen:      time.Now(),
			IsFiltered:    false,
		},
		{
			ID:            "claude-sonnet-4-20250514",
			Name:          "Claude Sonnet 4",
			Description:   "High-performance model with exceptional reasoning and efficiency",
			Provider:      "anthropic",
			ContextLength: 200000,
			CreatedDate:   1747526400, // 2025-05-14
			LastSeen:      time.Now(),
			IsFiltered:    false,
		},
		{
			ID:            "claude-3-5-sonnet-20241022",
			Name:          "Claude 3.5 Sonnet",
			Description:   "Previous generation intelligent model",
			Provider:      "anthropic",
			ContextLength: 200000,
			CreatedDate:   1729555200, // 2024-10-22
			LastSeen:      time.Now(),
			IsFiltered:    false,
		},
		{
			ID:            "claude-3-5-haiku-20241022",
			Name:          "Claude 3.5 Haiku",
			Description:   "Fast model for simple tasks",
			Provider:      "anthropic",
			ContextLength: 200000,
			CreatedDate:   1729555200, // 2024-10-22
			LastSeen:      time.Now(),
			IsFiltered:    false,
		},
	}

	return ProviderMenu{
		ID:          "anthropic",
		Name:        "Anthropic",
		Description: "Claude models for advanced reasoning",
		ModelCount:  len(models),
		Models:      models,
	}
}

// getOpenAIProvider returns hardcoded OpenAI provider
func getOpenAIProvider() ProviderMenu {
	models := []ModelInfo{
		{
			ID:            "gpt-4o",
			Name:          "GPT-4o",
			Description:   "Most capable GPT-4 model",
			Provider:      "openai",
			ContextLength: 128000,
			CreatedDate:   1715299200, // 2024-05-10
			LastSeen:      time.Now(),
			IsFiltered:    false,
		},
		{
			ID:            "gpt-4o-mini",
			Name:          "GPT-4o Mini",
			Description:   "Faster and cheaper GPT-4 model",
			Provider:      "openai",
			ContextLength: 128000,
			CreatedDate:   1721088000, // 2024-07-16
			LastSeen:      time.Now(),
			IsFiltered:    false,
		},
		{
			ID:            "o1",
			Name:          "O1",
			Description:   "Advanced reasoning model",
			Provider:      "openai",
			ContextLength: 200000,
			CreatedDate:   1726704000, // 2024-09-19
			LastSeen:      time.Now(),
			IsFiltered:    false,
		},
		{
			ID:            "o1-mini",
			Name:          "O1 Mini",
			Description:   "Faster reasoning model",
			Provider:      "openai",
			ContextLength: 128000,
			CreatedDate:   1726704000, // 2024-09-19
			LastSeen:      time.Now(),
			IsFiltered:    false,
		},
	}

	return ProviderMenu{
		ID:          "openai",
		Name:        "OpenAI",
		Description: "GPT models for general purpose tasks",
		ModelCount:  len(models),
		Models:      models,
	}
}

// getOpenRouterProvider queries database for OpenRouter models
func getOpenRouterProvider(ctx context.Context, db *vectordb.VectorDB) (ProviderMenu, error) {
	// Query all OpenRouter models from database
	query := `
		SELECT model_id, name, description, context_length, created_date, last_seen, provider
		FROM openrouter_models
		WHERE last_seen > datetime('now', '-24 hours')
		ORDER BY provider, name
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return ProviderMenu{}, fmt.Errorf("failed to query OpenRouter models: %w", err)
	}
	defer rows.Close()

	var models []ModelInfo
	providerCounts := make(map[string]int)

	for rows.Next() {
		var model ModelInfo
		var lastSeenStr string
		var provider sql.NullString

		err := rows.Scan(
			&model.ID, &model.Name, &model.Description,
			&model.ContextLength, &model.CreatedDate, &lastSeenStr, &provider,
		)
		if err != nil {
			log.Printf("Warning: Failed to scan model row: %v", err)
			continue
		}

		// Parse last_seen timestamp
		if lastSeen, err := time.Parse("2006-01-02 15:04:05", lastSeenStr); err == nil {
			model.LastSeen = lastSeen
		}

		// Set provider (transform to lowercase)
		if provider.Valid {
			model.Provider = strings.ToLower(provider.String)
		} else {
			model.Provider = "unknown"
		}

		// Filter out non-coding models
		if isCodeGenerationModel(model.ID) {
			model.IsFiltered = false
			models = append(models, model)
			providerCounts[model.Provider]++
		} else {
			model.IsFiltered = true
		}
	}

	// Create filters with counts
	filters := getOpenRouterFilters()
	for i := range filters {
		if filters[i].ProviderKey != "" {
			filters[i].ModelCount = providerCounts[filters[i].ProviderKey]
		} else {
			// "All Providers" filter
			filters[i].ModelCount = len(models)
		}
	}

	return ProviderMenu{
		ID:          "openrouter",
		Name:        "OpenRouter",
		Description: "Access to multiple providers through OpenRouter",
		ModelCount:  len(models),
		Models:      models,
		Filters:     filters,
	}, nil
}

// getOpenRouterFilters returns the standard OpenRouter filters with lowercase provider keys
func getOpenRouterFilters() []Filter {
	return []Filter{
		{Name: "All Providers", ProviderKey: "", Description: "Show models from all providers"},
		{Name: "Anthropic", ProviderKey: "anthropic", Description: "Claude models via OpenRouter"},
		{Name: "OpenAI", ProviderKey: "openai", Description: "GPT models via OpenRouter"},
		{Name: "Google", ProviderKey: "google", Description: "Gemini models via OpenRouter"},
		{Name: "Meta/Llama", ProviderKey: "meta", Description: "Llama models via OpenRouter"},
		{Name: "Mistral AI", ProviderKey: "mistral ai", Description: "Mistral models via OpenRouter"},
		{Name: "Qwen", ProviderKey: "qwen", Description: "Qwen models via OpenRouter"},
		{Name: "DeepSeek", ProviderKey: "deepseek", Description: "DeepSeek models via OpenRouter"},
		{Name: "Nvidia", ProviderKey: "nvidia", Description: "Nvidia models via OpenRouter"},
		{Name: "Microsoft", ProviderKey: "microsoft", Description: "Microsoft models via OpenRouter"},
	}
}

// isCodeGenerationModel filters out only obvious non-coding models
func isCodeGenerationModel(modelID string) bool {
	// Only filter out clearly non-text models
	excludePatterns := []string{
		"whisper", "tts", "dall-e", "embedding", "moderation",
		"stable-diffusion", "flux", "midjourney",
	}

	modelLower := strings.ToLower(modelID)
	for _, pattern := range excludePatterns {
		if strings.Contains(modelLower, pattern) {
			return false
		}
	}

	return true
}
