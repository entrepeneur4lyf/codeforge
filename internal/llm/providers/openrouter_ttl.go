package providers

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

// TTLService manages automatic TTL enforcement for OpenRouter models
type TTLService struct {
	handler   *OpenRouterHandler
	apiKey    string
	stopChan  chan struct{}
	isRunning bool
}

// NewTTLService creates a new TTL enforcement service
func NewTTLService(apiKey string) *TTLService {
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}

	options := llm.ApiHandlerOptions{
		OpenRouterAPIKey: apiKey,
	}

	db := vectordb.GetInstance()
	handler := NewOpenRouterHandlerWithDB(options, db)

	return &TTLService{
		handler:   handler,
		apiKey:    apiKey,
		stopChan:  make(chan struct{}),
		isRunning: false,
	}
}

// Start begins the TTL enforcement background service
func (s *TTLService) Start(ctx context.Context) {
	if s.isRunning {
		fmt.Println("⚠️ TTL service already running")
		return
	}

	if s.apiKey == "" {
		fmt.Println("⚠️ No OpenRouter API key available, TTL service disabled")
		return
	}

	s.isRunning = true
	fmt.Println("🚀 Starting OpenRouter TTL enforcement service")

	go s.runTTLEnforcement(ctx)
}

// Stop stops the TTL enforcement service
func (s *TTLService) Stop() {
	if !s.isRunning {
		return
	}

	fmt.Println("🛑 Stopping OpenRouter TTL enforcement service")
	close(s.stopChan)
	s.isRunning = false
}

// runTTLEnforcement runs the background TTL enforcement loop
func (s *TTLService) runTTLEnforcement(ctx context.Context) {
	// Check every 6 hours
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	// Run initial check after 1 minute
	initialTimer := time.NewTimer(1 * time.Minute)
	defer initialTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("🔄 TTL service stopped due to context cancellation")
			return

		case <-s.stopChan:
			fmt.Println("🔄 TTL service stopped")
			return

		case <-initialTimer.C:
			s.performTTLCheck(ctx)

		case <-ticker.C:
			s.performTTLCheck(ctx)
		}
	}
}

// performTTLCheck checks and enforces TTL policies
func (s *TTLService) performTTLCheck(ctx context.Context) {
	fmt.Println("🕒 Performing TTL check...")

	// Check if models need refresh
	if !s.handler.isDatabaseCacheValid(ctx) {
		fmt.Println("🔄 Model cache expired, refreshing...")

		// Cleanup expired data
		if err := s.handler.cleanupExpiredModels(ctx); err != nil {
			fmt.Printf("⚠️ Model cleanup error: %v\n", err)
		}

		if err := s.handler.cleanupStaleMetadata(ctx); err != nil {
			fmt.Printf("⚠️ Metadata cleanup error: %v\n", err)
		}

		// Refresh models
		if _, err := s.handler.GetOpenRouterModels(ctx); err != nil {
			fmt.Printf("❌ Failed to refresh models: %v\n", err)
		} else {
			fmt.Println("✅ Models refreshed successfully")
		}
	} else {
		fmt.Println("✅ Model cache is still valid")

		// Still run cleanup for very old data
		if err := s.handler.cleanupExpiredModels(ctx); err != nil {
			fmt.Printf("⚠️ Cleanup warning: %v\n", err)
		}
	}
}

// ForceRefresh manually triggers a model refresh regardless of TTL
func (s *TTLService) ForceRefresh(ctx context.Context) error {
	fmt.Println("🔄 Force refreshing OpenRouter models...")

	// Cleanup first
	if err := s.handler.cleanupExpiredModels(ctx); err != nil {
		fmt.Printf("⚠️ Cleanup warning: %v\n", err)
	}

	if err := s.handler.cleanupStaleMetadata(ctx); err != nil {
		fmt.Printf("⚠️ Metadata cleanup warning: %v\n", err)
	}

	// Force refresh
	_, err := s.handler.refreshModelsInDatabase(ctx)
	if err != nil {
		return fmt.Errorf("failed to force refresh models: %w", err)
	}

	fmt.Println("✅ Force refresh completed successfully")
	return nil
}

// GetCacheStatus returns current cache status information
func (s *TTLService) GetCacheStatus(ctx context.Context) (map[string]interface{}, error) {
	status := make(map[string]interface{})

	// Check cache validity
	isValid := s.handler.isDatabaseCacheValid(ctx)
	status["cache_valid"] = isValid
	status["ttl_hours"] = 24

	// Get model count
	if s.handler.db != nil {
		var modelCount int
		query := "SELECT COUNT(*) FROM openrouter_models"
		row := s.handler.db.QueryRowContext(ctx, query)
		if err := row.Scan(&modelCount); err == nil {
			status["model_count"] = modelCount
		}

		// Get metadata count
		var metadataCount int
		query = "SELECT COUNT(*) FROM openrouter_model_metadata"
		row = s.handler.db.QueryRowContext(ctx, query)
		if err := row.Scan(&metadataCount); err == nil {
			status["metadata_count"] = metadataCount
		}

		// Get last update time
		var lastUpdate time.Time
		query = "SELECT MAX(last_seen) FROM openrouter_models"
		row = s.handler.db.QueryRowContext(ctx, query)
		if err := row.Scan(&lastUpdate); err == nil {
			status["last_update"] = lastUpdate.Format("2006-01-02 15:04:05")
			status["hours_since_update"] = time.Since(lastUpdate).Hours()
		}
	}

	status["service_running"] = s.isRunning
	return status, nil
}

// Global TTL service instance
var globalTTLService *TTLService

// StartGlobalTTLService starts the global TTL enforcement service
func StartGlobalTTLService(ctx context.Context) {
	if globalTTLService == nil {
		globalTTLService = NewTTLService("")
	}
	globalTTLService.Start(ctx)
}

// StopGlobalTTLService stops the global TTL enforcement service
func StopGlobalTTLService() {
	if globalTTLService != nil {
		globalTTLService.Stop()
	}
}

// GetGlobalTTLService returns the global TTL service instance
func GetGlobalTTLService() *TTLService {
	return globalTTLService
}
