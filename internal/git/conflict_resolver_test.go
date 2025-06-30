package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConflictDetection(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "conflict_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a mock conflict file
	conflictContent := `package main

import "fmt"

func main() {
<<<<<<< HEAD
	fmt.Println("Hello from main branch")
	fmt.Println("This is the current version")
=======
	fmt.Println("Hello from feature branch")
	fmt.Println("This is the incoming version")
>>>>>>> feature-branch
	fmt.Println("Common code after conflict")
}
`

	conflictFile := filepath.Join(tempDir, "main.go")
	err = os.WriteFile(conflictFile, []byte(conflictContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write conflict file: %v", err)
	}

	// Test parseConflictFile
	repo := &Repository{workingDir: tempDir}
	conflict, err := repo.ParseConflictFile(context.Background(), "main.go")
	if err != nil {
		t.Fatalf("Failed to parse conflict file: %v", err)
	}

	// Verify conflict detection
	if conflict.FilePath != "main.go" {
		t.Errorf("Expected file path 'main.go', got '%s'", conflict.FilePath)
	}

	if len(conflict.Conflicts) != 1 {
		t.Errorf("Expected 1 conflict section, got %d", len(conflict.Conflicts))
	}

	section := conflict.Conflicts[0]
	if !strings.Contains(section.CurrentCode, "Hello from main branch") {
		t.Errorf("Current code not parsed correctly: %s", section.CurrentCode)
	}

	if !strings.Contains(section.IncomingCode, "Hello from feature branch") {
		t.Errorf("Incoming code not parsed correctly: %s", section.IncomingCode)
	}
}

func TestConflictResolutionGeneration(t *testing.T) {
	// Create test conflict
	conflict := GitConflict{
		FilePath:     "test.go",
		ConflictType: "merge",
		Conflicts: []ConflictSection{
			{
				StartLine:    5,
				EndLine:      10,
				CurrentCode:  `fmt.Println("Version A")`,
				IncomingCode: `fmt.Println("Version B")`,
				Context:      "func main() {\n\t// context\n}",
			},
		},
		FileContent: `package main
import "fmt"
func main() {
<<<<<<< HEAD
	fmt.Println("Version A")
=======
	fmt.Println("Version B")
>>>>>>> branch
}`,
	}

	// Test resolution generation without actual LLM call
	resolver := &ConflictResolver{}

	// Test generateResolvedContent with different strategies
	resolutions := []SectionResolution{
		{
			SectionIndex: 0,
			Resolution:   "keep_current",
			Reasoning:    "Keep current version",
		},
	}

	resolvedContent, err := resolver.generateResolvedContent(conflict, resolutions)
	if err != nil {
		t.Fatalf("Failed to generate resolved content: %v", err)
	}

	if !strings.Contains(resolvedContent, `fmt.Println("Version A")`) {
		t.Errorf("Resolved content should contain current version")
	}

	if strings.Contains(resolvedContent, "<<<<<<< HEAD") {
		t.Errorf("Resolved content should not contain conflict markers")
	}

	// Test merge_both strategy
	resolutions[0].Resolution = "merge_both"
	resolvedContent, err = resolver.generateResolvedContent(conflict, resolutions)
	if err != nil {
		t.Fatalf("Failed to generate resolved content for merge_both: %v", err)
	}

	if !strings.Contains(resolvedContent, `fmt.Println("Version A")`) {
		t.Errorf("Resolved content should contain current version")
	}

	if !strings.Contains(resolvedContent, `fmt.Println("Version B")`) {
		t.Errorf("Resolved content should contain incoming version")
	}
}

func TestConflictAnalysis(t *testing.T) {
	conflict := GitConflict{
		FilePath:     "example.go",
		ConflictType: "merge",
		Conflicts: []ConflictSection{
			{
				StartLine:    10,
				EndLine:      15,
				CurrentCode:  "// Current implementation",
				IncomingCode: "// Incoming implementation",
				BaseCode:     "// Base implementation",
				Context:      "func example() {\n\t// context\n}",
			},
		},
	}

	resolver := &ConflictResolver{}
	analysis := resolver.analyzeConflicts(conflict)

	// Verify analysis contains expected sections
	if !strings.Contains(analysis, "CONFLICT ANALYSIS") {
		t.Errorf("Analysis should contain header")
	}

	if !strings.Contains(analysis, "example.go") {
		t.Errorf("Analysis should contain file path")
	}

	if !strings.Contains(analysis, "CURRENT BRANCH") {
		t.Errorf("Analysis should contain current branch section")
	}

	if !strings.Contains(analysis, "INCOMING BRANCH") {
		t.Errorf("Analysis should contain incoming branch section")
	}

	if !strings.Contains(analysis, "COMMON ANCESTOR") {
		t.Errorf("Analysis should contain base code section when available")
	}

	if !strings.Contains(analysis, "SURROUNDING CONTEXT") {
		t.Errorf("Analysis should contain context section")
	}
}

func TestConflictTypeDetection(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "conflict_type_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	err = os.MkdirAll(gitDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create .git dir: %v", err)
	}

	repo := &Repository{workingDir: tempDir}

	// Test merge detection
	mergeHead := filepath.Join(gitDir, "MERGE_HEAD")
	err = os.WriteFile(mergeHead, []byte("commit-hash"), 0644)
	if err != nil {
		t.Fatalf("Failed to create MERGE_HEAD: %v", err)
	}

	conflictType := repo.determineConflictType(context.Background())
	if conflictType != "merge" {
		t.Errorf("Expected 'merge', got '%s'", conflictType)
	}

	// Clean up and test rebase detection
	os.Remove(mergeHead)
	rebaseDir := filepath.Join(gitDir, "rebase-merge")
	err = os.MkdirAll(rebaseDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create rebase-merge dir: %v", err)
	}

	conflictType = repo.determineConflictType(context.Background())
	if conflictType != "rebase" {
		t.Errorf("Expected 'rebase', got '%s'", conflictType)
	}

	// Clean up and test cherry-pick detection
	os.RemoveAll(rebaseDir)
	cherryPickHead := filepath.Join(gitDir, "CHERRY_PICK_HEAD")
	err = os.WriteFile(cherryPickHead, []byte("commit-hash"), 0644)
	if err != nil {
		t.Fatalf("Failed to create CHERRY_PICK_HEAD: %v", err)
	}

	conflictType = repo.determineConflictType(context.Background())
	if conflictType != "cherry-pick" {
		t.Errorf("Expected 'cherry-pick', got '%s'", conflictType)
	}

	// Clean up and test unknown
	os.Remove(cherryPickHead)
	conflictType = repo.determineConflictType(context.Background())
	if conflictType != "unknown" {
		t.Errorf("Expected 'unknown', got '%s'", conflictType)
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test max function
	if max(5, 3) != 5 {
		t.Errorf("max(5, 3) should return 5")
	}
	if max(2, 8) != 8 {
		t.Errorf("max(2, 8) should return 8")
	}

	// Test min function
	if min(5, 3) != 3 {
		t.Errorf("min(5, 3) should return 3")
	}
	if min(2, 8) != 2 {
		t.Errorf("min(2, 8) should return 2")
	}
}
