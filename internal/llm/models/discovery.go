package models

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ModelDiscoveryService handles automatic model discovery and updates
type ModelDiscoveryService struct {
	registry    *ModelRegistry
	manager     *ModelManager
	httpClient  *http.Client
	updateQueue chan DiscoveryTask
	workers     int
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// DiscoveryTask represents a model discovery task
type DiscoveryTask struct {
	Provider    ProviderID
	TaskType    DiscoveryTaskType
	Priority    int
	RetryCount  int
	ScheduledAt time.Time
}

// DiscoveryTaskType defines types of discovery tasks
type DiscoveryTaskType string

const (
	TaskDiscoverModels    DiscoveryTaskType = "discover_models"
	TaskUpdatePricing     DiscoveryTaskType = "update_pricing"
	TaskCheckAvailability DiscoveryTaskType = "check_availability"
	TaskUpdateLimits      DiscoveryTaskType = "update_limits"
)

// ProviderModelInfo represents model information from a provider
type ProviderModelInfo struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description,omitempty"`
	MaxTokens    int                    `json:"max_tokens,omitempty"`
	InputPrice   float64                `json:"input_price,omitempty"`
	OutputPrice  float64                `json:"output_price,omitempty"`
	Capabilities map[string]interface{} `json:"capabilities,omitempty"`
	Available    bool                   `json:"available"`
	Deprecated   bool                   `json:"deprecated,omitempty"`
}

// DiscoveryConfig holds configuration for model discovery
type DiscoveryConfig struct {
	EnableAutoDiscovery bool                     `json:"enable_auto_discovery"`
	UpdateInterval      time.Duration            `json:"update_interval"`
	MaxRetries          int                      `json:"max_retries"`
	Timeout             time.Duration            `json:"timeout"`
	EnabledProviders    []ProviderID             `json:"enabled_providers"`
	RateLimits          map[ProviderID]RateLimit `json:"rate_limits"`
}

// RateLimit defines rate limiting for provider API calls
type RateLimit struct {
	RequestsPerMinute int           `json:"requests_per_minute"`
	BurstSize         int           `json:"burst_size"`
	Cooldown          time.Duration `json:"cooldown"`
}

// NewModelDiscoveryService creates a new model discovery service
func NewModelDiscoveryService(registry *ModelRegistry, manager *ModelManager) *ModelDiscoveryService {
	return &ModelDiscoveryService{
		registry: registry,
		manager:  manager,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		updateQueue: make(chan DiscoveryTask, 1000),
		workers:     3,
		stopCh:      make(chan struct{}),
	}
}

// Start begins the discovery service
func (ds *ModelDiscoveryService) Start(ctx context.Context) error {
	// Start worker goroutines
	for i := 0; i < ds.workers; i++ {
		ds.wg.Add(1)
		go ds.worker(ctx)
	}

	// Start periodic discovery scheduler
	ds.wg.Add(1)
	go ds.scheduler(ctx)

	return nil
}

// Stop stops the discovery service
func (ds *ModelDiscoveryService) Stop() error {
	close(ds.stopCh)
	ds.wg.Wait()
	return nil
}

// ScheduleDiscovery schedules a discovery task
func (ds *ModelDiscoveryService) ScheduleDiscovery(provider ProviderID, taskType DiscoveryTaskType, priority int) {
	task := DiscoveryTask{
		Provider:    provider,
		TaskType:    taskType,
		Priority:    priority,
		ScheduledAt: time.Now(),
	}

	select {
	case ds.updateQueue <- task:
	default:
		// Queue is full, skip this task
	}
}

// DiscoverModelsForProvider discovers models for a specific provider
func (ds *ModelDiscoveryService) DiscoverModelsForProvider(ctx context.Context, provider ProviderID) error {
	switch provider {
	case ProviderOpenAI:
		return ds.discoverOpenAIModels(ctx)
	case ProviderAnthropic:
		return ds.discoverAnthropicModels(ctx)
	case ProviderOpenRouter:
		return ds.discoverOpenRouterModels(ctx)
	case ProviderGemini:
		return ds.discoverGeminiModels(ctx)
	default:
		return fmt.Errorf("discovery not implemented for provider: %s", provider)
	}
}

// worker processes discovery tasks from the queue
func (ds *ModelDiscoveryService) worker(ctx context.Context) {
	defer ds.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ds.stopCh:
			return
		case task := <-ds.updateQueue:
			ds.processTask(ctx, task)
		}
	}
}

