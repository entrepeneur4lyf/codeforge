package providers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/models"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/transform"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

// OpenRouterModelsCache represents cached model data with TTL
type OpenRouterModelsCache struct {
	models    []OpenRouterModel
	timestamp time.Time
	mutex     sync.RWMutex
}

// Global cache instance with 24-hour TTL
var (
	modelsCache = &OpenRouterModelsCache{}
	cacheTTL    = 24 * time.Hour // 86400 seconds
)

// OpenRouterHandler implements the ApiHandler interface for OpenRouter's unified API
// Provides access to 100+ models from 50+ providers with intelligent routing
type OpenRouterHandler struct {
	options llm.ApiHandlerOptions
	client  *http.Client
	baseURL string
	db      *vectordb.VectorDB // Database connection for model storage
}

// OpenRouterRequest represents the request to OpenRouter API
type OpenRouterRequest struct {
	Model         string                    `json:"model"`
	Messages      []transform.OpenAIMessage `json:"messages"`
	MaxTokens     *int                      `json:"max_tokens,omitempty"`
	Temperature   *float64                  `json:"temperature,omitempty"`
	Stream        bool                      `json:"stream"`
	StreamOptions *OpenRouterStreamOptions  `json:"stream_options,omitempty"`
	Tools         []OpenRouterTool          `json:"tools,omitempty"`
	ToolChoice    interface{}               `json:"tool_choice,omitempty"`

	// OpenRouter-specific parameters
	Models     []string                 `json:"models,omitempty"`     // Model fallback list
	Route      string                   `json:"route,omitempty"`      // "fallback" for automatic fallback
	Provider   *OpenRouterProviderPrefs `json:"provider,omitempty"`   // Provider preferences
	Transforms []string                 `json:"transforms,omitempty"` // Message transforms
	User       string                   `json:"user,omitempty"`       // User identifier for abuse detection

	// Standard parameters
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
	TopP             *float64 `json:"top_p,omitempty"`
	TopK             *int     `json:"top_k,omitempty"`
	Stop             []string `json:"stop,omitempty"`
	Seed             *int     `json:"seed,omitempty"`
}

// OpenRouterStreamOptions configures streaming behavior
type OpenRouterStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

// OpenRouterProviderPrefs represents provider routing preferences
type OpenRouterProviderPrefs struct {
	AllowFallbacks    bool     `json:"allow_fallbacks,omitempty"`
	RequireParameters bool     `json:"require_parameters,omitempty"`
	DataCollection    string   `json:"data_collection,omitempty"` // "deny" or "allow"
	Order             []string `json:"order,omitempty"`           // Provider preference order
}

// OpenRouterTool represents a tool definition
type OpenRouterTool struct {
	Type     string                `json:"type"`
	Function OpenRouterFunctionDef `json:"function"`
}

// OpenRouterFunctionDef represents a function definition
type OpenRouterFunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// OpenRouterStreamEvent represents a streaming event from OpenRouter
type OpenRouterStreamEvent struct {
	ID      string                   `json:"id"`
	Object  string                   `json:"object"`
	Created int64                    `json:"created"`
	Model   string                   `json:"model"`
	Choices []OpenRouterStreamChoice `json:"choices"`
	Usage   *OpenRouterUsage         `json:"usage,omitempty"`
}

// OpenRouterStreamChoice represents a choice in the stream
type OpenRouterStreamChoice struct {
	Index        int                   `json:"index"`
	Delta        OpenRouterStreamDelta `json:"delta"`
	FinishReason *string               `json:"finish_reason"`
	Error        *OpenRouterError      `json:"error,omitempty"`
}

// OpenRouterStreamDelta represents delta content
type OpenRouterStreamDelta struct {
	Role      string                     `json:"role,omitempty"`
	Content   string                     `json:"content,omitempty"`
	ToolCalls []transform.OpenAIToolCall `json:"tool_calls,omitempty"`
}

// OpenRouterUsage represents token usage with cost information
type OpenRouterUsage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	TotalCost        float64 `json:"total_cost,omitempty"` // OpenRouter provides cost directly
}

