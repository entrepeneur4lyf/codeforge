package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/analysis"
	"github.com/entrepeneur4lyf/codeforge/internal/chunking"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
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

	// Implement actual code analysis using existing infrastructure
	ctx := r.Context()

	// Use symbol extractor for analysis
	symbolExtractor := analysis.NewSymbolExtractor()
	symbols, err := symbolExtractor.ExtractSymbols(ctx, "temp."+req.Language, req.Code, req.Language)
	if err != nil {
		// Continue with basic analysis if symbol extraction fails
		symbols = []vectordb.Symbol{}
	}

	// Use chunker for detailed analysis
	chunker := chunking.NewCodeChunker(chunking.DefaultConfig())
	chunks, err := chunker.ChunkFile(ctx, "temp."+req.Language, req.Code, req.Language)
	if err != nil {
		chunks = []*vectordb.CodeChunk{}
	}

	// Analyze the code and build response
	response := s.buildCodeAnalysisResponse(req.Code, req.Language, symbols, chunks)

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

	// Implement actual symbol extraction using existing infrastructure
	ctx := r.Context()

	// Use symbol extractor for analysis
	symbolExtractor := analysis.NewSymbolExtractor()
	extractedSymbols, err := symbolExtractor.ExtractSymbols(ctx, "temp."+req.Language, req.Code, req.Language)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Symbol extraction failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert vectordb.Symbol to API Symbol format
	symbols := s.convertSymbolsToAPIFormat(extractedSymbols)

	response := SymbolResponse{
		Symbols: symbols,
		Total:   len(symbols),
	}

	s.writeJSON(w, response)
}

// buildCodeAnalysisResponse builds a comprehensive code analysis response
func (s *Server) buildCodeAnalysisResponse(code, language string, symbols []vectordb.Symbol, chunks []*vectordb.CodeChunk) CodeAnalysisResponse {
	lines := strings.Split(code, "\n")
	lineCount := len(lines)

	// Extract functions and classes from symbols
	var functions []Function
	var classes []Class
	var imports []string

	for _, symbol := range symbols {
		switch symbol.Kind {
		case "function", "method":
			function := Function{
				Name:       symbol.Name,
				StartLine:  symbol.Location.StartLine,
				EndLine:    symbol.Location.EndLine,
				Parameters: s.extractParametersFromSignature(symbol.Signature),
				ReturnType: s.extractReturnTypeFromSignature(symbol.Signature),
				Complexity: 1, // Basic complexity
			}
			functions = append(functions, function)
		case "class", "struct", "type":
			class := Class{
				Name:      symbol.Name,
				StartLine: symbol.Location.StartLine,
				EndLine:   symbol.Location.EndLine,
				Methods:   []string{}, // Would need more analysis
				Fields:    []string{}, // Would need more analysis
			}
			classes = append(classes, class)
		}
	}

	// Extract imports from chunks
	for _, chunk := range chunks {
		imports = append(imports, chunk.Imports...)
	}

	// Remove duplicates from imports
	imports = s.removeDuplicateStrings(imports)

	// Basic code analysis
	complexity := s.calculateBasicComplexity(code)
	issues := s.analyzeCodeIssues(code, language)
	suggestions := s.generateCodeSuggestions(code, language)

	return CodeAnalysisResponse{
		Language:    language,
		LineCount:   lineCount,
		Complexity:  complexity,
		Functions:   functions,
		Classes:     classes,
		Imports:     imports,
		Issues:      issues,
		Suggestions: suggestions,
	}
}

// convertSymbolsToAPIFormat converts vectordb.Symbol to API Symbol format
func (s *Server) convertSymbolsToAPIFormat(symbols []vectordb.Symbol) []Symbol {
	var apiSymbols []Symbol

	for _, symbol := range symbols {
		apiSymbol := Symbol{
			Name:      symbol.Name,
			Type:      symbol.Kind,
			StartLine: symbol.Location.StartLine,
			EndLine:   symbol.Location.EndLine,
			Scope:     s.determineScope(symbol.Name),
			Signature: symbol.Signature,
			DocString: symbol.Documentation,
		}
		apiSymbols = append(apiSymbols, apiSymbol)
	}

	return apiSymbols
}

