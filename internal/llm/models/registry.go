package models

import (
	"fmt"
	"sync"
	"time"
)

// CanonicalModelID represents a canonical model identifier
type CanonicalModelID string

// ProviderID represents a provider identifier
type ProviderID string

// Provider constants
const (
	ProviderAnthropic  ProviderID = "anthropic"
	ProviderOpenAI     ProviderID = "openai"
	ProviderGemini     ProviderID = "gemini"
	ProviderOpenRouter ProviderID = "openrouter"
	ProviderBedrock    ProviderID = "bedrock"
	ProviderVertex     ProviderID = "vertex"
	ProviderDeepSeek   ProviderID = "deepseek"
	ProviderTogether   ProviderID = "together"
	ProviderFireworks  ProviderID = "fireworks"
	ProviderCerebras   ProviderID = "cerebras"
	ProviderGroq       ProviderID = "groq"
	ProviderOllama     ProviderID = "ollama"
	ProviderLMStudio   ProviderID = "lmstudio"
	ProviderXAI        ProviderID = "xai"
	ProviderMistral    ProviderID = "mistral"
	ProviderQwen       ProviderID = "qwen"
	ProviderDoubao     ProviderID = "doubao"
	ProviderSambanova  ProviderID = "sambanova"
	ProviderNebius     ProviderID = "nebius"
	ProviderAskSage    ProviderID = "asksage"
	ProviderSAPAICore  ProviderID = "sapaicore"
	ProviderLiteLLM    ProviderID = "litellm"
	ProviderRequesty   ProviderID = "requesty"
	ProviderClaudeCode ProviderID = "claude-code"
	ProviderGeminiCLI  ProviderID = "gemini-cli"
)

// Canonical model IDs for frontier models
const (
	ModelClaudeSonnet4  CanonicalModelID = "claude-sonnet-4"
	ModelClaudeOpus4    CanonicalModelID = "claude-opus-4"
	ModelClaude37Sonnet CanonicalModelID = "claude-3.7-sonnet"
	ModelClaude35Sonnet CanonicalModelID = "claude-3.5-sonnet"
	ModelClaude35Haiku  CanonicalModelID = "claude-3.5-haiku"
	ModelGPT4o          CanonicalModelID = "gpt-4o"
	ModelGPT4oMini      CanonicalModelID = "gpt-4o-mini"
	ModelO3Mini         CanonicalModelID = "o3-mini"
	ModelO1             CanonicalModelID = "o1"
	ModelO1Mini         CanonicalModelID = "o1-mini"
	ModelGemini25Flash  CanonicalModelID = "gemini-2.5-flash"
	ModelGemini20Flash  CanonicalModelID = "gemini-2.0-flash"
	ModelDeepSeekR1     CanonicalModelID = "deepseek-r1"
	ModelDeepSeekV3     CanonicalModelID = "deepseek-v3"
	ModelGrok3          CanonicalModelID = "grok-3"
)

// CanonicalModel represents a model in the canonical registry
type CanonicalModel struct {
	ID           CanonicalModelID                    `json:"id"`
	Name         string                              `json:"name"`
	Family       string                              `json:"family"`
	Version      string                              `json:"version"`
	Capabilities ModelCapabilities                   `json:"capabilities"`
	Pricing      ModelPricing                        `json:"pricing"`
	Limits       ModelLimits                         `json:"limits"`
	Providers    map[ProviderID]ProviderModelMapping `json:"providers"`
	CreatedAt    time.Time                           `json:"createdAt"`
	UpdatedAt    time.Time                           `json:"updatedAt"`
}

// ModelCapabilities represents what a model can do
type ModelCapabilities struct {
	SupportsImages      bool `json:"supportsImages"`
	SupportsPromptCache bool `json:"supportsPromptCache"`
	SupportsThinking    bool `json:"supportsThinking"`
	SupportsTools       bool `json:"supportsTools"`
	SupportsStreaming   bool `json:"supportsStreaming"`
	SupportsVision      bool `json:"supportsVision"`
	SupportsCode        bool `json:"supportsCode"`
	SupportsReasoning   bool `json:"supportsReasoning"`
}

