package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.lsp.dev/protocol"
)

// handleDiagnostics handles diagnostic notifications from the LSP server
func (c *Client) handleDiagnostics(params json.RawMessage) {
	var diagnosticsParams protocol.PublishDiagnosticsParams
	if err := json.Unmarshal(params, &diagnosticsParams); err != nil {
		fmt.Printf("Failed to unmarshal diagnostics: %v\n", err)
		return
	}

	uri := string(diagnosticsParams.URI)

	c.diagnosticsMu.Lock()
	c.diagnostics[uri] = diagnosticsParams.Diagnostics
	c.diagnosticsMu.Unlock()

	// Print diagnostics for debugging
	if len(diagnosticsParams.Diagnostics) > 0 {
		fmt.Printf("Diagnostics for %s:\n", uri)
		for _, diag := range diagnosticsParams.Diagnostics {
			fmt.Printf("  %s: %s\n", diag.Severity, diag.Message)
		}
	}
}

// handleShowMessage handles window/showMessage notifications
func (c *Client) handleShowMessage(params json.RawMessage) {
	var showMessageParams protocol.ShowMessageParams
	if err := json.Unmarshal(params, &showMessageParams); err != nil {
		fmt.Printf("Failed to unmarshal show message: %v\n", err)
		return
	}

	fmt.Printf("LSP Message [%s]: %s\n", showMessageParams.Type, showMessageParams.Message)
}

// GetDiagnostics returns diagnostics for a specific file
func (c *Client) GetDiagnostics(uri string) []protocol.Diagnostic {
	c.diagnosticsMu.RLock()
	defer c.diagnosticsMu.RUnlock()

	diagnostics, exists := c.diagnostics[uri]
	if !exists {
		return nil
	}

	// Return a copy to avoid race conditions
	result := make([]protocol.Diagnostic, len(diagnostics))
	copy(result, diagnostics)
	return result
}

// GetAllDiagnostics returns all diagnostics across all files
func (c *Client) GetAllDiagnostics() map[string][]protocol.Diagnostic {
	c.diagnosticsMu.RLock()
	defer c.diagnosticsMu.RUnlock()

	// Return a deep copy to avoid race conditions
	result := make(map[string][]protocol.Diagnostic)
	for uri, diagnostics := range c.diagnostics {
		if len(diagnostics) > 0 {
			diagCopy := make([]protocol.Diagnostic, len(diagnostics))
			copy(diagCopy, diagnostics)
			result[uri] = diagCopy
		}
	}
	return result
}

// OpenFile opens a file in the LSP server
func (c *Client) OpenFile(ctx context.Context, filepath string) error {
	uri := fmt.Sprintf("file://%s", filepath)

	c.openFilesMu.Lock()
	if _, exists := c.openFiles[uri]; exists {
		c.openFilesMu.Unlock()
		return nil // Already open
	}
	c.openFilesMu.Unlock()

	// Read file content
	content, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	params := protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        protocol.DocumentURI(uri),
			LanguageID: protocol.LanguageIdentifier(DetectLanguageID(uri)),
			Version:    1,
			Text:       string(content),
		},
	}

	if err := c.Notify(ctx, "textDocument/didOpen", params); err != nil {
		return err
	}

	// Store file info
	c.openFilesMu.Lock()
	c.openFiles[uri] = &OpenFileInfo{
		URI:        uri,
		LanguageID: DetectLanguageID(uri),
		Version:    1,
		Content:    string(content),
	}
	c.openFilesMu.Unlock()

	return nil
}

// CloseFile closes a file in the LSP server
func (c *Client) CloseFile(ctx context.Context, filepath string) error {
	uri := fmt.Sprintf("file://%s", filepath)

	c.openFilesMu.Lock()
	_, exists := c.openFiles[uri]
	if !exists {
		c.openFilesMu.Unlock()
		return nil // Not open
	}
	delete(c.openFiles, uri)
	c.openFilesMu.Unlock()

	params := protocol.DidCloseTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(uri),
		},
	}

	return c.Notify(ctx, "textDocument/didClose", params)
}

// GetCompletion requests code completion at a specific position
func (c *Client) GetCompletion(ctx context.Context, filepath string, line, character int) (*protocol.CompletionList, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.CompletionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: protocol.DocumentURI(uri),
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}

	var result protocol.CompletionList
	if err := c.Call(ctx, "textDocument/completion", params, &result); err != nil {
		return nil, fmt.Errorf("completion failed: %w", err)
	}

	return &result, nil
}

