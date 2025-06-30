package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/analysis"
	"github.com/entrepeneur4lyf/codeforge/internal/chunking"
	"github.com/entrepeneur4lyf/codeforge/internal/git"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
	"github.com/mark3labs/mcp-go/mcp"
)

// textSearchResult represents a text-based search result
type textSearchResult struct {
	Chunk *vectordb.CodeChunk
	Score float64
}

// Tool Handlers

// handleGitCommitAI handles AI-powered git commit requests
func (cfs *CodeForgeServer) handleGitCommitAI(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	staged := false
	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if val, ok := args["staged"].(bool); ok {
			staged = val
		}
	}

	// Create git repository instance
	repo := git.NewRepository(cfs.workspaceRoot)

	// Check if git is available and this is a git repository
	if !git.IsGitInstalled() {
		return mcp.NewToolResultError("Git is not installed"), nil
	}

	if !repo.IsGitRepository() {
		return mcp.NewToolResultError("Not a git repository"), nil
	}

	// Generate and commit with AI message
	commitMessage, err := repo.CommitWithAIMessage(ctx, staged)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to commit with AI message: %v", err)), nil
	}

	// Format results
	result := map[string]interface{}{
		"success":        true,
		"commit_message": commitMessage,
		"staged":         staged,
		"message":        fmt.Sprintf("Successfully committed with AI-generated message: %s", commitMessage),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGitGenerateCommitMessage handles AI commit message generation without committing
func (cfs *CodeForgeServer) handleGitGenerateCommitMessage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	staged := false
	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if val, ok := args["staged"].(bool); ok {
			staged = val
		}
	}

	// Create git repository instance
	repo := git.NewRepository(cfs.workspaceRoot)

	// Check if git is available and this is a git repository
	if !git.IsGitInstalled() {
		return mcp.NewToolResultError("Git is not installed"), nil
	}

	if !repo.IsGitRepository() {
		return mcp.NewToolResultError("Not a git repository"), nil
	}

	// Create commit message generator
	generator, err := git.NewCommitMessageGenerator()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create commit message generator: %v", err)), nil
	}

	// Generate commit message
	commitMessage, err := generator.GenerateCommitMessage(ctx, repo, staged)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to generate commit message: %v", err)), nil
	}

	// Format results
	result := map[string]interface{}{
		"success":        true,
		"commit_message": commitMessage,
		"staged":         staged,
		"message":        "AI-generated commit message ready",
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGitDetectConflicts handles git conflict detection requests
func (cfs *CodeForgeServer) handleGitDetectConflicts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Create git repository instance
	repo := git.NewRepository(cfs.workspaceRoot)

	// Check if git is available and this is a git repository
	if !git.IsGitInstalled() {
		return mcp.NewToolResultError("Git is not installed"), nil
	}

	if !repo.IsGitRepository() {
		return mcp.NewToolResultError("Not a git repository"), nil
	}

	// Detect conflicts
	conflicts, err := repo.DetectConflicts(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to detect conflicts: %v", err)), nil
	}

	// Format results
	result := map[string]interface{}{
		"conflicts_found": len(conflicts) > 0,
		"conflict_count":  len(conflicts),
		"conflicts":       conflicts,
	}

	if len(conflicts) == 0 {
		result["message"] = "No merge conflicts detected"
	} else {
		result["message"] = fmt.Sprintf("Found %d file(s) with merge conflicts", len(conflicts))
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGitResolveConflicts handles git conflict resolution requests
func (cfs *CodeForgeServer) handleGitResolveConflicts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	autoApply := false
	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if val, ok := args["auto_apply"].(bool); ok {
			autoApply = val
		}
	}

	// Create git repository instance
	repo := git.NewRepository(cfs.workspaceRoot)

	// Check if git is available and this is a git repository
	if !git.IsGitInstalled() {
		return mcp.NewToolResultError("Git is not installed"), nil
	}

	if !repo.IsGitRepository() {
		return mcp.NewToolResultError("Not a git repository"), nil
	}

	// Detect conflicts first
	conflicts, err := repo.DetectConflicts(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to detect conflicts: %v", err)), nil
	}

	if len(conflicts) == 0 {
		result := map[string]interface{}{
			"success": true,
			"message": "No merge conflicts found to resolve",
		}
		resultJSON, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(resultJSON)), nil
	}

	// Create conflict resolver
	resolver, err := git.NewConflictResolver()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create conflict resolver: %v", err)), nil
	}

	// Get AI-powered resolutions
	resolutions, err := resolver.ResolveConflicts(ctx, conflicts)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to resolve conflicts: %v", err)), nil
	}

	// Apply resolutions if requested
	var appliedCount int
	if autoApply {
		for _, resolution := range resolutions {
			if err := repo.ApplyResolution(ctx, resolution); err == nil {
				appliedCount++
			}
		}
	}

	// Format results
	result := map[string]interface{}{
		"success":           true,
		"conflicts_found":   len(conflicts),
		"resolutions_count": len(resolutions),
		"resolutions":       resolutions,
		"auto_applied":      autoApply,
		"applied_count":     appliedCount,
	}

	if autoApply {
		result["message"] = fmt.Sprintf("Generated %d resolution(s), applied %d automatically", len(resolutions), appliedCount)
	} else {
		result["message"] = fmt.Sprintf("Generated %d AI-powered conflict resolution(s). Use auto_apply=true to apply automatically", len(resolutions))
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleSemanticSearch handles semantic code search requests
func (cfs *CodeForgeServer) handleSemanticSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	// Get optional parameters
	maxResults := int(request.GetFloat("max_results", 10))
	language := request.GetString("language", "")
	chunkType := request.GetString("chunk_type", "")

	// Build filters
	filters := make(map[string]string)
	if language != "" {
		filters["language"] = language
	}
	if chunkType != "" {
		filters["chunk_type"] = chunkType
	}

	// Generate embedding for query using embedding service
	queryEmbedding, err := cfs.GenerateQueryEmbedding(ctx, query)
	if err != nil {
		log.Printf("Failed to generate embedding for query: %v", err)
		// Fall back to text-based search instead of meaningless dummy embeddings
		return cfs.handleTextBasedSearch(ctx, query, maxResults, filters)
	}

	// Search using vector database
	results, err := cfs.vectorDB.SearchSimilarChunks(ctx, queryEmbedding, maxResults, filters)
	if err != nil {
		log.Printf("Vector search failed: %v", err)
		// Fall back to text-based search if vector search fails
		return cfs.handleTextBasedSearch(ctx, query, maxResults, filters)
	}

	// Format results
	var resultTexts []string
	for _, result := range results {
		resultText := fmt.Sprintf("Query: %s\n\nFile: %s\nType: %s\nLanguage: %s\nScore: %.3f\n\nContent:\n%s\n---",
			query,
			result.Chunk.FilePath,
			result.Chunk.ChunkType,
			result.Chunk.Language,
			result.Score,
			result.Chunk.Content,
		)
		resultTexts = append(resultTexts, resultText)
	}

	if len(resultTexts) == 0 {
		return mcp.NewToolResultText("No similar code found for the given query."), nil
	}

	return mcp.NewToolResultText(strings.Join(resultTexts, "\n\n")), nil
}