// OpenRouterError represents an error in the response
type OpenRouterError struct {
	Code     int                    `json:"code"`
	Message  string                 `json:"message"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewOpenRouterHandler creates a new OpenRouter handler
func NewOpenRouterHandler(options llm.ApiHandlerOptions) *OpenRouterHandler {
	baseURL := "https://openrouter.ai/api/v1"

	// Configure timeout
	timeout := 60 * time.Second
	if options.RequestTimeoutMs > 0 {
		timeout = time.Duration(options.RequestTimeoutMs) * time.Millisecond
	}

	return &OpenRouterHandler{
		options: options,
		client:  &http.Client{Timeout: timeout},
		baseURL: baseURL,
		db:      nil, // Will be set when database operations are needed
	}
}

// NewOpenRouterHandlerWithDB creates a new OpenRouter handler with database connection
func NewOpenRouterHandlerWithDB(options llm.ApiHandlerOptions, db *vectordb.VectorDB) *OpenRouterHandler {
	handler := NewOpenRouterHandler(options)
	handler.db = db
	return handler
}

// CreateMessage implements the ApiHandler interface
func (h *OpenRouterHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	model := h.GetModel()

	// Convert messages to OpenAI format (OpenRouter uses OpenAI-compatible format)
	openAIMessages, err := h.convertMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Prepare request
	request := OpenRouterRequest{
		Model:    model.ID,
		Messages: openAIMessages,
		Stream:   true,
		StreamOptions: &OpenRouterStreamOptions{
			IncludeUsage: true,
		},
	}

	// Set max tokens if specified
	if model.Info.MaxTokens > 0 {
		request.MaxTokens = &model.Info.MaxTokens
	}

	// Set temperature if specified
	if model.Info.Temperature != nil {
		request.Temperature = model.Info.Temperature
	}

	// Add OpenRouter-specific options
	if h.options.OpenRouterProviderSorting != "" {
		request.Provider = &OpenRouterProviderPrefs{
			Order:             []string{h.options.OpenRouterProviderSorting},
			AllowFallbacks:    true,
			RequireParameters: false,
			DataCollection:    "deny", // Privacy-focused default
		}
	}

	// Enable fallback routing for reliability
	request.Route = "fallback"

	// Add user identifier if available
	if h.options.TaskID != "" {
		request.User = h.options.TaskID
	}

	return h.streamRequest(ctx, request)
}

// GetModel implements the ApiHandler interface
func (h *OpenRouterHandler) GetModel() llm.ModelResponse {
	// Use OpenRouter model ID if specified, otherwise use regular model ID
	modelID := h.options.ModelID
	if h.options.OpenRouterModelID != "" {
		modelID = h.options.OpenRouterModelID
	}

	// Try to get model from registry first
	registry := models.NewModelRegistry()
	if canonicalModel, exists := registry.GetModelByProvider(models.ProviderOpenRouterCanonical, modelID); exists {
		return llm.ModelResponse{
			ID:   modelID,
			Info: h.convertToLLMModelInfo(canonicalModel),
		}
	}

	// Use OpenRouter model info if provided
	if h.options.OpenRouterModelInfo != nil {
		return llm.ModelResponse{
			ID:   modelID,
			Info: *h.options.OpenRouterModelInfo,
		}
	}

	// Fallback to default model info based on model type
	return llm.ModelResponse{
		ID:   modelID,
		Info: h.getDefaultModelInfo(modelID),
	}
}

// GetApiStreamUsage implements the ApiHandler interface
func (h *OpenRouterHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// OpenRouter includes usage in the stream, so this is not needed
	return nil, nil
}

// convertMessages converts LLM messages to OpenAI format for OpenRouter
func (h *OpenRouterHandler) convertMessages(systemPrompt string, messages []llm.Message) ([]transform.OpenAIMessage, error) {
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
			Content: convertContentBlocksOpenRouter(msg.Content),
		}
	}

	convertedMessages, err := transform.ConvertToOpenAIMessages(transformMessages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	openAIMessages = append(openAIMessages, convertedMessages...)
	return openAIMessages, nil
}

// convertContentBlocksOpenRouter converts llm.ContentBlock to transform.ContentBlock
func convertContentBlocksOpenRouter(blocks []llm.ContentBlock) []transform.ContentBlock {
	result := make([]transform.ContentBlock, len(blocks))
	for i, block := range blocks {
		switch b := block.(type) {
		case llm.TextBlock:
			result[i] = transform.TextBlock{Text: b.Text}
		case llm.ImageBlock:
			result[i] = transform.ImageBlock{
				Source: transform.ImageSource{
					Type:      b.Source.Type,
					MediaType: b.Source.MediaType,
					Data:      b.Source.Data,
				},
			}
		case llm.ToolUseBlock:
			result[i] = transform.ToolUseBlock{
				ID:    b.ID,
				Name:  b.Name,
				Input: b.Input,
			}
		case llm.ToolResultBlock:
			// Convert ToolResultBlock content to string
			var content string
			for _, contentBlock := range b.Content {
				if textBlock, ok := contentBlock.(llm.TextBlock); ok {
					content += textBlock.Text
				}
			}
			result[i] = transform.ToolResultBlock{
				ToolUseID: b.ToolUseID,
				Content:   content,
				IsError:   b.IsError,
			}
		default:
			// Fallback to text block
			result[i] = transform.TextBlock{Text: fmt.Sprintf("%v", block)}
		}
	}
	return result
}

// getDefaultModelInfo provides default model info based on model ID
func (h *OpenRouterHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default values for OpenRouter models
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       128000,
		SupportsImages:      false,
		SupportsPromptCache: false,
		InputPrice:          0.0, // Will be determined by OpenRouter's dynamic pricing
		OutputPrice:         0.0, // Will be determined by OpenRouter's dynamic pricing
		Temperature:         &[]float64{1.0}[0],
		Description:         "Model via OpenRouter (100+ models, 50+ providers)",
	}

	// Model-specific configurations based on common OpenRouter models
	switch {
	case strings.Contains(modelID, "claude"):
		info.SupportsImages = true
		info.SupportsPromptCache = true
		info.ContextWindow = 200000
		info.MaxTokens = 8192

	case strings.Contains(modelID, "gpt-4"):
		info.SupportsImages = strings.Contains(modelID, "vision") || strings.Contains(modelID, "4o")
		info.ContextWindow = 128000
		info.MaxTokens = 4096

	case strings.Contains(modelID, "gemini"):
		info.SupportsImages = true
		info.ContextWindow = 1000000
		info.MaxTokens = 8192

	case strings.Contains(modelID, "llama"):
		info.ContextWindow = 131072
		info.MaxTokens = 8192

	case strings.Contains(modelID, "mixtral"):
		info.ContextWindow = 32768
		info.MaxTokens = 4096
	}

	return info
}

// convertToLLMModelInfo converts canonical model to LLM model info
func (h *OpenRouterHandler) convertToLLMModelInfo(canonicalModel *models.CanonicalModel) llm.ModelInfo {
	return llm.ModelInfo{
		MaxTokens:           canonicalModel.Limits.MaxTokens,
		ContextWindow:       canonicalModel.Limits.ContextWindow,
		SupportsImages:      canonicalModel.Capabilities.SupportsImages,
		SupportsPromptCache: canonicalModel.Capabilities.SupportsPromptCache,
		InputPrice:          canonicalModel.Pricing.InputPrice,
		OutputPrice:         canonicalModel.Pricing.OutputPrice,
		CacheWritesPrice:    canonicalModel.Pricing.CacheWritesPrice,
		CacheReadsPrice:     canonicalModel.Pricing.CacheReadsPrice,
		Description:         fmt.Sprintf("%s - %s (via OpenRouter)", canonicalModel.Name, canonicalModel.Family),
		Temperature:         &canonicalModel.Limits.DefaultTemperature,
	}
}

// streamRequest makes a streaming request to the OpenRouter API
func (h *OpenRouterHandler) streamRequest(ctx context.Context, request OpenRouterRequest) (llm.ApiStream, error) {
	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", h.baseURL+"/chat/completions", bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers - OpenRouter has specific header requirements
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+h.options.OpenRouterAPIKey)

	// Optional headers for app identification and ranking
	if h.options.HTTPReferer != "" {
		req.Header.Set("HTTP-Referer", h.options.HTTPReferer)
	}
	if h.options.XTitle != "" {
		req.Header.Set("X-Title", h.options.XTitle)
	}

	// Make request
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, llm.WrapHTTPError(fmt.Errorf("request failed: %w", err), resp)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, llm.WrapHTTPError(fmt.Errorf("API error %d: %s", resp.StatusCode, string(body)), resp)
	}

	// Create stream channel
	streamChan := make(chan llm.ApiStreamChunk, 100)

	// Start streaming goroutine
	go func() {
		defer close(streamChan)
		defer resp.Body.Close()

		h.processStream(resp.Body, streamChan)
	}()

	return streamChan, nil
}

// processStream processes the streaming response from OpenRouter
func (h *OpenRouterHandler) processStream(reader io.Reader, streamChan chan<- llm.ApiStreamChunk) {
	scanner := NewSSEScanner(reader)

	for scanner.Scan() {
		event := scanner.Event()

		// Skip non-data events and comments
		if event.Type != "data" {
			continue
		}

		// Handle [DONE] marker
		if strings.TrimSpace(event.Data) == "[DONE]" {
			break
		}

		// Parse the event data
		var streamEvent OpenRouterStreamEvent
		if err := json.Unmarshal([]byte(event.Data), &streamEvent); err != nil {
			continue // Skip malformed events
		}

		// Process choices
		for _, choice := range streamEvent.Choices {
			// Handle errors in choice
			if choice.Error != nil {
				// Log error but continue processing
				continue
			}

			// Handle content delta
			if choice.Delta.Content != "" {
				streamChan <- llm.ApiStreamTextChunk{Text: choice.Delta.Content}
			}
		}

		// Handle usage information with OpenRouter's cost data
		if streamEvent.Usage != nil {
			usage := llm.ApiStreamUsageChunk{
				InputTokens:  streamEvent.Usage.PromptTokens,
				OutputTokens: streamEvent.Usage.CompletionTokens,
			}

			// OpenRouter provides direct cost information
			if streamEvent.Usage.TotalCost > 0 {
				usage.TotalCost = &streamEvent.Usage.TotalCost
			}

			streamChan <- usage
		}
	}
}

// getCachedModels returns cached models if valid
func (c *OpenRouterModelsCache) getCachedModels() ([]OpenRouterModel, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if time.Since(c.timestamp) < cacheTTL && len(c.models) > 0 {
		return c.models, true
	}
	return nil, false
}

// setCachedModels stores models in cache
func (c *OpenRouterModelsCache) setCachedModels(models []OpenRouterModel) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.models = models
	c.timestamp = time.Now()
}

// GetOpenRouterModels fetches available models from OpenRouter with database caching
func (h *OpenRouterHandler) GetOpenRouterModels(ctx context.Context) ([]OpenRouterModel, error) {
	// First, try to get models from database cache
	if models, err := h.getModelsFromDatabase(ctx); err == nil && len(models) > 0 {
		// Check if cache is still valid (24 hours TTL)
		if h.isDatabaseCacheValid(ctx) {
			return models, nil
		}
	}

	// Cache expired or empty, fetch fresh data and store in database
	return h.refreshModelsInDatabase(ctx)
}

// getModelsFromDatabase retrieves cached models from database
func (h *OpenRouterHandler) getModelsFromDatabase(ctx context.Context) ([]OpenRouterModel, error) {
	if h.db == nil {
		// Fall back to memory cache if no database
		if cachedModels, valid := modelsCache.getCachedModels(); valid {
			return cachedModels, nil
		}
		return nil, fmt.Errorf("no database connection available")
	}

	// Query from openrouter_models table
	query := `
		SELECT model_id, name, description, context_length, created_date, last_updated
		FROM openrouter_models
		WHERE last_updated > datetime('now', '-24 hours')
		ORDER BY name
	`

	rows, err := h.db.QueryContext(ctx, query)
	if err != nil {
		// Fall back to memory cache on database error
		if cachedModels, valid := modelsCache.getCachedModels(); valid {
			return cachedModels, nil
		}
		return nil, fmt.Errorf("failed to query models from database: %w", err)
	}
	defer rows.Close()

	var models []OpenRouterModel
	for rows.Next() {
		var model OpenRouterModel
		var lastUpdated string

		err := rows.Scan(
			&model.ID, &model.Name, &model.Description,
			&model.ContextLength, &model.Created, &lastUpdated,
		)
		if err != nil {
			continue // Skip invalid rows
		}

		models = append(models, model)
	}

	if len(models) == 0 {
		// Fall back to memory cache if no database results
		if cachedModels, valid := modelsCache.getCachedModels(); valid {
			return cachedModels, nil
		}
		return nil, fmt.Errorf("no models found in database")
	}

	return models, nil
}

// getModelsFromDatabaseByProvider retrieves models from database filtered by provider
func (h *OpenRouterHandler) getModelsFromDatabaseByProvider(ctx context.Context, providerFilter string) ([]OpenRouterModel, error) {
	if h.db == nil {
		return nil, fmt.Errorf("no database connection available")
	}

	// Query from openrouter_models table with provider filter
	var query string
	var args []interface{}

	if providerFilter == "" {
		// No filter - get all models
		query = `
			SELECT model_id, name, description, context_length, created_date, last_seen
			FROM openrouter_models
			WHERE last_seen > datetime('now', '-24 hours')
			ORDER BY name
		`
	} else {
		// Filter by specific provider
		query = `
			SELECT model_id, name, description, context_length, created_date, last_seen
			FROM openrouter_models
			WHERE provider = ? AND last_seen > datetime('now', '-24 hours')
			ORDER BY name
		`
		args = append(args, providerFilter)
	}

	rows, err := h.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query models from database: %w", err)
	}
	defer rows.Close()

	var models []OpenRouterModel
	for rows.Next() {
		var model OpenRouterModel
		var lastSeen string

		err := rows.Scan(
			&model.ID, &model.Name, &model.Description,
			&model.ContextLength, &model.Created, &lastSeen,
		)
		if err != nil {
			continue // Skip invalid rows
		}

		models = append(models, model)
	}

	return models, nil
}

// isDatabaseCacheValid checks if the database cache is still valid (24 hour TTL)
func (h *OpenRouterHandler) isDatabaseCacheValid(ctx context.Context) bool {
	if h.db == nil {
		// Fallback to memory cache timestamp
		modelsCache.mutex.RLock()
		defer modelsCache.mutex.RUnlock()
		cacheTTL := 24 * time.Hour
		return time.Since(modelsCache.timestamp) < cacheTTL
	}

	// Check database timestamp for TTL enforcement
	var lastUpdate time.Time
	query := `
		SELECT MAX(last_seen) as last_update
		FROM openrouter_models
		WHERE last_seen IS NOT NULL
	`

	row := h.db.QueryRowContext(ctx, query)
	if err := row.Scan(&lastUpdate); err != nil {
		// If no models in database, cache is invalid
		return false
	}

	cacheTTL := 24 * time.Hour
	isValid := time.Since(lastUpdate) < cacheTTL

	if !isValid {
		fmt.Printf("🕒 Database cache expired (last update: %v)\n", lastUpdate.Format("2006-01-02 15:04:05"))
	}

	return isValid
}

// refreshModelsInDatabase fetches fresh models from API and stores in database
func (h *OpenRouterHandler) refreshModelsInDatabase(ctx context.Context) ([]OpenRouterModel, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", h.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+h.options.OpenRouterAPIKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var response OpenRouterModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Filter out non-coding models (audio, video, image, etc.)
	var filteredModels []OpenRouterModel
	for _, model := range response.Data {
		if h.isCodeGenerationModel(model.ID) {
			filteredModels = append(filteredModels, model)
		}
	}
	response.Data = filteredModels

	// EFFICIENT APPROACH: Store lightweight model list only
	// Metadata will be fetched on-demand when users actually need it

	// Sort models by release date DESC (newest first) - using basic data
	sort.Slice(response.Data, func(i, j int) bool {
		return parseModelReleaseDate(response.Data[i]) > parseModelReleaseDate(response.Data[j])
	})

	// Store lightweight model list in database (fast!)
	if err := h.storeModelsInDatabase(ctx, response.Data); err != nil {
		// Log warning but don't fail - we still have the data
		fmt.Printf("Warning: Failed to store models in database: %v\n", err)
	}

	// Update memory cache as backup
	modelsCache.setCachedModels(response.Data)

	return response.Data, nil
}

// isCodeGenerationModel filters out non-coding models (audio, video, image, etc.)
func (h *OpenRouterHandler) isCodeGenerationModel(modelName string) bool {
	modelLower := strings.ToLower(modelName)

	// Exclude audio/video/image models
	excludePatterns := []string{
		"audio", "video", "realtime", "transcribe", "tts", "image", "vision",
		"whisper", "dall-e", "tts-1", "embedding", "moderation",
	}

	for _, pattern := range excludePatterns {
		if strings.Contains(modelLower, pattern) {
			return false
		}
	}

	// Include text-based coding models
	includePatterns := []string{
		"gpt-4o", "gpt-4", "o1", "gpt-3.5", "claude", "gemini", "llama", "mistral",
		"deepseek", "qwen", "coder", "code", "instruct", "chat", "text",
	}

	for _, pattern := range includePatterns {
		if strings.Contains(modelLower, pattern) {
			return true
		}
	}

	// Default to false for unknown models
	return false
}

// storeModelsInDatabase stores models using efficient two-table architecture
func (h *OpenRouterHandler) storeModelsInDatabase(ctx context.Context, models []OpenRouterModel) error {
	if h.db == nil {
		// No database available, store in memory cache only
		modelsCache.setCachedModels(models)
		return nil
	}

	// This implements the smart two-table approach:
	// 1. Sync lightweight model list (fast, frequent)
	if err := h.syncModelList(ctx, models); err != nil {
		return fmt.Errorf("failed to sync model list: %w", err)
	}

	// 2. Metadata will be populated on-demand or via background jobs
	// No need to fetch heavy metadata here - that's the whole point!

	return nil
}

// ensureTablesExist creates the OpenRouter tables if they don't exist
func (h *OpenRouterHandler) ensureTablesExist(ctx context.Context) error {
	if h.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Use VectorDB's ExecContext method

	// Create models table with backward compatibility
	modelsTable := `
	CREATE TABLE IF NOT EXISTS openrouter_models (
		model_id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		created_date INTEGER,
		context_length INTEGER,
		provider_name TEXT,
		last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		added_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	if _, err := h.db.ExecContext(ctx, modelsTable); err != nil {
		return fmt.Errorf("failed to create openrouter_models table: %w", err)
	}

	// Migrate existing schema: rename provider_name to provider
	if err := h.migrateProviderColumn(ctx); err != nil {
		return fmt.Errorf("failed to migrate provider column: %w", err)
	}

	// Create metadata table
	metadataTable := `
	CREATE TABLE IF NOT EXISTS openrouter_model_metadata (
		model_id TEXT PRIMARY KEY,
		architecture_json TEXT,
		endpoints_json TEXT,
		pricing_summary_json TEXT,
		max_context_length INTEGER,
		supported_modalities TEXT,
		provider_count INTEGER,
		best_price_prompt REAL,
		best_price_completion REAL,
		uptime_average REAL,
		last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		metadata_version INTEGER DEFAULT 1,
		FOREIGN KEY (model_id) REFERENCES openrouter_models(model_id) ON DELETE CASCADE
	)`

	if _, err := h.db.ExecContext(ctx, metadataTable); err != nil {
		return fmt.Errorf("failed to create openrouter_model_metadata table: %w", err)
	}

	// Create cleanup trigger
	trigger := `
	CREATE TRIGGER IF NOT EXISTS cleanup_openrouter_metadata
	AFTER DELETE ON openrouter_models
	FOR EACH ROW
	BEGIN
		DELETE FROM openrouter_model_metadata
		WHERE model_id = OLD.model_id;
	END`

	if _, err := h.db.ExecContext(ctx, trigger); err != nil {
		return fmt.Errorf("failed to create cleanup trigger: %w", err)
	}

	// Create indexes
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_openrouter_models_last_seen ON openrouter_models(last_seen)",
		"CREATE INDEX IF NOT EXISTS idx_openrouter_models_provider ON openrouter_models(provider)",
		"CREATE INDEX IF NOT EXISTS idx_openrouter_metadata_updated ON openrouter_model_metadata(last_updated)",
	}

	for _, index := range indexes {
		if _, err := h.db.ExecContext(ctx, index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// syncModelList efficiently syncs the lightweight model list
func (h *OpenRouterHandler) syncModelList(ctx context.Context, models []OpenRouterModel) error {
	if h.db == nil {
		fmt.Printf("No database connection - using memory cache only\n")
		return nil
	}

	// Ensure tables exist
	if err := h.ensureTablesExist(ctx); err != nil {
		return fmt.Errorf("failed to ensure tables exist: %w", err)
	}

	// 1. Get current model IDs from database
	existingModels, err := h.getExistingModelIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to get existing models: %w", err)
	}

	// 2. Find new models to add
	newModels := h.findNewModels(models, existingModels)
	if len(newModels) > 0 {
		fmt.Printf("➕ Adding %d new models\n", len(newModels))
		if err := h.insertNewModels(ctx, newModels); err != nil {
			return fmt.Errorf("failed to insert new models: %w", err)
		}
	}

	// 3. Update last_seen for existing models
	currentModelIDs := make(map[string]bool)
	for _, model := range models {
		currentModelIDs[model.ID] = true
	}
	if err := h.updateLastSeen(ctx, currentModelIDs); err != nil {
		return fmt.Errorf("failed to update last_seen: %w", err)
	}

	// 4. Remove models not in current list (trigger will clean metadata)
	removedCount, err := h.removeObsoleteModels(ctx, currentModelIDs, existingModels)
	if err != nil {
		return fmt.Errorf("failed to remove obsolete models: %w", err)
	}
	if removedCount > 0 {
		fmt.Printf("🗑️ Removed %d obsolete models\n", removedCount)
	}

	return nil
}

// getModelMetadata fetches detailed metadata for a specific model (on-demand)
func (h *OpenRouterHandler) getModelMetadata(ctx context.Context, modelID string) (*OpenRouterModel, error) {
	// 1. Check database first for cached metadata
	if cachedModel, err := h.getModelMetadataFromDB(ctx, modelID); err == nil {
		// Check if metadata is still fresh (7 days TTL for metadata)
		if !h.isMetadataStale(ctx, modelID) {
			fmt.Printf("Using cached metadata for %s\n", modelID)
			return cachedModel, nil
		}
	}

	// 2. Metadata missing or stale, fetch from API
	fmt.Printf("Fetching fresh metadata for %s\n", modelID)
	model, err := h.getDetailedModelMetadata(ctx, OpenRouterModel{ID: modelID})
	if err != nil {
		return nil, err
	}

	// 3. Metadata is automatically stored in getDetailedModelMetadata
	return &model, nil
}

// getModelMetadataFromDB retrieves cached metadata from database
func (h *OpenRouterHandler) getModelMetadataFromDB(ctx context.Context, modelID string) (*OpenRouterModel, error) {
	if h.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	query := `
		SELECT m.model_id, m.name, m.description, m.created_date, m.context_length,
		       md.architecture_json, md.endpoints_json, md.max_context_length,
		       md.supported_modalities, md.provider_count, md.last_updated
		FROM openrouter_models m
		LEFT JOIN openrouter_model_metadata md ON m.model_id = md.model_id
		WHERE m.model_id = ?
	`

	row := h.db.QueryRowContext(ctx, query, modelID)

	var model OpenRouterModel
	var architectureJSON, endpointsJSON, supportedModalities sql.NullString
	var maxContextLength, providerCount sql.NullInt64
	var lastUpdated sql.NullTime

	err := row.Scan(
		&model.ID, &model.Name, &model.Description, &model.Created, &model.ContextLength,
		&architectureJSON, &endpointsJSON, &maxContextLength,
		&supportedModalities, &providerCount, &lastUpdated,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("model not found in database")
		}
		return nil, fmt.Errorf("failed to query model metadata: %w", err)
	}

	// Deserialize JSON fields if available
	if architectureJSON.Valid && architectureJSON.String != "" {
		if err := json.Unmarshal([]byte(architectureJSON.String), &model.Architecture); err != nil {
			fmt.Printf("Failed to unmarshal architecture for %s: %v\n", modelID, err)
		}
	}

	if endpointsJSON.Valid && endpointsJSON.String != "" {
		if err := json.Unmarshal([]byte(endpointsJSON.String), &model.Endpoints); err != nil {
			fmt.Printf("Failed to unmarshal endpoints for %s: %v\n", modelID, err)
		}
	}

	return &model, nil
}

// isMetadataStale checks if metadata needs refresh (7 day TTL)
func (h *OpenRouterHandler) isMetadataStale(ctx context.Context, modelID string) bool {
	if h.db == nil {
		return true
	}

	query := `SELECT last_updated FROM openrouter_model_metadata WHERE model_id = ?`
	row := h.db.QueryRowContext(ctx, query, modelID)

	var lastUpdated time.Time
	if err := row.Scan(&lastUpdated); err != nil {
		return true // No metadata found, consider stale
	}

	metadataTTL := 7 * 24 * time.Hour // 7 days
	return time.Since(lastUpdated) > metadataTTL
}

// Database helper functions for efficient model sync

// getExistingModelIDs retrieves current model IDs from database
func (h *OpenRouterHandler) getExistingModelIDs(ctx context.Context) (map[string]bool, error) {
	if h.db == nil {
		return make(map[string]bool), nil
	}

	rows, err := h.db.QueryContext(ctx, "SELECT model_id FROM openrouter_models")
	if err != nil {
		return nil, fmt.Errorf("failed to query existing models: %w", err)
	}
	defer rows.Close()

	existing := make(map[string]bool)
	for rows.Next() {
		var modelID string
		if err := rows.Scan(&modelID); err != nil {
			return nil, fmt.Errorf("failed to scan model ID: %w", err)
		}
		existing[modelID] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return existing, nil
}

// findNewModels identifies models not in database
func (h *OpenRouterHandler) findNewModels(models []OpenRouterModel, existing map[string]bool) []OpenRouterModel {
	var newModels []OpenRouterModel
	for _, model := range models {
		if !existing[model.ID] {
			newModels = append(newModels, model)
		}
	}
	return newModels
}

// insertNewModels adds new models to database (lightweight data only)
func (h *OpenRouterHandler) insertNewModels(ctx context.Context, models []OpenRouterModel) error {
	if h.db == nil {
		return nil
	}

	stmt, err := h.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO openrouter_models
		(model_id, name, description, created_date, context_length, provider, last_seen, added_date)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`)

	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}

	for _, model := range models {
		provider := normalizeProviderName(extractProviderFromID(model.ID))

		_, err := h.db.ExecContext(ctx, `
			INSERT OR IGNORE INTO openrouter_models
			(model_id, name, description, created_date, context_length, provider, last_seen, added_date)
			VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		`, model.ID, model.Name, model.Description, model.Created, model.ContextLength, provider)

		if err != nil {
			return fmt.Errorf("failed to insert model %s: %w", model.ID, err)
		}

		fmt.Printf("  ➕ %s (%s)\n", model.Name, provider)
	}

	_ = stmt // Suppress unused variable warning
	return nil
}

// updateLastSeen updates last_seen timestamp for existing models
func (h *OpenRouterHandler) updateLastSeen(ctx context.Context, currentModels map[string]bool) error {
	if h.db == nil || len(currentModels) == 0 {
		return nil
	}

	// Build IN clause for batch update
	modelIDs := make([]string, 0, len(currentModels))
	for modelID := range currentModels {
		modelIDs = append(modelIDs, modelID)
	}

	// Create placeholders for IN clause
	placeholders := make([]string, len(modelIDs))
	args := make([]interface{}, len(modelIDs))
	for i, modelID := range modelIDs {
		placeholders[i] = "?"
		args[i] = modelID
	}

	query := fmt.Sprintf(`
		UPDATE openrouter_models
		SET last_seen = CURRENT_TIMESTAMP
		WHERE model_id IN (%s)
	`, strings.Join(placeholders, ","))

	_, err := h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update last_seen: %w", err)
	}

	return nil
}

// removeObsoleteModels removes models not in current list
func (h *OpenRouterHandler) removeObsoleteModels(ctx context.Context, current map[string]bool, existing map[string]bool) (int, error) {
	if h.db == nil {
		return 0, nil
	}

	// Find models to remove
	toRemove := make([]string, 0)
	for modelID := range existing {
		if !current[modelID] {
			toRemove = append(toRemove, modelID)
		}
	}

	if len(toRemove) == 0 {
		return 0, nil
	}

	// Create placeholders for IN clause
	placeholders := make([]string, len(toRemove))
	args := make([]interface{}, len(toRemove))
	for i, modelID := range toRemove {
		placeholders[i] = "?"
		args[i] = modelID
		fmt.Printf("  🗑️ Removing obsolete model: %s\n", modelID)
	}

	query := fmt.Sprintf(`
		DELETE FROM openrouter_models
		WHERE model_id IN (%s)
	`, strings.Join(placeholders, ","))

	result, err := h.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to remove obsolete models: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// getDetailedModelMetadata fetches comprehensive metadata for a model
func (h *OpenRouterHandler) getDetailedModelMetadata(ctx context.Context, model OpenRouterModel) (OpenRouterModel, error) {
	// Construct the endpoints URL for this specific model
	endpointsURL := fmt.Sprintf("%s/models/%s/endpoints", h.baseURL, model.ID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpointsURL, nil)
	if err != nil {
		return model, fmt.Errorf("failed to create endpoints request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+h.options.OpenRouterAPIKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return model, fmt.Errorf("endpoints request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return model, fmt.Errorf("endpoints API error %d: %s", resp.StatusCode, string(body))
	}

	var endpointsResponse OpenRouterEndpointsResponse
	if err := json.NewDecoder(resp.Body).Decode(&endpointsResponse); err != nil {
		return model, fmt.Errorf("failed to decode endpoints response: %w", err)
	}

	// Enrich the model with detailed endpoint information
	enrichedModel := endpointsResponse.Data

	// Preserve original fields that might not be in endpoints response
	if enrichedModel.ID == "" {
		enrichedModel.ID = model.ID
	}
	if enrichedModel.Name == "" {
		enrichedModel.Name = model.Name
	}
	if enrichedModel.Created == 0 {
		enrichedModel.Created = model.Created
	}

	// Store the comprehensive metadata in database
	if err := h.storeModelMetadata(ctx, enrichedModel); err != nil {
		// Log warning but don't fail - we still have the enriched model
		fmt.Printf("Failed to store metadata for %s: %v\n", enrichedModel.ID, err)
	}

	return enrichedModel, nil
}

// storeModelMetadata stores comprehensive metadata in database
func (h *OpenRouterHandler) storeModelMetadata(ctx context.Context, model OpenRouterModel) error {
	if h.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Serialize architecture data
	architectureJSON, err := json.Marshal(model.Architecture)
	if err != nil {
		return fmt.Errorf("failed to marshal architecture: %w", err)
	}

	// Serialize endpoints data
	endpointsJSON, err := json.Marshal(model.Endpoints)
	if err != nil {
		return fmt.Errorf("failed to marshal endpoints: %w", err)
	}

	// Compute aggregated fields
	maxContextLength := 0
	providerCount := len(model.Endpoints)
	var bestPricePrompt, bestPriceCompletion float64 = -1, -1
	var uptimeSum float64
	var supportedModalities []string

	// Process endpoints to compute aggregated data
	for _, endpoint := range model.Endpoints {
		// Max context length
		if endpoint.ContextLength > maxContextLength {
			maxContextLength = endpoint.ContextLength
		}

		// Best pricing (lowest non-zero prices)
		if endpoint.Pricing.Prompt != "" {
			if price, err := parsePrice(endpoint.Pricing.Prompt); err == nil && price > 0 {
				if bestPricePrompt < 0 || price < bestPricePrompt {
					bestPricePrompt = price
				}
			}
		}
		if endpoint.Pricing.Completion != "" {
			if price, err := parsePrice(endpoint.Pricing.Completion); err == nil && price > 0 {
				if bestPriceCompletion < 0 || price < bestPriceCompletion {
					bestPriceCompletion = price
				}
			}
		}

		// Average uptime
		uptimeSum += endpoint.UptimeLast30m
	}

	// Calculate average uptime
	var uptimeAverage float64
	if providerCount > 0 {
		uptimeAverage = uptimeSum / float64(providerCount)
	}

	// Extract supported modalities
	if len(model.Architecture.InputModalities) > 0 {
		supportedModalities = model.Architecture.InputModalities
	}
	modalitiesStr := strings.Join(supportedModalities, ",")

	// Create pricing summary
	pricingSummary := map[string]interface{}{
		"best_prompt_price":     bestPricePrompt,
		"best_completion_price": bestPriceCompletion,
		"provider_count":        providerCount,
		"average_uptime":        uptimeAverage,
	}
	pricingSummaryJSON, err := json.Marshal(pricingSummary)
	if err != nil {
		return fmt.Errorf("failed to marshal pricing summary: %w", err)
	}

	// Insert or update metadata
	query := `
		INSERT OR REPLACE INTO openrouter_model_metadata
		(model_id, architecture_json, endpoints_json, pricing_summary_json,
		 max_context_length, supported_modalities, provider_count,
		 best_price_prompt, best_price_completion, uptime_average,
		 last_updated, metadata_version)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, 1)
	`

	_, err = h.db.ExecContext(ctx, query,
		model.ID,
		string(architectureJSON),
		string(endpointsJSON),
		string(pricingSummaryJSON),
		maxContextLength,
		modalitiesStr,
		providerCount,
		bestPricePrompt,
		bestPriceCompletion,
		uptimeAverage,
	)

	if err != nil {
		return fmt.Errorf("failed to store metadata for %s: %w", model.ID, err)
	}

	fmt.Printf("💾 Stored metadata for %s (%d providers, %d context)\n",
		model.ID, providerCount, maxContextLength)
	return nil
}

// parsePrice parses a price string to float64
func parsePrice(priceStr string) (float64, error) {
	// Remove any currency symbols and whitespace
	cleaned := strings.TrimSpace(priceStr)
	cleaned = strings.TrimPrefix(cleaned, "$")
	cleaned = strings.TrimPrefix(cleaned, "€")
	cleaned = strings.TrimPrefix(cleaned, "£")

	// Handle scientific notation and regular decimals
	if cleaned == "" || cleaned == "0" {
		return 0, nil
	}

	return strconv.ParseFloat(cleaned, 64)
}

// GetModelWithMetadata retrieves model with on-demand metadata loading
func GetModelWithMetadata(ctx context.Context, apiKey, modelID string) (*OpenRouterModel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key required for metadata")
	}

	options := llm.ApiHandlerOptions{
		OpenRouterAPIKey: apiKey,
	}

	// Use database-enabled handler
	db := vectordb.GetInstance()
	handler := NewOpenRouterHandlerWithDB(options, db)
	return handler.getModelMetadata(ctx, modelID)
}

// cleanupExpiredModels removes models that haven't been seen for longer than TTL
func (h *OpenRouterHandler) cleanupExpiredModels(ctx context.Context) error {
	if h.db == nil {
		return nil
	}

	// Remove models not seen for more than 7 days (longer than sync TTL)
	cleanupTTL := 7 * 24 * time.Hour
	cutoffTime := time.Now().Add(-cleanupTTL)

	query := `
		DELETE FROM openrouter_models
		WHERE last_seen < ? OR last_seen IS NULL
	`

	result, err := h.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired models: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get cleanup count: %w", err)
	}

	if rowsAffected > 0 {
		fmt.Printf("🧹 Cleaned up %d expired models (older than %v)\n", rowsAffected, cleanupTTL)
	}

	return nil
}

// cleanupStaleMetadata removes metadata that hasn't been updated for longer than metadata TTL
func (h *OpenRouterHandler) cleanupStaleMetadata(ctx context.Context) error {
	if h.db == nil {
		return nil
	}

	// Remove metadata older than 30 days
	metadataTTL := 30 * 24 * time.Hour
	cutoffTime := time.Now().Add(-metadataTTL)

	query := `
		DELETE FROM openrouter_model_metadata
		WHERE last_updated < ? OR last_updated IS NULL
	`

	result, err := h.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to cleanup stale metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get metadata cleanup count: %w", err)
	}

	if rowsAffected > 0 {
		fmt.Printf("🧹 Cleaned up %d stale metadata entries (older than %v)\n", rowsAffected, metadataTTL)
	}

	return nil
}

// RefreshModelsIfExpired checks TTL and refreshes models if needed
func RefreshModelsIfExpired(ctx context.Context, apiKey string) error {
	if apiKey == "" {
		return fmt.Errorf("API key required for model refresh")
	}

	options := llm.ApiHandlerOptions{
		OpenRouterAPIKey: apiKey,
	}

	db := vectordb.GetInstance()
	handler := NewOpenRouterHandlerWithDB(options, db)

	// Check if refresh is needed
	if handler.isDatabaseCacheValid(ctx) {
		fmt.Println("Model cache is still valid, no refresh needed")
		return nil
	}

	fmt.Println("Model cache expired, refreshing...")

	// Cleanup expired data first
	if err := handler.cleanupExpiredModels(ctx); err != nil {
		fmt.Printf("Cleanup warning: %v\n", err)
	}

	if err := handler.cleanupStaleMetadata(ctx); err != nil {
		fmt.Printf("Metadata cleanup warning: %v\n", err)
	}

	// Refresh models
	_, err := handler.GetOpenRouterModels(ctx)
	if err != nil {
		return fmt.Errorf("failed to refresh models: %w", err)
	}

	fmt.Println("Model cache refreshed successfully")
	return nil
}

// parseModelReleaseDate extracts release date from model for sorting
func parseModelReleaseDate(model OpenRouterModel) int64 {
	// Extract date from model ID (e.g., "anthropic/claude-3.5-sonnet-20241022")
	id := model.ID

	// Look for date patterns in the ID
	if strings.Contains(id, "20241022") {
		return 20241022 // Claude 3.5 Sonnet latest
	}
	if strings.Contains(id, "20240620") {
		return 20240620 // Claude 3.5 Sonnet original
	}
	if strings.Contains(id, "20240229") {
		return 20240229 // Claude 3 Opus
	}
	if strings.Contains(id, "gpt-4o") {
		return 20240513 // GPT-4o release
	}
	if strings.Contains(id, "gpt-4-turbo") {
		return 20240409 // GPT-4 Turbo
	}
	if strings.Contains(id, "gemini-1.5-pro") {
		return 20240215 // Gemini 1.5 Pro
	}
	if strings.Contains(id, "llama-3.1") {
		return 20240723 // Llama 3.1
	}
	if strings.Contains(id, "llama-3") {
		return 20240418 // Llama 3
	}

	// Default to 0 for unknown models (will be sorted last)
	return 0
}

// GetModelsByProvider returns models categorized by provider
func (h *OpenRouterHandler) GetModelsByProvider(ctx context.Context) (map[string][]OpenRouterModel, error) {
	// Get all models
	allModels, err := h.GetOpenRouterModels(ctx)
	if err != nil {
		return nil, err
	}

	// Categorize by provider
	providerModels := make(map[string][]OpenRouterModel)

	for _, model := range allModels {
		provider := normalizeProviderName(extractProviderFromID(model.ID))
		providerModels[provider] = append(providerModels[provider], model)
	}

	// Sort each provider's models by release date DESC
	for provider := range providerModels {
		sort.Slice(providerModels[provider], func(i, j int) bool {
			return parseModelReleaseDate(providerModels[provider][i]) > parseModelReleaseDate(providerModels[provider][j])
		})
	}

	return providerModels, nil
}

// extractProviderFromID extracts provider name from model ID
func extractProviderFromID(modelID string) string {
	parts := strings.Split(modelID, "/")
	if len(parts) >= 2 {
		return parts[0]
	}
	return "unknown"
}

// migrateProviderColumn migrates the provider_name column to provider
func (h *OpenRouterHandler) migrateProviderColumn(ctx context.Context) error {
	// Check if provider column already exists
	checkQuery := `PRAGMA table_info(openrouter_models)`
	rows, err := h.db.QueryContext(ctx, checkQuery)
	if err != nil {
		return fmt.Errorf("failed to check table schema: %w", err)
	}
	defer rows.Close()

	hasProvider := false
	hasProviderName := false

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue interface{}

		err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk)
		if err != nil {
			continue
		}

		if name == "provider" {
			hasProvider = true
		}
		if name == "provider_name" {
			hasProviderName = true
		}
	}

	// If provider column exists, we're already migrated
	if hasProvider {
		return nil
	}

	// If provider_name doesn't exist either, this is a fresh install
	if !hasProviderName {
		// Add provider column to fresh table
		addColumnQuery := `ALTER TABLE openrouter_models ADD COLUMN provider TEXT`
		if _, err := h.db.ExecContext(ctx, addColumnQuery); err != nil {
			return fmt.Errorf("failed to add provider column: %w", err)
		}
		return nil
	}

	// Migration needed: rename provider_name to provider
	// SQLite doesn't support RENAME COLUMN directly, so we need to recreate the table
	migrationQueries := []string{
		// Create new table with correct schema
		`CREATE TABLE openrouter_models_new (
			model_id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			created_date INTEGER,
			context_length INTEGER,
			provider TEXT,
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			added_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		// Copy data from old table to new table
		`INSERT INTO openrouter_models_new
		 SELECT model_id, name, description, created_date, context_length,
		        provider_name, last_seen, added_date
		 FROM openrouter_models`,
		// Drop old table
		`DROP TABLE openrouter_models`,
		// Rename new table
		`ALTER TABLE openrouter_models_new RENAME TO openrouter_models`,
	}

	for _, query := range migrationQueries {
		if _, err := h.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute migration query '%s': %w", query, err)
		}
	}

	return nil
}

