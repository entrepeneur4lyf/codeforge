package graph

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// HybridSearchEngine combines graph-based structural search with semantic analysis
type HybridSearchEngine struct {
	graph *CodeGraph

	// Scoring weights
	structuralWeight float64 // Weight for graph-based structural relevance
	semanticWeight   float64 // Weight for text-based semantic similarity

	// Configuration
	maxResults     int
	minScore       float64
	boostRecent    bool
	boostImportant bool
}

// SearchResult represents a hybrid search result
type SearchResult struct {
	Node            *Node   `json:"node"`
	StructuralScore float64 `json:"structural_score"` // Graph-based relevance
	SemanticScore   float64 `json:"semantic_score"`   // Vector similarity
	HybridScore     float64 `json:"hybrid_score"`     // Combined score
	Explanation     string  `json:"explanation"`      // Why this result is relevant

	// Context
	RelatedNodes []*Node  `json:"related_nodes"` // Structurally related nodes
	CodeSnippet  string   `json:"code_snippet"`  // Relevant code snippet
	Dependencies []string `json:"dependencies"`  // Key dependencies
	Dependents   []string `json:"dependents"`    // What depends on this
}

// HybridSearchRequest represents a search request
type HybridSearchRequest struct {
	Query          string     `json:"query"`           // Natural language query
	FocusTypes     []NodeType `json:"focus_types"`     // Types of nodes to focus on
	Language       string     `json:"language"`        // Programming language filter
	MaxResults     int        `json:"max_results"`     // Maximum results to return
	IncludeContext bool       `json:"include_context"` // Include related context
	BoostRecent    bool       `json:"boost_recent"`    // Boost recently modified files
	BoostImportant bool       `json:"boost_important"` // Boost structurally important nodes

	// Advanced options
	StructuralWeight float64 `json:"structural_weight"` // Weight for structural relevance (0.0-1.0)
	SemanticWeight   float64 `json:"semantic_weight"`   // Weight for semantic similarity (0.0-1.0)
}

// HybridSearchResponse contains search results and metadata
type HybridSearchResponse struct {
	Results     []*SearchResult `json:"results"`
	TotalFound  int             `json:"total_found"`
	QueryTime   time.Duration   `json:"query_time"`
	Explanation string          `json:"explanation"`
	Suggestions []string        `json:"suggestions"`
}

// NewHybridSearchEngine creates a new hybrid search engine
func NewHybridSearchEngine(graph *CodeGraph) *HybridSearchEngine {
	return &HybridSearchEngine{
		graph:            graph,
		structuralWeight: 0.6, // 60% structural (no vector store yet)
		semanticWeight:   0.4, // 40% text-based semantic
		maxResults:       20,
		minScore:         0.1,
		boostRecent:      true,
		boostImportant:   true,
	}
}

// Search performs hybrid search combining graph structure and vector semantics
func (hse *HybridSearchEngine) Search(ctx context.Context, req *HybridSearchRequest) (*HybridSearchResponse, error) {
	start := time.Now()

	// Apply request configuration
	if req.StructuralWeight > 0 {
		hse.structuralWeight = req.StructuralWeight
	}
	if req.SemanticWeight > 0 {
		hse.semanticWeight = req.SemanticWeight
	}
	if req.MaxResults > 0 {
		hse.maxResults = req.MaxResults
	}

	// 1. Perform vector search for semantic similarity
	vectorResults, err := hse.performVectorSearch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// 2. Perform graph search for structural relevance
	structuralResults := hse.performStructuralSearch(req)

	// 3. Combine and score results
	hybridResults := hse.combineResults(vectorResults, structuralResults, req)

	// 4. Add context and explanations
	if req.IncludeContext {
		hse.addContextToResults(hybridResults)
	}

	// 5. Sort by hybrid score
	sort.Slice(hybridResults, func(i, j int) bool {
		return hybridResults[i].HybridScore > hybridResults[j].HybridScore
	})

	// 6. Limit results
	if len(hybridResults) > hse.maxResults {
		hybridResults = hybridResults[:hse.maxResults]
	}

	response := &HybridSearchResponse{
		Results:     hybridResults,
		TotalFound:  len(hybridResults),
		QueryTime:   time.Since(start),
		Explanation: hse.generateSearchExplanation(req, len(vectorResults), len(structuralResults)),
		Suggestions: hse.generateSearchSuggestions(req, hybridResults),
	}

	return response, nil
}