// handleTextBasedSearch performs text-based search as fallback when embeddings fail
func (cfs *CodeForgeServer) handleTextBasedSearch(ctx context.Context, query string, maxResults int, filters map[string]string) (*mcp.CallToolResult, error) {
	log.Printf("Falling back to text-based search for query: %s", query)

	// Use a dummy embedding to get chunks from the database
	// This is a workaround since there's no GetAllChunks method
	dummyEmbedding := make([]float32, 384)
	for i := range dummyEmbedding {
		dummyEmbedding[i] = 0.0 // Use zeros instead of 0.1 to avoid bias
	}

	// Get a large number of chunks to search through
	allResults, err := cfs.vectorDB.SearchSimilarChunks(ctx, dummyEmbedding, 1000, filters)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to retrieve chunks for text search: %v", err)), nil
	}

	// Perform text-based matching
	var matches []textSearchResult
	queryLower := strings.ToLower(query)

	for _, result := range allResults {
		chunk := &result.Chunk

		// Calculate text similarity score
		score := cfs.calculateTextSimilarity(queryLower, chunk.Content)
		if score > 0.1 { // Minimum relevance threshold
			matches = append(matches, textSearchResult{
				Chunk: chunk,
				Score: score,
			})
		}
	}

	// Sort by score (highest first)
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	// Limit results
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	// Format results
	var resultTexts []string
	for _, match := range matches {
		resultText := fmt.Sprintf("Query: %s\n\nFile: %s\nType: %s\nLanguage: %s\nScore: %.3f (text-based)\n\nContent:\n%s\n---",
			query,
			match.Chunk.FilePath,
			match.Chunk.ChunkType.Type,
			match.Chunk.Language,
			match.Score,
			match.Chunk.Content,
		)
		resultTexts = append(resultTexts, resultText)
	}

	if len(resultTexts) == 0 {
		return mcp.NewToolResultText("No matching code found using text-based search. Note: Embedding service is unavailable."), nil
	}

	return mcp.NewToolResultText(strings.Join(resultTexts, "\n\n")), nil
}

