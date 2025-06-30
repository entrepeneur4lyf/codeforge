package chunking

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

// ChunkingStrategy defines different approaches to code chunking
type ChunkingStrategy int

const (
	StrategyTreeSitter ChunkingStrategy = iota // Semantic chunking using tree-sitter
	StrategyFunction                           // Function-level chunking
	StrategyClass                              // Class-level chunking
	StrategyFile                               // File-level chunking
	StrategyText                               // Simple text chunking
)

// ChunkingConfig holds configuration for the chunker
type ChunkingConfig struct {
	MaxChunkSize    int              // Maximum characters per chunk
	OverlapSize     int              // Overlap between chunks
	Strategy        ChunkingStrategy // Chunking strategy to use
	IncludeContext  bool             // Include surrounding context
	ExtractSymbols  bool             // Extract symbol information
	ExtractImports  bool             // Extract import statements
	ExtractComments bool             // Extract documentation comments
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() ChunkingConfig {
	return ChunkingConfig{
		MaxChunkSize:    2000,
		OverlapSize:     200,
		Strategy:        StrategyTreeSitter,
		IncludeContext:  true,
		ExtractSymbols:  true,
		ExtractImports:  true,
		ExtractComments: true,
	}
}

// CodeChunker handles intelligent code chunking using tree-sitter
type CodeChunker struct {
	config  ChunkingConfig
	parsers map[string]*sitter.Parser
	mu      sync.RWMutex
}

// NewCodeChunker creates a new code chunker with the given configuration
func NewCodeChunker(config ChunkingConfig) *CodeChunker {
	return &CodeChunker{
		config: config,
	}
}

// ChunkFile chunks a file into semantic code chunks
func (c *CodeChunker) ChunkFile(ctx context.Context, filePath, content, language string) ([]*vectordb.CodeChunk, error) {
	switch c.config.Strategy {
	case StrategyTreeSitter:
		return c.chunkWithTreeSitter(ctx, filePath, content, language)
	case StrategyFunction:
		return c.chunkByFunction(ctx, filePath, content, language)
	case StrategyClass:
		return c.chunkByClass(ctx, filePath, content, language)
	case StrategyFile:
		return c.chunkByFile(ctx, filePath, content, language)
	case StrategyText:
		return c.chunkByText(ctx, filePath, content, language)
	default:
		return c.chunkWithTreeSitter(ctx, filePath, content, language)
	}
}

// chunkWithTreeSitter uses tree-sitter for semantic chunking
func (c *CodeChunker) chunkWithTreeSitter(ctx context.Context, filePath, content, language string) ([]*vectordb.CodeChunk, error) {
	// Use tree-sitter for precise AST-based chunking
	parser, err := c.getTreeSitterParser(language)
	if err != nil {
		// Fallback to function-based chunking if tree-sitter is not available
		return c.chunkByFunction(ctx, filePath, content, language)
	}

	// Parse the source code into an AST
	tree := parser.Parse([]byte(content), nil)
	if tree == nil {
		return c.chunkByFunction(ctx, filePath, content, language)
	}
	defer tree.Close()

	// Extract chunks from the AST
	return c.extractChunksFromAST(tree, filePath, content, language)
}

// Fallback chunking strategies

// chunkByFunction chunks code by function boundaries
func (c *CodeChunker) chunkByFunction(ctx context.Context, filePath, content, language string) ([]*vectordb.CodeChunk, error) {
	lines := strings.Split(content, "\n")

	switch strings.ToLower(language) {
	case "go":
		return c.chunkGoFunctions(filePath, content, lines)
	case "rust":
		return c.chunkRustFunctions(filePath, content, lines)
	case "python":
		return c.chunkPythonFunctions(filePath, content, lines)
	case "javascript", "typescript":
		return c.chunkJSFunctions(filePath, content, lines)
	case "java":
		return c.chunkJavaFunctions(filePath, content, lines)
	case "c", "cpp", "c++":
		return c.chunkCFunctions(filePath, content, lines)
	case "php":
		return c.chunkPHPFunctions(filePath, content, lines)
	default:
		// Fallback to text chunking for unsupported languages
		return c.chunkByText(ctx, filePath, content, language)
	}
}

// chunkByClass chunks code by class boundaries
func (c *CodeChunker) chunkByClass(ctx context.Context, filePath, content, language string) ([]*vectordb.CodeChunk, error) {
	// This would use regex or simple parsing to find class boundaries
	// For now, fallback to text chunking
	return c.chunkByText(ctx, filePath, content, language)
}

// chunkByFile creates a single chunk for the entire file
func (c *CodeChunker) chunkByFile(_ context.Context, filePath, content, language string) ([]*vectordb.CodeChunk, error) {
	chunk := &vectordb.CodeChunk{
		ID:       fmt.Sprintf("%s_file", filepath.Base(filePath)),
		FilePath: filePath,
		Content:  content,
		ChunkType: vectordb.ChunkType{
			Type: "file",
			Data: map[string]any{
				"full_file": true,
			},
		},
		Language: language,
		Location: vectordb.SourceLocation{
			StartLine:   1,
			EndLine:     strings.Count(content, "\n") + 1,
			StartColumn: 1,
			EndColumn:   1,
		},
		Metadata: map[string]string{
			"chunk_strategy": "file",
		},
	}

	return []*vectordb.CodeChunk{chunk}, nil
}

// chunkByText performs simple text-based chunking
func (c *CodeChunker) chunkByText(_ context.Context, filePath, content, language string) ([]*vectordb.CodeChunk, error) {
	chunks := []*vectordb.CodeChunk{}
	lines := strings.Split(content, "\n")

	chunkSize := c.config.MaxChunkSize
	overlap := c.config.OverlapSize

	currentChunk := ""
	startLine := 1
	chunkIndex := 0

	for i, line := range lines {
		if len(currentChunk)+len(line)+1 > chunkSize && len(currentChunk) > 0 {
			// Create chunk
			chunk := &vectordb.CodeChunk{
				ID:       fmt.Sprintf("%s_text_%d", filepath.Base(filePath), chunkIndex),
				FilePath: filePath,
				Content:  strings.TrimSpace(currentChunk),
				ChunkType: vectordb.ChunkType{
					Type: "text",
					Data: map[string]any{
						"chunk_index": chunkIndex,
					},
				},
				Language: language,
				Location: vectordb.SourceLocation{
					StartLine:   startLine,
					EndLine:     i,
					StartColumn: 1,
					EndColumn:   len(line),
				},
				Metadata: map[string]string{
					"chunk_strategy": "text",
				},
			}
			chunks = append(chunks, chunk)

			// Start new chunk with overlap
			overlapLines := []string{}
			if overlap > 0 && len(lines) > i-overlap {
				overlapStart := max(0, i-overlap)
				overlapLines = lines[overlapStart:i]
			}

			currentChunk = strings.Join(overlapLines, "\n")
			if len(currentChunk) > 0 {
				currentChunk += "\n"
			}
			startLine = i - len(overlapLines) + 1
			chunkIndex++
		}

		currentChunk += line + "\n"
	}

	// Add final chunk if there's remaining content
	if len(strings.TrimSpace(currentChunk)) > 0 {
		chunk := &vectordb.CodeChunk{
			ID:       fmt.Sprintf("%s_text_%d", filepath.Base(filePath), chunkIndex),
			FilePath: filePath,
			Content:  strings.TrimSpace(currentChunk),
			ChunkType: vectordb.ChunkType{
				Type: "text",
				Data: map[string]any{
					"chunk_index": chunkIndex,
				},
			},
			Language: language,
			Location: vectordb.SourceLocation{
				StartLine:   startLine,
				EndLine:     len(lines),
				StartColumn: 1,
				EndColumn:   len(lines[len(lines)-1]),
			},
			Metadata: map[string]string{
				"chunk_strategy": "text",
			},
		}
		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// Language-specific function chunkers

// chunkGoFunctions chunks Go code by function boundaries
func (c *CodeChunker) chunkGoFunctions(filePath, content string, lines []string) ([]*vectordb.CodeChunk, error) {
	// Validate input consistency
	if content != strings.Join(lines, "\n") {
		lines = strings.Split(content, "\n") // Use authoritative content
	}

	chunks := []*vectordb.CodeChunk{}
	var currentFunc strings.Builder
	var funcName string
	var startLine int
	inFunction := false
	braceCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect function start
		if strings.HasPrefix(trimmed, "func ") {
			// Save previous function if exists
			if inFunction && currentFunc.Len() > 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i-1))
			}

			// Start new function
			funcName = c.extractGoFunctionName(trimmed)
			currentFunc.Reset()
			currentFunc.WriteString(line + "\n")
			startLine = i + 1
			inFunction = true
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
		} else if inFunction {
			currentFunc.WriteString(line + "\n")
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Function ends when braces are balanced
			if braceCount <= 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i+1))
				inFunction = false
				currentFunc.Reset()
			}
		}
	}

	// Handle final function
	if inFunction && currentFunc.Len() > 0 {
		chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, len(lines)))
	}

	return chunks, nil
}

