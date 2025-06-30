package analysis

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/entrepeneur4lyf/codeforge/internal/lsp"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
	"go.lsp.dev/protocol"
)

// SymbolExtractor provides enhanced symbol extraction using LSP
type SymbolExtractor struct {
	lspManager      *lsp.Manager
	mu              sync.RWMutex
	openDocuments   map[string]bool
	sourceContent   string // Current source content for documentation extraction
	currentLanguage string // Current language for optimization
}

// NewSymbolExtractor creates a new symbol extractor
func NewSymbolExtractor() *SymbolExtractor {
	return &SymbolExtractor{
		lspManager:    lsp.GetManager(),
		openDocuments: make(map[string]bool),
	}
}

// ExtractSymbols extracts symbols from a file using LSP
func (se *SymbolExtractor) ExtractSymbols(ctx context.Context, filePath, content, language string) ([]vectordb.Symbol, error) {
	// Store source content for documentation extraction
	se.sourceContent = content

	if se.lspManager == nil {
		return se.fallbackSymbolExtraction(content, language), nil
	}

	// Get LSP client for the language
	client := se.lspManager.GetClientForLanguage(language)
	if client == nil || client.GetState() != lsp.StateReady {
		return se.fallbackSymbolExtraction(content, language), nil
	}

	// Open the document in LSP if not already open
	if err := se.ensureDocumentOpen(ctx, client, filePath, content, language); err != nil {
		return se.fallbackSymbolExtraction(content, language), nil
	}

	// Get document symbols from LSP
	symbols, err := client.GetDocumentSymbols(ctx, filePath)
	if err != nil {
		return se.fallbackSymbolExtraction(content, language), nil
	}

	// Convert LSP symbols to our format
	return se.convertLSPSymbols(symbols), nil
}

// ExtractWorkspaceSymbols extracts symbols from the entire workspace
func (se *SymbolExtractor) ExtractWorkspaceSymbols(ctx context.Context, query string) ([]vectordb.Symbol, error) {
	if se.lspManager == nil {
		return []vectordb.Symbol{}, nil
	}

	allSymbols := []vectordb.Symbol{}

	// Get symbols from all available LSP clients
	if se.lspManager != nil {
		clients := se.lspManager.GetAllClients()
		for _, client := range clients {
			if client != nil && client.GetState() == lsp.StateReady {
				symbols, err := client.GetWorkspaceSymbols(ctx, query)
				if err != nil {
					continue // Skip failed clients
				}

				// Convert and add symbols
				for _, symbol := range symbols {
					convertedSymbol := se.convertLSPSymbolInfo(symbol)
					allSymbols = append(allSymbols, convertedSymbol)
				}
				break // Use first available client
			}
		}
	}

	return allSymbols, nil
}

// GetDefinition gets the definition of a symbol at a specific position
func (se *SymbolExtractor) GetDefinition(ctx context.Context, filePath string, line, character int, language string) ([]vectordb.SourceLocation, error) {
	if se.lspManager == nil {
		return []vectordb.SourceLocation{}, fmt.Errorf("LSP manager not available")
	}

	client := se.lspManager.GetClientForLanguage(language)
	if client == nil || client.GetState() != lsp.StateReady {
		return []vectordb.SourceLocation{}, fmt.Errorf("LSP client not available for language: %s", language)
	}

	locations, err := client.GetDefinition(ctx, filePath, line, character)
	if err != nil {
		return []vectordb.SourceLocation{}, err
	}

	// Convert LSP locations to our format
	result := make([]vectordb.SourceLocation, len(locations))
	for i, loc := range locations {
		result[i] = se.convertLSPLocation(loc)
	}

	return result, nil
}

