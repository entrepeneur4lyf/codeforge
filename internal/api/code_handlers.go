package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// CodeAnalysisRequest represents a code analysis request
type CodeAnalysisRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
	FilePath string `json:"file_path,omitempty"`
}

// CodeAnalysisResponse represents a code analysis response
type CodeAnalysisResponse struct {
	Language    string      `json:"language"`
	LineCount   int         `json:"line_count"`
	Complexity  int         `json:"complexity"`
	Functions   []Function  `json:"functions"`
	Classes     []Class     `json:"classes"`
	Imports     []string    `json:"imports"`
	Issues      []CodeIssue `json:"issues"`
	Suggestions []string    `json:"suggestions"`
}

// Function represents a function in the code
type Function struct {
	Name       string   `json:"name"`
	StartLine  int      `json:"start_line"`
	EndLine    int      `json:"end_line"`
	Parameters []string `json:"parameters"`
	ReturnType string   `json:"return_type,omitempty"`
	Complexity int      `json:"complexity"`
}

// Class represents a class/struct in the code
type Class struct {
	Name      string   `json:"name"`
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	Methods   []string `json:"methods"`
	Fields    []string `json:"fields"`
}

// CodeIssue represents a code issue
type CodeIssue struct {
	Type       string `json:"type"` // "error", "warning", "info"
	Message    string `json:"message"`
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	Severity   string `json:"severity"`
	Suggestion string `json:"suggestion,omitempty"`
}

// SymbolRequest represents a symbol extraction request
type SymbolRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
	FilePath string `json:"file_path,omitempty"`
}

// SymbolResponse represents a symbol extraction response
type SymbolResponse struct {
	Symbols []Symbol `json:"symbols"`
	Total   int      `json:"total"`
}

// Symbol represents a code symbol
type Symbol struct {
	Name      string `json:"name"`
	Type      string `json:"type"` // "function", "class", "variable", "constant", "interface"
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Scope     string `json:"scope"` // "public", "private", "protected"
	Signature string `json:"signature,omitempty"`
	DocString string `json:"doc_string,omitempty"`
}

// handleCodeAnalysis analyzes code and returns insights
func (s *Server) handleCodeAnalysis(w http.ResponseWriter, r *http.Request) {
	var req CodeAnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" {
		s.writeError(w, "Code is required", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual code analysis using tree-sitter or similar
	// For now, return mock analysis
	response := CodeAnalysisResponse{
		Language:   req.Language,
		LineCount:  len(strings.Split(req.Code, "\n")),
		Complexity: 5,
		Functions: []Function{
			{
				Name:       "calculateSum",
				StartLine:  1,
				EndLine:    5,
				Parameters: []string{"a int", "b int"},
				ReturnType: "int",
				Complexity: 1,
			},
		},
		Classes: []Class{
			{
				Name:      "Calculator",
				StartLine: 7,
				EndLine:   20,
				Methods:   []string{"Add", "Subtract", "Multiply"},
				Fields:    []string{"result"},
			},
		},
		Imports: []string{"fmt", "math"},
		Issues: []CodeIssue{
			{
				Type:       "warning",
				Message:    "Unused variable 'temp'",
				Line:       15,
				Column:     5,
				Severity:   "low",
				Suggestion: "Remove unused variable or use it",
			},
		},
		Suggestions: []string{
			"Consider adding error handling",
			"Add documentation comments",
			"Use more descriptive variable names",
		},
	}

	s.writeJSON(w, response)
}

// handleCodeSymbols extracts symbols from code
func (s *Server) handleCodeSymbols(w http.ResponseWriter, r *http.Request) {
	var req SymbolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Code == "" {
		s.writeError(w, "Code is required", http.StatusBadRequest)
		return
	}

	// TODO: Implement actual symbol extraction using tree-sitter
	// For now, return mock symbols
	symbols := []Symbol{
		{
			Name:      "main",
			Type:      "function",
			StartLine: 1,
			EndLine:   10,
			Scope:     "public",
			Signature: "func main()",
			DocString: "Main entry point of the application",
		},
		{
			Name:      "Server",
			Type:      "class",
			StartLine: 12,
			EndLine:   50,
			Scope:     "public",
			Signature: "type Server struct",
			DocString: "Server represents the API server",
		},
		{
			Name:      "Start",
			Type:      "function",
			StartLine: 25,
			EndLine:   35,
			Scope:     "public",
			Signature: "func (s *Server) Start(port int) error",
			DocString: "Start starts the server on the specified port",
		},
	}

	response := SymbolResponse{
		Symbols: symbols,
		Total:   len(symbols),
	}

	s.writeJSON(w, response)
}