// scheduler periodically schedules discovery tasks
func (ds *ModelDiscoveryService) scheduler(ctx context.Context) {
	defer ds.wg.Done()

	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ds.stopCh:
			return
		case <-ticker.C:
			ds.schedulePeriodicTasks()
		}
	}
}

// processTask processes a single discovery task
func (ds *ModelDiscoveryService) processTask(ctx context.Context, task DiscoveryTask) {
	switch task.TaskType {
	case TaskDiscoverModels:
		err := ds.DiscoverModelsForProvider(ctx, task.Provider)
		if err != nil && task.RetryCount < 3 {
			// Retry with exponential backoff
			task.RetryCount++
			task.ScheduledAt = time.Now().Add(time.Duration(task.RetryCount*task.RetryCount) * time.Minute)

			select {
			case ds.updateQueue <- task:
			default:
			}
		}
	case TaskCheckAvailability:
		ds.checkProviderAvailability(ctx, task.Provider)
	case TaskUpdatePricing:
		ds.updateProviderPricing(ctx, task.Provider)
	}
}

// schedulePeriodicTasks schedules regular discovery tasks
func (ds *ModelDiscoveryService) schedulePeriodicTasks() {
	providers := []ProviderID{
		ProviderOpenAI,
		ProviderAnthropic,
		ProviderOpenRouter,
		ProviderGemini,
	}

	for _, provider := range providers {
		// Schedule model discovery (low priority, once per day)
		ds.ScheduleDiscovery(provider, TaskDiscoverModels, 1)

		// Schedule availability check (high priority, every hour)
		ds.ScheduleDiscovery(provider, TaskCheckAvailability, 5)
	}
}

// discoverOpenAIModels discovers OpenAI models
func (ds *ModelDiscoveryService) discoverOpenAIModels(ctx context.Context) error {
	// This would make an API call to OpenAI's models endpoint
	// For now, we'll simulate with known models

	knownModels := []ProviderModelInfo{
		{
			ID:          "gpt-4o",
			Name:        "GPT-4o",
			MaxTokens:   128000,
			InputPrice:  5.0,
			OutputPrice: 15.0,
			Available:   true,
		},
		{
			ID:          "gpt-4o-mini",
			Name:        "GPT-4o Mini",
			MaxTokens:   128000,
			InputPrice:  0.15,
			OutputPrice: 0.6,
			Available:   true,
		},
		{
			ID:          "o1-preview",
			Name:        "o1 Preview",
			MaxTokens:   128000,
			InputPrice:  15.0,
			OutputPrice: 60.0,
			Available:   true,
		},
	}

	return ds.updateModelsFromProvider(ProviderOpenAI, knownModels)
}

// discoverAnthropicModels discovers Anthropic models
func (ds *ModelDiscoveryService) discoverAnthropicModels(ctx context.Context) error {
	knownModels := []ProviderModelInfo{
		{
			ID:          "claude-opus-4-20250514",
			Name:        "Claude Opus 4",
			MaxTokens:   200000,
			InputPrice:  15.0,
			OutputPrice: 75.0,
			Available:   true,
		},
		{
			ID:          "claude-sonnet-4-20250514",
			Name:        "Claude Sonnet 4",
			MaxTokens:   200000,
			InputPrice:  3.0,
			OutputPrice: 15.0,
			Available:   true,
		},
		{
			ID:          "claude-3-5-sonnet-20241022",
			Name:        "Claude 3.5 Sonnet",
			MaxTokens:   200000,
			InputPrice:  3.0,
			OutputPrice: 15.0,
			Available:   true,
		},
		{
			ID:          "claude-3-5-haiku-20241022",
			Name:        "Claude 3.5 Haiku",
			MaxTokens:   200000,
			InputPrice:  0.8,
			OutputPrice: 4.0,
			Available:   true,
		},
	}

	return ds.updateModelsFromProvider(ProviderAnthropic, knownModels)
}

