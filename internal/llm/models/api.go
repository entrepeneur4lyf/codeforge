package models

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ModelAPI provides a unified interface for model management
type ModelAPI struct {
	registry  *ModelRegistry
	manager   *ModelManager
	selector  *ModelSelector
	discovery *ModelDiscoveryService

	// State management
	initialized bool
	mu          sync.RWMutex
}

// ModelListOptions defines options for listing models
type ModelListOptions struct {
	Provider           ProviderID `json:"provider,omitempty"`
	Family             string     `json:"family,omitempty"`            // "gpt", "claude", "gemini"
	RequiredFeatures   []string   `json:"required_features,omitempty"` // "vision", "tools", "reasoning"
	MaxCost            float64    `json:"max_cost,omitempty"`          // Maximum cost per million tokens
	IncludeUnavailable bool       `json:"include_unavailable"`         // Include unavailable models
	SortBy             string     `json:"sort_by"`                     // "name", "cost", "quality", "popularity"
	SortOrder          string     `json:"sort_order"`                  // "asc", "desc"
	Limit              int        `json:"limit,omitempty"`             // Maximum number of results
}

// ModelSummary provides a summary view of a model
type ModelSummary struct {
	ID            CanonicalModelID `json:"id"`
	Name          string           `json:"name"`
	Family        string           `json:"family"`
	Provider      string           `json:"provider"` // Primary provider
	Description   string           `json:"description"`
	Capabilities  []string         `json:"capabilities"` // Human-readable capabilities
	CostTier      string           `json:"cost_tier"`    // "free", "low", "medium", "high"
	QualityTier   string           `json:"quality_tier"` // "basic", "good", "excellent"
	SpeedTier     string           `json:"speed_tier"`   // "slow", "medium", "fast"
	ContextSize   string           `json:"context_size"` // "small", "medium", "large", "xlarge"
	IsFavorite    bool             `json:"is_favorite"`
	IsRecommended bool             `json:"is_recommended"`
	Tags          []string         `json:"tags"`
}

// ModelDetails provides detailed information about a model
type ModelDetails struct {
	*CanonicalModel

	// Additional computed fields
	AvailableProviders []ProviderSummary `json:"available_providers"`
	UsageStats         UsageStats        `json:"usage_stats"`
	UserRating         float64           `json:"user_rating"`
	CommunityRating    float64           `json:"community_rating"`
	RecentPerformance  PerformanceStats  `json:"recent_performance"`
	SimilarModels      []ModelSummary    `json:"similar_models"`
}

// ProviderSummary summarizes a provider for a model
type ProviderSummary struct {
	ID             ProviderID `json:"id"`
	Name           string     `json:"name"`
	Available      bool       `json:"available"`
	ResponseTime   string     `json:"response_time"`   // "fast", "medium", "slow"
	Reliability    float64    `json:"reliability"`     // 0-1
	CostMultiplier float64    `json:"cost_multiplier"` // Relative cost vs base price
}

// UsageStats tracks model usage statistics
type UsageStats struct {
	TotalRequests      int        `json:"total_requests"`
	SuccessfulRequests int        `json:"successful_requests"`
	FailedRequests     int        `json:"failed_requests"`
	AverageLatency     string     `json:"average_latency"`
	TotalTokens        int        `json:"total_tokens"`
	TotalCost          float64    `json:"total_cost"`
	LastUsed           *time.Time `json:"last_used,omitempty"`
}

// PerformanceStats tracks recent performance metrics
type PerformanceStats struct {
	Uptime          float64 `json:"uptime"`          // Percentage uptime
	AverageLatency  int     `json:"average_latency"` // Milliseconds
	P95Latency      int     `json:"p95_latency"`     // 95th percentile latency
	ErrorRate       float64 `json:"error_rate"`      // Percentage of failed requests
	TokensPerSecond float64 `json:"tokens_per_second"`
}

// NewModelAPI creates a new model API instance
func NewModelAPI() *ModelAPI {
	registry := NewModelRegistry()
	manager := NewModelManager(registry)
	discovery := NewModelDiscoveryService(registry, manager)
	selector := NewModelSelector(manager, registry, discovery)

	return &ModelAPI{
		registry:  registry,
		manager:   manager,
		selector:  selector,
		discovery: discovery,
	}
}

// Initialize initializes the model API
func (api *ModelAPI) Initialize(ctx context.Context) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	if api.initialized {
		return nil
	}

	// Start discovery service
	if err := api.discovery.Start(ctx); err != nil {
		return fmt.Errorf("failed to start discovery service: %w", err)
	}

	api.initialized = true
	return nil
}

// Shutdown shuts down the model API
func (api *ModelAPI) Shutdown() error {
	api.mu.Lock()
	defer api.mu.Unlock()

	if !api.initialized {
		return nil
	}

	// Stop discovery service
	if err := api.discovery.Stop(); err != nil {
		return fmt.Errorf("failed to stop discovery service: %w", err)
	}

	api.initialized = false
	return nil
}

