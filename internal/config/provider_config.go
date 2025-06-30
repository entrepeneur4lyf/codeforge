package config

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/models"
)

// ProviderStatus represents the health status of a provider
type ProviderStatus string

const (
	StatusHealthy     ProviderStatus = "healthy"
	StatusDegraded    ProviderStatus = "degraded"
	StatusUnhealthy   ProviderStatus = "unhealthy"
	StatusMaintenance ProviderStatus = "maintenance"
	StatusUnknown     ProviderStatus = "unknown"
)

// ProviderHealthMetrics tracks health and performance metrics for providers
type ProviderHealthMetrics struct {
	Status              ProviderStatus `json:"status"`
	LastHealthCheck     time.Time      `json:"last_health_check"`
	ResponseTime        time.Duration  `json:"response_time"`
	SuccessRate         float64        `json:"success_rate"`
	ErrorRate           float64        `json:"error_rate"`
	RateLimitHits       int64          `json:"rate_limit_hits"`
	TotalRequests       int64          `json:"total_requests"`
	FailedRequests      int64          `json:"failed_requests"`
	AverageLatency      time.Duration  `json:"average_latency"`
	LastError           string         `json:"last_error,omitempty"`
	LastErrorTime       time.Time      `json:"last_error_time,omitempty"`
	ConsecutiveFailures int            `json:"consecutive_failures"`
}

// ProviderRateLimits defines rate limiting configuration
type ProviderRateLimits struct {
	RequestsPerMinute  int           `json:"requests_per_minute"`
	RequestsPerHour    int           `json:"requests_per_hour"`
	RequestsPerDay     int           `json:"requests_per_day"`
	TokensPerMinute    int           `json:"tokens_per_minute"`
	TokensPerHour      int           `json:"tokens_per_hour"`
	TokensPerDay       int           `json:"tokens_per_day"`
	ConcurrentRequests int           `json:"concurrent_requests"`
	BurstLimit         int           `json:"burst_limit"`
	CooldownPeriod     time.Duration `json:"cooldown_period"`
}

// ProviderCostConfig defines cost management settings
type ProviderCostConfig struct {
	BudgetPerHour        float64 `json:"budget_per_hour"`
	BudgetPerDay         float64 `json:"budget_per_day"`
	BudgetPerMonth       float64 `json:"budget_per_month"`
	AlertThreshold       float64 `json:"alert_threshold"` // Percentage of budget
	StopThreshold        float64 `json:"stop_threshold"`  // Percentage of budget
	CostMultiplier       float64 `json:"cost_multiplier"` // For markup/discount
	FreeCreditsRemaining float64 `json:"free_credits_remaining"`
	BillingCycle         string  `json:"billing_cycle"` // monthly, daily, etc.
}

// ProviderLoadBalancing defines load balancing configuration
type ProviderLoadBalancing struct {
	Enabled         bool                   `json:"enabled"`
	Strategy        LoadBalancingStrategy  `json:"strategy"`
	Weights         map[string]int         `json:"weights"`          // Provider weights for weighted round-robin
	HealthThreshold float64                `json:"health_threshold"` // Minimum health score to receive traffic
	StickySession   bool                   `json:"sticky_session"`   // Route same session to same provider
	Failover        ProviderFailoverConfig `json:"failover"`
}

// LoadBalancingStrategy defines different load balancing strategies
type LoadBalancingStrategy string

const (
	StrategyRoundRobin         LoadBalancingStrategy = "round_robin"
	StrategyWeightedRoundRobin LoadBalancingStrategy = "weighted_round_robin"
	StrategyLeastConnections   LoadBalancingStrategy = "least_connections"
	StrategyLeastLatency       LoadBalancingStrategy = "least_latency"
	StrategyLowestCost         LoadBalancingStrategy = "lowest_cost"
	StrategyRandom             LoadBalancingStrategy = "random"
)

