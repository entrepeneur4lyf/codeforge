package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/lsp"
	"go.lsp.dev/protocol"
)

type DiagnosticsParams struct {
	FilePath string `json:"file_path"`
}
type diagnosticsTool struct {
	lspClients map[string]*lsp.Client
}

const (
	DiagnosticsToolName    = "diagnostics"
	diagnosticsDescription = `Get diagnostics for a file and/or project.
WHEN TO USE THIS TOOL:
- Use when you need to check for errors or warnings in your code
- Helpful for debugging and ensuring code quality
- Good for getting a quick overview of issues in a file or project
HOW TO USE:
- Provide a path to a file to get diagnostics for that file
- Leave the path empty to get diagnostics for the entire project
- Results are displayed in a structured format with severity levels
FEATURES:
- Displays errors, warnings, and hints
- Groups diagnostics by severity
- Provides detailed information about each diagnostic
LIMITATIONS:
- Results are limited to the diagnostics provided by the LSP clients
- May not cover all possible issues in the code
- Does not provide suggestions for fixing issues
TIPS:
- Use in conjunction with other tools for a comprehensive code review
- Combine with the LSP client for real-time diagnostics
`
)

func NewDiagnosticsTool(lspClients map[string]*lsp.Client) BaseTool {
	return &diagnosticsTool{
		lspClients,
	}
}

func (b *diagnosticsTool) Info() ToolInfo {
	return ToolInfo{
		Name:        DiagnosticsToolName,
		Description: diagnosticsDescription,
		Parameters: map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The path to the file to get diagnostics for (leave w empty for project diagnostics)",
			},
		},
		Required: []string{},
	}
}

func (b *diagnosticsTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params DiagnosticsParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse(fmt.Sprintf("error parsing parameters: %s", err)), nil
	}

	lsps := b.lspClients

	if len(lsps) == 0 {
		return NewTextErrorResponse("no LSP clients available"), nil
	}

	if params.FilePath != "" {
		notifyLspOpenFile(ctx, params.FilePath, lsps)
		waitForLspDiagnostics(ctx, params.FilePath, lsps)
	}

	output := getDiagnostics(params.FilePath, lsps)

	return NewTextResponse(output), nil
}

func notifyLspOpenFile(ctx context.Context, filePath string, lsps map[string]*lsp.Client) {
	for _, client := range lsps {
		err := client.OpenFile(ctx, filePath)
		if err != nil {
			continue
		}
	}
}

func waitForLspDiagnostics(ctx context.Context, filePath string, lsps map[string]*lsp.Client) {
	if len(lsps) == 0 {
		return
	}

	diagChan := make(chan struct{}, 1)

	for _, client := range lsps {
		// We don't need to register a handler because the client already handles diagnostics internally

		if client.IsFileOpen(filePath) {
			// Just reopen the file to trigger diagnostics
			err := client.CloseFile(ctx, filePath)
			if err == nil {
				err = client.OpenFile(ctx, filePath)
			}
			if err != nil {
				continue
			}
		} else {
			err := client.OpenFile(ctx, filePath)
			if err != nil {
				continue
			}
		}
	}

	select {
	case <-diagChan:
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
	}
}