// GetReferences gets all references to a symbol at a specific position
func (se *SymbolExtractor) GetReferences(ctx context.Context, filePath string, line, character int, language string, includeDeclaration bool) ([]vectordb.SourceLocation, error) {
	if se.lspManager == nil {
		return []vectordb.SourceLocation{}, fmt.Errorf("LSP manager not available")
	}

	client := se.lspManager.GetClientForLanguage(language)
	if client == nil || client.GetState() != lsp.StateReady {
		return []vectordb.SourceLocation{}, fmt.Errorf("LSP client not available for language: %s", language)
	}

	locations, err := client.GetReferences(ctx, filePath, line, character, includeDeclaration)
	if err != nil {
		return []vectordb.SourceLocation{}, err
	}

	// Convert LSP locations to our format
	result := make([]vectordb.SourceLocation, len(locations))
	for i, loc := range locations {
		result[i] = se.convertLSPLocation(loc)
	}

	return result, nil
}

// GetHover gets hover information for a symbol at a specific position
func (se *SymbolExtractor) GetHover(ctx context.Context, filePath string, line, character int, language string) (string, error) {
	if se.lspManager == nil {
		return "", fmt.Errorf("LSP manager not available")
	}

	client := se.lspManager.GetClientForLanguage(language)
	if client == nil || client.GetState() != lsp.StateReady {
		return "", fmt.Errorf("LSP client not available for language: %s", language)
	}

	hover, err := client.GetHover(ctx, filePath, line, character)
	if err != nil {
		return "", err
	}

	if hover == nil {
		return "", nil
	}

	// Extract text from hover contents
	return se.extractHoverText(hover.Contents), nil
}

// ensureDocumentOpen ensures a document is open in the LSP client
func (se *SymbolExtractor) ensureDocumentOpen(ctx context.Context, client *lsp.Client, filePath, content, language string) error {
	// Check if document is already tracked
	if se.isDocumentOpen(filePath) {
		return nil
	}

	// Store content for potential fallback symbol extraction
	se.sourceContent = content

	// Use language information for better symbol extraction
	if language != "" {
		// Store language for potential fallback extraction optimization
		se.currentLanguage = language
	}

	// Open the document in the LSP client
	err := client.OpenFile(ctx, filePath)
	if err != nil {
		return fmt.Errorf("failed to open document %s: %w", filePath, err)
	}

	// Track the opened document
	se.trackDocument(filePath)

	return nil
}

// isDocumentOpen checks if a document is already tracked as open
func (se *SymbolExtractor) isDocumentOpen(filePath string) bool {
	se.mu.RLock()
	defer se.mu.RUnlock()

	if se.openDocuments == nil {
		se.openDocuments = make(map[string]bool)
	}

	return se.openDocuments[filePath]
}

// trackDocument adds a document to the tracking list
func (se *SymbolExtractor) trackDocument(filePath string) {
	se.mu.Lock()
	defer se.mu.Unlock()

	if se.openDocuments == nil {
		se.openDocuments = make(map[string]bool)
	}

	se.openDocuments[filePath] = true
}

// convertLSPSymbols converts LSP DocumentSymbols to our Symbol format
func (se *SymbolExtractor) convertLSPSymbols(lspSymbols []protocol.DocumentSymbol) []vectordb.Symbol {
	symbols := []vectordb.Symbol{}

	for _, lspSymbol := range lspSymbols {
		symbol := vectordb.Symbol{
			Name:      lspSymbol.Name,
			Kind:      se.convertSymbolKind(lspSymbol.Kind),
			Signature: lspSymbol.Detail,
			Location: vectordb.SourceLocation{
				StartLine:   int(lspSymbol.Range.Start.Line) + 1, // LSP is 0-based, we use 1-based
				EndLine:     int(lspSymbol.Range.End.Line) + 1,
				StartColumn: int(lspSymbol.Range.Start.Character) + 1,
				EndColumn:   int(lspSymbol.Range.End.Character) + 1,
			},
		}

		// Extract documentation from source code
		if se.sourceContent != "" {
			symbol.Documentation = se.extractSourceDocumentation(lspSymbol, se.sourceContent)
		}

		symbols = append(symbols, symbol)

		// Recursively process child symbols
		if len(lspSymbol.Children) > 0 {
			childSymbols := se.convertLSPSymbols(lspSymbol.Children)
			symbols = append(symbols, childSymbols...)
		}
	}

	return symbols
}

