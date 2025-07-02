package models

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// ModelDiscoveryService handles automatic model discovery and updates
type ModelDiscoveryService struct {
	registry    *ModelRegistry
	manager     *ModelManager
	updateQueue chan DiscoveryTask
	workers     int
	running     bool
	stopCh      chan struct{}
	wg          sync.WaitGroup
	mu          sync.RWMutex
}

// DiscoveryTask represents a model discovery task
type DiscoveryTask struct {
	Type        DiscoveryTaskType `json:"type"`
	ProviderID  ProviderID        `json:"provider_id"`
	ModelID     CanonicalModelID  `json:"model_id,omitempty"`
	Priority    int               `json:"priority"`    // Higher = more urgent
	RetryCount  int               `json:"retry_count"` // Number of retries attempted
	ScheduledAt time.Time         `json:"scheduled_at"`
	Context     context.Context   `json:"-"`
}

// DiscoveryTaskType defines the type of discovery task
type DiscoveryTaskType string

const (
	TaskTypeDiscoverModels     DiscoveryTaskType = "discover_models"
	TaskTypeCheckAvailability  DiscoveryTaskType = "check_availability"
	TaskTypeUpdatePricing      DiscoveryTaskType = "update_pricing"
	TaskTypeUpdateCapabilities DiscoveryTaskType = "update_capabilities"
	TaskTypeHealthCheck        DiscoveryTaskType = "health_check"
)

// DiscoveryConfig configures the discovery service
type DiscoveryConfig struct {
	Workers              int           `json:"workers"`
	DiscoveryInterval    time.Duration `json:"discovery_interval"`    // How often to discover new models
	AvailabilityInterval time.Duration `json:"availability_interval"` // How often to check availability
	PricingInterval      time.Duration `json:"pricing_interval"`      // How often to update pricing
	MaxRetries           int           `json:"max_retries"`
	RetryDelay           time.Duration `json:"retry_delay"`
	QueueSize            int           `json:"queue_size"`
}

// DefaultDiscoveryConfig returns default configuration
func DefaultDiscoveryConfig() DiscoveryConfig {
	return DiscoveryConfig{
		Workers:              3,
		DiscoveryInterval:    24 * time.Hour, // Daily discovery
		AvailabilityInterval: 6 * time.Hour,  // Check availability every 6 hours
		PricingInterval:      12 * time.Hour, // Update pricing twice daily
		MaxRetries:           3,
		RetryDelay:           5 * time.Minute,
		QueueSize:            1000,
	}
}

// NewModelDiscoveryService creates a new model discovery service
func NewModelDiscoveryService(registry *ModelRegistry, manager *ModelManager, config DiscoveryConfig) *ModelDiscoveryService {
	return &ModelDiscoveryService{
		registry:    registry,
		manager:     manager,
		updateQueue: make(chan DiscoveryTask, config.QueueSize),
		workers:     config.Workers,
		stopCh:      make(chan struct{}),
	}
}

// Start starts the discovery service
func (mds *ModelDiscoveryService) Start(ctx context.Context) error {
	mds.mu.Lock()
	defer mds.mu.Unlock()

	if mds.running {
		return fmt.Errorf("discovery service is already running")
	}

	mds.running = true

	// Start worker goroutines
	for i := 0; i < mds.workers; i++ {
		mds.wg.Add(1)
		go mds.worker(ctx, i)
	}

	// Start scheduler
	mds.wg.Add(1)
	go mds.scheduler(ctx)

	log.Printf("Model discovery service started with %d workers", mds.workers)
	return nil
}

// Stop stops the discovery service
func (mds *ModelDiscoveryService) Stop() error {
	mds.mu.Lock()
	defer mds.mu.Unlock()

	if !mds.running {
		return fmt.Errorf("discovery service is not running")
	}

	close(mds.stopCh)
	mds.wg.Wait()
	mds.running = false

	log.Println("Model discovery service stopped")
	return nil
}

// ScheduleTask schedules a discovery task
func (mds *ModelDiscoveryService) ScheduleTask(task DiscoveryTask) error {
	mds.mu.RLock()
	defer mds.mu.RUnlock()

	if !mds.running {
		return fmt.Errorf("discovery service is not running")
	}

	select {
	case mds.updateQueue <- task:
		return nil
	default:
		return fmt.Errorf("discovery queue is full")
	}
}

