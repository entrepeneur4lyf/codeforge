package models

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// ModelSelector provides high-level model selection functionality
type ModelSelector struct {
	manager   *ModelManager
	registry  *ModelRegistry
	discovery *ModelDiscoveryService
}

// SelectionRequest represents a request for model selection
type SelectionRequest struct {
	// Task information
	TaskType        string `json:"task_type"`        // "chat", "code", "analysis", "creative", "reasoning"
	TaskDescription string `json:"task_description"` // Free-form description
	InputLength     int    `json:"input_length"`     // Estimated input length in tokens

	// Requirements
	RequiredFeatures []string `json:"required_features"` // "vision", "tools", "reasoning", "streaming"
	MaxCost          float64  `json:"max_cost"`          // Maximum cost per million tokens
	MinQuality       float64  `json:"min_quality"`       // Minimum quality score (1-10)

	// Preferences
	PreferredSpeed    string     `json:"preferred_speed"` // "fastest", "balanced", "quality"
	PreferredProvider ProviderID `json:"preferred_provider,omitempty"`
	AllowFallback     bool       `json:"allow_fallback"` // Allow fallback models

	// Context
	UserID    string            `json:"user_id,omitempty"`
	SessionID string            `json:"session_id,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// SelectionResponse contains the selected model and alternatives
type SelectionResponse struct {
	// Primary selection
	SelectedModel    *CanonicalModel `json:"selected_model"`
	SelectedProvider ProviderID      `json:"selected_provider"`

	// Recommendation details
	Score      float64  `json:"score"`      // Selection score (0-100)
	Confidence float64  `json:"confidence"` // Confidence in selection (0-1)
	Reasoning  []string `json:"reasoning"`  // Why this model was selected

	// Alternatives
	Alternatives []ModelRecommendation `json:"alternatives,omitempty"`

	// Cost estimation
	EstimatedCost CostEstimate `json:"estimated_cost"`

	// Performance prediction
	PredictedPerformance PerformancePrediction `json:"predicted_performance"`
}

// CostEstimate provides cost estimation for the selected model
type CostEstimate struct {
	InputCost       float64 `json:"input_cost"`       // Cost for input tokens
	OutputCost      float64 `json:"output_cost"`      // Estimated cost for output tokens
	TotalCost       float64 `json:"total_cost"`       // Total estimated cost
	Currency        string  `json:"currency"`         // Currency (USD, EUR, etc.)
	EstimatedTokens int     `json:"estimated_tokens"` // Estimated total tokens
}

// PerformancePrediction predicts model performance for the task
type PerformancePrediction struct {
	ExpectedLatency  string  `json:"expected_latency"`  // "fast", "medium", "slow"
	QualityScore     float64 `json:"quality_score"`     // Predicted quality (1-10)
	ReliabilityScore float64 `json:"reliability_score"` // Predicted reliability (0-1)
	SuccessRate      float64 `json:"success_rate"`      // Predicted success rate (0-1)
	TokensPerSecond  float64 `json:"tokens_per_second"` // Predicted generation speed
}

// QuickSelectOptions provides common model selection shortcuts
type QuickSelectOptions struct {
	Fastest     *ModelRecommendation `json:"fastest"`
	Cheapest    *ModelRecommendation `json:"cheapest"`
	BestQuality *ModelRecommendation `json:"best_quality"`
	Balanced    *ModelRecommendation `json:"balanced"`
	Recommended *ModelRecommendation `json:"recommended"`
}

// NewModelSelector creates a new model selector
func NewModelSelector(manager *ModelManager, registry *ModelRegistry, discovery *ModelDiscoveryService) *ModelSelector {
	return &ModelSelector{
		manager:   manager,
		registry:  registry,
		discovery: discovery,
	}
}

// SelectModel selects the best model for a given request
func (ms *ModelSelector) SelectModel(ctx context.Context, req SelectionRequest) (*SelectionResponse, error) {
	// Convert request to selection criteria
	criteria := ms.requestToCriteria(req)

	// Get recommendation from manager
	recommendation, err := ms.manager.GetRecommendation(ctx, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to get model recommendation: %w", err)
	}

	// Build response
	response := &SelectionResponse{
		SelectedModel:    recommendation.Model,
		SelectedProvider: recommendation.Provider,
		Score:            recommendation.Score,
		Confidence:       recommendation.Confidence,
		Reasoning:        recommendation.Reasoning,
		Alternatives:     recommendation.Fallbacks,
	}

	// Add cost estimation
	response.EstimatedCost = ms.estimateCost(recommendation.Model, req.InputLength)

	// Add performance prediction
	response.PredictedPerformance = ms.predictPerformance(recommendation.Model, req)

	return response, nil
}

// GetQuickSelectOptions returns common model selection options
func (ms *ModelSelector) GetQuickSelectOptions(ctx context.Context, req SelectionRequest) (*QuickSelectOptions, error) {
	// Get all suitable models
	criteria := ms.requestToCriteria(req)
	allModels := ms.registry.ListModels()
	candidates := ms.manager.filterModelsByCriteria(allModels, criteria)

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no suitable models found")
	}

	options := &QuickSelectOptions{}

	// Find fastest model (lowest latency)
	options.Fastest = ms.findFastestModel(candidates, criteria)

	// Find cheapest model (lowest cost)
	options.Cheapest = ms.findCheapestModel(candidates, criteria)

	// Find best quality model (highest quality score)
	options.BestQuality = ms.findBestQualityModel(candidates, criteria)

	// Find balanced model (good balance of speed, cost, quality)
	options.Balanced = ms.findBalancedModel(candidates, criteria)

	// Get recommended model (highest overall score)
	recommendation, err := ms.manager.GetRecommendation(ctx, criteria)
	if err == nil {
		options.Recommended = recommendation
	}

	return options, nil
}

// GetModelComparison compares multiple models for a task
func (ms *ModelSelector) GetModelComparison(ctx context.Context, modelIDs []CanonicalModelID, req SelectionRequest) ([]ModelComparison, error) {
	var comparisons []ModelComparison

	for _, modelID := range modelIDs {
		model, exists := ms.registry.GetModel(modelID)
		if !exists {
			continue
		}

		// Calculate scores and metrics for this model
		criteria := ms.requestToCriteria(req)
		score := ms.manager.calculateModelScore(model, criteria)
		cost := ms.estimateCost(model, req.InputLength)
		performance := ms.predictPerformance(model, req)

		comparison := ModelComparison{
			Model:                model,
			Score:                score,
			EstimatedCost:        cost,
			PredictedPerformance: performance,
			Pros:                 ms.getModelPros(model, req),
			Cons:                 ms.getModelCons(model, req),
		}

		comparisons = append(comparisons, comparison)
	}

	// Sort by score (highest first)
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].Score > comparisons[j].Score
	})

	return comparisons, nil
}

// ModelComparison represents a model comparison result
type ModelComparison struct {
	Model                *CanonicalModel       `json:"model"`
	Score                float64               `json:"score"`
	EstimatedCost        CostEstimate          `json:"estimated_cost"`
	PredictedPerformance PerformancePrediction `json:"predicted_performance"`
	Pros                 []string              `json:"pros"`
	Cons                 []string              `json:"cons"`
}

// Helper methods

func (ms *ModelSelector) requestToCriteria(req SelectionRequest) ModelSelectionCriteria {
	return ModelSelectionCriteria{
		TaskType:         req.TaskType,
		RequiredFeatures: req.RequiredFeatures,
		MaxCost:          req.MaxCost,
		MinQuality:       req.MinQuality,
		PreferredSpeed:   req.PreferredSpeed,
		ContextLength:    req.InputLength,
		Provider:         req.PreferredProvider,
	}
}

func (ms *ModelSelector) estimateCost(model *CanonicalModel, inputTokens int) CostEstimate {
	// Estimate output tokens (typically 1:1 to 1:3 ratio)
	estimatedOutputTokens := inputTokens / 2 // Conservative estimate

	inputCost := float64(inputTokens) * model.Pricing.InputPrice / 1000000
	outputCost := float64(estimatedOutputTokens) * model.Pricing.OutputPrice / 1000000

	return CostEstimate{
		InputCost:       inputCost,
		OutputCost:      outputCost,
		TotalCost:       inputCost + outputCost,
		Currency:        model.Pricing.Currency,
		EstimatedTokens: inputTokens + estimatedOutputTokens,
	}
}

func (ms *ModelSelector) predictPerformance(model *CanonicalModel, _ SelectionRequest) PerformancePrediction {
	// Base predictions on model family and capabilities
	var latency string
	var qualityScore float64
	var tokensPerSecond float64

	// Predict based on model family
	switch model.Family {
	case "claude":
		latency = "medium"
		qualityScore = 8.5
		tokensPerSecond = 25.0
	case "gpt":
		latency = "fast"
		qualityScore = 8.0
		tokensPerSecond = 40.0
	case "gemini":
		latency = "fast"
		qualityScore = 7.5
		tokensPerSecond = 35.0
	default:
		latency = "medium"
		qualityScore = 7.0
		tokensPerSecond = 20.0
	}

	// Adjust for model size/version
	if strings.Contains(strings.ToLower(model.Name), "mini") || strings.Contains(strings.ToLower(model.Name), "haiku") {
		latency = "fast"
		qualityScore -= 1.0
		tokensPerSecond += 20.0
	}

	return PerformancePrediction{
		ExpectedLatency:  latency,
		QualityScore:     qualityScore,
		ReliabilityScore: 0.95, // Most models are quite reliable
		SuccessRate:      0.98,
		TokensPerSecond:  tokensPerSecond,
	}
}

// Quick selection helper methods

func (ms *ModelSelector) findFastestModel(candidates []*CanonicalModel, criteria ModelSelectionCriteria) *ModelRecommendation {
	if len(candidates) == 0 {
		return nil
	}

	// Sort by predicted speed (tokens per second)
	sort.Slice(candidates, func(i, j int) bool {
		perfI := ms.predictPerformance(candidates[i], SelectionRequest{TaskType: criteria.TaskType})
		perfJ := ms.predictPerformance(candidates[j], SelectionRequest{TaskType: criteria.TaskType})
		return perfI.TokensPerSecond > perfJ.TokensPerSecond
	})

	fastest := candidates[0]
	return &ModelRecommendation{
		Model:      fastest,
		Provider:   ms.manager.getBestProvider(fastest, criteria),
		Score:      85.0,
		Reasoning:  []string{"Fastest response time", "Optimized for speed"},
		Confidence: 0.9,
	}
}

func (ms *ModelSelector) findCheapestModel(candidates []*CanonicalModel, criteria ModelSelectionCriteria) *ModelRecommendation {
	if len(candidates) == 0 {
		return nil
	}

	// Sort by cost (output price as primary factor)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Pricing.OutputPrice < candidates[j].Pricing.OutputPrice
	})

	cheapest := candidates[0]
	return &ModelRecommendation{
		Model:      cheapest,
		Provider:   ms.manager.getBestProvider(cheapest, criteria),
		Score:      80.0,
		Reasoning:  []string{"Most cost-effective option", "Low per-token pricing"},
		Confidence: 0.95,
	}
}

func (ms *ModelSelector) findBestQualityModel(candidates []*CanonicalModel, criteria ModelSelectionCriteria) *ModelRecommendation {
	if len(candidates) == 0 {
		return nil
	}

	// Sort by predicted quality
	sort.Slice(candidates, func(i, j int) bool {
		perfI := ms.predictPerformance(candidates[i], SelectionRequest{TaskType: criteria.TaskType})
		perfJ := ms.predictPerformance(candidates[j], SelectionRequest{TaskType: criteria.TaskType})
		return perfI.QualityScore > perfJ.QualityScore
	})

	bestQuality := candidates[0]
	return &ModelRecommendation{
		Model:      bestQuality,
		Provider:   ms.manager.getBestProvider(bestQuality, criteria),
		Score:      95.0,
		Reasoning:  []string{"Highest quality output", "Best-in-class performance"},
		Confidence: 0.9,
	}
}

func (ms *ModelSelector) findBalancedModel(candidates []*CanonicalModel, criteria ModelSelectionCriteria) *ModelRecommendation {
	if len(candidates) == 0 {
		return nil
	}

	// Score models based on balanced criteria (cost, speed, quality)
	type scoredModel struct {
		model *CanonicalModel
		score float64
	}

	var scored []scoredModel
	for _, model := range candidates {
		perf := ms.predictPerformance(model, SelectionRequest{TaskType: criteria.TaskType})

		// Balanced scoring: quality (40%) + speed (30%) + cost efficiency (30%)
		qualityScore := perf.QualityScore / 10.0 * 40.0
		speedScore := min(perf.TokensPerSecond/50.0, 1.0) * 30.0
		costScore := (1.0 - min(model.Pricing.OutputPrice/20.0, 1.0)) * 30.0

		totalScore := qualityScore + speedScore + costScore
		scored = append(scored, scoredModel{model: model, score: totalScore})
	}

	// Sort by balanced score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	balanced := scored[0].model
	return &ModelRecommendation{
		Model:      balanced,
		Provider:   ms.manager.getBestProvider(balanced, criteria),
		Score:      scored[0].score,
		Reasoning:  []string{"Well-balanced option", "Good mix of quality, speed, and cost"},
		Confidence: 0.85,
	}
}

func (ms *ModelSelector) getModelPros(model *CanonicalModel, _ SelectionRequest) []string {
	var pros []string

	// Capability-based pros
	if model.Capabilities.SupportsVision {
		pros = append(pros, "Supports image analysis")
	}
	if model.Capabilities.SupportsReasoning {
		pros = append(pros, "Advanced reasoning capabilities")
	}
	if model.Capabilities.SupportsTools {
		pros = append(pros, "Function calling support")
	}
	if model.Capabilities.SupportsStreaming {
		pros = append(pros, "Real-time streaming responses")
	}

	// Cost-based pros
	if model.Pricing.OutputPrice < 5.0 {
		pros = append(pros, "Cost-effective pricing")
	}

	// Context-based pros
	if model.Limits.ContextWindow >= 100000 {
		pros = append(pros, "Large context window")
	}

	// Family-based pros
	switch model.Family {
	case "claude":
		pros = append(pros, "Excellent for complex reasoning", "High-quality outputs")
	case "gpt":
		pros = append(pros, "Fast response times", "Versatile capabilities")
	case "gemini":
		pros = append(pros, "Strong multimodal support", "Good value for money")
	}

	return pros
}

func (ms *ModelSelector) getModelCons(model *CanonicalModel, req SelectionRequest) []string {
	var cons []string

	// Cost-based cons
	if model.Pricing.OutputPrice > 15.0 {
		cons = append(cons, "Higher cost per token")
	}

	// Capability-based cons
	if !model.Capabilities.SupportsVision && contains(req.RequiredFeatures, "vision") {
		cons = append(cons, "No image analysis support")
	}
	if !model.Capabilities.SupportsReasoning && req.TaskType == "reasoning" {
		cons = append(cons, "Limited reasoning capabilities")
	}

	// Context-based cons
	if model.Limits.ContextWindow < 50000 {
		cons = append(cons, "Limited context window")
	}

	// Family-based cons
	switch model.Family {
	case "claude":
		cons = append(cons, "Can be slower for simple tasks")
	case "gpt":
		cons = append(cons, "May lack depth for complex reasoning")
	case "gemini":
		cons = append(cons, "Newer model with less track record")
	}

	return cons
}