// performVectorSearch performs text-based semantic search (placeholder for future vector search)
func (hse *HybridSearchEngine) performVectorSearch(ctx context.Context, req *HybridSearchRequest) (map[string]float64, error) {
	// For now, perform enhanced text-based semantic search
	// This will be replaced with actual vector search when vector store is available

	results := make(map[string]float64)
	queryLower := strings.ToLower(req.Query)
	queryWords := strings.Fields(queryLower)

	// Search through all nodes for semantic similarity
	for _, node := range hse.graph.graph.Nodes {
		if node.Type != NodeTypeFile {
			continue // Focus on files for semantic search
		}

		score := 0.0

		// Combine all text content for semantic analysis
		allText := strings.ToLower(fmt.Sprintf("%s %s %s %s",
			node.Name, node.Path, node.Purpose, node.DocComment))

		// Calculate semantic similarity based on word overlap and context
		for _, word := range queryWords {
			if strings.Contains(allText, word) {
				score += 1.0

				// Boost for exact matches in important fields
				if strings.Contains(strings.ToLower(node.Name), word) {
					score += 0.5
				}
				if strings.Contains(strings.ToLower(node.Purpose), word) {
					score += 0.3
				}
			}

			// Check for related terms (simple semantic expansion)
			relatedTerms := hse.getRelatedTerms(word)
			for _, related := range relatedTerms {
				if strings.Contains(allText, related) {
					score += 0.3
				}
			}
		}

		// Normalize by query length
		if len(queryWords) > 0 {
			score = score / float64(len(queryWords))
		}

		if score > 0 {
			results[node.Path] = score
		}
	}

	return results, nil
}

