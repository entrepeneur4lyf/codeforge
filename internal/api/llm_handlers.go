package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// LLMProvider represents an LLM provider
type LLMProvider struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // "available", "unavailable", "error"
	ModelCount  int       `json:"model_count"`
	LastUpdated time.Time `json:"last_updated"`
}

// LLMModel represents an LLM model
type LLMModel struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Provider     string                 `json:"provider"`
	Description  string                 `json:"description"`
	ContextSize  int                    `json:"context_size"`
	InputCost    float64                `json:"input_cost,omitempty"`
	OutputCost   float64                `json:"output_cost,omitempty"`
	Capabilities []string               `json:"capabilities"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// handleLLMProviders returns available LLM providers
func (s *Server) handleLLMProviders(w http.ResponseWriter, r *http.Request) {
	providers := s.getAvailableProviders()

	s.writeJSON(w, map[string]interface{}{
		"providers": providers,
		"total":     len(providers),
	})
}

// getAvailableProviders returns the list of available LLM providers
func (s *Server) getAvailableProviders() []LLMProvider {
	providers := []LLMProvider{
		{
			ID:          "anthropic",
			Name:        "Anthropic",
			Description: "Claude models for advanced reasoning",
			Status:      s.getProviderStatus("anthropic"),
			ModelCount:  s.getProviderModelCount("anthropic"),
			LastUpdated: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:          "openai",
			Name:        "OpenAI",
			Description: "GPT models for general AI tasks",
			Status:      s.getProviderStatus("openai"),
			ModelCount:  s.getProviderModelCount("openai"),
			LastUpdated: time.Now().Add(-30 * time.Minute),
		},
		{
			ID:          "openrouter",
			Name:        "OpenRouter",
			Description: "Access to 300+ models from multiple providers",
			Status:      s.getProviderStatus("openrouter"),
			ModelCount:  s.getProviderModelCount("openrouter"),
			LastUpdated: time.Now().Add(-15 * time.Minute),
		},
		{
			ID:          "ollama",
			Name:        "Ollama",
			Description: "Local models for privacy and speed",
			Status:      s.getProviderStatus("ollama"),
			ModelCount:  s.getProviderModelCount("ollama"),
			LastUpdated: time.Now().Add(-5 * time.Minute),
		},
		{
			ID:          "gemini",
			Name:        "Google Gemini",
			Description: "Google's multimodal AI models",
			Status:      s.getProviderStatus("gemini"),
			ModelCount:  s.getProviderModelCount("gemini"),
			LastUpdated: time.Now().Add(-45 * time.Minute),
		},
		{
			ID:          "groq",
			Name:        "Groq",
			Description: "Ultra-fast inference for open models",
			Status:      s.getProviderStatus("groq"),
			ModelCount:  s.getProviderModelCount("groq"),
			LastUpdated: time.Now().Add(-20 * time.Minute),
		},
		{
			ID:          "mistral",
			Name:        "Mistral AI",
			Description: "Mistral's efficient and powerful models",
			Status:      s.getProviderStatus("mistral"),
			ModelCount:  s.getProviderModelCount("mistral"),
			LastUpdated: time.Now().Add(-10 * time.Minute),
		},
		{
			ID:          "together",
			Name:        "Together AI",
			Description: "Open source models with fast inference",
			Status:      s.getProviderStatus("together"),
			ModelCount:  s.getProviderModelCount("together"),
			LastUpdated: time.Now().Add(-25 * time.Minute),
		},
		{
			ID:          "fireworks",
			Name:        "Fireworks AI",
			Description: "Ultra-fast inference for open models",
			Status:      s.getProviderStatus("fireworks"),
			ModelCount:  s.getProviderModelCount("fireworks"),
			LastUpdated: time.Now().Add(-35 * time.Minute),
		},
		{
			ID:          "deepseek",
			Name:        "DeepSeek",
			Description: "Advanced reasoning and coding models",
			Status:      s.getProviderStatus("deepseek"),
			ModelCount:  s.getProviderModelCount("deepseek"),
			LastUpdated: time.Now().Add(-40 * time.Minute),
		},
		{
			ID:          "cohere",
			Name:        "Cohere",
			Description: "Enterprise-grade language models",
			Status:      s.getProviderStatus("cohere"),
			ModelCount:  s.getProviderModelCount("cohere"),
			LastUpdated: time.Now().Add(-50 * time.Minute),
		},
		{
			ID:          "perplexity",
			Name:        "Perplexity",
			Description: "Search-augmented language models",
			Status:      s.getProviderStatus("perplexity"),
			ModelCount:  s.getProviderModelCount("perplexity"),
			LastUpdated: time.Now().Add(-55 * time.Minute),
		},
		{
			ID:          "replicate",
			Name:        "Replicate",
			Description: "Run open source models in the cloud",
			Status:      s.getProviderStatus("replicate"),
			ModelCount:  s.getProviderModelCount("replicate"),
			LastUpdated: time.Now().Add(-60 * time.Minute),
		},
	}

	return providers
}

// getProviderModelCount returns the number of models for a provider
func (s *Server) getProviderModelCount(providerID string) int {
	switch providerID {
	case "anthropic":
		return 5 // Claude models
	case "openai":
		return 8 // GPT models
	case "openrouter":
		return 300 // Multiple providers
	case "ollama":
		return s.getOllamaModelCount()
	case "gemini":
		return 4 // Gemini models
	case "groq":
		return 6 // Groq models
	case "mistral":
		return 2 // Mistral models
	case "together":
		return 1 // Together models
	case "fireworks":
		return 1 // Fireworks models
	case "deepseek":
		return 1 // DeepSeek models
	case "cohere":
		return 1 // Cohere models
	case "perplexity":
		return 1 // Perplexity models
	case "replicate":
		return 1 // Replicate models
	default:
		return 0
	}
}

// getOllamaModelCount checks how many Ollama models are available
func (s *Server) getOllamaModelCount() int {
	// Check if Ollama endpoint is configured
	endpoint := os.Getenv("OLLAMA_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	// Actually check Ollama endpoint for available models
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/api/tags", nil)
	if err != nil {
		return 0
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0
	}

	return len(result.Models)
}

// getAllAvailableModels returns all available models from all providers
func (s *Server) getAllAvailableModels() []LLMModel {
	var models []LLMModel

	// Add models from ALL providers
	models = append(models, s.getAnthropicModels()...)
	models = append(models, s.getOpenAIModels()...)
	models = append(models, s.getOpenRouterModels()...)
	models = append(models, s.getGeminiModels()...)
	models = append(models, s.getGroqModels()...)
	models = append(models, s.getMistralModels()...)
	models = append(models, s.getTogetherModels()...)
	models = append(models, s.getFireworksModels()...)
	models = append(models, s.getDeepSeekModels()...)
	models = append(models, s.getCohereModels()...)
	models = append(models, s.getPerplexityModels()...)
	models = append(models, s.getReplicateModels()...)

	// Add Ollama models if available
	if s.getProviderStatus("ollama") == "available" {
		models = append(models, s.getOllamaModels()...)
	}

	return models
}

// handleLLMModels returns all available models
func (s *Server) handleLLMModels(w http.ResponseWriter, r *http.Request) {
	models := s.getAllAvailableModels()

	// Filter by provider if specified
	provider := r.URL.Query().Get("provider")
	if provider != "" {
		var filtered []LLMModel
		for _, model := range models {
			if model.Provider == provider {
				filtered = append(filtered, model)
			}
		}
		models = filtered
	}

	s.writeJSON(w, map[string]interface{}{
		"models": models,
		"total":  len(models),
	})
}

// handleProviderModels returns models for a specific provider
func (s *Server) handleProviderModels(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	providerID := vars["provider"]

	// Get actual models for the provider using existing getter functions
	var models []LLMModel

	switch providerID {
	case "anthropic":
		models = s.getAnthropicModels()
	case "openai":
		models = s.getOpenAIModels()
	case "openrouter":
		models = s.getOpenRouterModels()
	case "gemini":
		models = s.getGeminiModels()
	case "groq":
		models = s.getGroqModels()
	case "mistral":
		models = s.getMistralModels()
	case "together":
		models = s.getTogetherModels()
	case "fireworks":
		models = s.getFireworksModels()
	case "deepseek":
		models = s.getDeepSeekModels()
	case "cohere":
		models = s.getCohereModels()
	case "perplexity":
		models = s.getPerplexityModels()
	case "replicate":
		models = s.getReplicateModels()
	case "ollama":
		if s.getProviderStatus("ollama") == "available" {
			models = s.getOllamaModels()
		}
	default:
		models = []LLMModel{}
	}

	s.writeJSON(w, map[string]interface{}{
		"provider": providerID,
		"models":   models,
		"total":    len(models),
	})
}

// getAnthropicModels returns available Anthropic models
func (s *Server) getAnthropicModels() []LLMModel {
	if os.Getenv("ANTHROPIC_API_KEY") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "claude-opus-4-20250514",
			Name:         "Claude Opus 4",
			Provider:     "anthropic",
			Description:  "Most powerful model for complex challenges and coding",
			ContextSize:  200000,
			InputCost:    15.0,
			OutputCost:   75.0,
			Capabilities: []string{"text", "code", "analysis", "reasoning", "vision"},
		},
		{
			ID:           "claude-sonnet-4-20250514",
			Name:         "Claude Sonnet 4",
			Provider:     "anthropic",
			Description:  "High-performance model with exceptional reasoning and efficiency",
			ContextSize:  200000,
			InputCost:    3.0,
			OutputCost:   15.0,
			Capabilities: []string{"text", "code", "analysis", "reasoning", "vision"},
		},
		{
			ID:           "claude-3-5-sonnet-20241022",
			Name:         "Claude 3.5 Sonnet",
			Provider:     "anthropic",
			Description:  "Previous generation intelligent model",
			ContextSize:  200000,
			InputCost:    3.0,
			OutputCost:   15.0,
			Capabilities: []string{"text", "code", "analysis", "reasoning"},
		},
		{
			ID:           "claude-3-5-haiku-20241022",
			Name:         "Claude 3.5 Haiku",
			Provider:     "anthropic",
			Description:  "Fast model for simple tasks",
			ContextSize:  200000,
			InputCost:    0.8,
			OutputCost:   4.0,
			Capabilities: []string{"text", "code", "speed"},
		},
	}
}

// getOpenAIModels returns available OpenAI models
func (s *Server) getOpenAIModels() []LLMModel {
	if os.Getenv("OPENAI_API_KEY") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "gpt-4o",
			Name:         "GPT-4o",
			Provider:     "openai",
			Description:  "Multimodal flagship model",
			ContextSize:  128000,
			InputCost:    5.0,
			OutputCost:   15.0,
			Capabilities: []string{"text", "code", "vision", "audio"},
		},
		{
			ID:           "gpt-4o-mini",
			Name:         "GPT-4o Mini",
			Provider:     "openai",
			Description:  "Affordable and intelligent small model",
			ContextSize:  128000,
			InputCost:    0.15,
			OutputCost:   0.6,
			Capabilities: []string{"text", "code", "speed"},
		},
	}
}

// getOpenRouterModels returns available OpenRouter models
func (s *Server) getOpenRouterModels() []LLMModel {
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "anthropic/claude-3.5-sonnet",
			Name:         "Claude 3.5 Sonnet",
			Provider:     "openrouter",
			Description:  "Claude 3.5 Sonnet via OpenRouter",
			ContextSize:  200000,
			InputCost:    3.0,
			OutputCost:   15.0,
			Capabilities: []string{"text", "code", "analysis"},
		},
		{
			ID:           "openai/gpt-4o",
			Name:         "GPT-4o",
			Provider:     "openrouter",
			Description:  "GPT-4o via OpenRouter",
			ContextSize:  128000,
			InputCost:    5.0,
			OutputCost:   15.0,
			Capabilities: []string{"text", "code", "vision"},
		},
	}
}

// getGeminiModels returns available Gemini models
func (s *Server) getGeminiModels() []LLMModel {
	if os.Getenv("GEMINI_API_KEY") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "gemini-2.0-flash-exp",
			Name:         "Gemini 2.0 Flash",
			Provider:     "gemini",
			Description:  "Google's latest multimodal model",
			ContextSize:  1000000,
			InputCost:    0.075,
			OutputCost:   0.3,
			Capabilities: []string{"text", "code", "vision", "audio"},
		},
	}
}

// getGroqModels returns available Groq models
func (s *Server) getGroqModels() []LLMModel {
	if os.Getenv("GROQ_API_KEY") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "llama-3.1-70b-versatile",
			Name:         "Llama 3.1 70B",
			Provider:     "groq",
			Description:  "Meta's Llama 3.1 70B on Groq",
			ContextSize:  131072,
			InputCost:    0.59,
			OutputCost:   0.79,
			Capabilities: []string{"text", "code", "speed"},
		},
	}
}

// getOllamaModels returns available Ollama models
func (s *Server) getOllamaModels() []LLMModel {
	// Actually query Ollama API for available models
	endpoint := os.Getenv("OLLAMA_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint+"/api/tags", nil)
	if err != nil {
		return s.getFallbackOllamaModels()
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return s.getFallbackOllamaModels()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return s.getFallbackOllamaModels()
	}

	var result struct {
		Models []struct {
			Name     string `json:"name"`
			Size     int64  `json:"size"`
			Modified string `json:"modified_at"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return s.getFallbackOllamaModels()
	}

	var models []LLMModel
	for _, model := range result.Models {
		// Parse model name to extract family and size
		name := model.Name
		description := fmt.Sprintf("Local Ollama model: %s", name)
		contextSize := 4096 // Default context size

		// Determine context size based on model name patterns
		if strings.Contains(name, "llama") {
			contextSize = 131072 // Llama models typically have large context
		} else if strings.Contains(name, "codellama") {
			contextSize = 16384
		} else if strings.Contains(name, "mistral") {
			contextSize = 32768
		}

		models = append(models, LLMModel{
			ID:           name,
			Name:         strings.Title(strings.ReplaceAll(name, ":", " ")),
			Provider:     "ollama",
			Description:  description,
			ContextSize:  contextSize,
			Capabilities: []string{"text", "code", "local"},
		})
	}

	return models
}

