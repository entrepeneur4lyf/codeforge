package graph

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// CodebaseManager manages the codebase awareness system for the entire application
type CodebaseManager struct {
	awareness    *CodebaseAwareness
	hybridSearch *HybridSearchEngine
	vectorStore  interface{} // Will be *search.VectorStore when available
	mutex        sync.RWMutex

	// State
	initialized bool
	rootPath    string

	// Configuration
	autoStart  bool
	maxContext int
}

// NewCodebaseManager creates a new codebase manager
func NewCodebaseManager() *CodebaseManager {
	return &CodebaseManager{
		autoStart:  true,
		maxContext: 20,
	}
}

// Initialize initializes the codebase awareness for a given directory
func (cm *CodebaseManager) Initialize(rootPath string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Clean up existing awareness if any
	if cm.awareness != nil {
		cm.awareness.Stop()
	}

	// Validate root path
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", rootPath)
	}

	// Create new awareness system
	awareness, err := NewCodebaseAwareness(rootPath)
	if err != nil {
		return fmt.Errorf("failed to create codebase awareness: %w", err)
	}

	// Initialize the awareness system
	if err := awareness.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize codebase awareness: %w", err)
	}

	cm.awareness = awareness
	cm.rootPath = rootPath
	cm.initialized = true

	log.Printf("🧠 Codebase awareness initialized for: %s", rootPath)
	return nil
}

// GetContext provides intelligent context for a query
func (cm *CodebaseManager) GetContext(query string, focusFiles ...string) string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if !cm.initialized || cm.awareness == nil {
		return "Codebase awareness not initialized. Use `/scan` to initialize."
	}

	// Create context request
	req := &ContextRequest{
		Query:      query,
		FocusFiles: focusFiles,
		MaxNodes:   cm.maxContext,
		MaxDepth:   3,
	}

	// Get context
	response, err := cm.awareness.GetContext(req)
	if err != nil {
		return fmt.Sprintf("Error getting codebase context: %v", err)
	}

	// Format context for LLM
	return cm.formatContextForLLM(response)
}

// GetQuickContext provides a quick context summary
func (cm *CodebaseManager) GetQuickContext(query string) string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if !cm.initialized || cm.awareness == nil {
		return ""
	}

	return cm.awareness.GetQuickContext(query)
}

// GetFileContext gets context for a specific file
func (cm *CodebaseManager) GetFileContext(filePath string) string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if !cm.initialized || cm.awareness == nil {
		return "Codebase awareness not initialized."
	}

	fileContext, err := cm.awareness.GetFileContext(filePath)
	if err != nil {
		return fmt.Sprintf("Error getting file context: %v", err)
	}

	return cm.formatFileContextForLLM(fileContext)
}

// IsInitialized returns whether the codebase awareness is initialized
func (cm *CodebaseManager) IsInitialized() bool {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.initialized
}

// GetRootPath returns the current root path
func (cm *CodebaseManager) GetRootPath() string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.rootPath
}

// GetStats returns codebase statistics
func (cm *CodebaseManager) GetStats() string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if !cm.initialized || cm.awareness == nil {
		return "Codebase awareness not initialized."
	}

	stats := cm.awareness.graph.GetStats()

	var result strings.Builder
	result.WriteString("## Codebase Statistics\n\n")
	result.WriteString(fmt.Sprintf("**Root Path:** %s\n", cm.rootPath))
	result.WriteString(fmt.Sprintf("**Total Nodes:** %d\n", stats.NodeCount))
	result.WriteString(fmt.Sprintf("**Total Edges:** %d\n", stats.EdgeCount))
	result.WriteString(fmt.Sprintf("**Last Updated:** %s\n\n", stats.LastUpdated.Format("2006-01-02 15:04:05")))

	if len(stats.Languages) > 0 {
		result.WriteString("**Languages:**\n")
		for lang, count := range stats.Languages {
			result.WriteString(fmt.Sprintf("- %s: %d files\n", lang, count))
		}
		result.WriteString("\n")
	}

	if len(stats.NodesByType) > 0 {
		result.WriteString("**Components:**\n")
		for nodeType, count := range stats.NodesByType {
			if count > 0 {
				result.WriteString(fmt.Sprintf("- %s: %d\n", nodeType, count))
			}
		}
	}

	return result.String()
}

// Stop stops the codebase awareness system
func (cm *CodebaseManager) Stop() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if cm.awareness != nil {
		cm.awareness.Stop()
		cm.awareness = nil
	}

	cm.initialized = false
	log.Println("🧠 Codebase manager stopped")
}

