package providers

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/llm"
)

// RetryConfig defines retry behavior for API calls
type RetryConfig struct {
	MaxRetries      int           `json:"maxRetries"`
	BaseDelay       time.Duration `json:"baseDelay"`
	MaxDelay        time.Duration `json:"maxDelay"`
	BackoffFactor   float64       `json:"backoffFactor"`
	JitterFactor    float64       `json:"jitterFactor"`
	RetryableErrors []int         `json:"retryableErrors"`
}

// DefaultRetryConfig returns a sensible default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		BaseDelay:     1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		JitterFactor:  0.1,
		RetryableErrors: []int{
			http.StatusTooManyRequests,     // 429 - Rate limited
			http.StatusInternalServerError, // 500 - Server error
			http.StatusBadGateway,          // 502 - Bad gateway
			http.StatusServiceUnavailable,  // 503 - Service unavailable
			http.StatusGatewayTimeout,      // 504 - Gateway timeout
		},
	}
}

// RateLimitInfo contains rate limit information from API responses
type RateLimitInfo struct {
	Limit      int           `json:"limit"`
	Remaining  int           `json:"remaining"`
	Reset      time.Time     `json:"reset"`
	RetryAfter time.Duration `json:"retryAfter"`
}

// ExtractRateLimitInfo extracts rate limit information from HTTP headers
func ExtractRateLimitInfo(headers http.Header) *RateLimitInfo {
	info := &RateLimitInfo{}

	// Standard rate limit headers
	if limit := headers.Get("X-RateLimit-Limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			info.Limit = val
		}
	}

	if remaining := headers.Get("X-RateLimit-Remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			info.Remaining = val
		}
	}

	if reset := headers.Get("X-RateLimit-Reset"); reset != "" {
		if val, err := strconv.ParseInt(reset, 10, 64); err == nil {
			info.Reset = time.Unix(val, 0)
		}
	}

	// Retry-After header (can be seconds or HTTP date)
	if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
		if seconds, err := strconv.Atoi(retryAfter); err == nil {
			info.RetryAfter = time.Duration(seconds) * time.Second
		} else if t, err := time.Parse(time.RFC1123, retryAfter); err == nil {
			info.RetryAfter = time.Until(t)
		}
	}

	return info
}

// IsRetryableError determines if an error should be retried
func IsRetryableError(err error, config RetryConfig) bool {
	if err == nil {
		return false
	}

	// Check for HTTP errors
	if retryErr, ok := err.(*llm.RetryableError); ok {
		for _, code := range config.RetryableErrors {
			if retryErr.StatusCode == code {
				return true
			}
		}
		return false
	}

	// Check for network errors (timeouts, connection issues)
	errStr := strings.ToLower(err.Error())
	networkErrors := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"no such host",
		"network is unreachable",
		"temporary failure",
		"i/o timeout",
		"context deadline exceeded",
	}

	for _, netErr := range networkErrors {
		if strings.Contains(errStr, netErr) {
			return true
		}
	}

	return false
}

// CalculateBackoffDelay calculates the delay for the next retry attempt
func CalculateBackoffDelay(attempt int, config RetryConfig) time.Duration {
	if attempt <= 0 {
		return config.BaseDelay
	}

	// Exponential backoff: baseDelay * (backoffFactor ^ attempt)
	delay := float64(config.BaseDelay) * math.Pow(config.BackoffFactor, float64(attempt))

	// Add jitter to prevent thundering herd
	jitter := delay * config.JitterFactor * (2*rand.Float64() - 1) // Random between -jitterFactor and +jitterFactor
	delay += jitter

	// Cap at max delay
	if delay > float64(config.MaxDelay) {
		delay = float64(config.MaxDelay)
	}

	// Ensure minimum delay
	if delay < float64(config.BaseDelay) {
		delay = float64(config.BaseDelay)
	}

	return time.Duration(delay)
}

// RetryableOperation represents an operation that can be retried
type RetryableOperation[T any] func(ctx context.Context, attempt int) (T, error)