func getDiagnostics(filePath string, lsps map[string]*lsp.Client) string {
	fileDiagnostics := []string{}
	projectDiagnostics := []string{}

	formatDiagnostic := func(pth string, diagnostic protocol.Diagnostic, source string) string {
		severity := "Info"
		switch diagnostic.Severity {
		case protocol.DiagnosticSeverityError:
			severity = "Error"
		case protocol.DiagnosticSeverityWarning:
			severity = "Warn"
		case protocol.DiagnosticSeverityHint:
			severity = "Hint"
		}

		location := fmt.Sprintf("%s:%d:%d", pth, diagnostic.Range.Start.Line+1, diagnostic.Range.Start.Character+1)

		sourceInfo := ""
		if diagnostic.Source != "" {
			sourceInfo = diagnostic.Source
		} else if source != "" {
			sourceInfo = source
		}

		codeInfo := ""
		if diagnostic.Code != nil {
			codeInfo = fmt.Sprintf("[%v]", diagnostic.Code)
		}

		tagsInfo := ""
		if len(diagnostic.Tags) > 0 {
			tags := []string{}
			for _, tag := range diagnostic.Tags {
				switch tag {
				case 1: // Unnecessary
					tags = append(tags, "unnecessary")
				case 2: // Deprecated
					tags = append(tags, "deprecated")
				}
			}
			if len(tags) > 0 {
				tagsInfo = fmt.Sprintf(" (%s)", strings.Join(tags, ", "))
			}
		}

		return fmt.Sprintf("%s: %s [%s]%s%s %s",
			severity,
			location,
			sourceInfo,
			codeInfo,
			tagsInfo,
			diagnostic.Message)
	}

	// For now, only get diagnostics for the current file
	// TODO: Add support for getting all project diagnostics
	uri := fmt.Sprintf("file://%s", filePath)
	for lspName, client := range lsps {
		diagnostics := client.GetDiagnostics(uri)
		if len(diagnostics) > 0 {
			for _, diag := range diagnostics {
				formattedDiag := formatDiagnostic(filePath, diag, lspName)
				fileDiagnostics = append(fileDiagnostics, formattedDiag)
			}
		}
	}

	sort.Slice(fileDiagnostics, func(i, j int) bool {
		iIsError := strings.HasPrefix(fileDiagnostics[i], "Error")
		jIsError := strings.HasPrefix(fileDiagnostics[j], "Error")
		if iIsError != jIsError {
			return iIsError // Errors come first
		}
		return fileDiagnostics[i] < fileDiagnostics[j] // Then alphabetically
	})

	sort.Slice(projectDiagnostics, func(i, j int) bool {
		iIsError := strings.HasPrefix(projectDiagnostics[i], "Error")
		jIsError := strings.HasPrefix(projectDiagnostics[j], "Error")
		if iIsError != jIsError {
			return iIsError
		}
		return projectDiagnostics[i] < projectDiagnostics[j]
	})

	output := ""

	if len(fileDiagnostics) > 0 {
		output += "\n<file_diagnostics>\n"
		if len(fileDiagnostics) > 10 {
			output += strings.Join(fileDiagnostics[:10], "\n")
			output += fmt.Sprintf("\n... and %d more diagnostics", len(fileDiagnostics)-10)
		} else {
			output += strings.Join(fileDiagnostics, "\n")
		}
		output += "\n</file_diagnostics>\n"
	}

	if len(projectDiagnostics) > 0 {
		output += "\n<project_diagnostics>\n"
		if len(projectDiagnostics) > 10 {
			output += strings.Join(projectDiagnostics[:10], "\n")
			output += fmt.Sprintf("\n... and %d more diagnostics", len(projectDiagnostics)-10)
		} else {
			output += strings.Join(projectDiagnostics, "\n")
		}
		output += "\n</project_diagnostics>\n"
	}

	if len(fileDiagnostics) > 0 || len(projectDiagnostics) > 0 {
		fileErrors := countSeverity(fileDiagnostics, "Error")
		fileWarnings := countSeverity(fileDiagnostics, "Warn")
		projectErrors := countSeverity(projectDiagnostics, "Error")
		projectWarnings := countSeverity(projectDiagnostics, "Warn")

		output += "\n<diagnostic_summary>\n"
		output += fmt.Sprintf("Current file: %d errors, %d warnings\n", fileErrors, fileWarnings)
		output += fmt.Sprintf("Project: %d errors, %d warnings\n", projectErrors, projectWarnings)
		output += "</diagnostic_summary>\n"
	}

	return output
}

func countSeverity(diagnostics []string, severity string) int {
	count := 0
	for _, diag := range diagnostics {
		if strings.HasPrefix(diag, severity) {
			count++
		}
	}
	return count
}