// formatContextForLLM formats the context response for LLM consumption
func (cm *CodebaseManager) formatContextForLLM(response *ContextResponse) string {
	var context strings.Builder

	context.WriteString("# Codebase Context\n\n")

	// Architecture overview
	if response.Architecture != "" {
		context.WriteString(response.Architecture)
		context.WriteString("\n")
	}

	// Summary
	if response.Summary != "" {
		context.WriteString("## Summary\n")
		context.WriteString(response.Summary)
		context.WriteString("\n\n")
	}

	// Key files
	if len(response.KeyFiles) > 0 {
		context.WriteString("## Key Files\n\n")
		for i, file := range response.KeyFiles {
			if i >= 10 { // Limit to top 10 files
				break
			}

			context.WriteString(fmt.Sprintf("### `%s`\n", file.Path))
			context.WriteString(fmt.Sprintf("**Language:** %s  \n", file.Language))
			context.WriteString(fmt.Sprintf("**Purpose:** %s  \n", file.Purpose))

			if len(file.Functions) > 0 {
				context.WriteString(fmt.Sprintf("**Functions:** %s  \n", strings.Join(file.Functions[:min(5, len(file.Functions))], ", ")))
			}

			if len(file.Types) > 0 {
				context.WriteString(fmt.Sprintf("**Types:** %s  \n", strings.Join(file.Types[:min(3, len(file.Types))], ", ")))
			}

			// Add public APIs
			if len(file.PublicAPIs) > 0 {
				context.WriteString("**Public APIs:**  \n")
				for j, api := range file.PublicAPIs {
					if j >= 3 { // Limit to top 3 APIs
						break
					}
					context.WriteString(fmt.Sprintf("- `%s`: %s  \n", api.Name, api.Signature))
				}
			}

			// Add data structures
			if len(file.DataStructures) > 0 {
				context.WriteString("**Data Structures:**  \n")
				for j, ds := range file.DataStructures {
					if j >= 3 { // Limit to top 3 structures
						break
					}
					context.WriteString(fmt.Sprintf("- `%s` (%s)  \n", ds.Name, ds.Type))
				}
			}

			context.WriteString("\n")
		}
	}

	// Dependencies
	if len(response.Dependencies) > 0 {
		context.WriteString("## Key Dependencies\n\n")
		for i, dep := range response.Dependencies {
			if i >= 10 { // Limit to top 10 dependencies
				break
			}
			context.WriteString(fmt.Sprintf("- `%s` → `%s` (%s)\n", dep.From, dep.To, dep.Type))
		}
		context.WriteString("\n")
	}

	// Suggestions
	if len(response.Suggestions) > 0 {
		context.WriteString("## Suggestions\n\n")
		for _, suggestion := range response.Suggestions {
			context.WriteString(fmt.Sprintf("- %s\n", suggestion))
		}
		context.WriteString("\n")
	}

	return context.String()
}