// ExecuteWithRetry executes an operation with retry logic
func ExecuteWithRetry[T any](
	ctx context.Context,
	operation RetryableOperation[T],
	config RetryConfig,
	onRetry func(attempt, maxRetries int, delay time.Duration, err error) error,
) (T, error) {
	var lastErr error
	var result T

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Execute the operation
		result, lastErr = operation(ctx, attempt)

		// Success case
		if lastErr == nil {
			return result, nil
		}

		// Check if we should retry
		if attempt >= config.MaxRetries || !IsRetryableError(lastErr, config) {
			break
		}

		// Calculate delay for next attempt
		delay := CalculateBackoffDelay(attempt, config)

		// Handle rate limiting with Retry-After
		if retryErr, ok := lastErr.(*llm.RetryableError); ok && retryErr.StatusCode == http.StatusTooManyRequests {
			if retryErr.Headers != nil {
				if retryAfter, exists := retryErr.Headers["retry-after"]; exists {
					if rateLimitInfo := ExtractRateLimitInfo(http.Header{"Retry-After": []string{retryAfter}}); rateLimitInfo.RetryAfter > 0 {
						delay = rateLimitInfo.RetryAfter
					}
				}
			}
		}

		// Call retry callback if provided
		if onRetry != nil {
			if err := onRetry(attempt+1, config.MaxRetries, delay, lastErr); err != nil {
				return result, fmt.Errorf("retry callback failed: %w", err)
			}
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return result, fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for API calls
type CircuitBreaker struct {
	maxFailures     int
	resetTimeout    time.Duration
	state           CircuitBreakerState
	failures        int
	lastFailureTime time.Time
	successCount    int
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitBreakerClosed,
	}
}

// Execute executes an operation through the circuit breaker
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func(ctx context.Context) error) error {
	// Check if circuit should be reset
	if cb.state == CircuitBreakerOpen && time.Since(cb.lastFailureTime) > cb.resetTimeout {
		cb.state = CircuitBreakerHalfOpen
		cb.successCount = 0
	}

	// Reject if circuit is open
	if cb.state == CircuitBreakerOpen {
		return fmt.Errorf("circuit breaker is open")
	}

	// Execute operation
	err := operation(ctx)

	if err != nil {
		cb.onFailure()
		return err
	}

	cb.onSuccess()
	return nil
}

// onFailure handles operation failure
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailureTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.state = CircuitBreakerOpen
	}
}

// onSuccess handles operation success
func (cb *CircuitBreaker) onSuccess() {
	switch cb.state {
	case CircuitBreakerHalfOpen:
		cb.successCount++
		// Reset circuit after successful operation in half-open state
		cb.state = CircuitBreakerClosed
		cb.failures = 0
	case CircuitBreakerClosed:
		cb.failures = 0
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return cb.state
}

// ProviderHealthChecker monitors provider health and availability
type ProviderHealthChecker struct {
	circuitBreakers map[string]*CircuitBreaker
	retryConfigs    map[string]RetryConfig
}

// NewProviderHealthChecker creates a new provider health checker
func NewProviderHealthChecker() *ProviderHealthChecker {
	return &ProviderHealthChecker{
		circuitBreakers: make(map[string]*CircuitBreaker),
		retryConfigs:    make(map[string]RetryConfig),
	}
}

// RegisterProvider registers a provider with health checking
func (phc *ProviderHealthChecker) RegisterProvider(providerName string, config RetryConfig) {
	phc.retryConfigs[providerName] = config
	phc.circuitBreakers[providerName] = NewCircuitBreaker(5, 60*time.Second) // 5 failures, 60s reset
}

// ExecuteWithHealthCheck executes an operation with health checking and retry logic
func ExecuteWithHealthCheck[T any](
	ctx context.Context,
	phc *ProviderHealthChecker,
	providerName string,
	operation RetryableOperation[T],
	onRetry func(attempt, maxRetries int, delay time.Duration, err error) error,
) (T, error) {
	var result T

	// Get or create circuit breaker
	cb, exists := phc.circuitBreakers[providerName]
	if !exists {
		cb = NewCircuitBreaker(5, 60*time.Second)
		phc.circuitBreakers[providerName] = cb
	}

	// Get retry config
	config, exists := phc.retryConfigs[providerName]
	if !exists {
		config = DefaultRetryConfig()
	}

	// Execute with circuit breaker
	err := cb.Execute(ctx, func(ctx context.Context) error {
		var opErr error
		result, opErr = ExecuteWithRetry(ctx, operation, config, onRetry)
		return opErr
	})

	return result, err
}

// GetProviderHealth returns the health status of a provider
func (phc *ProviderHealthChecker) GetProviderHealth(providerName string) map[string]interface{} {
	cb, exists := phc.circuitBreakers[providerName]
	if !exists {
		return map[string]interface{}{
			"status": "unknown",
		}
	}

	status := "healthy"
	switch cb.GetState() {
	case CircuitBreakerOpen:
		status = "unhealthy"
	case CircuitBreakerHalfOpen:
		status = "recovering"
	}

	return map[string]interface{}{
		"status":       status,
		"failures":     cb.failures,
		"lastFailure":  cb.lastFailureTime,
		"circuitState": cb.GetState(),
	}
}
