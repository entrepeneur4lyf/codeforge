package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/providers"
)

// CommitMessageGenerator generates AI-powered commit messages
type CommitMessageGenerator struct {
	handler llm.ApiHandler
}

// NewCommitMessageGenerator creates a new commit message generator
func NewCommitMessageGenerator() (*CommitMessageGenerator, error) {
	// Detect best model and configure API options
	modelID := detectBestModel()

	options := llm.ApiHandlerOptions{
		ModelID: modelID,
	}

	// Set appropriate API key based on detected model
	if strings.Contains(modelID, "claude") {
		options.APIKey = getEnvVar("ANTHROPIC_API_KEY")
	} else if strings.Contains(modelID, "gpt") {
		options.APIKey = getEnvVar("OPENAI_API_KEY")
	} else if strings.Contains(modelID, "llama") && strings.Contains(modelID, "groq") {
		options.APIKey = getEnvVar("GROQ_API_KEY")
	} else if strings.Contains(modelID, "deepseek") {
		options.APIKey = getEnvVar("DEEPSEEK_API_KEY")
	} else if strings.Contains(modelID, "gemini") {
		options.APIKey = getEnvVar("GEMINI_API_KEY")
	}

	handler, err := providers.BuildApiHandler(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM handler: %w", err)
	}

	return &CommitMessageGenerator{
		handler: handler,
	}, nil
}

// GenerateCommitMessage generates an AI-powered commit message based on git diff
func (cmg *CommitMessageGenerator) GenerateCommitMessage(ctx context.Context, repo *Repository, staged bool) (string, error) {
	// Get git diff
	diffs, err := repo.GetDiff(ctx, staged)
	if err != nil {
		return "", fmt.Errorf("failed to get git diff: %w", err)
	}

	if len(diffs) == 0 {
		return "", fmt.Errorf("no changes to commit")
	}

	// Analyze the changes
	analysis := cmg.analyzeDiffs(diffs)

	// Generate commit message using LLM
	systemPrompt := `You are an expert software developer who writes excellent git commit messages. 

Your task is to analyze git diffs and generate a concise, descriptive commit message that follows conventional commit format.

Rules:
1. Use conventional commit format: type(scope): description
2. Types: feat, fix, docs, style, refactor, test, chore, perf, ci, build
3. Keep the description under 50 characters when possible
4. Be specific about what changed, not just which files
5. Focus on the "what" and "why", not the "how"
6. Use imperative mood (e.g., "add", "fix", "update")

Examples:
- feat(auth): add OAuth2 login support
- fix(api): resolve null pointer in user handler
- docs(readme): update installation instructions
- refactor(db): extract connection pooling logic
- test(user): add unit tests for validation

Analyze the provided git diff and generate an appropriate commit message.

IMPORTANT: Respond with ONLY the commit message in the exact format specified. Do not include any explanations, preamble, or additional text.`

	userMessage := fmt.Sprintf("Analyze this git diff and generate a commit message:\n\n%s", analysis)

	// Create message
	messages := []llm.Message{
		{
			Role: "user",
			Content: []llm.ContentBlock{
				llm.TextBlock{Text: userMessage},
			},
		},
	}

	// Get completion
	stream, err := cmg.handler.CreateMessage(ctx, systemPrompt, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate commit message: %w", err)
	}

	// Collect response
	var commitMessage string
	for chunk := range stream {
		if textChunk, ok := chunk.(llm.ApiStreamTextChunk); ok {
			commitMessage += textChunk.Text
		}
		// Note: Errors are handled at the CreateMessage level, not as stream chunks
	}

	// Clean up the response
	commitMessage = strings.TrimSpace(commitMessage)

	// Remove any quotes or extra formatting
	commitMessage = strings.Trim(commitMessage, "\"'`")

	// Extract conventional commit message from response
	lines := strings.Split(commitMessage, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip empty lines and explanatory text
		if line == "" || strings.Contains(strings.ToLower(line), "commit message") ||
			strings.Contains(strings.ToLower(line), "git diff") ||
			strings.Contains(strings.ToLower(line), "based on") {
			continue
		}
		// If this looks like a conventional commit, use it
		if strings.Contains(line, ":") && (strings.HasPrefix(line, "feat") ||
			strings.HasPrefix(line, "fix") || strings.HasPrefix(line, "docs") ||
			strings.HasPrefix(line, "style") || strings.HasPrefix(line, "refactor") ||
			strings.HasPrefix(line, "test") || strings.HasPrefix(line, "chore") ||
			strings.HasPrefix(line, "perf") || strings.HasPrefix(line, "ci") ||
			strings.HasPrefix(line, "build")) {
			commitMessage = line
			break
		}
	}

	// If no conventional commit found, use the first meaningful line
	if !strings.Contains(commitMessage, ":") {
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && !strings.Contains(strings.ToLower(line), "commit message") &&
				!strings.Contains(strings.ToLower(line), "git diff") &&
				!strings.Contains(strings.ToLower(line), "based on") {
				commitMessage = line
				break
			}
		}
	}

	if commitMessage == "" {
		return "", fmt.Errorf("LLM returned empty response - check API key and model availability")
	}

	return commitMessage, nil
}

