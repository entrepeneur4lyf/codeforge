package context

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
)

// ContextManager orchestrates all context management features
type ContextManager struct {
	config          *config.Config
	tokenCounter    *TokenCounter
	summarizer      *Summarizer
	slidingWindow   *SlidingWindow
	cache           *ContextCache
	compressor      *ContextCompressor
	relevanceScorer *RelevanceScorer
}

// NewContextManager creates a new context manager
func NewContextManager(cfg *config.Config) *ContextManager {
	contextConfig := cfg.GetContextConfig()

	// Initialize cache
	var cache *ContextCache
	if contextConfig.CacheEnabled {
		cache = NewContextCache(contextConfig.MaxCacheSize, int64(contextConfig.CacheTTL))
	}

	return &ContextManager{
		config:          cfg,
		tokenCounter:    NewTokenCounter(),
		summarizer:      NewSummarizer(cfg),
		slidingWindow:   NewSlidingWindow(cfg),
		cache:           cache,
		compressor:      NewContextCompressor(contextConfig.CompressionLevel),
		relevanceScorer: NewRelevanceScorer(),
	}
}

// ProcessConversation processes a conversation for optimal context management
func (cm *ContextManager) ProcessConversation(ctx context.Context, messages []ConversationMessage, modelID string) (*ProcessedContext, error) {
	return cm.ProcessConversationWithOptions(ctx, messages, modelID, ProcessingOptions{})
}

// ProcessingOptions defines options for context processing
type ProcessingOptions struct {
	FullContext        bool `json:"full_context"`        // Use full context mode
	DisableSummary     bool `json:"disable_summary"`     // Disable auto-summarization
	DisableWindow      bool `json:"disable_window"`      // Disable sliding window
	DisableCompression bool `json:"disable_compression"` // Disable compression
	ForceRefresh       bool `json:"force_refresh"`       // Force cache refresh
}

// ProcessConversationWithOptions processes a conversation with specific options
func (cm *ContextManager) ProcessConversationWithOptions(ctx context.Context, messages []ConversationMessage, modelID string, options ProcessingOptions) (*ProcessedContext, error) {
	log.Printf("Processing conversation with %d messages for model %s (full_context: %v)", len(messages), modelID, options.FullContext)

	// Check cache first (unless force refresh)
	cacheKey := cm.generateCacheKeyWithOptions(messages, modelID, options)
	if cm.cache != nil && !options.ForceRefresh {
		if cached, found := cm.cache.Get(cacheKey); found {
			if processedCtx, ok := cached.(*ProcessedContext); ok {
				log.Printf("Context cache hit for key: %s", cacheKey)
				return processedCtx, nil
			}
		}
	}

	// Start with original messages
	processedMessages := messages
	var summaryResult *SummaryResult
	var windowResult *WindowResult
	var compressionResult *CompressionResult

	// Full context mode: skip all optimizations
	if options.FullContext {
		log.Printf("Using full context mode - skipping all optimizations")
	} else {
		// Step 1: Check if summarization is needed
		if !options.DisableSummary && cm.summarizer.ShouldSummarize(processedMessages, modelID) {
			log.Printf("Conversation needs summarization")

			summary, err := cm.summarizer.SummarizeConversation(ctx, processedMessages, modelID)
			if err != nil {
				log.Printf("Summarization failed: %v", err)
			} else {
				summaryResult = summary
				// Replace messages with summary + recent messages
				processedMessages = cm.applySummary(processedMessages, summary, modelID)
			}
		}

		// Step 2: Apply sliding window if needed
		contextConfig := cm.config.GetContextConfig()
		if !options.DisableWindow && contextConfig.SlidingWindow {
			window, err := cm.slidingWindow.ApplyWindow(processedMessages, modelID)
			if err != nil {
				log.Printf("Sliding window failed: %v", err)
			} else {
				windowResult = window
				processedMessages = window.Messages
			}
		}

		// Step 3: Apply compression
		if !options.DisableCompression && contextConfig.CompressionLevel > 0 {
			compressed, err := cm.compressor.CompressMessages(processedMessages, modelID)
			if err != nil {
				log.Printf("Compression failed: %v", err)
			} else {
				compressionResult = compressed
				processedMessages = compressed.Messages
			}
		}
	}

	// Calculate final metrics
	originalUsage := cm.tokenCounter.CountConversationTokens(messages, modelID)
	finalUsage := cm.tokenCounter.CountConversationTokens(processedMessages, modelID)

	result := &ProcessedContext{
		Messages:          processedMessages,
		OriginalCount:     len(messages),
		FinalCount:        len(processedMessages),
		OriginalTokens:    originalUsage.TotalTokens,
		FinalTokens:       finalUsage.TotalTokens,
		CompressionRatio:  float64(finalUsage.TotalTokens) / float64(originalUsage.TotalTokens),
		ModelID:           modelID,
		SummaryResult:     summaryResult,
		WindowResult:      windowResult,
		CompressionResult: compressionResult,
		CacheKey:          cacheKey,
		ProcessingSteps:   cm.getProcessingSteps(summaryResult, windowResult, compressionResult),
	}

	// Cache the result
	if cm.cache != nil {
		cm.cache.Set(cacheKey, result, finalUsage.TotalTokens)
		log.Printf("Cached processed context with key: %s", cacheKey)
	}

	log.Printf("Context processing complete: %d -> %d messages, %d -> %d tokens (%.2f%% compression)",
		result.OriginalCount, result.FinalCount,
		result.OriginalTokens, result.FinalTokens,
		(1.0-result.CompressionRatio)*100)

	return result, nil
}