// normalizeProviderName maps OpenRouter provider names to our standard naming convention
func normalizeProviderName(openRouterProvider string) string {
	switch strings.ToLower(openRouterProvider) {
	case "anthropic":
		return "anthropic"
	case "openai":
		return "openai"
	case "google":
		return "google"
	case "meta-llama", "meta":
		return "meta"
	case "mistralai", "mistral":
		return "mistral"
	case "deepseek":
		return "deepseek"
	case "x-ai", "xai":
		return "xai"
	case "qwen":
		return "qwen"
	case "cohere":
		return "cohere"
	case "perplexity":
		return "perplexity"
	case "nvidia":
		return "nvidia"
	case "microsoft":
		return "microsoft"
	case "01-ai":
		return "01ai"
	default:
		// For unknown providers, use the original name in lowercase
		return strings.ToLower(openRouterProvider)
	}
}

// GetTopOpenRouterModels fetches the top N models from OpenRouter (cached)
func (h *OpenRouterHandler) GetTopOpenRouterModels(ctx context.Context, limit int) ([]OpenRouterModel, error) {
	// Get all models first (this will use cache if available)
	allModels, err := h.GetOpenRouterModels(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models: %w", err)
	}

	// Get popular models from rankings (with fallback to curated list)
	popularModelIDs, err := scrapeOpenRouterRankings(ctx)
	if err != nil {
		// Fallback to curated list if scraping fails
		popularModelIDs = getCuratedTopModels()
	}

	// Create a map for quick lookup
	modelMap := make(map[string]OpenRouterModel)
	for _, model := range allModels {
		modelMap[model.ID] = model
	}

	// First, try to get popular models in order
	var selectedModels []OpenRouterModel
	for _, modelID := range popularModelIDs {
		if model, exists := modelMap[modelID]; exists {
			selectedModels = append(selectedModels, model)
			if len(selectedModels) >= limit {
				break
			}
		}
	}

	// If we don't have enough popular models, add others
	if len(selectedModels) < limit {
		for _, model := range allModels {
			// Skip if already added
			found := false
			for _, selected := range selectedModels {
				if selected.ID == model.ID {
					found = true
					break
				}
			}
			if !found {
				selectedModels = append(selectedModels, model)
				if len(selectedModels) >= limit {
					break
				}
			}
		}
	}

	return selectedModels, nil
}

