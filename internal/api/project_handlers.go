package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/embeddings"
)

// ProjectStructure represents the project file structure
type ProjectStructure struct {
	Name     string             `json:"name"`
	Type     string             `json:"type"` // "file" or "directory"
	Path     string             `json:"path"`
	Size     int64              `json:"size,omitempty"`
	Children []ProjectStructure `json:"children,omitempty"`
}

// FileInfo represents file information
type FileInfo struct {
	Path         string `json:"path"`
	Name         string `json:"name"`
	Size         int64  `json:"size"`
	Language     string `json:"language"`
	LastModified string `json:"last_modified"`
	LineCount    int    `json:"line_count,omitempty"`
}

// SearchRequest represents a project search request
type SearchRequest struct {
	Query       string   `json:"query"`
	FileTypes   []string `json:"file_types,omitempty"`
	Languages   []string `json:"languages,omitempty"`
	MaxResults  int      `json:"max_results,omitempty"`
	SearchType  string   `json:"search_type"` // "semantic", "text", "symbol"
	IncludeCode bool     `json:"include_code"`
}

// SearchResult represents a search result
type SearchResult struct {
	Path     string                 `json:"path"`
	Name     string                 `json:"name"`
	Language string                 `json:"language"`
	Score    float64                `json:"score"`
	Matches  []SearchMatch          `json:"matches"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// SearchMatch represents a match within a file
type SearchMatch struct {
	LineNumber int    `json:"line_number"`
	Line       string `json:"line"`
	Column     int    `json:"column,omitempty"`
	Length     int    `json:"length,omitempty"`
	Context    string `json:"context,omitempty"`
}

// handleProjectStructure returns the project file structure
func (s *Server) handleProjectStructure(w http.ResponseWriter, r *http.Request) {
	// Implement actual project structure scanning using filesystem
	workingDir := "."
	if s.config != nil && s.config.WorkingDir != "" {
		workingDir = s.config.WorkingDir
	}

	// Build directory tree with limited depth to avoid performance issues
	maxDepth := 3
	if depthStr := r.URL.Query().Get("depth"); depthStr != "" {
		if depth, err := strconv.Atoi(depthStr); err == nil && depth > 0 && depth <= 10 {
			maxDepth = depth
		}
	}

	structure, err := s.buildProjectStructure(workingDir, maxDepth)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to scan project structure: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSON(w, structure)
}

// buildProjectStructure recursively builds the project directory structure
func (s *Server) buildProjectStructure(rootPath string, maxDepth int) (ProjectStructure, error) {
	return s.buildProjectStructureRecursive(rootPath, "", 0, maxDepth)
}

// buildProjectStructureRecursive recursively scans directories
func (s *Server) buildProjectStructureRecursive(rootPath, relativePath string, currentDepth, maxDepth int) (ProjectStructure, error) {
	if currentDepth > maxDepth {
		return ProjectStructure{}, nil
	}

	fullPath := filepath.Join(rootPath, relativePath)
	info, err := os.Stat(fullPath)
	if err != nil {
		return ProjectStructure{}, err
	}

	name := info.Name()
	if relativePath == "" {
		name = filepath.Base(rootPath)
	}

	structure := ProjectStructure{
		Name: name,
		Path: "/" + strings.ReplaceAll(relativePath, "\\", "/"),
		Size: info.Size(),
	}

	if info.IsDir() {
		structure.Type = "directory"

		// Skip hidden directories and common ignore patterns
		if s.shouldIgnoreDirectory(name) {
			return structure, nil
		}

		// Read directory contents
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return structure, err
		}

		for _, entry := range entries {
			childRelPath := filepath.Join(relativePath, entry.Name())
			child, err := s.buildProjectStructureRecursive(rootPath, childRelPath, currentDepth+1, maxDepth)
			if err != nil {
				continue // Skip files with errors
			}

			// Only add non-empty structures
			if child.Name != "" {
				structure.Children = append(structure.Children, child)
			}
		}
	} else {
		structure.Type = "file"

		// Skip hidden files and common ignore patterns
		if s.shouldIgnoreFile(name) {
			return ProjectStructure{}, nil
		}
	}

	return structure, nil
}

// shouldIgnoreDirectory checks if a directory should be ignored
func (s *Server) shouldIgnoreDirectory(dirPath string) bool {
	// Use gitignore filter if available
	if s.gitignoreFilter != nil {
		if s.gitignoreFilter.IsIgnored(dirPath) {
			return true
		}
	}

	// Fallback to common ignore patterns
	name := filepath.Base(dirPath)
	ignoreDirs := []string{
		".git", ".svn", ".hg",
		"node_modules", "vendor", "target",
		".vscode", ".idea",
		"__pycache__", ".pytest_cache",
		"dist", "build", "out",
	}

	for _, ignore := range ignoreDirs {
		if name == ignore {
			return true
		}
	}

	return strings.HasPrefix(name, ".")
}

// shouldIgnoreFile checks if a file should be ignored
func (s *Server) shouldIgnoreFile(filePath string) bool {
	// Use gitignore filter if available
	if s.gitignoreFilter != nil {
		if s.gitignoreFilter.IsIgnored(filePath) {
			return true
		}
	}

	// Fallback to common ignore patterns
	name := filepath.Base(filePath)
	ignoreFiles := []string{
		".DS_Store", "Thumbs.db",
		".gitignore", ".gitkeep",
	}

	for _, ignore := range ignoreFiles {
		if name == ignore {
			return true
		}
	}

	// Ignore hidden files
	if strings.HasPrefix(name, ".") {
		return true
	}

	// Ignore binary files by extension
	ext := strings.ToLower(filepath.Ext(name))
	binaryExts := []string{
		".exe", ".dll", ".so", ".dylib",
		".jpg", ".jpeg", ".png", ".gif", ".bmp",
		".mp3", ".mp4", ".avi", ".mov",
		".zip", ".tar", ".gz", ".rar",
		".pdf", ".doc", ".docx",
	}

	for _, binExt := range binaryExts {
		if ext == binExt {
			return true
		}
	}

	return false
}

// scanProjectFiles scans the project directory for files
func (s *Server) scanProjectFiles(rootPath, languageFilter, typeFilter string) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		// Skip directories
		if info.IsDir() {
			// Skip ignored directories
			if s.shouldIgnoreDirectory(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip ignored files
		if s.shouldIgnoreFile(info.Name()) {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			return nil
		}

		// Normalize path separators
		relPath = "/" + strings.ReplaceAll(relPath, "\\", "/")

		// Detect language
		language := s.detectLanguage(path)

		// Apply language filter
		if languageFilter != "" && language != languageFilter {
			return nil
		}

		// Apply type filter
		if typeFilter != "" {
			ext := strings.TrimPrefix(filepath.Ext(path), ".")
			if ext != typeFilter {
				return nil
			}
		}

		// Count lines in text files
		lineCount := s.countLines(path)

		file := FileInfo{
			Path:         relPath,
			Name:         info.Name(),
			Size:         info.Size(),
			Language:     language,
			LastModified: info.ModTime().Format("2006-01-02T15:04:05Z"),
			LineCount:    lineCount,
		}

		files = append(files, file)
		return nil
	})

	return files, err
}

// detectLanguage detects the programming language from file extension
func (s *Server) detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".go":   "go",
		".js":   "javascript",
		".ts":   "typescript",
		".py":   "python",
		".java": "java",
		".cpp":  "cpp",
		".c":    "c",
		".h":    "c",
		".hpp":  "cpp",
		".rs":   "rust",
		".php":  "php",
		".rb":   "ruby",
		".sh":   "shell",
		".bash": "shell",
		".zsh":  "shell",
		".fish": "shell",
		".ps1":  "powershell",
		".sql":  "sql",
		".html": "html",
		".css":  "css",
		".scss": "scss",
		".sass": "sass",
		".less": "less",
		".xml":  "xml",
		".json": "json",
		".yaml": "yaml",
		".yml":  "yaml",
		".toml": "toml",
		".md":   "markdown",
		".txt":  "text",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "unknown"
}

// countLines counts the number of lines in a text file
func (s *Server) countLines(filePath string) int {
	// Only count lines for text files to avoid reading large binary files
	if s.shouldIgnoreFile(filepath.Base(filePath)) {
		return 0
	}

	// Check if it's a text file by extension
	ext := strings.ToLower(filepath.Ext(filePath))
	textExts := []string{
		".go", ".js", ".ts", ".py", ".java", ".cpp", ".c", ".h", ".hpp",
		".rs", ".php", ".rb", ".sh", ".bash", ".zsh", ".fish", ".ps1",
		".sql", ".html", ".css", ".scss", ".sass", ".less", ".xml",
		".json", ".yaml", ".yml", ".toml", ".md", ".txt",
	}

	isText := false
	for _, textExt := range textExts {
		if ext == textExt {
			isText = true
			break
		}
	}

	if !isText {
		return 0
	}

	file, err := os.Open(filePath)
	if err != nil {
		return 0
	}
	defer file.Close()

	// Read file and count lines
	content, err := os.ReadFile(filePath)
	if err != nil {
		return 0
	}

	// Count newlines
	lines := strings.Count(string(content), "\n")
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		lines++ // Add 1 if file doesn't end with newline
	}

	return lines
}

// handleProjectFiles returns a list of project files
func (s *Server) handleProjectFiles(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	language := r.URL.Query().Get("language")
	fileType := r.URL.Query().Get("type")

	// Implement actual file scanning
	workingDir := "."
	if s.config != nil && s.config.WorkingDir != "" {
		workingDir = s.config.WorkingDir
	}

	files, err := s.scanProjectFiles(workingDir, language, fileType)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to scan project files: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter by language if specified
	if language != "" {
		var filtered []FileInfo
		for _, file := range files {
			if file.Language == language {
				filtered = append(filtered, file)
			}
		}
		files = filtered
	}

	// Filter by file type if specified
	if fileType != "" {
		var filtered []FileInfo
		for _, file := range files {
			ext := strings.TrimPrefix(filepath.Ext(file.Path), ".")
			if ext == fileType {
				filtered = append(filtered, file)
			}
		}
		files = filtered
	}

	s.writeJSON(w, map[string]interface{}{
		"files": files,
		"total": len(files),
	})
}

// handleProjectSearch handles project-wide search
func (s *Server) handleProjectSearch(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set defaults
	if req.MaxResults == 0 {
		req.MaxResults = 50
	}
	if req.SearchType == "" {
		req.SearchType = "text"
	}

	// Implement actual search using vector database and text search
	var results []SearchResult
	var err error

	switch req.SearchType {
	case "vector", "semantic":
		results, err = s.performVectorSearch(req)
	case "text", "literal":
		results, err = s.performTextSearch(req)
	default:
		// Hybrid search: combine vector and text search
		vectorResults, _ := s.performVectorSearch(req)
		textResults, _ := s.performTextSearch(req)
		results = s.combineSearchResults(vectorResults, textResults, req.MaxResults)
	}

	if err != nil {
		s.writeError(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter results based on search criteria
	if len(req.Languages) > 0 {
		var filtered []SearchResult
		for _, result := range results {
			for _, lang := range req.Languages {
				if result.Language == lang {
					filtered = append(filtered, result)
					break
				}
			}
		}
		results = filtered
	}

	// Limit results
	if len(results) > req.MaxResults {
		results = results[:req.MaxResults]
	}

	response := map[string]interface{}{
		"results":     results,
		"total":       len(results),
		"search_type": req.SearchType,
		"query":       req.Query,
	}

	s.writeJSON(w, response)
}

// performVectorSearch performs semantic search using the vector database
func (s *Server) performVectorSearch(req SearchRequest) ([]SearchResult, error) {
	if s.vectorDB == nil {
		return []SearchResult{}, nil
	}

	// Generate embedding for the search query
	ctx := context.Background()
	queryEmbedding, err := s.generateQueryEmbedding(ctx, req.Query)
	if err != nil {
		return []SearchResult{}, err
	}

	// Prepare filters
	filters := make(map[string]string)
	if len(req.Languages) > 0 {
		filters["language"] = req.Languages[0] // Use first language for now
	}

	// Search vector database
	vectorResults, err := s.vectorDB.SearchSimilarChunks(ctx, queryEmbedding, req.MaxResults, filters)
	if err != nil {
		return []SearchResult{}, err
	}

	// Convert vector results to search results
	var results []SearchResult
	for _, vr := range vectorResults {
		result := SearchResult{
			Path:     vr.Chunk.FilePath,
			Name:     filepath.Base(vr.Chunk.FilePath),
			Language: vr.Chunk.Language,
			Score:    float64(vr.Score),
			Matches: []SearchMatch{
				{
					LineNumber: vr.Chunk.Location.StartLine,
					Line:       vr.Chunk.Content,
					Column:     0,
					Length:     len(req.Query),
					Context:    "Vector search match",
				},
			},
		}
		results = append(results, result)
	}

	return results, nil
}

// performTextSearch performs literal text search in files
func (s *Server) performTextSearch(req SearchRequest) ([]SearchResult, error) {
	workingDir := "."
	if s.config != nil && s.config.WorkingDir != "" {
		workingDir = s.config.WorkingDir
	}

	var results []SearchResult
	query := strings.ToLower(req.Query)

	err := filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Skip directories and ignored files
		if info.IsDir() {
			if s.shouldIgnoreDirectory(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if s.shouldIgnoreFile(info.Name()) {
			return nil
		}

		// Check language filter
		language := s.detectLanguage(path)
		if len(req.Languages) > 0 {
			found := false
			for _, lang := range req.Languages {
				if language == lang {
					found = true
					break
				}
			}
			if !found {
				return nil
			}
		}

		// Search in file content
		matches, err := s.searchInFile(path, query)
		if err != nil || len(matches) == 0 {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(workingDir, path)
		if err != nil {
			return nil
		}
		relPath = "/" + strings.ReplaceAll(relPath, "\\", "/")

		result := SearchResult{
			Path:     relPath,
			Name:     info.Name(),
			Language: language,
			Score:    float64(len(matches)) / 10.0, // Simple scoring
			Matches:  matches,
		}

		results = append(results, result)
		return nil
	})

	return results, err
}

// combineSearchResults combines vector and text search results
func (s *Server) combineSearchResults(vectorResults, textResults []SearchResult, maxResults int) []SearchResult {
	// Simple combination: merge and deduplicate by path
	resultMap := make(map[string]SearchResult)

	// Add vector results first (higher priority)
	for _, result := range vectorResults {
		resultMap[result.Path] = result
	}

	// Add text results if not already present
	for _, result := range textResults {
		if _, exists := resultMap[result.Path]; !exists {
			resultMap[result.Path] = result
		}
	}

	// Convert back to slice and limit
	var combined []SearchResult
	for _, result := range resultMap {
		combined = append(combined, result)
		if len(combined) >= maxResults {
			break
		}
	}

	return combined
}

// generateQueryEmbedding generates an embedding for the search query
func (s *Server) generateQueryEmbedding(ctx context.Context, query string) ([]float32, error) {
	// Use the embeddings service to generate embedding
	embeddingService := embeddings.Get()
	if embeddingService == nil {
		// Fallback to dummy embedding
		embedding := make([]float32, 384)
		for i := range embedding {
			embedding[i] = 0.1
		}
		return embedding, nil
	}

	return embeddings.GetEmbedding(ctx, query)
}

// searchInFile searches for a query string in a file and returns matches
func (s *Server) searchInFile(filePath, query string) ([]SearchMatch, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []SearchMatch
	scanner := bufio.NewScanner(file)
	lineNumber := 1

	for scanner.Scan() {
		line := scanner.Text()
		lowerLine := strings.ToLower(line)

		// Find all occurrences of the query in this line
		index := 0
		for {
			pos := strings.Index(lowerLine[index:], query)
			if pos == -1 {
				break
			}

			actualPos := index + pos
			match := SearchMatch{
				LineNumber: lineNumber,
				Line:       line,
				Column:     actualPos,
				Length:     len(query),
				Context:    fmt.Sprintf("Line %d", lineNumber),
			}
			matches = append(matches, match)

			index = actualPos + 1
		}

		lineNumber++
	}

	return matches, scanner.Err()
}