// ProcessedContext represents the result of context processing
type ProcessedContext struct {
	Messages          []ConversationMessage `json:"messages"`
	OriginalCount     int                   `json:"original_count"`
	FinalCount        int                   `json:"final_count"`
	OriginalTokens    int                   `json:"original_tokens"`
	FinalTokens       int                   `json:"final_tokens"`
	CompressionRatio  float64               `json:"compression_ratio"`
	ModelID           string                `json:"model_id"`
	SummaryResult     *SummaryResult        `json:"summary_result,omitempty"`
	WindowResult      *WindowResult         `json:"window_result,omitempty"`
	CompressionResult *CompressionResult    `json:"compression_result,omitempty"`
	CacheKey          string                `json:"cache_key"`
	ProcessingSteps   []string              `json:"processing_steps"`
}

// applySummary applies summarization result to messages
func (cm *ContextManager) applySummary(messages []ConversationMessage, summary *SummaryResult, modelID string) []ConversationMessage {
	// Find the last summary index
	lastSummaryIndex := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if metadata, ok := messages[i].Metadata["summary"]; ok {
			if isSummary, ok := metadata.(bool); ok && isSummary {
				lastSummaryIndex = i
				break
			}
		}
	}

	// Keep messages before last summary and create new summary
	var result []ConversationMessage

	// Keep system messages and previous summaries
	if lastSummaryIndex >= 0 {
		result = append(result, messages[:lastSummaryIndex+1]...)
	}

	// Add new summary message
	summaryMsg := cm.summarizer.CreateSummaryMessage(summary.Summary, "")
	result = append(result, summaryMsg)

	// Keep recent messages that fit in context window
	modelConfig := cm.config.GetModelConfig(modelID)
	currentTokens := cm.tokenCounter.CountConversationTokens(result, modelID).TotalTokens
	availableTokens := modelConfig.ContextWindow - currentTokens

	// Add recent messages that fit
	recentStart := lastSummaryIndex + 1
	if recentStart < 0 {
		recentStart = 0
	}

	for i := len(messages) - 1; i >= recentStart; i-- {
		msg := messages[i]
		msgTokens := cm.tokenCounter.CountMessageTokens(msg, modelID).TotalTokens

		if currentTokens+msgTokens <= availableTokens {
			result = append(result, msg)
			currentTokens += msgTokens
		} else {
			break
		}
	}

	return result
}

// generateCacheKeyWithOptions creates a cache key including processing options
func (cm *ContextManager) generateCacheKeyWithOptions(messages []ConversationMessage, modelID string, options ProcessingOptions) string {
	// Create a hash based on message content, model, and options
	var content strings.Builder
	content.WriteString(modelID)
	content.WriteString(":")

	// Add options to cache key
	content.WriteString(fmt.Sprintf("full:%v|sum:%v|win:%v|comp:%v|",
		options.FullContext, options.DisableSummary, options.DisableWindow, options.DisableCompression))

	for _, msg := range messages {
		content.WriteString(fmt.Sprintf("%s:%d:%s|", msg.Role, msg.Timestamp, msg.Content[:min(100, len(msg.Content))]))
	}

	return hashString(content.String())
}

// getProcessingSteps returns a list of processing steps applied
func (cm *ContextManager) getProcessingSteps(summary *SummaryResult, window *WindowResult, compression *CompressionResult) []string {
	var steps []string

	if summary != nil {
		steps = append(steps, "summarization")
	}
	if window != nil && window.WindowsApplied > 0 {
		steps = append(steps, "sliding_window")
	}
	if compression != nil && compression.CompressionRatio < 1.0 {
		steps = append(steps, "compression")
	}

	if len(steps) == 0 {
		steps = append(steps, "no_processing")
	}

	return steps
}