// ProviderFailoverConfig defines failover behavior
type ProviderFailoverConfig struct {
	Enabled                 bool          `json:"enabled"`
	MaxRetries              int           `json:"max_retries"`
	RetryDelay              time.Duration `json:"retry_delay"`
	BackoffMultiplier       float64       `json:"backoff_multiplier"`
	CircuitBreakerThreshold int           `json:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   time.Duration `json:"circuit_breaker_timeout"`
}

// EnhancedProviderConfig extends basic provider configuration
type EnhancedProviderConfig struct {
	// Basic provider info
	ID       models.ModelProvider `json:"id"`
	Name     string               `json:"name"`
	BaseURL  string               `json:"base_url"`
	APIKey   string               `json:"api_key,omitempty"`
	Enabled  bool                 `json:"enabled"`
	Priority int                  `json:"priority"`

	// Health and monitoring
	Health ProviderHealthMetrics `json:"health"`

	// Rate limiting and quotas
	RateLimits ProviderRateLimits `json:"rate_limits"`

	// Cost management
	CostConfig ProviderCostConfig `json:"cost_config"`

	// Load balancing
	LoadBalancing ProviderLoadBalancing `json:"load_balancing"`

	// Custom headers and authentication
	CustomHeaders map[string]string `json:"custom_headers,omitempty"`
	AuthType      string            `json:"auth_type"`

	// Timeouts and retries
	RequestTimeout time.Duration `json:"request_timeout"`
	ConnectTimeout time.Duration `json:"connect_timeout"`
	MaxRetries     int           `json:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay"`

	// Regional settings
	Region           string   `json:"region,omitempty"`
	AvailableRegions []string `json:"available_regions,omitempty"`

	// Maintenance windows
	MaintenanceWindows []MaintenanceWindow `json:"maintenance_windows,omitempty"`

	// Provider-specific settings
	ProviderSpecific map[string]interface{} `json:"provider_specific,omitempty"`
}

// MaintenanceWindow defines scheduled maintenance periods
type MaintenanceWindow struct {
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
	Description string    `json:"description"`
	Recurring   bool      `json:"recurring"`
	Timezone    string    `json:"timezone"`
}

// ProviderRequestRecord tracks individual provider requests for rate limiting
type ProviderRequestRecord struct {
	ProviderID models.ModelProvider `json:"provider_id"`
	Timestamp  time.Time            `json:"timestamp"`
	Active     bool                 `json:"active"`
	RequestID  string               `json:"request_id"`
}

// ProviderManager manages provider configurations and health monitoring
type ProviderManager struct {
	providers map[models.ModelProvider]*EnhancedProviderConfig
	mu        sync.RWMutex

	// Health checking
	healthCheckInterval time.Duration
	healthCheckTimeout  time.Duration
	stopHealthCheck     chan struct{}

	// Load balancing state
	roundRobinCounters map[models.ModelProvider]int

	// Circuit breaker state
	circuitBreakers map[models.ModelProvider]*CircuitBreaker

	// Request tracking for rate limiting
	requestHistory []ProviderRequestRecord
}

// CircuitBreaker implements circuit breaker pattern for providers
type CircuitBreaker struct {
	failures    int
	lastFailure time.Time
	state       CircuitBreakerState
	threshold   int
	timeout     time.Duration
	mu          sync.RWMutex
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitClosed   CircuitBreakerState = "closed"
	CircuitOpen     CircuitBreakerState = "open"
	CircuitHalfOpen CircuitBreakerState = "half_open"
)

// NewProviderManager creates a new provider manager
func NewProviderManager() *ProviderManager {
	return &ProviderManager{
		providers:           make(map[models.ModelProvider]*EnhancedProviderConfig),
		healthCheckInterval: 30 * time.Second,
		healthCheckTimeout:  10 * time.Second,
		stopHealthCheck:     make(chan struct{}),
		roundRobinCounters:  make(map[models.ModelProvider]int),
		circuitBreakers:     make(map[models.ModelProvider]*CircuitBreaker),
		requestHistory:      make([]ProviderRequestRecord, 0),
	}
}

// GetProviderConfig returns the configuration for a provider
func (pm *ProviderManager) GetProviderConfig(providerID models.ModelProvider) *EnhancedProviderConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if config, exists := pm.providers[providerID]; exists {
		return config
	}

	// Return default configuration
	return pm.getDefaultProviderConfig(providerID)
}

