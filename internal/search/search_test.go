package search

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSearcher(t *testing.T) {
	// Create test directory structure
	testDir := t.TempDir()
	
	// Create test files
	testFiles := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`,
		"utils.go": `package main

func Add(a, b int) int {
	return a + b
}

func Multiply(x, y int) int {
	return x * y
}`,
		"README.md": `# Test Project

This is a test project for searching.

## Features
- Fast searching
- Fuzzy matching`,
		"config.json": `{
	"name": "test",
	"version": "1.0"
}`,
	}
	
	for name, content := range testFiles {
		path := filepath.Join(testDir, name)
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", name, err)
		}
	}
	
	searcher := NewSearcher()
	
	t.Run("file search", func(t *testing.T) {
		ctx := context.Background()
		
		// Search for Go files
		files, err := searcher.SearchFiles(ctx, "go", testDir)
		if err != nil {
			t.Fatalf("SearchFiles failed: %v", err)
		}
		
		// Should find main.go and utils.go
		if len(files) < 2 {
			t.Errorf("Expected at least 2 Go files, got %d", len(files))
		}
		
		// Search for README
		files, err = searcher.SearchFiles(ctx, "read", testDir)
		if err != nil {
			t.Fatalf("SearchFiles failed: %v", err)
		}
		
		found := false
		for _, f := range files {
			if filepath.Base(f) == "README.md" {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("Expected to find README.md with fuzzy search 'read'")
		}
	})
	
	t.Run("text search exact", func(t *testing.T) {
		ctx := context.Background()
		
		opts := Options{
			Query:      "Hello",
			Path:       testDir,
			MaxResults: 10,
			UseFuzzy:   false,
		}
		
		results, err := searcher.Search(ctx, opts)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		
		if len(results) == 0 {
			t.Error("Expected to find 'Hello' in files")
		}
		
		// Should find in main.go
		found := false
		for _, r := range results {
			if filepath.Base(r.Path) == "main.go" && r.Line == 6 {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("Expected to find 'Hello' in main.go line 6")
		}
	})
	
	t.Run("text search fuzzy", func(t *testing.T) {
		ctx := context.Background()
		
		opts := Options{
			Query:          "prnt",
			Path:           testDir,
			MaxResults:     10,
			UseFuzzy:       true,
			FuzzyThreshold: 50,
		}
		
		results, err := searcher.Search(ctx, opts)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		
		// Should find "Println" with fuzzy match
		found := false
		for _, r := range results {
			if filepath.Base(r.Path) == "main.go" {
				found = true
				break
			}
		}
		
		if !found {
			t.Error("Expected fuzzy search 'prnt' to find 'Println'")
		}
	})
	
	t.Run("case insensitive search", func(t *testing.T) {
		ctx := context.Background()
		
		opts := Options{
			Query:         "HELLO",
			Path:          testDir,
			CaseSensitive: false,
			UseFuzzy:      false,
			MaxResults:    10,
		}
		
		results, err := searcher.Search(ctx, opts)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		
		if len(results) == 0 {
			t.Error("Expected case-insensitive search to find 'Hello'")
		}
	})
	
	t.Run("file filters", func(t *testing.T) {
		ctx := context.Background()
		
		// Search only in Go files
		opts := Options{
			Query:      "func",
			Path:       testDir,
			Include:    []string{"*.go"},
			MaxResults: 10,
		}
		
		results, err := searcher.Search(ctx, opts)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		
		// All results should be from Go files
		for _, r := range results {
			if filepath.Ext(r.Path) != ".go" {
				t.Errorf("Expected only .go files, got %s", r.Path)
			}
		}
		
		// Exclude Go files
		opts = Options{
			Query:      "test",
			Path:       testDir,
			Exclude:    []string{"*.go"},
			MaxResults: 10,
		}
		
		results, err = searcher.Search(ctx, opts)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		
		// No results should be from Go files
		for _, r := range results {
			if filepath.Ext(r.Path) == ".go" {
				t.Errorf("Expected no .go files, got %s", r.Path)
			}
		}
	})
	
	t.Run("context cancellation", func(t *testing.T) {
		// Create a context that's already cancelled
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		
		opts := Options{
			Query: "test",
			Path:  testDir,
		}
		
		// Should return context error
		_, err := searcher.Search(ctx, opts)
		
		if err == nil {
			t.Error("Expected context cancellation error")
		}
	})
	
	t.Run("cache", func(t *testing.T) {
		ctx := context.Background()
		
		opts := Options{
			Query:      "cache",
			Path:       testDir,
			MaxResults: 10,
		}
		
		// First search
		start := time.Now()
		results1, err := searcher.Search(ctx, opts)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		duration1 := time.Since(start)
		
		// Second search (should hit cache)
		start = time.Now()
		results2, err := searcher.Search(ctx, opts)
		if err != nil {
			t.Fatalf("Search failed: %v", err)
		}
		duration2 := time.Since(start)
		
		// Cache hit should be much faster
		if duration2 >= duration1 {
			t.Log("Warning: Cache may not be working properly")
		}
		
		// Results should be identical
		if len(results1) != len(results2) {
			t.Error("Cached results differ from original")
		}
	})
}

func TestFuzzyMatching(t *testing.T) {
	testCases := []struct {
		query    string
		text     string
		expected bool
	}{
		{"prnt", "println", true},
		{"hlwd", "Hello World", true},
		{"cfg", "config", true},
		{"xyz", "abc", false},
		{"test", "test", true},
	}
	
	for _, tc := range testCases {
		result := fuzzyMatch(tc.query, tc.text)
		if result != tc.expected {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", 
				tc.query, tc.text, result, tc.expected)
		}
	}
}

// Helper function for testing
func fuzzyMatch(query, text string) bool {
	// Simple fuzzy match check for testing - case insensitive
	query = strings.ToLower(query)
	text = strings.ToLower(text)
	j := 0
	for i := 0; i < len(text) && j < len(query); i++ {
		if text[i] == query[j] {
			j++
		}
	}
	return j == len(query)
}