// discoverOpenRouterModels discovers OpenRouter models
func (ds *ModelDiscoveryService) discoverOpenRouterModels(ctx context.Context) error {
	// OpenRouter has a public API for model discovery
	url := "https://openrouter.ai/api/v1/models"

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := ds.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenRouter API returned status %d", resp.StatusCode)
	}

	var response struct {
		Data []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Pricing struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
			ContextLength int `json:"context_length"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err
	}

	var models []ProviderModelInfo
	for _, model := range response.Data {
		// Parse pricing (OpenRouter returns strings like "0.000005")
		inputPrice := parsePrice(model.Pricing.Prompt) * 1000000 // Convert to per million tokens
		outputPrice := parsePrice(model.Pricing.Completion) * 1000000

		models = append(models, ProviderModelInfo{
			ID:          model.ID,
			Name:        model.Name,
			MaxTokens:   model.ContextLength,
			InputPrice:  inputPrice,
			OutputPrice: outputPrice,
			Available:   true,
		})
	}

	return ds.updateModelsFromProvider(ProviderOpenRouter, models)
}

// discoverGeminiModels discovers Google Gemini models
func (ds *ModelDiscoveryService) discoverGeminiModels(ctx context.Context) error {
	knownModels := []ProviderModelInfo{
		{
			ID:          "gemini-2.0-flash-exp",
			Name:        "Gemini 2.0 Flash",
			MaxTokens:   1000000,
			InputPrice:  0.075,
			OutputPrice: 0.3,
			Available:   true,
		},
		{
			ID:          "gemini-1.5-pro",
			Name:        "Gemini 1.5 Pro",
			MaxTokens:   2000000,
			InputPrice:  1.25,
			OutputPrice: 5.0,
			Available:   true,
		},
	}

	return ds.updateModelsFromProvider(ProviderGemini, knownModels)
}

// updateModelsFromProvider updates the registry with discovered models
func (ds *ModelDiscoveryService) updateModelsFromProvider(provider ProviderID, models []ProviderModelInfo) error {
	for _, model := range models {
		// Try to map to existing canonical model or create new one
		canonicalModel := ds.mapToCanonicalModel(provider, model)
		if canonicalModel != nil {
			// Update the registry (this would need to be implemented in the registry)
			// For now, we'll just log the discovery
			fmt.Printf("Discovered model: %s from %s\n", model.Name, provider)
		}
	}
	return nil
}

// mapToCanonicalModel maps a provider model to a canonical model
func (ds *ModelDiscoveryService) mapToCanonicalModel(provider ProviderID, model ProviderModelInfo) *CanonicalModel {
	// This would contain logic to map provider-specific models to canonical models
	// For now, return nil as this requires more complex mapping logic
	return nil
}

// checkProviderAvailability checks if a provider is available
func (ds *ModelDiscoveryService) checkProviderAvailability(ctx context.Context, provider ProviderID) {
	// Implementation would ping provider endpoints to check availability
	// Update the cache with availability status
}

// updateProviderPricing updates pricing information for a provider
func (ds *ModelDiscoveryService) updateProviderPricing(ctx context.Context, provider ProviderID) {
	// Implementation would fetch latest pricing from provider APIs
}

// parsePrice parses a price string to float64 with comprehensive error handling
func parsePrice(priceStr string) float64 {
	if priceStr == "" {
		return 0.0
	}

	// Clean the string - remove common currency symbols and whitespace
	cleaned := strings.TrimSpace(priceStr)
	cleaned = strings.ReplaceAll(cleaned, "$", "")
	cleaned = strings.ReplaceAll(cleaned, "¢", "")
	cleaned = strings.ReplaceAll(cleaned, "€", "")
	cleaned = strings.ReplaceAll(cleaned, "£", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")

	// Handle scientific notation and decimal formats
	price, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		// Try alternative parsing for edge cases
		if strings.Contains(cleaned, "e") || strings.Contains(cleaned, "E") {
			// Already handled by ParseFloat
			return 0.0
		}

		// Handle fractional formats like "1/1000"
		if strings.Contains(cleaned, "/") {
			parts := strings.Split(cleaned, "/")
			if len(parts) == 2 {
				numerator, err1 := strconv.ParseFloat(parts[0], 64)
				denominator, err2 := strconv.ParseFloat(parts[1], 64)
				if err1 == nil && err2 == nil && denominator != 0 {
					return numerator / denominator
				}
			}
		}

		// Handle percentage formats
		if strings.HasSuffix(cleaned, "%") {
			percentStr := strings.TrimSuffix(cleaned, "%")
			if percent, err := strconv.ParseFloat(percentStr, 64); err == nil {
				return percent / 100.0
			}
		}

		// Log warning for unparseable price but don't fail
		log.Printf("Warning: Could not parse price string '%s': %v", priceStr, err)
		return 0.0
	}

	// Validate reasonable price range (0 to $1000 per million tokens)
	if price < 0 {
		log.Printf("Warning: Negative price detected: %f, setting to 0", price)
		return 0.0
	}
	if price > 1000.0 {
		log.Printf("Warning: Unusually high price detected: %f", price)
	}

	return price
}
