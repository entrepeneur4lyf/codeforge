package ml

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/graph"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

// Service provides ML-powered code intelligence as a service
type Service struct {
	intelligence *CodeIntelligence
	manager      *graph.CodebaseManager

	// Configuration
	config      *config.Config
	enabled     bool
	initialized bool

	// State management
	mutex        sync.RWMutex
	lastScan     time.Time
	scanInterval time.Duration
}

var (
	// Global service instance
	globalService *Service
	serviceMutex  sync.RWMutex
)

// Initialize sets up the ML service with the given configuration
func Initialize(cfg *config.Config) error {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()

	if globalService != nil {
		return nil // Already initialized
	}

	service := &Service{
		config:       cfg,
		enabled:      true,            // Enable by default
		scanInterval: 5 * time.Minute, // Rescan every 5 minutes
	}

	// Use existing vectordb database for ML operations
	vdb := vectordb.Get()
	if vdb == nil {
		log.Printf("⚠️  ML Service: Vector database not available, ML features disabled")
		service.enabled = false
		globalService = service
		return nil // Don't fail the entire application
	}

	// Initialize codebase manager
	if err := service.initializeCodebase(vdb); err != nil {
		log.Printf("⚠️  ML Service: Codebase initialization failed, ML features disabled: %v", err)
		service.enabled = false
		globalService = service
		return nil // Don't fail the entire application
	}

	globalService = service

	// ML Service initialized silently

	return nil
}

// GetService returns the global ML service instance
func GetService() *Service {
	serviceMutex.RLock()
	defer serviceMutex.RUnlock()
	return globalService
}

// IsEnabled returns whether ML features are enabled
func (s *Service) IsEnabled() bool {
	if s == nil {
		return false
	}
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.enabled && s.initialized
}