// GetTopOpenRouterModelsByRanking is a public function to get top OpenRouter models
func GetTopOpenRouterModelsByRanking(ctx context.Context, apiKey string, limit int) ([]OpenRouterModel, error) {
	if apiKey == "" {
		// No API key - try scraping rankings instead of using API
		return getTopModelsFromScraping(ctx, limit)
	}

	options := llm.ApiHandlerOptions{
		OpenRouterAPIKey: apiKey,
	}

	// Use database-enabled handler
	db := vectordb.GetInstance()
	handler := NewOpenRouterHandlerWithDB(options, db)
	return handler.GetTopOpenRouterModels(ctx, limit)
}

// getTopModelsFromScraping gets models from scraping when no API key is available
func getTopModelsFromScraping(ctx context.Context, limit int) ([]OpenRouterModel, error) {
	// Get model IDs from scraping
	modelIDs, err := scrapeOpenRouterRankings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to scrape rankings: %w", err)
	}

	// Convert model IDs to OpenRouterModel structs
	var models []OpenRouterModel
	for i, modelID := range modelIDs {
		if i >= limit {
			break
		}

		// Extract name from ID (provider/model-name)
		parts := strings.Split(modelID, "/")
		name := modelID
		if len(parts) == 2 {
			caser := cases.Title(language.English)
			name = caser.String(strings.ReplaceAll(parts[1], "-", " "))
		}

		model := OpenRouterModel{
			ID:   modelID,
			Name: name,
		}
		models = append(models, model)
	}

	return models, nil
}

