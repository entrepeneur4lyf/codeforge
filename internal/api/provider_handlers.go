package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/models"
	"github.com/gorilla/mux"
)

// ProviderConfig represents a provider configuration
type ProviderConfig struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "llm", "embedding", "vector"
	Enabled     bool                   `json:"enabled"`
	Default     bool                   `json:"default"`
	Settings    map[string]interface{} `json:"settings"`
	Status      string                 `json:"status"` // "available", "configured", "error"
	LastChecked string                 `json:"last_checked"`
	Models      []string               `json:"models,omitempty"`
}

// ProviderUpdateRequest represents a provider update request
type ProviderUpdateRequest struct {
	Enabled  *bool                  `json:"enabled,omitempty"`
	Default  *bool                  `json:"default,omitempty"`
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// EmbeddingProviderConfig represents embedding provider settings
type EmbeddingProviderConfig struct {
	Provider   string `json:"provider"`   // "ollama", "openai", "fallback"
	Model      string `json:"model"`      // specific model name
	Dimensions int    `json:"dimensions"` // embedding dimensions
	Endpoint   string `json:"endpoint,omitempty"`
	APIKey     string `json:"api_key,omitempty"`
}

// LLMProviderConfig represents LLM provider settings
type LLMProviderConfig struct {
	Provider    string  `json:"provider"` // "anthropic", "openai", "openrouter", etc.
	Model       string  `json:"model"`    // default model for this provider
	APIKey      string  `json:"api_key,omitempty"`
	BaseURL     string  `json:"base_url,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Timeout     int     `json:"timeout,omitempty"`
}

// handleProviders handles GET /providers
func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	providerType := r.URL.Query().Get("type") // "llm", "embedding", "all"

	providers := []ProviderConfig{}

	// LLM Providers
	if providerType == "" || providerType == "all" || providerType == "llm" {
		llmProviders := []ProviderConfig{
			{
				ID:      "anthropic",
				Name:    "Anthropic",
				Type:    "llm",
				Enabled: s.isProviderEnabled("anthropic"),
				Default: s.isDefaultProvider("llm", "anthropic"),
				Settings: map[string]interface{}{
					"api_key":     s.getProviderSetting("anthropic", "api_key"),
					"model":       "claude-3-5-sonnet-20241022",
					"temperature": 0.7,
					"max_tokens":  4096,
				},
				Status:      s.getProviderStatus("anthropic"),
				LastChecked: "2024-06-30T00:30:00Z",
				Models:      []string{"claude-3-5-sonnet-20241022", "claude-3-haiku-20240307"},
			},
			{
				ID:      "openai",
				Name:    "OpenAI",
				Type:    "llm",
				Enabled: s.isProviderEnabled("openai"),
				Default: s.isDefaultProvider("llm", "openai"),
				Settings: map[string]interface{}{
					"api_key":     s.getProviderSetting("openai", "api_key"),
					"model":       "gpt-4o",
					"temperature": 0.7,
					"max_tokens":  4096,
				},
				Status:      s.getProviderStatus("openai"),
				LastChecked: "2024-06-30T00:30:00Z",
				Models:      []string{"gpt-4o", "gpt-4o-mini", "gpt-3.5-turbo"},
			},
			{
				ID:      "openrouter",
				Name:    "OpenRouter",
				Type:    "llm",
				Enabled: s.isProviderEnabled("openrouter"),
				Default: s.isDefaultProvider("llm", "openrouter"),
				Settings: map[string]interface{}{
					"api_key": s.getProviderSetting("openrouter", "api_key"),
					"model":   "anthropic/claude-3.5-sonnet",
				},
				Status:      s.getProviderStatus("openrouter"),
				LastChecked: "2024-06-30T00:30:00Z",
				Models:      []string{"anthropic/claude-3.5-sonnet", "openai/gpt-4o", "meta-llama/llama-3.1-70b-instruct"},
			},
			{
				ID:      "ollama",
				Name:    "Ollama",
				Type:    "llm",
				Enabled: s.isProviderEnabled("ollama"),
				Default: s.isDefaultProvider("llm", "ollama"),
				Settings: map[string]interface{}{
					"endpoint": "http://localhost:11434",
					"model":    "llama3.1",
				},
				Status:      s.getProviderStatus("ollama"),
				LastChecked: "2024-06-30T00:30:00Z",
				Models:      []string{"llama3.1", "codellama", "mistral"},
			},
		}
		providers = append(providers, llmProviders...)
	}

	// Embedding Providers
	if providerType == "" || providerType == "all" || providerType == "embedding" {
		embeddingProviders := []ProviderConfig{
			{
				ID:      "ollama-embedding",
				Name:    "Ollama Embeddings",
				Type:    "embedding",
				Enabled: s.isProviderEnabled("ollama-embedding"),
				Default: s.isDefaultProvider("embedding", "ollama"),
				Settings: map[string]interface{}{
					"endpoint":   "http://localhost:11434",
					"model":      "nomic-embed-text",
					"dimensions": 768,
				},
				Status:      s.getProviderStatus("ollama-embedding"),
				LastChecked: "2024-06-30T00:30:00Z",
				Models:      []string{"nomic-embed-text", "all-minilm", "mxbai-embed-large"},
			},
			{
				ID:      "openai-embedding",
				Name:    "OpenAI Embeddings",
				Type:    "embedding",
				Enabled: s.isProviderEnabled("openai-embedding"),
				Default: s.isDefaultProvider("embedding", "openai"),
				Settings: map[string]interface{}{
					"api_key":    s.getProviderSetting("openai", "api_key"),
					"model":      "text-embedding-3-small",
					"dimensions": 1536,
				},
				Status:      s.getProviderStatus("openai-embedding"),
				LastChecked: "2024-06-30T00:30:00Z",
				Models:      []string{"text-embedding-3-small", "text-embedding-3-large", "text-embedding-ada-002"},
			},
			{
				ID:      "fallback-embedding",
				Name:    "Fallback Embeddings",
				Type:    "embedding",
				Enabled: true, // Always enabled as fallback
				Default: s.isDefaultProvider("embedding", "fallback"),
				Settings: map[string]interface{}{
					"dimensions": 384,
					"algorithm":  "hash-based",
				},
				Status:      "available",
				LastChecked: "2024-06-30T00:30:00Z",
			},
		}
		providers = append(providers, embeddingProviders...)
	}

	s.writeJSON(w, map[string]interface{}{
		"providers": providers,
		"total":     len(providers),
		"type":      providerType,
	})
}

// handleProvider handles GET/PUT/DELETE /providers/{id}
func (s *Server) handleProvider(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerID := vars["id"]

	switch r.Method {
	case "GET":
		s.getProvider(w, r, providerID)
	case "PUT":
		s.updateProvider(w, r, providerID)
	case "DELETE":
		s.deleteProvider(w, r, providerID)
	default:
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getProvider returns a specific provider configuration
func (s *Server) getProvider(w http.ResponseWriter, r *http.Request, providerID string) {
	// Get actual provider from configuration
	if s.config == nil {
		s.writeError(w, "Configuration not available", http.StatusInternalServerError)
		return
	}

	// Check if provider exists in configuration
	providerType := models.ModelProvider(providerID)
	if _, exists := s.config.Providers[providerType]; !exists {
		s.writeError(w, "Provider not found", http.StatusNotFound)
		return
	}

	provider := ProviderConfig{
		ID:      providerID,
		Name:    strings.Title(providerID),
		Type:    "llm",
		Enabled: s.isProviderEnabled(providerID),
		Default: s.isDefaultProvider("llm", providerID),
		Settings: map[string]interface{}{
			"api_key": s.getProviderSetting(providerID, "api_key"),
		},
		Status:      s.getProviderStatus(providerID),
		LastChecked: "2024-06-30T00:30:00Z",
	}

	s.writeJSON(w, provider)
}

// updateProvider updates a provider configuration
func (s *Server) updateProvider(w http.ResponseWriter, r *http.Request, providerID string) {
	var req ProviderUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Implement actual provider update logic
	if s.config == nil {
		s.writeError(w, "Configuration not available", http.StatusInternalServerError)
		return
	}

	// Validate provider exists
	providerType := models.ModelProvider(providerID)
	if _, exists := s.config.Providers[providerType]; !exists {
		s.writeError(w, "Provider not found", http.StatusNotFound)
		return
	}

	// Update provider configuration
	if req.Settings != nil {
		if apiKey, ok := req.Settings["api_key"].(string); ok && apiKey != "" {
			// Update API key in configuration
			providerConfig := s.config.Providers[providerType]
			providerConfig.APIKey = apiKey
			s.config.Providers[providerType] = providerConfig
		}
	}

	if req.Enabled != nil {
		// Update enabled status
		providerConfig := s.config.Providers[providerType]
		providerConfig.Disabled = !*req.Enabled
		s.config.Providers[providerType] = providerConfig
	}

	response := map[string]interface{}{
		"success":     true,
		"provider_id": providerID,
		"message":     "Provider updated successfully",
		"changes":     req,
	}

	s.writeJSON(w, response)
}

// deleteProvider disables/removes a provider
func (s *Server) deleteProvider(w http.ResponseWriter, r *http.Request, providerID string) {
	// Implement provider deletion/disabling
	if s.config == nil {
		s.writeError(w, "Configuration not available", http.StatusInternalServerError)
		return
	}

	// Check if provider exists
	providerType := models.ModelProvider(providerID)
	if _, exists := s.config.Providers[providerType]; !exists {
		s.writeError(w, "Provider not found", http.StatusNotFound)
		return
	}

	// Disable the provider instead of deleting it
	providerConfig := s.config.Providers[providerType]
	providerConfig.Disabled = true
	s.config.Providers[providerType] = providerConfig

	response := map[string]interface{}{
		"success":     true,
		"provider_id": providerID,
		"message":     "Provider disabled successfully",
	}

	s.writeJSON(w, response)
}

// handleEmbeddingProvider handles embedding provider management
func (s *Server) handleEmbeddingProvider(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.getCurrentEmbeddingProvider(w, r)
	case "PUT":
		s.setEmbeddingProvider(w, r)
	default:
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getCurrentEmbeddingProvider returns current embedding provider
func (s *Server) getCurrentEmbeddingProvider(w http.ResponseWriter, r *http.Request) {
	// Get actual current provider from configuration
	embeddingConfig := EmbeddingProviderConfig{
		Provider:   "fallback",
		Model:      "hash-based",
		Dimensions: 384,
	}

	if s.config != nil && s.config.Embedding.Provider != "" {
		embeddingConfig.Provider = s.config.Embedding.Provider
		embeddingConfig.Model = s.config.Embedding.Model
		embeddingConfig.Endpoint = s.config.Embedding.BaseURL
	}

	s.writeJSON(w, embeddingConfig)
}

// setEmbeddingProvider changes the embedding provider
func (s *Server) setEmbeddingProvider(w http.ResponseWriter, r *http.Request) {
	var config EmbeddingProviderConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate and apply embedding provider change
	if s.config == nil {
		s.writeError(w, "Configuration not available", http.StatusInternalServerError)
		return
	}

	// Validate provider
	validProviders := []string{"ollama", "openai", "fallback"}
	valid := false
	for _, provider := range validProviders {
		if config.Provider == provider {
			valid = true
			break
		}
	}
	if !valid {
		s.writeError(w, fmt.Sprintf("Invalid embedding provider: %s", config.Provider), http.StatusBadRequest)
		return
	}

	// Update embedding configuration
	s.config.Embedding.Provider = config.Provider
	s.config.Embedding.Model = config.Model
	s.config.Embedding.BaseURL = config.Endpoint

	response := map[string]interface{}{
		"success": true,
		"message": "Embedding provider updated successfully",
		"config":  config,
	}

	s.writeJSON(w, response)
}

// Helper methods for provider status
func (s *Server) isProviderEnabled(providerID string) bool {
	// Check actual provider status from configuration
	if s.config == nil {
		return false
	}

	providerType := models.ModelProvider(providerID)
	if providerConfig, exists := s.config.Providers[providerType]; exists {
		return !providerConfig.Disabled && providerConfig.APIKey != ""
	}

	// Special cases for providers not in main config
	switch providerID {
	case "ollama", "ollama-embedding":
		return s.getProviderStatus(providerID) == "available"
	case "fallback-embedding":
		return true
	default:
		return false
	}
}

func (s *Server) isDefaultProvider(providerType, providerID string) bool {
	// Get from actual configuration
	if s.config == nil {
		// Fallback to defaults
		if providerType == "llm" {
			return providerID == "anthropic"
		}
		if providerType == "embedding" {
			return providerID == "fallback"
		}
		return false
	}

	if providerType == "llm" {
		// Find the first enabled provider as default
		providerOrder := []models.ModelProvider{
			models.ProviderCopilot,
			models.ProviderAnthropic,
			models.ProviderOpenAI,
			models.ProviderGemini,
			models.ProviderGROQ,
			models.ProviderOpenRouter,
		}

		for _, provider := range providerOrder {
			if providerConfig, exists := s.config.Providers[provider]; exists && !providerConfig.Disabled {
				return string(provider) == providerID
			}
		}
	}

	if providerType == "embedding" {
		return s.config.Embedding.Provider == providerID
	}

	return false
}

func (s *Server) getProviderSetting(providerID, setting string) string {
	// Get from actual configuration first
	if s.config != nil {
		providerType := models.ModelProvider(providerID)
		if providerConfig, exists := s.config.Providers[providerType]; exists {
			if setting == "api_key" {
				return maskAPIKey(providerConfig.APIKey)
			}
		}
	}

	// Fallback to environment variables
	if setting == "api_key" {
		switch providerID {
		case "anthropic":
			return maskAPIKey(getEnvVar("ANTHROPIC_API_KEY"))
		case "openai":
			return maskAPIKey(getEnvVar("OPENAI_API_KEY"))
		case "openrouter":
			return maskAPIKey(getEnvVar("OPENROUTER_API_KEY"))
		case "gemini":
			return maskAPIKey(getEnvVar("GEMINI_API_KEY"))
		case "groq":
			return maskAPIKey(getEnvVar("GROQ_API_KEY"))
		}
	}
	return ""
}

func (s *Server) getProviderStatus(providerID string) string {
	// Check actual provider availability
	switch providerID {
	case "anthropic":
		if s.config != nil {
			if providerConfig, exists := s.config.Providers[models.ProviderAnthropic]; exists && providerConfig.APIKey != "" {
				return "configured"
			}
		}
		if getEnvVar("ANTHROPIC_API_KEY") != "" {
			return "configured"
		}
		return "available"
	case "openai":
		if s.config != nil {
			if providerConfig, exists := s.config.Providers[models.ProviderOpenAI]; exists && providerConfig.APIKey != "" {
				return "configured"
			}
		}
		if getEnvVar("OPENAI_API_KEY") != "" {
			return "configured"
		}
		return "available"
	case "openrouter":
		if s.config != nil {
			if providerConfig, exists := s.config.Providers[models.ProviderOpenRouter]; exists && providerConfig.APIKey != "" {
				return "configured"
			}
		}
		if getEnvVar("OPENROUTER_API_KEY") != "" {
			return "configured"
		}
		return "available"
	case "ollama", "ollama-embedding":
		// Check if Ollama is running by trying to connect
		return s.checkOllamaAvailability()
	default:
		return "available"
	}
}

// Helper functions
func maskAPIKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func getEnvVar(name string) string {
	return os.Getenv(name)
}

// checkOllamaAvailability checks if Ollama is running and available
func (s *Server) checkOllamaAvailability() string {
	endpoint := os.Getenv("OLLAMA_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	// Try to connect to Ollama API
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(endpoint + "/api/tags")
	if err != nil {
		return "unavailable"
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return "available"
	}

	return "unavailable"
}
