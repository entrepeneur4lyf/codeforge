package context

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/graph"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/prompt"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/tools"
	"github.com/entrepeneur4lyf/codeforge/internal/lsp"
	"github.com/entrepeneur4lyf/codeforge/internal/mcp"
	"github.com/entrepeneur4lyf/codeforge/internal/models"
	"github.com/entrepeneur4lyf/codeforge/internal/project"
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
	projectLoader   *ProjectContextLoader
	baseContext     *ProjectContext
	repoMapCache    string
	repoMapTime     time.Time
	
	// Code intelligence providers
	codebaseManager *graph.CodebaseManager
	symbolExtractor *lsp.SymbolExtractor
	mcpManager      *mcp.MCPManager
	toolRegistry    *tools.ToolRegistry
}

// NewContextManager creates a new context manager
func NewContextManager(cfg *config.Config) *ContextManager {
	contextConfig := cfg.GetContextConfig()

	// Initialize cache
	var cache *ContextCache
	if contextConfig.CacheEnabled {
		cache = NewContextCache(contextConfig.MaxCacheSize, int64(contextConfig.CacheTTL))
	}

	// Get working directory from config
	workingDir := cfg.WorkingDir
	if workingDir == "" {
		workingDir = "."
	}

	cm := &ContextManager{
		config:          cfg,
		tokenCounter:    NewTokenCounter(),
		summarizer:      NewSummarizer(cfg),
		slidingWindow:   NewSlidingWindow(cfg),
		cache:           cache,
		compressor:      NewContextCompressor(contextConfig.CompressionLevel),
		relevanceScorer: NewRelevanceScorer(),
		projectLoader:   NewProjectContextLoader(cfg, workingDir),
	}

	// Load base project context on initialization
	var err error
	cm.baseContext, err = cm.projectLoader.LoadProjectContext()
	if err != nil {
		log.Printf("Warning: Failed to load base project context: %v", err)
	}

	// Initialize code intelligence providers
	cm.initializeCodeIntelligence(workingDir)

	return cm
}

// SetMCPManager sets the MCP manager for tool context integration
func (cm *ContextManager) SetMCPManager(mcpManager *mcp.MCPManager) {
	cm.mcpManager = mcpManager
}

// SetToolRegistry sets the tool registry for built-in tool context integration
func (cm *ContextManager) SetToolRegistry(toolRegistry *tools.ToolRegistry) {
	cm.toolRegistry = toolRegistry
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

	// Build full context with base project context
	processedMessages := cm.buildFullContext(messages, modelID)
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

// buildFullContext builds the full context including project context
func (cm *ContextManager) buildFullContext(messages []ConversationMessage, modelID string) []ConversationMessage {
	var fullContext []ConversationMessage

	// Add system message if not already present
	hasSystemMessage := false
	for _, msg := range messages {
		if msg.Role == "system" {
			hasSystemMessage = true
			break
		}
	}

	if !hasSystemMessage {
		// Build system prompt with project context
		systemPrompt := cm.buildSystemPrompt(modelID)
		fullContext = append(fullContext, ConversationMessage{
			Role:      "system",
			Content:   systemPrompt,
			Timestamp: time.Now().Unix(),
			Metadata:  map[string]interface{}{"type": "system_prompt"},
		})
	}

	// Add project context from AGENTS.md if available
	if cm.baseContext != nil && cm.baseContext.CombinedContent != "" {
		fullContext = append(fullContext, ConversationMessage{
			Role:      "system",
			Content:   fmt.Sprintf("## Project Overview\n\n%s", cm.baseContext.CombinedContent),
			Timestamp: time.Now().Unix(),
			Metadata:  map[string]interface{}{"type": "project_context"},
		})
	}

	// Add repository structure if available
	repoMap := cm.getRepositoryMap()
	if repoMap != "" {
		fullContext = append(fullContext, ConversationMessage{
			Role:      "system",
			Content:   fmt.Sprintf("## Repository Structure\n\n%s", repoMap),
			Timestamp: time.Now().Unix(),
			Metadata:  map[string]interface{}{"type": "repository_map"},
		})
	}

	// Add relevant symbols if available
	if cm.symbolExtractor != nil && len(messages) > 0 {
		// Extract query from the last user message
		query := ""
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				query = messages[i].Content
				break
			}
		}
		
		if query != "" {
			// Get relevant files from the repository context
			var files []string
			if cm.baseContext != nil {
				// Use project files from base context
				for _, file := range cm.baseContext.Files {
					files = append(files, file.Path)
				}
			}
			
			// Limit to reasonable number of files for symbol extraction
			maxFiles := 20
			if len(files) > maxFiles {
				files = files[:maxFiles]
			}
			
			symbols, err := cm.symbolExtractor.GetRelevantSymbols(context.Background(), query, files, 15)
			if err == nil && len(symbols) > 0 {
				var symbolsContent strings.Builder
				symbolsContent.WriteString("## Relevant Code Symbols\n\n")
				
				for _, sym := range symbols {
					symbolsContent.WriteString(fmt.Sprintf("- **%s** (%s) at %s", sym.Name, sym.Kind, sym.Location))
					if sym.Detail != "" {
						symbolsContent.WriteString(fmt.Sprintf(" - %s", sym.Detail))
					}
					symbolsContent.WriteString("\n")
					
					if len(sym.Children) > 0 {
						symbolsContent.WriteString(fmt.Sprintf("  Children: %s\n", strings.Join(sym.Children, ", ")))
					}
				}
				
				fullContext = append(fullContext, ConversationMessage{
					Role:      "system",
					Content:   symbolsContent.String(),
					Timestamp: time.Now().Unix(),
					Metadata:  map[string]interface{}{"type": "symbols", "symbol_count": len(symbols)},
				})
				
				log.Printf("Added %d relevant symbols to context", len(symbols))
			}
		}
	}

	// Add available tools (both built-in and MCP)
	toolsContext := cm.buildToolsContext()
	if toolsContext != "" {
		fullContext = append(fullContext, ConversationMessage{
			Role:      "system",
			Content:   toolsContext,
			Timestamp: time.Now().Unix(),
			Metadata:  map[string]interface{}{"type": "tools_context"},
		})
	}

	// Add original messages
	fullContext = append(fullContext, messages...)

	return fullContext
}

