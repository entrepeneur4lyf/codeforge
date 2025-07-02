package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/models"
)

// ModelCapability represents different model capabilities
type ModelCapability string

const (
	CapabilityReasoning      ModelCapability = "reasoning"
	CapabilityToolCalling    ModelCapability = "tool_calling"
	CapabilityVision         ModelCapability = "vision"
	CapabilityCodeGeneration ModelCapability = "code_generation"
	CapabilityLongContext    ModelCapability = "long_context"
	CapabilityStreaming      ModelCapability = "streaming"
	CapabilityFunctionCall   ModelCapability = "function_call"
	CapabilityJSONMode       ModelCapability = "json_mode"
)

// ModelPerformanceMetrics tracks performance data for models
type ModelPerformanceMetrics struct {
	AverageLatency   time.Duration `json:"average_latency"`
	TokensPerSecond  float64       `json:"tokens_per_second"`
	SuccessRate      float64       `json:"success_rate"`
	ErrorRate        float64       `json:"error_rate"`
	LastUpdated      time.Time     `json:"last_updated"`
	TotalRequests    int64         `json:"total_requests"`
	TotalTokens      int64         `json:"total_tokens"`
	TotalCost        float64       `json:"total_cost"`
	QualityScore     float64       `json:"quality_score"`     // 0.0-1.0
	ReliabilityScore float64       `json:"reliability_score"` // 0.0-1.0
}

// ModelLimits defines operational limits for models
type ModelLimits struct {
	MaxRequestsPerMinute int           `json:"max_requests_per_minute"`
	MaxTokensPerRequest  int           `json:"max_tokens_per_request"`
	MaxConcurrentReqs    int           `json:"max_concurrent_requests"`
	RequestTimeout       time.Duration `json:"request_timeout"`
	CooldownPeriod       time.Duration `json:"cooldown_period"`
	MaxRetries           int           `json:"max_retries"`
	BackoffMultiplier    float64       `json:"backoff_multiplier"`
}

// ModelFallbackConfig defines fallback behavior
type ModelFallbackConfig struct {
	Enabled           bool              `json:"enabled"`
	FallbackModels    []models.CanonicalModelID  `json:"fallback_models"`
	TriggerConditions []FallbackTrigger `json:"trigger_conditions"`
	MaxFallbackDepth  int               `json:"max_fallback_depth"`
	FallbackDelay     time.Duration     `json:"fallback_delay"`
}

// FallbackTrigger defines when to trigger fallback
type FallbackTrigger string

const (
	TriggerRateLimit   FallbackTrigger = "rate_limit"
	TriggerTimeout     FallbackTrigger = "timeout"
	TriggerError       FallbackTrigger = "error"
	TriggerHighLatency FallbackTrigger = "high_latency"
	TriggerLowQuality  FallbackTrigger = "low_quality"
	TriggerCostLimit   FallbackTrigger = "cost_limit"
)

// EnhancedModelConfig extends the basic ModelConfig with advanced features
type EnhancedModelConfig struct {
	// Basic configuration (from existing ModelConfig)
	ContextWindow      int     `json:"context_window"`
	MaxOutputTokens    int     `json:"max_output_tokens"`
	CostPer1KInput     float64 `json:"cost_per_1k_input"`
	CostPer1KOutput    float64 `json:"cost_per_1k_output"`
	CostPer1KCached    float64 `json:"cost_per_1k_cached"`
	SupportsTools      bool    `json:"supports_tools"`
	SupportsReasoning  bool    `json:"supports_reasoning"`
	SummarizeThreshold float64 `json:"summarize_threshold"`

	// Enhanced capabilities
	Capabilities []ModelCapability `json:"capabilities"`

	// Performance and limits
	Performance ModelPerformanceMetrics `json:"performance"`
	Limits      ModelLimits             `json:"limits"`
	Fallback    ModelFallbackConfig     `json:"fallback"`

	// Optimization settings
	OptimalTemperature   float64 `json:"optimal_temperature"`
	OptimalTopP          float64 `json:"optimal_top_p"`
	RecommendedMaxTokens int     `json:"recommended_max_tokens"`
	PreferredBatchSize   int     `json:"preferred_batch_size"`

	// Quality and reliability
	QualityThreshold     float64    `json:"quality_threshold"`
	ReliabilityThreshold float64    `json:"reliability_threshold"`
	MaintenanceWindows   []string   `json:"maintenance_windows"`
	DeprecationDate      *time.Time `json:"deprecation_date,omitempty"`

	// Cost management
	CostBudgetPerHour  float64 `json:"cost_budget_per_hour"`
	CostBudgetPerDay   float64 `json:"cost_budget_per_day"`
	CostAlertThreshold float64 `json:"cost_alert_threshold"`

	// Usage tracking
	LastUsed        time.Time `json:"last_used"`
	UsageCount      int64     `json:"usage_count"`
	PreferenceScore float64   `json:"preference_score"`
}