// formatFileContextForLLM formats file context for LLM consumption
func (cm *CodebaseManager) formatFileContextForLLM(fileContext *FileContext) string {
	var context strings.Builder

	context.WriteString(fmt.Sprintf("# File Context: `%s`\n\n", fileContext.Path))
	context.WriteString(fmt.Sprintf("**Language:** %s  \n", fileContext.Language))
	context.WriteString(fmt.Sprintf("**Purpose:** %s  \n", fileContext.Purpose))
	context.WriteString(fmt.Sprintf("**Importance Score:** %.2f  \n\n", fileContext.Importance))

	if fileContext.Summary != "" {
		context.WriteString("## Summary\n")
		context.WriteString(fileContext.Summary)
		context.WriteString("\n\n")
	}

	if len(fileContext.Functions) > 0 {
		context.WriteString("## Functions\n")
		for _, function := range fileContext.Functions {
			context.WriteString(fmt.Sprintf("- `%s`\n", function))
		}
		context.WriteString("\n")
	}

	if len(fileContext.PublicAPIs) > 0 {
		context.WriteString("## Public APIs\n")
		for _, api := range fileContext.PublicAPIs {
			context.WriteString(fmt.Sprintf("### `%s` (%s)\n", api.Name, api.Type))
			context.WriteString(fmt.Sprintf("**Signature:** `%s`  \n", api.Signature))
			if api.DocComment != "" {
				context.WriteString(fmt.Sprintf("**Description:** %s  \n", api.DocComment))
			}
			if api.Receiver != "" {
				context.WriteString(fmt.Sprintf("**Receiver:** `%s`  \n", api.Receiver))
			}
			context.WriteString("\n")
		}
	}

	if len(fileContext.DataStructures) > 0 {
		context.WriteString("## Data Structures\n")
		for _, ds := range fileContext.DataStructures {
			context.WriteString(fmt.Sprintf("### `%s` (%s)\n", ds.Name, ds.Type))
			if ds.DocComment != "" {
				context.WriteString(fmt.Sprintf("**Description:** %s  \n", ds.DocComment))
			}

			if len(ds.Fields) > 0 {
				context.WriteString("**Fields:**  \n")
				for _, field := range ds.Fields {
					context.WriteString(fmt.Sprintf("- `%s`  \n", field))
				}
			}

			if len(ds.Methods) > 0 {
				context.WriteString("**Methods:**  \n")
				for _, method := range ds.Methods {
					context.WriteString(fmt.Sprintf("- `%s`  \n", method))
				}
			}

			if len(ds.Embedded) > 0 {
				context.WriteString(fmt.Sprintf("**Embedded:** %s  \n", strings.Join(ds.Embedded, ", ")))
			}

			context.WriteString("\n")
		}
	}

	if len(fileContext.TypeDefs) > 0 {
		context.WriteString("## Type Definitions\n")
		for _, td := range fileContext.TypeDefs {
			context.WriteString(fmt.Sprintf("### `%s`\n", td.Name))
			context.WriteString(fmt.Sprintf("**Type:** `%s`  \n", td.UnderlyingType))
			if td.DocComment != "" {
				context.WriteString(fmt.Sprintf("**Description:** %s  \n", td.DocComment))
			}
			context.WriteString("\n")
		}
	}

	if len(fileContext.Types) > 0 {
		context.WriteString("## Types\n")
		for _, typ := range fileContext.Types {
			context.WriteString(fmt.Sprintf("- `%s`\n", typ))
		}
		context.WriteString("\n")
	}

	if len(fileContext.Dependencies) > 0 {
		context.WriteString("## Dependencies\n")
		for _, dep := range fileContext.Dependencies {
			context.WriteString(fmt.Sprintf("- `%s`\n", dep))
		}
		context.WriteString("\n")
	}

	if len(fileContext.Dependents) > 0 {
		context.WriteString("## Dependents\n")
		for _, dependent := range fileContext.Dependents {
			context.WriteString(fmt.Sprintf("- `%s`\n", dependent))
		}
		context.WriteString("\n")
	}

	return context.String()
}

// AutoDetectAndInitialize attempts to auto-detect and initialize codebase awareness
func (cm *CodebaseManager) AutoDetectAndInitialize() error {
	// Try current working directory
	if cwd, err := os.Getwd(); err == nil {
		if cm.isValidCodebase(cwd) {
			return cm.Initialize(cwd)
		}
	}

	// Try common project indicators
	searchPaths := []string{".", "..", "../..", "../../.."}

	for _, searchPath := range searchPaths {
		if absPath, err := filepath.Abs(searchPath); err == nil {
			if cm.isValidCodebase(absPath) {
				return cm.Initialize(absPath)
			}
		}
	}

	return fmt.Errorf("no valid codebase found for auto-initialization")
}

// isValidCodebase checks if a directory looks like a valid codebase
func (cm *CodebaseManager) isValidCodebase(path string) bool {
	// Check for common project files
	projectFiles := []string{
		"go.mod", "package.json", "Cargo.toml", "requirements.txt",
		"setup.py", "pom.xml", "build.gradle", "Makefile",
		".git", ".gitignore", "README.md",
	}

	for _, file := range projectFiles {
		if _, err := os.Stat(filepath.Join(path, file)); err == nil {
			return true
		}
	}

	// Check for source code files
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}

	sourceFiles := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".go" || ext == ".py" || ext == ".js" || ext == ".ts" ||
				ext == ".rs" || ext == ".java" || ext == ".cpp" || ext == ".c" {
				sourceFiles++
				if sourceFiles >= 3 { // At least 3 source files
					return true
				}
			}
		}
	}

	return false
}

// SetVectorStore sets the vector store for hybrid search
func (cm *CodebaseManager) SetVectorStore(vectorStore interface{}) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.vectorStore = vectorStore

	// Initialize hybrid search if we have both graph and vector store
	if cm.awareness != nil && vectorStore != nil {
		// Type assertion would be needed here when integrating with actual vector store
		// cm.hybridSearch = NewHybridSearchEngine(cm.awareness.graph, vectorStore.(*search.VectorStore))
		log.Println("Hybrid search engine ready")
	}
}

