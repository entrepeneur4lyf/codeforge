package chat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/builder"
	"github.com/entrepeneur4lyf/codeforge/internal/embeddings"
	"github.com/entrepeneur4lyf/codeforge/internal/git"
	"github.com/entrepeneur4lyf/codeforge/internal/lsp"
	"github.com/entrepeneur4lyf/codeforge/internal/ml"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

// CommandRouter handles natural language commands and routes them to appropriate functionality
type CommandRouter struct {
	workingDir string
}

// NewCommandRouter creates a new command router
func NewCommandRouter(workingDir string) *CommandRouter {
	return &CommandRouter{
		workingDir: workingDir,
	}
}

// RouteDirectCommand handles commands that should be executed directly (build, file ops, git)
func (cr *CommandRouter) RouteDirectCommand(ctx context.Context, userInput string) (string, bool) {
	input := strings.ToLower(strings.TrimSpace(userInput))

	// Build commands - these are direct actions
	if cr.isBuildCommand(input) {
		return cr.handleBuildCommand(ctx, userInput)
	}

	// File operations - these are direct actions
	if cr.isFileCommand(input) {
		return cr.handleFileCommand(ctx, userInput)
	}

	// Git AI commit commands - these are direct actions
	if cr.isGitCommitCommand(input) {
		return cr.handleGitCommitCommand(ctx, userInput)
	}

	// Git conflict resolution commands - these are direct actions
	if cr.isGitConflictCommand(input) {
		return cr.handleGitConflictCommand(ctx, userInput)
	}

	// Not a direct command
	return "", false
}