// ModelConfigManager manages enhanced model configurations
type ModelConfigManager struct {
	configs map[models.CanonicalModelID]*EnhancedModelConfig
	mu      sync.RWMutex
}

// NewModelConfigManager creates a new model configuration manager
func NewModelConfigManager() *ModelConfigManager {
	return &ModelConfigManager{
		configs: make(map[models.CanonicalModelID]*EnhancedModelConfig),
	}
}

// GetModelConfig returns the enhanced configuration for a model
func (mcm *ModelConfigManager) GetModelConfig(modelID models.CanonicalModelID) *EnhancedModelConfig {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	if config, exists := mcm.configs[modelID]; exists {
		return config
	}

	// Return default configuration if not found
	return mcm.getDefaultConfig(modelID)
}

// SetModelConfig sets the configuration for a model
func (mcm *ModelConfigManager) SetModelConfig(modelID models.CanonicalModelID, config *EnhancedModelConfig) {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()

	mcm.configs[modelID] = config
}

// UpdatePerformanceMetrics updates performance metrics for a model
func (mcm *ModelConfigManager) UpdatePerformanceMetrics(modelID models.CanonicalModelID, latency time.Duration, tokens int64, cost float64, success bool) {
	mcm.mu.Lock()
	defer mcm.mu.Unlock()

	config := mcm.configs[modelID]
	if config == nil {
		config = mcm.getDefaultConfig(modelID)
		mcm.configs[modelID] = config
	}

	// Update metrics
	config.Performance.TotalRequests++
	config.Performance.TotalTokens += tokens
	config.Performance.TotalCost += cost
	config.Performance.LastUpdated = time.Now()
	config.LastUsed = time.Now()
	config.UsageCount++

	// Calculate running averages
	if config.Performance.TotalRequests > 0 {
		config.Performance.AverageLatency = time.Duration(
			(int64(config.Performance.AverageLatency)*config.Performance.TotalRequests + int64(latency)) /
				(config.Performance.TotalRequests + 1),
		)

		if latency > 0 {
			config.Performance.TokensPerSecond = float64(tokens) / latency.Seconds()
		}
	}

	// Update success/error rates
	if success {
		config.Performance.SuccessRate = (config.Performance.SuccessRate*float64(config.Performance.TotalRequests-1) + 1.0) / float64(config.Performance.TotalRequests)
	} else {
		config.Performance.ErrorRate = (config.Performance.ErrorRate*float64(config.Performance.TotalRequests-1) + 1.0) / float64(config.Performance.TotalRequests)
	}
}

// ShouldFallback determines if a model should fallback based on current conditions
func (mcm *ModelConfigManager) ShouldFallback(modelID models.CanonicalModelID, trigger FallbackTrigger) bool {
	config := mcm.GetModelConfig(modelID)
	if !config.Fallback.Enabled {
		return false
	}

	for _, condition := range config.Fallback.TriggerConditions {
		if condition == trigger {
			return true
		}
	}

	return false
}

