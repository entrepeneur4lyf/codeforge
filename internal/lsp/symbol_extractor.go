package lsp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
)

// SymbolExtractor provides symbol extraction functionality for context building
type SymbolExtractor struct {
	manager *Manager
}

// NewSymbolExtractor creates a new symbol extractor
func NewSymbolExtractor(manager *Manager) *SymbolExtractor {
	return &SymbolExtractor{
		manager: manager,
	}
}

// ExtractedSymbol represents a simplified symbol for context
type ExtractedSymbol struct {
	Name         string   `json:"name"`
	Kind         string   `json:"kind"`
	Location     string   `json:"location"`
	Detail       string   `json:"detail,omitempty"`
	Documentation string  `json:"documentation,omitempty"`
	Children     []string `json:"children,omitempty"`
}

// GetRelevantSymbols extracts symbols relevant to a query
func (se *SymbolExtractor) GetRelevantSymbols(ctx context.Context, query string, files []string, maxSymbols int) ([]ExtractedSymbol, error) {
	if se.manager == nil {
		return nil, fmt.Errorf("symbol extractor not initialized")
	}

	var allSymbols []ExtractedSymbol
	
	// First, try workspace-wide symbol search if query is provided
	if query != "" {
		workspaceSymbols, err := se.searchWorkspaceSymbols(ctx, query)
		if err == nil {
			allSymbols = append(allSymbols, workspaceSymbols...)
		}
	}
	
	// Then get symbols from specific files
	for _, file := range files {
		fileSymbols, err := se.getFileSymbols(ctx, file)
		if err != nil {
			continue // Skip files that fail
		}
		allSymbols = append(allSymbols, fileSymbols...)
		
		if len(allSymbols) >= maxSymbols {
			break
		}
	}
	
	// Filter and rank symbols by relevance
	relevantSymbols := se.filterRelevantSymbols(allSymbols, query, maxSymbols)
	
	return relevantSymbols, nil
}

// searchWorkspaceSymbols searches for symbols across the workspace
func (se *SymbolExtractor) searchWorkspaceSymbols(ctx context.Context, query string) ([]ExtractedSymbol, error) {
	// Get appropriate LSP client based on workspace
	clients := se.manager.GetActiveClients()
	if len(clients) == 0 {
		return nil, fmt.Errorf("no active LSP clients")
	}
	
	var symbols []ExtractedSymbol
	
	// Use the first available client for workspace search
	for _, client := range clients {
		wsSymbols, err := client.GetWorkspaceSymbols(ctx, query)
		if err != nil {
			continue
		}
		
		for _, sym := range wsSymbols {
			symbols = append(symbols, ExtractedSymbol{
				Name:     sym.Name,
				Kind:     symbolKindToString(sym.Kind),
				Location: uriToFilePath(string(sym.Location.URI)),
			})
		}
		
		if len(symbols) > 0 {
			break // Found some symbols, stop searching
		}
	}
	
	return symbols, nil
}

// getFileSymbols gets symbols from a specific file
func (se *SymbolExtractor) getFileSymbols(ctx context.Context, filePath string) ([]ExtractedSymbol, error) {
	// Detect language and get appropriate client
	languageID := DetectLanguageID(filePath)
	client := se.manager.GetClientForLanguage(languageID)
	if client == nil {
		return nil, fmt.Errorf("no LSP client for language: %s", languageID)
	}
	
	// Ensure file is open in LSP
	if err := client.OpenFile(ctx, filePath); err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	// Get document symbols
	docSymbols, err := client.GetDocumentSymbols(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get document symbols: %w", err)
	}
	
	// Convert to extracted symbols
	var symbols []ExtractedSymbol
	for _, sym := range docSymbols {
		symbols = append(symbols, se.convertDocumentSymbol(sym, filePath)...)
	}
	
	return symbols, nil
}

