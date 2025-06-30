package models

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ModelManager provides high-level model management functionality
type ModelManager struct {
	registry    *ModelRegistry
	preferences *UserPreferences
	favorites   map[CanonicalModelID]bool
	cache       *ModelCache
	mu          sync.RWMutex
}

// UserPreferences stores user model preferences and settings
type UserPreferences struct {
	DefaultModel       CanonicalModelID   `json:"default_model"`
	PreferredProviders []ProviderID       `json:"preferred_providers"`
	MaxCostPerMToken   float64            `json:"max_cost_per_m_token"` // Maximum cost per million tokens
	PreferredTier      string             `json:"preferred_tier"`       // "free", "paid", "premium"
	RequiredFeatures   []string           `json:"required_features"`    // "vision", "tools", "reasoning"
	ExcludedModels     []CanonicalModelID `json:"excluded_models"`
	AutoSelectBest     bool               `json:"auto_select_best"`  // Auto-select best model for task
	FallbackEnabled    bool               `json:"fallback_enabled"`  // Enable fallback to alternative models
	CachePreferences   bool               `json:"cache_preferences"` // Cache model selection decisions
}

// ModelCache caches model availability and performance data
type ModelCache struct {
	availability map[string]ModelAvailability // provider:model -> availability
	performance  map[CanonicalModelID]ModelPerformance
	lastUpdated  map[string]time.Time
	mu           sync.RWMutex
}

// ModelAvailability tracks model availability status
type ModelAvailability struct {
	Available    bool          `json:"available"`
	LastChecked  time.Time     `json:"last_checked"`
	ErrorCount   int           `json:"error_count"`
	LastError    string        `json:"last_error,omitempty"`
	ResponseTime time.Duration `json:"response_time"`
}

// ModelPerformance tracks model performance metrics
type ModelPerformance struct {
	AverageLatency   time.Duration `json:"average_latency"`
	SuccessRate      float64       `json:"success_rate"`
	TokensPerSecond  float64       `json:"tokens_per_second"`
	QualityScore     float64       `json:"quality_score"`     // User-rated quality (1-10)
	ReliabilityScore float64       `json:"reliability_score"` // Uptime/availability score
	CostEfficiency   float64       `json:"cost_efficiency"`   // Quality per dollar
	LastEvaluated    time.Time     `json:"last_evaluated"`
	UsageCount       int           `json:"usage_count"`
}

// ModelSelectionCriteria defines criteria for model selection
type ModelSelectionCriteria struct {
	TaskType         string     `json:"task_type"`          // "chat", "code", "analysis", "creative"
	RequiredFeatures []string   `json:"required_features"`  // "vision", "tools", "reasoning"
	MaxCost          float64    `json:"max_cost"`           // Maximum cost per million tokens
	MinQuality       float64    `json:"min_quality"`        // Minimum quality score
	PreferredSpeed   string     `json:"preferred_speed"`    // "fast", "balanced", "quality"
	ContextLength    int        `json:"context_length"`     // Required context length
	Provider         ProviderID `json:"provider,omitempty"` // Specific provider preference
}

// ModelRecommendation represents a recommended model with reasoning
type ModelRecommendation struct {
	Model      *CanonicalModel       `json:"model"`
	Provider   ProviderID            `json:"provider"`
	Score      float64               `json:"score"`               // Recommendation score (0-100)
	Reasoning  []string              `json:"reasoning"`           // Why this model was recommended
	Confidence float64               `json:"confidence"`          // Confidence in recommendation (0-1)
	Fallbacks  []ModelRecommendation `json:"fallbacks,omitempty"` // Alternative recommendations
}

// NewModelManager creates a new model manager
func NewModelManager(registry *ModelRegistry) *ModelManager {
	return &ModelManager{
		registry:  registry,
		favorites: make(map[CanonicalModelID]bool),
		cache: &ModelCache{
			availability: make(map[string]ModelAvailability),
			performance:  make(map[CanonicalModelID]ModelPerformance),
			lastUpdated:  make(map[string]time.Time),
		},
		preferences: &UserPreferences{
			DefaultModel:       ModelClaudeSonnet4, // Default to Claude Sonnet 4
			PreferredProviders: []ProviderID{ProviderAnthropic, ProviderOpenAI, ProviderGemini},
			MaxCostPerMToken:   10.0, // $10 per million tokens
			PreferredTier:      "paid",
			RequiredFeatures:   []string{},
			AutoSelectBest:     true,
			FallbackEnabled:    true,
			CachePreferences:   true,
		},
	}
}

