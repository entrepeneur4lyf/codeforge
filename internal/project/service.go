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
	// Save comprehensive project overview
	if err := s.SaveProjectOverview(overview); err != nil {
		return fmt.Errorf("failed to save project overview: %w", err)
	}

	// Save concise project summary for context (AGENT.md only)
	if err := s.SaveProjectSummary(overview); err != nil {
		return fmt.Errorf("failed to save project summary: %w", err)
	}

	return nil
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