// PricingTier represents tiered pricing based on usage
type PricingTier struct {
	ContextWindow    int     `json:"contextWindow"`
	InputPrice       float64 `json:"inputPrice"`
	OutputPrice      float64 `json:"outputPrice"`
	CacheReadsPrice  float64 `json:"cacheReadsPrice"`
	CacheWritesPrice float64 `json:"cacheWritesPrice"`
}

// ModelPricing represents pricing information
type ModelPricing struct {
	InputPrice       float64       `json:"inputPrice"`       // Per million tokens
	OutputPrice      float64       `json:"outputPrice"`      // Per million tokens
	CacheWritesPrice float64       `json:"cacheWritesPrice"` // Per million tokens
	CacheReadsPrice  float64       `json:"cacheReadsPrice"`  // Per million tokens
	ThinkingPrice    float64       `json:"thinkingPrice"`    // Per million thinking tokens
	Tiers            []PricingTier `json:"tiers,omitempty"`
	Currency         string        `json:"currency"`
}

// ModelLimits represents model limitations
type ModelLimits struct {
	MaxTokens          int     `json:"maxTokens"`
	ContextWindow      int     `json:"contextWindow"`
	MaxOutputTokens    int     `json:"maxOutputTokens"`
	MaxThinkingTokens  int     `json:"maxThinkingTokens,omitempty"`
	RequestsPerMinute  int     `json:"requestsPerMinute,omitempty"`
	TokensPerMinute    int     `json:"tokensPerMinute,omitempty"`
	DefaultTemperature float64 `json:"defaultTemperature"`
}

// ProviderModelMapping represents how a canonical model maps to a provider
type ProviderModelMapping struct {
	ProviderModelID string                 `json:"providerModelId"`           // Provider-specific model name
	ProviderConfig  ProviderSpecificConfig `json:"providerConfig"`            // Provider-specific settings
	Available       bool                   `json:"available"`                 // Is this model available?
	LastChecked     time.Time              `json:"lastChecked"`               // When availability was last verified
	PricingOverride *ModelPricing          `json:"pricingOverride,omitempty"` // Provider-specific pricing
	LimitsOverride  *ModelLimits           `json:"limitsOverride,omitempty"`  // Provider-specific limits
}

// ProviderSpecificConfig holds provider-specific configuration
type ProviderSpecificConfig struct {
	RequiresSpecialFormat bool                   `json:"requiresSpecialFormat,omitempty"`
	SpecialHeaders        map[string]string      `json:"specialHeaders,omitempty"`
	CustomEndpoint        string                 `json:"customEndpoint,omitempty"`
	AuthMethod            string                 `json:"authMethod,omitempty"`
	ExtraParams           map[string]interface{} `json:"extraParams,omitempty"`
}

// ModelRegistry manages the canonical model registry
type ModelRegistry struct {
	models map[CanonicalModelID]*CanonicalModel
	mutex  sync.RWMutex
}

// NewModelRegistry creates a new model registry
func NewModelRegistry() *ModelRegistry {
	registry := &ModelRegistry{
		models: make(map[CanonicalModelID]*CanonicalModel),
	}

	// Initialize with hardcoded frontier models
	registry.initializeFrontierModels()

	return registry
}

// GetModel retrieves a model by canonical ID
func (mr *ModelRegistry) GetModel(id CanonicalModelID) (*CanonicalModel, bool) {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	model, exists := mr.models[id]
	return model, exists
}

// GetModelByProvider retrieves a model by provider and provider model ID
func (mr *ModelRegistry) GetModelByProvider(providerID ProviderID, providerModelID string) (*CanonicalModel, bool) {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	for _, model := range mr.models {
		if mapping, exists := model.Providers[providerID]; exists {
			if mapping.ProviderModelID == providerModelID {
				return model, true
			}
		}
	}

	return nil, false
}

// ListModels returns all models in the registry
func (mr *ModelRegistry) ListModels() []*CanonicalModel {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	models := make([]*CanonicalModel, 0, len(mr.models))
	for _, model := range mr.models {
		models = append(models, model)
	}

	return models
}

// ListModelsByProvider returns all models available for a specific provider
func (mr *ModelRegistry) ListModelsByProvider(providerID ProviderID) []*CanonicalModel {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	var models []*CanonicalModel
	for _, model := range mr.models {
		if mapping, exists := model.Providers[providerID]; exists && mapping.Available {
			models = append(models, model)
		}
	}

	return models
}

