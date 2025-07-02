package models

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// ModelAPI provides a unified interface for model operations
type ModelAPI struct {
	registry  *ModelRegistry
	manager   *ModelManager
	selector  *ModelSelector
	discovery *ModelDiscoveryService
}

// ListModelsRequest represents a request to list models
type ListModelsRequest struct {
	// Filtering
	Provider         ProviderID `json:"provider,omitempty"`
	Family           string     `json:"family,omitempty"`            // "claude", "gpt", "gemini"
	RequiredFeatures []string   `json:"required_features,omitempty"` // "vision", "tools", "reasoning"
	MaxCost          float64    `json:"max_cost,omitempty"`          // Maximum cost per million tokens
	MinContextWindow int        `json:"min_context_window,omitempty"`
	AvailableOnly    bool       `json:"available_only"` // Only show available models

	// Sorting
	SortBy    string `json:"sort_by"`    // "name", "cost", "quality", "speed", "context_window"
	SortOrder string `json:"sort_order"` // "asc", "desc"

	// Pagination
	Limit  int `json:"limit,omitempty"`  // Maximum number of results
	Offset int `json:"offset,omitempty"` // Number of results to skip

	// Search
	Query string `json:"query,omitempty"` // Search query for model names/descriptions
}

// ListModelsResponse represents the response from listing models
type ListModelsResponse struct {
	Models    []*CanonicalModel `json:"models"`
	Total     int               `json:"total"`     // Total number of models (before pagination)
	Limit     int               `json:"limit"`     // Applied limit
	Offset    int               `json:"offset"`    // Applied offset
	HasMore   bool              `json:"has_more"`  // Whether there are more results
	Providers []ProviderID      `json:"providers"` // Available providers
	Families  []string          `json:"families"`  // Available model families
}

// CompareModelsRequest represents a request to compare models
type CompareModelsRequest struct {
	ModelIDs []CanonicalModelID     `json:"model_ids"`
	TaskType string                 `json:"task_type,omitempty"`
	Criteria ModelSelectionCriteria `json:"criteria,omitempty"`
}

// CompareModelsResponse represents the response from comparing models
type CompareModelsResponse struct {
	Comparisons []ModelComparison `json:"comparisons"`
	Winner      *CanonicalModel   `json:"winner,omitempty"` // Best model based on criteria
}

// NewModelAPI creates a new model API instance
func NewModelAPI() *ModelAPI {
	registry := NewModelRegistry()
	manager := NewModelManager(registry)
	discovery := NewModelDiscoveryService(registry, manager, DefaultDiscoveryConfig())
	selector := NewModelSelector(manager, registry, discovery)

	return &ModelAPI{
		registry:  registry,
		manager:   manager,
		selector:  selector,
		discovery: discovery,
	}
}

// Start starts the model API and its background services
func (api *ModelAPI) Start(ctx context.Context) error {
	return api.discovery.Start(ctx)
}

// Stop stops the model API and its background services
func (api *ModelAPI) Stop() error {
	return api.discovery.Stop()
}

// ListModels lists models based on the provided criteria
func (api *ModelAPI) ListModels(ctx context.Context, req ListModelsRequest) (*ListModelsResponse, error) {
	// Get all models
	allModels := api.registry.ListModels()

	// Apply filters
	filtered := api.filterModels(allModels, req)

	// Apply search
	if req.Query != "" {
		filtered = api.searchModels(filtered, req.Query)
	}

	// Sort models
	api.sortModels(filtered, req.SortBy, req.SortOrder)

	// Calculate pagination
	total := len(filtered)
	start := req.Offset
	end := start + req.Limit

	if start > total {
		start = total
	}
	if end > total || req.Limit == 0 {
		end = total
	}

	// Apply pagination
	var paginatedModels []*CanonicalModel
	if start < end {
		paginatedModels = filtered[start:end]
	}

	// Collect metadata
	providers := api.collectProviders(allModels)
	families := api.collectFamilies(allModels)

	return &ListModelsResponse{
		Models:    paginatedModels,
		Total:     total,
		Limit:     req.Limit,
		Offset:    req.Offset,
		HasMore:   end < total,
		Providers: providers,
		Families:  families,
	}, nil
}

