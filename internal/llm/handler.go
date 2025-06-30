package llm

import (
	"context"
	"fmt"
	"time"
)

// Message represents a conversation message
// Based on Anthropic's message format (Cline's internal standard)
type Message struct {
	Role    string         `json:"role"` // "user", "assistant", "system"
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents different types of content in a message
type ContentBlock interface {
	Type() string
}

// TextBlock represents text content
type TextBlock struct {
	Text string `json:"text"`
}

func (t TextBlock) Type() string { return "text" }

// ImageBlock represents image content
type ImageBlock struct {
	Source ImageSource `json:"source"`
}

func (i ImageBlock) Type() string { return "image" }

// ImageSource represents image data
type ImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/jpeg", "image/png", etc.
	Data      string `json:"data"`       // base64 encoded image data
}

// ToolUseBlock represents a tool call
type ToolUseBlock struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

func (t ToolUseBlock) Type() string { return "tool_use" }

// ToolResultBlock represents a tool result
type ToolResultBlock struct {
	ToolUseID string         `json:"tool_use_id"`
	Content   []ContentBlock `json:"content"`
	IsError   bool           `json:"is_error,omitempty"`
}

func (t ToolResultBlock) Type() string { return "tool_result" }

// ModelInfo represents model capabilities and pricing
// Based on Cline's ModelInfo interface
type ModelInfo struct {
	MaxTokens           int             `json:"maxTokens"`
	ContextWindow       int             `json:"contextWindow"`
	SupportsImages      bool            `json:"supportsImages"`
	SupportsPromptCache bool            `json:"supportsPromptCache"`
	InputPrice          float64         `json:"inputPrice"`       // Per million tokens
	OutputPrice         float64         `json:"outputPrice"`      // Per million tokens
	CacheWritesPrice    float64         `json:"cacheWritesPrice"` // Per million tokens
	CacheReadsPrice     float64         `json:"cacheReadsPrice"`  // Per million tokens
	Description         string          `json:"description,omitempty"`
	ThinkingConfig      *ThinkingConfig `json:"thinkingConfig,omitempty"`

	Temperature        *float64 `json:"temperature,omitempty"`
	IsR1FormatRequired bool     `json:"isR1FormatRequired,omitempty"`
}

// ThinkingConfig represents configuration for reasoning models
type ThinkingConfig struct {
	MaxBudget        int       `json:"maxBudget"`
	OutputPrice      float64   `json:"outputPrice"`
	OutputPriceTiers []float64 `json:"outputPriceTiers,omitempty"`
}

// ApiHandler represents the core interface for LLM providers
// Based on Cline's ApiHandler interface from api/index.ts
type ApiHandler interface {
	// CreateMessage sends a message and returns a streaming response
	CreateMessage(ctx context.Context, systemPrompt string, messages []Message) (ApiStream, error)

	// GetModel returns the model ID and info for the current configuration
	GetModel() ModelResponse

	// GetApiStreamUsage returns usage information if available
	GetApiStreamUsage() (*ApiStreamUsageChunk, error)
}

// ModelResponse represents a model ID and its information
type ModelResponse struct {
	ID   string    `json:"id"`
	Info ModelInfo `json:"info"`
}

// ApiHandlerOptions represents configuration options for API handlers
// Based on Cline's ApiHandlerOptions
type ApiHandlerOptions struct {
	// Core configuration
	APIKey  string `json:"apiKey"`
	ModelID string `json:"modelId"`
	TaskID  string `json:"taskId,omitempty"`

	// Provider-specific URLs
	AnthropicBaseURL string `json:"anthropicBaseUrl,omitempty"`
	OpenAIBaseURL    string `json:"openAiBaseUrl,omitempty"`
	GeminiBaseURL    string `json:"geminiBaseUrl,omitempty"`

	// Headers and authentication
	OpenAIHeaders map[string]string `json:"openAiHeaders,omitempty"`

	// Model configuration
	ModelInfo            *ModelInfo `json:"modelInfo,omitempty"`
	ThinkingBudgetTokens int        `json:"thinkingBudgetTokens,omitempty"`
	ReasoningEffort      string     `json:"reasoningEffort,omitempty"`
	RequestTimeoutMs     int        `json:"requestTimeoutMs,omitempty"`

	// Azure-specific
	AzureAPIVersion string `json:"azureApiVersion,omitempty"`

	// AWS Bedrock-specific
	AWSAccessKey               string `json:"awsAccessKey,omitempty"`
	AWSSecretKey               string `json:"awsSecretKey,omitempty"`
	AWSSessionToken            string `json:"awsSessionToken,omitempty"`
	AWSRegion                  string `json:"awsRegion,omitempty"`
	AWSUseCrossRegionInference bool   `json:"awsUseCrossRegionInference,omitempty"`
	AWSBedrockUsePromptCache   bool   `json:"awsBedrockUsePromptCache,omitempty"`

	// Google Vertex AI-specific
	VertexProjectID string `json:"vertexProjectId,omitempty"`
	VertexRegion    string `json:"vertexRegion,omitempty"`

	// OpenRouter-specific
	OpenRouterAPIKey          string     `json:"openRouterApiKey,omitempty"`
	OpenRouterModelID         string     `json:"openRouterModelId,omitempty"`
	OpenRouterModelInfo       *ModelInfo `json:"openRouterModelInfo,omitempty"`
	OpenRouterProviderSorting string     `json:"openRouterProviderSorting,omitempty"`

	// GitHub Models-specific
	GitHubOrg string `json:"githubOrg,omitempty"`

	// OpenRouter app identification headers
	HTTPReferer string `json:"httpReferer,omitempty"`
	XTitle      string `json:"xTitle,omitempty"`

	// Callbacks
	OnRetryAttempt func(attempt, maxRetries int, delay time.Duration, err error) error `json:"-"`
}

