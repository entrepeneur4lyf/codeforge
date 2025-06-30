package graph

import (
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CodebaseAwareness provides intelligent codebase context for LLM models
type CodebaseAwareness struct {
	graph   *CodeGraph
	watcher *FileWatcher
	scanner *SimpleScanner

	// Configuration
	rootPath        string
	maxContextNodes int
	maxContextDepth int
	autoUpdate      bool

	// Cache
	lastUpdate   time.Time
	contextCache map[string]*ContextResponse
	cacheTimeout time.Duration
}

// ContextRequest represents a request for codebase context
type ContextRequest struct {
	Query         string     `json:"query"`          // Natural language query
	FocusFiles    []string   `json:"focus_files"`    // Specific files to focus on
	IncludeTypes  []NodeType `json:"include_types"`  // Types of nodes to include
	MaxDepth      int        `json:"max_depth"`      // Maximum relationship depth
	MaxNodes      int        `json:"max_nodes"`      // Maximum nodes to return
	IncludeSource bool       `json:"include_source"` // Include source code snippets
	Language      string     `json:"language"`       // Filter by language
}

// ContextResponse contains the intelligent context for the model
type ContextResponse struct {
	Summary      string         `json:"summary"`      // High-level summary
	Architecture string         `json:"architecture"` // Architecture overview
	KeyFiles     []*FileContext `json:"key_files"`    // Important files with context
	Dependencies []*Dependency  `json:"dependencies"` // Key dependencies
	Suggestions  []string       `json:"suggestions"`  // Suggestions for the model
	LastUpdated  time.Time      `json:"last_updated"` // When context was generated
}

// FileContext represents a file with rich context information
type FileContext struct {
	Path         string   `json:"path"`         // File path
	Language     string   `json:"language"`     // Programming language
	Purpose      string   `json:"purpose"`      // What this file does
	Importance   float64  `json:"importance"`   // Calculated importance score
	Functions    []string `json:"functions"`    // Key functions/methods
	Types        []string `json:"types"`        // Key types/classes
	Dependencies []string `json:"dependencies"` // What this file depends on
	Dependents   []string `json:"dependents"`   // What depends on this file
	Summary      string   `json:"summary"`      // Brief summary of the file

	// Enhanced API information
	PublicAPIs     []APIInfo       `json:"public_apis"`     // Public functions and methods
	DataStructures []StructureInfo `json:"data_structures"` // Structs and interfaces
	TypeDefs       []TypeDefInfo   `json:"type_defs"`       // Type definitions
}

// APIInfo represents public API information
type APIInfo struct {
	Name        string       `json:"name"`
	Type        string       `json:"type"`      // "function", "method"
	Signature   string       `json:"signature"` // Human-readable signature
	Parameters  []Parameter  `json:"parameters"`
	ReturnTypes []ReturnType `json:"return_types"`
	Receiver    string       `json:"receiver,omitempty"` // For methods
	DocComment  string       `json:"doc_comment"`
	IsExported  bool         `json:"is_exported"`
}

// StructureInfo represents struct/interface information
type StructureInfo struct {
	Name       string   `json:"name"`
	Type       string   `json:"type"`     // "struct", "interface"
	Fields     []string `json:"fields"`   // Field names for structs
	Methods    []string `json:"methods"`  // Method names for interfaces
	Embedded   []string `json:"embedded"` // Embedded types
	DocComment string   `json:"doc_comment"`
	IsExported bool     `json:"is_exported"`
}

// TypeDefInfo represents type definition information
type TypeDefInfo struct {
	Name           string `json:"name"`
	UnderlyingType string `json:"underlying_type"`
	DocComment     string `json:"doc_comment"`
	IsExported     bool   `json:"is_exported"`
}

// Dependency represents a dependency relationship
type Dependency struct {
	From        string  `json:"from"`        // Source file/module
	To          string  `json:"to"`          // Target file/module
	Type        string  `json:"type"`        // Type of dependency (import, call, etc.)
	Strength    float64 `json:"strength"`    // Strength of the relationship
	Description string  `json:"description"` // Human-readable description
}

// NewCodebaseAwareness creates a new codebase awareness system
func NewCodebaseAwareness(rootPath string) (*CodebaseAwareness, error) {
	// Create graph
	graph := NewCodeGraph(rootPath)

	// Create scanner
	scanner := NewSimpleScanner(graph)

	// Create file watcher
	watcher, err := NewFileWatcher(graph, rootPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	ca := &CodebaseAwareness{
		graph:           graph,
		watcher:         watcher,
		scanner:         scanner,
		rootPath:        rootPath,
		maxContextNodes: 50,
		maxContextDepth: 3,
		autoUpdate:      true,
		contextCache:    make(map[string]*ContextResponse),
		cacheTimeout:    5 * time.Minute,
	}

	return ca, nil
}

// Initialize scans the codebase and starts watching for changes
func (ca *CodebaseAwareness) Initialize() error {
	log.Printf("🧠 Initializing codebase awareness for: %s", ca.rootPath)

	// Initial scan
	if err := ca.scanner.ScanRepository(ca.rootPath); err != nil {
		return fmt.Errorf("failed to scan repository: %w", err)
	}

	stats := ca.graph.GetStats()
	log.Printf("✅ Scanned %d files, %d functions, %d types",
		stats.NodesByType[NodeTypeFile],
		stats.NodesByType[NodeTypeFunction],
		stats.NodesByType[NodeTypeStruct]+stats.NodesByType[NodeTypeInterface])

	// Start file watcher if auto-update is enabled
	if ca.autoUpdate {
		if err := ca.watcher.Start(); err != nil {
			return fmt.Errorf("failed to start file watcher: %w", err)
		}
	}

	ca.lastUpdate = time.Now()
	return nil
}

// Stop stops the codebase awareness system
func (ca *CodebaseAwareness) Stop() {
	if ca.watcher != nil {
		ca.watcher.Stop()
	}
	log.Println("🧠 Codebase awareness stopped")
}

// GetContext provides intelligent context based on a request
func (ca *CodebaseAwareness) GetContext(req *ContextRequest) (*ContextResponse, error) {
	// Check cache first
	cacheKey := ca.generateCacheKey(req)
	if cached, exists := ca.contextCache[cacheKey]; exists {
		if time.Since(cached.LastUpdated) < ca.cacheTimeout {
			return cached, nil
		}
	}

	// Generate new context
	response := &ContextResponse{
		KeyFiles:     make([]*FileContext, 0),
		Dependencies: make([]*Dependency, 0),
		Suggestions:  make([]string, 0),
		LastUpdated:  time.Now(),
	}

	// 1. Generate architecture overview
	response.Architecture = ca.generateArchitectureOverview()

	// 2. Find relevant files
	relevantFiles := ca.findRelevantFiles(req)

	// 3. Build file contexts
	response.KeyFiles = ca.buildFileContexts(relevantFiles, req)

	// 4. Extract dependencies
	response.Dependencies = ca.extractDependencies(relevantFiles)

	// 5. Generate summary
	response.Summary = ca.generateSummary(response)

	// 6. Add suggestions
	response.Suggestions = ca.generateSuggestions(req, response)

	// Cache the response
	ca.contextCache[cacheKey] = response

	return response, nil
}

// GetQuickContext provides a quick context summary for immediate use
func (ca *CodebaseAwareness) GetQuickContext(query string) string {
	req := &ContextRequest{
		Query:    query,
		MaxNodes: 10,
		MaxDepth: 2,
	}

	response, err := ca.GetContext(req)
	if err != nil {
		return fmt.Sprintf("Error getting context: %v", err)
	}

	var context strings.Builder
	context.WriteString("## Codebase Context\n\n")
	context.WriteString(response.Summary)
	context.WriteString("\n\n")

	if len(response.KeyFiles) > 0 {
		context.WriteString("### Key Files:\n")
		for _, file := range response.KeyFiles[:min(5, len(response.KeyFiles))] {
			context.WriteString(fmt.Sprintf("- `%s`: %s\n", file.Path, file.Purpose))
		}
		context.WriteString("\n")
	}

	if len(response.Suggestions) > 0 {
		context.WriteString("### Suggestions:\n")
		for _, suggestion := range response.Suggestions[:min(3, len(response.Suggestions))] {
			context.WriteString(fmt.Sprintf("- %s\n", suggestion))
		}
	}

	return context.String()
}

// GetFileContext gets context for a specific file
func (ca *CodebaseAwareness) GetFileContext(filePath string) (*FileContext, error) {
	node, exists := ca.graph.GetNodeByPath(filePath)
	if !exists {
		return nil, fmt.Errorf("file not found in graph: %s", filePath)
	}

	return ca.buildFileContext(node), nil
}

// generateArchitectureOverview creates a high-level architecture description
func (ca *CodebaseAwareness) generateArchitectureOverview() string {
	stats := ca.graph.GetStats()

	var overview strings.Builder
	overview.WriteString("## Codebase Architecture\n\n")

	// Project info
	projectName := filepath.Base(ca.rootPath)
	overview.WriteString(fmt.Sprintf("**Project:** %s\n", projectName))
	overview.WriteString(fmt.Sprintf("**Total Files:** %d\n", stats.NodesByType[NodeTypeFile]))
	overview.WriteString(fmt.Sprintf("**Last Updated:** %s\n\n", ca.lastUpdate.Format("2006-01-02 15:04:05")))

	// Language breakdown
	if len(stats.Languages) > 0 {
		overview.WriteString("**Languages:**\n")
		for lang, count := range stats.Languages {
			if count > 0 {
				overview.WriteString(fmt.Sprintf("- %s: %d files\n", lang, count))
			}
		}
		overview.WriteString("\n")
	}

	// Key directories
	directories := ca.graph.GetNodesByType(NodeTypeDirectory)
	if len(directories) > 0 {
		overview.WriteString("**Key Directories:**\n")
		for _, dir := range directories[:min(8, len(directories))] {
			if dir.Path != "" { // Skip root
				overview.WriteString(fmt.Sprintf("- `%s/`: %s\n", dir.Path, dir.Purpose))
			}
		}
		overview.WriteString("\n")
	}

	return overview.String()
}

// findRelevantFiles finds files relevant to the context request
func (ca *CodebaseAwareness) findRelevantFiles(req *ContextRequest) []*Node {
	var relevantFiles []*Node

	// Start with focus files if specified
	if len(req.FocusFiles) > 0 {
		for _, filePath := range req.FocusFiles {
			if node, exists := ca.graph.GetNodeByPath(filePath); exists {
				relevantFiles = append(relevantFiles, node)
			}
		}
	}

	// Add files based on query keywords
	if req.Query != "" {
		queryFiles := ca.searchFilesByQuery(req.Query)
		relevantFiles = append(relevantFiles, queryFiles...)
	}

	// If no specific files, get most important files
	if len(relevantFiles) == 0 {
		allFiles := ca.graph.GetNodesByType(NodeTypeFile)
		importantFiles := ca.getMostImportantFiles(allFiles, req.MaxNodes)
		relevantFiles = append(relevantFiles, importantFiles...)
	}

	// Filter by language if specified
	if req.Language != "" && req.Language != "all" {
		filtered := make([]*Node, 0)
		for _, file := range relevantFiles {
			if file.Language == req.Language {
				filtered = append(filtered, file)
			}
		}
		relevantFiles = filtered
	}

	// Remove duplicates and limit
	seen := make(map[string]bool)
	unique := make([]*Node, 0)
	for _, file := range relevantFiles {
		if !seen[file.ID] && len(unique) < req.MaxNodes {
			seen[file.ID] = true
			unique = append(unique, file)
		}
	}

	return unique
}

// Helper functions

func (ca *CodebaseAwareness) generateCacheKey(req *ContextRequest) string {
	return fmt.Sprintf("%s_%v_%v_%d_%d",
		req.Query, req.FocusFiles, req.IncludeTypes, req.MaxDepth, req.MaxNodes)
}

func (ca *CodebaseAwareness) searchFilesByQuery(query string) []*Node {
	queryLower := strings.ToLower(query)
	var matches []*Node

	for _, node := range ca.graph.graph.Nodes {
		if node.Type == NodeTypeFile {
			// Search in file name, path, and purpose
			if strings.Contains(strings.ToLower(node.Name), queryLower) ||
				strings.Contains(strings.ToLower(node.Path), queryLower) ||
				strings.Contains(strings.ToLower(node.Purpose), queryLower) {
				matches = append(matches, node)
			}
		}
	}

	return matches
}

func (ca *CodebaseAwareness) getMostImportantFiles(files []*Node, limit int) []*Node {
	// Calculate importance scores
	type fileWithScore struct {
		node  *Node
		score float64
	}

	scored := make([]fileWithScore, 0, len(files))

	for _, file := range files {
		score := ca.calculateFileImportance(file)
		scored = append(scored, fileWithScore{file, score})
	}

	// Sort by score
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Return top files
	result := make([]*Node, 0, limit)
	for i, fs := range scored {
		if i >= limit {
			break
		}
		result = append(result, fs.node)
	}

	return result
}

func (ca *CodebaseAwareness) calculateFileImportance(file *Node) float64 {
	score := 1.0

	// Boost for main files
	if strings.Contains(strings.ToLower(file.Name), "main") {
		score += 2.0
	}

	// Boost for Go files (since we parse them better)
	if file.Language == "go" {
		score += 1.0
	}

	// Boost based on connections
	outEdges := ca.graph.GetOutgoingEdges(file.ID)
	inEdges := ca.graph.GetIncomingEdges(file.ID)
	score += float64(len(outEdges)+len(inEdges)) * 0.1

	// Boost for recent files
	if time.Since(file.LastModified) < 24*time.Hour {
		score += 0.5
	}

	return score
}

// buildFileContexts creates file contexts with rich information
func (ca *CodebaseAwareness) buildFileContexts(files []*Node, req *ContextRequest) []*FileContext {
	contexts := make([]*FileContext, 0, len(files))

	for _, file := range files {
		context := ca.buildFileContext(file)
		contexts = append(contexts, context)
	}

	// Sort by importance
	sort.Slice(contexts, func(i, j int) bool {
		return contexts[i].Importance > contexts[j].Importance
	})

	return contexts
}

// buildFileContext creates a file context for a single file
func (ca *CodebaseAwareness) buildFileContext(file *Node) *FileContext {
	context := &FileContext{
		Path:           file.Path,
		Language:       file.Language,
		Purpose:        file.Purpose,
		Importance:     ca.calculateFileImportance(file),
		Functions:      make([]string, 0),
		Types:          make([]string, 0),
		Dependencies:   make([]string, 0),
		Dependents:     make([]string, 0),
		PublicAPIs:     make([]APIInfo, 0),
		DataStructures: make([]StructureInfo, 0),
		TypeDefs:       make([]TypeDefInfo, 0),
	}

	// Find functions and types defined in this file
	outEdges := ca.graph.GetOutgoingEdges(file.ID)
	for _, edge := range outEdges {
		if edge.Type == EdgeTypeDefines {
			if target, exists := ca.graph.GetNode(edge.Target); exists {
				switch target.Type {
				case NodeTypeFunction:
					context.Functions = append(context.Functions, target.Name)

					// Add to public APIs if exported
					if target.APISignature != nil && target.APISignature.IsExported {
						apiInfo := APIInfo{
							Name:        target.APISignature.Name,
							Type:        "function",
							Signature:   target.Signature,
							Parameters:  target.APISignature.Parameters,
							ReturnTypes: target.APISignature.ReturnTypes,
							DocComment:  target.APISignature.DocComment,
							IsExported:  target.APISignature.IsExported,
						}

						if target.APISignature.IsMethod && target.APISignature.Receiver != nil {
							apiInfo.Type = "method"
							apiInfo.Receiver = target.APISignature.Receiver.Type
						}

						context.PublicAPIs = append(context.PublicAPIs, apiInfo)
					}

				case NodeTypeStruct, NodeTypeInterface:
					context.Types = append(context.Types, target.Name)

					// Add to data structures if exported
					if target.DataStructure != nil && target.DataStructure.IsExported {
						structInfo := StructureInfo{
							Name:       target.DataStructure.Name,
							Type:       target.DataStructure.Type,
							DocComment: target.DataStructure.DocComment,
							IsExported: target.DataStructure.IsExported,
							Fields:     make([]string, 0),
							Methods:    make([]string, 0),
							Embedded:   target.DataStructure.Embedded,
						}

						// Extract field names for structs
						for _, field := range target.DataStructure.Fields {
							if field.IsExported {
								structInfo.Fields = append(structInfo.Fields, fmt.Sprintf("%s %s", field.Name, field.Type))
							}
						}

						// Extract method names for interfaces
						for _, method := range target.DataStructure.Methods {
							if method.IsExported {
								structInfo.Methods = append(structInfo.Methods, method.Name)
							}
						}

						context.DataStructures = append(context.DataStructures, structInfo)
					}

				case NodeTypeType:
					context.Types = append(context.Types, target.Name)

					// Add to type definitions if exported
					if target.TypeInfo != nil && target.TypeInfo.IsExported {
						typeDefInfo := TypeDefInfo{
							Name:           target.TypeInfo.Name,
							UnderlyingType: target.TypeInfo.UnderlyingType,
							DocComment:     target.TypeInfo.DocComment,
							IsExported:     target.TypeInfo.IsExported,
						}
						context.TypeDefs = append(context.TypeDefs, typeDefInfo)
					}
				}
			}
		} else if edge.Type == EdgeTypeImports {
			if target, exists := ca.graph.GetNode(edge.Target); exists {
				context.Dependencies = append(context.Dependencies, target.Name)
			}
		}
	}

	// Find what depends on this file
	inEdges := ca.graph.GetIncomingEdges(file.ID)
	for _, edge := range inEdges {
		if edge.Type == EdgeTypeImports || edge.Type == EdgeTypeUses {
			if source, exists := ca.graph.GetNode(edge.Source); exists {
				context.Dependents = append(context.Dependents, source.Path)
			}
		}
	}

	// Generate summary
	context.Summary = ca.generateFileSummary(context)

	return context
}

// extractDependencies extracts key dependencies between files
func (ca *CodebaseAwareness) extractDependencies(files []*Node) []*Dependency {
	dependencies := make([]*Dependency, 0)
	seen := make(map[string]bool)

	for _, file := range files {
		outEdges := ca.graph.GetOutgoingEdges(file.ID)
		for _, edge := range outEdges {
			if edge.Type == EdgeTypeImports {
				if target, exists := ca.graph.GetNode(edge.Target); exists {
					depKey := fmt.Sprintf("%s->%s", file.Path, target.Name)
					if !seen[depKey] {
						seen[depKey] = true

						dep := &Dependency{
							From:        file.Path,
							To:          target.Name,
							Type:        "import",
							Strength:    1.0,
							Description: fmt.Sprintf("%s imports %s", file.Name, target.Name),
						}
						dependencies = append(dependencies, dep)
					}
				}
			}
		}
	}

	return dependencies
}

// generateSummary generates a summary of the context
func (ca *CodebaseAwareness) generateSummary(response *ContextResponse) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("This codebase contains %d key files", len(response.KeyFiles)))

	// Language breakdown
	languages := make(map[string]int)
	for _, file := range response.KeyFiles {
		if file.Language != "" {
			languages[file.Language]++
		}
	}

	if len(languages) > 0 {
		summary.WriteString(" written in ")
		langList := make([]string, 0, len(languages))
		for lang, count := range languages {
			langList = append(langList, fmt.Sprintf("%s (%d)", lang, count))
		}
		summary.WriteString(strings.Join(langList, ", "))
	}

	summary.WriteString(".")

	// Key components
	if len(response.KeyFiles) > 0 {
		summary.WriteString(" Key components include:")
		for i, file := range response.KeyFiles[:min(3, len(response.KeyFiles))] {
			if i == 0 {
				summary.WriteString(" ")
			} else if i == len(response.KeyFiles)-1 {
				summary.WriteString(" and ")
			} else {
				summary.WriteString(", ")
			}
			summary.WriteString(fmt.Sprintf("`%s` (%s)", file.Path, file.Purpose))
		}
		summary.WriteString(".")
	}

	return summary.String()
}

