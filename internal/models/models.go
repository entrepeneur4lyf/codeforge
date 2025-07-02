package models

import "maps"

type (
	ModelID       string
	ModelProvider string
)

type Model struct {
	ID                  ModelID       `json:"id"`
	Name                string        `json:"name"`
	Provider            ModelProvider `json:"provider"`
	APIModel            string        `json:"api_model"`
	CostPer1MIn         float64       `json:"cost_per_1m_in"`
	CostPer1MOut        float64       `json:"cost_per_1m_out"`
	CostPer1MInCached   float64       `json:"cost_per_1m_in_cached"`
	CostPer1MOutCached  float64       `json:"cost_per_1m_out_cached"`
	ContextWindow       int64         `json:"context_window"`
	DefaultMaxTokens    int64         `json:"default_max_tokens"`
	CanReason           bool          `json:"can_reason"`
	SupportsAttachments bool          `json:"supports_attachments"`
}

// Provider constants
const (
	ProviderCopilot    ModelProvider = "copilot"
	ProviderAnthropic  ModelProvider = "anthropic"
	ProviderOpenAI     ModelProvider = "openai"
	ProviderGemini     ModelProvider = "gemini"
	ProviderGROQ       ModelProvider = "groq"
	ProviderOpenRouter ModelProvider = "openrouter"
	ProviderBedrock    ModelProvider = "bedrock"
	ProviderAzure      ModelProvider = "azure"
	ProviderVertexAI   ModelProvider = "vertexai"
	ProviderXAI        ModelProvider = "xai"
	ProviderLocal      ModelProvider = "local"
	ProviderMock       ModelProvider = "__mock"
)

// Providers in order of popularity (same as OpenCode)
var ProviderPopularity = map[ModelProvider]int{
	ProviderCopilot:    1,
	ProviderAnthropic:  2,
	ProviderOpenAI:     3,
	ProviderGemini:     4,
	ProviderGROQ:       5,
	ProviderOpenRouter: 6,
	ProviderBedrock:    7,
	ProviderAzure:      8,
	ProviderVertexAI:   9,
	ProviderXAI:        10,
	ProviderLocal:      11,
}

// OpenAI Models
const (
	GPT4o     ModelID = "gpt-4o"
	GPT4oMini ModelID = "gpt-4o-mini"
	GPT41     ModelID = "gpt-4.1"
	GPT41Mini ModelID = "gpt-4.1-mini"
	O1        ModelID = "o1"
	O1Mini    ModelID = "o1-mini"
)

var OpenAIModels = map[ModelID]Model{
	GPT4o: {
		ID:                  GPT4o,
		Name:                "GPT-4o",
		Provider:            ProviderOpenAI,
		APIModel:            "gpt-4o",
		CostPer1MIn:         2.50,
		CostPer1MOut:        10.00,
		ContextWindow:       128000,
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	},
	GPT4oMini: {
		ID:                  GPT4oMini,
		Name:                "GPT-4o mini",
		Provider:            ProviderOpenAI,
		APIModel:            "gpt-4o-mini",
		CostPer1MIn:         0.15,
		CostPer1MOut:        0.60,
		ContextWindow:       128000,
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	},
	O1: {
		ID:                  O1,
		Name:                "O1",
		Provider:            ProviderOpenAI,
		APIModel:            "o1",
		CostPer1MIn:         15.0,
		CostPer1MOut:        60.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    100000,
		CanReason:           true,
		SupportsAttachments: true,
	},
	O1Mini: {
		ID:                  O1Mini,
		Name:                "O1 mini",
		Provider:            ProviderOpenAI,
		APIModel:            "o1-mini",
		CostPer1MIn:         3.0,
		CostPer1MOut:        12.0,
		ContextWindow:       128000,
		DefaultMaxTokens:    65536,
		CanReason:           true,
		SupportsAttachments: true,
	},
}

// Anthropic Models
const (
	ClaudeOpus4    ModelID = "claude-opus-4"
	ClaudeSonnet4  ModelID = "claude-sonnet-4"
	Claude35Sonnet ModelID = "claude-3.5-sonnet"
	Claude35Haiku  ModelID = "claude-3.5-haiku"
	Claude3Haiku   ModelID = "claude-3-haiku"
)

var AnthropicModels = map[ModelID]Model{
	ClaudeOpus4: {
		ID:                  ClaudeOpus4,
		Name:                "Claude Opus 4",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-opus-4-20250514",
		CostPer1MIn:         15.0,
		CostPer1MOut:        75.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    8192,
		SupportsAttachments: true,
	},
	ClaudeSonnet4: {
		ID:                  ClaudeSonnet4,
		Name:                "Claude Sonnet 4",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-sonnet-4-20250514",
		CostPer1MIn:         3.0,
		CostPer1MOut:        15.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    8192,
		SupportsAttachments: true,
	},
	Claude35Sonnet: {
		ID:                  Claude35Sonnet,
		Name:                "Claude 3.5 Sonnet",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-3-5-sonnet-latest",
		CostPer1MIn:         3.0,
		CostPer1MOut:        15.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    5000,
		SupportsAttachments: true,
	},
	Claude35Haiku: {
		ID:                  Claude35Haiku,
		Name:                "Claude 3.5 Haiku",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-3-5-haiku-latest",
		CostPer1MIn:         1.0,
		CostPer1MOut:        5.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	},
}

// Gemini Models
const (
	Gemini25Flash ModelID = "gemini-2.5-flash"
	Gemini20Flash ModelID = "gemini-2.0-flash"
)