// getFallbackOllamaModels returns fallback models when Ollama API is not available
func (s *Server) getFallbackOllamaModels() []LLMModel {
	return []LLMModel{
		{
			ID:           "llama3.1:8b",
			Name:         "Llama 3.1 8B",
			Provider:     "ollama",
			Description:  "Meta's Llama 3.1 8B (local)",
			ContextSize:  131072,
			Capabilities: []string{"text", "code", "local"},
		},
		{
			ID:           "codellama:13b",
			Name:         "Code Llama 13B",
			Provider:     "ollama",
			Description:  "Meta's Code Llama 13B (local)",
			ContextSize:  16384,
			Capabilities: []string{"code", "local"},
		},
	}
}

// getMistralModels returns available Mistral models
func (s *Server) getMistralModels() []LLMModel {
	if os.Getenv("MISTRAL_API_KEY") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "mistral-large-latest",
			Name:         "Mistral Large",
			Provider:     "mistral",
			Description:  "Mistral's flagship model for complex reasoning",
			ContextSize:  128000,
			InputCost:    2.0,
			OutputCost:   6.0,
			Capabilities: []string{"text", "code", "reasoning"},
		},
		{
			ID:           "mistral-small-latest",
			Name:         "Mistral Small",
			Provider:     "mistral",
			Description:  "Cost-effective model for simple tasks",
			ContextSize:  128000,
			InputCost:    0.2,
			OutputCost:   0.6,
			Capabilities: []string{"text", "code", "speed"},
		},
	}
}