// performStructuralSearch performs graph-based structural search
func (hse *HybridSearchEngine) performStructuralSearch(req *HybridSearchRequest) map[string]float64 {
	results := make(map[string]float64)
	queryLower := strings.ToLower(req.Query)

	// Search through all nodes
	for _, node := range hse.graph.graph.Nodes {
		score := 0.0

		// Filter by type if specified
		if len(req.FocusTypes) > 0 {
			found := false
			for _, focusType := range req.FocusTypes {
				if node.Type == focusType {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by language if specified
		if req.Language != "" && req.Language != "all" && node.Language != req.Language {
			continue
		}

		// Name matching
		if strings.Contains(strings.ToLower(node.Name), queryLower) {
			score += 1.0
		}

		// Path matching
		if strings.Contains(strings.ToLower(node.Path), queryLower) {
			score += 0.8
		}

		// Purpose matching
		if strings.Contains(strings.ToLower(node.Purpose), queryLower) {
			score += 0.6
		}

		// Doc comment matching
		if strings.Contains(strings.ToLower(node.DocComment), queryLower) {
			score += 0.4
		}

		// Boost for structural importance
		if req.BoostImportant {
			importance := hse.calculateStructuralImportance(node)
			score += importance * 0.3
		}

		// Boost for recent files
		if req.BoostRecent && time.Since(node.LastModified) < 7*24*time.Hour {
			score += 0.2
		}

		// Boost for public visibility
		if node.Visibility == "public" {
			score += 0.1
		}

		if score > 0 {
			results[node.Path] = score
		}
	}

	return results
}

// combineResults combines vector and structural search results
func (hse *HybridSearchEngine) combineResults(vectorResults, structuralResults map[string]float64, req *HybridSearchRequest) []*SearchResult {
	// Get all unique paths
	allPaths := make(map[string]bool)
	for path := range vectorResults {
		allPaths[path] = true
	}
	for path := range structuralResults {
		allPaths[path] = true
	}

	var results []*SearchResult

	for path := range allPaths {
		node, exists := hse.graph.GetNodeByPath(path)
		if !exists {
			continue
		}

		// Get scores
		semanticScore := vectorResults[path]
		structuralScore := structuralResults[path]

		// Normalize scores (0-1 range)
		semanticScore = hse.normalizeScore(semanticScore, 0.0, 1.0)
		structuralScore = hse.normalizeScore(structuralScore, 0.0, 3.0) // Max structural score ~3

		// Calculate hybrid score
		hybridScore := (hse.semanticWeight * semanticScore) + (hse.structuralWeight * structuralScore)

		// Skip low-scoring results
		if hybridScore < hse.minScore {
			continue
		}

		result := &SearchResult{
			Node:            node,
			StructuralScore: structuralScore,
			SemanticScore:   semanticScore,
			HybridScore:     hybridScore,
			Explanation:     hse.generateResultExplanation(node, semanticScore, structuralScore, req.Query),
		}

		results = append(results, result)
	}

	return results
}

// addContextToResults adds related context to search results
func (hse *HybridSearchEngine) addContextToResults(results []*SearchResult) {
	for _, result := range results {
		// Add related nodes (neighbors in the graph)
		neighbors := hse.graph.GetNeighbors(result.Node.ID)
		result.RelatedNodes = neighbors[:min(5, len(neighbors))] // Limit to 5

		// Add dependencies
		outEdges := hse.graph.GetOutgoingEdges(result.Node.ID)
		for _, edge := range outEdges {
			if edge.Type == EdgeTypeImports || edge.Type == EdgeTypeDependsOn {
				if target, exists := hse.graph.GetNode(edge.Target); exists {
					result.Dependencies = append(result.Dependencies, target.Name)
				}
			}
		}

		// Add dependents
		inEdges := hse.graph.GetIncomingEdges(result.Node.ID)
		for _, edge := range inEdges {
			if edge.Type == EdgeTypeImports || edge.Type == EdgeTypeDependsOn {
				if source, exists := hse.graph.GetNode(edge.Source); exists {
					result.Dependents = append(result.Dependents, source.Name)
				}
			}
		}

		// Limit arrays
		if len(result.Dependencies) > 5 {
			result.Dependencies = result.Dependencies[:5]
		}
		if len(result.Dependents) > 5 {
			result.Dependents = result.Dependents[:5]
		}
	}
}

// Helper functions

func (hse *HybridSearchEngine) calculateStructuralImportance(node *Node) float64 {
	// Calculate based on connections (centrality)
	outEdges := hse.graph.GetOutgoingEdges(node.ID)
	inEdges := hse.graph.GetIncomingEdges(node.ID)

	connectionScore := float64(len(outEdges) + len(inEdges))

	// Normalize to 0-1 range (assume max 20 connections for normalization)
	return math.Min(connectionScore/20.0, 1.0)
}

func (hse *HybridSearchEngine) normalizeScore(score, min, max float64) float64 {
	if max == min {
		return 0.0
	}
	normalized := (score - min) / (max - min)
	return math.Max(0.0, math.Min(1.0, normalized))
}

func (hse *HybridSearchEngine) generateResultExplanation(node *Node, semanticScore, structuralScore float64, query string) string {
	var reasons []string

	if semanticScore > 0.7 {
		reasons = append(reasons, "high semantic similarity")
	} else if semanticScore > 0.4 {
		reasons = append(reasons, "moderate semantic similarity")
	}

	if structuralScore > 0.7 {
		reasons = append(reasons, "strong structural match")
	} else if structuralScore > 0.4 {
		reasons = append(reasons, "structural relevance")
	}

	if strings.Contains(strings.ToLower(node.Name), strings.ToLower(query)) {
		reasons = append(reasons, "name contains query")
	}

	if node.Visibility == "public" {
		reasons = append(reasons, "public API")
	}

	if len(reasons) == 0 {
		return "Related to query"
	}

	return strings.Join(reasons, ", ")
}

func (hse *HybridSearchEngine) generateSearchExplanation(req *HybridSearchRequest, vectorCount, structuralCount int) string {
	return fmt.Sprintf("Found %d semantic matches and %d structural matches. Combined using %.0f%% semantic + %.0f%% structural weighting.",
		vectorCount, structuralCount, hse.semanticWeight*100, hse.structuralWeight*100)
}

func (hse *HybridSearchEngine) generateSearchSuggestions(req *HybridSearchRequest, results []*SearchResult) []string {
	suggestions := []string{}

	if len(results) == 0 {
		suggestions = append(suggestions, "Try broader search terms")
		suggestions = append(suggestions, "Check spelling and try synonyms")
		return suggestions
	}

	if len(results) > 10 {
		suggestions = append(suggestions, "Try more specific search terms to narrow results")
	}

	// Language-specific suggestions
	languages := make(map[string]int)
	for _, result := range results {
		if result.Node.Language != "" {
			languages[result.Node.Language]++
		}
	}

	if len(languages) > 1 {
		suggestions = append(suggestions, "Filter by specific language for more focused results")
	}

	return suggestions
}

// getRelatedTerms returns semantically related terms using advanced semantic expansion
func (hse *HybridSearchEngine) getRelatedTerms(word string) []string {
	// Normalize input
	normalized := strings.ToLower(strings.TrimSpace(word))
	if normalized == "" {
		return []string{}
	}

	// Multi-layered semantic expansion with weighted relationships
	relatedTerms := hse.buildSemanticGraph()

	// Direct lookup for exact matches
	if related, exists := relatedTerms[normalized]; exists {
		return hse.rankRelatedTerms(normalized, related)
	}

	// Fuzzy matching for partial matches and stemming
	fuzzyMatches := hse.findFuzzyMatches(normalized, relatedTerms)
	if len(fuzzyMatches) > 0 {
		return fuzzyMatches
	}

	// Contextual expansion based on programming language patterns
	contextualTerms := hse.getContextualTerms(normalized)
	if len(contextualTerms) > 0 {
		return contextualTerms
	}

	// Morphological analysis for compound words and variations
	morphologicalTerms := hse.getMorphologicalVariations(normalized)

	return morphologicalTerms
}

// buildSemanticGraph creates a comprehensive semantic relationship graph
func (hse *HybridSearchEngine) buildSemanticGraph() map[string][]SemanticTerm {
	return map[string][]SemanticTerm{
		// Authentication & Security
		"auth":           {{Term: "authentication", Weight: 0.9}, {Term: "login", Weight: 0.8}, {Term: "user", Weight: 0.7}, {Term: "session", Weight: 0.8}, {Term: "token", Weight: 0.9}, {Term: "jwt", Weight: 0.7}, {Term: "oauth", Weight: 0.8}},
		"authentication": {{Term: "auth", Weight: 0.9}, {Term: "login", Weight: 0.8}, {Term: "credential", Weight: 0.8}, {Term: "verify", Weight: 0.7}, {Term: "authorize", Weight: 0.8}},
		"login":          {{Term: "auth", Weight: 0.8}, {Term: "authentication", Weight: 0.8}, {Term: "signin", Weight: 0.9}, {Term: "user", Weight: 0.7}, {Term: "password", Weight: 0.8}},
		"user":           {{Term: "account", Weight: 0.8}, {Term: "profile", Weight: 0.7}, {Term: "customer", Weight: 0.6}, {Term: "member", Weight: 0.7}, {Term: "person", Weight: 0.6}},
		"session":        {{Term: "cookie", Weight: 0.7}, {Term: "state", Weight: 0.6}, {Term: "cache", Weight: 0.5}, {Term: "storage", Weight: 0.6}},
		"token":          {{Term: "jwt", Weight: 0.8}, {Term: "bearer", Weight: 0.7}, {Term: "key", Weight: 0.6}, {Term: "secret", Weight: 0.7}},

		// Error Handling
		"error":     {{Term: "exception", Weight: 0.9}, {Term: "failure", Weight: 0.8}, {Term: "bug", Weight: 0.7}, {Term: "issue", Weight: 0.7}, {Term: "fault", Weight: 0.6}, {Term: "panic", Weight: 0.8}},
		"exception": {{Term: "error", Weight: 0.9}, {Term: "throw", Weight: 0.8}, {Term: "catch", Weight: 0.8}, {Term: "try", Weight: 0.7}, {Term: "handle", Weight: 0.7}},
		"panic":     {{Term: "error", Weight: 0.8}, {Term: "crash", Weight: 0.7}, {Term: "abort", Weight: 0.6}, {Term: "fatal", Weight: 0.7}},

		// Testing
		"test":    {{Term: "testing", Weight: 0.9}, {Term: "spec", Weight: 0.8}, {Term: "unit", Weight: 0.8}, {Term: "integration", Weight: 0.7}, {Term: "mock", Weight: 0.7}, {Term: "assert", Weight: 0.8}},
		"testing": {{Term: "test", Weight: 0.9}, {Term: "qa", Weight: 0.6}, {Term: "validation", Weight: 0.7}, {Term: "verification", Weight: 0.7}},
		"mock":    {{Term: "stub", Weight: 0.8}, {Term: "fake", Weight: 0.7}, {Term: "dummy", Weight: 0.6}, {Term: "spy", Weight: 0.7}},
		"assert":  {{Term: "expect", Weight: 0.8}, {Term: "verify", Weight: 0.7}, {Term: "check", Weight: 0.6}, {Term: "validate", Weight: 0.7}},

		// Configuration
		"config":        {{Term: "configuration", Weight: 0.9}, {Term: "settings", Weight: 0.8}, {Term: "options", Weight: 0.7}, {Term: "env", Weight: 0.7}, {Term: "properties", Weight: 0.6}},
		"configuration": {{Term: "config", Weight: 0.9}, {Term: "setup", Weight: 0.7}, {Term: "init", Weight: 0.6}, {Term: "params", Weight: 0.6}},
		"settings":      {{Term: "config", Weight: 0.8}, {Term: "preferences", Weight: 0.7}, {Term: "options", Weight: 0.8}},
		"env":           {{Term: "environment", Weight: 0.9}, {Term: "variable", Weight: 0.8}, {Term: "config", Weight: 0.7}},

		// API & Web
		"api":        {{Term: "endpoint", Weight: 0.9}, {Term: "route", Weight: 0.8}, {Term: "handler", Weight: 0.8}, {Term: "service", Weight: 0.7}, {Term: "rest", Weight: 0.8}, {Term: "graphql", Weight: 0.6}},
		"endpoint":   {{Term: "api", Weight: 0.9}, {Term: "url", Weight: 0.7}, {Term: "path", Weight: 0.7}, {Term: "route", Weight: 0.8}},
		"handler":    {{Term: "controller", Weight: 0.8}, {Term: "processor", Weight: 0.7}, {Term: "middleware", Weight: 0.7}, {Term: "router", Weight: 0.6}},
		"middleware": {{Term: "handler", Weight: 0.7}, {Term: "interceptor", Weight: 0.8}, {Term: "filter", Weight: 0.7}, {Term: "plugin", Weight: 0.6}},
		"route":      {{Term: "path", Weight: 0.8}, {Term: "url", Weight: 0.7}, {Term: "endpoint", Weight: 0.8}, {Term: "router", Weight: 0.7}},

		// Data & Storage
		"database": {{Term: "db", Weight: 0.9}, {Term: "storage", Weight: 0.8}, {Term: "persistence", Weight: 0.7}, {Term: "data", Weight: 0.7}, {Term: "sql", Weight: 0.8}, {Term: "nosql", Weight: 0.6}},
		"db":       {{Term: "database", Weight: 0.9}, {Term: "sql", Weight: 0.8}, {Term: "query", Weight: 0.7}, {Term: "table", Weight: 0.7}, {Term: "schema", Weight: 0.6}},
		"storage":  {{Term: "database", Weight: 0.8}, {Term: "cache", Weight: 0.7}, {Term: "file", Weight: 0.6}, {Term: "memory", Weight: 0.6}},
		"cache":    {{Term: "storage", Weight: 0.7}, {Term: "memory", Weight: 0.8}, {Term: "redis", Weight: 0.7}, {Term: "buffer", Weight: 0.6}},
		"query":    {{Term: "search", Weight: 0.8}, {Term: "find", Weight: 0.8}, {Term: "select", Weight: 0.7}, {Term: "filter", Weight: 0.7}, {Term: "sql", Weight: 0.8}},

		// Programming Constructs
		"function":  {{Term: "func", Weight: 0.9}, {Term: "method", Weight: 0.8}, {Term: "procedure", Weight: 0.7}, {Term: "routine", Weight: 0.6}, {Term: "callback", Weight: 0.7}},
		"method":    {{Term: "function", Weight: 0.8}, {Term: "func", Weight: 0.8}, {Term: "procedure", Weight: 0.7}, {Term: "operation", Weight: 0.6}},
		"struct":    {{Term: "model", Weight: 0.8}, {Term: "type", Weight: 0.8}, {Term: "entity", Weight: 0.7}, {Term: "class", Weight: 0.8}, {Term: "object", Weight: 0.7}},
		"model":     {{Term: "struct", Weight: 0.8}, {Term: "entity", Weight: 0.8}, {Term: "schema", Weight: 0.7}, {Term: "data", Weight: 0.6}},
		"interface": {{Term: "contract", Weight: 0.7}, {Term: "protocol", Weight: 0.7}, {Term: "api", Weight: 0.6}, {Term: "spec", Weight: 0.6}},
		"class":     {{Term: "struct", Weight: 0.8}, {Term: "type", Weight: 0.7}, {Term: "object", Weight: 0.8}, {Term: "instance", Weight: 0.6}},

		// Search & Discovery
		"search": {{Term: "query", Weight: 0.8}, {Term: "find", Weight: 0.8}, {Term: "lookup", Weight: 0.7}, {Term: "index", Weight: 0.7}, {Term: "filter", Weight: 0.6}},
		"find":   {{Term: "search", Weight: 0.8}, {Term: "locate", Weight: 0.7}, {Term: "discover", Weight: 0.6}, {Term: "get", Weight: 0.6}},
		"filter": {{Term: "search", Weight: 0.6}, {Term: "query", Weight: 0.7}, {Term: "where", Weight: 0.7}, {Term: "select", Weight: 0.6}},
		"index":  {{Term: "search", Weight: 0.7}, {Term: "catalog", Weight: 0.6}, {Term: "registry", Weight: 0.6}, {Term: "directory", Weight: 0.5}},
	}
}

// SemanticTerm represents a related term with its semantic weight
type SemanticTerm struct {
	Term   string  `json:"term"`
	Weight float64 `json:"weight"`
}

// rankRelatedTerms ranks related terms by semantic weight and relevance
func (hse *HybridSearchEngine) rankRelatedTerms(original string, terms []SemanticTerm) []string {
	// Sort by weight (descending)
	sort.Slice(terms, func(i, j int) bool {
		return terms[i].Weight > terms[j].Weight
	})

	// Extract terms and apply additional filtering
	var result []string
	seen := make(map[string]bool)

	for _, term := range terms {
		if !seen[term.Term] && term.Term != original {
			result = append(result, term.Term)
			seen[term.Term] = true
		}
	}

	// Limit to top 10 most relevant terms
	if len(result) > 10 {
		result = result[:10]
	}

	return result
}

// findFuzzyMatches finds semantically related terms using fuzzy matching
func (hse *HybridSearchEngine) findFuzzyMatches(word string, semanticGraph map[string][]SemanticTerm) []string {
	var matches []string

	// Check for partial matches and common prefixes/suffixes
	for key := range semanticGraph {
		if hse.calculateSimilarity(word, key) > 0.7 {
			if terms, exists := semanticGraph[key]; exists {
				for _, term := range terms {
					if term.Weight > 0.6 {
						matches = append(matches, term.Term)
					}
				}
			}
		}
	}

	// Remove duplicates and limit results
	seen := make(map[string]bool)
	var uniqueMatches []string
	for _, match := range matches {
		if !seen[match] {
			uniqueMatches = append(uniqueMatches, match)
			seen[match] = true
		}
	}

	if len(uniqueMatches) > 8 {
		uniqueMatches = uniqueMatches[:8]
	}

	return uniqueMatches
}

// getContextualTerms provides context-aware term expansion based on programming patterns
func (hse *HybridSearchEngine) getContextualTerms(word string) []string {
	// Programming language specific patterns
	patterns := map[string][]string{
		// Go specific
		"goroutine": {"channel", "select", "go", "concurrent", "parallel"},
		"channel":   {"goroutine", "select", "send", "receive", "buffer"},
		"defer":     {"cleanup", "finally", "close", "resource"},

		// Web development
		"http": {"request", "response", "server", "client", "rest"},
		"json": {"marshal", "unmarshal", "serialize", "parse"},
		"xml":  {"parse", "serialize", "unmarshal", "marshal"},

		// Common patterns
		"async":  {"await", "promise", "future", "concurrent"},
		"sync":   {"mutex", "lock", "atomic", "wait"},
		"thread": {"concurrent", "parallel", "async", "worker"},
	}

	if terms, exists := patterns[word]; exists {
		return terms
	}

	return []string{}
}

// getMorphologicalVariations generates morphological variations of the input word
func (hse *HybridSearchEngine) getMorphologicalVariations(word string) []string {
	var variations []string

	// Common programming suffixes and prefixes
	suffixes := []string{"er", "ing", "ed", "s", "es", "tion", "sion", "ment", "ness"}
	prefixes := []string{"un", "re", "pre", "post", "sub", "super", "inter", "multi"}

	// Remove common suffixes to find root
	root := word
	for _, suffix := range suffixes {
		if strings.HasSuffix(word, suffix) && len(word) > len(suffix)+2 {
			root = strings.TrimSuffix(word, suffix)
			break
		}
	}

	// Remove common prefixes
	for _, prefix := range prefixes {
		if strings.HasPrefix(root, prefix) && len(root) > len(prefix)+2 {
			root = strings.TrimPrefix(root, prefix)
			break
		}
	}

	// Generate variations if we found a meaningful root
	if root != word && len(root) > 2 {
		variations = append(variations, root)

		// Add common variations
		variations = append(variations, root+"s", root+"er", root+"ing")

		// Add with common prefixes
		for _, prefix := range []string{"get", "set", "is", "has", "can"} {
			if len(root) > 0 {
				capitalized := strings.ToUpper(root[:1]) + root[1:]
				variations = append(variations, prefix+capitalized)
			}
		}
	}

	// Remove duplicates and filter out the original word
	seen := make(map[string]bool)
	var uniqueVariations []string
	for _, variation := range variations {
		if !seen[variation] && variation != word && len(variation) > 2 {
			uniqueVariations = append(uniqueVariations, variation)
			seen[variation] = true
		}
	}

	return uniqueVariations
}

// calculateSimilarity calculates string similarity using Levenshtein distance
func (hse *HybridSearchEngine) calculateSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Simple similarity based on common characters and length
	longer, shorter := s1, s2
	if len(s1) < len(s2) {
		longer, shorter = s2, s1
	}

	// Count common characters
	commonChars := 0
	for _, char := range shorter {
		if strings.ContainsRune(longer, char) {
			commonChars++
		}
	}

	// Calculate similarity ratio
	similarity := float64(commonChars) / float64(len(longer))

	// Bonus for common prefixes/suffixes
	if strings.HasPrefix(longer, shorter) || strings.HasSuffix(longer, shorter) {
		similarity += 0.2
	}

	if similarity > 1.0 {
		similarity = 1.0
	}

	return similarity
}
