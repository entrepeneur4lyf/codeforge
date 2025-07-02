package models

import (
	"fmt"
	"maps"
	"strings"
	"time"
)

// Legacy types for backward compatibility
type (
	ModelID       string
	ModelProvider string
)

// Legacy Model struct for backward compatibility
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

// New canonical model system types
type (
	CanonicalModelID string
	ProviderID       string
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

// ModelPricing represents pricing information
type ModelPricing struct {
	InputPrice       float64 `json:"inputPrice"`       // Per million tokens
	OutputPrice      float64 `json:"outputPrice"`      // Per million tokens
	CacheWritesPrice float64 `json:"cacheWritesPrice"` // Per million tokens
	CacheReadsPrice  float64 `json:"cacheReadsPrice"`  // Per million tokens
	ThinkingPrice    float64 `json:"thinkingPrice"`    // Per million thinking tokens
	Currency         string  `json:"currency"`
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
	RequiresSpecialFormat bool              `json:"requiresSpecialFormat,omitempty"`
	SpecialHeaders        map[string]string `json:"specialHeaders,omitempty"`
	CustomEndpoint        string            `json:"customEndpoint,omitempty"`
	AuthMethod            string            `json:"authMethod,omitempty"`
	ExtraParams           map[string]any    `json:"extraParams,omitempty"`
}

// Legacy provider constants for backward compatibility
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

// New canonical provider constants
const (
	ProviderAnthropicCanonical  ProviderID = "anthropic"
	ProviderOpenAICanonical     ProviderID = "openai"
	ProviderGeminiCanonical     ProviderID = "gemini"
	ProviderOpenRouterCanonical ProviderID = "openrouter"
	ProviderBedrockCanonical    ProviderID = "bedrock"
	ProviderVertexCanonical     ProviderID = "vertex"
	ProviderDeepSeekCanonical   ProviderID = "deepseek"
	ProviderTogetherCanonical   ProviderID = "together"
	ProviderFireworksCanonical  ProviderID = "fireworks"
	ProviderCerebrasCanonical   ProviderID = "cerebras"
	ProviderGroqCanonical       ProviderID = "groq"
	ProviderOllamaCanonical     ProviderID = "ollama"
	ProviderLMStudioCanonical   ProviderID = "lmstudio"
	ProviderXAICanonical        ProviderID = "xai"
	ProviderMistralCanonical    ProviderID = "mistral"
	ProviderQwenCanonical       ProviderID = "qwen"
	ProviderDoubaoCanonical     ProviderID = "doubao"
	ProviderSambanovaCanonical  ProviderID = "sambanova"
	ProviderNebiusCanonical     ProviderID = "nebius"
	ProviderAskSageCanonical    ProviderID = "asksage"
	ProviderSAPAICoreCanonical  ProviderID = "sapaicore"
	ProviderLiteLLMCanonical    ProviderID = "litellm"
	ProviderRequestyCanonical   ProviderID = "requesty"
	ProviderClaudeCodeCanonical ProviderID = "claude-code"
	ProviderGeminiCLICanonical  ProviderID = "gemini-cli"
)

// Canonical model IDs for frontier models
const (
	// Anthropic Models
	ModelClaude35SonnetCanonical CanonicalModelID = "claude-3.5-sonnet"
	ModelClaude3HaikuCanonical   CanonicalModelID = "claude-3-haiku"
	ModelClaude37SonnetCanonical CanonicalModelID = "claude-3.7-sonnet"
	ModelClaude35HaikuCanonical  CanonicalModelID = "claude-3.5-haiku"
	ModelClaude4SonnetCanonical  CanonicalModelID = "claude-4-sonnet"
	ModelClaude4OpusCanonical    CanonicalModelID = "claude-4-opus"

	// OpenAI Models
	ModelGPT41Canonical     CanonicalModelID = "gpt-4.1"
	ModelGPT41MiniCanonical CanonicalModelID = "gpt-4.1-mini"
	ModelGPT41NanoCanonical CanonicalModelID = "gpt-4.1-nano"
	ModelGPT4oCanonical     CanonicalModelID = "gpt-4o"
	ModelGPT4oMiniCanonical CanonicalModelID = "gpt-4o-mini"
	ModelO1Canonical        CanonicalModelID = "o1"
	ModelO1ProCanonical     CanonicalModelID = "o1-pro"
	ModelO3Canonical        CanonicalModelID = "o3"
	ModelO3MiniCanonical    CanonicalModelID = "o3-mini"
	ModelO4MiniCanonical    CanonicalModelID = "o4-mini"

	// Google Models
	ModelGemini25FlashCanonical     CanonicalModelID = "gemini-2.5-flash"
	ModelGemini25Canonical          CanonicalModelID = "gemini-2.5"
	ModelGemini20FlashCanonical     CanonicalModelID = "gemini-2.0-flash"
	ModelGemini20FlashLiteCanonical CanonicalModelID = "gemini-2.0-flash-lite"

	// GROQ Models
	ModelQwenQwqCanonical                CanonicalModelID = "qwen-qwq"
	ModelLlama4ScoutCanonical            CanonicalModelID = "llama-4-scout"
	ModelLlama4MaverickCanonical         CanonicalModelID = "llama-4-maverick"
	ModelLlama33VersatileCanonical       CanonicalModelID = "llama-3.3-70b-versatile"
	ModelDeepseekR1DistillLlamaCanonical CanonicalModelID = "deepseek-r1-distill-llama-70b"

	// XAI Models
	ModelGrok3BetaCanonical         CanonicalModelID = "grok-3-beta"
	ModelGrok3MiniBetaCanonical     CanonicalModelID = "grok-3-mini-beta"
	ModelGrok3FastBetaCanonical     CanonicalModelID = "grok-3-fast-beta"
	ModelGrok3MiniFastBetaCanonical CanonicalModelID = "grok-3-mini-fast-beta"

	// DeepSeek Models
	ModelDeepSeekR1FreeCanonical CanonicalModelID = "deepseek-r1-free"
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
	GPT41     ModelID = "gpt-4.1"
	GPT41Mini ModelID = "gpt-4.1-mini"
	GPT41Nano ModelID = "gpt-4.1-nano"
	GPT4o     ModelID = "gpt-4o"
	GPT4oMini ModelID = "gpt-4o-mini"
	O1        ModelID = "o1"
	O1Pro     ModelID = "o1-pro"
	O3        ModelID = "o3"
	O3Mini    ModelID = "o3-mini"
	O4Mini    ModelID = "o4-mini"
)

var OpenAIModels = map[ModelID]Model{
	GPT41: {
		ID:                  GPT41,
		Name:                "GPT 4.1",
		Provider:            ProviderOpenAI,
		APIModel:            "gpt-4.1",
		CostPer1MIn:         10.0,
		CostPer1MInCached:   5.0,
		CostPer1MOut:        30.0,
		CostPer1MOutCached:  15.0,
		ContextWindow:       1_000_000,
		DefaultMaxTokens:    16384,
		SupportsAttachments: true,
	},
	GPT41Mini: {
		ID:                  GPT41Mini,
		Name:                "GPT 4.1 mini",
		Provider:            ProviderOpenAI,
		APIModel:            "gpt-4.1-mini",
		CostPer1MIn:         1.0,
		CostPer1MInCached:   0.5,
		CostPer1MOut:        4.0,
		CostPer1MOutCached:  2.0,
		ContextWindow:       1_000_000,
		DefaultMaxTokens:    16384,
		SupportsAttachments: true,
	},
	GPT41Nano: {
		ID:                  GPT41Nano,
		Name:                "GPT 4.1 nano",
		Provider:            ProviderOpenAI,
		APIModel:            "gpt-4.1-nano",
		CostPer1MIn:         0.25,
		CostPer1MInCached:   0.125,
		CostPer1MOut:        1.0,
		CostPer1MOutCached:  0.5,
		ContextWindow:       1_000_000,
		DefaultMaxTokens:    16384,
		SupportsAttachments: true,
	},
	GPT4o: {
		ID:                  GPT4o,
		Name:                "GPT 4o",
		Provider:            ProviderOpenAI,
		APIModel:            "gpt-4o",
		CostPer1MIn:         2.50,
		CostPer1MInCached:   1.25,
		CostPer1MOut:        10.00,
		CostPer1MOutCached:  5.0,
		ContextWindow:       128_000,
		DefaultMaxTokens:    16384,
		SupportsAttachments: true,
	},
	GPT4oMini: {
		ID:                  GPT4oMini,
		Name:                "GPT 4o mini",
		Provider:            ProviderOpenAI,
		APIModel:            "gpt-4o-mini",
		CostPer1MIn:         0.15,
		CostPer1MInCached:   0.075,
		CostPer1MOut:        0.60,
		CostPer1MOutCached:  0.30,
		ContextWindow:       128_000,
		DefaultMaxTokens:    16384,
		SupportsAttachments: true,
	},
	O1: {
		ID:                  O1,
		Name:                "O1",
		Provider:            ProviderOpenAI,
		APIModel:            "o1",
		CostPer1MIn:         15.0,
		CostPer1MInCached:   0,
		CostPer1MOut:        60.0,
		CostPer1MOutCached:  0,
		ContextWindow:       200_000,
		DefaultMaxTokens:    100_000,
		CanReason:           true,
		SupportsAttachments: false,
	},
	O1Pro: {
		ID:                  O1Pro,
		Name:                "O1 Pro",
		Provider:            ProviderOpenAI,
		APIModel:            "o1-pro",
		CostPer1MIn:         60.0,
		CostPer1MInCached:   0,
		CostPer1MOut:        240.0,
		CostPer1MOutCached:  0,
		ContextWindow:       200_000,
		DefaultMaxTokens:    100_000,
		CanReason:           true,
		SupportsAttachments: false,
	},
	O3: {
		ID:                  O3,
		Name:                "O3",
		Provider:            ProviderOpenAI,
		APIModel:            "o3",
		CostPer1MIn:         60.0,
		CostPer1MInCached:   0,
		CostPer1MOut:        240.0,
		CostPer1MOutCached:  0,
		ContextWindow:       200_000,
		DefaultMaxTokens:    100_000,
		CanReason:           true,
		SupportsAttachments: false,
	},
	O3Mini: {
		ID:                  O3Mini,
		Name:                "O3 Mini",
		Provider:            ProviderOpenAI,
		APIModel:            "o3-mini-high",
		CostPer1MIn:         3.0,
		CostPer1MInCached:   0,
		CostPer1MOut:        12.0,
		CostPer1MOutCached:  0,
		ContextWindow:       200_000,
		DefaultMaxTokens:    100_000,
		CanReason:           true,
		SupportsAttachments: false,
	},
	O4Mini: {
		ID:                  O4Mini,
		Name:                "O4 Mini",
		Provider:            ProviderOpenAI,
		APIModel:            "o4-mini-high",
		CostPer1MIn:         1.0,
		CostPer1MInCached:   0,
		CostPer1MOut:        4.0,
		CostPer1MOutCached:  0,
		ContextWindow:       128_000,
		DefaultMaxTokens:    16_384,
		CanReason:           true,
		SupportsAttachments: true,
	},
}

// Anthropic Models
const (
	Claude35Sonnet ModelID = "claude-3.5-sonnet"
	Claude3Haiku   ModelID = "claude-3-haiku"
	Claude37Sonnet ModelID = "claude-3.7-sonnet"
	Claude35Haiku  ModelID = "claude-3.5-haiku"
	Claude4Opus    ModelID = "claude-4-opus"
	Claude4Sonnet  ModelID = "claude-4-sonnet"
)

var AnthropicModels = map[ModelID]Model{
	Claude35Sonnet: {
		ID:                  Claude35Sonnet,
		Name:                "Claude 3.5 Sonnet",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-3-5-sonnet-20241022",
		CostPer1MIn:         3.0,
		CostPer1MInCached:   0.30,
		CostPer1MOut:        15.0,
		CostPer1MOutCached:  0,
		ContextWindow:       200_000,
		DefaultMaxTokens:    8192,
		SupportsAttachments: true,
	},
	Claude3Haiku: {
		ID:                  Claude3Haiku,
		Name:                "Claude 3 Haiku",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-3-haiku-20240307",
		CostPer1MIn:         0.25,
		CostPer1MInCached:   0.03,
		CostPer1MOut:        1.25,
		CostPer1MOutCached:  0,
		ContextWindow:       200_000,
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	},
	Claude37Sonnet: {
		ID:                  Claude37Sonnet,
		Name:                "Claude 3.7 Sonnet",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-3-7-sonnet-20250109",
		CostPer1MIn:         3.0,
		CostPer1MInCached:   0.30,
		CostPer1MOut:        15.0,
		CostPer1MOutCached:  0,
		ContextWindow:       200_000,
		DefaultMaxTokens:    8192,
		CanReason:           true,
		SupportsAttachments: true,
	},
	Claude35Haiku: {
		ID:                  Claude35Haiku,
		Name:                "Claude 3.5 Haiku",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-3-5-haiku-20241022",
		CostPer1MIn:         1.0,
		CostPer1MInCached:   0.10,
		CostPer1MOut:        5.0,
		CostPer1MOutCached:  0,
		ContextWindow:       200_000,
		DefaultMaxTokens:    8192,
		SupportsAttachments: true,
	},
	Claude4Opus: {
		ID:                  Claude4Opus,
		Name:                "Claude 4 Opus",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-4-opus-20250109",
		CostPer1MIn:         15.0,
		CostPer1MInCached:   1.50,
		CostPer1MOut:        75.0,
		CostPer1MOutCached:  0,
		ContextWindow:       200_000,
		DefaultMaxTokens:    8192,
		SupportsAttachments: true,
	},
	Claude4Sonnet: {
		ID:                  Claude4Sonnet,
		Name:                "Claude 4 Sonnet",
		Provider:            ProviderAnthropic,
		APIModel:            "claude-4-sonnet-20250109",
		CostPer1MIn:         3.0,
		CostPer1MInCached:   0.30,
		CostPer1MOut:        15.0,
		CostPer1MOutCached:  0,
		ContextWindow:       200_000,
		DefaultMaxTokens:    8192,
		SupportsAttachments: true,
	},
}

// Gemini Models
const (
	Gemini25Flash     ModelID = "gemini-2.5-flash"
	Gemini25          ModelID = "gemini-2.5"
	Gemini20Flash     ModelID = "gemini-2.0-flash"
	Gemini20FlashLite ModelID = "gemini-2.0-flash-lite"
)

var GeminiModels = map[ModelID]Model{
	Gemini25Flash: {
		ID:                  Gemini25Flash,
		Name:                "Gemini 2.5 Flash",
		Provider:            ProviderGemini,
		APIModel:            "gemini-2.5-flash-preview-04-17",
		CostPer1MIn:         0.15,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        0.60,
		ContextWindow:       1_000_000,
		DefaultMaxTokens:    50_000,
		SupportsAttachments: true,
	},
	Gemini25: {
		ID:                  Gemini25,
		Name:                "Gemini 2.5 Pro",
		Provider:            ProviderGemini,
		APIModel:            "gemini-2.5-pro-preview-05-06",
		CostPer1MIn:         1.25,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        10,
		ContextWindow:       1_000_000,
		DefaultMaxTokens:    50_000,
		SupportsAttachments: true,
	},
	Gemini20Flash: {
		ID:                  Gemini20Flash,
		Name:                "Gemini 2.0 Flash",
		Provider:            ProviderGemini,
		APIModel:            "gemini-2.0-flash",
		CostPer1MIn:         0.10,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        0.40,
		ContextWindow:       1_000_000,
		DefaultMaxTokens:    6_000,
		SupportsAttachments: true,
	},
	Gemini20FlashLite: {
		ID:                  Gemini20FlashLite,
		Name:                "Gemini 2.0 Flash Lite",
		Provider:            ProviderGemini,
		APIModel:            "gemini-2.0-flash-lite",
		CostPer1MIn:         0.05,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        0.30,
		ContextWindow:       1_000_000,
		DefaultMaxTokens:    6_000,
		SupportsAttachments: true,
	},
}

// GROQ Models
const (
	QWENQwq                   ModelID = "qwen-qwq"
	Llama4Scout               ModelID = "meta-llama/llama-4-scout-17b-16e-instruct"
	Llama4Maverick            ModelID = "meta-llama/llama-4-maverick-17b-128e-instruct"
	Llama3_3_70BVersatile     ModelID = "llama-3.3-70b-versatile"
	DeepseekR1DistillLlama70b ModelID = "deepseek-r1-distill-llama-70b"
)

var GroqModels = map[ModelID]Model{
	QWENQwq: {
		ID:                  QWENQwq,
		Name:                "Qwen Qwq",
		Provider:            ProviderGROQ,
		APIModel:            "qwen-qwq-32b",
		CostPer1MIn:         0.29,
		CostPer1MInCached:   0.275,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.39,
		ContextWindow:       128_000,
		DefaultMaxTokens:    50_000,
		CanReason:           false, // for some reason, the groq api doesn't like the reasoningEffort parameter
		SupportsAttachments: false,
	},
	Llama4Scout: {
		ID:                  Llama4Scout,
		Name:                "Llama4Scout",
		Provider:            ProviderGROQ,
		APIModel:            "meta-llama/llama-4-scout-17b-16e-instruct",
		CostPer1MIn:         0.11,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        0.34,
		ContextWindow:       128_000,
		SupportsAttachments: true,
	},
	Llama4Maverick: {
		ID:                  Llama4Maverick,
		Name:                "Llama4Maverick",
		Provider:            ProviderGROQ,
		APIModel:            "meta-llama/llama-4-maverick-17b-128e-instruct",
		CostPer1MIn:         0.20,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        0.20,
		ContextWindow:       128_000,
		SupportsAttachments: true,
	},
	Llama3_3_70BVersatile: {
		ID:                  Llama3_3_70BVersatile,
		Name:                "Llama3_3_70BVersatile",
		Provider:            ProviderGROQ,
		APIModel:            "llama-3.3-70b-versatile",
		CostPer1MIn:         0.59,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        0.79,
		ContextWindow:       128_000,
		SupportsAttachments: false,
	},
	DeepseekR1DistillLlama70b: {
		ID:                  DeepseekR1DistillLlama70b,
		Name:                "DeepseekR1DistillLlama70b",
		Provider:            ProviderGROQ,
		APIModel:            "deepseek-r1-distill-llama-70b",
		CostPer1MIn:         0.75,
		CostPer1MInCached:   0,
		CostPer1MOutCached:  0,
		CostPer1MOut:        0.99,
		ContextWindow:       128_000,
		CanReason:           true,
		SupportsAttachments: false,
	},
}

// GitHub Copilot Models
const (
	CopilotGTP35Turbo      ModelID = "copilot.gpt-3.5-turbo"
	CopilotGPT4o           ModelID = "copilot.gpt-4o"
	CopilotGPT4oMini       ModelID = "copilot.gpt-4o-mini"
	CopilotGPT41           ModelID = "copilot.gpt-4.1"
	CopilotClaude35        ModelID = "copilot.claude-3.5-sonnet"
	CopilotClaude37        ModelID = "copilot.claude-3.7-sonnet"
	CopilotClaude4         ModelID = "copilot.claude-sonnet-4"
	CopilotO1              ModelID = "copilot.o1"
	CopilotO3Mini          ModelID = "copilot.o3-mini"
	CopilotO4Mini          ModelID = "copilot.o4-mini"
	CopilotGemini20        ModelID = "copilot.gemini-2.0-flash"
	CopilotGemini25        ModelID = "copilot.gemini-2.5-pro"
	CopilotGPT4            ModelID = "copilot.gpt-4"
	CopilotClaude37Thought ModelID = "copilot.claude-3.7-sonnet-thought"
)

var CopilotModels = map[ModelID]Model{
	CopilotGTP35Turbo: {
		ID:                  CopilotGTP35Turbo,
		Name:                "GitHub Copilot GPT-3.5-turbo",
		Provider:            ProviderCopilot,
		APIModel:            "gpt-3.5-turbo",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       16384,
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	},
	CopilotGPT4o: {
		ID:                  CopilotGPT4o,
		Name:                "GitHub Copilot GPT-4o",
		Provider:            ProviderCopilot,
		APIModel:            "gpt-4o",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       128000,
		DefaultMaxTokens:    16384,
		SupportsAttachments: true,
	},
	CopilotGPT4oMini: {
		ID:                  CopilotGPT4oMini,
		Name:                "GitHub Copilot GPT-4o Mini",
		Provider:            ProviderCopilot,
		APIModel:            "gpt-4o-mini",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       128000,
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	},
	CopilotGPT41: {
		ID:                  CopilotGPT41,
		Name:                "GitHub Copilot GPT-4.1",
		Provider:            ProviderCopilot,
		APIModel:            "gpt-4.1",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       128000,
		DefaultMaxTokens:    16384,
		CanReason:           true,
		SupportsAttachments: true,
	},
	CopilotClaude35: {
		ID:                  CopilotClaude35,
		Name:                "GitHub Copilot Claude 3.5 Sonnet",
		Provider:            ProviderCopilot,
		APIModel:            "claude-3.5-sonnet",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       90000,
		DefaultMaxTokens:    8192,
		SupportsAttachments: true,
	},
	CopilotClaude37: {
		ID:                  CopilotClaude37,
		Name:                "GitHub Copilot Claude 3.7 Sonnet",
		Provider:            ProviderCopilot,
		APIModel:            "claude-3.7-sonnet",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    16384,
		SupportsAttachments: true,
	},
	CopilotClaude4: {
		ID:                  CopilotClaude4,
		Name:                "GitHub Copilot Claude Sonnet 4",
		Provider:            ProviderCopilot,
		APIModel:            "claude-sonnet-4",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       128000,
		DefaultMaxTokens:    16000,
		SupportsAttachments: true,
	},
	CopilotO1: {
		ID:                  CopilotO1,
		Name:                "GitHub Copilot o1",
		Provider:            ProviderCopilot,
		APIModel:            "o1",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    100000,
		CanReason:           true,
		SupportsAttachments: false,
	},
	CopilotO3Mini: {
		ID:                  CopilotO3Mini,
		Name:                "GitHub Copilot o3-mini",
		Provider:            ProviderCopilot,
		APIModel:            "o3-mini",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    100000,
		CanReason:           true,
		SupportsAttachments: false,
	},
	CopilotO4Mini: {
		ID:                  CopilotO4Mini,
		Name:                "GitHub Copilot o4-mini",
		Provider:            ProviderCopilot,
		APIModel:            "o4-mini",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       128000,
		DefaultMaxTokens:    16384,
		CanReason:           true,
		SupportsAttachments: true,
	},
	CopilotGemini20: {
		ID:                  CopilotGemini20,
		Name:                "GitHub Copilot Gemini 2.0 Flash",
		Provider:            ProviderCopilot,
		APIModel:            "gemini-2.0-flash-001",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       1000000,
		DefaultMaxTokens:    8192,
		SupportsAttachments: true,
	},
	CopilotGemini25: {
		ID:                  CopilotGemini25,
		Name:                "GitHub Copilot Gemini 2.5 Pro",
		Provider:            ProviderCopilot,
		APIModel:            "gemini-2.5-pro",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       128000,
		DefaultMaxTokens:    64000,
		SupportsAttachments: true,
	},
	CopilotGPT4: {
		ID:                  CopilotGPT4,
		Name:                "GitHub Copilot GPT-4",
		Provider:            ProviderCopilot,
		APIModel:            "gpt-4",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       32768,
		DefaultMaxTokens:    4096,
		SupportsAttachments: true,
	},
	CopilotClaude37Thought: {
		ID:                  CopilotClaude37Thought,
		Name:                "GitHub Copilot Claude 3.7 Sonnet Thinking",
		Provider:            ProviderCopilot,
		APIModel:            "claude-3.7-sonnet-thought",
		CostPer1MIn:         0.0, // Included in GitHub Copilot subscription
		CostPer1MInCached:   0.0,
		CostPer1MOutCached:  0.0,
		CostPer1MOut:        0.0,
		ContextWindow:       200000,
		DefaultMaxTokens:    16384,
		CanReason:           true,
		SupportsAttachments: true,
	},
}

// OpenRouter Models (proxy to other providers)
const (
	OpenRouterGPT41          ModelID = "openrouter.gpt-4.1"
	OpenRouterGPT41Mini      ModelID = "openrouter.gpt-4.1-mini"
	OpenRouterGPT41Nano      ModelID = "openrouter.gpt-4.1-nano"
	OpenRouterGPT4o          ModelID = "openrouter.gpt-4o"
	OpenRouterGPT4oMini      ModelID = "openrouter.gpt-4o-mini"
	OpenRouterO1             ModelID = "openrouter.o1"
	OpenRouterO1Pro          ModelID = "openrouter.o1-pro"
	OpenRouterO3             ModelID = "openrouter.o3"
	OpenRouterO3Mini         ModelID = "openrouter.o3-mini"
	OpenRouterO4Mini         ModelID = "openrouter.o4-mini"
	OpenRouterGemini25Flash  ModelID = "openrouter.gemini-2.5-flash"
	OpenRouterGemini25       ModelID = "openrouter.gemini-2.5"
	OpenRouterClaude35Sonnet ModelID = "openrouter.claude-3.5-sonnet"
	OpenRouterClaude3Haiku   ModelID = "openrouter.claude-3-haiku"
	OpenRouterClaude37Sonnet ModelID = "openrouter.claude-3.7-sonnet"
	OpenRouterClaude35Haiku  ModelID = "openrouter.claude-3.5-haiku"
	OpenRouterDeepSeekR1Free ModelID = "openrouter.deepseek-r1-free"
)

var OpenRouterModels = map[ModelID]Model{
	OpenRouterGPT41: {
		ID:                  OpenRouterGPT41,
		Name:                "OpenRouter – GPT 4.1",
		Provider:            ProviderOpenRouter,
		APIModel:            "openai/gpt-4.1",
		CostPer1MIn:         OpenAIModels[GPT41].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[GPT41].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[GPT41].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[GPT41].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[GPT41].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[GPT41].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	OpenRouterGPT41Mini: {
		ID:                  OpenRouterGPT41Mini,
		Name:                "OpenRouter – GPT 4.1 mini",
		Provider:            ProviderOpenRouter,
		APIModel:            "openai/gpt-4.1-mini",
		CostPer1MIn:         OpenAIModels[GPT41Mini].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[GPT41Mini].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[GPT41Mini].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[GPT41Mini].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[GPT41Mini].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[GPT41Mini].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	OpenRouterGPT41Nano: {
		ID:                  OpenRouterGPT41Nano,
		Name:                "OpenRouter – GPT 4.1 nano",
		Provider:            ProviderOpenRouter,
		APIModel:            "openai/gpt-4.1-nano",
		CostPer1MIn:         OpenAIModels[GPT41Nano].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[GPT41Nano].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[GPT41Nano].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[GPT41Nano].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[GPT41Nano].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[GPT41Nano].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	OpenRouterGPT4o: {
		ID:                  OpenRouterGPT4o,
		Name:                "OpenRouter – GPT 4o",
		Provider:            ProviderOpenRouter,
		APIModel:            "openai/gpt-4o",
		CostPer1MIn:         OpenAIModels[GPT4o].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[GPT4o].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[GPT4o].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[GPT4o].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[GPT4o].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[GPT4o].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	OpenRouterGPT4oMini: {
		ID:                  OpenRouterGPT4oMini,
		Name:                "OpenRouter – GPT 4o mini",
		Provider:            ProviderOpenRouter,
		APIModel:            "openai/gpt-4o-mini",
		CostPer1MIn:         OpenAIModels[GPT4oMini].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[GPT4oMini].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[GPT4oMini].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[GPT4oMini].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[GPT4oMini].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[GPT4oMini].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	OpenRouterO1: {
		ID:                 OpenRouterO1,
		Name:               "OpenRouter – O1",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/o1",
		CostPer1MIn:        OpenAIModels[O1].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[O1].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[O1].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[O1].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[O1].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[O1].DefaultMaxTokens,
		CanReason:          OpenAIModels[O1].CanReason,
	},
	OpenRouterO1Pro: {
		ID:                 OpenRouterO1Pro,
		Name:               "OpenRouter – o1 pro",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/o1-pro",
		CostPer1MIn:        OpenAIModels[O1Pro].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[O1Pro].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[O1Pro].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[O1Pro].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[O1Pro].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[O1Pro].DefaultMaxTokens,
		CanReason:          OpenAIModels[O1Pro].CanReason,
	},
	OpenRouterO3: {
		ID:                 OpenRouterO3,
		Name:               "OpenRouter – o3",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/o3",
		CostPer1MIn:        OpenAIModels[O3].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[O3].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[O3].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[O3].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[O3].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[O3].DefaultMaxTokens,
		CanReason:          OpenAIModels[O3].CanReason,
	},
	OpenRouterO3Mini: {
		ID:                 OpenRouterO3Mini,
		Name:               "OpenRouter – o3 mini",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/o3-mini-high",
		CostPer1MIn:        OpenAIModels[O3Mini].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[O3Mini].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[O3Mini].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[O3Mini].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[O3Mini].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[O3Mini].DefaultMaxTokens,
		CanReason:          OpenAIModels[O3Mini].CanReason,
	},
	OpenRouterO4Mini: {
		ID:                 OpenRouterO4Mini,
		Name:               "OpenRouter – o4 mini",
		Provider:           ProviderOpenRouter,
		APIModel:           "openai/o4-mini-high",
		CostPer1MIn:        OpenAIModels[O4Mini].CostPer1MIn,
		CostPer1MInCached:  OpenAIModels[O4Mini].CostPer1MInCached,
		CostPer1MOut:       OpenAIModels[O4Mini].CostPer1MOut,
		CostPer1MOutCached: OpenAIModels[O4Mini].CostPer1MOutCached,
		ContextWindow:      OpenAIModels[O4Mini].ContextWindow,
		DefaultMaxTokens:   OpenAIModels[O4Mini].DefaultMaxTokens,
		CanReason:          OpenAIModels[O4Mini].CanReason,
	},
	OpenRouterGemini25Flash: {
		ID:                 OpenRouterGemini25Flash,
		Name:               "OpenRouter – Gemini 2.5 Flash",
		Provider:           ProviderOpenRouter,
		APIModel:           "google/gemini-2.5-flash-preview:thinking",
		CostPer1MIn:        GeminiModels[Gemini25Flash].CostPer1MIn,
		CostPer1MInCached:  GeminiModels[Gemini25Flash].CostPer1MInCached,
		CostPer1MOut:       GeminiModels[Gemini25Flash].CostPer1MOut,
		CostPer1MOutCached: GeminiModels[Gemini25Flash].CostPer1MOutCached,
		ContextWindow:      GeminiModels[Gemini25Flash].ContextWindow,
		DefaultMaxTokens:   GeminiModels[Gemini25Flash].DefaultMaxTokens,
	},
	OpenRouterGemini25: {
		ID:                 OpenRouterGemini25,
		Name:               "OpenRouter – Gemini 2.5 Pro",
		Provider:           ProviderOpenRouter,
		APIModel:           "google/gemini-2.5-pro-preview-03-25",
		CostPer1MIn:        GeminiModels[Gemini25].CostPer1MIn,
		CostPer1MInCached:  GeminiModels[Gemini25].CostPer1MInCached,
		CostPer1MOut:       GeminiModels[Gemini25].CostPer1MOut,
		CostPer1MOutCached: GeminiModels[Gemini25].CostPer1MOutCached,
		ContextWindow:      GeminiModels[Gemini25].ContextWindow,
		DefaultMaxTokens:   GeminiModels[Gemini25].DefaultMaxTokens,
	},
	OpenRouterClaude35Sonnet: {
		ID:                 OpenRouterClaude35Sonnet,
		Name:               "OpenRouter – Claude 3.5 Sonnet",
		Provider:           ProviderOpenRouter,
		APIModel:           "anthropic/claude-3.5-sonnet",
		CostPer1MIn:        AnthropicModels[Claude35Sonnet].CostPer1MIn,
		CostPer1MInCached:  AnthropicModels[Claude35Sonnet].CostPer1MInCached,
		CostPer1MOut:       AnthropicModels[Claude35Sonnet].CostPer1MOut,
		CostPer1MOutCached: AnthropicModels[Claude35Sonnet].CostPer1MOutCached,
		ContextWindow:      AnthropicModels[Claude35Sonnet].ContextWindow,
		DefaultMaxTokens:   AnthropicModels[Claude35Sonnet].DefaultMaxTokens,
	},
	OpenRouterClaude3Haiku: {
		ID:                 OpenRouterClaude3Haiku,
		Name:               "OpenRouter – Claude 3 Haiku",
		Provider:           ProviderOpenRouter,
		APIModel:           "anthropic/claude-3-haiku",
		CostPer1MIn:        AnthropicModels[Claude3Haiku].CostPer1MIn,
		CostPer1MInCached:  AnthropicModels[Claude3Haiku].CostPer1MInCached,
		CostPer1MOut:       AnthropicModels[Claude3Haiku].CostPer1MOut,
		CostPer1MOutCached: AnthropicModels[Claude3Haiku].CostPer1MOutCached,
		ContextWindow:      AnthropicModels[Claude3Haiku].ContextWindow,
		DefaultMaxTokens:   AnthropicModels[Claude3Haiku].DefaultMaxTokens,
	},
	OpenRouterClaude37Sonnet: {
		ID:                 OpenRouterClaude37Sonnet,
		Name:               "OpenRouter – Claude 3.7 Sonnet",
		Provider:           ProviderOpenRouter,
		APIModel:           "anthropic/claude-3.7-sonnet",
		CostPer1MIn:        AnthropicModels[Claude37Sonnet].CostPer1MIn,
		CostPer1MInCached:  AnthropicModels[Claude37Sonnet].CostPer1MInCached,
		CostPer1MOut:       AnthropicModels[Claude37Sonnet].CostPer1MOut,
		CostPer1MOutCached: AnthropicModels[Claude37Sonnet].CostPer1MOutCached,
		ContextWindow:      AnthropicModels[Claude37Sonnet].ContextWindow,
		DefaultMaxTokens:   AnthropicModels[Claude37Sonnet].DefaultMaxTokens,
		CanReason:          AnthropicModels[Claude37Sonnet].CanReason,
	},
	OpenRouterClaude35Haiku: {
		ID:                 OpenRouterClaude35Haiku,
		Name:               "OpenRouter – Claude 3.5 Haiku",
		Provider:           ProviderOpenRouter,
		APIModel:           "anthropic/claude-3.5-haiku",
		CostPer1MIn:        AnthropicModels[Claude35Haiku].CostPer1MIn,
		CostPer1MInCached:  AnthropicModels[Claude35Haiku].CostPer1MInCached,
		CostPer1MOut:       AnthropicModels[Claude35Haiku].CostPer1MOut,
		CostPer1MOutCached: AnthropicModels[Claude35Haiku].CostPer1MOutCached,
		ContextWindow:      AnthropicModels[Claude35Haiku].ContextWindow,
		DefaultMaxTokens:   AnthropicModels[Claude35Haiku].DefaultMaxTokens,
	},
	OpenRouterDeepSeekR1Free: {
		ID:                 OpenRouterDeepSeekR1Free,
		Name:               "OpenRouter – DeepSeek R1 Free",
		Provider:           ProviderOpenRouter,
		APIModel:           "deepseek/deepseek-r1-0528:free",
		CostPer1MIn:        0,
		CostPer1MInCached:  0,
		CostPer1MOut:       0,
		CostPer1MOutCached: 0,
		ContextWindow:      163840,
		DefaultMaxTokens:   10000,
	},
}

// XAI Models
const (
	XAIGrok3Beta         ModelID = "grok-3-beta"
	XAIGrok3MiniBeta     ModelID = "grok-3-mini-beta"
	XAIGrok3FastBeta     ModelID = "grok-3-fast-beta"
	XAiGrok3MiniFastBeta ModelID = "grok-3-mini-fast-beta"
)

var XAIModels = map[ModelID]Model{
	XAIGrok3Beta: {
		ID:                 XAIGrok3Beta,
		Name:               "Grok3 Beta",
		Provider:           ProviderXAI,
		APIModel:           "grok-3-beta",
		CostPer1MIn:        3.0,
		CostPer1MInCached:  0,
		CostPer1MOut:       15,
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
	},
	XAIGrok3MiniBeta: {
		ID:                 XAIGrok3MiniBeta,
		Name:               "Grok3 Mini Beta",
		Provider:           ProviderXAI,
		APIModel:           "grok-3-mini-beta",
		CostPer1MIn:        0.3,
		CostPer1MInCached:  0,
		CostPer1MOut:       0.5,
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
	},
	XAIGrok3FastBeta: {
		ID:                 XAIGrok3FastBeta,
		Name:               "Grok3 Fast Beta",
		Provider:           ProviderXAI,
		APIModel:           "grok-3-fast-beta",
		CostPer1MIn:        5,
		CostPer1MInCached:  0,
		CostPer1MOut:       25,
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
	},
	XAiGrok3MiniFastBeta: {
		ID:                 XAiGrok3MiniFastBeta,
		Name:               "Grok3 Mini Fast Beta",
		Provider:           ProviderXAI,
		APIModel:           "grok-3-mini-fast-beta",
		CostPer1MIn:        0.6,
		CostPer1MInCached:  0,
		CostPer1MOut:       4.0,
		CostPer1MOutCached: 0,
		ContextWindow:      131_072,
		DefaultMaxTokens:   20_000,
	},
}

// Azure Models
const (
	AzureGPT41     ModelID = "azure.gpt-4.1"
	AzureGPT41Mini ModelID = "azure.gpt-4.1-mini"
	AzureGPT41Nano ModelID = "azure.gpt-4.1-nano"
	AzureGPT4o     ModelID = "azure.gpt-4o"
	AzureGPT4oMini ModelID = "azure.gpt-4o-mini"
	AzureO1        ModelID = "azure.o1"
	AzureO3        ModelID = "azure.o3"
	AzureO3Mini    ModelID = "azure.o3-mini"
	AzureO4Mini    ModelID = "azure.o4-mini"
)

var AzureModels = map[ModelID]Model{
	AzureGPT41: {
		ID:                  AzureGPT41,
		Name:                "Azure OpenAI – GPT 4.1",
		Provider:            ProviderAzure,
		APIModel:            "gpt-4.1",
		CostPer1MIn:         OpenAIModels[GPT41].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[GPT41].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[GPT41].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[GPT41].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[GPT41].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[GPT41].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	AzureGPT41Mini: {
		ID:                  AzureGPT41Mini,
		Name:                "Azure OpenAI – GPT 4.1 mini",
		Provider:            ProviderAzure,
		APIModel:            "gpt-4.1-mini",
		CostPer1MIn:         OpenAIModels[GPT41Mini].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[GPT41Mini].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[GPT41Mini].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[GPT41Mini].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[GPT41Mini].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[GPT41Mini].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	AzureGPT41Nano: {
		ID:                  AzureGPT41Nano,
		Name:                "Azure OpenAI – GPT 4.1 nano",
		Provider:            ProviderAzure,
		APIModel:            "gpt-4.1-nano",
		CostPer1MIn:         OpenAIModels[GPT41Nano].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[GPT41Nano].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[GPT41Nano].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[GPT41Nano].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[GPT41Nano].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[GPT41Nano].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	AzureGPT4o: {
		ID:                  AzureGPT4o,
		Name:                "Azure OpenAI – GPT-4o",
		Provider:            ProviderAzure,
		APIModel:            "gpt-4o",
		CostPer1MIn:         OpenAIModels[GPT4o].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[GPT4o].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[GPT4o].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[GPT4o].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[GPT4o].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[GPT4o].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	AzureGPT4oMini: {
		ID:                  AzureGPT4oMini,
		Name:                "Azure OpenAI – GPT-4o mini",
		Provider:            ProviderAzure,
		APIModel:            "gpt-4o-mini",
		CostPer1MIn:         OpenAIModels[GPT4oMini].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[GPT4oMini].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[GPT4oMini].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[GPT4oMini].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[GPT4oMini].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[GPT4oMini].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	AzureO1: {
		ID:                  AzureO1,
		Name:                "Azure OpenAI – O1",
		Provider:            ProviderAzure,
		APIModel:            "o1",
		CostPer1MIn:         OpenAIModels[O1].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[O1].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[O1].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[O1].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[O1].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[O1].DefaultMaxTokens,
		CanReason:           OpenAIModels[O1].CanReason,
		SupportsAttachments: true,
	},
	AzureO3: {
		ID:                  AzureO3,
		Name:                "Azure OpenAI – O3",
		Provider:            ProviderAzure,
		APIModel:            "o3",
		CostPer1MIn:         OpenAIModels[O3].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[O3].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[O3].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[O3].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[O3].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[O3].DefaultMaxTokens,
		CanReason:           OpenAIModels[O3].CanReason,
		SupportsAttachments: true,
	},
	AzureO3Mini: {
		ID:                  AzureO3Mini,
		Name:                "Azure OpenAI – O3 mini",
		Provider:            ProviderAzure,
		APIModel:            "o3-mini",
		CostPer1MIn:         OpenAIModels[O3Mini].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[O3Mini].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[O3Mini].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[O3Mini].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[O3Mini].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[O3Mini].DefaultMaxTokens,
		CanReason:           OpenAIModels[O3Mini].CanReason,
		SupportsAttachments: false,
	},
	AzureO4Mini: {
		ID:                  AzureO4Mini,
		Name:                "Azure OpenAI – O4 mini",
		Provider:            ProviderAzure,
		APIModel:            "o4-mini",
		CostPer1MIn:         OpenAIModels[O4Mini].CostPer1MIn,
		CostPer1MInCached:   OpenAIModels[O4Mini].CostPer1MInCached,
		CostPer1MOut:        OpenAIModels[O4Mini].CostPer1MOut,
		CostPer1MOutCached:  OpenAIModels[O4Mini].CostPer1MOutCached,
		ContextWindow:       OpenAIModels[O4Mini].ContextWindow,
		DefaultMaxTokens:    OpenAIModels[O4Mini].DefaultMaxTokens,
		CanReason:           OpenAIModels[O4Mini].CanReason,
		SupportsAttachments: true,
	},
}

// VertexAI Models
const (
	VertexAIGemini25Flash ModelID = "vertexai.gemini-2.5-flash"
	VertexAIGemini25      ModelID = "vertexai.gemini-2.5"
)

var VertexAIGeminiModels = map[ModelID]Model{
	VertexAIGemini25Flash: {
		ID:                  VertexAIGemini25Flash,
		Name:                "VertexAI: Gemini 2.5 Flash",
		Provider:            ProviderVertexAI,
		APIModel:            "gemini-2.5-flash-preview-04-17",
		CostPer1MIn:         GeminiModels[Gemini25Flash].CostPer1MIn,
		CostPer1MInCached:   GeminiModels[Gemini25Flash].CostPer1MInCached,
		CostPer1MOut:        GeminiModels[Gemini25Flash].CostPer1MOut,
		CostPer1MOutCached:  GeminiModels[Gemini25Flash].CostPer1MOutCached,
		ContextWindow:       GeminiModels[Gemini25Flash].ContextWindow,
		DefaultMaxTokens:    GeminiModels[Gemini25Flash].DefaultMaxTokens,
		SupportsAttachments: true,
	},
	VertexAIGemini25: {
		ID:                  VertexAIGemini25,
		Name:                "VertexAI: Gemini 2.5 Pro",
		Provider:            ProviderVertexAI,
		APIModel:            "gemini-2.5-pro-preview-03-25",
		CostPer1MIn:         GeminiModels[Gemini25].CostPer1MIn,
		CostPer1MInCached:   GeminiModels[Gemini25].CostPer1MInCached,
		CostPer1MOut:        GeminiModels[Gemini25].CostPer1MOut,
		CostPer1MOutCached:  GeminiModels[Gemini25].CostPer1MOutCached,
		ContextWindow:       GeminiModels[Gemini25].ContextWindow,
		DefaultMaxTokens:    GeminiModels[Gemini25].DefaultMaxTokens,
		SupportsAttachments: true,
	},
}

// SupportedModels contains all available models (legacy)
var SupportedModels = map[ModelID]Model{}

// CanonicalModels contains the new canonical model registry
var CanonicalModels = map[CanonicalModelID]*CanonicalModel{}

func init() {
	// Initialize legacy models
	maps.Copy(SupportedModels, OpenAIModels)
	maps.Copy(SupportedModels, AnthropicModels)
	maps.Copy(SupportedModels, GeminiModels)
	maps.Copy(SupportedModels, GroqModels)
	maps.Copy(SupportedModels, CopilotModels)
	maps.Copy(SupportedModels, OpenRouterModels)
	maps.Copy(SupportedModels, XAIModels)
	maps.Copy(SupportedModels, AzureModels)
	maps.Copy(SupportedModels, VertexAIGeminiModels)

	// Initialize canonical models
	initializeCanonicalModels()
}

// initializeCanonicalModels initializes the canonical model registry
func initializeCanonicalModels() {
	now := time.Now()

	// Claude 4 Sonnet - The flagship model
	claudeSonnet4 := &CanonicalModel{
		ID:      ModelClaude4SonnetCanonical,
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
			ProviderAnthropicCanonical: {
				ProviderModelID: "claude-sonnet-4-20250514",
				Available:       true,
				LastChecked:     now,
			},
			ProviderBedrockCanonical: {
				ProviderModelID: "anthropic.claude-sonnet-4-20250514-v1:0",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouterCanonical: {
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
		ID:      ModelGPT4oCanonical,
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
			ProviderOpenAICanonical: {
				ProviderModelID: "gpt-4o",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouterCanonical: {
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
		ID:      ModelO3MiniCanonical,
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
			ProviderOpenAICanonical: {
				ProviderModelID: "o3-mini",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouterCanonical: {
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
		ID:      ModelGemini25FlashCanonical,
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
			ProviderGeminiCanonical: {
				ProviderModelID: "gemini-2.5-flash",
				Available:       true,
				LastChecked:     now,
			},
			ProviderVertexCanonical: {
				ProviderModelID: "gemini-2.5-flash",
				Available:       true,
				LastChecked:     now,
			},
			ProviderOpenRouterCanonical: {
				ProviderModelID: "google/gemini-2.5-flash",
				Available:       true,
				LastChecked:     now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Add models to canonical registry
	CanonicalModels[ModelClaude4SonnetCanonical] = claudeSonnet4
	CanonicalModels[ModelGPT4oCanonical] = gpt4o
	CanonicalModels[ModelO3MiniCanonical] = o3Mini
	CanonicalModels[ModelGemini25FlashCanonical] = gemini25Flash
}

// Legacy functions for backward compatibility

// GetModel returns a model by ID (legacy)
func GetModel(id ModelID) (Model, bool) {
	model, exists := SupportedModels[id]
	return model, exists
}

// GetModelsByProvider returns all models for a specific provider (legacy)
func GetModelsByProvider(provider ModelProvider) []Model {
	var models []Model
	for _, model := range SupportedModels {
		if model.Provider == provider {
			models = append(models, model)
		}
	}
	return models
}

// GetDefaultModelForProvider returns the default model for a provider (legacy)
func GetDefaultModelForProvider(provider ModelProvider) (Model, bool) {
	switch provider {
	case ProviderOpenAI:
		return GetModel(GPT4o)
	case ProviderAnthropic:
		return GetModel(Claude35Sonnet)
	case ProviderGemini:
		return GetModel(Gemini25Flash)
	case ProviderGROQ:
		return GetModel(Llama3_3_70BVersatile)
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

// New canonical model functions

// GetCanonicalModel returns a canonical model by ID
func GetCanonicalModel(id CanonicalModelID) (*CanonicalModel, bool) {
	model, exists := CanonicalModels[id]
	return model, exists
}

// GetCanonicalModelByProvider retrieves a canonical model by provider and provider model ID
func GetCanonicalModelByProvider(providerID ProviderID, providerModelID string) (*CanonicalModel, bool) {
	for _, model := range CanonicalModels {
		if mapping, exists := model.Providers[providerID]; exists {
			if mapping.ProviderModelID == providerModelID {
				return model, true
			}
		}
	}
	return nil, false
}

// ListCanonicalModels returns all canonical models
func ListCanonicalModels() []*CanonicalModel {
	models := make([]*CanonicalModel, 0, len(CanonicalModels))
	for _, model := range CanonicalModels {
		models = append(models, model)
	}
	return models
}

// ListCanonicalModelsByProvider returns all canonical models available for a specific provider
func ListCanonicalModelsByProvider(providerID ProviderID) []*CanonicalModel {
	var models []*CanonicalModel
	for _, model := range CanonicalModels {
		if mapping, exists := model.Providers[providerID]; exists && mapping.Available {
			models = append(models, model)
		}
	}
	return models
}

// GetProviderModelID gets the provider-specific model ID for a canonical model
func GetProviderModelID(modelID CanonicalModelID, providerID ProviderID) (string, error) {
	model, exists := CanonicalModels[modelID]
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

// GetDefaultCanonicalModelForProvider returns the default canonical model for a provider
func GetDefaultCanonicalModelForProvider(provider ProviderID) (*CanonicalModel, bool) {
	switch provider {
	case ProviderOpenAICanonical:
		return GetCanonicalModel(ModelGPT4oCanonical)
	case ProviderAnthropicCanonical:
		return GetCanonicalModel(ModelClaude4SonnetCanonical)
	case ProviderGeminiCanonical:
		return GetCanonicalModel(ModelGemini25FlashCanonical)
	case ProviderOpenRouterCanonical:
		return GetCanonicalModel(ModelClaude4SonnetCanonical)
	default:
		return nil, false
	}
}

// ConvertLegacyToCanonical converts a legacy model to canonical format
func ConvertLegacyToCanonical(legacy Model) *CanonicalModel {
	now := time.Now()

	// Map legacy provider to canonical provider
	var canonicalProvider ProviderID
	switch legacy.Provider {
	case ProviderAnthropic:
		canonicalProvider = ProviderAnthropicCanonical
	case ProviderOpenAI:
		canonicalProvider = ProviderOpenAICanonical
	case ProviderGemini:
		canonicalProvider = ProviderGeminiCanonical
	case ProviderOpenRouter:
		canonicalProvider = ProviderOpenRouterCanonical
	default:
		canonicalProvider = ProviderID(legacy.Provider)
	}

	return &CanonicalModel{
		ID:      CanonicalModelID(legacy.ID),
		Name:    legacy.Name,
		Family:  extractFamily(legacy.Name),
		Version: "1.0",
		Capabilities: ModelCapabilities{
			SupportsImages:      legacy.SupportsAttachments,
			SupportsPromptCache: false,
			SupportsThinking:    legacy.CanReason,
			SupportsTools:       true,
			SupportsStreaming:   true,
			SupportsVision:      legacy.SupportsAttachments,
			SupportsCode:        true,
			SupportsReasoning:   legacy.CanReason,
		},
		Pricing: ModelPricing{
			InputPrice:       legacy.CostPer1MIn,
			OutputPrice:      legacy.CostPer1MOut,
			CacheWritesPrice: legacy.CostPer1MInCached,
			CacheReadsPrice:  legacy.CostPer1MOutCached,
			Currency:         "USD",
		},
		Limits: ModelLimits{
			MaxTokens:          int(legacy.DefaultMaxTokens),
			ContextWindow:      int(legacy.ContextWindow),
			MaxOutputTokens:    int(legacy.DefaultMaxTokens),
			DefaultTemperature: 1.0,
		},
		Providers: map[ProviderID]ProviderModelMapping{
			canonicalProvider: {
				ProviderModelID: legacy.APIModel,
				Available:       true,
				LastChecked:     now,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// extractFamily extracts model family from model name
func extractFamily(name string) string {
	name = strings.ToLower(name)
	if strings.Contains(name, "claude") {
		return "claude"
	}
	if strings.Contains(name, "gpt") {
		return "gpt"
	}
	if strings.Contains(name, "gemini") {
		return "gemini"
	}
	if strings.Contains(name, "llama") {
		return "llama"
	}
	if strings.Contains(name, "o1") || strings.Contains(name, "o3") {
		return strings.Fields(name)[0] // Extract "o1" or "o3"
	}
	return "unknown"
}