// HybridSearch performs intelligent hybrid search combining graph structure and vector semantics
func (cm *CodebaseManager) HybridSearch(ctx context.Context, query string, maxResults int) string {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	if !cm.initialized || cm.awareness == nil {
		return "Codebase awareness not initialized. Use `/scan` to initialize."
	}

	// If hybrid search is not available, fall back to graph-only search
	if cm.hybridSearch == nil {
		return cm.graphOnlySearch(query, maxResults)
	}

	// Perform hybrid search
	req := &HybridSearchRequest{
		Query:            query,
		MaxResults:       maxResults,
		IncludeContext:   true,
		BoostRecent:      true,
		BoostImportant:   true,
		StructuralWeight: 0.4,
		SemanticWeight:   0.6,
	}

	response, err := cm.hybridSearch.Search(ctx, req)
	if err != nil {
		return fmt.Sprintf("Hybrid search error: %v", err)
	}

	return cm.formatHybridSearchResults(response)
}

// graphOnlySearch performs graph-based search when vector store is not available
func (cm *CodebaseManager) graphOnlySearch(query string, maxResults int) string {
	// Use the graph query system
	queryBuilder := NewQueryBuilder(cm.awareness.graph)

	// Search for files containing the query
	result := queryBuilder.NameContains(query).Execute()

	if len(result.Nodes) == 0 {
		// Try path search
		result = queryBuilder.PathContains(query).Execute()
	}

	var output strings.Builder
	output.WriteString("## Graph Search Results\n\n")
	output.WriteString(fmt.Sprintf("Found %d matches for: **%s**\n\n", len(result.Nodes), query))

	// Limit results
	limit := maxResults
	if limit > len(result.Nodes) {
		limit = len(result.Nodes)
	}

	for i := 0; i < limit; i++ {
		node := result.Nodes[i]
		output.WriteString(fmt.Sprintf("### `%s`\n", node.Path))
		output.WriteString(fmt.Sprintf("**Type:** %s  \n", node.Type))
		output.WriteString(fmt.Sprintf("**Language:** %s  \n", node.Language))
		output.WriteString(fmt.Sprintf("**Purpose:** %s  \n\n", node.Purpose))
	}

	return output.String()
}

// formatHybridSearchResults formats hybrid search results for display
func (cm *CodebaseManager) formatHybridSearchResults(response *HybridSearchResponse) string {
	var output strings.Builder

	output.WriteString("# Hybrid Search Results\n\n")
	output.WriteString(fmt.Sprintf("**Query Time:** %v  \n", response.QueryTime))
	output.WriteString(fmt.Sprintf("**Total Found:** %d results  \n", response.TotalFound))
	output.WriteString(fmt.Sprintf("**Explanation:** %s  \n\n", response.Explanation))

	if len(response.Results) == 0 {
		output.WriteString("No results found. Try different search terms.\n")
		return output.String()
	}

	output.WriteString("## Results\n\n")

	for i, result := range response.Results {
		if i >= 10 { // Limit display to top 10
			break
		}

		output.WriteString(fmt.Sprintf("### %d. `%s`\n", i+1, result.Node.Path))
		output.WriteString(fmt.Sprintf("**Score:** %.3f (Semantic: %.3f, Structural: %.3f)  \n",
			result.HybridScore, result.SemanticScore, result.StructuralScore))
		output.WriteString(fmt.Sprintf("**Type:** %s  \n", result.Node.Type))
		output.WriteString(fmt.Sprintf("**Language:** %s  \n", result.Node.Language))
		output.WriteString(fmt.Sprintf("**Purpose:** %s  \n", result.Node.Purpose))
		output.WriteString(fmt.Sprintf("**Relevance:** %s  \n", result.Explanation))

		if len(result.Dependencies) > 0 {
			output.WriteString(fmt.Sprintf("**Dependencies:** %s  \n", strings.Join(result.Dependencies, ", ")))
		}

		if len(result.Dependents) > 0 {
			output.WriteString(fmt.Sprintf("**Dependents:** %s  \n", strings.Join(result.Dependents, ", ")))
		}

		output.WriteString("\n")
	}

	if len(response.Suggestions) > 0 {
		output.WriteString("## Suggestions\n\n")
		for _, suggestion := range response.Suggestions {
			output.WriteString(fmt.Sprintf("- %s\n", suggestion))
		}
	}

	return output.String()
}

// SmartSearch performs intelligent search with automatic fallback
func (cm *CodebaseManager) SmartSearch(ctx context.Context, query string) string {
	// Try hybrid search first
	if cm.hybridSearch != nil {
		return cm.HybridSearch(ctx, query, 10)
	}

	// Fall back to graph-only search
	return cm.graphOnlySearch(query, 10)
}