// SetProviderConfig sets the configuration for a provider
func (pm *ProviderManager) SetProviderConfig(providerID models.ModelProvider, config *EnhancedProviderConfig) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.providers[providerID] = config

	// Initialize circuit breaker if not exists
	if _, exists := pm.circuitBreakers[providerID]; !exists {
		pm.circuitBreakers[providerID] = &CircuitBreaker{
			threshold: config.LoadBalancing.Failover.CircuitBreakerThreshold,
			timeout:   config.LoadBalancing.Failover.CircuitBreakerTimeout,
			state:     CircuitClosed,
		}
	}
}

// GetHealthyProviders returns a list of healthy providers
func (pm *ProviderManager) GetHealthyProviders() []models.ModelProvider {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var healthy []models.ModelProvider
	for providerID, config := range pm.providers {
		if config.Enabled && config.Health.Status == StatusHealthy {
			healthy = append(healthy, providerID)
		}
	}

	return healthy
}

// SelectProvider selects the best provider based on load balancing strategy
func (pm *ProviderManager) SelectProvider(providers []models.ModelProvider, strategy LoadBalancingStrategy) (models.ModelProvider, error) {
	if len(providers) == 0 {
		return "", fmt.Errorf("no providers available")
	}

	// Filter out unhealthy providers
	healthyProviders := make([]models.ModelProvider, 0)
	for _, provider := range providers {
		config := pm.GetProviderConfig(provider)
		if config.Enabled && config.Health.Status == StatusHealthy {
			// Check circuit breaker
			if cb := pm.circuitBreakers[provider]; cb != nil && cb.CanRequest() {
				healthyProviders = append(healthyProviders, provider)
			}
		}
	}

	if len(healthyProviders) == 0 {
		return "", fmt.Errorf("no healthy providers available")
	}

	switch strategy {
	case StrategyRoundRobin:
		return pm.selectRoundRobin(healthyProviders), nil
	case StrategyLeastLatency:
		return pm.selectLeastLatency(healthyProviders), nil
	case StrategyLowestCost:
		return pm.selectLowestCost(healthyProviders), nil
	default:
		// Default to round robin
		return pm.selectRoundRobin(healthyProviders), nil
	}
}

// StartHealthChecking starts the health checking goroutine
func (pm *ProviderManager) StartHealthChecking(ctx context.Context) {
	go pm.healthCheckLoop(ctx)
}

// StopHealthChecking stops the health checking goroutine
func (pm *ProviderManager) StopHealthChecking() {
	close(pm.stopHealthCheck)
}

// UpdateProviderMetrics updates metrics for a provider after a request
func (pm *ProviderManager) UpdateProviderMetrics(providerID models.ModelProvider, latency time.Duration, success bool, errorMsg string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	config := pm.providers[providerID]
	if config == nil {
		return
	}

	// Update metrics
	config.Health.TotalRequests++
	config.Health.AverageLatency = time.Duration(
		(int64(config.Health.AverageLatency)*config.Health.TotalRequests + int64(latency)) /
			(config.Health.TotalRequests + 1),
	)

	if success {
		config.Health.SuccessRate = (config.Health.SuccessRate*float64(config.Health.TotalRequests-1) + 1.0) / float64(config.Health.TotalRequests)
		config.Health.ConsecutiveFailures = 0

		// Record success in circuit breaker
		if cb := pm.circuitBreakers[providerID]; cb != nil {
			cb.RecordSuccess()
		}
	} else {
		config.Health.FailedRequests++
		config.Health.ErrorRate = (config.Health.ErrorRate*float64(config.Health.TotalRequests-1) + 1.0) / float64(config.Health.TotalRequests)
		config.Health.ConsecutiveFailures++
		config.Health.LastError = errorMsg
		config.Health.LastErrorTime = time.Now()

		// Record failure in circuit breaker
		if cb := pm.circuitBreakers[providerID]; cb != nil {
			cb.RecordFailure()
		}
	}

	// Update health status based on metrics
	pm.updateHealthStatus(config)
}