// GetRecommendation returns the best model recommendation for given criteria
func (mm *ModelManager) GetRecommendation(ctx context.Context, criteria ModelSelectionCriteria) (*ModelRecommendation, error) {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	// Get all available models
	allModels := mm.registry.ListModels()

	// Filter models based on criteria
	candidates := mm.filterModelsByCriteria(allModels, criteria)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no models match the specified criteria")
	}

	// Score and rank candidates
	scored := mm.scoreModels(candidates, criteria)
	if len(scored) == 0 {
		return nil, fmt.Errorf("no suitable models found")
	}

	// Sort by score (highest first)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})

	// Return top recommendation with fallbacks
	recommendation := scored[0]
	if len(scored) > 1 {
		endIdx := len(scored)
		if endIdx > 4 {
			endIdx = 4
		}
		recommendation.Fallbacks = scored[1:endIdx] // Up to 3 fallbacks
	}

	return &recommendation, nil
}

// GetFavoriteModels returns user's favorite models
func (mm *ModelManager) GetFavoriteModels() []*CanonicalModel {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	var favorites []*CanonicalModel
	for modelID := range mm.favorites {
		if model, exists := mm.registry.GetModel(modelID); exists {
			favorites = append(favorites, model)
		}
	}

	// Sort by name for consistent ordering
	sort.Slice(favorites, func(i, j int) bool {
		return favorites[i].Name < favorites[j].Name
	})

	return favorites
}

// AddFavorite adds a model to favorites
func (mm *ModelManager) AddFavorite(modelID CanonicalModelID) error {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Verify model exists
	if _, exists := mm.registry.GetModel(modelID); !exists {
		return fmt.Errorf("model %s not found", modelID)
	}

	mm.favorites[modelID] = true
	return nil
}

// RemoveFavorite removes a model from favorites
func (mm *ModelManager) RemoveFavorite(modelID CanonicalModelID) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	delete(mm.favorites, modelID)
}

// UpdatePreferences updates user preferences
func (mm *ModelManager) UpdatePreferences(prefs UserPreferences) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.preferences = &prefs
}

// GetPreferences returns current user preferences
func (mm *ModelManager) GetPreferences() UserPreferences {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	return *mm.preferences
}

// filterModelsByCriteria filters models based on selection criteria
func (mm *ModelManager) filterModelsByCriteria(models []*CanonicalModel, criteria ModelSelectionCriteria) []*CanonicalModel {
	var filtered []*CanonicalModel

	for _, model := range models {
		// Check if model is excluded
		if mm.isModelExcluded(model.ID) {
			continue
		}

		// Check cost constraint
		if criteria.MaxCost > 0 && model.Pricing.OutputPrice > criteria.MaxCost {
			continue
		}

		// Check context length requirement
		if criteria.ContextLength > 0 && model.Limits.ContextWindow < criteria.ContextLength {
			continue
		}

		// Check required features
		if !mm.hasRequiredFeatures(model, criteria.RequiredFeatures) {
			continue
		}

		// Check provider preference
		if criteria.Provider != "" {
			if _, hasProvider := model.Providers[criteria.Provider]; !hasProvider {
				continue
			}
		}

		filtered = append(filtered, model)
	}

	return filtered
}

// scoreModels scores and ranks models based on criteria
func (mm *ModelManager) scoreModels(models []*CanonicalModel, criteria ModelSelectionCriteria) []ModelRecommendation {
	var recommendations []ModelRecommendation

	for _, model := range models {
		// Calculate base score
		score := mm.calculateModelScore(model, criteria)

		// Apply user preferences
		score = mm.applyUserPreferences(model, score)

		// Get best provider for this model
		provider := mm.getBestProvider(model, criteria)

		// Build reasoning
		reasoning := mm.buildReasoning(model, criteria, score)

		recommendation := ModelRecommendation{
			Model:      model,
			Provider:   provider,
			Score:      score,
			Reasoning:  reasoning,
			Confidence: mm.calculateConfidence(model, criteria),
		}

		recommendations = append(recommendations, recommendation)
	}

	return recommendations
}

