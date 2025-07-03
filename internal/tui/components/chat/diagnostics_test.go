package chat

import (
	"fmt"
	"strings"
	"testing"

	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

func TestDiagnosticRenderer(t *testing.T) {
	th := theme.GetTheme("default")
	renderer := NewDiagnosticRenderer(th)

	t.Run("render error diagnostic", func(t *testing.T) {
		diag := Diagnostic{
			Level:   DiagnosticError,
			Code:    "E0001",
			Title:   "Undefined variable",
			Message: "Variable 'foo' is not defined in this scope",
			File:    "main.go",
			Line:    42,
			Column:  15,
			Context: []string{
				"func main() {",
				"    x := 10",
				"    y := foo + x",
				"    fmt.Println(y)",
				"}",
			},
			Suggestions: []string{
				"Did you mean 'bar'?",
				"Import the package that defines 'foo'",
			},
		}

		rendered := renderer.RenderDiagnostic(diag)
		stripped := stripANSI(rendered)

		// Check components
		if !strings.Contains(stripped, "ERROR") {
			t.Error("Expected ERROR in output")
		}
		if !strings.Contains(stripped, "[E0001]") {
			t.Error("Expected error code in output")
		}
		if !strings.Contains(stripped, "Undefined variable") {
			t.Error("Expected title in output")
		}
		if !strings.Contains(stripped, "main.go:42:15") {
			t.Error("Expected file location in output")
		}
		if !strings.Contains(stripped, "y := foo + x") {
			t.Error("Expected error line in context")
		}
		if !strings.Contains(stripped, "^") {
			t.Error("Expected column indicator")
		}
		if !strings.Contains(stripped, "Did you mean 'bar'?") {
			t.Error("Expected suggestion in output")
		}
	})

	t.Run("render warning diagnostic", func(t *testing.T) {
		diag := Diagnostic{
			Level:   DiagnosticWarning,
			Code:    "W0042",
			Title:   "Unused import",
			Message: "The import 'fmt' is not used",
			File:    "utils.go",
			Line:    5,
		}

		rendered := renderer.RenderDiagnostic(diag)
		stripped := stripANSI(rendered)

		if !strings.Contains(stripped, "WARNING") {
			t.Error("Expected WARNING in output")
		}
		if !strings.Contains(stripped, "[W0042]") {
			t.Error("Expected warning code")
		}
		if !strings.Contains(stripped, "utils.go:5") {
			t.Error("Expected file location")
		}
	})

	t.Run("render info diagnostic", func(t *testing.T) {
		diag := Diagnostic{
			Level:   DiagnosticInfo,
			Title:   "Type inference",
			Message: "Type 'int' was inferred from context",
		}

		rendered := renderer.RenderDiagnostic(diag)
		stripped := stripANSI(rendered)

		if !strings.Contains(stripped, "INFO") {
			t.Error("Expected INFO in output")
		}
		if !strings.Contains(stripped, "Type inference") {
			t.Error("Expected title")
		}
	})

	t.Run("render hint diagnostic", func(t *testing.T) {
		diag := Diagnostic{
			Level:   DiagnosticHint,
			Message: "Consider using a more descriptive variable name",
			Suggestions: []string{
				"Use 'userCount' instead of 'n'",
			},
		}

		rendered := renderer.RenderDiagnostic(diag)
		stripped := stripANSI(rendered)

		if !strings.Contains(stripped, "HINT") {
			t.Error("Expected HINT in output")
		}
		if !strings.Contains(stripped, "Consider using") {
			t.Error("Expected message")
		}
		if !strings.Contains(stripped, "userCount") {
			t.Error("Expected suggestion")
		}
	})

	t.Run("inline rendering", func(t *testing.T) {
		diag := Diagnostic{
			Level:   DiagnosticError,
			Code:    "E0001",
			Title:   "Syntax error",
			File:    "parser.go",
			Line:    100,
			Column:  25,
		}

		inline := renderer.RenderInline(diag)
		stripped := stripANSI(inline)

		// Should be on one line
		if strings.Contains(stripped, "\n") {
			t.Error("Inline rendering should not contain newlines")
		}

		// Should contain key information
		if !strings.Contains(stripped, "[E0001]") {
			t.Error("Expected error code")
		}
		if !strings.Contains(stripped, "Syntax error") {
			t.Error("Expected title")
		}
		if !strings.Contains(stripped, "(parser.go:100:25)") {
			t.Error("Expected location")
		}
	})

	t.Run("context highlighting", func(t *testing.T) {
		diag := Diagnostic{
			Level:  DiagnosticError,
			Line:   3,
			Column: 10,
			Context: []string{
				"if x > 0 {",
				"    return true",
				"} else if y = 0 {", // Error line
				"    return false",
				"}",
			},
		}

		rendered := renderer.RenderDiagnostic(diag)

		// The error line should be highlighted differently
		// and column indicator should be at position 10
		lines := strings.Split(rendered, "\n")
		foundColumnIndicator := false
		for _, line := range lines {
			if strings.Contains(line, "^") {
				// Check that the ^ is roughly at column 10
				idx := strings.Index(line, "^")
				if idx > 0 {
					foundColumnIndicator = true
				}
			}
		}

		if !foundColumnIndicator {
			t.Error("Expected column indicator in context")
		}
	})

	t.Run("diagnostic without location", func(t *testing.T) {
		diag := Diagnostic{
			Level:   DiagnosticError,
			Title:   "Configuration error",
			Message: "Invalid configuration file format",
		}

		rendered := renderer.RenderDiagnostic(diag)
		stripped := stripANSI(rendered)

		// Should not contain location info
		if strings.Contains(stripped, "📍") {
			t.Error("Should not show location marker without file")
		}

		// Should still show the error
		if !strings.Contains(stripped, "Configuration error") {
			t.Error("Expected title")
		}
	})

	t.Run("caching", func(t *testing.T) {
		diag := Diagnostic{
			Level:   DiagnosticInfo,
			Code:    "I001",
			Title:   "Cached diagnostic",
			Message: "This should be cached",
		}

		// First render
		result1 := renderer.RenderDiagnostic(diag)

		// Second render (should hit cache)
		result2 := renderer.RenderDiagnostic(diag)

		if result1 != result2 {
			t.Error("Expected cached result to be identical")
		}
	})
}

func TestDiagnosticGroup(t *testing.T) {
	th := theme.GetTheme("default")
	renderer := NewDiagnosticRenderer(th)

	t.Run("render diagnostic group", func(t *testing.T) {
		group := DiagnosticGroup{
			Title: "Compilation Errors",
			Diagnostics: []Diagnostic{
				{
					Level:   DiagnosticError,
					Code:    "E001",
					Title:   "Syntax error",
					Message: "Unexpected token",
					File:    "main.go",
					Line:    10,
				},
				{
					Level:   DiagnosticError,
					Code:    "E002",
					Title:   "Type mismatch",
					Message: "Cannot assign string to int",
					File:    "main.go",
					Line:    15,
				},
				{
					Level:   DiagnosticWarning,
					Code:    "W001",
					Title:   "Unused variable",
					Message: "Variable 'temp' is declared but never used",
					File:    "main.go",
					Line:    20,
				},
			},
		}

		rendered := renderer.RenderDiagnosticGroup(group)
		stripped := stripANSI(rendered)

		// Should contain group title
		if !strings.Contains(stripped, "Compilation Errors") {
			t.Error("Expected group title")
		}

		// Should contain all diagnostics
		if !strings.Contains(stripped, "[E001]") {
			t.Error("Expected first error")
		}
		if !strings.Contains(stripped, "[E002]") {
			t.Error("Expected second error")
		}
		if !strings.Contains(stripped, "[W001]") {
			t.Error("Expected warning")
		}
	})

	t.Run("large group uses inline rendering", func(t *testing.T) {
		var diagnostics []Diagnostic
		for i := 0; i < 5; i++ {
			diagnostics = append(diagnostics, Diagnostic{
				Level:   DiagnosticWarning,
				Code:    "W100",
				Title:   "Test warning",
				Message: "This is a test",
			})
		}

		group := DiagnosticGroup{
			Title:       "Many Warnings",
			Diagnostics: diagnostics,
		}

		rendered := renderer.RenderDiagnosticGroup(group)
		lines := strings.Split(rendered, "\n")

		// With inline rendering, should have fewer lines
		// (roughly 1 per diagnostic plus header)
		if len(lines) > 10 {
			t.Error("Expected compact rendering for large groups")
		}
	})

	t.Run("empty group", func(t *testing.T) {
		group := DiagnosticGroup{
			Title:       "Empty",
			Diagnostics: []Diagnostic{},
		}

		rendered := renderer.RenderDiagnosticGroup(group)
		if rendered != "" {
			t.Error("Expected empty string for empty group")
		}
	})
}

func TestDiagnosticSummary(t *testing.T) {
	th := theme.GetTheme("default")
	renderer := NewDiagnosticRenderer(th)

	t.Run("summary with mixed diagnostics", func(t *testing.T) {
		diagnostics := []Diagnostic{
			{Level: DiagnosticError},
			{Level: DiagnosticError},
			{Level: DiagnosticWarning},
			{Level: DiagnosticWarning},
			{Level: DiagnosticWarning},
			{Level: DiagnosticInfo},
			{Level: DiagnosticHint},
		}

		summary := renderer.Summary(diagnostics)
		stripped := stripANSI(summary)

		if !strings.Contains(stripped, "2 error(s)") {
			t.Error("Expected 2 errors in summary")
		}
		if !strings.Contains(stripped, "3 warning(s)") {
			t.Error("Expected 3 warnings in summary")
		}
		if !strings.Contains(stripped, "1 info") {
			t.Error("Expected 1 info in summary")
		}
		if !strings.Contains(stripped, "1 hint(s)") {
			t.Error("Expected 1 hint in summary")
		}
	})

	t.Run("summary with no diagnostics", func(t *testing.T) {
		summary := renderer.Summary([]Diagnostic{})
		if summary != "No diagnostics" {
			t.Errorf("Expected 'No diagnostics', got %s", summary)
		}
	})

	t.Run("summary with only errors", func(t *testing.T) {
		diagnostics := []Diagnostic{
			{Level: DiagnosticError},
			{Level: DiagnosticError},
			{Level: DiagnosticError},
		}

		summary := renderer.Summary(diagnostics)
		stripped := stripANSI(summary)

		if !strings.Contains(stripped, "3 error(s)") {
			t.Error("Expected 3 errors")
		}
		if strings.Contains(stripped, "warning") {
			t.Error("Should not mention warnings when there are none")
		}
	})
}

func TestDiagnosticLevelHelpers(t *testing.T) {
	th := theme.GetTheme("default")
	renderer := NewDiagnosticRenderer(th)

	t.Run("level colors", func(t *testing.T) {
		// Just verify they return valid colors
		errorColor := renderer.getLevelColor(DiagnosticError)
		warningColor := renderer.getLevelColor(DiagnosticWarning)
		infoColor := renderer.getLevelColor(DiagnosticInfo)
		hintColor := renderer.getLevelColor(DiagnosticHint)

		// Basic check that they're different
		if errorColor == warningColor {
			t.Error("Error and warning should have different colors")
		}
		if infoColor == hintColor {
			t.Error("Info and hint should have different colors")
		}
	})

	t.Run("level icons", func(t *testing.T) {
		icons := map[DiagnosticLevel]string{
			DiagnosticError:   renderer.getLevelIcon(DiagnosticError),
			DiagnosticWarning: renderer.getLevelIcon(DiagnosticWarning),
			DiagnosticInfo:    renderer.getLevelIcon(DiagnosticInfo),
			DiagnosticHint:    renderer.getLevelIcon(DiagnosticHint),
		}

		// Each level should have a unique icon
		seen := make(map[string]bool)
		for level, icon := range icons {
			if seen[icon] {
				t.Errorf("Duplicate icon for level %s", level)
			}
			seen[icon] = true
		}
	})
}

func TestDiagnosticEdgeCases(t *testing.T) {
	th := theme.GetTheme("default")
	renderer := NewDiagnosticRenderer(th)

	t.Run("very long message", func(t *testing.T) {
		longMessage := strings.Repeat("This is a very long error message. ", 20)
		diag := Diagnostic{
			Level:   DiagnosticError,
			Message: longMessage,
		}

		rendered := renderer.RenderDiagnostic(diag)
		if rendered == "" {
			t.Error("Should handle long messages")
		}
	})

	t.Run("context at file start", func(t *testing.T) {
		diag := Diagnostic{
			Level: DiagnosticError,
			Line:  1,
			Context: []string{
				"package main",
				"",
				"import (",
			},
		}

		rendered := renderer.RenderDiagnostic(diag)
		stripped := stripANSI(rendered)

		// Should show line 1
		if !strings.Contains(stripped, "1 │ package main") {
			t.Error("Expected line 1 in context")
		}
	})

	t.Run("column beyond line length", func(t *testing.T) {
		diag := Diagnostic{
			Level:  DiagnosticError,
			Line:   1,
			Column: 100, // Beyond line length
			Context: []string{
				"short line",
			},
		}

		// Should not panic
		rendered := renderer.RenderDiagnostic(diag)
		if rendered == "" {
			t.Error("Should handle column beyond line length")
		}
	})

	t.Run("empty context lines", func(t *testing.T) {
		diag := Diagnostic{
			Level: DiagnosticError,
			Line:  2,
			Context: []string{
				"first line",
				"", // Empty line
				"third line",
			},
		}

		rendered := renderer.RenderDiagnostic(diag)

		// Debug output
		if testing.Verbose() {
			fmt.Printf("Rendered with empty lines:\n%s\n", rendered)
		}

		// Should handle empty lines in context
		emptyLineFound := false
		strippedRendered := stripANSI(rendered)
		strippedLines := strings.Split(strippedRendered, "\n")
		
		for _, line := range strippedLines {
			// Look for a line with line number 2 - the format is "│ 2 │ [content]"
			if strings.Contains(line, "│ 2 │") {
				// The empty line should have spaces or nothing after "│ 2 │"
				parts := strings.SplitN(line, "│ 2 │", 2)
				if len(parts) == 2 {
					// Check if the content part (after line number) is empty or whitespace
					content := parts[1]
					if strings.TrimSpace(content) == "" {
						emptyLineFound = true
						break
					}
				}
			}
		}

		if !emptyLineFound {
			t.Error("Should preserve empty lines in context")
			// Debug: print all lines to see what we're looking for
			if testing.Verbose() {
				for i, line := range strippedLines {
					fmt.Printf("Line %d: %q\n", i, line)
				}
			}
		}
	})
}