// convertLSPSymbolInfo converts LSP SymbolInformation to our Symbol format
func (se *SymbolExtractor) convertLSPSymbolInfo(lspSymbol protocol.SymbolInformation) vectordb.Symbol {
	return vectordb.Symbol{
		Name: lspSymbol.Name,
		Kind: se.convertSymbolKind(lspSymbol.Kind),
		Location: vectordb.SourceLocation{
			StartLine:   int(lspSymbol.Location.Range.Start.Line) + 1,
			EndLine:     int(lspSymbol.Location.Range.End.Line) + 1,
			StartColumn: int(lspSymbol.Location.Range.Start.Character) + 1,
			EndColumn:   int(lspSymbol.Location.Range.End.Character) + 1,
		},
	}
}

// convertLSPLocation converts LSP Location to our SourceLocation format
func (se *SymbolExtractor) convertLSPLocation(lspLocation protocol.Location) vectordb.SourceLocation {
	return vectordb.SourceLocation{
		StartLine:   int(lspLocation.Range.Start.Line) + 1,
		EndLine:     int(lspLocation.Range.End.Line) + 1,
		StartColumn: int(lspLocation.Range.Start.Character) + 1,
		EndColumn:   int(lspLocation.Range.End.Character) + 1,
	}
}

// convertSymbolKind converts LSP SymbolKind to our string representation
func (se *SymbolExtractor) convertSymbolKind(kind protocol.SymbolKind) string {
	switch kind {
	case protocol.SymbolKindFile:
		return "file"
	case protocol.SymbolKindModule:
		return "module"
	case protocol.SymbolKindNamespace:
		return "namespace"
	case protocol.SymbolKindPackage:
		return "package"
	case protocol.SymbolKindClass:
		return "class"
	case protocol.SymbolKindMethod:
		return "method"
	case protocol.SymbolKindProperty:
		return "property"
	case protocol.SymbolKindField:
		return "field"
	case protocol.SymbolKindConstructor:
		return "constructor"
	case protocol.SymbolKindEnum:
		return "enum"
	case protocol.SymbolKindInterface:
		return "interface"
	case protocol.SymbolKindFunction:
		return "function"
	case protocol.SymbolKindVariable:
		return "variable"
	case protocol.SymbolKindConstant:
		return "constant"
	case protocol.SymbolKindString:
		return "string"
	case protocol.SymbolKindNumber:
		return "number"
	case protocol.SymbolKindBoolean:
		return "boolean"
	case protocol.SymbolKindArray:
		return "array"
	case protocol.SymbolKindObject:
		return "object"
	case protocol.SymbolKindKey:
		return "key"
	case protocol.SymbolKindNull:
		return "null"
	case protocol.SymbolKindEnumMember:
		return "enum_member"
	case protocol.SymbolKindStruct:
		return "struct"
	case protocol.SymbolKindEvent:
		return "event"
	case protocol.SymbolKindOperator:
		return "operator"
	case protocol.SymbolKindTypeParameter:
		return "type_parameter"
	default:
		return "unknown"
	}
}

// extractHoverText extracts text from LSP hover contents
func (se *SymbolExtractor) extractHoverText(contents any) string {
	switch c := contents.(type) {
	case string:
		return c
	case protocol.MarkupContent:
		return c.Value
	default:
		return ""
	}
}

