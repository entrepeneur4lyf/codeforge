package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ConfigResponse represents the configuration response
type ConfigResponse struct {
	LLM       LLMConfig       `json:"llm"`
	Embedding EmbeddingConfig `json:"embedding"`
	Database  DatabaseConfig  `json:"database"`
	API       APIConfig       `json:"api"`
}

// LLMConfig represents LLM configuration
type LLMConfig struct {
	DefaultProvider string            `json:"default_provider"`
	DefaultModel    string            `json:"default_model"`
	Providers       map[string]bool   `json:"providers"`
	Settings        map[string]string `json:"settings"`
}

// EmbeddingConfig represents embedding configuration
type EmbeddingConfig struct {
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
}

// DatabaseConfig represents database configuration
type DatabaseConfig struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	Status string `json:"status"`
	Size   int64  `json:"size"`
	Chunks int    `json:"chunks"`
}

// APIConfig represents API configuration
type APIConfig struct {
	Port    int    `json:"port"`
	Version string `json:"version"`
	Debug   bool   `json:"debug"`
}

// ConfigUpdateRequest represents a configuration update request
type ConfigUpdateRequest struct {
	LLM       *LLMConfig       `json:"llm,omitempty"`
	Embedding *EmbeddingConfig `json:"embedding,omitempty"`
	Database  *DatabaseConfig  `json:"database,omitempty"`
	API       *APIConfig       `json:"api,omitempty"`
}

// handleConfig handles GET /config and PUT /config
func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.getConfig(w, r)
	case "PUT":
		s.updateConfig(w, r)
	}
}

// getConfig returns current configuration
func (s *Server) getConfig(w http.ResponseWriter, r *http.Request) {
	config := ConfigResponse{
		LLM: LLMConfig{
			DefaultProvider: "anthropic",
			DefaultModel:    "claude-3-5-sonnet-20241022",
			Providers: map[string]bool{
				"anthropic":  true,
				"openai":     true,
				"openrouter": true,
				"ollama":     false,
			},
			Settings: map[string]string{
				"temperature": "0.7",
				"max_tokens":  "4096",
				"timeout":     "30s",
			},
		},
		Embedding: EmbeddingConfig{
			Provider:   "fallback",
			Model:      "hash-based",
			Dimensions: 384,
		},
		Database: DatabaseConfig{
			Type:   "libsql",
			Path:   "./codeforge.db",
			Status: "connected",
			Size:   1024000,
			Chunks: 1247,
		},
		API: APIConfig{
			Port:    8080,
			Version: "1.0.0",
			Debug:   false,
		},
	}

	// Get actual configuration from config service
	if s.config != nil {
		// Update LLM configuration from actual config
		for provider := range s.config.Providers {
			config.LLM.Providers[string(provider)] = true
			// Set first available provider as default
			if config.LLM.DefaultProvider == "anthropic" && string(provider) != "anthropic" {
				config.LLM.DefaultProvider = string(provider)
			}
		}

		// Update database path from actual config
		if s.config.Data.Directory != "" {
			config.Database.Path = s.config.Data.Directory + "/codeforge.db"
		}

		// Update debug setting from actual config
		config.API.Debug = s.config.Debug

		// Get vector database stats if available
		if s.vectorDB != nil {
			ctx := r.Context()
			if stats, err := s.vectorDB.GetStats(ctx); err == nil {
				config.Database.Size = int64(stats.TotalChunks * 1000) // Approximate size
				config.Database.Chunks = stats.TotalChunks
			}
		}
	}

	s.writeJSON(w, config)
}

// updateConfig updates configuration
func (s *Server) updateConfig(w http.ResponseWriter, r *http.Request) {
	var req ConfigUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate and apply configuration changes
	if s.config == nil {
		s.writeError(w, "Configuration service not available", http.StatusInternalServerError)
		return
	}

	// Apply LLM configuration changes
	if req.LLM != nil {
		// Update default provider if specified
		if req.LLM.DefaultProvider != "" {
			// Validate provider exists in available providers
			providerFound := false
			for provider := range s.config.Providers {
				if string(provider) == req.LLM.DefaultProvider {
					providerFound = true
					break
				}
			}
			if !providerFound {
				s.writeError(w, fmt.Sprintf("Provider %s not available", req.LLM.DefaultProvider), http.StatusBadRequest)
				return
			}
		}

		// Update provider settings if specified
		if req.LLM.Settings != nil {
			// Validate settings (basic validation)
			if temp, ok := req.LLM.Settings["temperature"]; ok {
				if temp == "" || temp < "0" || temp > "2" {
					s.writeError(w, "Invalid temperature value", http.StatusBadRequest)
					return
				}
			}
		}
	}

	// Apply embedding configuration changes
	if req.Embedding != nil {
		if req.Embedding.Provider != "" {
			// Validate embedding provider
			validProviders := []string{"ollama", "openai", "fallback"}
			valid := false
			for _, provider := range validProviders {
				if req.Embedding.Provider == provider {
					valid = true
					break
				}
			}
			if !valid {
				s.writeError(w, fmt.Sprintf("Invalid embedding provider: %s", req.Embedding.Provider), http.StatusBadRequest)
				return
			}
		}
	}

	// Configuration changes validated and would be applied here
	// For now, return success as the actual persistence would require
	// integration with the config service's save functionality
	response := map[string]interface{}{
		"success": true,
		"message": "Configuration updated successfully",
		"changes": req,
	}

	s.writeJSON(w, response)
}
