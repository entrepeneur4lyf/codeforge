package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/permissions"
)

// Service handles project overview and PRD management
type Service struct {
	config        *config.Config
	workingDir    string
	fileManager   *permissions.FileOperationManager
	workspaceRoot string
}

// NewService creates a new project service
func NewService(cfg *config.Config, workspaceRoot string, fileManager *permissions.FileOperationManager) *Service {
	workingDir := workspaceRoot
	if cfg != nil && cfg.WorkingDir != "" {
		workingDir = cfg.WorkingDir
	}

	return &Service{
		config:        cfg,
		workingDir:    workingDir,
		fileManager:   fileManager,
		workspaceRoot: workspaceRoot,
	}
}

// CheckForExistingPRD checks if a project overview already exists
func (s *Service) CheckForExistingPRD() (bool, string, error) {
	// Check for AGENTS.md first (primary context file)
	if content, err := s.readFile("AGENTS.md"); err == nil {
		return true, content, nil
	}

	// Check for project-overview.md
	if content, err := s.readFile("project-overview.md"); err == nil {
		return true, content, nil
	}

	// Check for README.md with project info
	if content, err := s.readFile("README.md"); err == nil && s.containsProjectInfo(content) {
		return true, content, nil
	}

	return false, "", nil
}

// readFile reads content from a file using the file manager if available
func (s *Service) readFile(path string) (string, error) {
	fullPath := filepath.Join(s.workingDir, path)

	if s.fileManager != nil {
		// Use permission-aware file reading
		result, err := s.fileManager.ReadFile(context.Background(), &permissions.FileOperationRequest{
			SessionID: "system", // System session for PRD operations
			Operation: "read",
			Path:      path,
		})
		if err != nil {
			return "", err
		}
		if !result.Success {
			return "", fmt.Errorf("file read failed: %s", result.Error)
		}
		return string(result.Content), nil
	}

	// Fallback to direct file reading
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// writeFile writes content to a file using the file manager if available
func (s *Service) writeFile(path string, content []byte) error {
	if s.fileManager != nil {
		// Use permission-aware file writing
		result, err := s.fileManager.WriteFile(context.Background(), &permissions.FileOperationRequest{
			SessionID: "system", // System session for PRD operations
			Operation: "write",
			Path:      path,
			Content:   content,
			Mode:      0644,
		})
		if err != nil {
			return err
		}
		if !result.Success {
			return fmt.Errorf("file write failed: %s", result.Error)
		}
		return nil
	}

	// Fallback to direct file writing
	fullPath := filepath.Join(s.workingDir, path)
	return os.WriteFile(fullPath, content, 0644)
}

// containsProjectInfo checks if content contains project information
func (s *Service) containsProjectInfo(content string) bool {
	indicators := []string{
		"project overview",
		"project description",
		"what we're building",
		"application overview",
		"product requirements",
		"tech stack",
		"target users",
		"project requirements document",
		"prd",
	}

	contentLower := strings.ToLower(content)
	for _, indicator := range indicators {
		if strings.Contains(contentLower, indicator) {
			return true
		}
	}
	return false
}

// extractProjectName extracts project name from current directory
func (s *Service) extractProjectName() string {
	if s.workspaceRoot != "" {
		return filepath.Base(s.workspaceRoot)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "New Project"
	}
	return filepath.Base(cwd)
}

// SaveProjectOverview saves the project overview to project-overview.md
func (s *Service) SaveProjectOverview(overview *ProjectOverview) error {
	markdown := s.GenerateProjectOverviewMarkdown(overview)

	err := s.writeFile("project-overview.md", []byte(markdown))
	if err != nil {
		return fmt.Errorf("failed to save project-overview.md: %w", err)
	}

	return nil
}

// SaveProjectSummary saves the project summary to AGENTS.md (context file only)
func (s *Service) SaveProjectSummary(overview *ProjectOverview) error {
	summary := s.GenerateProjectSummary(overview)

	err := s.writeFile("AGENTS.md", []byte(summary))
	if err != nil {
		return fmt.Errorf("failed to save AGENTS.md: %w", err)
	}

	return nil
}

// CreatePRDFiles creates both project-overview.md and AGENTS.md from overview
func (s *Service) CreatePRDFiles(overview *ProjectOverview) error {
	// Always use direct file operations for automatic analysis to avoid permission hangs
	return s.createPRDFilesDirect(overview)
}

// createPRDFilesDirect creates PRD files using direct file operations
func (s *Service) createPRDFilesDirect(overview *ProjectOverview) error {
	// Generate content
	overviewMarkdown := s.GenerateProjectOverviewMarkdown(overview)
	summaryMarkdown := s.GenerateProjectSummary(overview)

	// Write files directly
	overviewPath := filepath.Join(s.workingDir, "project-overview.md")
	if err := os.WriteFile(overviewPath, []byte(overviewMarkdown), 0644); err != nil {
		return fmt.Errorf("failed to save project-overview.md: %w", err)
	}

	summaryPath := filepath.Join(s.workingDir, "AGENTS.md")
	if err := os.WriteFile(summaryPath, []byte(summaryMarkdown), 0644); err != nil {
		return fmt.Errorf("failed to save AGENTS.md: %w", err)
	}

	return nil
}

// UpdateProjectSummary intelligently merges existing AGENTS.md with current project analysis
func (s *Service) UpdateProjectSummary(overview *ProjectOverview, existingContent string) string {
	// Generate new summary from current analysis
	newSummary := s.GenerateProjectSummary(overview)

	// Parse existing content into structured sections
	existingSections := s.parseExistingContent(existingContent)

	// Parse new summary into structured sections
	newSections := s.parseExistingContent(newSummary)

	// Merge sections intelligently
	mergedSections := s.mergeSections(existingSections, newSections)

	// Reconstruct the final content
	return s.reconstructContent(mergedSections)
}

// parseExistingContent parses markdown content into structured sections
func (s *Service) parseExistingContent(content string) map[string]ContentSection {
	sections := make(map[string]ContentSection)
	lines := strings.Split(content, "\n")

	var currentSection string
	var currentContent []string
	var currentLevel int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect section headers
		if strings.HasPrefix(trimmed, "#") {
			// Save previous section if exists
			if currentSection != "" {
				sections[currentSection] = ContentSection{
					Title:     currentSection,
					Content:   strings.Join(currentContent, "\n"),
					Level:     currentLevel,
					LineStart: i - len(currentContent),
					LineEnd:   i - 1,
					IsCustom:  s.isCustomSection(currentSection),
				}
			}

			// Start new section
			currentLevel = s.getHeaderLevel(trimmed)
			currentSection = s.extractHeaderTitle(trimmed)
			currentContent = []string{}
		} else if currentSection != "" {
			currentContent = append(currentContent, line)
		}
	}

	// Save final section
	if currentSection != "" {
		sections[currentSection] = ContentSection{
			Title:     currentSection,
			Content:   strings.Join(currentContent, "\n"),
			Level:     currentLevel,
			LineStart: len(lines) - len(currentContent),
			LineEnd:   len(lines) - 1,
			IsCustom:  s.isCustomSection(currentSection),
		}
	}

	return sections
}