// calculateModelScore calculates a score for a model based on criteria
func (mm *ModelManager) calculateModelScore(model *CanonicalModel, criteria ModelSelectionCriteria) float64 {
	score := 50.0 // Base score

	// Task-specific scoring
	switch criteria.TaskType {
	case "code":
		if model.Capabilities.SupportsCode {
			score += 20
		}
	case "reasoning":
		if model.Capabilities.SupportsReasoning {
			score += 25
		}
	case "vision":
		if model.Capabilities.SupportsVision {
			score += 20
		}
	}

	// Feature scoring
	for _, feature := range criteria.RequiredFeatures {
		if mm.modelHasFeature(model, feature) {
			score += 10
		}
	}

	// Cost efficiency (lower cost = higher score)
	if model.Pricing.OutputPrice > 0 {
		costScore := 20 * (1.0 - min(model.Pricing.OutputPrice/criteria.MaxCost, 1.0))
		score += costScore
	}

	// Performance scoring from cache
	if perf, exists := mm.cache.performance[model.ID]; exists {
		score += perf.QualityScore * 2       // Quality weight
		score += perf.ReliabilityScore * 1.5 // Reliability weight
		score += perf.CostEfficiency * 1.0   // Cost efficiency weight
	}

	if score > 100.0 {
		score = 100.0
	}
	return score // Cap at 100
}

// Helper functions
func (mm *ModelManager) isModelExcluded(modelID CanonicalModelID) bool {
	for _, excluded := range mm.preferences.ExcludedModels {
		if excluded == modelID {
			return true
		}
	}
	return false
}

func (mm *ModelManager) hasRequiredFeatures(model *CanonicalModel, features []string) bool {
	for _, feature := range features {
		if !mm.modelHasFeature(model, feature) {
			return false
		}
	}
	return true
}

func (mm *ModelManager) modelHasFeature(model *CanonicalModel, feature string) bool {
	switch strings.ToLower(feature) {
	case "vision":
		return model.Capabilities.SupportsVision
	case "tools", "function_calling":
		return model.Capabilities.SupportsTools
	case "reasoning", "thinking":
		return model.Capabilities.SupportsReasoning
	case "code":
		return model.Capabilities.SupportsCode
	case "streaming":
		return model.Capabilities.SupportsStreaming
	case "images":
		return model.Capabilities.SupportsImages
	default:
		return false
	}
}

func (mm *ModelManager) applyUserPreferences(model *CanonicalModel, baseScore float64) float64 {
	score := baseScore

	// Favorite bonus
	if mm.favorites[model.ID] {
		score += 15
	}

	// Provider preference
	for i, prefProvider := range mm.preferences.PreferredProviders {
		if _, hasProvider := model.Providers[prefProvider]; hasProvider {
			// Earlier in preference list = higher bonus
			bonus := float64(len(mm.preferences.PreferredProviders)-i) * 2
			score += bonus
			break
		}
	}

	return score
}

func (mm *ModelManager) getBestProvider(model *CanonicalModel, criteria ModelSelectionCriteria) ProviderID {
	// If specific provider requested, use it
	if criteria.Provider != "" {
		if _, exists := model.Providers[criteria.Provider]; exists {
			return criteria.Provider
		}
	}

	// Use user's preferred providers
	for _, prefProvider := range mm.preferences.PreferredProviders {
		if _, exists := model.Providers[prefProvider]; exists {
			return prefProvider
		}
	}

	// Fallback to first available provider
	for providerID := range model.Providers {
		return providerID
	}

	return ""
}

func (mm *ModelManager) buildReasoning(model *CanonicalModel, criteria ModelSelectionCriteria, score float64) []string {
	var reasons []string

	if score >= 80 {
		reasons = append(reasons, "Excellent match for your requirements")
	} else if score >= 60 {
		reasons = append(reasons, "Good match for your requirements")
	}

	if mm.favorites[model.ID] {
		reasons = append(reasons, "One of your favorite models")
	}

	if model.Capabilities.SupportsReasoning && contains(criteria.RequiredFeatures, "reasoning") {
		reasons = append(reasons, "Supports advanced reasoning capabilities")
	}

	if model.Pricing.OutputPrice <= criteria.MaxCost*0.5 {
		reasons = append(reasons, "Cost-effective option")
	}

	return reasons
}

func (mm *ModelManager) calculateConfidence(model *CanonicalModel, criteria ModelSelectionCriteria) float64 {
	confidence := 0.7 // Base confidence

	// Higher confidence for well-known models
	if model.Family == "claude" || model.Family == "gpt" {
		confidence += 0.2
	}

	// Higher confidence if we have performance data
	if _, exists := mm.cache.performance[model.ID]; exists {
		confidence += 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}
	return confidence
}

// Utility functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