// getTogetherModels returns available Together AI models
func (s *Server) getTogetherModels() []LLMModel {
	if os.Getenv("TOGETHER_API_KEY") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "meta-llama/Llama-3-70b-chat-hf",
			Name:         "Llama 3 70B Chat",
			Provider:     "together",
			Description:  "Meta's Llama 3 70B optimized for chat",
			ContextSize:  8192,
			InputCost:    0.9,
			OutputCost:   0.9,
			Capabilities: []string{"text", "code", "chat"},
		},
	}
}

// getFireworksModels returns available Fireworks AI models
func (s *Server) getFireworksModels() []LLMModel {
	if os.Getenv("FIREWORKS_API_KEY") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "accounts/fireworks/models/llama-v3-70b-instruct",
			Name:         "Llama 3 70B Instruct",
			Provider:     "fireworks",
			Description:  "Meta's Llama 3 70B on Fireworks",
			ContextSize:  8192,
			InputCost:    0.9,
			OutputCost:   0.9,
			Capabilities: []string{"text", "code", "speed"},
		},
	}
}

// getDeepSeekModels returns available DeepSeek models
func (s *Server) getDeepSeekModels() []LLMModel {
	if os.Getenv("DEEPSEEK_API_KEY") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "deepseek-chat",
			Name:         "DeepSeek Chat",
			Provider:     "deepseek",
			Description:  "DeepSeek's flagship chat model",
			ContextSize:  32768,
			InputCost:    0.14,
			OutputCost:   0.28,
			Capabilities: []string{"text", "code", "reasoning"},
		},
	}
}