// chunkPythonFunctions chunks Python code by function boundaries
func (c *CodeChunker) chunkPythonFunctions(filePath, content string, lines []string) ([]*vectordb.CodeChunk, error) {
	// Validate input consistency
	if content != strings.Join(lines, "\n") {
		lines = strings.Split(content, "\n") // Use authoritative content
	}

	chunks := []*vectordb.CodeChunk{}
	var currentFunc strings.Builder
	var funcName string
	var startLine int
	inFunction := false
	baseIndent := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect function start
		if strings.HasPrefix(trimmed, "def ") {
			// Save previous function if exists
			if inFunction && currentFunc.Len() > 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i-1))
			}

			// Start new function
			funcName = c.extractPythonFunctionName(trimmed)
			currentFunc.Reset()
			currentFunc.WriteString(line + "\n")
			startLine = i + 1
			inFunction = true
			baseIndent = len(line) - len(strings.TrimLeft(line, " \t"))
		} else if inFunction {
			currentIndent := len(line) - len(strings.TrimLeft(line, " \t"))

			// Function ends when we reach same or lower indentation level (and line is not empty)
			if trimmed != "" && currentIndent <= baseIndent {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i))
				inFunction = false
				currentFunc.Reset()

				// Check if this line starts a new function
				if strings.HasPrefix(trimmed, "def ") {
					funcName = c.extractPythonFunctionName(trimmed)
					currentFunc.WriteString(line + "\n")
					startLine = i + 1
					inFunction = true
					baseIndent = currentIndent
				}
			} else {
				currentFunc.WriteString(line + "\n")
			}
		}
	}

	// Handle final function
	if inFunction && currentFunc.Len() > 0 {
		chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, len(lines)))
	}

	return chunks, nil
}