// Helper methods for code analysis

// extractParametersFromSignature extracts parameters from a function signature
func (s *Server) extractParametersFromSignature(signature string) []string {
	if signature == "" {
		return []string{}
	}

	// Simple extraction for common patterns
	start := strings.Index(signature, "(")
	end := strings.LastIndex(signature, ")")
	if start == -1 || end == -1 || start >= end {
		return []string{}
	}

	params := strings.TrimSpace(signature[start+1 : end])
	if params == "" {
		return []string{}
	}

	// Split by comma and clean up
	parts := strings.Split(params, ",")
	var result []string
	for _, part := range parts {
		result = append(result, strings.TrimSpace(part))
	}

	return result
}

// extractReturnTypeFromSignature extracts return type from a function signature
func (s *Server) extractReturnTypeFromSignature(signature string) string {
	if signature == "" {
		return ""
	}

	// Simple extraction for Go-style signatures
	end := strings.LastIndex(signature, ")")
	if end == -1 {
		return ""
	}

	remaining := strings.TrimSpace(signature[end+1:])
	if remaining == "" {
		return "void"
	}

	// Remove "error" from return types for simplicity
	if strings.Contains(remaining, "error") {
		remaining = strings.ReplaceAll(remaining, "error", "")
		remaining = strings.TrimSpace(strings.Trim(remaining, ",()"))
	}

	return remaining
}

// removeDuplicateStrings removes duplicate strings from a slice
func (s *Server) removeDuplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] && item != "" {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// calculateBasicComplexity calculates basic cyclomatic complexity
func (s *Server) calculateBasicComplexity(code string) int {
	complexity := 1 // Base complexity

	// Count decision points
	keywords := []string{"if", "else", "for", "while", "switch", "case", "catch", "&&", "||"}
	for _, keyword := range keywords {
		complexity += strings.Count(strings.ToLower(code), keyword)
	}

	return complexity
}

// analyzeCodeIssues performs basic code issue analysis
func (s *Server) analyzeCodeIssues(code, language string) []CodeIssue {
	var issues []CodeIssue
	lines := strings.Split(code, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Check for common issues
		if strings.Contains(line, "TODO") || strings.Contains(line, "FIXME") {
			issues = append(issues, CodeIssue{
				Type:       "info",
				Message:    "TODO or FIXME comment found",
				Line:       i + 1,
				Column:     1,
				Severity:   "low",
				Suggestion: "Consider addressing this TODO/FIXME",
			})
		}

		// Check for long lines
		if len(line) > 120 {
			issues = append(issues, CodeIssue{
				Type:       "style",
				Message:    "Line too long",
				Line:       i + 1,
				Column:     120,
				Severity:   "low",
				Suggestion: "Consider breaking this line",
			})
		}
	}

	return issues
}

// generateCodeSuggestions generates code improvement suggestions
func (s *Server) generateCodeSuggestions(code, language string) []string {
	var suggestions []string

	// Basic suggestions based on code patterns
	if !strings.Contains(code, "error") && language == "go" {
		suggestions = append(suggestions, "Consider adding error handling")
	}

	if !strings.Contains(code, "//") && !strings.Contains(code, "/*") {
		suggestions = append(suggestions, "Add documentation comments")
	}

	if strings.Count(code, "\n") > 50 {
		suggestions = append(suggestions, "Consider breaking this into smaller functions")
	}

	return suggestions
}

// determineScope determines the scope of a symbol based on naming conventions
func (s *Server) determineScope(name string) string {
	if name == "" {
		return "unknown"
	}

	// Simple heuristic based on naming conventions
	if strings.ToUpper(name[:1]) == name[:1] {
		return "public"
	}

	return "private"
}