// fallbackSymbolExtraction provides basic symbol extraction without LSP
func (se *SymbolExtractor) fallbackSymbolExtraction(content, language string) []vectordb.Symbol {
	symbols := []vectordb.Symbol{}
	lines := strings.Split(content, "\n")

	// Simple regex-based symbol extraction for common patterns
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		// Extract symbols based on language patterns
		switch strings.ToLower(language) {
		case "go", "golang":
			symbols = append(symbols, se.extractGoSymbols(line, i+1)...)
		case "javascript", "js", "typescript", "ts":
			symbols = append(symbols, se.extractJSSymbols(line, i+1)...)
		case "python", "py":
			symbols = append(symbols, se.extractPythonSymbols(line, i+1)...)
		case "rust", "rs":
			symbols = append(symbols, se.extractRustSymbols(line, i+1)...)
		}
	}

	return symbols
}

// extractGoSymbols extracts Go symbols from a line
func (se *SymbolExtractor) extractGoSymbols(line string, lineNum int) []vectordb.Symbol {
	symbols := []vectordb.Symbol{}

	// Function declarations
	if strings.HasPrefix(line, "func ") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			funcName := strings.Split(parts[1], "(")[0]
			symbols = append(symbols, vectordb.Symbol{
				Name: funcName,
				Kind: "function",
				Location: vectordb.SourceLocation{
					StartLine:   lineNum,
					EndLine:     lineNum,
					StartColumn: 1,
					EndColumn:   len(line),
				},
			})
		}
	}

	// Type declarations
	if strings.HasPrefix(line, "type ") {
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			typeName := parts[1]
			symbols = append(symbols, vectordb.Symbol{
				Name: typeName,
				Kind: "type",
				Location: vectordb.SourceLocation{
					StartLine:   lineNum,
					EndLine:     lineNum,
					StartColumn: 1,
					EndColumn:   len(line),
				},
			})
		}
	}

	return symbols
}

// extractJSSymbols extracts JavaScript/TypeScript symbols from a line
func (se *SymbolExtractor) extractJSSymbols(line string, lineNum int) []vectordb.Symbol {
	symbols := []vectordb.Symbol{}

	// Function declarations
	if strings.Contains(line, "function ") {
		// Extract function name
		start := strings.Index(line, "function ") + 9
		if start < len(line) {
			remaining := line[start:]
			end := strings.Index(remaining, "(")
			if end > 0 {
				funcName := strings.TrimSpace(remaining[:end])
				symbols = append(symbols, vectordb.Symbol{
					Name: funcName,
					Kind: "function",
					Location: vectordb.SourceLocation{
						StartLine:   lineNum,
						EndLine:     lineNum,
						StartColumn: 1,
						EndColumn:   len(line),
					},
				})
			}
		}
	}

	// Class declarations
	if strings.Contains(line, "class ") {
		start := strings.Index(line, "class ") + 6
		if start < len(line) {
			remaining := strings.TrimSpace(line[start:])
			parts := strings.Fields(remaining)
			if len(parts) > 0 {
				className := parts[0]
				symbols = append(symbols, vectordb.Symbol{
					Name: className,
					Kind: "class",
					Location: vectordb.SourceLocation{
						StartLine:   lineNum,
						EndLine:     lineNum,
						StartColumn: 1,
						EndColumn:   len(line),
					},
				})
			}
		}
	}

	return symbols
}

// extractPythonSymbols extracts Python symbols from a line
func (se *SymbolExtractor) extractPythonSymbols(line string, lineNum int) []vectordb.Symbol {
	symbols := []vectordb.Symbol{}

	// Function definitions
	if strings.HasPrefix(line, "def ") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			funcName := strings.Split(parts[1], "(")[0]
			symbols = append(symbols, vectordb.Symbol{
				Name: funcName,
				Kind: "function",
				Location: vectordb.SourceLocation{
					StartLine:   lineNum,
					EndLine:     lineNum,
					StartColumn: 1,
					EndColumn:   len(line),
				},
			})
		}
	}

	// Class definitions
	if strings.HasPrefix(line, "class ") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			className := strings.TrimSuffix(parts[1], ":")
			className = strings.Split(className, "(")[0]
			symbols = append(symbols, vectordb.Symbol{
				Name: className,
				Kind: "class",
				Location: vectordb.SourceLocation{
					StartLine:   lineNum,
					EndLine:     lineNum,
					StartColumn: 1,
					EndColumn:   len(line),
				},
			})
		}
	}

	return symbols
}

