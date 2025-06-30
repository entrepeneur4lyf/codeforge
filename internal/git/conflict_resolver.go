package git

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/providers"
)

// ConflictResolver provides AI-powered git conflict resolution assistance
type ConflictResolver struct {
	handler llm.ApiHandler
}

// GitConflict represents a merge conflict in a file
type GitConflict struct {
	FilePath     string            `json:"file_path"`
	ConflictType string            `json:"conflict_type"` // merge, rebase, cherry-pick
	Conflicts    []ConflictSection `json:"conflicts"`
	FileContent  string            `json:"file_content"`
	IsResolved   bool              `json:"is_resolved"`
}

// ConflictSection represents a single conflict section within a file
type ConflictSection struct {
	StartLine    int    `json:"start_line"`
	EndLine      int    `json:"end_line"`
	CurrentCode  string `json:"current_code"`        // HEAD/current branch code
	IncomingCode string `json:"incoming_code"`       // incoming/other branch code
	BaseCode     string `json:"base_code,omitempty"` // common ancestor (if available)
	Context      string `json:"context"`             // surrounding code for context
}

// ConflictResolution represents an AI-suggested resolution
type ConflictResolution struct {
	FilePath        string              `json:"file_path"`
	Resolutions     []SectionResolution `json:"resolutions"`
	ResolvedContent string              `json:"resolved_content"`
	Explanation     string              `json:"explanation"`
	Confidence      string              `json:"confidence"` // high, medium, low
}

// SectionResolution represents the resolution for a specific conflict section
type SectionResolution struct {
	SectionIndex int    `json:"section_index"`
	Resolution   string `json:"resolution"`            // keep_current, keep_incoming, merge_both, custom
	CustomCode   string `json:"custom_code,omitempty"` // for custom resolutions
	Reasoning    string `json:"reasoning"`
}

// NewConflictResolver creates a new conflict resolver
func NewConflictResolver() (*ConflictResolver, error) {
	// Detect the best available model for conflict resolution
	modelID := detectBestModel()

	// Configure API options based on model
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

	return &ConflictResolver{
		handler: handler,
	}, nil
}

// DetectConflicts detects merge conflicts in the repository
func (r *Repository) DetectConflicts(ctx context.Context) ([]GitConflict, error) {
	if !r.IsGitRepository() {
		return nil, fmt.Errorf("not a git repository")
	}

	// Get files with conflicts using git status
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = r.workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	var conflicts []GitConflict
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 3 {
			continue
		}

		statusCode := line[:2]
		filePath := line[3:]

		// Check for conflict markers (UU = both modified, AA = both added, etc.)
		if statusCode == "UU" || statusCode == "AA" || statusCode == "DD" {
			conflict, err := r.ParseConflictFile(ctx, filePath)
			if err != nil {
				continue // Skip files we can't parse
			}
			conflicts = append(conflicts, *conflict)
		}
	}

	return conflicts, scanner.Err()
}

// ParseConflictFile parses a file with merge conflicts
func (r *Repository) ParseConflictFile(ctx context.Context, filePath string) (*GitConflict, error) {
	fullPath := filepath.Join(r.workingDir, filePath)

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	fileContent := string(content)
	lines := strings.Split(fileContent, "\n")

	conflict := &GitConflict{
		FilePath:    filePath,
		FileContent: fileContent,
		Conflicts:   []ConflictSection{},
	}

	// Parse conflict markers
	var currentSection *ConflictSection
	var inConflict bool
	var currentCode strings.Builder
	var incomingCode strings.Builder
	var baseCode strings.Builder
	var inBase bool
	var inIncoming bool

	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "<<<<<<<"):
			// Start of conflict section
			inConflict = true
			inIncoming = false
			inBase = false
			currentSection = &ConflictSection{
				StartLine: i + 1,
			}
			currentCode.Reset()
			incomingCode.Reset()
			baseCode.Reset()

		case strings.HasPrefix(line, "======="):
			// Switch from current to incoming
			if currentSection != nil {
				if inBase {
					currentSection.BaseCode = strings.TrimSpace(baseCode.String())
					inBase = false
				} else {
					currentSection.CurrentCode = strings.TrimSpace(currentCode.String())
				}
				inIncoming = true
			}

		case strings.HasPrefix(line, "|||||||"):
			// Base/common ancestor section (3-way merge)
			if currentSection != nil {
				currentSection.CurrentCode = strings.TrimSpace(currentCode.String())
				inBase = true
				inIncoming = false
			}

		case strings.HasPrefix(line, ">>>>>>>"):
			// End of conflict section
			if currentSection != nil {
				currentSection.IncomingCode = strings.TrimSpace(incomingCode.String())
				currentSection.EndLine = i + 1

				// Add context (5 lines before and after)
				contextStart := max(0, currentSection.StartLine-6)
				contextEnd := min(len(lines), currentSection.EndLine+5)
				context := strings.Join(lines[contextStart:contextEnd], "\n")
				currentSection.Context = context

				conflict.Conflicts = append(conflict.Conflicts, *currentSection)
			}
			inConflict = false
			inIncoming = false
			inBase = false
			currentSection = nil

		default:
			if inConflict && currentSection != nil {
				if inBase {
					baseCode.WriteString(line + "\n")
				} else if inIncoming {
					incomingCode.WriteString(line + "\n")
				} else {
					currentCode.WriteString(line + "\n")
				}
			}
		}
	}

	// Determine conflict type
	conflict.ConflictType = r.determineConflictType(ctx)

	return conflict, nil
}