// buildSystemPrompt builds the system prompt including any coder prompts
func (cm *ContextManager) buildSystemPrompt(modelID string) string {
	// Parse provider from model ID
	provider := cm.parseProviderFromModelID(modelID)
	
	// Get the appropriate agent prompt - default to coder agent
	systemPrompt := prompt.GetAgentPrompt(config.AgentCoder, provider)
	
	return systemPrompt
}

// parseProviderFromModelID extracts the provider from a model ID
func (cm *ContextManager) parseProviderFromModelID(modelID string) models.ModelProvider {
	// Handle provider/model format
	if strings.Contains(modelID, "/") {
		parts := strings.Split(modelID, "/")
		if len(parts) >= 2 {
			providerStr := strings.ToLower(parts[0])
			switch providerStr {
			case "anthropic":
				return models.ProviderAnthropic
			case "openai":
				return models.ProviderOpenAI
			case "gemini", "google":
				return models.ProviderGemini
			case "groq":
				return models.ProviderGROQ
			case "openrouter":
				return models.ProviderOpenRouter
			}
		}
	}
	
	// Detect provider from model ID patterns
	modelLower := strings.ToLower(modelID)
	switch {
	case strings.HasPrefix(modelLower, "claude-"):
		return models.ProviderAnthropic
	case strings.HasPrefix(modelLower, "gpt-") || strings.HasPrefix(modelLower, "o1-"):
		return models.ProviderOpenAI
	case strings.HasPrefix(modelLower, "gemini-"):
		return models.ProviderGemini
	case strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mixtral"):
		return models.ProviderGROQ
	default:
		// Default to OpenAI-style prompts
		return models.ProviderOpenAI
	}
}

// getRepositoryMap returns the cached repository map or generates a new one
func (cm *ContextManager) getRepositoryMap() string {
	// Check if cache is still valid (refresh every hour)
	if cm.repoMapCache != "" && time.Since(cm.repoMapTime) < time.Hour {
		return cm.repoMapCache
	}

	// Generate new repository map
	workingDir := cm.config.WorkingDir
	if workingDir == "" {
		workingDir = "."
	}
	
	analyzer := project.NewRepositoryAnalyzer(workingDir)
	repoMap, err := analyzer.GenerateRepoMap()
	if err != nil {
		log.Printf("Warning: Failed to generate repository map: %v", err)
		return ""
	}
	
	// Generate markdown representation
	mapContent := repoMap.GenerateMarkdown()
	
	// Cache the result
	cm.repoMapCache = mapContent
	cm.repoMapTime = time.Now()
	
	log.Printf("Generated repository map: %d directories, %d files", 
		repoMap.Summary.TotalDirectories, repoMap.Summary.TotalFiles)
	
	return mapContent
}