// calculateTextSimilarity calculates a simple text similarity score between query and content
func (cfs *CodeForgeServer) calculateTextSimilarity(queryLower, content string) float64 {
	contentLower := strings.ToLower(content)

	// Split query into words
	queryWords := strings.Fields(queryLower)
	if len(queryWords) == 0 {
		return 0.0
	}

	// Count matches
	var matches int
	var totalScore float64

	for _, word := range queryWords {
		if len(word) < 2 { // Skip very short words
			continue
		}

		// Exact word match (highest score)
		if strings.Contains(contentLower, word) {
			matches++
			totalScore += 1.0
		}

		// Partial match (lower score)
		for _, contentWord := range strings.Fields(contentLower) {
			if len(contentWord) >= 3 && strings.Contains(contentWord, word) {
				totalScore += 0.3
				break
			}
		}
	}

	// Calculate final score
	if matches == 0 {
		return 0.0
	}

	// Normalize by query length and add bonus for multiple matches
	baseScore := totalScore / float64(len(queryWords))
	matchRatio := float64(matches) / float64(len(queryWords))

	// Bonus for high match ratio
	if matchRatio > 0.5 {
		baseScore *= 1.5
	}

	// Cap at 1.0
	if baseScore > 1.0 {
		baseScore = 1.0
	}

	return baseScore
}

// handleReadFile handles file reading requests
func (cfs *CodeForgeServer) handleReadFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("path parameter is required"), nil
	}

	// Validate and resolve path
	fullPath, err := cfs.validatePath(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	// Check if file exists
	if !cfs.fileExists(fullPath) {
		return mcp.NewToolResultError(fmt.Sprintf("file not found: %s", path)), nil
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read file: %v", err)), nil
	}

	return mcp.NewToolResultText(string(content)), nil
}

// handleWriteFile handles file writing requests
func (cfs *CodeForgeServer) handleWriteFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("path parameter is required"), nil
	}

	content, err := request.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("content parameter is required"), nil
	}

	// Validate and resolve path
	fullPath, err := cfs.validatePath(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to create directory: %v", err)), nil
	}

	// Write file content
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to write file: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path)), nil
}

// handleCodeAnalysis handles code analysis requests
func (cfs *CodeForgeServer) handleCodeAnalysis(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("path parameter is required"), nil
	}

	// Validate and resolve path
	fullPath, err := cfs.validatePath(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	// Check if file exists
	if !cfs.fileExists(fullPath) {
		return mcp.NewToolResultError(fmt.Sprintf("file not found: %s", path)), nil
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read file: %v", err)), nil
	}

	// Get file info
	info, err := os.Stat(fullPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get file info: %v", err)), nil
	}

	language := detectLanguage(path)

	// Extract symbols using LSP-enhanced symbol extractor
	symbolExtractor := analysis.NewSymbolExtractor()
	symbols, err := symbolExtractor.ExtractSymbols(ctx, fullPath, string(content), language)
	if err != nil {
		// Fallback to basic analysis if symbol extraction fails
		symbols = []vectordb.Symbol{}
	}

	// Use enhanced chunking to analyze code structure
	chunker := chunking.NewCodeChunker(chunking.DefaultConfig())
	chunks, err := chunker.ChunkFile(ctx, fullPath, string(content), language)
	if err != nil {
		chunks = []*vectordb.CodeChunk{}
	}

	// Prepare analysis result
	analysis := map[string]interface{}{
		"file_path":    path,
		"size":         info.Size(),
		"modified":     info.ModTime(),
		"language":     language,
		"line_count":   strings.Count(string(content), "\n") + 1,
		"symbols":      convertSymbolsToMap(symbols),
		"chunks":       convertChunksToMap(chunks),
		"chunk_count":  len(chunks),
		"symbol_count": len(symbols),
	}

	analysisJSON, _ := json.MarshalIndent(analysis, "", "  ")
	return mcp.NewToolResultText(string(analysisJSON)), nil
}