// GetIntelligentContext provides ML-enhanced context for user queries
func (s *Service) GetIntelligentContext(ctx context.Context, query string, maxNodes int) string {
	if !s.IsEnabled() {
		return "" // Graceful degradation
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check if we need to rescan
	if time.Since(s.lastScan) > s.scanInterval {
		go s.backgroundRescan() // Non-blocking rescan
	}

	// Get intelligent context
	context, err := s.intelligence.GetIntelligentContext(ctx, query, maxNodes)
	if err != nil {
		log.Printf("ML Service: Failed to get intelligent context: %v", err)
		return "" // Graceful degradation
	}

	return context
}

// SmartSearch performs ML-enhanced code search
func (s *Service) SmartSearch(ctx context.Context, query string) string {
	if !s.IsEnabled() {
		return "" // Graceful degradation
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Perform smart search
	result, err := s.intelligence.SmartSearch(ctx, query, "")
	if err != nil {
		log.Printf("ML Service: Smart search failed: %v", err)
		return "" // Graceful degradation
	}

	// Format result for CLI display
	return s.formatSearchResult(result)
}

// LearnFromInteraction learns from user interactions
func (s *Service) LearnFromInteraction(ctx context.Context, query string, selectedFiles []string, userSatisfaction float64) {
	if !s.IsEnabled() {
		return // Graceful degradation
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Learn from feedback asynchronously to not block the UI
	go func() {
		err := s.intelligence.LearnFromFeedback(ctx, query, selectedFiles, userSatisfaction)
		if err != nil {
			log.Printf("ML Service: Learning from feedback failed: %v", err)
		}
	}()
}

// GetStats returns ML performance statistics
func (s *Service) GetStats(ctx context.Context) string {
	if !s.IsEnabled() {
		return "ML Service: Disabled"
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	stats, err := s.intelligence.GetMLStats(ctx)
	if err != nil {
		return fmt.Sprintf("ML Service: Error getting stats: %v", err)
	}

	return s.formatStats(stats)
}

// GetTDStats returns TD learning specific statistics
func (s *Service) GetTDStats() *TDStats {
	if !s.IsEnabled() {
		return nil
	}

	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.intelligence.GetTDStats()
}

// SetEnabled enables or disables ML features
func (s *Service) SetEnabled(enabled bool) {
	if s == nil {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.enabled = enabled

	if s.intelligence != nil {
		s.intelligence.SetEnabled(enabled)
	}

	// ML Service state changed silently
}

// Shutdown gracefully shuts down the ML service
func (s *Service) Shutdown() {
	if s == nil {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.intelligence != nil {
		s.intelligence.SetEnabled(false)
	}

	if s.manager != nil {
		s.manager.Stop()
	}

	// No database cleanup needed for in-memory ML system

	// ML Service shutdown complete
}

// Private methods

// Database initialization removed - using in-memory ML system

func (s *Service) initializeCodebase(vdb *vectordb.VectorDB) error {
	// Get working directory
	workingDir := s.config.WorkingDir
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}
	}

	// Initialize codebase manager
	s.manager = graph.NewCodebaseManager()

	// Try to initialize with current directory
	if err := s.manager.Initialize(workingDir); err != nil {
		log.Printf("ML Service: Failed to initialize with %s, trying auto-detection", workingDir)

		// Try auto-detection
		if err := s.manager.AutoDetectAndInitialize(); err != nil {
			return fmt.Errorf("failed to initialize codebase: %w", err)
		}
	}

	// Create code graph from manager (we'll need to add a getter method)
	codeGraph := graph.NewCodeGraph(s.manager.GetRootPath())
	scanner := graph.NewSimpleScanner(codeGraph)
	if err := scanner.ScanRepository(s.manager.GetRootPath()); err != nil {
		return fmt.Errorf("failed to scan repository: %w", err)
	}

	// Create ML intelligence using existing vectordb
	intelligence, err := NewCodeIntelligence(codeGraph, vdb)
	if err != nil {
		return fmt.Errorf("failed to create ML intelligence: %w", err)
	}

	s.intelligence = intelligence
	s.initialized = true
	s.lastScan = time.Now()

	return nil
}

func (s *Service) backgroundRescan() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.enabled || !s.initialized {
		return
	}

	log.Println("🔄 ML Service: Background rescan started")

	// Rescan the codebase
	codeGraph := graph.NewCodeGraph(s.manager.GetRootPath())
	scanner := graph.NewSimpleScanner(codeGraph)
	if err := scanner.ScanRepository(s.manager.GetRootPath()); err != nil {
		log.Printf("ML Service: Background rescan failed: %v", err)
		return
	}

	// Update intelligence with new graph using vectordb
	vdb := vectordb.Get()
	if vdb == nil {
		log.Printf("ML Service: Vector database not available for rescan")
		return
	}

	newIntelligence, err := NewCodeIntelligence(codeGraph, vdb)
	if err != nil {
		log.Printf("ML Service: Failed to update intelligence: %v", err)
		return
	}

	s.intelligence = newIntelligence
	s.lastScan = time.Now()

	log.Println("✅ ML Service: Background rescan completed")
}

func (s *Service) formatSearchResult(result *SearchResult) string {
	if result == nil {
		return ""
	}

	var output strings.Builder
	output.WriteString("## 🧠 ML-Enhanced Search Results\n\n")
	output.WriteString(fmt.Sprintf("**Confidence:** %.2f  \n", result.Confidence))
	output.WriteString(fmt.Sprintf("**Relevance:** %.2f  \n", result.Relevance))

	if len(result.BestPath) > 0 {
		output.WriteString("**Relevant Code Path:**\n")
		for i, nodeID := range result.BestPath {
			if i >= 5 { // Limit to top 5
				break
			}
			output.WriteString(fmt.Sprintf("%d. Node: %s\n", i+1, nodeID))
		}
	}

	if result.Explanation != "" {
		output.WriteString(fmt.Sprintf("\n**Analysis:** %s\n", result.Explanation))
	}

	return output.String()
}

func (s *Service) formatStats(stats *MLMetrics) string {
	var output strings.Builder
	output.WriteString("## 🧠 ML Performance Statistics\n\n")
	output.WriteString(fmt.Sprintf("**Learning Entries:** %d\n", stats.QLearningTotalEntries))
	output.WriteString(fmt.Sprintf("**Average Performance:** %.3f\n", stats.AverageReward))
	output.WriteString(fmt.Sprintf("**Total Experiences:** %d\n", stats.TotalExperiences))
	output.WriteString(fmt.Sprintf("**Last Updated:** %s\n", stats.LastUpdated.Format("15:04:05")))

	return output.String()
}

// Shutdown gracefully shuts down the global ML service
func Shutdown() {
	serviceMutex.Lock()
	defer serviceMutex.Unlock()

	if globalService != nil {
		globalService.Shutdown()
		globalService = nil
	}
}