// GetOpenRouterModels returns all OpenRouter models from database
func GetOpenRouterModels(ctx context.Context, apiKey string) ([]OpenRouterModel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key required")
	}

	options := llm.ApiHandlerOptions{
		OpenRouterAPIKey: apiKey,
	}

	// Use database-enabled handler
	db := vectordb.GetInstance()
	handler := NewOpenRouterHandlerWithDB(options, db)
	return handler.getModelsFromDatabase(ctx)
}

// GetOpenRouterModelsByProvider returns all OpenRouter models categorized by provider
func GetOpenRouterModelsByProvider(ctx context.Context, apiKey string) (map[string][]OpenRouterModel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key required")
	}

	options := llm.ApiHandlerOptions{
		OpenRouterAPIKey: apiKey,
	}

	// Use database-enabled handler
	db := vectordb.GetInstance()
	handler := NewOpenRouterHandlerWithDB(options, db)
	return handler.GetModelsByProvider(ctx)
}

// GetOpenRouterModelsBySpecificProvider returns models from a specific provider only
func GetOpenRouterModelsBySpecificProvider(ctx context.Context, apiKey string, providerFilter string) ([]OpenRouterModel, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key required")
	}

	options := llm.ApiHandlerOptions{
		OpenRouterAPIKey: apiKey,
	}

	// Use database-enabled handler
	db := vectordb.GetInstance()
	handler := NewOpenRouterHandlerWithDB(options, db)
	return handler.getModelsFromDatabaseByProvider(ctx, providerFilter)
}

