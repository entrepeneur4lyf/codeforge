package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/entrepeneur4lyf/codeforge/internal/git"
)

func main() {
	// Test AI commit message generation
	fmt.Println("🧪 Testing AI Commit Message Generation")
	fmt.Println("=====================================")

	// Get current working directory
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Create repository instance
	repo := git.NewRepository(workingDir)

	// Check if this is a git repository
	if !repo.IsGitRepository() {
		fmt.Println("❌ This directory is not a git repository")
		fmt.Println("💡 Initialize a git repository first with: git init")
		return
	}

	// Check if git is installed
	if !git.IsGitInstalled() {
		fmt.Println("❌ Git is not installed on this system")
		return
	}

	fmt.Printf("📁 Working directory: %s\n", workingDir)
	fmt.Println()

	// Test 1: Check git status
	fmt.Println("🔍 Checking git status...")
	status, err := repo.GetStatus(context.Background())
	if err != nil {
		log.Printf("Failed to get git status: %v", err)
		return
	}

	fmt.Printf("📊 Git Status:\n")
	fmt.Printf("  Branch: %s\n", status.Branch)
	fmt.Printf("  Status: %s\n", status.Status)
	fmt.Printf("  Modified files: %d\n", len(status.Modified))
	fmt.Printf("  Untracked files: %d\n", len(status.Untracked))
	fmt.Printf("  Staged files: %d\n", len(status.Staged))
	fmt.Println()

	// Test 2: Get git diff
	fmt.Println("📝 Getting git diff...")
	diffs, err := repo.GetDiff(context.Background(), false) // Get unstaged changes
	if err != nil {
		log.Printf("Failed to get git diff: %v", err)
		return
	}

	if len(diffs) == 0 {
		fmt.Println("ℹ️  No changes detected in working directory")
		
		// Try staged changes
		stagedDiffs, err := repo.GetDiff(context.Background(), true)
		if err != nil {
			log.Printf("Failed to get staged diff: %v", err)
			return
		}
		
		if len(stagedDiffs) == 0 {
			fmt.Println("ℹ️  No staged changes detected either")
			fmt.Println("💡 Make some changes to files and try again")
			return
		}
		
		fmt.Printf("✅ Found %d staged changes\n", len(stagedDiffs))
		diffs = stagedDiffs
	} else {
		fmt.Printf("✅ Found %d unstaged changes\n", len(diffs))
	}

	// Show diff summary
	for i, diff := range diffs {
		if i >= 3 { // Limit output
			fmt.Printf("  ... and %d more files\n", len(diffs)-i)
			break
		}
		fmt.Printf("  %s: %s (+%d -%d)\n", diff.Status, diff.FilePath, diff.Additions, diff.Deletions)
	}
	fmt.Println()

	// Test 3: Generate AI commit message
	fmt.Println("🤖 Generating AI commit message...")
	
	// Check if we have API keys for LLM providers
	hasAPIKey := false
	providers := []string{"ANTHROPIC_API_KEY", "OPENAI_API_KEY", "GROQ_API_KEY", "DEEPSEEK_API_KEY"}
	for _, provider := range providers {
		if os.Getenv(provider) != "" {
			fmt.Printf("✅ Found %s\n", provider)
			hasAPIKey = true
			break
		}
	}

	if !hasAPIKey {
		fmt.Println("⚠️  No API keys found for LLM providers")
		fmt.Println("💡 Set one of these environment variables:")
		for _, provider := range providers {
			fmt.Printf("   export %s=your_api_key\n", provider)
		}
		fmt.Println()
		fmt.Println("🔄 Continuing with test (will fail at LLM call)...")
	}

	generator, err := git.NewCommitMessageGenerator()
	if err != nil {
		log.Printf("❌ Failed to create commit message generator: %v", err)
		return
	}

	commitMessage, err := generator.GenerateCommitMessage(context.Background(), repo, false)
	if err != nil {
		log.Printf("❌ Failed to generate commit message: %v", err)
		fmt.Println()
		fmt.Println("💡 This is expected if no API keys are configured")
		return
	}

	fmt.Printf("✅ Generated commit message:\n")
	fmt.Printf("   %s\n", commitMessage)
	fmt.Println()

	fmt.Println("🎉 AI commit message generation test completed successfully!")
	fmt.Println()
	fmt.Println("💡 To test the full commit functionality:")
	fmt.Println("   1. Make sure you have changes to commit")
	fmt.Println("   2. Set an API key for an LLM provider")
	fmt.Println("   3. Use: go run . chat")
	fmt.Println("   4. Type: 'commit' or 'generate commit message'")
}