// GetModel retrieves a specific model by ID
func (api *ModelAPI) GetModel(ctx context.Context, modelID CanonicalModelID) (*CanonicalModel, error) {
	model, exists := api.registry.GetModel(modelID)
	if !exists {
		return nil, fmt.Errorf("model %s not found", modelID)
	}
	return model, nil
}

// SelectModel selects the best model for a given task
func (api *ModelAPI) SelectModel(ctx context.Context, req SelectionRequest) (*SelectionResponse, error) {
	return api.selector.SelectModel(ctx, req)
}

// GetQuickSelectOptions returns quick selection options for a task
func (api *ModelAPI) GetQuickSelectOptions(ctx context.Context, req SelectionRequest) (*QuickSelectOptions, error) {
	return api.selector.GetQuickSelectOptions(ctx, req)
}

// CompareModels compares multiple models for a specific task
func (api *ModelAPI) CompareModels(ctx context.Context, req CompareModelsRequest) (*CompareModelsResponse, error) {
	comparisons, err := api.selector.GetModelComparison(ctx, req.ModelIDs, SelectionRequest{
		TaskType:          req.TaskType,
		RequiredFeatures:  req.Criteria.RequiredFeatures,
		MaxCost:           req.Criteria.MaxCost,
		MinQuality:        req.Criteria.MinQuality,
		PreferredSpeed:    req.Criteria.PreferredSpeed,
		PreferredProvider: req.Criteria.Provider,
	})
	if err != nil {
		return nil, err
	}

	response := &CompareModelsResponse{
		Comparisons: comparisons,
	}

	// Set winner (highest scoring model)
	if len(comparisons) > 0 {
		response.Winner = comparisons[0].Model
	}

	return response, nil
}

// GetFavorites returns user's favorite models
func (api *ModelAPI) GetFavorites(ctx context.Context) ([]*CanonicalModel, error) {
	return api.manager.GetFavoriteModels(), nil
}

// AddFavorite adds a model to user's favorites
func (api *ModelAPI) AddFavorite(ctx context.Context, modelID CanonicalModelID) error {
	return api.manager.AddFavorite(modelID)
}

// RemoveFavorite removes a model from user's favorites
func (api *ModelAPI) RemoveFavorite(ctx context.Context, modelID CanonicalModelID) error {
	api.manager.RemoveFavorite(modelID)
	return nil
}

// GetUserPreferences returns current user preferences
func (api *ModelAPI) GetUserPreferences(ctx context.Context) (*UserPreferences, error) {
	prefs := api.manager.GetPreferences()
	return &prefs, nil
}

// UpdateUserPreferences updates user preferences
func (api *ModelAPI) UpdateUserPreferences(ctx context.Context, prefs UserPreferences) error {
	api.manager.UpdatePreferences(prefs)
	return nil
}

// GetProviderModels returns all models available for a specific provider
func (api *ModelAPI) GetProviderModels(ctx context.Context, providerID ProviderID) ([]*CanonicalModel, error) {
	return api.registry.ListModelsByProvider(providerID), nil
}

// GetModelsByFamily returns all models in a specific family
func (api *ModelAPI) GetModelsByFamily(ctx context.Context, family string) ([]*CanonicalModel, error) {
	allModels := api.registry.ListModels()
	var familyModels []*CanonicalModel

	for _, model := range allModels {
		if strings.EqualFold(model.Family, family) {
			familyModels = append(familyModels, model)
		}
	}

	// Sort by name
	sort.Slice(familyModels, func(i, j int) bool {
		return familyModels[i].Name < familyModels[j].Name
	})

	return familyModels, nil
}

// RefreshModels triggers a refresh of model data from providers
func (api *ModelAPI) RefreshModels(ctx context.Context, providerID ProviderID) error {
	if providerID == "" {
		// Refresh all providers
		providers := []ProviderID{
			ProviderAnthropicCanonical,
			ProviderOpenAICanonical,
			ProviderGeminiCanonical,
			ProviderOpenRouterCanonical,
		}

		for _, provider := range providers {
			if err := api.discovery.DiscoverModelsForProvider(provider); err != nil {
				return fmt.Errorf("failed to refresh models for provider %s: %w", provider, err)
			}
		}
	} else {
		// Refresh specific provider
		if err := api.discovery.DiscoverModelsForProvider(providerID); err != nil {
			return fmt.Errorf("failed to refresh models for provider %s: %w", providerID, err)
		}
	}

	return nil
}

