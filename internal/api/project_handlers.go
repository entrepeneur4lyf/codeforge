package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
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
	// TODO: Implement actual project structure scanning
	// For now, return a mock structure
	structure := ProjectStructure{
		Name: "CodeForge",
		Type: "directory",
		Path: "/",
		Children: []ProjectStructure{
			{
				Name: "cmd",
				Type: "directory",
				Path: "/cmd",
				Children: []ProjectStructure{
					{
						Name: "codeforge",
						Type: "directory",
						Path: "/cmd/codeforge",
						Children: []ProjectStructure{
							{Name: "main.go", Type: "file", Path: "/cmd/codeforge/main.go", Size: 1024},
						},
					},
				},
			},
			{
				Name: "internal",
				Type: "directory",
				Path: "/internal",
				Children: []ProjectStructure{
					{Name: "api", Type: "directory", Path: "/internal/api"},
					{Name: "chat", Type: "directory", Path: "/internal/chat"},
					{Name: "llm", Type: "directory", Path: "/internal/llm"},
					{Name: "vectordb", Type: "directory", Path: "/internal/vectordb"},
				},
			},
			{Name: "README.md", Type: "file", Path: "/README.md", Size: 15420},
			{Name: "go.mod", Type: "file", Path: "/go.mod", Size: 2048},
		},
	}

	s.writeJSON(w, structure)
}

// handleProjectFiles returns a list of project files
func (s *Server) handleProjectFiles(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	language := r.URL.Query().Get("language")
	fileType := r.URL.Query().Get("type")

	// TODO: Implement actual file scanning
	// For now, return mock files
	files := []FileInfo{
		{
			Path:         "/cmd/codeforge/main.go",
			Name:         "main.go",
			Size:         1024,
			Language:     "go",
			LastModified: "2024-01-15T10:30:00Z",
			LineCount:    45,
		},
		{
			Path:         "/internal/api/server.go",
			Name:         "server.go",
			Size:         8192,
			Language:     "go",
			LastModified: "2024-01-15T14:20:00Z",
			LineCount:    320,
		},
		{
			Path:         "/internal/chat/engine.go",
			Name:         "engine.go",
			Size:         4096,
			Language:     "go",
			LastModified: "2024-01-15T12:15:00Z",
			LineCount:    180,
		},
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

	// TODO: Implement actual search using vector database and text search
	// For now, return mock results
	results := []SearchResult{
		{
			Path:     "/internal/api/server.go",
			Name:     "server.go",
			Language: "go",
			Score:    0.95,
			Matches: []SearchMatch{
				{
					LineNumber: 42,
					Line:       "func (s *Server) Start(port int) error {",
					Column:     15,
					Length:     5,
					Context:    "Server startup function",
				},
			},
		},
		{
			Path:     "/internal/chat/engine.go",
			Name:     "engine.go",
			Language: "go",
			Score:    0.87,
			Matches: []SearchMatch{
				{
					LineNumber: 28,
					Line:       "// Start the chat engine",
					Column:     11,
					Length:     5,
					Context:    "Chat engine initialization",
				},
			},
		},
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