// GetFallbackModel returns the next fallback model for a given model
func (mcm *ModelConfigManager) GetFallbackModel(modelID models.CanonicalModelID, depth int) (models.CanonicalModelID, error) {
	config := mcm.GetModelConfig(modelID)

	if !config.Fallback.Enabled || depth >= config.Fallback.MaxFallbackDepth {
		return "", fmt.Errorf("no fallback available for model %s at depth %d", modelID, depth)
	}

	if depth < len(config.Fallback.FallbackModels) {
		return config.Fallback.FallbackModels[depth], nil
	}

	return "", fmt.Errorf("no fallback model available at depth %d", depth)
}

// IsWithinBudget checks if using a model is within cost budget
func (mcm *ModelConfigManager) IsWithinBudget(modelID models.CanonicalModelID, estimatedCost float64) bool {
	config := mcm.GetModelConfig(modelID)

	// Check hourly budget
	if config.CostBudgetPerHour > 0 {
		// Calculate current hourly spending for this model
		hourlySpending := mcm.calculateHourlySpending(modelID)
		if (hourlySpending + estimatedCost) > config.CostBudgetPerHour {
			return false
		}
	}

	// Check daily budget
	if config.CostBudgetPerDay > 0 {
		// Calculate current daily spending for this model
		dailySpending := mcm.calculateDailySpending(modelID)
		if (dailySpending + estimatedCost) > config.CostBudgetPerDay {
			return false
		}
	}

	return true
}

// getDefaultConfig returns a default configuration for a model
func (mcm *ModelConfigManager) getDefaultConfig(modelID models.CanonicalModelID) *EnhancedModelConfig {
	// Get canonical model info
	model, exists := models.GetCanonicalModel(modelID)
	if !exists {
		// Return generic default
		return &EnhancedModelConfig{
			ContextWindow:      8192,
			MaxOutputTokens:    2048,
			CostPer1KInput:     0.001,
			CostPer1KOutput:    0.002,
			SummarizeThreshold: 0.9,
			Capabilities:       []ModelCapability{CapabilityCodeGeneration},
			Limits: ModelLimits{
				MaxRequestsPerMinute: 60,
				MaxTokensPerRequest:  8192,
				MaxConcurrentReqs:    5,
				RequestTimeout:       30 * time.Second,
				MaxRetries:           3,
				BackoffMultiplier:    2.0,
			},
			Fallback: ModelFallbackConfig{
				Enabled:          true,
				MaxFallbackDepth: 2,
				FallbackDelay:    1 * time.Second,
				TriggerConditions: []FallbackTrigger{
					TriggerRateLimit,
					TriggerTimeout,
					TriggerError,
				},
			},
			OptimalTemperature:   0.7,
			OptimalTopP:          0.9,
			QualityThreshold:     0.8,
			ReliabilityThreshold: 0.95,
		}
	}

	// Create enhanced config from canonical model
	config := &EnhancedModelConfig{
		ContextWindow:      model.Limits.ContextWindow,
		MaxOutputTokens:    model.Limits.MaxOutputTokens,
		CostPer1KInput:     model.Pricing.InputPrice / 1000,
		CostPer1KOutput:    model.Pricing.OutputPrice / 1000,
		CostPer1KCached:    model.Pricing.CacheReadsPrice / 1000,
		SummarizeThreshold: 0.9,

		// Set capabilities based on model features
		Capabilities: mcm.inferCanonicalCapabilities(model),

		// Default limits
		Limits: ModelLimits{
			MaxRequestsPerMinute: 60,
			MaxTokensPerRequest:  model.Limits.MaxOutputTokens,
			MaxConcurrentReqs:    5,
			RequestTimeout:       30 * time.Second,
			MaxRetries:           3,
			BackoffMultiplier:    2.0,
		},

		// Default fallback configuration
		Fallback: ModelFallbackConfig{
			Enabled:          true,
			MaxFallbackDepth: 2,
			FallbackDelay:    1 * time.Second,
			TriggerConditions: []FallbackTrigger{
				TriggerRateLimit,
				TriggerTimeout,
				TriggerError,
			},
		},

		// Optimization defaults
		OptimalTemperature:   0.7,
		OptimalTopP:          0.9,
		QualityThreshold:     0.8,
		ReliabilityThreshold: 0.95,
	}

	// Set model-specific optimizations
	mcm.setCanonicalModelSpecificDefaults(config, model)

	return config
}