// GetServiceStatus returns the status of the model API services
func (api *ModelAPI) GetServiceStatus(ctx context.Context) map[string]any {
	return map[string]any{
		"registry_models": len(api.registry.ListModels()),
		"discovery":       api.discovery.GetQueueStatus(),
	}
}

// Helper methods for filtering, sorting, and searching

func (api *ModelAPI) filterModels(models []*CanonicalModel, req ListModelsRequest) []*CanonicalModel {
	var filtered []*CanonicalModel

	for _, model := range models {
		// Provider filter
		if req.Provider != "" {
			if _, hasProvider := model.Providers[req.Provider]; !hasProvider {
				continue
			}
		}

		// Family filter
		if req.Family != "" && !strings.EqualFold(model.Family, req.Family) {
			continue
		}

		// Cost filter
		if req.MaxCost > 0 && model.Pricing.OutputPrice > req.MaxCost {
			continue
		}

		// Context window filter
		if req.MinContextWindow > 0 && model.Limits.ContextWindow < req.MinContextWindow {
			continue
		}

		// Required features filter
		if !api.hasRequiredFeatures(model, req.RequiredFeatures) {
			continue
		}

		// Availability filter
		if req.AvailableOnly {
			hasAvailableProvider := false
			for _, mapping := range model.Providers {
				if mapping.Available {
					hasAvailableProvider = true
					break
				}
			}
			if !hasAvailableProvider {
				continue
			}
		}

		filtered = append(filtered, model)
	}

	return filtered
}

func (api *ModelAPI) searchModels(models []*CanonicalModel, query string) []*CanonicalModel {
	if query == "" {
		return models
	}

	query = strings.ToLower(query)
	var matched []*CanonicalModel

	for _, model := range models {
		// Search in name, family, and ID
		if strings.Contains(strings.ToLower(model.Name), query) ||
			strings.Contains(strings.ToLower(model.Family), query) ||
			strings.Contains(strings.ToLower(string(model.ID)), query) {
			matched = append(matched, model)
		}
	}

	return matched
}

func (api *ModelAPI) sortModels(models []*CanonicalModel, sortBy, sortOrder string) {
	if sortBy == "" {
		sortBy = "name"
	}
	if sortOrder == "" {
		sortOrder = "asc"
	}

	sort.Slice(models, func(i, j int) bool {
		var less bool

		switch sortBy {
		case "name":
			less = models[i].Name < models[j].Name
		case "cost":
			less = models[i].Pricing.OutputPrice < models[j].Pricing.OutputPrice
		case "context_window":
			less = models[i].Limits.ContextWindow < models[j].Limits.ContextWindow
		case "family":
			less = models[i].Family < models[j].Family
		default:
			less = models[i].Name < models[j].Name
		}

		if sortOrder == "desc" {
			return !less
		}
		return less
	})
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
		case "reasoning", "thinking":
			if !model.Capabilities.SupportsReasoning {
				return false
			}
		case "code":
			if !model.Capabilities.SupportsCode {
				return false
			}
		case "streaming":
			if !model.Capabilities.SupportsStreaming {
				return false
			}
		case "images":
			if !model.Capabilities.SupportsImages {
				return false
			}
		}
	}
	return true
}

func (api *ModelAPI) collectProviders(models []*CanonicalModel) []ProviderID {
	providerSet := make(map[ProviderID]bool)

	for _, model := range models {
		for providerID := range model.Providers {
			providerSet[providerID] = true
		}
	}

	var providers []ProviderID
	for providerID := range providerSet {
		providers = append(providers, providerID)
	}

	// Sort providers
	sort.Slice(providers, func(i, j int) bool {
		return string(providers[i]) < string(providers[j])
	})

	return providers
}

func (api *ModelAPI) collectFamilies(models []*CanonicalModel) []string {
	familySet := make(map[string]bool)

	for _, model := range models {
		if model.Family != "" {
			familySet[model.Family] = true
		}
	}

	var families []string
	for family := range familySet {
		families = append(families, family)
	}

	// Sort families
	sort.Strings(families)

	return families
}