// ContentSection represents a parsed markdown section
type ContentSection struct {
	Title     string
	Content   string
	Level     int
	LineStart int
	LineEnd   int
	IsCustom  bool
	Priority  int
}

// mergeSections intelligently merges existing and new sections
func (s *Service) mergeSections(existing, new map[string]ContentSection) map[string]ContentSection {
	merged := make(map[string]ContentSection)

	// Standard sections that should be updated from new analysis
	standardSections := map[string]bool{
		"Project Overview":   true,
		"Architecture":       true,
		"Key Components":     true,
		"Technologies":       true,
		"File Structure":     true,
		"Dependencies":       true,
		"Development Status": true,
		"Recent Changes":     true,
	}

	// Add all new sections (they have latest analysis)
	for title, section := range new {
		merged[title] = section
	}

	// Preserve custom sections from existing content
	for title, section := range existing {
		if section.IsCustom || !standardSections[title] {
			// Check if this custom section conflicts with new content
			if _, exists := merged[title]; !exists {
				merged[title] = section
			} else {
				// Rename conflicting custom section
				newTitle := title + " (Preserved)"
				merged[newTitle] = section
			}
		}
	}

	return merged
}

// reconstructContent rebuilds markdown from merged sections
func (s *Service) reconstructContent(sections map[string]ContentSection) string {
	// Define section order for consistent output
	sectionOrder := []string{
		"Project Overview",
		"Architecture",
		"Key Components",
		"Technologies",
		"File Structure",
		"Dependencies",
		"Development Status",
		"Recent Changes",
	}

	var result strings.Builder

	// Add sections in preferred order
	for _, title := range sectionOrder {
		if section, exists := sections[title]; exists {
			result.WriteString(s.formatSection(section))
			result.WriteString("\n\n")
			delete(sections, title)
		}
	}

	// Add remaining custom sections
	for _, section := range sections {
		result.WriteString(s.formatSection(section))
		result.WriteString("\n\n")
	}

	return strings.TrimSpace(result.String())
}

// Helper functions for content parsing and formatting
func (s *Service) getHeaderLevel(line string) int {
	count := 0
	for _, char := range line {
		if char == '#' {
			count++
		} else {
			break
		}
	}
	return count
}

func (s *Service) extractHeaderTitle(line string) string {
	// Remove leading # characters and trim whitespace
	title := strings.TrimLeft(line, "#")
	return strings.TrimSpace(title)
}

func (s *Service) isCustomSection(title string) bool {
	standardTitles := map[string]bool{
		"Project Overview":   true,
		"Architecture":       true,
		"Key Components":     true,
		"Technologies":       true,
		"File Structure":     true,
		"Dependencies":       true,
		"Development Status": true,
		"Recent Changes":     true,
	}

	return !standardTitles[title] ||
		strings.Contains(strings.ToLower(title), "custom") ||
		strings.Contains(strings.ToLower(title), "note") ||
		strings.Contains(strings.ToLower(title), "additional")
}

func (s *Service) formatSection(section ContentSection) string {
	header := strings.Repeat("#", section.Level) + " " + section.Title
	if strings.TrimSpace(section.Content) == "" {
		return header
	}
	return header + "\n" + section.Content
}

// HasExistingProject checks if the current directory has project indicators
func (s *Service) HasExistingProject() bool {
	// Check for common project files
	projectFiles := []string{
		"package.json",
		"go.mod",
		"requirements.txt",
		"Cargo.toml",
		"pom.xml",
		"build.gradle",
		"composer.json",
		"Gemfile",
		"setup.py",
		"pyproject.toml",
	}

	for _, file := range projectFiles {
		if s.hasFile(file) {
			return true
		}
	}

	return false
}