// GetOpenRouterCacheStatus returns cache information for debugging
func GetOpenRouterCacheStatus() (bool, time.Time, int) {
	modelsCache.mutex.RLock()
	defer modelsCache.mutex.RUnlock()

	isValid := time.Since(modelsCache.timestamp) < cacheTTL && len(modelsCache.models) > 0
	return isValid, modelsCache.timestamp, len(modelsCache.models)
}

// OpenRouterModel represents a model from OpenRouter's model list
type OpenRouterModel struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Pricing       OpenRouterModelPricing `json:"pricing"`
	ContextLength int                    `json:"context_length"`
	Architecture  OpenRouterArchitecture `json:"architecture"`
	TopProvider   OpenRouterTopProvider  `json:"top_provider"`
	Created       int64                  `json:"created"`

	// Enhanced endpoint information for detailed model data
	Endpoints []OpenRouterEndpoint `json:"endpoints,omitempty"`
}

// OpenRouterEndpoint represents a provider endpoint for a model
type OpenRouterEndpoint struct {
	Name                string                    `json:"name"`
	ContextLength       int                       `json:"context_length"`
	Pricing             OpenRouterEndpointPricing `json:"pricing"`
	ProviderName        string                    `json:"provider_name"`
	Tag                 string                    `json:"tag"`
	Quantization        string                    `json:"quantization"`
	MaxCompletionTokens int                       `json:"max_completion_tokens"`
	MaxPromptTokens     *int                      `json:"max_prompt_tokens"`
	SupportedParameters []string                  `json:"supported_parameters"`
	Status              int                       `json:"status"`
	UptimeLast30m       float64                   `json:"uptime_last_30m"`
}