// calculateHourlySpending calculates spending for a model in the last hour
func (mcm *ModelConfigManager) calculateHourlySpending(modelID models.CanonicalModelID) float64 {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	config := mcm.GetModelConfig(modelID)

	// In a production system, this would query actual usage records from the last hour
	// For now, we'll estimate based on recent performance metrics

	// Estimate hourly requests as a fraction of total requests
	hourlyRequestEstimate := float64(config.Performance.TotalRequests) * 0.04 // ~1/24th for hourly
	avgCost := (config.CostPer1KInput + config.CostPer1KOutput) / 2
	totalSpending := hourlyRequestEstimate * avgCost

	return totalSpending
}

// calculateDailySpending calculates spending for a model in the last 24 hours
func (mcm *ModelConfigManager) calculateDailySpending(modelID models.CanonicalModelID) float64 {
	mcm.mu.RLock()
	defer mcm.mu.RUnlock()

	config := mcm.GetModelConfig(modelID)

	// In a production system, this would query actual usage records from the last 24 hours
	// For now, we'll use the total cost from performance metrics as a proxy

	// Use the total cost tracked in performance metrics
	// In a real system, this would be filtered to the last 24 hours
	totalSpending := config.Performance.TotalCost

	return totalSpending
}

// inferCanonicalCapabilities infers capabilities from canonical model features
func (mcm *ModelConfigManager) inferCanonicalCapabilities(model *models.CanonicalModel) []ModelCapability {
	var capabilities []ModelCapability

	// Always assume code generation for our use case
	capabilities = append(capabilities, CapabilityCodeGeneration)

	if model.Capabilities.SupportsReasoning {
		capabilities = append(capabilities, CapabilityReasoning)
	}

	if model.Capabilities.SupportsVision {
		capabilities = append(capabilities, CapabilityVision)
	}

	if model.Capabilities.SupportsTools {
		capabilities = append(capabilities, CapabilityToolCalling)
	}

	if model.Capabilities.SupportsStreaming {
		capabilities = append(capabilities, CapabilityStreaming)
	}

	// Infer other capabilities based on model features
	if model.Limits.ContextWindow > 100000 {
		capabilities = append(capabilities, CapabilityLongContext)
	}

	return capabilities
}

// setCanonicalModelSpecificDefaults sets model-specific optimization defaults for canonical models
func (mcm *ModelConfigManager) setCanonicalModelSpecificDefaults(config *EnhancedModelConfig, model *models.CanonicalModel) {
	// Set defaults based on model family
	switch model.Family {
	case "claude":
		config.OptimalTemperature = 0.7
		config.RecommendedMaxTokens = 4096
		config.PreferredBatchSize = 1

	case "gpt", "o1", "o3":
		config.OptimalTemperature = 0.7
		config.RecommendedMaxTokens = 4096
		config.PreferredBatchSize = 1

	case "gemini":
		config.OptimalTemperature = 0.9
		config.RecommendedMaxTokens = 8192
		config.PreferredBatchSize = 1

	case "llama":
		config.OptimalTemperature = 0.7
		config.RecommendedMaxTokens = 8192
		config.PreferredBatchSize = 1
		// GROQ/Llama is fast, allow higher concurrency
		if _, hasGroq := model.Providers[models.ProviderGroqCanonical]; hasGroq {
			config.Limits.MaxConcurrentReqs = 10
		}
	}
}