// Placeholder implementations for other languages
func (c *CodeChunker) chunkRustFunctions(filePath, content string, lines []string) ([]*vectordb.CodeChunk, error) {
	// Validate input consistency
	if content != strings.Join(lines, "\n") {
		lines = strings.Split(content, "\n") // Use authoritative content
	}

	chunks := []*vectordb.CodeChunk{}
	var currentFunc strings.Builder
	var funcName string
	var startLine int
	inFunction := false
	braceCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect function start (fn keyword)
		if strings.HasPrefix(trimmed, "fn ") || strings.Contains(trimmed, " fn ") {
			// Save previous function if exists
			if inFunction && currentFunc.Len() > 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i-1))
			}

			// Start new function
			funcName = c.extractRustFunctionName(trimmed)
			currentFunc.Reset()
			currentFunc.WriteString(line + "\n")
			startLine = i + 1
			inFunction = true
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
		} else if inFunction {
			currentFunc.WriteString(line + "\n")
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Function ends when braces are balanced
			if braceCount <= 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i+1))
				inFunction = false
				currentFunc.Reset()
			}
		}
	}

	// Handle final function
	if inFunction && currentFunc.Len() > 0 {
		chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, len(lines)))
	}

	return chunks, nil
}

func (c *CodeChunker) chunkJSFunctions(filePath, content string, lines []string) ([]*vectordb.CodeChunk, error) {
	// Validate input consistency
	if content != strings.Join(lines, "\n") {
		lines = strings.Split(content, "\n") // Use authoritative content
	}

	chunks := []*vectordb.CodeChunk{}
	var currentFunc strings.Builder
	var funcName string
	var startLine int
	inFunction := false
	braceCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect function start (various JS/TS patterns)
		if c.isJSFunctionStart(trimmed) {
			// Save previous function if exists
			if inFunction && currentFunc.Len() > 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i-1))
			}

			// Start new function
			funcName = c.extractJSFunctionName(trimmed)
			currentFunc.Reset()
			currentFunc.WriteString(line + "\n")
			startLine = i + 1
			inFunction = true
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
		} else if inFunction {
			currentFunc.WriteString(line + "\n")
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Function ends when braces are balanced
			if braceCount <= 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i+1))
				inFunction = false
				currentFunc.Reset()
			}
		}
	}

	// Handle final function
	if inFunction && currentFunc.Len() > 0 {
		chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, len(lines)))
	}

	return chunks, nil
}