// OpenRouterEndpointPricing represents comprehensive pricing for a specific endpoint
type OpenRouterEndpointPricing struct {
	Prompt            string  `json:"prompt"`
	Completion        string  `json:"completion"`
	Request           string  `json:"request"`
	Image             string  `json:"image"`
	WebSearch         string  `json:"web_search"`
	InternalReasoning string  `json:"internal_reasoning"`
	InputCacheRead    string  `json:"input_cache_read"`
	InputCacheWrite   string  `json:"input_cache_write"`
	Discount          float64 `json:"discount"`
}

// OpenRouterModelPricing represents pricing information
type OpenRouterModelPricing struct {
	Prompt     string `json:"prompt"`     // Price per token as string
	Completion string `json:"completion"` // Price per token as string
	Image      string `json:"image"`      // Price per image as string
	Request    string `json:"request"`    // Price per request as string
}

// OpenRouterArchitecture represents comprehensive model architecture info
type OpenRouterArchitecture struct {
	Tokenizer        string   `json:"tokenizer"`         // Tokenizer type
	InstructType     *string  `json:"instruct_type"`     // Instruction format (can be null)
	Modality         string   `json:"modality"`          // "text->text", "text+image->text", etc.
	InputModalities  []string `json:"input_modalities"`  // ["text"], ["text", "image"], etc.
	OutputModalities []string `json:"output_modalities"` // ["text"], etc.
}

