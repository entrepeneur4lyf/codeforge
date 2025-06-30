package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/models"
)

// CostPeriod represents different time periods for cost tracking
type CostPeriod string

const (
	PeriodHourly  CostPeriod = "hourly"
	PeriodDaily   CostPeriod = "daily"
	PeriodWeekly  CostPeriod = "weekly"
	PeriodMonthly CostPeriod = "monthly"
)

// TokenUsageRecord represents a single token usage record
type TokenUsageRecord struct {
	ID        string               `json:"id"`
	Timestamp time.Time            `json:"timestamp"`
	SessionID string               `json:"session_id"`
	UserID    string               `json:"user_id,omitempty"`
	ModelID   models.ModelID       `json:"model_id"`
	Provider  models.ModelProvider `json:"provider"`

	// Token counts
	InputTokens     int64 `json:"input_tokens"`
	OutputTokens    int64 `json:"output_tokens"`
	CachedTokens    int64 `json:"cached_tokens"`
	ReasoningTokens int64 `json:"reasoning_tokens"`
	TotalTokens     int64 `json:"total_tokens"`

	// Costs
	InputCost     float64 `json:"input_cost"`
	OutputCost    float64 `json:"output_cost"`
	CachedCost    float64 `json:"cached_cost"`
	ReasoningCost float64 `json:"reasoning_cost"`
	TotalCost     float64 `json:"total_cost"`

	// Performance metrics
	Latency      time.Duration `json:"latency"`
	Success      bool          `json:"success"`
	ErrorMessage string        `json:"error_message,omitempty"`

	// Request metadata
	RequestType  string  `json:"request_type"` // chat, completion, embedding, etc.
	ContextSize  int     `json:"context_size"`
	ResponseSize int     `json:"response_size"`
	Temperature  float64 `json:"temperature"`
	MaxTokens    int     `json:"max_tokens"`
}

// CostSummary represents aggregated cost data for a period
type CostSummary struct {
	Period    CostPeriod `json:"period"`
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time"`

	// Aggregated token counts
	TotalInputTokens  int64 `json:"total_input_tokens"`
	TotalOutputTokens int64 `json:"total_output_tokens"`
	TotalCachedTokens int64 `json:"total_cached_tokens"`
	TotalTokens       int64 `json:"total_tokens"`

	// Aggregated costs
	TotalCost  float64 `json:"total_cost"`
	InputCost  float64 `json:"input_cost"`
	OutputCost float64 `json:"output_cost"`
	CachedCost float64 `json:"cached_cost"`

	// Breakdown by model
	ModelBreakdown map[models.ModelID]*ModelCostSummary `json:"model_breakdown"`

	// Breakdown by provider
	ProviderBreakdown map[models.ModelProvider]*ProviderCostSummary `json:"provider_breakdown"`

	// Performance metrics
	AverageLatency time.Duration `json:"average_latency"`
	SuccessRate    float64       `json:"success_rate"`
	TotalRequests  int64         `json:"total_requests"`
	FailedRequests int64         `json:"failed_requests"`
}

// ModelCostSummary represents cost summary for a specific model
type ModelCostSummary struct {
	ModelID        models.ModelID `json:"model_id"`
	TotalCost      float64        `json:"total_cost"`
	TotalTokens    int64          `json:"total_tokens"`
	TotalRequests  int64          `json:"total_requests"`
	AverageLatency time.Duration  `json:"average_latency"`
	SuccessRate    float64        `json:"success_rate"`
	CostPerToken   float64        `json:"cost_per_token"`
	CostPerRequest float64        `json:"cost_per_request"`
}

// ProviderCostSummary represents cost summary for a specific provider
type ProviderCostSummary struct {
	Provider       models.ModelProvider `json:"provider"`
	TotalCost      float64              `json:"total_cost"`
	TotalTokens    int64                `json:"total_tokens"`
	TotalRequests  int64                `json:"total_requests"`
	AverageLatency time.Duration        `json:"average_latency"`
	SuccessRate    float64              `json:"success_rate"`
	ModelsUsed     []models.ModelID     `json:"models_used"`
}

// BudgetAlert represents a budget alert configuration
type BudgetAlert struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Period           CostPeriod    `json:"period"`
	Threshold        float64       `json:"threshold"`         // Dollar amount
	ThresholdPercent float64       `json:"threshold_percent"` // Percentage of budget
	Enabled          bool          `json:"enabled"`
	LastTriggered    time.Time     `json:"last_triggered,omitempty"`
	Actions          []AlertAction `json:"actions"`
}