func (c *CodeChunker) chunkJavaFunctions(filePath, content string, lines []string) ([]*vectordb.CodeChunk, error) {
	// Validate input consistency
	if content != strings.Join(lines, "\n") {
		lines = strings.Split(content, "\n") // Use authoritative content
	}

	chunks := []*vectordb.CodeChunk{}
	var currentFunc strings.Builder
	var funcName string
	var startLine int
	inFunction := false
	braceCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect method start (public/private/protected + return type + method name)
		if c.isJavaMethodStart(trimmed) {
			// Save previous function if exists
			if inFunction && currentFunc.Len() > 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i-1))
			}

			// Start new function
			funcName = c.extractJavaMethodName(trimmed)
			currentFunc.Reset()
			currentFunc.WriteString(line + "\n")
			startLine = i + 1
			inFunction = true
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
		} else if inFunction {
			currentFunc.WriteString(line + "\n")
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Function ends when braces are balanced
			if braceCount <= 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i+1))
				inFunction = false
				currentFunc.Reset()
			}
		}
	}

	// Handle final function
	if inFunction && currentFunc.Len() > 0 {
		chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, len(lines)))
	}

	return chunks, nil
}

func (c *CodeChunker) chunkCFunctions(filePath, content string, lines []string) ([]*vectordb.CodeChunk, error) {
	// Validate input consistency
	if content != strings.Join(lines, "\n") {
		lines = strings.Split(content, "\n") // Use authoritative content
	}

	chunks := []*vectordb.CodeChunk{}
	var currentFunc strings.Builder
	var funcName string
	var startLine int
	inFunction := false
	braceCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip preprocessor directives, comments, etc.
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") ||
			strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		// Detect function start (return type + function name + parameters)
		if c.isCFunctionStart(trimmed, i, lines) {
			// Save previous function if exists
			if inFunction && currentFunc.Len() > 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i-1))
			}

			// Start new function
			funcName = c.extractCFunctionName(trimmed)
			currentFunc.Reset()
			currentFunc.WriteString(line + "\n")
			startLine = i + 1
			inFunction = true
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
		} else if inFunction {
			currentFunc.WriteString(line + "\n")
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Function ends when braces are balanced
			if braceCount <= 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i+1))
				inFunction = false
				currentFunc.Reset()
			}
		}
	}

	// Handle final function
	if inFunction && currentFunc.Len() > 0 {
		chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, len(lines)))
	}

	return chunks, nil
}

func (c *CodeChunker) chunkPHPFunctions(filePath, content string, lines []string) ([]*vectordb.CodeChunk, error) {
	// Validate input consistency
	if content != strings.Join(lines, "\n") {
		lines = strings.Split(content, "\n") // Use authoritative content
	}

	chunks := []*vectordb.CodeChunk{}
	var currentFunc strings.Builder
	var funcName string
	var startLine int
	inFunction := false
	braceCount := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}

		// Detect PHP function start
		if c.isPHPFunctionStart(trimmed) {
			// Save previous function if exists
			if inFunction && currentFunc.Len() > 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i-1))
			}

			// Start new function
			funcName = c.extractPHPFunctionName(trimmed)
			currentFunc.Reset()
			currentFunc.WriteString(line + "\n")
			startLine = i + 1
			inFunction = true
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
		} else if inFunction {
			currentFunc.WriteString(line + "\n")
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")

			// Function ends when braces are balanced
			if braceCount <= 0 {
				chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, i+1))
				inFunction = false
				currentFunc.Reset()
			}
		}
	}

	// Handle final function
	if inFunction && currentFunc.Len() > 0 {
		chunks = append(chunks, c.createFunctionChunk(filePath, funcName, currentFunc.String(), startLine, len(lines)))
	}

	return chunks, nil
}

// Helper methods