// DiscoverModelsForProvider schedules model discovery for a specific provider
func (mds *ModelDiscoveryService) DiscoverModelsForProvider(providerID ProviderID) error {
	task := DiscoveryTask{
		Type:        TaskTypeDiscoverModels,
		ProviderID:  providerID,
		Priority:    5,
		ScheduledAt: time.Now(),
		Context:     context.Background(),
	}

	return mds.ScheduleTask(task)
}

// CheckModelAvailability schedules availability check for a specific model
func (mds *ModelDiscoveryService) CheckModelAvailability(modelID CanonicalModelID, providerID ProviderID) error {
	task := DiscoveryTask{
		Type:        TaskTypeCheckAvailability,
		ProviderID:  providerID,
		ModelID:     modelID,
		Priority:    3,
		ScheduledAt: time.Now(),
		Context:     context.Background(),
	}

	return mds.ScheduleTask(task)
}

// UpdateModelPricing schedules pricing update for a specific model
func (mds *ModelDiscoveryService) UpdateModelPricing(modelID CanonicalModelID, providerID ProviderID) error {
	task := DiscoveryTask{
		Type:        TaskTypeUpdatePricing,
		ProviderID:  providerID,
		ModelID:     modelID,
		Priority:    2,
		ScheduledAt: time.Now(),
		Context:     context.Background(),
	}

	return mds.ScheduleTask(task)
}

// GetQueueStatus returns the current queue status
func (mds *ModelDiscoveryService) GetQueueStatus() map[string]any {
	mds.mu.RLock()
	defer mds.mu.RUnlock()

	return map[string]any{
		"running":    mds.running,
		"workers":    mds.workers,
		"queue_size": len(mds.updateQueue),
		"queue_cap":  cap(mds.updateQueue),
	}
}

// worker processes discovery tasks
func (mds *ModelDiscoveryService) worker(ctx context.Context, workerID int) {
	defer mds.wg.Done()

	log.Printf("Discovery worker %d started", workerID)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Discovery worker %d stopped (context cancelled)", workerID)
			return
		case <-mds.stopCh:
			log.Printf("Discovery worker %d stopped", workerID)
			return
		case task := <-mds.updateQueue:
			mds.processTask(ctx, task, workerID)
		}
	}
}

// scheduler schedules periodic discovery tasks
func (mds *ModelDiscoveryService) scheduler(ctx context.Context) {
	defer mds.wg.Done()

	config := DefaultDiscoveryConfig()

	discoveryTicker := time.NewTicker(config.DiscoveryInterval)
	availabilityTicker := time.NewTicker(config.AvailabilityInterval)
	pricingTicker := time.NewTicker(config.PricingInterval)

	defer discoveryTicker.Stop()
	defer availabilityTicker.Stop()
	defer pricingTicker.Stop()

	log.Println("Discovery scheduler started")

	for {
		select {
		case <-ctx.Done():
			log.Println("Discovery scheduler stopped (context cancelled)")
			return
		case <-mds.stopCh:
			log.Println("Discovery scheduler stopped")
			return
		case <-discoveryTicker.C:
			mds.schedulePeriodicDiscovery()
		case <-availabilityTicker.C:
			mds.scheduleAvailabilityChecks()
		case <-pricingTicker.C:
			mds.schedulePricingUpdates()
		}
	}
}

// processTask processes a single discovery task
func (mds *ModelDiscoveryService) processTask(ctx context.Context, task DiscoveryTask, workerID int) {
	log.Printf("Worker %d processing task: %s for provider %s", workerID, task.Type, task.ProviderID)

	switch task.Type {
	case TaskTypeDiscoverModels:
		mds.processDiscoverModels(ctx, task)
	case TaskTypeCheckAvailability:
		mds.processCheckAvailability(ctx, task)
	case TaskTypeUpdatePricing:
		mds.processUpdatePricing(ctx, task)
	case TaskTypeUpdateCapabilities:
		mds.processUpdateCapabilities(ctx, task)
	case TaskTypeHealthCheck:
		mds.processHealthCheck(ctx, task)
	default:
		log.Printf("Unknown task type: %s", task.Type)
	}
}