// AddModel adds a new model to the registry
func (mr *ModelRegistry) AddModel(model *CanonicalModel) error {
	mr.mutex.Lock()
	defer mr.mutex.Unlock()

	if model.ID == "" {
		return fmt.Errorf("model ID cannot be empty")
	}

	if _, exists := mr.models[model.ID]; exists {
		return fmt.Errorf("model %s already exists", model.ID)
	}

	now := time.Now()
	model.CreatedAt = now
	model.UpdatedAt = now

	mr.models[model.ID] = model
	return nil
}

// UpdateModel updates an existing model in the registry
func (mr *ModelRegistry) UpdateModel(model *CanonicalModel) error {
	mr.mutex.Lock()
	defer mr.mutex.Unlock()

	if _, exists := mr.models[model.ID]; !exists {
		return fmt.Errorf("model %s does not exist", model.ID)
	}

	model.UpdatedAt = time.Now()
	mr.models[model.ID] = model
	return nil
}

// UpdateModelAvailability updates the availability of a model for a specific provider
func (mr *ModelRegistry) UpdateModelAvailability(modelID CanonicalModelID, providerID ProviderID, available bool) error {
	mr.mutex.Lock()
	defer mr.mutex.Unlock()

	model, exists := mr.models[modelID]
	if !exists {
		return fmt.Errorf("model %s does not exist", modelID)
	}

	if mapping, exists := model.Providers[providerID]; exists {
		mapping.Available = available
		mapping.LastChecked = time.Now()
		model.Providers[providerID] = mapping
		model.UpdatedAt = time.Now()
	}

	return nil
}

// GetProviderModelID gets the provider-specific model ID for a canonical model
func (mr *ModelRegistry) GetProviderModelID(modelID CanonicalModelID, providerID ProviderID) (string, error) {
	mr.mutex.RLock()
	defer mr.mutex.RUnlock()

	model, exists := mr.models[modelID]
	if !exists {
		return "", fmt.Errorf("model %s does not exist", modelID)
	}

	mapping, exists := model.Providers[providerID]
	if !exists {
		return "", fmt.Errorf("model %s is not available on provider %s", modelID, providerID)
	}

	if !mapping.Available {
		return "", fmt.Errorf("model %s is not currently available on provider %s", modelID, providerID)
	}

	return mapping.ProviderModelID, nil
}