// analyzeDiffs creates a structured analysis of git diffs for the LLM
func (cmg *CommitMessageGenerator) analyzeDiffs(diffs []GitDiff) string {
	var analysis strings.Builder

	analysis.WriteString("=== CHANGE SUMMARY ===\n")
	analysis.WriteString(fmt.Sprintf("Total files changed: %d\n\n", len(diffs)))

	// Categorize changes
	var added, modified, deleted, renamed []string

	for _, diff := range diffs {
		switch {
		case diff.IsNewFile:
			added = append(added, diff.FilePath)
		case diff.IsDeleted:
			deleted = append(deleted, diff.FilePath)
		case diff.IsRenamed:
			renamed = append(renamed, fmt.Sprintf("%s -> %s", diff.OldPath, diff.FilePath))
		default:
			modified = append(modified, diff.FilePath)
		}
	}

	if len(added) > 0 {
		analysis.WriteString(fmt.Sprintf("Added files (%d): %s\n", len(added), strings.Join(added, ", ")))
	}
	if len(modified) > 0 {
		analysis.WriteString(fmt.Sprintf("Modified files (%d): %s\n", len(modified), strings.Join(modified, ", ")))
	}
	if len(deleted) > 0 {
		analysis.WriteString(fmt.Sprintf("Deleted files (%d): %s\n", len(deleted), strings.Join(deleted, ", ")))
	}
	if len(renamed) > 0 {
		analysis.WriteString(fmt.Sprintf("Renamed files (%d): %s\n", len(renamed), strings.Join(renamed, ", ")))
	}

	analysis.WriteString("\n=== DETAILED CHANGES ===\n")

	// Include diff content for analysis (limit to prevent token overflow)
	for i, diff := range diffs {
		if i >= 5 { // Limit to first 5 files to avoid token limits
			analysis.WriteString(fmt.Sprintf("... and %d more files\n", len(diffs)-i))
			break
		}

		analysis.WriteString(fmt.Sprintf("\n--- %s ---\n", diff.FilePath))
		analysis.WriteString(fmt.Sprintf("Status: %s\n", diff.Status))

		if diff.Content != "" {
			// Truncate very long diffs
			content := diff.Content
			if len(content) > 2000 {
				content = content[:2000] + "\n... (truncated)"
			}
			analysis.WriteString(content)
		}
		analysis.WriteString("\n")
	}

	return analysis.String()
}

// detectBestModel detects the best available model for commit message generation
func detectBestModel() string {
	// Prefer fast, efficient models for commit message generation
	// Check environment variables for available providers

	if apiKey := getEnvVar("ANTHROPIC_API_KEY"); apiKey != "" {
		return "claude-3-haiku-20240307" // Fast and efficient
	}

	if apiKey := getEnvVar("OPENAI_API_KEY"); apiKey != "" {
		return "gpt-3.5-turbo" // Fast and cost-effective
	}

	if apiKey := getEnvVar("GROQ_API_KEY"); apiKey != "" {
		return "llama3-8b-8192" // Very fast inference
	}

	if apiKey := getEnvVar("DEEPSEEK_API_KEY"); apiKey != "" {
		return "deepseek-chat" // Good and affordable
	}

	// Check for local models
	if isOllamaAvailable() {
		return "llama3.2:3b" // Small, fast local model
	}

	// Fallback to Claude (will error if no API key)
	return "claude-3-haiku-20240307"
}

// getEnvVar safely gets environment variable
func getEnvVar(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

// isOllamaAvailable checks if Ollama is available locally
func isOllamaAvailable() bool {
	// Simple check - could be enhanced to actually ping Ollama
	return getEnvVar("OLLAMA_HOST") != "" || fileExists("/usr/local/bin/ollama")
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// CommitWithAIMessage commits changes with an AI-generated message
func (r *Repository) CommitWithAIMessage(ctx context.Context, staged bool) (string, error) {
	if !r.IsGitRepository() {
		return "", fmt.Errorf("not a git repository")
	}

	// Create commit message generator
	generator, err := NewCommitMessageGenerator()
	if err != nil {
		return "", fmt.Errorf("failed to create commit message generator: %w", err)
	}

	// Generate commit message
	message, err := generator.GenerateCommitMessage(ctx, r, staged)
	if err != nil {
		return "", fmt.Errorf("failed to generate commit message: %w", err)
	}

	// Perform the commit
	var cmd *exec.Cmd
	if staged {
		// Commit only staged changes
		cmd = exec.CommandContext(ctx, "git", "commit", "-m", message)
	} else {
		// Stage and commit all changes
		cmd = exec.CommandContext(ctx, "git", "commit", "-am", message)
	}
	cmd.Dir = r.workingDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git commit failed: %w\nOutput: %s", err, string(output))
	}

	return message, nil
}