// GetHover requests hover information at a specific position
func (c *Client) GetHover(ctx context.Context, filepath string, line, character int) (*protocol.Hover, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: protocol.DocumentURI(uri),
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}

	var result protocol.Hover
	if err := c.Call(ctx, "textDocument/hover", params, &result); err != nil {
		return nil, fmt.Errorf("hover failed: %w", err)
	}

	return &result, nil
}

// GetDefinition requests definition information at a specific position
func (c *Client) GetDefinition(ctx context.Context, filepath string, line, character int) ([]protocol.Location, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: protocol.DocumentURI(uri),
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}

	var result []protocol.Location
	if err := c.Call(ctx, "textDocument/definition", params, &result); err != nil {
		return nil, fmt.Errorf("definition failed: %w", err)
	}

	return result, nil
}

// GetWorkspaceSymbols searches for symbols across the workspace
func (c *Client) GetWorkspaceSymbols(ctx context.Context, query string) ([]protocol.SymbolInformation, error) {
	params := protocol.WorkspaceSymbolParams{
		Query: query,
	}

	var result []protocol.SymbolInformation
	if err := c.Call(ctx, "workspace/symbol", params, &result); err != nil {
		return nil, fmt.Errorf("workspace symbol search failed: %w", err)
	}

	return result, nil
}

// GetDocumentSymbols gets symbols for a specific document
func (c *Client) GetDocumentSymbols(ctx context.Context, filepath string) ([]protocol.DocumentSymbol, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.DocumentSymbolParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(uri),
		},
	}

	var result []protocol.DocumentSymbol
	if err := c.Call(ctx, "textDocument/documentSymbol", params, &result); err != nil {
		return nil, fmt.Errorf("document symbol search failed: %w", err)
	}

	return result, nil
}

// GetReferences finds all references to a symbol
func (c *Client) GetReferences(ctx context.Context, filepath string, line, character int, includeDeclaration bool) ([]protocol.Location, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: protocol.DocumentURI(uri),
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: includeDeclaration,
		},
	}

	var result []protocol.Location
	if err := c.Call(ctx, "textDocument/references", params, &result); err != nil {
		return nil, fmt.Errorf("references search failed: %w", err)
	}

	return result, nil
}

// GetImplementation finds implementations of an interface or abstract method
func (c *Client) GetImplementation(ctx context.Context, filepath string, line, character int) ([]protocol.Location, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.ImplementationParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: protocol.DocumentURI(uri),
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}

	var result []protocol.Location
	if err := c.Call(ctx, "textDocument/implementation", params, &result); err != nil {
		return nil, fmt.Errorf("implementation search failed: %w", err)
	}

	return result, nil
}

// GetTypeDefinition finds the type definition of a symbol
func (c *Client) GetTypeDefinition(ctx context.Context, filepath string, line, character int) ([]protocol.Location, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.TypeDefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: protocol.DocumentURI(uri),
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}

	var result []protocol.Location
	if err := c.Call(ctx, "textDocument/typeDefinition", params, &result); err != nil {
		return nil, fmt.Errorf("type definition search failed: %w", err)
	}

	return result, nil
}

// GetCodeActions gets available code actions for a range
func (c *Client) GetCodeActions(ctx context.Context, filepath string, startLine, startChar, endLine, endChar int, diagnostics []protocol.Diagnostic) ([]protocol.CodeAction, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.CodeActionParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(uri),
		},
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(startLine),
				Character: uint32(startChar),
			},
			End: protocol.Position{
				Line:      uint32(endLine),
				Character: uint32(endChar),
			},
		},
		Context: protocol.CodeActionContext{
			Diagnostics: diagnostics,
		},
	}

	var result []protocol.CodeAction
	if err := c.Call(ctx, "textDocument/codeAction", params, &result); err != nil {
		return nil, fmt.Errorf("code actions failed: %w", err)
	}

	return result, nil
}

// ExecuteCommand executes a workspace command
func (c *Client) ExecuteCommand(ctx context.Context, command string, args []any) (any, error) {
	params := protocol.ExecuteCommandParams{
		Command:   command,
		Arguments: args,
	}

	var result any
	if err := c.Call(ctx, "workspace/executeCommand", params, &result); err != nil {
		return nil, fmt.Errorf("execute command failed: %w", err)
	}

	return result, nil
}

