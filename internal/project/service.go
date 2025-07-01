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
	// Check for AGENT.md first (primary context file)
	if content, err := s.readFile("AGENT.md"); err == nil {
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

// SaveProjectSummary saves the project summary to AGENT.md (context file only)
func (s *Service) SaveProjectSummary(overview *ProjectOverview) error {
	summary := s.GenerateProjectSummary(overview)

	err := s.writeFile("AGENT.md", []byte(summary))
	if err != nil {
		return fmt.Errorf("failed to save AGENT.md: %w", err)
	}

	return nil
}

// CreatePRDFiles creates both project-overview.md and AGENT.md from overview
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

	summaryPath := filepath.Join(s.workingDir, "AGENT.md")
	if err := os.WriteFile(summaryPath, []byte(summaryMarkdown), 0644); err != nil {
		return fmt.Errorf("failed to save AGENT.md: %w", err)
	}

	return nil
}

// UpdateProjectSummary intelligently updates existing AGENT.md with current project analysis
func (s *Service) UpdateProjectSummary(overview *ProjectOverview, existingContent string) string {
	// Generate new summary from current analysis
	newSummary := s.GenerateProjectSummary(overview)

	// For now, do intelligent merging - preserve custom sections but update core info
	// This is a simplified approach - in the future we could do more sophisticated merging

	// Check if existing content has custom sections we should preserve
	lines := strings.Split(existingContent, "\n")
	var customSections []string
	inCustomSection := false

	for _, line := range lines {
		// Look for custom sections (anything not in standard template)
		if strings.HasPrefix(line, "## Custom") || strings.HasPrefix(line, "## Notes") ||
			strings.HasPrefix(line, "## Additional") || strings.Contains(line, "CUSTOM:") {
			inCustomSection = true
		}
		if inCustomSection {
			customSections = append(customSections, line)
			if strings.TrimSpace(line) == "" && len(customSections) > 1 {
				inCustomSection = false
			}
		}
	}

	// Append custom sections to new summary if they exist
	if len(customSections) > 0 {
		newSummary += "\n\n## Preserved Custom Sections\n"
		newSummary += strings.Join(customSections, "\n")
	}

	return newSummary
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
		if _, err := s.readFile(file); err == nil {
			return true
		}
	}

	return false
}
