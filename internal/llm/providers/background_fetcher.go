package providers

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"
)

// BackgroundModelFetcher handles async model fetching and caching for all providers
type BackgroundModelFetcher struct {
	mu       sync.RWMutex
	started  bool
	stopChan chan struct{}
}

var (
	globalFetcher *BackgroundModelFetcher
	fetcherOnce   sync.Once
)

// GetBackgroundFetcher returns the singleton background fetcher
func GetBackgroundFetcher() *BackgroundModelFetcher {
	fetcherOnce.Do(func() {
		globalFetcher = &BackgroundModelFetcher{
			stopChan: make(chan struct{}),
		}
	})
	return globalFetcher
}

// Start begins background model fetching for all providers
func (bf *BackgroundModelFetcher) Start() {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	if bf.started {
		return
	}

	bf.started = true

	// Start background fetching
	go bf.fetchAllModels()
}

// Stop stops the background fetcher
func (bf *BackgroundModelFetcher) Stop() {
	bf.mu.Lock()
	defer bf.mu.Unlock()

	if !bf.started {
		return
	}

	close(bf.stopChan)
	bf.started = false
}

// fetchAllModels fetches models for all providers in the background
func (bf *BackgroundModelFetcher) fetchAllModels() {
	// Initial fetch
	bf.refreshAllProviders()

	// Set up periodic refresh (every 6 hours)
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-bf.stopChan:
			return
		case <-ticker.C:
			bf.refreshAllProviders()
		}
	}
}

// refreshAllProviders refreshes models for all available providers
func (bf *BackgroundModelFetcher) refreshAllProviders() {
	var wg sync.WaitGroup

	// OpenAI
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RefreshOpenAIModelsAsync(apiKey)
		}()
	}

	// Anthropic
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RefreshAnthropicModelsAsync(apiKey)
		}()
	}

	// OpenRouter
	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RefreshOpenRouterModelsAsync(apiKey)
		}()
	}

	// Gemini
	if apiKey := os.Getenv("GEMINI_API_KEY"); apiKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			RefreshGeminiModelsAsync(apiKey)
		}()
	}

	// Wait for all providers to complete (with timeout)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All providers completed
	case <-time.After(2 * time.Minute):
		// Timeout - some providers took too long
		fmt.Printf("Background model refresh timed out after 2 minutes\n")
	}
}

// RefreshOpenRouterModelsAsync refreshes OpenRouter models in the background
func RefreshOpenRouterModelsAsync(apiKey string) {
	if apiKey == "" {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Use existing OpenRouter fetching logic
		_, err := GetOpenRouterModelsByProvider(ctx, apiKey)
		if err != nil {
			// Silent failure for background refresh
			fmt.Printf("Background OpenRouter model refresh failed: %v\n", err)
		}
	}()
}

// InitializeBackgroundFetching starts the background model fetching service
func InitializeBackgroundFetching() {
	fetcher := GetBackgroundFetcher()
	fetcher.Start()
}

// StopBackgroundFetching stops the background model fetching service
func StopBackgroundFetching() {
	if globalFetcher != nil {
		globalFetcher.Stop()
	}
}