// createFunctionChunk creates a function chunk
func (c *CodeChunker) createFunctionChunk(filePath, funcName, content string, startLine, endLine int) *vectordb.CodeChunk {
	return &vectordb.CodeChunk{
		ID:       fmt.Sprintf("%s_func_%s", filepath.Base(filePath), funcName),
		FilePath: filePath,
		Content:  strings.TrimSpace(content),
		ChunkType: vectordb.ChunkType{
			Type: "function",
			Data: map[string]any{
				"function_name": funcName,
			},
		},
		Language: c.detectLanguageFromPath(filePath),
		Location: vectordb.SourceLocation{
			StartLine:   startLine,
			EndLine:     endLine,
			StartColumn: 1,
			EndColumn:   1,
		},
	}
}

// extractGoFunctionName extracts function name from Go function declaration
func (c *CodeChunker) extractGoFunctionName(line string) string {
	// func functionName(...) or func (receiver) functionName(...)
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "unknown"
	}

	// Handle receiver functions: func (r *Type) methodName(...)
	if strings.HasPrefix(parts[1], "(") {
		for i := 2; i < len(parts); i++ {
			if !strings.Contains(parts[i], ")") {
				continue
			}
			if i+1 < len(parts) {
				name := strings.Split(parts[i+1], "(")[0]
				return name
			}
		}
	} else {
		// Regular function: func functionName(...)
		name := strings.Split(parts[1], "(")[0]
		return name
	}

	return "unknown"
}

// extractPythonFunctionName extracts function name from Python function declaration
func (c *CodeChunker) extractPythonFunctionName(line string) string {
	// def function_name(...):
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return "unknown"
	}

	name := strings.Split(parts[1], "(")[0]
	return name
}

// extractRustFunctionName extracts function name from Rust function declaration
func (c *CodeChunker) extractRustFunctionName(line string) string {
	// fn function_name(...) or pub fn function_name(...)
	parts := strings.Fields(line)

	// Find the "fn" keyword
	fnIndex := -1
	for i, part := range parts {
		if part == "fn" {
			fnIndex = i
			break
		}
	}

	if fnIndex == -1 || fnIndex+1 >= len(parts) {
		return "unknown"
	}

	// Function name is after "fn"
	name := strings.Split(parts[fnIndex+1], "(")[0]
	return name
}

// isJSFunctionStart checks if a line starts a JavaScript/TypeScript function
func (c *CodeChunker) isJSFunctionStart(line string) bool {
	// Various JS/TS function patterns
	patterns := []string{
		"function ",              // function name() {}
		"async function ",        // async function name() {}
		"export function ",       // export function name() {}
		"export async function ", // export async function name() {}
		"const ",                 // const name = () => {}
		"let ",                   // let name = () => {}
		"var ",                   // var name = function() {}
	}

	for _, pattern := range patterns {
		if strings.HasPrefix(line, pattern) {
			// Check if it contains function-like syntax
			if strings.Contains(line, "(") && (strings.Contains(line, "=>") || strings.Contains(line, "function")) {
				return true
			}
		}
	}

	// Method definitions: methodName() {}
	if strings.Contains(line, "(") && strings.Contains(line, ")") && strings.Contains(line, "{") {
		// Simple heuristic: if it looks like a method
		if !strings.Contains(line, "=") && !strings.Contains(line, "if") && !strings.Contains(line, "for") {
			return true
		}
	}

	return false
}

// extractJSFunctionName extracts function name from JavaScript/TypeScript function declaration
func (c *CodeChunker) extractJSFunctionName(line string) string {
	// Handle different patterns
	if strings.HasPrefix(line, "function ") || strings.Contains(line, " function ") {
		// function name() {} or export function name() {}
		parts := strings.Fields(line)
		for i, part := range parts {
			if part == "function" && i+1 < len(parts) {
				name := strings.Split(parts[i+1], "(")[0]
				return name
			}
		}
	} else if strings.Contains(line, "const ") || strings.Contains(line, "let ") || strings.Contains(line, "var ") {
		// const name = () => {} or let name = function() {}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			name := strings.TrimSuffix(strings.TrimSpace(parts[1]), "=")
			return name
		}
	} else {
		// Method definition: methodName() {}
		if idx := strings.Index(line, "("); idx > 0 {
			beforeParen := strings.TrimSpace(line[:idx])
			parts := strings.Fields(beforeParen)
			if len(parts) > 0 {
				return parts[len(parts)-1]
			}
		}
	}

	return "unknown"
}