// ListModels returns a list of models based on the given options
func (api *ModelAPI) ListModels(options ModelListOptions) ([]ModelSummary, error) {
	api.mu.RLock()
	defer api.mu.RUnlock()

	models := api.registry.ListModels()
	var summaries []ModelSummary

	for _, model := range models {
		// Apply filters
		if options.Provider != "" {
			if _, hasProvider := model.Providers[options.Provider]; !hasProvider {
				continue
			}
		}

		if options.Family != "" && model.Family != options.Family {
			continue
		}

		if options.MaxCost > 0 && model.Pricing.OutputPrice > options.MaxCost {
			continue
		}

		if len(options.RequiredFeatures) > 0 {
			if !api.hasRequiredFeatures(model, options.RequiredFeatures) {
				continue
			}
		}

		// Create summary
		summary := api.createModelSummary(model)
		summaries = append(summaries, summary)
	}

	// Sort results
	api.sortModelSummaries(summaries, options.SortBy, options.SortOrder)

	// Apply limit
	if options.Limit > 0 && len(summaries) > options.Limit {
		summaries = summaries[:options.Limit]
	}

	return summaries, nil
}

// GetModel returns detailed information about a specific model
func (api *ModelAPI) GetModel(modelID CanonicalModelID) (*ModelDetails, error) {
	api.mu.RLock()
	defer api.mu.RUnlock()

	model, exists := api.registry.GetModel(modelID)
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}

	details := &ModelDetails{
		CanonicalModel:     model,
		AvailableProviders: api.getProviderSummaries(model),
		UsageStats:         api.getUsageStats(modelID),
		RecentPerformance:  api.getPerformanceStats(modelID),
		SimilarModels:      api.getSimilarModels(model),
	}

	return details, nil
}

// SelectModel selects the best model for a given request
func (api *ModelAPI) SelectModel(ctx context.Context, req SelectionRequest) (*SelectionResponse, error) {
	api.mu.RLock()
	defer api.mu.RUnlock()

	return api.selector.SelectModel(ctx, req)
}

// GetQuickSelectOptions returns quick selection options
func (api *ModelAPI) GetQuickSelectOptions(ctx context.Context, req SelectionRequest) (*QuickSelectOptions, error) {
	api.mu.RLock()
	defer api.mu.RUnlock()

	return api.selector.GetQuickSelectOptions(ctx, req)
}

// CompareModels compares multiple models for a task
func (api *ModelAPI) CompareModels(ctx context.Context, modelIDs []CanonicalModelID, req SelectionRequest) ([]ModelComparison, error) {
	api.mu.RLock()
	defer api.mu.RUnlock()

	return api.selector.GetModelComparison(ctx, modelIDs, req)
}

// GetFavorites returns user's favorite models
func (api *ModelAPI) GetFavorites() ([]ModelSummary, error) {
	api.mu.RLock()
	defer api.mu.RUnlock()

	favorites := api.manager.GetFavoriteModels()
	var summaries []ModelSummary

	for _, model := range favorites {
		summary := api.createModelSummary(model)
		summary.IsFavorite = true
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

// AddFavorite adds a model to favorites
func (api *ModelAPI) AddFavorite(modelID CanonicalModelID) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	return api.manager.AddFavorite(modelID)
}

// RemoveFavorite removes a model from favorites
func (api *ModelAPI) RemoveFavorite(modelID CanonicalModelID) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	api.manager.RemoveFavorite(modelID)
	return nil
}

// UpdatePreferences updates user preferences
func (api *ModelAPI) UpdatePreferences(prefs UserPreferences) error {
	api.mu.Lock()
	defer api.mu.Unlock()

	api.manager.UpdatePreferences(prefs)
	return nil
}

// GetPreferences returns current user preferences
func (api *ModelAPI) GetPreferences() UserPreferences {
	api.mu.RLock()
	defer api.mu.RUnlock()

	return api.manager.GetPreferences()
}

// RefreshModels triggers a refresh of model information
func (api *ModelAPI) RefreshModels(ctx context.Context, provider ProviderID) error {
	api.mu.RLock()
	defer api.mu.RUnlock()

	if provider == "" {
		// Refresh all providers
		providers := []ProviderID{ProviderOpenAI, ProviderAnthropic, ProviderGemini, ProviderOpenRouter}
		for _, p := range providers {
			api.discovery.ScheduleDiscovery(p, TaskDiscoverModels, 5)
		}
	} else {
		api.discovery.ScheduleDiscovery(provider, TaskDiscoverModels, 5)
	}

	return nil
}

// Helper methods