// OpenRouterTopProvider represents the top provider for a model
type OpenRouterTopProvider struct {
	MaxCompletionTokens  int  `json:"max_completion_tokens"`
	IsModerationRequired bool `json:"is_moderation_required"`
}

// OpenRouterModelsResponse represents the response from /models endpoint
type OpenRouterModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

// OpenRouterEndpointsResponse represents the response from /models/{id}/endpoints endpoint
type OpenRouterEndpointsResponse struct {
	Data OpenRouterModel `json:"data"`
}

// scrapeOpenRouterRankings fetches models from OpenRouter API
func scrapeOpenRouterRankings(ctx context.Context) ([]string, error) {
	// Use the public OpenRouter API to get all models
	req, err := http.NewRequestWithContext(ctx, "GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return getCuratedTopModels(), nil
	}

	req.Header.Set("User-Agent", "CodeForge/1.0")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return getCuratedTopModels(), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return getCuratedTopModels(), nil
	}

	var response struct {
		Data []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Created int64  `json:"created"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return getCuratedTopModels(), nil
	}

	// Sort by creation date (newest first) and take top 20
	models := response.Data
	if len(models) == 0 {
		return getCuratedTopModels(), nil
	}

	// Sort by created timestamp (newest first)
	for i := 0; i < len(models)-1; i++ {
		for j := i + 1; j < len(models); j++ {
			if models[i].Created < models[j].Created {
				models[i], models[j] = models[j], models[i]
			}
		}
	}

	// Extract model IDs, prioritizing popular providers
	var modelIDs []string
	popularProviders := []string{"anthropic", "openai", "google", "mistralai", "deepseek", "x-ai", "meta-llama"}

	// First pass: get models from popular providers
	for _, provider := range popularProviders {
		for _, model := range models {
			if strings.HasPrefix(model.ID, provider+"/") {
				modelIDs = append(modelIDs, model.ID)
				if len(modelIDs) >= 20 {
					return modelIDs, nil
				}
			}
		}
	}

	// Second pass: fill remaining slots with any models
	for _, model := range models {
		found := false
		for _, existing := range modelIDs {
			if existing == model.ID {
				found = true
				break
			}
		}
		if !found {
			modelIDs = append(modelIDs, model.ID)
			if len(modelIDs) >= 20 {
				break
			}
		}
	}

	if len(modelIDs) > 0 {
		return modelIDs, nil
	}

	return getCuratedTopModels(), nil
}

// getCuratedTopModels returns a curated list of top models based on popularity
func getCuratedTopModels() []string {
	return []string{
		// Latest Anthropic models (June 2025)
		"anthropic/claude-3.5-sonnet-20241022",
		"anthropic/claude-3.5-haiku-20241022",
		"anthropic/claude-3-opus-20240229",

		// Latest OpenAI models (June 2025)
		"openai/gpt-4o-2024-08-06",
		"openai/gpt-4o-mini-2024-07-18",
		"openai/o1-preview-2024-09-12",
		"openai/o1-mini-2024-09-12",
		"openai/chatgpt-4o-latest",

		// Latest Google models (June 2025)
		"google/gemini-2.5-pro",
		"google/gemini-2.5-flash",
		"google/gemini-pro-1.5-latest",

		// Latest Meta/Llama models (June 2025)
		"meta-llama/llama-3.3-70b-instruct",
		"meta-llama/llama-3.1-405b-instruct",
		"meta-llama/llama-3.1-70b-instruct",

		// Latest Mistral models (June 2025)
		"mistralai/mistral-large-2407",
		"mistralai/mistral-small-3.2-24b-instruct",
		"mistralai/magistral-medium-2506",

		// Latest DeepSeek models (June 2025)
		"deepseek/deepseek-r1-0528",
		"deepseek/deepseek-r1-distill-qwen-7b",

		// Latest xAI models (June 2025)
		"x-ai/grok-3",
		"x-ai/grok-3-mini",

		// Other current models (June 2025)
		"cohere/command-r-plus-08-2024",
		"minimax/minimax-m1",
		"moonshotai/kimi-dev-72b",
		"inception/mercury",
	}
}