// convertDocumentSymbol converts LSP document symbols to extracted symbols
func (se *SymbolExtractor) convertDocumentSymbol(sym protocol.DocumentSymbol, filePath string) []ExtractedSymbol {
	var symbols []ExtractedSymbol
	
	// Create main symbol
	extracted := ExtractedSymbol{
		Name:     sym.Name,
		Kind:     symbolKindToString(sym.Kind),
		Location: fmt.Sprintf("%s:%d", filepath.Base(filePath), sym.Range.Start.Line+1),
		Detail:   sym.Detail,
	}
	
	// Add children names if any
	if len(sym.Children) > 0 {
		for _, child := range sym.Children {
			extracted.Children = append(extracted.Children, child.Name)
			// Recursively add child symbols
			symbols = append(symbols, se.convertDocumentSymbol(child, filePath)...)
		}
	}
	
	symbols = append([]ExtractedSymbol{extracted}, symbols...)
	return symbols
}

// filterRelevantSymbols filters and ranks symbols by relevance
func (se *SymbolExtractor) filterRelevantSymbols(symbols []ExtractedSymbol, query string, maxSymbols int) []ExtractedSymbol {
	if len(symbols) <= maxSymbols {
		return symbols
	}
	
	// Simple relevance scoring based on query match
	queryLower := strings.ToLower(query)
	
	type scoredSymbol struct {
		symbol ExtractedSymbol
		score  int
	}
	
	var scored []scoredSymbol
	for _, sym := range symbols {
		score := 0
		nameLower := strings.ToLower(sym.Name)
		
		// Exact match
		if nameLower == queryLower {
			score += 100
		} else if strings.Contains(nameLower, queryLower) {
			score += 50
		}
		
		// Prefer certain symbol kinds
		switch sym.Kind {
		case "Class", "Interface", "Function", "Method":
			score += 20
		case "Variable", "Constant":
			score += 10
		}
		
		scored = append(scored, scoredSymbol{symbol: sym, score: score})
	}
	
	// Sort by score (simple bubble sort for now)
	for i := 0; i < len(scored)-1; i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}
	
	// Return top symbols
	result := make([]ExtractedSymbol, 0, maxSymbols)
	for i := 0; i < maxSymbols && i < len(scored); i++ {
		result = append(result, scored[i].symbol)
	}
	
	return result
}

// symbolKindToString converts protocol.SymbolKind to string
func symbolKindToString(kind protocol.SymbolKind) string {
	switch kind {
	case protocol.SymbolKindFile:
		return "File"
	case protocol.SymbolKindModule:
		return "Module"
	case protocol.SymbolKindNamespace:
		return "Namespace"
	case protocol.SymbolKindPackage:
		return "Package"
	case protocol.SymbolKindClass:
		return "Class"
	case protocol.SymbolKindMethod:
		return "Method"
	case protocol.SymbolKindProperty:
		return "Property"
	case protocol.SymbolKindField:
		return "Field"
	case protocol.SymbolKindConstructor:
		return "Constructor"
	case protocol.SymbolKindEnum:
		return "Enum"
	case protocol.SymbolKindInterface:
		return "Interface"
	case protocol.SymbolKindFunction:
		return "Function"
	case protocol.SymbolKindVariable:
		return "Variable"
	case protocol.SymbolKindConstant:
		return "Constant"
	case protocol.SymbolKindString:
		return "String"
	case protocol.SymbolKindNumber:
		return "Number"
	case protocol.SymbolKindBoolean:
		return "Boolean"
	case protocol.SymbolKindArray:
		return "Array"
	case protocol.SymbolKindObject:
		return "Object"
	case protocol.SymbolKindKey:
		return "Key"
	case protocol.SymbolKindNull:
		return "Null"
	case protocol.SymbolKindEnumMember:
		return "EnumMember"
	case protocol.SymbolKindStruct:
		return "Struct"
	case protocol.SymbolKindEvent:
		return "Event"
	case protocol.SymbolKindOperator:
		return "Operator"
	case protocol.SymbolKindTypeParameter:
		return "TypeParameter"
	default:
		return "Unknown"
	}
}

// uriToFilePath converts file URI to file path
func uriToFilePath(uri string) string {
	if strings.HasPrefix(uri, "file://") {
		return uri[7:]
	}
	return uri
}