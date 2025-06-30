package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/entrepeneur4lyf/codeforge/internal/git"
)

func main() {
	fmt.Println("🧪 Testing AI Commit with Staged Changes")
	fmt.Println("========================================")

	// Get current working directory
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Create repository instance
	repo := git.NewRepository(workingDir)

	// Test staged changes
	fmt.Println("📝 Getting staged diff...")
	stagedDiffs, err := repo.GetDiff(context.Background(), true)
	if err != nil {
		log.Printf("Failed to get staged diff: %v", err)
		return
	}

	if len(stagedDiffs) == 0 {
		fmt.Println("ℹ️  No staged changes detected")
		return
	}

	fmt.Printf("✅ Found %d staged changes\n", len(stagedDiffs))
	for i, diff := range stagedDiffs {
		if i >= 3 {
			fmt.Printf("  ... and %d more files\n", len(stagedDiffs)-i)
			break
		}
		fmt.Printf("  %s: %s\n", diff.Status, diff.FilePath)
		if diff.Content != "" {
			fmt.Printf("    Content length: %d characters\n", len(diff.Content))
		}
	}
	fmt.Println()

	// Test AI commit message generation
	fmt.Println("🤖 Generating AI commit message for staged changes...")
	generator, err := git.NewCommitMessageGenerator()
	if err != nil {
		log.Printf("❌ Failed to create commit message generator: %v", err)
		return
	}

	commitMessage, err := generator.GenerateCommitMessage(context.Background(), repo, true)
	if err != nil {
		log.Printf("❌ Failed to generate commit message: %v", err)
		return
	}

	fmt.Printf("✅ Generated commit message:\n")
	fmt.Printf("   %s\n", commitMessage)
	fmt.Println()

	fmt.Println("🎉 AI commit message generation test completed successfully!")
	fmt.Println()
	fmt.Println("💡 To commit with this message, run:")
	fmt.Printf("   git commit -m \"%s\"\n", commitMessage)
}