// Git commit command detection and handling
func (cr *CommandRouter) isGitCommitCommand(input string) bool {
	gitCommitKeywords := []string{
		"commit", "git commit", "ai commit", "smart commit", "auto commit",
		"generate commit", "commit message", "commit with ai", "ai commit message",
	}

	for _, keyword := range gitCommitKeywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func (cr *CommandRouter) handleGitCommitCommand(ctx context.Context, userInput string) (string, bool) {
	input := strings.ToLower(strings.TrimSpace(userInput))

	// Create git repository instance
	repo := git.NewRepository(cr.workingDir)

	// Check if git is available and this is a git repository
	if !git.IsGitInstalled() {
		return "Git is not installed on this system", true
	}

	if !repo.IsGitRepository() {
		return "This directory is not a git repository", true
	}

	// Determine if user wants to commit staged changes only
	staged := strings.Contains(input, "staged") || strings.Contains(input, "index")

	// Check if user just wants to generate a message without committing
	generateOnly := strings.Contains(input, "generate") || strings.Contains(input, "message only") || strings.Contains(input, "preview")

	if generateOnly {
		// Generate commit message without committing
		generator, err := git.NewCommitMessageGenerator()
		if err != nil {
			return fmt.Sprintf("Failed to create commit message generator: %v", err), true
		}

		commitMessage, err := generator.GenerateCommitMessage(ctx, repo, staged)
		if err != nil {
			return fmt.Sprintf("Failed to generate commit message: %v", err), true
		}

		return fmt.Sprintf("Generated commit message:\n\n%s\n\nTo commit with this message, say 'commit' or 'git commit'", commitMessage), true
	}

	// Commit with AI-generated message
	commitMessage, err := repo.CommitWithAIMessage(ctx, staged)
	if err != nil {
		return fmt.Sprintf("Failed to commit with AI message: %v", err), true
	}

	stagedText := ""
	if staged {
		stagedText = " (staged changes only)"
	}

	return fmt.Sprintf("Successfully committed%s with AI-generated message:\n\n%s", stagedText, commitMessage), true
}

// Git conflict command detection and handling
func (cr *CommandRouter) isGitConflictCommand(input string) bool {
	conflictKeywords := []string{
		"conflict", "conflicts", "merge conflict", "merge conflicts", "resolve conflict",
		"resolve conflicts", "conflict resolution", "fix conflict", "fix conflicts",
		"git conflict", "git conflicts", "conflict help", "conflict assistance",
		"merge issue", "merge issues", "conflict detector", "detect conflicts",
	}

	for _, keyword := range conflictKeywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func (cr *CommandRouter) handleGitConflictCommand(ctx context.Context, userInput string) (string, bool) {
	input := strings.ToLower(strings.TrimSpace(userInput))

	// Create git repository instance
	repo := git.NewRepository(cr.workingDir)

	// Check if git is available and this is a git repository
	if !git.IsGitInstalled() {
		return "Git is not installed", true
	}

	if !repo.IsGitRepository() {
		return "Not a git repository", true
	}

	// Check if user just wants to detect conflicts
	detectOnly := strings.Contains(input, "detect") || strings.Contains(input, "check") || strings.Contains(input, "find")

	// Detect conflicts
	conflicts, err := repo.DetectConflicts(ctx)
	if err != nil {
		return fmt.Sprintf("Failed to detect conflicts: %v", err), true
	}

	if len(conflicts) == 0 {
		return "No merge conflicts detected in the repository", true
	}

	if detectOnly {
		var response strings.Builder
		response.WriteString(fmt.Sprintf("Found %d file(s) with merge conflicts:\n\n", len(conflicts)))

		for i, conflict := range conflicts {
			response.WriteString(fmt.Sprintf("%d. %s (%s) - %d conflict section(s)\n",
				i+1, conflict.FilePath, conflict.ConflictType, len(conflict.Conflicts)))
		}

		response.WriteString("\nTo get AI-powered resolution suggestions, say 'resolve conflicts' or 'conflict help'")
		return response.String(), true
	}

	// Create conflict resolver and get AI suggestions
	resolver, err := git.NewConflictResolver()
	if err != nil {
		return fmt.Sprintf("Failed to create conflict resolver: %v", err), true
	}

	resolutions, err := resolver.ResolveConflicts(ctx, conflicts)
	if err != nil {
		return fmt.Sprintf("Failed to get conflict resolutions: %v", err), true
	}

	// Check if user wants to auto-apply
	autoApply := strings.Contains(input, "apply") || strings.Contains(input, "fix") || strings.Contains(input, "auto")

	var response strings.Builder
	response.WriteString(fmt.Sprintf("AI Conflict Resolution Analysis for %d file(s):\n\n", len(resolutions)))

	appliedCount := 0
	for _, resolution := range resolutions {
		response.WriteString(fmt.Sprintf("**%s** (Confidence: %s)\n", resolution.FilePath, resolution.Confidence))
		response.WriteString(fmt.Sprintf("   %s\n\n", resolution.Explanation))

		for j, sectionRes := range resolution.Resolutions {
			response.WriteString(fmt.Sprintf("   Conflict %d: %s\n", j+1, sectionRes.Resolution))
			response.WriteString(fmt.Sprintf("   Reasoning: %s\n", sectionRes.Reasoning))
		}

		if autoApply {
			if err := repo.ApplyResolution(ctx, resolution); err == nil {
				response.WriteString("   Applied automatically\n")
				appliedCount++
			} else {
				response.WriteString(fmt.Sprintf("   Failed to apply: %v\n", err))
			}
		}
		response.WriteString("\n")
	}

	if autoApply {
		response.WriteString(fmt.Sprintf("Applied %d out of %d resolutions automatically\n", appliedCount, len(resolutions)))
		if appliedCount > 0 {
			response.WriteString("Review the changes and commit when ready")
		}
	} else {
		response.WriteString("To apply these resolutions automatically, say 'apply conflict resolutions' or 'fix conflicts'")
	}

	return response.String(), true
}

// GatherContext collects relevant context using ML-powered code intelligence
func (cr *CommandRouter) GatherContext(ctx context.Context, userInput string) string {
	// Only gather context for code-related queries to avoid unnecessary ML searches
	if !cr.isCodeRelatedQuery(userInput) {
		return ""
	}

	// Try to get ML service for intelligent context
	mlService := ml.GetService()
	if mlService != nil && mlService.IsEnabled() {
		// Use ML-powered intelligent context gathering
		context := mlService.GetIntelligentContext(ctx, userInput, 10)

		// If ML context is empty, try smart search as fallback
		if context == "" {
			context = mlService.SmartSearch(ctx, userInput)
		}

		if context != "" {
			return context
		}
	}

	// Graceful degradation - return empty context if ML is not available
	// This allows the existing chat system to work normally
	return ""
}

// isCodeRelatedQuery determines if a query is code-related and needs context
func (cr *CommandRouter) isCodeRelatedQuery(input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))

	// Skip simple greetings and non-code queries
	simpleGreetings := []string{
		"hi", "hello", "hey", "good morning", "good afternoon", "good evening",
		"thanks", "thank you", "bye", "goodbye", "ok", "okay", "yes", "no",
		"sure", "great", "awesome", "cool", "nice",
	}

	for _, greeting := range simpleGreetings {
		if input == greeting || input == greeting+"!" || input == greeting+"." {
			return false
		}
	}

	// Check for code-related keywords
	codeKeywords := []string{
		"code", "function", "method", "class", "variable", "import", "export",
		"error", "bug", "debug", "fix", "implement", "create", "add", "remove",
		"search", "find", "look", "show", "explain", "how", "what", "where",
		"file", "directory", "project", "build", "compile", "test", "run",
		"git", "commit", "merge", "branch", "pull", "push", "diff", "status", "conflict", "conflicts",
		"api", "endpoint", "database", "query", "sql", "json", "xml", "yaml",
		"config", "configuration", "setting", "option", "parameter", "argument",
		"library", "package", "dependency", "module", "component", "service",
		"interface", "struct", "type", "enum", "constant", "global", "local",
		"async", "await", "promise", "callback", "event", "listener", "handler",
		"loop", "condition", "if", "else", "switch", "case", "for", "while",
		"return", "throw", "catch", "try", "finally", "exception", "error",
	}

	for _, keyword := range codeKeywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}

	// If input is longer than a simple greeting, it might be code-related
	if len(strings.Fields(input)) > 3 {
		return true
	}

	return false
}