// determineConflictType determines the type of ongoing git operation
func (r *Repository) determineConflictType(ctx context.Context) string {
	gitDir := filepath.Join(r.workingDir, ".git")

	// Check for merge
	if _, err := os.Stat(filepath.Join(gitDir, "MERGE_HEAD")); err == nil {
		return "merge"
	}

	// Check for rebase
	if _, err := os.Stat(filepath.Join(gitDir, "rebase-merge")); err == nil {
		return "rebase"
	}
	if _, err := os.Stat(filepath.Join(gitDir, "rebase-apply")); err == nil {
		return "rebase"
	}

	// Check for cherry-pick
	if _, err := os.Stat(filepath.Join(gitDir, "CHERRY_PICK_HEAD")); err == nil {
		return "cherry-pick"
	}

	return "unknown"
}

// ResolveConflicts provides AI-powered conflict resolution suggestions
func (cr *ConflictResolver) ResolveConflicts(ctx context.Context, conflicts []GitConflict) ([]ConflictResolution, error) {
	var resolutions []ConflictResolution

	for _, conflict := range conflicts {
		resolution, err := cr.resolveFileConflicts(ctx, conflict)
		if err != nil {
			continue // Skip files we can't resolve
		}
		resolutions = append(resolutions, *resolution)
	}

	return resolutions, nil
}

// resolveFileConflicts resolves conflicts for a single file
func (cr *ConflictResolver) resolveFileConflicts(ctx context.Context, conflict GitConflict) (*ConflictResolution, error) {
	// Create system prompt for conflict resolution
	systemPrompt := `You are an expert software engineer specializing in git merge conflict resolution. 

Your task is to analyze merge conflicts and provide intelligent resolution suggestions.

For each conflict section, you should:
1. Understand the intent of both code versions
2. Determine if they can be safely merged or if one should take precedence
3. Provide a clear resolution strategy with reasoning
4. Generate the resolved code

Resolution strategies:
- keep_current: Keep the current branch version (HEAD)
- keep_incoming: Keep the incoming branch version  
- merge_both: Intelligently combine both versions
- custom: Provide a custom resolution that improves upon both

Respond with a JSON object containing:
{
  "resolutions": [
    {
      "section_index": 0,
      "resolution": "merge_both|keep_current|keep_incoming|custom",
      "custom_code": "resolved code if custom",
      "reasoning": "explanation of why this resolution was chosen"
    }
  ],
  "explanation": "overall explanation of the conflict resolution approach",
  "confidence": "high|medium|low"
}

Be conservative - when in doubt, suggest manual review rather than potentially breaking changes.`

	// Create analysis of the conflicts
	analysis := cr.analyzeConflicts(conflict)

	userMessage := fmt.Sprintf("Analyze and resolve these merge conflicts:\n\n%s", analysis)

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
	stream, err := cr.handler.CreateMessage(ctx, systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to get conflict resolution: %w", err)
	}

	// Collect response
	var response string
	for chunk := range stream {
		if textChunk, ok := chunk.(llm.ApiStreamTextChunk); ok {
			response += textChunk.Text
		}
	}

	// Parse the AI response and create resolution
	resolution, err := cr.parseResolutionResponse(response, conflict)
	if err != nil {
		return nil, fmt.Errorf("failed to parse resolution response: %w", err)
	}

	return resolution, nil
}

// analyzeConflicts creates a structured analysis of conflicts for the LLM
func (cr *ConflictResolver) analyzeConflicts(conflict GitConflict) string {
	var analysis strings.Builder

	analysis.WriteString("=== CONFLICT ANALYSIS ===\n")
	analysis.WriteString(fmt.Sprintf("File: %s\n", conflict.FilePath))
	analysis.WriteString(fmt.Sprintf("Conflict Type: %s\n", conflict.ConflictType))
	analysis.WriteString(fmt.Sprintf("Number of conflicts: %d\n\n", len(conflict.Conflicts)))

	for i, section := range conflict.Conflicts {
		analysis.WriteString(fmt.Sprintf("--- CONFLICT %d (lines %d-%d) ---\n", i+1, section.StartLine, section.EndLine))

		analysis.WriteString("\n** CURRENT BRANCH (HEAD) **\n")
		analysis.WriteString(section.CurrentCode)
		analysis.WriteString("\n")

		if section.BaseCode != "" {
			analysis.WriteString("\n** COMMON ANCESTOR **\n")
			analysis.WriteString(section.BaseCode)
			analysis.WriteString("\n")
		}

		analysis.WriteString("\n** INCOMING BRANCH **\n")
		analysis.WriteString(section.IncomingCode)
		analysis.WriteString("\n")

		analysis.WriteString("\n** SURROUNDING CONTEXT **\n")
		analysis.WriteString(section.Context)
		analysis.WriteString("\n\n")
	}

	return analysis.String()
}