// handleProjectStructure handles project structure requests
func (cfs *CodeForgeServer) handleProjectStructure(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path := request.GetString("path", ".")
	maxDepth := int(request.GetFloat("max_depth", 3))

	// Validate and resolve path
	fullPath, err := cfs.validatePath(path)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	// Build directory tree
	tree, err := cfs.buildDirectoryTree(fullPath, maxDepth)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to build directory tree: %v", err)), nil
	}

	return mcp.NewToolResultText(tree), nil
}

// handleSymbolSearch handles workspace symbol search requests
func (cfs *CodeForgeServer) handleSymbolSearch(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	kindFilter := request.GetString("kind", "")

	// Use symbol extractor to search workspace symbols
	symbolExtractor := analysis.NewSymbolExtractor()
	symbols, err := symbolExtractor.ExtractWorkspaceSymbols(ctx, query)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("symbol search failed: %v", err)), nil
	}

	// Filter by kind if specified
	if kindFilter != "" {
		filteredSymbols := []vectordb.Symbol{}
		for _, symbol := range symbols {
			if strings.EqualFold(symbol.Kind, kindFilter) {
				filteredSymbols = append(filteredSymbols, symbol)
			}
		}
		symbols = filteredSymbols
	}

	// Format results
	result := map[string]interface{}{
		"query":        query,
		"kind_filter":  kindFilter,
		"symbol_count": len(symbols),
		"symbols":      convertSymbolsToMap(symbols),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGetDefinition handles get definition requests
func (cfs *CodeForgeServer) handleGetDefinition(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath, err := request.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError("file_path parameter is required"), nil
	}

	line := int(request.GetFloat("line", 0))
	character := int(request.GetFloat("character", 0))

	if line <= 0 || character <= 0 {
		return mcp.NewToolResultError("line and character must be positive integers"), nil
	}

	// Validate and resolve path
	fullPath, err := cfs.validatePath(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	// Check if file exists
	if !cfs.fileExists(fullPath) {
		return mcp.NewToolResultError(fmt.Sprintf("file not found: %s", filePath)), nil
	}

	// Detect language
	language := detectLanguage(filePath)

	// Use symbol extractor to get definition
	symbolExtractor := analysis.NewSymbolExtractor()
	locations, err := symbolExtractor.GetDefinition(ctx, fullPath, line-1, character-1, language) // Convert to 0-based
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get definition: %v", err)), nil
	}

	// Format results
	result := map[string]interface{}{
		"file_path":        filePath,
		"line":             line,
		"character":        character,
		"definition_count": len(locations),
		"definitions":      convertLocationsToMap(locations),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGetReferences handles get references requests
func (cfs *CodeForgeServer) handleGetReferences(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath, err := request.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError("file_path parameter is required"), nil
	}

	line := int(request.GetFloat("line", 0))
	character := int(request.GetFloat("character", 0))
	includeDeclaration := true
	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if val, ok := args["include_declaration"].(bool); ok {
			includeDeclaration = val
		}
	}

	if line <= 0 || character <= 0 {
		return mcp.NewToolResultError("line and character must be positive integers"), nil
	}

	// Validate and resolve path
	fullPath, err := cfs.validatePath(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid path: %v", err)), nil
	}

	// Check if file exists
	if !cfs.fileExists(fullPath) {
		return mcp.NewToolResultError(fmt.Sprintf("file not found: %s", filePath)), nil
	}

	// Detect language
	language := detectLanguage(filePath)

	// Use symbol extractor to get references
	symbolExtractor := analysis.NewSymbolExtractor()
	locations, err := symbolExtractor.GetReferences(ctx, fullPath, line-1, character-1, language, includeDeclaration) // Convert to 0-based
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get references: %v", err)), nil
	}

	// Format results
	result := map[string]interface{}{
		"file_path":           filePath,
		"line":                line,
		"character":           character,
		"include_declaration": includeDeclaration,
		"reference_count":     len(locations),
		"references":          convertLocationsToMap(locations),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleGitDiff handles git diff requests
func (cfs *CodeForgeServer) handleGitDiff(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	staged := false
	if args, ok := request.Params.Arguments.(map[string]interface{}); ok {
		if val, ok := args["staged"].(bool); ok {
			staged = val
		}
	}

	// Create git repository instance
	repo := git.NewRepository(cfs.workspaceRoot)

	// Check if git is available and this is a git repository
	if !git.IsGitInstalled() {
		return mcp.NewToolResultError("Git is not installed"), nil
	}

	if !repo.IsGitRepository() {
		return mcp.NewToolResultError("Not a git repository"), nil
	}

	// Get git diff
	diffs, err := repo.GetDiff(ctx, staged)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get git diff: %v", err)), nil
	}

	// Format results
	result := map[string]interface{}{
		"staged":     staged,
		"diff_count": len(diffs),
		"diffs":      convertDiffsToMap(diffs),
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(resultJSON)), nil
}