// SingleCompletionHandler represents a simple completion interface
// Based on Cline's SingleCompletionHandler
type SingleCompletionHandler interface {
	CompletePrompt(ctx context.Context, prompt string) (string, error)
}

// RetryOptions represents configuration for retry behavior
// Based on Cline's retry mechanism
type RetryOptions struct {
	MaxRetries     int           `json:"maxRetries"`
	BaseDelay      time.Duration `json:"baseDelay"`
	MaxDelay       time.Duration `json:"maxDelay"`
	RetryAllErrors bool          `json:"retryAllErrors"`
}

// DefaultRetryOptions provides sensible defaults for retry behavior
var DefaultRetryOptions = RetryOptions{
	MaxRetries:     3,
	BaseDelay:      1 * time.Second,
	MaxDelay:       10 * time.Second,
	RetryAllErrors: false,
}

// ProviderType represents different LLM provider types
type ProviderType string

const (
	ProviderAnthropic  ProviderType = "anthropic"
	ProviderOpenAI     ProviderType = "openai"
	ProviderGemini     ProviderType = "gemini"
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderBedrock    ProviderType = "bedrock"
	ProviderVertex     ProviderType = "vertex"
	ProviderDeepSeek   ProviderType = "deepseek"
	ProviderTogether   ProviderType = "together"
	ProviderFireworks  ProviderType = "fireworks"
	ProviderCerebras   ProviderType = "cerebras"
	ProviderGroq       ProviderType = "groq"
	ProviderOllama     ProviderType = "ollama"
	ProviderLMStudio   ProviderType = "lmstudio"
	ProviderXAI        ProviderType = "xai"
	ProviderMistral    ProviderType = "mistral"
	ProviderQwen       ProviderType = "qwen"
	ProviderDoubao     ProviderType = "doubao"
	ProviderSambanova  ProviderType = "sambanova"
	ProviderNebius     ProviderType = "nebius"
	ProviderAskSage    ProviderType = "asksage"
	ProviderSAPAICore  ProviderType = "sapaicore"
	ProviderLiteLLM    ProviderType = "litellm"
	ProviderRequesty   ProviderType = "requesty"
	ProviderClaudeCode ProviderType = "claude-code"
	ProviderGeminiCLI  ProviderType = "gemini-cli"
	ProviderGitHub     ProviderType = "github"
)

// CompletionRequest represents a simple completion request
type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature *float64  `json:"temperature,omitempty"`
}

// CompletionResponse represents a completion response
type CompletionResponse struct {
	Content string `json:"content"`
	Usage   *Usage `json:"usage,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	TotalCost        float64 `json:"total_cost,omitempty"`
}

// Global variables for backward compatibility
var (
	defaultHandler ApiHandler
	initialized    bool
)

// Initialize initializes the LLM package with default configuration
func Initialize(cfg interface{}) error {
	// For now, just mark as initialized
	// Provider setup is handled by the CLI initialization
	initialized = true
	return nil
}

// GetDefaultModel returns a default model configuration
func GetDefaultModel() ModelResponse {
	return ModelResponse{
		ID: "claude-3-5-sonnet-20241022",
		Info: ModelInfo{
			MaxTokens:           8192,
			ContextWindow:       200000,
			SupportsImages:      true,
			SupportsPromptCache: true,
			InputPrice:          3.0,
			OutputPrice:         15.0,
			Description:         "Claude 3.5 Sonnet - Anthropic's most capable model",
		},
	}
}

// DefaultMaxTokens is a backward compatibility field
const DefaultMaxTokens = 8192

// GetCompletion provides a simple completion interface for backward compatibility
func GetCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !initialized {
		return nil, fmt.Errorf("LLM package not initialized")
	}

	if defaultHandler == nil {
		return nil, fmt.Errorf("no default handler configured")
	}

	// Convert to system prompt + messages format
	var systemPrompt string
	var messages []Message

	for _, msg := range req.Messages {
		if msg.Role == "system" {
			// Extract text from system message
			for _, block := range msg.Content {
				if textBlock, ok := block.(TextBlock); ok {
					systemPrompt += textBlock.Text
				}
			}
		} else {
			messages = append(messages, msg)
		}
	}

	// Create streaming request
	stream, err := defaultHandler.CreateMessage(ctx, systemPrompt, messages)
	if err != nil {
		return nil, err
	}

	// Collect all text chunks
	var content string
	var usage *Usage

	for chunk := range stream {
		switch c := chunk.(type) {
		case ApiStreamTextChunk:
			content += c.Text
		case ApiStreamUsageChunk:
			usage = &Usage{
				PromptTokens:     c.InputTokens,
				CompletionTokens: c.OutputTokens,
				TotalTokens:      c.InputTokens + c.OutputTokens,
			}
			if c.TotalCost != nil {
				usage.TotalCost = *c.TotalCost
			}
		}
	}

	return &CompletionResponse{
		Content: content,
		Usage:   usage,
	}, nil
}

// SetDefaultHandler sets the default handler for simple completion requests
func SetDefaultHandler(handler ApiHandler) {
	defaultHandler = handler
}