// isJavaMethodStart checks if a line starts a Java method
func (c *CodeChunker) isJavaMethodStart(line string) bool {
	// Skip class declarations, interfaces, etc.
	if strings.Contains(line, "class ") || strings.Contains(line, "interface ") ||
		strings.Contains(line, "enum ") || strings.Contains(line, "import ") ||
		strings.Contains(line, "package ") {
		return false
	}

	// Must contain parentheses for method signature
	if !strings.Contains(line, "(") || !strings.Contains(line, ")") {
		return false
	}

	// Common method modifiers
	modifiers := []string{"public", "private", "protected", "static", "final", "abstract", "synchronized"}
	hasModifier := false
	for _, modifier := range modifiers {
		if strings.Contains(line, modifier+" ") {
			hasModifier = true
			break
		}
	}

	// If no explicit modifier, might be package-private method
	if !hasModifier {
		// Simple heuristic: contains parentheses and looks like method signature
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			// Look for pattern: returnType methodName(
			for i := 0; i < len(parts)-1; i++ {
				if strings.Contains(parts[i+1], "(") {
					return true
				}
			}
		}
	}

	return hasModifier
}

// extractJavaMethodName extracts method name from Java method declaration
func (c *CodeChunker) extractJavaMethodName(line string) string {
	// Find the method name (before the opening parenthesis)
	if idx := strings.Index(line, "("); idx > 0 {
		beforeParen := strings.TrimSpace(line[:idx])
		parts := strings.Fields(beforeParen)
		if len(parts) > 0 {
			// Method name is the last part before parentheses
			methodName := parts[len(parts)-1]
			// Remove any generic type parameters
			if genIdx := strings.Index(methodName, "<"); genIdx > 0 {
				methodName = methodName[:genIdx]
			}
			return methodName
		}
	}

	return "unknown"
}

// isCFunctionStart checks if a line starts a C/C++ function
func (c *CodeChunker) isCFunctionStart(line string, lineIndex int, lines []string) bool {
	// Skip obvious non-function lines
	if strings.Contains(line, "struct ") || strings.Contains(line, "typedef ") ||
		strings.Contains(line, "enum ") || strings.Contains(line, "#") ||
		strings.HasSuffix(line, ";") {
		return false
	}

	// Must contain parentheses
	if !strings.Contains(line, "(") {
		return false
	}

	// Look for function signature pattern: type name(params) {
	// or multi-line function declarations
	if strings.Contains(line, "(") && strings.Contains(line, ")") {
		// Check if this looks like a function declaration
		beforeParen := line[:strings.Index(line, "(")]
		parts := strings.Fields(beforeParen)

		// Need at least return type and function name
		if len(parts) >= 2 {
			// Last part should be function name
			funcName := parts[len(parts)-1]
			// Function names typically start with letter or underscore
			if len(funcName) > 0 && (funcName[0] >= 'a' && funcName[0] <= 'z' ||
				funcName[0] >= 'A' && funcName[0] <= 'Z' || funcName[0] == '_') {

				// Check if next few lines contain opening brace (for multi-line declarations)
				for j := lineIndex; j < len(lines) && j < lineIndex+3; j++ {
					if strings.Contains(lines[j], "{") {
						return true
					}
					// If we hit a semicolon, it's a declaration, not definition
					if strings.Contains(lines[j], ";") {
						return false
					}
				}
			}
		}
	}

	return false
}

// extractCFunctionName extracts function name from C/C++ function declaration
func (c *CodeChunker) extractCFunctionName(line string) string {
	// Find the function name (before the opening parenthesis)
	if idx := strings.Index(line, "("); idx > 0 {
		beforeParen := strings.TrimSpace(line[:idx])
		parts := strings.Fields(beforeParen)
		if len(parts) > 0 {
			// Function name is the last part before parentheses
			funcName := parts[len(parts)-1]
			// Remove any pointer indicators
			funcName = strings.TrimPrefix(funcName, "*")
			return funcName
		}
	}

	return "unknown"
}

// isPHPFunctionStart checks if a line starts a PHP function
func (c *CodeChunker) isPHPFunctionStart(line string) bool {
	trimmed := strings.TrimSpace(line)

	// PHP function patterns:
	// function functionName(...)
	// public function functionName(...)
	// private function functionName(...)
	// protected function functionName(...)
	// static function functionName(...)
	// public static function functionName(...)

	// Check for function keyword
	if strings.Contains(trimmed, "function ") {
		// Make sure it's not in a comment or string
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") ||
			strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			return false
		}

		// Check for opening parenthesis (function declaration)
		if strings.Contains(trimmed, "(") {
			return true
		}
	}

	return false
}