// initializeCodeIntelligence initializes code intelligence providers
func (cm *ContextManager) initializeCodeIntelligence(workingDir string) {
	// Initialize codebase manager for graph-based code intelligence
	cm.codebaseManager = graph.NewCodebaseManager()
	if err := cm.codebaseManager.Initialize(workingDir); err != nil {
		log.Printf("Warning: Failed to initialize codebase manager: %v", err)
		// Continue without graph-based intelligence
		cm.codebaseManager = nil
	} else {
		log.Printf("Codebase awareness initialized for advanced code intelligence")
	}
	
	// Initialize LSP symbol extractor
	lspManager := lsp.GetManager()
	if lspManager != nil {
		cm.symbolExtractor = lsp.NewSymbolExtractor(lspManager)
		log.Printf("LSP symbol extractor initialized")
	} else {
		log.Printf("Warning: LSP manager not available, symbol extraction disabled")
	}
}

// GetCodeContext returns enhanced code context for a query
func (cm *ContextManager) GetCodeContext(ctx context.Context, query string, files []string) (string, error) {
	var contextBuilder strings.Builder
	
	// Get graph-based code intelligence if available
	if cm.codebaseManager != nil && cm.codebaseManager.IsInitialized() {
		contextBuilder.WriteString("## Code Intelligence\n\n")
		
		// Get intelligent context from graph
		graphContext := cm.codebaseManager.GetContext(query, files...)
		contextBuilder.WriteString(graphContext)
		contextBuilder.WriteString("\n\n")
	}
	
	// Get LSP symbols if available
	if cm.symbolExtractor != nil && len(files) > 0 {
		symbols, err := cm.symbolExtractor.GetRelevantSymbols(ctx, query, files, 20)
		if err == nil && len(symbols) > 0 {
			contextBuilder.WriteString("## Code Symbols\n\n")
			for _, sym := range symbols {
				contextBuilder.WriteString(fmt.Sprintf("- **%s** (%s) at %s\n", sym.Name, sym.Kind, sym.Location))
				if sym.Detail != "" {
					contextBuilder.WriteString(fmt.Sprintf("  %s\n", sym.Detail))
				}
			}
			contextBuilder.WriteString("\n")
		}
	}
	
	return contextBuilder.String(), nil
}

// buildToolsContext builds context information about available tools
func (cm *ContextManager) buildToolsContext() string {
	var toolsContent strings.Builder
	hasTools := false
	
	// Add built-in tools
	if cm.toolRegistry != nil {
		builtinTools := cm.toolRegistry.GetToolInfos()
		if len(builtinTools) > 0 {
			hasTools = true
			toolsContent.WriteString("## Available Tools\n\n")
			toolsContent.WriteString("You have access to the following tools to help users with their tasks:\n\n")
			toolsContent.WriteString("### Built-in Tools\n\n")
			
			for _, tool := range builtinTools {
				toolsContent.WriteString(fmt.Sprintf("- **%s**", tool.Name))
				if tool.Description != "" {
					toolsContent.WriteString(fmt.Sprintf(" - %s", tool.Description))
				}
				toolsContent.WriteString("\n")
			}
			toolsContent.WriteString("\n")
			
			log.Printf("Added %d built-in tools to context", len(builtinTools))
		}
	}
	
	// Add MCP tools
	if cm.mcpManager != nil {
		mcpTools := cm.mcpManager.GetAllAvailableTools()
		if len(mcpTools) > 0 {
			if !hasTools {
				toolsContent.WriteString("## Available Tools\n\n")
				toolsContent.WriteString("You have access to the following tools to help users with their tasks:\n\n")
				hasTools = true
			}
			
			toolsContent.WriteString("### MCP Server Tools\n\n")
			
			for serverName, serverTools := range mcpTools {
				if len(serverTools) > 0 {
					toolsContent.WriteString(fmt.Sprintf("**%s server:**\n", serverName))
					
					for _, tool := range serverTools {
						toolsContent.WriteString(fmt.Sprintf("- **%s**", tool.Name))
						if tool.Description != "" {
							toolsContent.WriteString(fmt.Sprintf(" - %s", tool.Description))
						}
						toolsContent.WriteString("\n")
					}
					toolsContent.WriteString("\n")
				}
			}
			
			log.Printf("Added MCP tools context for %d servers with tools", len(mcpTools))
		}
	}
	
	if hasTools {
		toolsContent.WriteString("Use these tools as needed to complete user requests. Each tool call should be properly formatted according to the tool's requirements.\n\n")
	}
	
	return toolsContent.String()
}