// initializeFrontierModels initializes the registry with hardcoded frontier models
// Based on Cline's hardcoded model definitions for instant access
func (mr *ModelRegistry) initializeFrontierModels() {
	now := time.Now()

	// Claude 4 Sonnet - The flagship model
	claudeSonnet4 := &CanonicalModel{
		ID:      ModelClaudeSonnet4,
		Name:    "Claude 4 Sonnet",
		Family:  "claude",
		Version: "4.0",
		Capabilities: ModelCapabilities{
			SupportsImages:      true,
			SupportsPromptCache: true,
			SupportsThinking:    true,
			SupportsTools:       true,
			SupportsStreaming:   true,
			SupportsVision:      true,
			SupportsCode:        true,
			SupportsReasoning:   true,
		},
		Pricing: ModelPricing{
			InputPrice:       3.0,
			OutputPrice:      15.0,
			CacheWritesPrice: 3.75,
			CacheReadsPrice:  0.3,
			Currency:         "USD",
		},
		Limits: ModelLimits{
			MaxTokens:          8192,
			ContextWindow:      200000,
			MaxOutputTokens:    8192,
			DefaultTemperature: 0.0,
		},
		Providers: map[ProviderID]ProviderModelMapping{
			ProviderAnthropic: {
				ProviderModelID: "claude-sonnet-4-20250514",
				Available:       true,
				LastChecked:     now,
			},
			ProviderBedrock: {
				ProviderModelID: "anthropic.claude-sonnet-4-20250514-v1:0",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouter: {
				ProviderModelID: "anthropic/claude-sonnet-4",
				Available:       true,
				LastChecked:     now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// GPT-4o - OpenAI's flagship
	gpt4o := &CanonicalModel{
		ID:      ModelGPT4o,
		Name:    "GPT-4o",
		Family:  "gpt",
		Version: "4.0",
		Capabilities: ModelCapabilities{
			SupportsImages:      true,
			SupportsPromptCache: true,
			SupportsThinking:    false,
			SupportsTools:       true,
			SupportsStreaming:   true,
			SupportsVision:      true,
			SupportsCode:        true,
			SupportsReasoning:   false,
		},
		Pricing: ModelPricing{
			InputPrice:       2.5,
			OutputPrice:      10.0,
			CacheWritesPrice: 1.25,
			CacheReadsPrice:  1.25,
			Currency:         "USD",
		},
		Limits: ModelLimits{
			MaxTokens:          4096,
			ContextWindow:      128000,
			MaxOutputTokens:    4096,
			DefaultTemperature: 1.0,
		},
		Providers: map[ProviderID]ProviderModelMapping{
			ProviderOpenAI: {
				ProviderModelID: "gpt-4o",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouter: {
				ProviderModelID: "openai/gpt-4o",
				Available:       true,
				LastChecked:     now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// O3-mini - OpenAI's latest reasoning model
	o3Mini := &CanonicalModel{
		ID:      ModelO3Mini,
		Name:    "o3-mini",
		Family:  "o3",
		Version: "1.0",
		Capabilities: ModelCapabilities{
			SupportsImages:      false,
			SupportsPromptCache: false,
			SupportsThinking:    false,
			SupportsTools:       true,
			SupportsStreaming:   true,
			SupportsVision:      false,
			SupportsCode:        true,
			SupportsReasoning:   true,
		},
		Pricing: ModelPricing{
			InputPrice:  1.1,
			OutputPrice: 4.4,
			Currency:    "USD",
		},
		Limits: ModelLimits{
			MaxTokens:          65536,
			ContextWindow:      200000,
			MaxOutputTokens:    65536,
			DefaultTemperature: 1.0,
		},
		Providers: map[ProviderID]ProviderModelMapping{
			ProviderOpenAI: {
				ProviderModelID: "o3-mini",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouter: {
				ProviderModelID: "openai/o3-mini",
				Available:       true,
				LastChecked:     now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Gemini 2.5 Flash - Google's latest flagship
	gemini25Flash := &CanonicalModel{
		ID:      ModelGemini25Flash,
		Name:    "Gemini 2.5 Flash",
		Family:  "gemini",
		Version: "2.5",
		Capabilities: ModelCapabilities{
			SupportsImages:      true,
			SupportsPromptCache: true,
			SupportsThinking:    false,
			SupportsTools:       true,
			SupportsStreaming:   true,
			SupportsVision:      true,
			SupportsCode:        true,
			SupportsReasoning:   false,
		},
		Pricing: ModelPricing{
			InputPrice:       0.075,
			OutputPrice:      0.3,
			CacheWritesPrice: 0.09375,
			CacheReadsPrice:  0.01875,
			Currency:         "USD",
		},
		Limits: ModelLimits{
			MaxTokens:          8192,
			ContextWindow:      1000000,
			MaxOutputTokens:    8192,
			DefaultTemperature: 0.0,
		},
		Providers: map[ProviderID]ProviderModelMapping{
			ProviderGemini: {
				ProviderModelID: "gemini-2.5-flash",
				Available:       true,
				LastChecked:     now,
			},
			ProviderVertex: {
				ProviderModelID: "gemini-2.5-flash",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouter: {
				ProviderModelID: "google/gemini-2.5-flash",
				Available:       true,
				LastChecked:     now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// DeepSeek R1 model
	deepSeekR1 := &CanonicalModel{
		ID:      "deepseek-r1",
		Name:    "DeepSeek R1",
		Family:  "deepseek",
		Version: "r1",
		Capabilities: ModelCapabilities{
			SupportsImages:      false,
			SupportsThinking:    true,
			SupportsStreaming:   true,
			SupportsPromptCache: false,
			SupportsTools:       true,
		},
		Limits: ModelLimits{
			MaxTokens:         8192,
			ContextWindow:     128000,
			MaxThinkingTokens: 65536,
		},
		Pricing: ModelPricing{
			InputPrice:    0.14, // $0.14 per 1M tokens
			OutputPrice:   0.28, // $0.28 per 1M tokens
			ThinkingPrice: 0.14, // $0.14 per 1M thinking tokens
		},
		Providers: map[ProviderID]ProviderModelMapping{
			ProviderDeepSeek: {
				ProviderModelID: "deepseek-reasoner",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouter: {
				ProviderModelID: "deepseek/deepseek-r1",
				Available:       true,
				LastChecked:     now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Qwen 2.5 Coder model
	qwen25Coder := &CanonicalModel{
		ID:      "qwen-2.5-coder-32b",
		Name:    "Qwen 2.5 Coder 32B",
		Family:  "qwen",
		Version: "2.5-coder-32b",
		Capabilities: ModelCapabilities{
			SupportsImages:      false,
			SupportsThinking:    false,
			SupportsStreaming:   true,
			SupportsPromptCache: false,
			SupportsTools:       true,
		},
		Limits: ModelLimits{
			MaxTokens:     8192,
			ContextWindow: 128000,
		},
		Pricing: ModelPricing{
			InputPrice:  1.0, // $1.00 per 1M tokens
			OutputPrice: 3.0, // $3.00 per 1M tokens
		},
		Providers: map[ProviderID]ProviderModelMapping{
			ProviderQwen: {
				ProviderModelID: "qwen2.5-coder-32b-instruct",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouter: {
				ProviderModelID: "qwen/qwen-2.5-coder-32b-instruct",
				Available:       true,
				LastChecked:     now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Llama 3.3 70B model
	llama33 := &CanonicalModel{
		ID:      "llama-3.3-70b",
		Name:    "Llama 3.3 70B",
		Family:  "llama",
		Version: "3.3-70b",
		Capabilities: ModelCapabilities{
			SupportsImages:      false,
			SupportsThinking:    false,
			SupportsStreaming:   true,
			SupportsPromptCache: false,
			SupportsTools:       true,
		},
		Limits: ModelLimits{
			MaxTokens:     8192,
			ContextWindow: 128000,
		},
		Pricing: ModelPricing{
			InputPrice:  0.59, // $0.59 per 1M tokens
			OutputPrice: 0.79, // $0.79 per 1M tokens
		},
		Providers: map[ProviderID]ProviderModelMapping{
			ProviderTogether: {
				ProviderModelID: "meta-llama/Llama-3.3-70B-Instruct-Turbo",
				Available:       true,
				LastChecked:     now,
			},
			ProviderFireworks: {
				ProviderModelID: "accounts/fireworks/models/llama-v3p3-70b-instruct",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouter: {
				ProviderModelID: "meta-llama/llama-3.3-70b-instruct",
				Available:       true,
				LastChecked:     now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Mistral Large model
	mistralLarge := &CanonicalModel{
		ID:      "mistral-large",
		Name:    "Mistral Large",
		Family:  "mistral",
		Version: "large",
		Capabilities: ModelCapabilities{
			SupportsImages:      false,
			SupportsThinking:    false,
			SupportsStreaming:   true,
			SupportsPromptCache: false,
			SupportsTools:       true,
		},
		Limits: ModelLimits{
			MaxTokens:     8192,
			ContextWindow: 128000,
		},
		Pricing: ModelPricing{
			InputPrice:  2.0, // $2.00 per 1M tokens
			OutputPrice: 6.0, // $6.00 per 1M tokens
		},
		Providers: map[ProviderID]ProviderModelMapping{
			ProviderMistral: {
				ProviderModelID: "mistral-large-latest",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouter: {
				ProviderModelID: "mistralai/mistral-large",
				Available:       true,
				LastChecked:     now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Add models to registry
	mr.models[ModelClaudeSonnet4] = claudeSonnet4
	mr.models[ModelGPT4o] = gpt4o
	mr.models[ModelO3Mini] = o3Mini
	mr.models[ModelGemini25Flash] = gemini25Flash
	mr.models["deepseek-r1"] = deepSeekR1
	mr.models["qwen-2.5-coder-32b"] = qwen25Coder
	mr.models["llama-3.3-70b"] = llama33
	mr.models["mistral-large"] = mistralLarge
}