// extractPHPFunctionName extracts function name from PHP function declaration
func (c *CodeChunker) extractPHPFunctionName(line string) string {
	// Find "function" keyword and extract name after it
	if idx := strings.Index(line, "function "); idx >= 0 {
		afterFunction := strings.TrimSpace(line[idx+9:]) // "function " is 9 chars

		// Find the function name (before the opening parenthesis)
		if parenIdx := strings.Index(afterFunction, "("); parenIdx > 0 {
			funcName := strings.TrimSpace(afterFunction[:parenIdx])
			// Remove any reference operator
			funcName = strings.TrimPrefix(funcName, "&")
			return funcName
		}
	}

	return "unknown"
}

// detectLanguageFromPath detects language from file path
func (c *CodeChunker) detectLanguageFromPath(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".py":
		return "python"
	case ".js", ".mjs":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".h", ".hpp":
		return "c"
	case ".php":
		return "php"
	default:
		return "text"
	}
}

// getTreeSitterParser returns a tree-sitter parser for the given language
func (c *CodeChunker) getTreeSitterParser(language string) (*sitter.Parser, error) {
	c.mu.RLock()
	if parser, exists := c.parsers[language]; exists {
		c.mu.RUnlock()
		return parser, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if parser, exists := c.parsers[language]; exists {
		return parser, nil
	}

	// Initialize parsers map if needed
	if c.parsers == nil {
		c.parsers = make(map[string]*sitter.Parser)
	}

	// Create parser for the language
	parser := sitter.NewParser()

	// For now, we'll use a simple approach without external language grammars
	// This is a placeholder until we can properly integrate language-specific parsers
	switch strings.ToLower(language) {
	case "go", "rust", "python", "javascript", "typescript", "java", "c", "cpp", "c++":
		// We'll implement basic parsing without language-specific grammars for now
		// This allows the tree-sitter infrastructure to work while we add proper language support
		c.parsers[language] = parser
		return parser, nil
	default:
		return nil, fmt.Errorf("unsupported language for tree-sitter: %s", language)
	}
}

// extractChunksFromAST extracts code chunks from a tree-sitter AST
func (c *CodeChunker) extractChunksFromAST(tree *sitter.Tree, filePath, content, language string) ([]*vectordb.CodeChunk, error) {
	chunks := []*vectordb.CodeChunk{}
	rootNode := tree.RootNode()

	// Extract function definitions and other top-level constructs
	chunks = append(chunks, c.extractFunctionChunks(rootNode, filePath, content, language)...)
	chunks = append(chunks, c.extractClassChunks(rootNode, filePath, content, language)...)
	chunks = append(chunks, c.extractStructChunks(rootNode, filePath, content, language)...)

	return chunks, nil
}

// extractFunctionChunks extracts function definitions from AST
func (c *CodeChunker) extractFunctionChunks(node *sitter.Node, filePath, content, language string) []*vectordb.CodeChunk {
	chunks := []*vectordb.CodeChunk{}

	// Define function node types for different languages
	functionTypes := c.getFunctionNodeTypes(language)

	// Traverse the AST to find function nodes
	c.traverseAST(node, func(n *sitter.Node) {
		nodeType := n.Kind()
		if slices.Contains(functionTypes, nodeType) {
			chunk := c.createChunkFromNode(n, filePath, content, "function")
			if chunk != nil {
				chunks = append(chunks, chunk)
			}
		}
	})

	return chunks
}

// extractClassChunks extracts class definitions from AST
func (c *CodeChunker) extractClassChunks(node *sitter.Node, filePath, content, language string) []*vectordb.CodeChunk {
	chunks := []*vectordb.CodeChunk{}

	classTypes := c.getClassNodeTypes(language)

	c.traverseAST(node, func(n *sitter.Node) {
		nodeType := n.Kind()
		if slices.Contains(classTypes, nodeType) {
			chunk := c.createChunkFromNode(n, filePath, content, "class")
			if chunk != nil {
				chunks = append(chunks, chunk)
			}
		}
	})

	return chunks
}

// extractStructChunks extracts struct definitions from AST
func (c *CodeChunker) extractStructChunks(node *sitter.Node, filePath, content, language string) []*vectordb.CodeChunk {
	chunks := []*vectordb.CodeChunk{}

	structTypes := c.getStructNodeTypes(language)

	c.traverseAST(node, func(n *sitter.Node) {
		nodeType := n.Kind()
		if slices.Contains(structTypes, nodeType) {
			chunk := c.createChunkFromNode(n, filePath, content, "struct")
			if chunk != nil {
				chunks = append(chunks, chunk)
			}
		}
	})

	return chunks
}

// getFunctionNodeTypes returns the AST node types for functions in each language
func (c *CodeChunker) getFunctionNodeTypes(language string) []string {
	switch strings.ToLower(language) {
	case "go":
		return []string{"function_declaration", "method_declaration"}
	case "rust":
		return []string{"function_item", "impl_item"}
	case "python":
		return []string{"function_definition"}
	case "javascript", "typescript":
		return []string{"function_declaration", "function_expression", "arrow_function", "method_definition"}
	case "java":
		return []string{"method_declaration", "constructor_declaration"}
	case "c", "cpp", "c++":
		return []string{"function_definition", "function_declarator"}
	default:
		return []string{}
	}
}

// getClassNodeTypes returns the AST node types for classes in each language
func (c *CodeChunker) getClassNodeTypes(language string) []string {
	switch strings.ToLower(language) {
	case "go":
		return []string{} // Go doesn't have classes
	case "rust":
		return []string{"struct_item", "enum_item", "trait_item"}
	case "python":
		return []string{"class_definition"}
	case "javascript", "typescript":
		return []string{"class_declaration"}
	case "java":
		return []string{"class_declaration", "interface_declaration", "enum_declaration"}
	case "c", "cpp", "c++":
		return []string{"class_specifier"}
	default:
		return []string{}
	}
}

// getStructNodeTypes returns the AST node types for structs in each language
func (c *CodeChunker) getStructNodeTypes(language string) []string {
	switch strings.ToLower(language) {
	case "go":
		return []string{"type_declaration"}
	case "rust":
		return []string{"struct_item"}
	case "python":
		return []string{} // Python doesn't have structs
	case "javascript", "typescript":
		return []string{} // JS doesn't have structs
	case "java":
		return []string{} // Java doesn't have structs
	case "c", "cpp", "c++":
		return []string{"struct_specifier"}
	default:
		return []string{}
	}
}

// traverseAST traverses the AST and calls the visitor function for each node
func (c *CodeChunker) traverseAST(node *sitter.Node, visitor func(*sitter.Node)) {
	visitor(node)

	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		if child != nil {
			c.traverseAST(child, visitor)
		}
	}
}

