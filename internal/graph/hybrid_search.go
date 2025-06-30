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

// getRelatedTerms returns semantically related terms for simple expansion
func (hse *HybridSearchEngine) getRelatedTerms(word string) []string {
	// Simple semantic expansion - can be enhanced with more sophisticated NLP
	relatedTerms := map[string][]string{
		"auth":           {"authentication", "login", "user", "session", "token"},
		"authentication": {"auth", "login", "user", "session", "token"},
		"login":          {"auth", "authentication", "user", "session"},
		"user":           {"auth", "authentication", "login", "account"},
		"error":          {"exception", "failure", "bug", "issue"},
		"exception":      {"error", "failure", "bug", "issue"},
		"test":           {"testing", "spec", "unit", "integration"},
		"testing":        {"test", "spec", "unit", "integration"},
		"config":         {"configuration", "settings", "options"},
		"configuration":  {"config", "settings", "options"},
		"api":            {"endpoint", "route", "handler", "service"},
		"endpoint":       {"api", "route", "handler", "service"},
		"handler":        {"api", "endpoint", "route", "controller"},
		"database":       {"db", "storage", "persistence", "data"},
		"db":             {"database", "storage", "persistence", "data"},
		"search":         {"query", "find", "lookup", "index"},
		"query":          {"search", "find", "lookup", "filter"},
		"model":          {"struct", "type", "entity", "data"},
		"struct":         {"model", "type", "entity", "class"},
		"function":       {"func", "method", "procedure", "routine"},
		"method":         {"function", "func", "procedure", "routine"},
	}

	if related, exists := relatedTerms[word]; exists {
		return related
	}
	return []string{}
}