// IsWithinRateLimit checks if a request is within rate limits
func (pm *ProviderManager) IsWithinRateLimit(providerID models.ModelProvider) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	config := pm.GetProviderConfig(providerID)
	now := time.Now()

	// Check requests per minute
	if config.RateLimits.RequestsPerMinute > 0 {
		minuteAgo := now.Add(-time.Minute)
		recentRequests := 0

		for _, record := range pm.requestHistory {
			if record.ProviderID == providerID && record.Timestamp.After(minuteAgo) {
				recentRequests++
			}
		}

		if recentRequests >= config.RateLimits.RequestsPerMinute {
			return false
		}
	}

	// Check concurrent requests
	if config.RateLimits.ConcurrentRequests > 0 {
		activeRequests := 0
		for _, record := range pm.requestHistory {
			if record.ProviderID == providerID && record.Active {
				activeRequests++
			}
		}

		if activeRequests >= config.RateLimits.ConcurrentRequests {
			return false
		}
	}

	return true
}

// IsWithinBudget checks if a request is within cost budget
func (pm *ProviderManager) IsWithinBudget(providerID models.ModelProvider, estimatedCost float64) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	config := pm.GetProviderConfig(providerID)
	now := time.Now()

	// Calculate current hourly spending
	hourAgo := now.Add(-time.Hour)
	hourlySpending := 0.0

	for _, record := range pm.requestHistory {
		if record.ProviderID == providerID && record.Timestamp.After(hourAgo) {
			// Calculate proportional cost based on request record
			hourlySpending += estimatedCost * 0.1 // Proportional cost estimate
		}
	}

	// Check against hourly budget
	if config.CostConfig.BudgetPerHour > 0 && (hourlySpending+estimatedCost) > config.CostConfig.BudgetPerHour {
		return false
	}

	// Calculate current daily spending
	dayAgo := now.Add(-24 * time.Hour)
	dailySpending := 0.0

	for _, record := range pm.requestHistory {
		if record.ProviderID == providerID && record.Timestamp.After(dayAgo) {
			dailySpending += estimatedCost * 0.1 // Proportional cost estimate
		}
	}

	// Check against daily budget
	if config.CostConfig.BudgetPerDay > 0 && (dailySpending+estimatedCost) > config.CostConfig.BudgetPerDay {
		return false
	}

	return true
}

// getDefaultProviderConfig returns default configuration for a provider
func (pm *ProviderManager) getDefaultProviderConfig(providerID models.ModelProvider) *EnhancedProviderConfig {
	return &EnhancedProviderConfig{
		ID:       providerID,
		Name:     string(providerID),
		Enabled:  true,
		Priority: 1,
		Health: ProviderHealthMetrics{
			Status:      StatusUnknown,
			SuccessRate: 1.0,
		},
		RateLimits: ProviderRateLimits{
			RequestsPerMinute:  60,
			ConcurrentRequests: 5,
			BurstLimit:         10,
		},
		CostConfig: ProviderCostConfig{
			BudgetPerHour:  100.0,
			BudgetPerDay:   1000.0,
			AlertThreshold: 0.8,
			StopThreshold:  0.95,
			CostMultiplier: 1.0,
		},
		LoadBalancing: ProviderLoadBalancing{
			Enabled:         true,
			Strategy:        StrategyRoundRobin,
			HealthThreshold: 0.8,
			Failover: ProviderFailoverConfig{
				Enabled:                 true,
				MaxRetries:              3,
				RetryDelay:              1 * time.Second,
				BackoffMultiplier:       2.0,
				CircuitBreakerThreshold: 5,
				CircuitBreakerTimeout:   30 * time.Second,
			},
		},
		RequestTimeout: 30 * time.Second,
		ConnectTimeout: 10 * time.Second,
		MaxRetries:     3,
		RetryDelay:     1 * time.Second,
	}
}

// selectRoundRobin selects provider using round-robin strategy
func (pm *ProviderManager) selectRoundRobin(providers []models.ModelProvider) models.ModelProvider {
	if len(providers) == 0 {
		return ""
	}

	// Simple round-robin implementation
	provider := providers[0]
	counter := pm.roundRobinCounters[provider]
	selected := providers[counter%len(providers)]
	pm.roundRobinCounters[provider] = counter + 1

	return selected
}