var GeminiModels = map[ModelID]Model{
	Gemini25Flash: {
		ID:                  Gemini25Flash,
		Name:                "Gemini 2.5 Flash",
		Provider:            ProviderGemini,
		APIModel:            "gemini-2.5-flash-preview-04-17",
		CostPer1MIn:         0.15,
		CostPer1MOut:        0.60,
		ContextWindow:       1000000,
		DefaultMaxTokens:    50000,
		SupportsAttachments: true,
	},
	Gemini20Flash: {
		ID:                  Gemini20Flash,
		Name:                "Gemini 2.0 Flash",
		Provider:            ProviderGemini,
		APIModel:            "gemini-2.0-flash",
		CostPer1MIn:         0.10,
		CostPer1MOut:        0.40,
		ContextWindow:       1000000,
		DefaultMaxTokens:    6000,
		SupportsAttachments: true,
	},
}

// Groq Models
const (
	Llama33_70BVersatile ModelID = "llama-3.3-70b-versatile"
	QWENQwq              ModelID = "qwen-qwq"
)

var GroqModels = map[ModelID]Model{
	Llama33_70BVersatile: {
		ID:               Llama33_70BVersatile,
		Name:             "Llama 3.3 70B Versatile",
		Provider:         ProviderGROQ,
		APIModel:         "llama-3.3-70b-versatile",
		CostPer1MIn:      0.59,
		CostPer1MOut:     0.79,
		ContextWindow:    128000,
		DefaultMaxTokens: 8192,
	},
	QWENQwq: {
		ID:               QWENQwq,
		Name:             "Qwen QwQ",
		Provider:         ProviderGROQ,
		APIModel:         "qwen-qwq-32b",
		CostPer1MIn:      0.29,
		CostPer1MOut:     0.39,
		ContextWindow:    128000,
		DefaultMaxTokens: 50000,
		CanReason:        true,
	},
}

// GitHub Copilot Models
const (
	CopilotGPT4o ModelID = "copilot-gpt-4o"
)

var CopilotModels = map[ModelID]Model{
	CopilotGPT4o: {
		ID:                  CopilotGPT4o,
		Name:                "GitHub Copilot GPT-4o",
		Provider:            ProviderCopilot,
		APIModel:            "gpt-4o",
		CostPer1MIn:         0, // Included in Copilot subscription
		CostPer1MOut:        0,
		ContextWindow:       128000,
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	},
}

// OpenRouter Models (proxy to other providers)
const (
	OpenRouterClaude35Sonnet ModelID = "openrouter-claude-3.5-sonnet"
	OpenRouterGPT4o          ModelID = "openrouter-gpt-4o"
)

var OpenRouterModels = map[ModelID]Model{
	OpenRouterClaude35Sonnet: {
		ID:                  OpenRouterClaude35Sonnet,
		Name:                "OpenRouter – Claude 3.5 Sonnet",
		Provider:            ProviderOpenRouter,
		APIModel:            "anthropic/claude-3.5-sonnet",
		CostPer1MIn:         3.0,
		CostPer1MOut:        15.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    5000,
		SupportsAttachments: true,
	},
	OpenRouterGPT4o: {
		ID:                  OpenRouterGPT4o,
		Name:                "OpenRouter – GPT-4o",
		Provider:            ProviderOpenRouter,
		APIModel:            "openai/gpt-4o",
		CostPer1MIn:         2.50,
		CostPer1MOut:        10.00,
		ContextWindow:       128000,
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	},
}

// XAI Models
const (
	XAIGrok3Beta ModelID = "xai-grok-3-beta"
)

var XAIModels = map[ModelID]Model{
	XAIGrok3Beta: {
		ID:               XAIGrok3Beta,
		Name:             "Grok 3 Beta",
		Provider:         ProviderXAI,
		APIModel:         "grok-3-beta",
		CostPer1MIn:      3.0,
		CostPer1MOut:     15.0,
		ContextWindow:    131072,
		DefaultMaxTokens: 20000,
	},
}

// SupportedModels contains all available models
var SupportedModels = map[ModelID]Model{}

func init() {
	maps.Copy(SupportedModels, OpenAIModels)
	maps.Copy(SupportedModels, AnthropicModels)
	maps.Copy(SupportedModels, GeminiModels)
	maps.Copy(SupportedModels, GroqModels)
	maps.Copy(SupportedModels, CopilotModels)
	maps.Copy(SupportedModels, OpenRouterModels)
	maps.Copy(SupportedModels, XAIModels)
}

// GetModel returns a model by ID
func GetModel(id ModelID) (Model, bool) {
	model, exists := SupportedModels[id]
	return model, exists
}

// GetModelsByProvider returns all models for a specific provider
func GetModelsByProvider(provider ModelProvider) []Model {
	var models []Model
	for _, model := range SupportedModels {
		if model.Provider == provider {
			models = append(models, model)
		}
	}
	return models
}

// GetDefaultModelForProvider returns the default model for a provider
func GetDefaultModelForProvider(provider ModelProvider) (Model, bool) {
	switch provider {
	case ProviderOpenAI:
		return GetModel(GPT4o)
	case ProviderAnthropic:
		return GetModel(Claude35Sonnet)
	case ProviderGemini:
		return GetModel(Gemini25Flash)
	case ProviderGROQ:
		return GetModel(Llama33_70BVersatile)
	case ProviderCopilot:
		return GetModel(CopilotGPT4o)
	case ProviderOpenRouter:
		return GetModel(OpenRouterClaude35Sonnet)
	case ProviderXAI:
		return GetModel(XAIGrok3Beta)
	default:
		return Model{}, false
	}
}
