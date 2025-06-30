package llm

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// RetryableError represents an error that can be retried
type RetryableError struct {
	Err        error
	StatusCode int
	Headers    map[string]string
	Retryable  bool
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryableError checks if an error is retryable
func IsRetryableError(err error) bool {
	if retryErr, ok := err.(*RetryableError); ok {
		return retryErr.Retryable
	}
	return false
}

// GetStatusCode extracts status code from error
func GetStatusCode(err error) int {
	if retryErr, ok := err.(*RetryableError); ok {
		return retryErr.StatusCode
	}
	return 0
}

// GetHeaders extracts headers from error
func GetHeaders(err error) map[string]string {
	if retryErr, ok := err.(*RetryableError); ok {
		return retryErr.Headers
	}
	return nil
}

// WithRetry wraps a function with retry logic
// Based on Cline's withRetry decorator pattern
func WithRetry(options RetryOptions) func(func(context.Context) (ApiStream, error)) func(context.Context) (ApiStream, error) {
	return func(fn func(context.Context) (ApiStream, error)) func(context.Context) (ApiStream, error) {
		return func(ctx context.Context) (ApiStream, error) {
			var lastErr error

			for attempt := 0; attempt < options.MaxRetries; attempt++ {
				stream, err := fn(ctx)
				if err == nil {
					return stream, nil
				}

				lastErr = err
				isRateLimit := IsRateLimitError(err)
				isLastAttempt := attempt == options.MaxRetries-1

				// Don't retry if it's not a rate limit error and we don't retry all errors
				if !isRateLimit && !options.RetryAllErrors {
					return nil, err
				}

				// Don't retry on last attempt
				if isLastAttempt {
					return nil, err
				}

				// Calculate delay
				delay := calculateDelay(err, attempt, options)

				// Wait before retry
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
					// Continue to next attempt
				}
			}

			return nil, lastErr
		}
	}
}

// WithRetryCompletion wraps a completion function with retry logic
func WithRetryCompletion(options RetryOptions) func(func(context.Context, string) (string, error)) func(context.Context, string) (string, error) {
	return func(fn func(context.Context, string) (string, error)) func(context.Context, string) (string, error) {
		return func(ctx context.Context, prompt string) (string, error) {
			var lastErr error

			for attempt := 0; attempt < options.MaxRetries; attempt++ {
				result, err := fn(ctx, prompt)
				if err == nil {
					return result, nil
				}

				lastErr = err
				isRateLimit := IsRateLimitError(err)
				isLastAttempt := attempt == options.MaxRetries-1

				// Don't retry if it's not a rate limit error and we don't retry all errors
				if !isRateLimit && !options.RetryAllErrors {
					return "", err
				}

				// Don't retry on last attempt
				if isLastAttempt {
					return "", err
				}

				// Calculate delay
				delay := calculateDelay(err, attempt, options)

				// Wait before retry
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-time.After(delay):
					// Continue to next attempt
				}
			}

			return "", lastErr
		}
	}
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	statusCode := GetStatusCode(err)
	if statusCode == 429 {
		return true
	}

	// Check error message for rate limit indicators
	errMsg := strings.ToLower(err.Error())
	rateLimitPatterns := []string{
		"rate limit",
		"too many requests",
		"quota exceeded",
		"rate exceeded",
		"throttled",
	}

	for _, pattern := range rateLimitPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// calculateDelay calculates the delay before retry
// Based on Cline's retry logic with exponential backoff and rate limit headers
func calculateDelay(err error, attempt int, options RetryOptions) time.Duration {
	headers := GetHeaders(err)

	// Check for retry-after header
	if headers != nil {
		if retryAfter, exists := headers["retry-after"]; exists {
			if delay := parseRetryAfter(retryAfter); delay > 0 {
				if delay > options.MaxDelay {
					return options.MaxDelay
				}
				return delay
			}
		}

		// Check for x-ratelimit-reset header
		if resetTime, exists := headers["x-ratelimit-reset"]; exists {
			if delay := parseRateLimitReset(resetTime); delay > 0 {
				if delay > options.MaxDelay {
					return options.MaxDelay
				}
				return delay
			}
		}

		// Check for ratelimit-reset header
		if resetTime, exists := headers["ratelimit-reset"]; exists {
			if delay := parseRateLimitReset(resetTime); delay > 0 {
				if delay > options.MaxDelay {
					return options.MaxDelay
				}
				return delay
			}
		}
	}

	// Use exponential backoff if no header
	delay := time.Duration(float64(options.BaseDelay) * math.Pow(2, float64(attempt)))
	if delay > options.MaxDelay {
		delay = options.MaxDelay
	}

	return delay
}

// parseRetryAfter parses the Retry-After header
func parseRetryAfter(retryAfter string) time.Duration {
	// Try parsing as seconds
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP date
	if t, err := http.ParseTime(retryAfter); err == nil {
		delay := time.Until(t)
		if delay > 0 {
			return delay
		}
	}

	return 0
}

// parseRateLimitReset parses rate limit reset headers
func parseRateLimitReset(resetTime string) time.Duration {
	// Try parsing as Unix timestamp
	if timestamp, err := strconv.ParseInt(resetTime, 10, 64); err == nil {
		// Check if it's a Unix timestamp (seconds since epoch)
		if timestamp > time.Now().Unix() {
			resetAt := time.Unix(timestamp, 0)
			delay := time.Until(resetAt)
			if delay > 0 {
				return delay
			}
		} else {
			// Might be delta seconds
			return time.Duration(timestamp) * time.Second
		}
	}

	return 0
}

// NewRetryableError creates a new retryable error
func NewRetryableError(err error, statusCode int, headers map[string]string) *RetryableError {
	retryable := statusCode == 429 || // Rate limit
		statusCode >= 500 || // Server errors
		statusCode == 408 || // Request timeout
		statusCode == 409 // Conflict (sometimes retryable)

	return &RetryableError{
		Err:        err,
		StatusCode: statusCode,
		Headers:    headers,
		Retryable:  retryable,
	}
}

// WrapHTTPError wraps an HTTP error with retry information
func WrapHTTPError(err error, resp *http.Response) error {
	if resp == nil {
		return err
	}

	headers := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers[strings.ToLower(key)] = values[0]
		}
	}

	return NewRetryableError(err, resp.StatusCode, headers)
}

// RetryHandler provides a convenient way to add retry logic to handlers
type RetryHandler struct {
	options RetryOptions
}

// NewRetryHandler creates a new retry handler
func NewRetryHandler(options RetryOptions) *RetryHandler {
	return &RetryHandler{options: options}
}

// WrapHandler wraps an API handler with retry logic
func (rh *RetryHandler) WrapHandler(handler ApiHandler) ApiHandler {
	return &retryableHandler{
		handler: handler,
		options: rh.options,
	}
}

// retryableHandler implements ApiHandler with retry logic
type retryableHandler struct {
	handler ApiHandler
	options RetryOptions
}

func (rh *retryableHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []Message) (ApiStream, error) {
	retryFn := WithRetry(rh.options)
	wrappedFn := retryFn(func(ctx context.Context) (ApiStream, error) {
		return rh.handler.CreateMessage(ctx, systemPrompt, messages)
	})
	return wrappedFn(ctx)
}

func (rh *retryableHandler) GetModel() ModelResponse {
	return rh.handler.GetModel()
}

func (rh *retryableHandler) GetApiStreamUsage() (*ApiStreamUsageChunk, error) {
	return rh.handler.GetApiStreamUsage()
}