func (api *ModelAPI) createModelSummary(model *CanonicalModel) ModelSummary {
	// Get primary provider (first available)
	var primaryProvider string
	for providerID := range model.Providers {
		primaryProvider = string(providerID)
		break
	}

	// Build capabilities list
	var capabilities []string
	if model.Capabilities.SupportsVision {
		capabilities = append(capabilities, "Vision")
	}
	if model.Capabilities.SupportsTools {
		capabilities = append(capabilities, "Tools")
	}
	if model.Capabilities.SupportsReasoning {
		capabilities = append(capabilities, "Reasoning")
	}
	if model.Capabilities.SupportsStreaming {
		capabilities = append(capabilities, "Streaming")
	}

	// Determine tiers
	costTier := api.getCostTier(model.Pricing.OutputPrice)
	qualityTier := api.getQualityTier(model)
	speedTier := api.getSpeedTier(model)
	contextSize := api.getContextSizeTier(model.Limits.ContextWindow)

	// Build tags
	var tags []string
	if model.Capabilities.SupportsVision {
		tags = append(tags, "multimodal")
	}
	if model.Capabilities.SupportsReasoning {
		tags = append(tags, "reasoning")
	}
	if costTier == "low" {
		tags = append(tags, "budget-friendly")
	}
	if qualityTier == "excellent" {
		tags = append(tags, "flagship")
	}

	return ModelSummary{
		ID:           model.ID,
		Name:         model.Name,
		Family:       model.Family,
		Provider:     primaryProvider,
		Description:  fmt.Sprintf("%s model with %s capabilities", model.Family, strings.Join(capabilities, ", ")),
		Capabilities: capabilities,
		CostTier:     costTier,
		QualityTier:  qualityTier,
		SpeedTier:    speedTier,
		ContextSize:  contextSize,
		Tags:         tags,
	}
}

func (api *ModelAPI) hasRequiredFeatures(model *CanonicalModel, features []string) bool {
	for _, feature := range features {
		switch strings.ToLower(feature) {
		case "vision":
			if !model.Capabilities.SupportsVision {
				return false
			}
		case "tools", "function_calling":
			if !model.Capabilities.SupportsTools {
				return false
			}
		case "reasoning":
			if !model.Capabilities.SupportsReasoning {
				return false
			}
		case "streaming":
			if !model.Capabilities.SupportsStreaming {
				return false
			}
		}
	}
	return true
}

func (api *ModelAPI) sortModelSummaries(summaries []ModelSummary, sortBy, sortOrder string) {
	if sortBy == "" {
		sortBy = "name"
	}
	if sortOrder == "" {
		sortOrder = "asc"
	}

	sort.Slice(summaries, func(i, j int) bool {
		var less bool
		switch sortBy {
		case "name":
			less = summaries[i].Name < summaries[j].Name
		case "family":
			less = summaries[i].Family < summaries[j].Family
		case "provider":
			less = summaries[i].Provider < summaries[j].Provider
		default:
			less = summaries[i].Name < summaries[j].Name
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})
}

func (api *ModelAPI) getCostTier(outputPrice float64) string {
	if outputPrice <= 1.0 {
		return "low"
	} else if outputPrice <= 10.0 {
		return "medium"
	} else {
		return "high"
	}
}

func (api *ModelAPI) getQualityTier(model *CanonicalModel) string {
	// Simple heuristic based on model name and capabilities
	name := strings.ToLower(model.Name)
	if strings.Contains(name, "opus") || strings.Contains(name, "4o") || model.Capabilities.SupportsReasoning {
		return "excellent"
	} else if strings.Contains(name, "sonnet") || strings.Contains(name, "turbo") {
		return "good"
	} else {
		return "basic"
	}
}

func (api *ModelAPI) getSpeedTier(model *CanonicalModel) string {
	name := strings.ToLower(model.Name)
	if strings.Contains(name, "mini") || strings.Contains(name, "haiku") || strings.Contains(name, "flash") {
		return "fast"
	} else if strings.Contains(name, "turbo") {
		return "medium"
	} else {
		return "slow"
	}
}

func (api *ModelAPI) getContextSizeTier(contextWindow int) string {
	if contextWindow >= 1000000 {
		return "xlarge"
	} else if contextWindow >= 200000 {
		return "large"
	} else if contextWindow >= 50000 {
		return "medium"
	} else {
		return "small"
	}
}

func (api *ModelAPI) getProviderSummaries(model *CanonicalModel) []ProviderSummary {
	var summaries []ProviderSummary
	for providerID := range model.Providers {
		summary := ProviderSummary{
			ID:             providerID,
			Name:           string(providerID),
			Available:      true, // Would check actual availability
			ResponseTime:   "medium",
			Reliability:    0.95,
			CostMultiplier: 1.0,
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func (api *ModelAPI) getUsageStats(modelID CanonicalModelID) UsageStats {
	// Would fetch from actual usage tracking
	return UsageStats{
		TotalRequests:      0,
		SuccessfulRequests: 0,
		FailedRequests:     0,
		AverageLatency:     "N/A",
		TotalTokens:        0,
		TotalCost:          0.0,
	}
}

func (api *ModelAPI) getPerformanceStats(modelID CanonicalModelID) PerformanceStats {
	// Would fetch from actual performance monitoring
	return PerformanceStats{
		Uptime:          99.5,
		AverageLatency:  1500,
		P95Latency:      3000,
		ErrorRate:       0.5,
		TokensPerSecond: 25.0,
	}
}

func (api *ModelAPI) getSimilarModels(model *CanonicalModel) []ModelSummary {
	// Would implement similarity matching
	return []ModelSummary{}
}
