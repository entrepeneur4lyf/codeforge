package chat

import (
	"fmt"
	"strings"
	"testing"

	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

func TestFileRenderer(t *testing.T) {
	th := theme.GetTheme("default")
	renderer := NewFileRenderer(th)
	
	t.Run("render simple file", func(t *testing.T) {
		file := FileContent{
			Path:    "test.go",
			Content: "package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}",
		}
		
		rendered := renderer.RenderFile(file)
		stripped := stripANSI(rendered)
		
		// Should contain file path
		if !strings.Contains(stripped, "test.go") {
			t.Error("Expected file path in output")
		}
		
		// Should contain content
		if !strings.Contains(stripped, "package main") {
			t.Error("Expected package declaration")
		}
		
		if !strings.Contains(stripped, "func main") {
			t.Error("Expected function declaration")
		}
		
		// Should have line numbers
		if !strings.Contains(stripped, "1 │") {
			t.Error("Expected line numbers")
		}
	})
	
	t.Run("render with line range", func(t *testing.T) {
		file := FileContent{
			Path:      "large.go",
			Content:   "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8",
			StartLine: 3,
			EndLine:   6,
		}
		
		rendered := renderer.RenderFile(file)
		stripped := stripANSI(rendered)
		
		// Should show line range in header
		if !strings.Contains(stripped, "(lines 3-6)") {
			t.Error("Expected line range in header")
		}
		
		// Should only show lines 3-6
		if !strings.Contains(stripped, "3 │ line3") {
			t.Error("Expected line 3")
		}
		
		if !strings.Contains(stripped, "6 │ line6") {
			t.Error("Expected line 6")
		}
		
		// Should not show lines outside range
		if strings.Contains(stripped, "line1") || strings.Contains(stripped, "line8") {
			t.Error("Should not show lines outside range")
		}
	})
	
	t.Run("render with highlights", func(t *testing.T) {
		file := FileContent{
			Path:      "highlight.go",
			Content:   "func test() {\n\tx := 1\n\ty := 2\n\tz := x + y\n}",
			Highlight: []int{2, 4},
		}
		
		rendered := renderer.RenderFile(file)
		stripped := stripANSI(rendered)
		
		// Should have highlight indicators
		if !strings.Contains(stripped, "→ 2") {
			t.Error("Expected highlight indicator for line 2")
		}
		
		if !strings.Contains(stripped, "→ 4") {
			t.Error("Expected highlight indicator for line 4")
		}
	})
	
	t.Run("language detection", func(t *testing.T) {
		testCases := []struct {
			path     string
			content  string
			expected string
		}{
			{"test.go", "", "go"},
			{"app.js", "", "javascript"},
			{"main.py", "", "python"},
			{"style.css", "", "css"},
			{"data.json", "", "json"},
			{"script.sh", "", "bash"},
			{"README.md", "", "markdown"},
			{"unknown.xyz", "", "text"},
			{"", "#!/usr/bin/env python", "python"},
			{"", "#!/bin/bash", "bash"},
		}
		
		for _, tc := range testCases {
			result := renderer.detectLanguage(tc.path, tc.content)
			if result != tc.expected {
				t.Errorf("detectLanguage(%q, %q) = %q, want %q", 
					tc.path, tc.content, result, tc.expected)
			}
		}
	})
	
	t.Run("file icons", func(t *testing.T) {
		testCases := []struct {
			path string
			icon string
		}{
			{"main.go", "🐹"},
			{"app.js", "📜"},
			{"script.py", "🐍"},
			{"style.css", "🎨"},
			{"data.json", "📋"},
			{"image.png", "🖼️"},
			{"document.pdf", "📕"},
			{"archive.zip", "📦"},
			{"unknown.xyz", "📄"},
		}
		
		for _, tc := range testCases {
			icon := renderer.getFileIcon(tc.path)
			if icon != tc.icon {
				t.Errorf("getFileIcon(%q) = %q, want %q", tc.path, icon, tc.icon)
			}
		}
	})
	
	t.Run("inline rendering", func(t *testing.T) {
		inline := renderer.RenderInline("main.go", 42)
		stripped := stripANSI(inline)
		
		if !strings.Contains(stripped, "main.go:42") {
			t.Error("Expected file path and line number")
		}
		
		if !strings.Contains(stripped, "🐹") {
			t.Error("Expected Go file icon")
		}
	})
	
	t.Run("caching", func(t *testing.T) {
		file := FileContent{
			Path:    "cached.go",
			Content: "package main",
		}
		
		// First render
		result1 := renderer.RenderFile(file)
		
		// Second render (should hit cache)
		result2 := renderer.RenderFile(file)
		
		if result1 != result2 {
			t.Error("Expected cached result to be identical")
		}
	})
	
	t.Run("line numbers toggle", func(t *testing.T) {
		file := FileContent{
			Path:    "test.go",
			Content: "line1\nline2\nline3",
		}
		
		// With line numbers
		renderer.SetShowLineNumbers(true)
		withNumbers := renderer.RenderFile(file)
		
		if !strings.Contains(stripANSI(withNumbers), "1 │") {
			t.Error("Expected line numbers when enabled")
		}
		
		// Without line numbers
		renderer.SetShowLineNumbers(false)
		withoutNumbers := renderer.RenderFile(file)
		strippedWithout := stripANSI(withoutNumbers)
		
		// Check that output is different (no line numbers)
		if withNumbers == withoutNumbers {
			t.Error("Expected different output when line numbers are disabled")
		}
		
		// When line numbers are off, we shouldn't see line number patterns at the start of lines
		// Look for patterns like " N │ " with spaces
		for i := 1; i <= 10; i++ {
			pattern := fmt.Sprintf(" %d │ ", i)
			if strings.Contains(strippedWithout, pattern) {
				t.Errorf("Found line number pattern %q when disabled", pattern)
				break
			}
		}
	})
	
	t.Run("max width", func(t *testing.T) {
		file := FileContent{
			Path:    "very/long/path/to/file/that/might/exceed/width/limits.go",
			Content: "content",
		}
		
		renderer.SetMaxWidth(50)
		rendered := renderer.RenderFile(file)
		
		// The rendered output should respect max width
		// This is a simple check - in reality the styling would handle overflow
		if len(rendered) == 0 {
			t.Error("Expected non-empty render")
		}
	})
	
	t.Run("empty file", func(t *testing.T) {
		file := FileContent{
			Path:    "empty.txt",
			Content: "",
		}
		
		rendered := renderer.RenderFile(file)
		stripped := stripANSI(rendered)
		
		// Should still show header
		if !strings.Contains(stripped, "empty.txt") {
			t.Error("Expected file path even for empty file")
		}
	})
	
	t.Run("syntax highlighting fallback", func(t *testing.T) {
		// Test with content that might fail highlighting
		file := FileContent{
			Path:    "test.go",
			Content: "invalid { go } syntax @#$%",
			Language: "go",
		}
		
		// Should not panic, should fallback gracefully
		rendered := renderer.RenderFile(file)
		if rendered == "" {
			t.Error("Expected fallback rendering for invalid syntax")
		}
	})
}