// createChunkFromNode creates a code chunk from an AST node
func (c *CodeChunker) createChunkFromNode(node *sitter.Node, filePath, content, chunkType string) *vectordb.CodeChunk {
	startByte := node.StartByte()
	endByte := node.EndByte()

	if startByte >= uint(len(content)) || endByte > uint(len(content)) {
		return nil
	}

	chunkContent := content[startByte:endByte]
	startPoint := node.StartPosition()
	endPoint := node.EndPosition()

	// Extract symbol name from the node
	symbolName := c.extractSymbolName(node, content)

	return &vectordb.CodeChunk{
		ID:       fmt.Sprintf("%s_%s_%s_%d", filepath.Base(filePath), chunkType, symbolName, startPoint.Row),
		FilePath: filePath,
		Content:  chunkContent,
		ChunkType: vectordb.ChunkType{
			Type: chunkType,
			Data: map[string]any{
				"symbol_name": symbolName,
				"node_type":   node.Kind(),
			},
		},
		Language: c.detectLanguageFromPath(filePath),
		Location: vectordb.SourceLocation{
			StartLine:   int(startPoint.Row) + 1, // Convert to 1-based
			EndLine:     int(endPoint.Row) + 1,   // Convert to 1-based
			StartColumn: int(startPoint.Column) + 1,
			EndColumn:   int(endPoint.Column) + 1,
		},
	}
}

// extractSymbolName extracts the symbol name from an AST node
func (c *CodeChunker) extractSymbolName(node *sitter.Node, content string) string {
	// Look for identifier nodes in the children
	childCount := int(node.ChildCount())
	for i := range childCount {
		child := node.Child(uint(i))
		if child != nil && child.Kind() == "identifier" {
			startByte := child.StartByte()
			endByte := child.EndByte()
			if startByte < uint(len(content)) && endByte <= uint(len(content)) {
				return content[startByte:endByte]
			}
		}
	}

	return "unknown"
}