// GetContextStats returns comprehensive context statistics
func (cm *ContextManager) GetContextStats(messages []ConversationMessage, modelID string) map[string]interface{} {
	usage := cm.tokenCounter.CountConversationTokens(messages, modelID)
	modelConfig := cm.config.GetModelConfig(modelID)
	contextConfig := cm.config.GetContextConfig()

	stats := map[string]interface{}{
		"message_count":        len(messages),
		"total_tokens":         usage.TotalTokens,
		"input_tokens":         usage.InputTokens,
		"output_tokens":        usage.OutputTokens,
		"context_window":       modelConfig.ContextWindow,
		"utilization":          float64(usage.TotalTokens) / float64(modelConfig.ContextWindow),
		"needs_summarization":  cm.summarizer.ShouldSummarize(messages, modelID),
		"needs_sliding_window": usage.TotalTokens > modelConfig.ContextWindow,
		"auto_summarize":       contextConfig.AutoSummarize,
		"sliding_window":       contextConfig.SlidingWindow,
		"cache_enabled":        contextConfig.CacheEnabled,
		"compression_level":    contextConfig.CompressionLevel,
	}

	// Add cache stats if available
	if cm.cache != nil {
		cacheStats := cm.cache.GetStats()
		stats["cache_stats"] = map[string]interface{}{
			"size":     cacheStats.Size,
			"max_size": cacheStats.MaxSize,
			"hit_rate": cacheStats.HitRate,
			"hits":     cacheStats.Hits,
			"misses":   cacheStats.Misses,
		}
	}

	return stats
}

// ProcessFullContext processes conversation in full-context mode (no optimizations)
func (cm *ContextManager) ProcessFullContext(ctx context.Context, messages []ConversationMessage, modelID string) (*ProcessedContext, error) {
	return cm.ProcessConversationWithOptions(ctx, messages, modelID, ProcessingOptions{
		FullContext: true,
	})
}

// ProcessForSinglePass processes conversation optimized for single-pass operations
func (cm *ContextManager) ProcessForSinglePass(ctx context.Context, messages []ConversationMessage, modelID string) (*ProcessedContext, error) {
	return cm.ProcessConversationWithOptions(ctx, messages, modelID, ProcessingOptions{
		FullContext:        true,
		DisableSummary:     true,
		DisableWindow:      true,
		DisableCompression: false, // Keep compression for token efficiency
	})
}

// ProcessWithCustomOptions processes conversation with custom optimization settings
func (cm *ContextManager) ProcessWithCustomOptions(ctx context.Context, messages []ConversationMessage, modelID string, enableSummary, enableWindow, enableCompression bool) (*ProcessedContext, error) {
	return cm.ProcessConversationWithOptions(ctx, messages, modelID, ProcessingOptions{
		FullContext:        false,
		DisableSummary:     !enableSummary,
		DisableWindow:      !enableWindow,
		DisableCompression: !enableCompression,
	})
}

// ProcessWithRelevanceFiltering processes conversation with relevance-based filtering
func (cm *ContextManager) ProcessWithRelevanceFiltering(ctx context.Context, messages []ConversationMessage, modelID string, query string, threshold float64) (*ProcessedContext, error) {
	// First apply relevance filtering
	filteredMessages, relevanceResult, err := cm.relevanceScorer.FilterByRelevance(messages, query, threshold, 100)
	if err != nil {
		log.Printf("Relevance filtering failed: %v", err)
		// Fall back to normal processing
		return cm.ProcessConversation(ctx, messages, modelID)
	}

	log.Printf("Relevance filtering: %d -> %d messages (threshold: %.2f, avg score: %.2f)",
		relevanceResult.OriginalCount, relevanceResult.FilteredCount, threshold, relevanceResult.AverageScore)

	// Process the filtered messages
	processedCtx, err := cm.ProcessConversation(ctx, filteredMessages, modelID)
	if err != nil {
		return nil, err
	}

	// Add relevance information to the result
	if processedCtx.SummaryResult == nil {
		processedCtx.SummaryResult = &SummaryResult{}
	}
	processedCtx.SummaryResult.Metadata["relevance_filtering"] = map[string]interface{}{
		"query":          query,
		"threshold":      threshold,
		"original_count": relevanceResult.OriginalCount,
		"filtered_count": relevanceResult.FilteredCount,
		"average_score":  relevanceResult.AverageScore,
	}

	return processedCtx, nil
}

// ScoreMessageRelevance scores messages for relevance to a query
func (cm *ContextManager) ScoreMessageRelevance(messages []ConversationMessage, query string) (*RelevanceResult, error) {
	contextConfig := cm.config.GetContextConfig()
	threshold := contextConfig.RelevanceThreshold
	if threshold <= 0 {
		threshold = 0.3 // Default threshold
	}

	return cm.relevanceScorer.ScoreRelevance(messages, query, threshold)
}

// FilterMessagesByRelevance filters messages based on relevance to a query
func (cm *ContextManager) FilterMessagesByRelevance(messages []ConversationMessage, query string, maxMessages int) ([]ConversationMessage, *RelevanceResult, error) {
	contextConfig := cm.config.GetContextConfig()
	threshold := contextConfig.RelevanceThreshold
	if threshold <= 0 {
		threshold = 0.3 // Default threshold
	}

	return cm.relevanceScorer.FilterByRelevance(messages, query, threshold, maxMessages)
}

// ClearCache clears the context cache
func (cm *ContextManager) ClearCache() {
	if cm.cache != nil {
		cm.cache.Clear()
		log.Printf("Context cache cleared")
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