// Resource Handlers

// handleProjectMetadata handles project metadata resource requests
func (cfs *CodeForgeServer) handleProjectMetadata(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	metadata := map[string]interface{}{
		"name":           "CodeForge Project",
		"workspace_root": cfs.workspaceRoot,
		"version":        "0.1.0",
		"description":    "AI-powered code intelligence platform",
	}

	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(metadataJSON),
		},
	}, nil
}

// handleFileResource handles file resource requests
func (cfs *CodeForgeServer) handleFileResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract file path from URI (codeforge://files/{path})
	uri := request.Params.URI
	if !strings.HasPrefix(uri, "codeforge://files/") {
		return nil, fmt.Errorf("invalid file resource URI: %s", uri)
	}

	path := strings.TrimPrefix(uri, "codeforge://files/")

	// Validate and resolve path
	fullPath, err := cfs.validatePath(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %v", err)
	}

	// Check if file exists
	if !cfs.fileExists(fullPath) {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	// Determine MIME type based on file extension
	mimeType := detectMIMEType(path)

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      uri,
			MIMEType: mimeType,
			Text:     string(content),
		},
	}, nil
}

// handleGitStatus handles git status resource requests
func (cfs *CodeForgeServer) handleGitStatus(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Create git repository instance
	repo := git.NewRepository(cfs.workspaceRoot)

	var gitStatus interface{}

	// Check if git is available and this is a git repository
	if !git.IsGitInstalled() {
		gitStatus = map[string]interface{}{
			"error":   "Git is not installed",
			"status":  "unavailable",
			"message": "Git command not found in PATH",
		}
	} else if !repo.IsGitRepository() {
		gitStatus = map[string]interface{}{
			"error":   "Not a git repository",
			"status":  "not_git",
			"message": "The workspace is not a git repository",
		}
	} else {
		// Get actual git status
		status, err := repo.GetStatus(ctx)
		if err != nil {
			gitStatus = map[string]interface{}{
				"error":   fmt.Sprintf("Failed to get git status: %v", err),
				"status":  "error",
				"message": "Could not retrieve git status",
			}
		} else {
			gitStatus = status
		}
	}

	statusJSON, _ := json.MarshalIndent(gitStatus, "", "  ")

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "application/json",
			Text:     string(statusJSON),
		},
	}, nil
}

// Helper functions

// detectLanguage detects programming language from file extension
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js", ".mjs":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".h", ".hpp":
		return "c_header"
	case ".php":
		return "php"
	default:
		return "unknown"
	}
}

// detectMIMEType detects MIME type from file extension
func detectMIMEType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".html", ".htm":
		return "text/html"
	case ".css":
		return "text/css"
	case ".js", ".mjs":
		return "application/javascript"
	case ".md":
		return "text/markdown"
	default:
		return "text/plain"
	}
}

// buildDirectoryTree builds a text representation of directory structure
func (cfs *CodeForgeServer) buildDirectoryTree(rootPath string, maxDepth int) (string, error) {
	var result strings.Builder

	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Calculate depth relative to root
		relPath, _ := filepath.Rel(rootPath, path)
		depth := strings.Count(relPath, string(filepath.Separator))

		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files and directories
		if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create indentation
		indent := strings.Repeat("  ", depth)

		if d.IsDir() {
			result.WriteString(fmt.Sprintf("%s%s/\n", indent, d.Name()))
		} else {
			result.WriteString(fmt.Sprintf("%s%s\n", indent, d.Name()))
		}

		return nil
	})

	return result.String(), err
}