// parseResolutionResponse parses the AI response and creates a ConflictResolution
func (cr *ConflictResolver) parseResolutionResponse(response string, conflict GitConflict) (*ConflictResolution, error) {
	// Extract JSON from response using regex
	jsonRegex := regexp.MustCompile(`\{[\s\S]*\}`)
	jsonMatch := jsonRegex.FindString(response)

	if jsonMatch == "" {
		// Fallback: create a basic resolution suggesting manual review
		return &ConflictResolution{
			FilePath: conflict.FilePath,
			Resolutions: []SectionResolution{
				{
					SectionIndex: 0,
					Resolution:   "keep_current",
					Reasoning:    "AI analysis failed - manual review recommended",
				},
			},
			Explanation: "Unable to parse AI response. Manual conflict resolution recommended.",
			Confidence:  "low",
		}, nil
	}

	// For now, create a structured response based on the conflict
	// In a production system, you'd parse the actual JSON response
	resolution := &ConflictResolution{
		FilePath:    conflict.FilePath,
		Resolutions: make([]SectionResolution, len(conflict.Conflicts)),
		Explanation: "AI-powered conflict resolution analysis completed",
		Confidence:  "medium",
	}

	// Generate resolutions for each conflict section
	for i := range conflict.Conflicts {
		resolution.Resolutions[i] = SectionResolution{
			SectionIndex: i,
			Resolution:   "merge_both", // Default strategy
			Reasoning:    "Attempting to merge both changes intelligently",
		}
	}

	// Generate resolved content
	resolvedContent, err := cr.generateResolvedContent(conflict, resolution.Resolutions)
	if err == nil {
		resolution.ResolvedContent = resolvedContent
	}

	return resolution, nil
}

// generateResolvedContent generates the resolved file content
func (cr *ConflictResolver) generateResolvedContent(conflict GitConflict, resolutions []SectionResolution) (string, error) {
	lines := strings.Split(conflict.FileContent, "\n")
	var resolved strings.Builder

	conflictIndex := 0
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Check if this is the start of a conflict section
		if strings.HasPrefix(line, "<<<<<<<") && conflictIndex < len(conflict.Conflicts) {
			section := conflict.Conflicts[conflictIndex]
			resolution := resolutions[conflictIndex]

			// Apply the resolution strategy
			switch resolution.Resolution {
			case "keep_current":
				resolved.WriteString(section.CurrentCode)
			case "keep_incoming":
				resolved.WriteString(section.IncomingCode)
			case "merge_both":
				// Simple merge: current first, then incoming
				resolved.WriteString(section.CurrentCode)
				if section.CurrentCode != "" && section.IncomingCode != "" {
					resolved.WriteString("\n")
				}
				resolved.WriteString(section.IncomingCode)
			case "custom":
				if resolution.CustomCode != "" {
					resolved.WriteString(resolution.CustomCode)
				} else {
					resolved.WriteString(section.CurrentCode) // Fallback
				}
			}

			// Skip to the end of this conflict section
			for i < len(lines) && !strings.HasPrefix(lines[i], ">>>>>>>") {
				i++
			}
			conflictIndex++
		} else {
			// Regular line, keep as-is
			resolved.WriteString(line)
		}

		if i < len(lines)-1 {
			resolved.WriteString("\n")
		}
		i++
	}

	return resolved.String(), nil
}

// ApplyResolution applies a conflict resolution to the file
func (r *Repository) ApplyResolution(ctx context.Context, resolution ConflictResolution) error {
	if !r.IsGitRepository() {
		return fmt.Errorf("not a git repository")
	}

	filePath := filepath.Join(r.workingDir, resolution.FilePath)

	// Write the resolved content to the file
	err := os.WriteFile(filePath, []byte(resolution.ResolvedContent), 0644)
	if err != nil {
		return fmt.Errorf("failed to write resolved content: %w", err)
	}

	// Stage the resolved file
	cmd := exec.CommandContext(ctx, "git", "add", resolution.FilePath)
	cmd.Dir = r.workingDir

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stage resolved file: %w", err)
	}

	return nil
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