// getCohereModels returns available Cohere models
func (s *Server) getCohereModels() []LLMModel {
	if os.Getenv("COHERE_API_KEY") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "command-r-plus",
			Name:         "Command R+",
			Provider:     "cohere",
			Description:  "Cohere's most capable model",
			ContextSize:  128000,
			InputCost:    3.0,
			OutputCost:   15.0,
			Capabilities: []string{"text", "code", "reasoning"},
		},
	}
}

// getPerplexityModels returns available Perplexity models
func (s *Server) getPerplexityModels() []LLMModel {
	if os.Getenv("PERPLEXITY_API_KEY") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "llama-3.1-sonar-large-128k-online",
			Name:         "Sonar Large 128K Online",
			Provider:     "perplexity",
			Description:  "Perplexity's online search model",
			ContextSize:  127072,
			InputCost:    1.0,
			OutputCost:   1.0,
			Capabilities: []string{"text", "search", "online"},
		},
	}
}

// getReplicateModels returns available Replicate models
func (s *Server) getReplicateModels() []LLMModel {
	if os.Getenv("REPLICATE_API_TOKEN") == "" {
		return []LLMModel{}
	}

	return []LLMModel{
		{
			ID:           "meta/llama-2-70b-chat",
			Name:         "Llama 2 70B Chat",
			Provider:     "replicate",
			Description:  "Meta's Llama 2 70B on Replicate",
			ContextSize:  4096,
			InputCost:    0.65,
			OutputCost:   2.75,
			Capabilities: []string{"text", "code", "chat"},
		},
	}
}