// selectLeastLatency selects provider with lowest latency
func (pm *ProviderManager) selectLeastLatency(providers []models.ModelProvider) models.ModelProvider {
	if len(providers) == 0 {
		return ""
	}

	bestProvider := providers[0]
	bestLatency := pm.GetProviderConfig(bestProvider).Health.AverageLatency

	for _, provider := range providers[1:] {
		config := pm.GetProviderConfig(provider)
		if config.Health.AverageLatency < bestLatency {
			bestProvider = provider
			bestLatency = config.Health.AverageLatency
		}
	}

	return bestProvider
}

// selectLowestCost selects provider with lowest cost
func (pm *ProviderManager) selectLowestCost(providers []models.ModelProvider) models.ModelProvider {
	if len(providers) == 0 {
		return ""
	}

	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var bestProvider models.ModelProvider
	lowestCost := float64(999999) // High initial value

	for _, providerID := range providers {
		config := pm.GetProviderConfig(providerID)

		// Calculate effective cost based on cost multiplier and current load
		effectiveCost := config.CostConfig.CostMultiplier

		// Adjust for current load (higher load = higher effective cost)
		activeRequests := 0
		for _, record := range pm.requestHistory {
			if record.ProviderID == providerID && record.Active {
				activeRequests++
			}
		}

		// Increase cost by 10% for each active request to encourage load balancing
		loadMultiplier := 1.0 + (float64(activeRequests) * 0.1)
		effectiveCost *= loadMultiplier

		// Adjust for health status
		if config.Health.Status == StatusDegraded {
			effectiveCost *= 1.5 // 50% penalty for degraded providers
		} else if config.Health.Status == StatusUnhealthy {
			continue // Skip unhealthy providers
		}

		if effectiveCost < lowestCost {
			lowestCost = effectiveCost
			bestProvider = providerID
		}
	}

	return bestProvider
}

// healthCheckLoop runs periodic health checks
func (pm *ProviderManager) healthCheckLoop(ctx context.Context) {
	ticker := time.NewTicker(pm.healthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopHealthCheck:
			return
		case <-ticker.C:
			pm.performHealthChecks()
		}
	}
}

// performHealthChecks performs health checks on all providers
func (pm *ProviderManager) performHealthChecks() {
	pm.mu.RLock()
	providers := make([]*EnhancedProviderConfig, 0, len(pm.providers))
	for _, config := range pm.providers {
		if config.Enabled {
			providers = append(providers, config)
		}
	}
	pm.mu.RUnlock()

	for _, config := range providers {
		go pm.checkProviderHealth(config)
	}
}

// checkProviderHealth performs a health check on a single provider
func (pm *ProviderManager) checkProviderHealth(config *EnhancedProviderConfig) {
	start := time.Now()

	// Simple HTTP health check (this would be provider-specific in reality)
	client := &http.Client{Timeout: pm.healthCheckTimeout}
	resp, err := client.Get(config.BaseURL)

	latency := time.Since(start)
	success := err == nil && resp != nil && resp.StatusCode < 400

	if resp != nil {
		resp.Body.Close()
	}

	pm.mu.Lock()
	config.Health.LastHealthCheck = time.Now()
	config.Health.ResponseTime = latency

	if success {
		config.Health.Status = StatusHealthy
	} else {
		if config.Health.ConsecutiveFailures > 3 {
			config.Health.Status = StatusUnhealthy
		} else {
			config.Health.Status = StatusDegraded
		}
	}
	pm.mu.Unlock()
}

// updateHealthStatus updates the health status based on current metrics
func (pm *ProviderManager) updateHealthStatus(config *EnhancedProviderConfig) {
	if config.Health.ConsecutiveFailures > 5 {
		config.Health.Status = StatusUnhealthy
	} else if config.Health.ErrorRate > 0.1 {
		config.Health.Status = StatusDegraded
	} else {
		config.Health.Status = StatusHealthy
	}
}

// Circuit Breaker methods

// CanRequest checks if the circuit breaker allows a request
func (cb *CircuitBreaker) CanRequest() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful request
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0
	cb.state = CircuitClosed
}

// RecordFailure records a failed request
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.threshold {
		cb.state = CircuitOpen
	}
}
