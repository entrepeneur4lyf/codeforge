package providers

import (
	"fmt"
	"os"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/models"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/transform"
)

// BuildApiHandler creates an API handler based on the provider type
// Based on Cline's buildApiHandler function from api/index.ts
func BuildApiHandler(options llm.ApiHandlerOptions) (llm.ApiHandler, error) {
	// Determine provider type from model ID or explicit provider
	providerType, err := determineProviderType(options)
	if err != nil {
		return nil, fmt.Errorf("failed to determine provider type: %w", err)
	}

	// Get model information from registry
	registry := models.NewModelRegistry()
	if canonicalModel, exists := registry.GetModelByProvider(models.ProviderID(providerType), options.ModelID); exists {
		// Update options with canonical model info
		if options.ModelInfo == nil {
			modelInfo := convertCanonicalToModelInfo(canonicalModel)
			options.ModelInfo = &modelInfo
		}
	}

	// Create handler based on provider type
	var handler llm.ApiHandler
	switch providerType {
	case llm.ProviderAnthropic:
		handler = NewAnthropicSDKHandler(options)
	case llm.ProviderOpenAI:
		handler = NewOpenAISDKHandler(options)
	case llm.ProviderGemini:
		handler = NewGeminiSDKHandler(options)
	case llm.ProviderOpenRouter:
		handler = NewOpenRouterSDKHandler(options)
	case llm.ProviderBedrock:
		handler = NewBedrockSDKHandler(options)
	case llm.ProviderVertex:
		// Use the enhanced Gemini SDK handler for Vertex AI
		handler = NewGeminiSDKHandler(options)
	case llm.ProviderDeepSeek:
		handler = NewDeepSeekHandler(options)
	case llm.ProviderTogether:
		handler = NewTogetherHandler(options)
	case llm.ProviderFireworks:
		handler = NewFireworksHandler(options)
	case llm.ProviderCerebras:
		handler = NewCerebrasHandler(options)
	case llm.ProviderGroq:
		handler = NewGroqHandler(options)
	case llm.ProviderOllama:
		handler = NewOllamaHandler(options)
	case llm.ProviderLMStudio:
		handler = NewLMStudioHandler(options)
	case llm.ProviderXAI:
		handler = NewXAIHandler(options)
	case llm.ProviderMistral:
		handler = NewMistralHandler(options)
	case llm.ProviderQwen:
		handler = NewQwenHandler(options)
	case llm.ProviderDoubao:
		handler = NewDoubaoHandler(options)
	case llm.ProviderSambanova:
		handler = NewSambanovaHandler(options)
	case llm.ProviderNebius:
		handler = NewNebiusHandler(options)
	case llm.ProviderAskSage:
		handler = NewAskSageHandler(options)
	case llm.ProviderSAPAICore:
		handler = NewSAPAICoreHandler(options)
	case llm.ProviderLiteLLM:
		handler = NewLiteLLMHandler(options)
	case llm.ProviderRequesty:
		handler = NewRequestyHandler(options)
	case llm.ProviderClaudeCode:
		handler = NewClaudeCodeHandler(options)
	case llm.ProviderGeminiCLI:
		handler = NewGeminiHandler(options)
	case llm.ProviderGitHub:
		handler = NewGitHubHandler(options)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	// Wrap with retry logic if enabled
	if options.OnRetryAttempt != nil {
		retryHandler := llm.NewRetryHandler(llm.DefaultRetryOptions)
		handler = retryHandler.WrapHandler(handler)
	}

	return handler, nil
}

// BuildApiHandlerWithRetry creates an API handler with retry logic
func BuildApiHandlerWithRetry(options llm.ApiHandlerOptions, retryOptions llm.RetryOptions) (llm.ApiHandler, error) {
	handler, err := BuildApiHandler(options)
	if err != nil {
		return nil, err
	}

	retryHandler := llm.NewRetryHandler(retryOptions)
	return retryHandler.WrapHandler(handler), nil
}

// determineProviderType determines the provider type from options
func determineProviderType(options llm.ApiHandlerOptions) (llm.ProviderType, error) {
	// Check for explicit provider configuration
	if options.AnthropicBaseURL != "" || isAnthropicModel(options.ModelID) {
		return llm.ProviderAnthropic, nil
	}

	if options.OpenAIBaseURL != "" || isOpenAIModel(options.ModelID) {
		return llm.ProviderOpenAI, nil
	}

	if options.GeminiBaseURL != "" || isGeminiModel(options.ModelID) {
		return llm.ProviderGemini, nil
	}

	if options.OpenRouterAPIKey != "" || options.OpenRouterModelID != "" || isOpenRouterModel(options.ModelID) {
		return llm.ProviderOpenRouter, nil
	}

	if options.AWSAccessKey != "" || isBedrock(options.ModelID) {
		return llm.ProviderBedrock, nil
	}

	if options.VertexProjectID != "" || isVertexModel(options.ModelID) {
		return llm.ProviderVertex, nil
	}

	// Check for provider-specific API keys via environment variables
	if os.Getenv("GROQ_API_KEY") != "" || (options.APIKey != "" && strings.Contains(options.ModelID, "groq")) {
		return llm.ProviderGroq, nil
	}

	if isOllamaModel(options.ModelID) || strings.Contains(options.ModelID, "ollama") {
		return llm.ProviderOllama, nil
	}

	if os.Getenv("XAI_API_KEY") != "" || isXAIModel(options.ModelID) {
		return llm.ProviderXAI, nil
	}

	if os.Getenv("MISTRAL_API_KEY") != "" || isMistralModel(options.ModelID) {
		return llm.ProviderMistral, nil
	}

	if os.Getenv("DEEPSEEK_API_KEY") != "" || isDeepSeekModel(options.ModelID) {
		return llm.ProviderDeepSeek, nil
	}

	if options.GitHubOrg != "" || isGitHubModel(options.ModelID) {
		return llm.ProviderGitHub, nil
	}

	// Try to determine from model ID patterns
	if provider := getProviderFromModelID(options.ModelID); provider != "" {
		return provider, nil
	}

	return "", fmt.Errorf("could not determine provider type from options")
}

// isAnthropicModel checks if a model ID belongs to Anthropic
func isAnthropicModel(modelID string) bool {
	anthropicPrefixes := []string{
		"claude-",
		"anthropic.",
		"anthropic/",
	}

	for _, prefix := range anthropicPrefixes {
		if len(modelID) >= len(prefix) && modelID[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// isOpenAIModel checks if a model ID belongs to OpenAI
func isOpenAIModel(modelID string) bool {
	openAIPrefixes := []string{
		"gpt-",
		"o1-",
		"o3-",
		"text-",
		"davinci-",
		"curie-",
		"babbage-",
		"ada-",
	}

	for _, prefix := range openAIPrefixes {
		if len(modelID) >= len(prefix) && modelID[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// isGeminiModel checks if a model ID belongs to Google Gemini
func isGeminiModel(modelID string) bool {
	geminiPrefixes := []string{
		"gemini-",
		"models/gemini-",
	}

	for _, prefix := range geminiPrefixes {
		if len(modelID) >= len(prefix) && modelID[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// isBedrock checks if a model ID is for AWS Bedrock
func isBedrock(modelID string) bool {
	bedrockPrefixes := []string{
		"anthropic.",
		"amazon.",
		"ai21.",
		"cohere.",
		"meta.",
		"mistral.",
	}

	for _, prefix := range bedrockPrefixes {
		if len(modelID) >= len(prefix) && modelID[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// isVertexModel checks if a model ID is for Google Vertex AI
func isVertexModel(modelID string) bool {
	// Vertex models often have @ in the name for versioning
	return len(modelID) > 0 && (modelID[len(modelID)-1:] == "@" ||
		(len(modelID) > 10 && modelID[len(modelID)-10:len(modelID)-8] == "@"))
}

// isGitHubModel checks if a model ID is for GitHub Models
func isGitHubModel(modelID string) bool {
	// GitHub Models use publisher/model format
	githubPrefixes := []string{
		"openai/",
		"microsoft/",
		"meta/",
		"mistralai/",
		"cohere/",
	}

	for _, prefix := range githubPrefixes {
		if len(modelID) >= len(prefix) && modelID[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// isOpenRouterModel checks if a model ID is for OpenRouter
func isOpenRouterModel(modelID string) bool {
	// OpenRouter models use provider/model format
	openRouterPrefixes := []string{
		"anthropic/",
		"openai/",
		"google/",
		"meta-llama/",
		"mistralai/",
		"cohere/",
		"deepseek/",
		"qwen/",
		"01-ai/",
		"microsoft/",
		"nvidia/",
		"huggingfaceh4/",
		"nousresearch/",
		"teknium/",
		"cognitivecomputations/",
	}

	for _, prefix := range openRouterPrefixes {
		if len(modelID) >= len(prefix) && modelID[:len(prefix)] == prefix {
			return true
		}
	}

	return false
}

// isXAIModel checks if a model ID is for xAI (Grok)
func isXAIModel(modelID string) bool {
	return strings.HasPrefix(modelID, "grok-")
}

// isMistralModel checks if a model ID is for Mistral
func isMistralModel(modelID string) bool {
	mistralPrefixes := []string{
		"mistral-",
		"mixtral-",
		"codestral-",
	}

	for _, prefix := range mistralPrefixes {
		if strings.HasPrefix(modelID, prefix) {
			return true
		}
	}
	return false
}

// isDeepSeekModel checks if a model ID is for DeepSeek
func isDeepSeekModel(modelID string) bool {
	return strings.HasPrefix(modelID, "deepseek-")
}

// isOllamaModel checks if a model ID is for Ollama (local models)
func isOllamaModel(modelID string) bool {
	// Ollama models are typically just the model name without provider prefix
	// Common Ollama models
	ollamaModels := []string{
		"llama3", "llama2", "codellama", "mistral", "mixtral", "gemma",
		"qwen", "phi", "llava", "deepseek-coder", "solar", "orca",
		"vicuna", "wizard", "openchat", "starling", "dolphin",
	}

	for _, model := range ollamaModels {
		if strings.Contains(modelID, model) {
			return true
		}
	}
	return false
}

// getProviderFromModelID attempts to determine provider from model ID patterns
func getProviderFromModelID(modelID string) llm.ProviderType {
	// DeepSeek models
	if len(modelID) >= 8 && modelID[:8] == "deepseek" {
		return llm.ProviderDeepSeek
	}

	// Grok models
	if len(modelID) >= 4 && modelID[:4] == "grok" {
		return llm.ProviderXAI
	}

	// Qwen models
	if len(modelID) >= 4 && modelID[:4] == "qwen" {
		return llm.ProviderQwen
	}

	// Mistral models
	if len(modelID) >= 7 && modelID[:7] == "mistral" {
		return llm.ProviderMistral
	}

	// Llama models (often on Together, Fireworks, etc.)
	if len(modelID) >= 5 && modelID[:5] == "llama" {
		return llm.ProviderTogether // Default to Together for Llama
	}

	return ""
}

// convertCanonicalToModelInfo converts canonical model to ModelInfo
func convertCanonicalToModelInfo(canonicalModel *models.CanonicalModel) llm.ModelInfo {
	modelInfo := llm.ModelInfo{
		MaxTokens:           canonicalModel.Limits.MaxTokens,
		ContextWindow:       canonicalModel.Limits.ContextWindow,
		SupportsImages:      canonicalModel.Capabilities.SupportsImages,
		SupportsPromptCache: canonicalModel.Capabilities.SupportsPromptCache,
		InputPrice:          canonicalModel.Pricing.InputPrice,
		OutputPrice:         canonicalModel.Pricing.OutputPrice,
		CacheWritesPrice:    canonicalModel.Pricing.CacheWritesPrice,
		CacheReadsPrice:     canonicalModel.Pricing.CacheReadsPrice,
		Description:         fmt.Sprintf("%s - %s", canonicalModel.Name, canonicalModel.Family),
	}

	// Add thinking config if supported
	if canonicalModel.Capabilities.SupportsThinking {
		modelInfo.ThinkingConfig = &llm.ThinkingConfig{
			MaxBudget:   canonicalModel.Limits.MaxThinkingTokens,
			OutputPrice: canonicalModel.Pricing.ThinkingPrice,
		}
	}

	// Note: Pricing tiers are handled at the canonical model level

	return modelInfo
}

// convertToOpenAIMessages converts LLM messages to OpenAI format (shared helper)
func convertToOpenAIMessages(systemPrompt string, messages []llm.Message) ([]transform.OpenAIMessage, error) {
	var openAIMessages []transform.OpenAIMessage

	// Add system message if provided
	if systemPrompt != "" {
		openAIMessages = append(openAIMessages, transform.CreateSystemMessage(systemPrompt))
	}

	// Convert messages using transform layer
	transformMessages := make([]transform.Message, len(messages))
	for i, msg := range messages {
		transformMessages[i] = transform.Message{
			Role:    msg.Role,
			Content: convertContentBlocks(msg.Content),
		}
	}

	convertedMessages, err := transform.ConvertToOpenAIMessages(transformMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	openAIMessages = append(openAIMessages, convertedMessages...)
	return openAIMessages, nil
}

// Note: convertContentBlocks is defined in github.go and shared across providers