// Build command detection and handling
func (cr *CommandRouter) isBuildCommand(input string) bool {
	buildKeywords := []string{
		"build", "compile", "make", "cargo build", "go build", "npm run build",
		"mvn compile", "tsc", "cmake", "fix build", "build error", "compilation",
	}

	for _, keyword := range buildKeywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func (cr *CommandRouter) handleBuildCommand(ctx context.Context, userInput string) (string, bool) {
	// Execute build
	output, err := builder.Build(cr.workingDir)

	if err != nil {
		// Build failed - provide detailed error analysis
		errorOutput := string(output)
		result := fmt.Sprintf("🔨 Build failed in %s\n\n", cr.workingDir)
		result += "**Error Output:**\n```\n" + errorOutput + "\n```\n\n"

		// Try to parse and explain the error
		if errorOutput != "" {
			result += "**Analysis:**\n"
			result += cr.analyzeBuildError(errorOutput)
		}

		return result, true
	}

	// Build succeeded
	result := fmt.Sprintf("Build successful in %s\n\n", cr.workingDir)
	if len(output) > 0 {
		result += "**Build Output:**\n```\n" + string(output) + "\n```"
	}

	return result, true
}

func (cr *CommandRouter) analyzeBuildError(errorOutput string) string {
	analysis := ""

	// Common error patterns
	if strings.Contains(errorOutput, "cannot find package") || strings.Contains(errorOutput, "no such file") {
		analysis += "- **Missing dependency**: The build is failing because a required package or file cannot be found.\n"
		analysis += "- **Solution**: Check your dependencies and ensure all required packages are installed.\n\n"
	}

	if strings.Contains(errorOutput, "syntax error") || strings.Contains(errorOutput, "expected") {
		analysis += "- **Syntax error**: There's a syntax error in your code.\n"
		analysis += "- **Solution**: Check the file and line number mentioned in the error.\n\n"
	}

	if strings.Contains(errorOutput, "undefined") || strings.Contains(errorOutput, "not declared") {
		analysis += "- **Undefined symbol**: A variable, function, or type is being used but not defined.\n"
		analysis += "- **Solution**: Check for typos or missing imports/declarations.\n\n"
	}

	if analysis == "" {
		analysis = "- Review the error output above for specific details about what went wrong.\n"
		analysis += "- Check the file paths and line numbers mentioned in the error.\n"
	}

	return analysis
}

// Search command detection and handling
func (cr *CommandRouter) isSearchCommand(input string) bool {
	searchKeywords := []string{
		"search", "find", "look for", "locate", "grep", "search for",
		"find code", "search code", "semantic search", "vector search",
	}

	for _, keyword := range searchKeywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func (cr *CommandRouter) handleSearchCommand(ctx context.Context, userInput string) (string, bool) {
	// Extract search query from user input
	query := cr.extractSearchQuery(userInput)
	if query == "" {
		return "Could not extract search query. Please specify what you want to search for.", true
	}

	// Get embedding for the search query using the package-level function
	embedding, err := embeddings.GetEmbedding(ctx, query)
	if err != nil {
		return fmt.Sprintf("Failed to generate embedding: %v", err), true
	}

	// Search vector database
	vdb := vectordb.Get()
	if vdb == nil {
		return "Vector database not available", true
	}

	results, err := vdb.SearchSimilarChunks(ctx, embedding, 5, map[string]string{})
	if err != nil {
		return fmt.Sprintf("Search failed: %v", err), true
	}

	if len(results) == 0 {
		return fmt.Sprintf("No results found for: %s", query), true
	}

	// Format results
	response := fmt.Sprintf("Search results for: **%s**\n\n", query)
	for i, result := range results {
		response += fmt.Sprintf("**%d. %s** (Score: %.3f)\n", i+1, result.Chunk.FilePath, result.Score)
		response += fmt.Sprintf("```%s\n%s\n```\n\n", result.Chunk.Language, result.Chunk.Content)
	}

	return response, true
}

func (cr *CommandRouter) extractSearchQuery(input string) string {
	// Remove common search prefixes
	prefixes := []string{
		"search for ", "find ", "look for ", "locate ", "search ",
		"find code ", "search code ", "semantic search ", "vector search ",
	}

	query := input
	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(query), prefix) {
			query = query[len(prefix):]
			break
		}
	}

	return strings.TrimSpace(query)
}