// AlertAction represents an action to take when an alert is triggered
type AlertAction struct {
	Type   AlertActionType        `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// AlertActionType defines different types of alert actions
type AlertActionType string

const (
	ActionNotify       AlertActionType = "notify"
	ActionSlowDown     AlertActionType = "slow_down"
	ActionSwitchModel  AlertActionType = "switch_model"
	ActionStopRequests AlertActionType = "stop_requests"
	ActionEmail        AlertActionType = "email"
	ActionWebhook      AlertActionType = "webhook"
)

// CostOptimizationRecommendation represents a cost optimization suggestion
type CostOptimizationRecommendation struct {
	Type             OptimizationType `json:"type"`
	Title            string           `json:"title"`
	Description      string           `json:"description"`
	PotentialSavings float64          `json:"potential_savings"`
	Confidence       float64          `json:"confidence"`
	Priority         int              `json:"priority"`
	Actions          []string         `json:"actions"`
	ModelSuggestions []models.ModelID `json:"model_suggestions,omitempty"`
}

// OptimizationType defines different types of cost optimizations
type OptimizationType string

const (
	OptimizationModelSwitch      OptimizationType = "model_switch"
	OptimizationProviderSwitch   OptimizationType = "provider_switch"
	OptimizationContextReduction OptimizationType = "context_reduction"
	OptimizationCaching          OptimizationType = "caching"
	OptimizationBatching         OptimizationType = "batching"
	OptimizationRateLimit        OptimizationType = "rate_limit"
)

// CostTracker manages cost tracking and optimization
type CostTracker struct {
	records   []TokenUsageRecord
	summaries map[string]*CostSummary
	alerts    map[string]*BudgetAlert
	mu        sync.RWMutex

	// Configuration
	maxRecords      int
	retentionPeriod time.Duration

	// Budget tracking
	budgets         map[CostPeriod]float64
	currentSpending map[CostPeriod]float64

	// Optimization
	optimizationEnabled bool
	lastOptimization    time.Time
}

// NewCostTracker creates a new cost tracker
func NewCostTracker() *CostTracker {
	return &CostTracker{
		records:             make([]TokenUsageRecord, 0),
		summaries:           make(map[string]*CostSummary),
		alerts:              make(map[string]*BudgetAlert),
		maxRecords:          10000,
		retentionPeriod:     30 * 24 * time.Hour, // 30 days
		budgets:             make(map[CostPeriod]float64),
		currentSpending:     make(map[CostPeriod]float64),
		optimizationEnabled: true,
	}
}

// RecordUsage records a token usage event
func (ct *CostTracker) RecordUsage(record TokenUsageRecord) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Set timestamp if not provided
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	// Generate ID if not provided
	if record.ID == "" {
		record.ID = fmt.Sprintf("%d-%s", record.Timestamp.Unix(), record.SessionID)
	}

	// Calculate total tokens and cost if not provided
	if record.TotalTokens == 0 {
		record.TotalTokens = record.InputTokens + record.OutputTokens + record.CachedTokens + record.ReasoningTokens
	}

	if record.TotalCost == 0 {
		record.TotalCost = record.InputCost + record.OutputCost + record.CachedCost + record.ReasoningCost
	}

	// Add to records
	ct.records = append(ct.records, record)

	// Update current spending
	ct.updateCurrentSpending(record)

	// Trim old records if necessary
	ct.trimRecords()

	// Check budget alerts
	ct.checkBudgetAlerts(record)

	return nil
}

// GetCostSummary returns cost summary for a specific period
func (ct *CostTracker) GetCostSummary(period CostPeriod, startTime, endTime time.Time) *CostSummary {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	summary := &CostSummary{
		Period:            period,
		StartTime:         startTime,
		EndTime:           endTime,
		ModelBreakdown:    make(map[models.ModelID]*ModelCostSummary),
		ProviderBreakdown: make(map[models.ModelProvider]*ProviderCostSummary),
	}

	var totalLatency time.Duration
	var requestCount int64

	for _, record := range ct.records {
		if record.Timestamp.After(startTime) && record.Timestamp.Before(endTime) {
			// Aggregate totals
			summary.TotalInputTokens += record.InputTokens
			summary.TotalOutputTokens += record.OutputTokens
			summary.TotalCachedTokens += record.CachedTokens
			summary.TotalTokens += record.TotalTokens
			summary.TotalCost += record.TotalCost
			summary.InputCost += record.InputCost
			summary.OutputCost += record.OutputCost
			summary.CachedCost += record.CachedCost
			summary.TotalRequests++

			if !record.Success {
				summary.FailedRequests++
			}

			totalLatency += record.Latency
			requestCount++

			// Update model breakdown
			if modelSummary, exists := summary.ModelBreakdown[record.ModelID]; exists {
				modelSummary.TotalCost += record.TotalCost
				modelSummary.TotalTokens += record.TotalTokens
				modelSummary.TotalRequests++
			} else {
				summary.ModelBreakdown[record.ModelID] = &ModelCostSummary{
					ModelID:       record.ModelID,
					TotalCost:     record.TotalCost,
					TotalTokens:   record.TotalTokens,
					TotalRequests: 1,
				}
			}

			// Update provider breakdown
			if providerSummary, exists := summary.ProviderBreakdown[record.Provider]; exists {
				providerSummary.TotalCost += record.TotalCost
				providerSummary.TotalTokens += record.TotalTokens
				providerSummary.TotalRequests++
			} else {
				summary.ProviderBreakdown[record.Provider] = &ProviderCostSummary{
					Provider:      record.Provider,
					TotalCost:     record.TotalCost,
					TotalTokens:   record.TotalTokens,
					TotalRequests: 1,
					ModelsUsed:    []models.ModelID{record.ModelID},
				}
			}
		}
	}

	// Calculate averages
	if requestCount > 0 {
		summary.AverageLatency = totalLatency / time.Duration(requestCount)
		summary.SuccessRate = float64(summary.TotalRequests-summary.FailedRequests) / float64(summary.TotalRequests)
	}

	// Calculate per-model metrics
	for _, modelSummary := range summary.ModelBreakdown {
		if modelSummary.TotalTokens > 0 {
			modelSummary.CostPerToken = modelSummary.TotalCost / float64(modelSummary.TotalTokens)
		}
		if modelSummary.TotalRequests > 0 {
			modelSummary.CostPerRequest = modelSummary.TotalCost / float64(modelSummary.TotalRequests)
		}
	}

	return summary
}

// GetCurrentSpending returns current spending for a period
func (ct *CostTracker) GetCurrentSpending(period CostPeriod) float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return ct.currentSpending[period]
}

// SetBudget sets a budget for a specific period
func (ct *CostTracker) SetBudget(period CostPeriod, amount float64) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.budgets[period] = amount
}

// GetBudget returns the budget for a specific period
func (ct *CostTracker) GetBudget(period CostPeriod) float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	return ct.budgets[period]
}

// IsWithinBudget checks if a cost would exceed the budget
func (ct *CostTracker) IsWithinBudget(period CostPeriod, additionalCost float64) bool {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	budget := ct.budgets[period]
	if budget <= 0 {
		return true // No budget set
	}

	currentSpending := ct.currentSpending[period]
	return (currentSpending + additionalCost) <= budget
}

// AddBudgetAlert adds a budget alert
func (ct *CostTracker) AddBudgetAlert(alert *BudgetAlert) {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	ct.alerts[alert.ID] = alert
}

// GetOptimizationRecommendations returns cost optimization recommendations
func (ct *CostTracker) GetOptimizationRecommendations() []CostOptimizationRecommendation {
	if !ct.optimizationEnabled {
		return nil
	}

	ct.mu.RLock()
	defer ct.mu.RUnlock()

	var recommendations []CostOptimizationRecommendation

	// Analyze recent usage patterns
	recentSummary := ct.getRecentSummary()

	// Check for expensive models that could be replaced
	recommendations = append(recommendations, ct.analyzeModelUsage(recentSummary)...)

	// Check for high token usage patterns
	recommendations = append(recommendations, ct.analyzeTokenUsage(recentSummary)...)

	// Check for provider optimization opportunities
	recommendations = append(recommendations, ct.analyzeProviderUsage(recentSummary)...)

	return recommendations
}

// EstimateCost estimates the cost for a request
func (ct *CostTracker) EstimateCost(modelID models.ModelID, inputTokens, outputTokens int64) float64 {
	// This would use the model configuration to calculate estimated cost
	// For now, return a simple estimate
	model, exists := models.SupportedModels[modelID]
	if !exists {
		return 0.001 * float64(inputTokens+outputTokens) // Default rate
	}

	inputCost := (model.CostPer1MIn / 1000000) * float64(inputTokens)
	outputCost := (model.CostPer1MOut / 1000000) * float64(outputTokens)

	return inputCost + outputCost
}

// updateCurrentSpending updates current spending for all periods
func (ct *CostTracker) updateCurrentSpending(record TokenUsageRecord) {
	now := time.Now()

	// Update hourly spending
	hourStart := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	if record.Timestamp.After(hourStart) {
		ct.currentSpending[PeriodHourly] += record.TotalCost
	}

	// Update daily spending
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if record.Timestamp.After(dayStart) {
		ct.currentSpending[PeriodDaily] += record.TotalCost
	}

	// Update monthly spending
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	if record.Timestamp.After(monthStart) {
		ct.currentSpending[PeriodMonthly] += record.TotalCost
	}
}

// trimRecords removes old records to maintain size limits
func (ct *CostTracker) trimRecords() {
	if len(ct.records) <= ct.maxRecords {
		return
	}

	// Remove oldest records
	removeCount := len(ct.records) - ct.maxRecords
	ct.records = ct.records[removeCount:]
}

// checkBudgetAlerts checks if any budget alerts should be triggered
func (ct *CostTracker) checkBudgetAlerts(_ TokenUsageRecord) {
	for _, alert := range ct.alerts {
		if !alert.Enabled {
			continue
		}

		currentSpending := ct.currentSpending[alert.Period]
		budget := ct.budgets[alert.Period]

		if budget > 0 {
			percentage := (currentSpending / budget) * 100

			if (alert.Threshold > 0 && currentSpending >= alert.Threshold) ||
				(alert.ThresholdPercent > 0 && percentage >= alert.ThresholdPercent) {

				// Trigger alert (simplified - would send notifications in real implementation)
				alert.LastTriggered = time.Now()
			}
		}
	}
}

// getRecentSummary returns a summary of recent usage
func (ct *CostTracker) getRecentSummary() *CostSummary {
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour) // Last 24 hours

	return ct.GetCostSummary(PeriodDaily, startTime, endTime)
}

// analyzeModelUsage analyzes model usage for optimization opportunities
func (ct *CostTracker) analyzeModelUsage(summary *CostSummary) []CostOptimizationRecommendation {
	var recommendations []CostOptimizationRecommendation

	// Find expensive models with low usage
	for modelID, modelSummary := range summary.ModelBreakdown {
		if modelSummary.CostPerRequest > 0.10 && modelSummary.TotalRequests < 10 {
			recommendations = append(recommendations, CostOptimizationRecommendation{
				Type:             OptimizationModelSwitch,
				Title:            fmt.Sprintf("Consider switching from %s", modelID),
				Description:      fmt.Sprintf("Model %s has high cost per request (%.4f) with low usage", modelID, modelSummary.CostPerRequest),
				PotentialSavings: modelSummary.TotalCost * 0.5, // Estimate 50% savings
				Confidence:       0.7,
				Priority:         2,
			})
		}
	}

	return recommendations
}

// analyzeTokenUsage analyzes token usage patterns
func (ct *CostTracker) analyzeTokenUsage(summary *CostSummary) []CostOptimizationRecommendation {
	var recommendations []CostOptimizationRecommendation

	// Check for high input token usage (could benefit from context optimization)
	if summary.TotalRequests > 0 {
		avgInputTokens := float64(summary.TotalInputTokens) / float64(summary.TotalRequests)
		if avgInputTokens > 8000 {
			recommendations = append(recommendations, CostOptimizationRecommendation{
				Type:             OptimizationContextReduction,
				Title:            "High context usage detected",
				Description:      fmt.Sprintf("Average input tokens: %.0f. Consider context optimization", avgInputTokens),
				PotentialSavings: summary.InputCost * 0.3, // Estimate 30% savings
				Confidence:       0.8,
				Priority:         1,
			})
		}
	}

	return recommendations
}

// analyzeProviderUsage analyzes provider usage for optimization
func (ct *CostTracker) analyzeProviderUsage(summary *CostSummary) []CostOptimizationRecommendation {
	var recommendations []CostOptimizationRecommendation

	// Find most expensive provider
	var mostExpensive models.ModelProvider
	var highestCost float64

	for provider, providerSummary := range summary.ProviderBreakdown {
		costPerRequest := providerSummary.TotalCost / float64(providerSummary.TotalRequests)
		if costPerRequest > highestCost {
			highestCost = costPerRequest
			mostExpensive = provider
		}
	}

	// Suggest switching if cost is significantly higher
	if highestCost > 0.05 && len(summary.ProviderBreakdown) > 1 {
		recommendations = append(recommendations, CostOptimizationRecommendation{
			Type:             OptimizationProviderSwitch,
			Title:            fmt.Sprintf("Consider switching from %s provider", mostExpensive),
			Description:      fmt.Sprintf("Provider %s has high cost per request (%.4f)", mostExpensive, highestCost),
			PotentialSavings: summary.ProviderBreakdown[mostExpensive].TotalCost * 0.4,
			Confidence:       0.6,
			Priority:         3,
		})
	}

	return recommendations
}