// PrepareRename checks if a symbol can be renamed
func (c *Client) PrepareRename(ctx context.Context, filepath string, line, character int) (*protocol.Range, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.PrepareRenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: protocol.DocumentURI(uri),
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}

	var result protocol.Range
	if err := c.Call(ctx, "textDocument/prepareRename", params, &result); err != nil {
		return nil, fmt.Errorf("prepare rename failed: %w", err)
	}

	return &result, nil
}

// Rename renames a symbol across the workspace
func (c *Client) Rename(ctx context.Context, filepath string, line, character int, newName string) (*protocol.WorkspaceEdit, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.RenameParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: protocol.DocumentURI(uri),
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
		NewName: newName,
	}

	var result protocol.WorkspaceEdit
	if err := c.Call(ctx, "textDocument/rename", params, &result); err != nil {
		return nil, fmt.Errorf("rename failed: %w", err)
	}

	return &result, nil
}

// GetSignatureHelp requests signature help at a specific position
func (c *Client) GetSignatureHelp(ctx context.Context, filepath string, line, character int) (*protocol.SignatureHelp, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.SignatureHelpParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: protocol.DocumentURI(uri),
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(character),
			},
		},
	}

	var result protocol.SignatureHelp
	if err := c.Call(ctx, "textDocument/signatureHelp", params, &result); err != nil {
		return nil, fmt.Errorf("signature help failed: %w", err)
	}

	return &result, nil
}

// UpdateFileContent updates the content of an open file and notifies the LSP server
func (c *Client) UpdateFileContent(ctx context.Context, filepath string, content []byte) error {
	uri := fmt.Sprintf("file://%s", filepath)

	c.openFilesMu.Lock()
	fileInfo, exists := c.openFiles[uri]
	if !exists {
		c.openFilesMu.Unlock()
		return fmt.Errorf("file not open: %s", filepath)
	}

	// Increment version
	fileInfo.Version++
	version := fileInfo.Version
	fileInfo.Content = string(content)
	c.openFilesMu.Unlock()

	params := protocol.DidChangeTextDocumentParams{
		TextDocument: protocol.VersionedTextDocumentIdentifier{
			TextDocumentIdentifier: protocol.TextDocumentIdentifier{
				URI: protocol.DocumentURI(uri),
			},
			Version: version,
		},
		ContentChanges: []protocol.TextDocumentContentChangeEvent{
			{
				Text: string(content),
			},
		},
	}

	return c.Notify(ctx, "textDocument/didChange", params)
}

// SaveFile notifies the LSP server that a file has been saved
func (c *Client) SaveFile(ctx context.Context, filepath string, content []byte) error {
	uri := fmt.Sprintf("file://%s", filepath)

	// First update the content if the file is open
	if c.IsFileOpen(filepath) {
		if err := c.UpdateFileContent(ctx, filepath, content); err != nil {
			return fmt.Errorf("failed to update file content: %w", err)
		}
	}

	// Send didSave notification
	params := protocol.DidSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(uri),
		},
		Text: string(content),
	}

	return c.Notify(ctx, "textDocument/didSave", params)
}

// WillSaveFile notifies the LSP server that a file is about to be saved
func (c *Client) WillSaveFile(ctx context.Context, filepath string, reason protocol.TextDocumentSaveReason) error {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.WillSaveTextDocumentParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(uri),
		},
		Reason: reason,
	}

	return c.Notify(ctx, "textDocument/willSave", params)
}

// GetFormattingEdits requests document formatting
func (c *Client) GetFormattingEdits(ctx context.Context, filepath string, options protocol.FormattingOptions) ([]protocol.TextEdit, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.DocumentFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(uri),
		},
		Options: options,
	}

	var result []protocol.TextEdit
	if err := c.Call(ctx, "textDocument/formatting", params, &result); err != nil {
		return nil, fmt.Errorf("formatting failed: %w", err)
	}

	return result, nil
}