// LSP command detection and handling
func (cr *CommandRouter) isLSPCommand(input string) bool {
	lspKeywords := []string{
		"definition", "find definition", "go to definition", "goto definition",
		"references", "find references", "find usages", "where is used",
		"hover", "type info", "symbol info", "documentation",
		"completion", "autocomplete", "code completion",
	}

	for _, keyword := range lspKeywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func (cr *CommandRouter) handleLSPCommand(ctx context.Context, userInput string) (string, bool) {
	lspManager := lsp.GetManager()
	if lspManager == nil {
		return "LSP manager not available", true
	}

	// For now, provide general LSP information
	// In a full implementation, this would parse the command and execute specific LSP operations
	response := "🔧 **LSP Features Available:**\n\n"
	response += "- **Find Definition**: Locate where symbols are defined\n"
	response += "- **Find References**: Find all usages of a symbol\n"
	response += "- **Hover Information**: Get type and documentation info\n"
	response += "- **Code Completion**: Get autocomplete suggestions\n\n"
	response += "**Note**: LSP features work best when you specify a file and position.\n"
	response += "Example: \"Find definition of MyFunction in main.go at line 25\""

	return response, true
}

// File command detection and handling
func (cr *CommandRouter) isFileCommand(input string) bool {
	fileKeywords := []string{
		"list files", "show files", "what files", "file tree", "directory",
		"ls", "dir", "files in", "show directory", "project structure",
	}

	for _, keyword := range fileKeywords {
		if strings.Contains(input, keyword) {
			return true
		}
	}
	return false
}

func (cr *CommandRouter) handleFileCommand(ctx context.Context, userInput string) (string, bool) {
	// List files in the working directory
	files, err := cr.listProjectFiles(cr.workingDir)
	if err != nil {
		return fmt.Sprintf("Failed to list files: %v", err), true
	}

	response := fmt.Sprintf("**Project files in %s:**\n\n", cr.workingDir)
	for _, file := range files {
		response += fmt.Sprintf("- %s\n", file)
	}

	return response, true
}

func (cr *CommandRouter) listProjectFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files and directories
		if strings.HasPrefix(info.Name(), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common build/cache directories
		skipDirs := []string{"node_modules", "target", "build", "dist", ".git", "__pycache__"}
		for _, skipDir := range skipDirs {
			if info.IsDir() && info.Name() == skipDir {
				return filepath.SkipDir
			}
		}

		if !info.IsDir() {
			// Get relative path
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				relPath = path
			}
			files = append(files, relPath)
		}

		return nil
	})

	return files, err
}