// generateSuggestions generates suggestions for the model
func (ca *CodebaseAwareness) generateSuggestions(req *ContextRequest, response *ContextResponse) []string {
	suggestions := make([]string, 0)

	// Query-based suggestions
	if req.Query != "" {
		queryLower := strings.ToLower(req.Query)

		if strings.Contains(queryLower, "test") {
			suggestions = append(suggestions, "Consider looking at test files to understand expected behavior")
		}

		if strings.Contains(queryLower, "build") || strings.Contains(queryLower, "compile") {
			suggestions = append(suggestions, "Check build configuration files like Makefile, go.mod, or package.json")
		}

		if strings.Contains(queryLower, "error") || strings.Contains(queryLower, "bug") {
			suggestions = append(suggestions, "Look at recent changes and error handling patterns in the codebase")
		}

		if strings.Contains(queryLower, "api") || strings.Contains(queryLower, "endpoint") {
			suggestions = append(suggestions, "Focus on HTTP handlers and API route definitions")
		}
	}

	// File-based suggestions
	hasMain := false
	hasTests := false
	for _, file := range response.KeyFiles {
		if strings.Contains(strings.ToLower(file.Path), "main") {
			hasMain = true
		}
		if strings.Contains(strings.ToLower(file.Path), "test") {
			hasTests = true
		}
	}

	if hasMain {
		suggestions = append(suggestions, "Start by examining the main entry points to understand program flow")
	}

	if hasTests {
		suggestions = append(suggestions, "Review test files to understand expected functionality and usage patterns")
	}

	// General suggestions
	suggestions = append(suggestions, "Use the dependency relationships to understand how components interact")
	suggestions = append(suggestions, "Pay attention to public vs private functions to understand the intended API")

	return suggestions
}

// generateFileSummary generates a summary for a single file
func (ca *CodebaseAwareness) generateFileSummary(context *FileContext) string {
	var summary strings.Builder

	summary.WriteString(context.Purpose)

	if len(context.Functions) > 0 {
		summary.WriteString(fmt.Sprintf(" Contains %d functions", len(context.Functions)))
		if len(context.Functions) <= 3 {
			summary.WriteString(": ")
			summary.WriteString(strings.Join(context.Functions, ", "))
		}
		summary.WriteString(".")
	}

	if len(context.Types) > 0 {
		summary.WriteString(fmt.Sprintf(" Defines %d types", len(context.Types)))
		if len(context.Types) <= 3 {
			summary.WriteString(": ")
			summary.WriteString(strings.Join(context.Types, ", "))
		}
		summary.WriteString(".")
	}

	if len(context.Dependencies) > 0 {
		summary.WriteString(fmt.Sprintf(" Imports %d dependencies.", len(context.Dependencies)))
	}

	return summary.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