// convertSymbolsToMap converts symbols to a map format for JSON serialization
func convertSymbolsToMap(symbols []vectordb.Symbol) []map[string]interface{} {
	result := make([]map[string]interface{}, len(symbols))
	for i, symbol := range symbols {
		result[i] = map[string]interface{}{
			"name":          symbol.Name,
			"kind":          symbol.Kind,
			"signature":     symbol.Signature,
			"documentation": symbol.Documentation,
			"location": map[string]interface{}{
				"start_line":   symbol.Location.StartLine,
				"end_line":     symbol.Location.EndLine,
				"start_column": symbol.Location.StartColumn,
				"end_column":   symbol.Location.EndColumn,
			},
		}
	}
	return result
}

// convertChunksToMap converts chunks to a map format for JSON serialization
func convertChunksToMap(chunks []*vectordb.CodeChunk) []map[string]interface{} {
	result := make([]map[string]interface{}, len(chunks))
	for i, chunk := range chunks {
		result[i] = map[string]interface{}{
			"id":         chunk.ID,
			"chunk_type": chunk.ChunkType.Type,
			"language":   chunk.Language,
			"location": map[string]interface{}{
				"start_line":   chunk.Location.StartLine,
				"end_line":     chunk.Location.EndLine,
				"start_column": chunk.Location.StartColumn,
				"end_column":   chunk.Location.EndColumn,
			},
			"content_length": len(chunk.Content),
			"symbol_count":   len(chunk.Symbols),
			"import_count":   len(chunk.Imports),
			"metadata":       chunk.Metadata,
		}
	}
	return result
}

// convertLocationsToMap converts source locations to a map format for JSON serialization
func convertLocationsToMap(locations []vectordb.SourceLocation) []map[string]interface{} {
	result := make([]map[string]interface{}, len(locations))
	for i, location := range locations {
		result[i] = map[string]interface{}{
			"start_line":   location.StartLine,
			"end_line":     location.EndLine,
			"start_column": location.StartColumn,
			"end_column":   location.EndColumn,
		}
	}
	return result
}

// convertDiffsToMap converts git diffs to a map format for JSON serialization
func convertDiffsToMap(diffs []git.GitDiff) []map[string]interface{} {
	result := make([]map[string]interface{}, len(diffs))
	for i, diff := range diffs {
		result[i] = map[string]interface{}{
			"file_path":   diff.FilePath,
			"old_path":    diff.OldPath,
			"status":      diff.Status,
			"additions":   diff.Additions,
			"deletions":   diff.Deletions,
			"is_binary":   diff.IsBinary,
			"is_renamed":  diff.IsRenamed,
			"is_new_file": diff.IsNewFile,
			"is_deleted":  diff.IsDeleted,
			"similarity":  diff.Similarity,
			"content":     diff.Content,
		}
	}
	return result
}

// GenerateQueryEmbedding generates an embedding for a search query
func (cfs *CodeForgeServer) GenerateQueryEmbedding(ctx context.Context, query string) ([]float32, error) {
	// For now, use simple hash-based embedding
	// In a future version, we could integrate with the embedding service
	return cfs.generateSimpleEmbedding(query), nil
}

// generateSimpleEmbedding creates a simple hash-based embedding
func (cfs *CodeForgeServer) generateSimpleEmbedding(text string) []float32 {
	// Create a simple hash-based embedding
	embedding := make([]float32, 384) // Standard dimension

	// Use a simple hash function to distribute values
	hash := 0
	for _, char := range text {
		hash = hash*31 + int(char)
	}

	// Distribute the hash across the embedding dimensions
	for i := range embedding {
		embedding[i] = float32((hash+i)%1000)/1000.0 - 0.5 // Normalize to [-0.5, 0.5]
	}

	// Normalize the embedding vector
	var norm float32
	for _, val := range embedding {
		norm += val * val
	}
	norm = float32(1.0 / (1e-8 + float64(norm))) // Avoid division by zero

	for i := range embedding {
		embedding[i] *= norm
	}

	return embedding
}