// schedulePeriodicDiscovery schedules discovery for all providers
func (mds *ModelDiscoveryService) schedulePeriodicDiscovery() {
	providers := []ProviderID{
		ProviderAnthropicCanonical,
		ProviderOpenAICanonical,
		ProviderGeminiCanonical,
		ProviderOpenRouterCanonical,
	}

	for _, provider := range providers {
		task := DiscoveryTask{
			Type:        TaskTypeDiscoverModels,
			ProviderID:  provider,
			Priority:    1,
			ScheduledAt: time.Now(),
			Context:     context.Background(),
		}

		if err := mds.ScheduleTask(task); err != nil {
			log.Printf("Failed to schedule discovery for provider %s: %v", provider, err)
		}
	}
}

// scheduleAvailabilityChecks schedules availability checks for all models
func (mds *ModelDiscoveryService) scheduleAvailabilityChecks() {
	models := mds.registry.ListModels()

	for _, model := range models {
		for providerID := range model.Providers {
			task := DiscoveryTask{
				Type:        TaskTypeCheckAvailability,
				ProviderID:  providerID,
				ModelID:     model.ID,
				Priority:    2,
				ScheduledAt: time.Now(),
				Context:     context.Background(),
			}

			if err := mds.ScheduleTask(task); err != nil {
				log.Printf("Failed to schedule availability check for model %s on provider %s: %v",
					model.ID, providerID, err)
			}
		}
	}
}

// schedulePricingUpdates schedules pricing updates for all models
func (mds *ModelDiscoveryService) schedulePricingUpdates() {
	models := mds.registry.ListModels()

	for _, model := range models {
		for providerID := range model.Providers {
			task := DiscoveryTask{
				Type:        TaskTypeUpdatePricing,
				ProviderID:  providerID,
				ModelID:     model.ID,
				Priority:    1,
				ScheduledAt: time.Now(),
				Context:     context.Background(),
			}

			if err := mds.ScheduleTask(task); err != nil {
				log.Printf("Failed to schedule pricing update for model %s on provider %s: %v",
					model.ID, providerID, err)
			}
		}
	}
}

// Task processing methods

func (mds *ModelDiscoveryService) processDiscoverModels(_ context.Context, task DiscoveryTask) {
	// In a real implementation, this would call provider APIs to discover new models
	// For now, we'll simulate discovery by logging
	log.Printf("Discovering models for provider %s", task.ProviderID)

	// Simulate discovery delay
	time.Sleep(100 * time.Millisecond)

	// In practice, this would:
	// 1. Call provider API to list available models
	// 2. Compare with existing models in registry
	// 3. Add new models or update existing ones
	// 4. Update availability status
}

func (mds *ModelDiscoveryService) processCheckAvailability(_ context.Context, task DiscoveryTask) {
	log.Printf("Checking availability for model %s on provider %s", task.ModelID, task.ProviderID)

	// Simulate availability check
	time.Sleep(50 * time.Millisecond)

	// In practice, this would:
	// 1. Make a test API call to the provider
	// 2. Update the model's availability status
	// 3. Record response time and error information

	// For now, assume models are available
	if err := mds.registry.UpdateModelAvailability(task.ModelID, task.ProviderID, true); err != nil {
		log.Printf("Failed to update availability for model %s: %v", task.ModelID, err)
	}
}

func (mds *ModelDiscoveryService) processUpdatePricing(_ context.Context, task DiscoveryTask) {
	log.Printf("Updating pricing for model %s on provider %s", task.ModelID, task.ProviderID)

	// Simulate pricing update
	time.Sleep(75 * time.Millisecond)

	// In practice, this would:
	// 1. Fetch latest pricing from provider API or documentation
	// 2. Update the model's pricing information
	// 3. Log pricing changes for audit purposes
}

func (mds *ModelDiscoveryService) processUpdateCapabilities(_ context.Context, task DiscoveryTask) {
	log.Printf("Updating capabilities for model %s on provider %s", task.ModelID, task.ProviderID)

	// Simulate capability update
	time.Sleep(60 * time.Millisecond)

	// In practice, this would:
	// 1. Check provider documentation for capability updates
	// 2. Test model capabilities if possible
	// 3. Update the model's capability flags
}

func (mds *ModelDiscoveryService) processHealthCheck(_ context.Context, task DiscoveryTask) {
	log.Printf("Performing health check for provider %s", task.ProviderID)

	// Simulate health check
	time.Sleep(30 * time.Millisecond)

	// In practice, this would:
	// 1. Check provider API status
	// 2. Verify authentication
	// 3. Test basic functionality
	// 4. Update provider health status
}