// GetRangeFormattingEdits requests range formatting
func (c *Client) GetRangeFormattingEdits(ctx context.Context, filepath string, startLine, startChar, endLine, endChar int, options protocol.FormattingOptions) ([]protocol.TextEdit, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.DocumentRangeFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(uri),
		},
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(startLine),
				Character: uint32(startChar),
			},
			End: protocol.Position{
				Line:      uint32(endLine),
				Character: uint32(endChar),
			},
		},
		Options: options,
	}

	var result []protocol.TextEdit
	if err := c.Call(ctx, "textDocument/rangeFormatting", params, &result); err != nil {
		return nil, fmt.Errorf("range formatting failed: %w", err)
	}

	return result, nil
}

// GetOnTypeFormattingEdits requests on-type formatting
func (c *Client) GetOnTypeFormattingEdits(ctx context.Context, filepath string, line, character int, ch string, options protocol.FormattingOptions) ([]protocol.TextEdit, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.DocumentOnTypeFormattingParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(uri),
		},
		Position: protocol.Position{
			Line:      uint32(line),
			Character: uint32(character),
		},
		Ch:      ch,
		Options: options,
	}

	var result []protocol.TextEdit
	if err := c.Call(ctx, "textDocument/onTypeFormatting", params, &result); err != nil {
		return nil, fmt.Errorf("on-type formatting failed: %w", err)
	}

	return result, nil
}

// GetDocumentLinks requests document links for a file
func (c *Client) GetDocumentLinks(ctx context.Context, filepath string) ([]protocol.DocumentLink, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.DocumentLinkParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(uri),
		},
	}

	var result []protocol.DocumentLink
	if err := c.Call(ctx, "textDocument/documentLink", params, &result); err != nil {
		return nil, fmt.Errorf("document links failed: %w", err)
	}

	return result, nil
}

// GetFoldingRanges requests folding ranges for a file
func (c *Client) GetFoldingRanges(ctx context.Context, filepath string) ([]protocol.FoldingRange, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.FoldingRangeParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: protocol.DocumentURI(uri),
			},
		},
	}

	var result []protocol.FoldingRange
	if err := c.Call(ctx, "textDocument/foldingRange", params, &result); err != nil {
		return nil, fmt.Errorf("folding ranges failed: %w", err)
	}

	return result, nil
}

// GetSelectionRanges requests selection ranges for positions in a file
func (c *Client) GetSelectionRanges(ctx context.Context, filepath string, positions []protocol.Position) ([]protocol.SelectionRange, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.SelectionRangeParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(uri),
		},
		Positions: positions,
	}

	var result []protocol.SelectionRange
	if err := c.Call(ctx, "textDocument/selectionRange", params, &result); err != nil {
		return nil, fmt.Errorf("selection ranges failed: %w", err)
	}

	return result, nil
}

// GetSemanticTokens requests semantic tokens for a file
func (c *Client) GetSemanticTokens(ctx context.Context, filepath string) (*protocol.SemanticTokens, error) {
	uri := fmt.Sprintf("file://%s", filepath)

	params := protocol.SemanticTokensParams{
		TextDocument: protocol.TextDocumentIdentifier{
			URI: protocol.DocumentURI(uri),
		},
	}

	var result protocol.SemanticTokens
	if err := c.Call(ctx, "textDocument/semanticTokens/full", params, &result); err != nil {
		return nil, fmt.Errorf("semantic tokens failed: %w", err)
	}

	return &result, nil
}

// DetectLanguageID detects the language ID from a file URI
func DetectLanguageID(uri string) string {
	ext := strings.ToLower(filepath.Ext(uri))

	switch ext {
	case ".go":
		return "go"
	case ".rs":
		return "rust"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".jsx":
		return "javascriptreact"
	case ".java":
		return "java"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".c":
		return "c"
	case ".h", ".hpp":
		return "c"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".md":
		return "markdown"
	case ".sh":
		return "shellscript"
	case ".ps1":
		return "powershell"
	default:
		return "plaintext"
	}
}

// CloseAllFiles closes all open files
func (c *Client) CloseAllFiles(ctx context.Context) {
	c.openFilesMu.RLock()
	files := make([]string, 0, len(c.openFiles))
	for uri := range c.openFiles {
		// Convert URI back to filepath
		if filepath, found := strings.CutPrefix(uri, "file://"); found {
			files = append(files, filepath)
		}
	}
	c.openFilesMu.RUnlock()

	for _, filepath := range files {
		_ = c.CloseFile(ctx, filepath)
	}
}