// extractRustSymbols extracts Rust symbols from a line
func (se *SymbolExtractor) extractRustSymbols(line string, lineNum int) []vectordb.Symbol {
	symbols := []vectordb.Symbol{}

	// Function definitions
	if strings.Contains(line, "fn ") {
		start := strings.Index(line, "fn ") + 3
		if start < len(line) {
			remaining := strings.TrimSpace(line[start:])
			parts := strings.Fields(remaining)
			if len(parts) > 0 {
				funcName := strings.Split(parts[0], "(")[0]
				symbols = append(symbols, vectordb.Symbol{
					Name: funcName,
					Kind: "function",
					Location: vectordb.SourceLocation{
						StartLine:   lineNum,
						EndLine:     lineNum,
						StartColumn: 1,
						EndColumn:   len(line),
					},
				})
			}
		}
	}

	// Struct definitions
	if strings.Contains(line, "struct ") {
		start := strings.Index(line, "struct ") + 7
		if start < len(line) {
			remaining := strings.TrimSpace(line[start:])
			parts := strings.Fields(remaining)
			if len(parts) > 0 {
				structName := parts[0]
				symbols = append(symbols, vectordb.Symbol{
					Name: structName,
					Kind: "struct",
					Location: vectordb.SourceLocation{
						StartLine:   lineNum,
						EndLine:     lineNum,
						StartColumn: 1,
						EndColumn:   len(line),
					},
				})
			}
		}
	}

	return symbols
}

// extractSourceDocumentation extracts documentation for a symbol from source content
func (se *SymbolExtractor) extractSourceDocumentation(lspSymbol protocol.DocumentSymbol, sourceContent string) string {
	lines := strings.Split(sourceContent, "\n")

	// Get the line number where the symbol starts (LSP uses 0-based indexing)
	symbolLine := int(lspSymbol.Range.Start.Line)

	if symbolLine >= len(lines) || symbolLine < 0 {
		return ""
	}

	var docLines []string

	// Look backwards from the symbol line to find documentation comments
	for i := symbolLine - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])

		// Stop if we hit an empty line or non-comment
		if line == "" {
			break
		}

		// Check for different comment styles
		if after, found := strings.CutPrefix(line, "//"); found {
			// Go, Rust, C++, JavaScript style comments
			comment := strings.TrimSpace(after)
			docLines = append([]string{comment}, docLines...)
		} else if after, found := strings.CutPrefix(line, "#"); found {
			// Python, Shell style comments
			comment := strings.TrimSpace(after)
			docLines = append([]string{comment}, docLines...)
		} else if strings.HasPrefix(line, "*") && i > 0 {
			// Multi-line comment continuation
			comment := strings.TrimSpace(strings.TrimPrefix(line, "*"))
			docLines = append([]string{comment}, docLines...)
		} else if strings.HasPrefix(line, "/**") || strings.HasPrefix(line, "/*") {
			// Start of multi-line comment
			comment := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(line, "/**"), "/*"))
			if comment != "" {
				docLines = append([]string{comment}, docLines...)
			}
			break
		} else if after, found := strings.CutPrefix(line, "\"\"\""); found {
			// Python docstring - extract content if present
			docstring := strings.TrimSpace(after)
			if docstring != "" && !strings.HasSuffix(docstring, "\"\"\"") {
				docLines = append([]string{docstring}, docLines...)
			}
			break // Handle docstrings separately if needed
		} else {
			// Not a comment, stop looking
			break
		}
	}

	// Join the documentation lines
	if len(docLines) > 0 {
		return strings.Join(docLines, " ")
	}

	return ""
}
