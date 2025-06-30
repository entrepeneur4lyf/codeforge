package context

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
)

// ProjectContextLoader handles loading project-specific context files
type ProjectContextLoader struct {
	config     *config.Config
	workingDir string
	cache      map[string]*ContextFile
	lastLoaded time.Time
	watcher    *ContextFileWatcher
	mutex      sync.RWMutex
}

// ContextFileWatcher handles file system watching for context files
type ContextFileWatcher struct {
	callbacks map[string]func(string)
	mutex     sync.RWMutex
	active    bool
}

// ContextFile represents a loaded context file
type ContextFile struct {
	Path         string    `json:"path"`
	Content      string    `json:"content"`
	Format       string    `json:"format"`
	Size         int64     `json:"size"`
	ModTime      time.Time `json:"mod_time"`
	LoadTime     time.Time `json:"load_time"`
	IsValid      bool      `json:"is_valid"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// NewProjectContextLoader creates a new project context loader
func NewProjectContextLoader(cfg *config.Config, workingDir string) *ProjectContextLoader {
	return &ProjectContextLoader{
		config:     cfg,
		workingDir: workingDir,
		cache:      make(map[string]*ContextFile),
	}
}

// LoadProjectContext loads all project-specific context files
func (pcl *ProjectContextLoader) LoadProjectContext() (*ProjectContext, error) {
	contextPaths := pcl.config.ContextPaths
	if len(contextPaths) == 0 {
		// Use default context paths
		contextPaths = []string{
			".codeforge/context.md",
			".codeforge/instructions.md",
			"CONTEXT.md",
			"INSTRUCTIONS.md",
			"README.md",
			"docs/context.md",
			"docs/instructions.md",
		}
	}

	log.Printf("Loading project context from %d paths", len(contextPaths))

	var contextFiles []*ContextFile
	var totalSize int64
	var errors []string

	for _, contextPath := range contextPaths {
		// Resolve relative paths
		fullPath := contextPath
		if !filepath.IsAbs(contextPath) {
			fullPath = filepath.Join(pcl.workingDir, contextPath)
		}

		contextFile, err := pcl.loadContextFile(fullPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", contextPath, err))
			continue
		}

		if contextFile != nil {
			contextFiles = append(contextFiles, contextFile)
			totalSize += contextFile.Size
		}
	}

	projectContext := &ProjectContext{
		Files:           contextFiles,
		TotalSize:       totalSize,
		LoadTime:        time.Now(),
		WorkingDir:      pcl.workingDir,
		Errors:          errors,
		CombinedContent: pcl.combineContextFiles(contextFiles),
	}

	pcl.lastLoaded = time.Now()
	log.Printf("Loaded %d context files, total size: %d bytes", len(contextFiles), totalSize)

	return projectContext, nil
}

// ProjectContext represents the combined project context
type ProjectContext struct {
	Files           []*ContextFile `json:"files"`
	TotalSize       int64          `json:"total_size"`
	LoadTime        time.Time      `json:"load_time"`
	WorkingDir      string         `json:"working_dir"`
	Errors          []string       `json:"errors,omitempty"`
	CombinedContent string         `json:"combined_content"`
}

// loadContextFile loads a single context file
func (pcl *ProjectContextLoader) loadContextFile(filePath string) (*ContextFile, error) {
	// Check cache first
	if cached, exists := pcl.cache[filePath]; exists {
		// Check if file has been modified
		if stat, err := os.Stat(filePath); err == nil {
			if stat.ModTime().Equal(cached.ModTime) {
				return cached, nil
			}
		}
	}

	// Check if file exists
	stat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // File doesn't exist, not an error
		}
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Check if it's a regular file
	if !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}

	// Check file size (limit to 1MB)
	if stat.Size() > 1024*1024 {
		return nil, fmt.Errorf("file too large: %d bytes", stat.Size())
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Determine format
	format := pcl.detectFormat(filePath, content)

	// Validate content
	isValid, errorMsg := pcl.validateContent(content, format)

	contextFile := &ContextFile{
		Path:         filePath,
		Content:      string(content),
		Format:       format,
		Size:         stat.Size(),
		ModTime:      stat.ModTime(),
		LoadTime:     time.Now(),
		IsValid:      isValid,
		ErrorMessage: errorMsg,
	}

	// Cache the file
	pcl.cache[filePath] = contextFile

	log.Printf("Loaded context file: %s (%s, %d bytes)", filePath, format, stat.Size())
	return contextFile, nil
}

// detectFormat detects the format of a context file
func (pcl *ProjectContextLoader) detectFormat(filePath string, content []byte) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".md", ".markdown":
		return "markdown"
	case ".txt":
		return "text"
	case ".context":
		return "context"
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	default:
		// Try to detect based on content
		contentStr := string(content)
		if strings.Contains(contentStr, "# ") || strings.Contains(contentStr, "## ") {
			return "markdown"
		}
		return "text"
	}
}

// validateContent validates the content of a context file
func (pcl *ProjectContextLoader) validateContent(content []byte, format string) (bool, string) {
	if len(content) == 0 {
		return false, "empty file"
	}

	// Check for binary content
	if pcl.isBinaryContent(content) {
		return false, "binary content detected"
	}

	// Format-specific validation
	switch format {
	case "markdown":
		return pcl.validateMarkdown(content)
	case "json":
		return pcl.validateJSON(content)
	case "yaml":
		return pcl.validateYAML(content)
	default:
		return true, ""
	}
}

// isBinaryContent checks if content appears to be binary
func (pcl *ProjectContextLoader) isBinaryContent(content []byte) bool {
	// Simple heuristic: if more than 30% of bytes are non-printable, consider it binary
	nonPrintable := 0
	for _, b := range content {
		if b < 32 && b != 9 && b != 10 && b != 13 { // Allow tab, LF, CR
			nonPrintable++
		}
	}

	if len(content) == 0 {
		return false
	}

	return float64(nonPrintable)/float64(len(content)) > 0.3
}

// validateMarkdown performs basic markdown validation
func (pcl *ProjectContextLoader) validateMarkdown(content []byte) (bool, string) {
	// Basic validation - check for common markdown issues
	contentStr := string(content)

	// Check for unmatched code blocks
	codeBlockCount := strings.Count(contentStr, "```")
	if codeBlockCount%2 != 0 {
		return false, "unmatched code blocks"
	}

	return true, ""
}

// validateJSON performs basic JSON validation
func (pcl *ProjectContextLoader) validateJSON(content []byte) (bool, string) {
	// For now, just check if it's valid UTF-8
	if !utf8.Valid(content) {
		return false, "invalid UTF-8"
	}
	return true, ""
}

// validateYAML performs basic YAML validation
func (pcl *ProjectContextLoader) validateYAML(content []byte) (bool, string) {
	// For now, just check if it's valid UTF-8
	if !utf8.Valid(content) {
		return false, "invalid UTF-8"
	}
	return true, ""
}

// combineContextFiles combines multiple context files into a single string
func (pcl *ProjectContextLoader) combineContextFiles(files []*ContextFile) string {
	if len(files) == 0 {
		return ""
	}

	var combined strings.Builder
	combined.WriteString("# Project Context\n\n")
	combined.WriteString("This context was automatically loaded from project-specific files.\n\n")

	for _, file := range files {
		if !file.IsValid {
			continue
		}

		// Add file header
		relPath, _ := filepath.Rel(pcl.workingDir, file.Path)
		combined.WriteString(fmt.Sprintf("## Context from %s\n\n", relPath))

		// Add content based on format
		switch file.Format {
		case "markdown":
			combined.WriteString(file.Content)
		case "text", "context":
			combined.WriteString(file.Content)
		default:
			// Wrap other formats in code blocks
			combined.WriteString(fmt.Sprintf("```%s\n%s\n```", file.Format, file.Content))
		}

		combined.WriteString("\n\n---\n\n")
	}

	return combined.String()
}

// GetContextSummary returns a summary of loaded context
func (pc *ProjectContext) GetContextSummary() string {
	if len(pc.Files) == 0 {
		return "No project context files found."
	}

	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Loaded %d context files:\n", len(pc.Files)))

	for _, file := range pc.Files {
		relPath, _ := filepath.Rel(pc.WorkingDir, file.Path)
		status := "✓"
		if !file.IsValid {
			status = "✗"
		}
		summary.WriteString(fmt.Sprintf("  %s %s (%s, %d bytes)\n", status, relPath, file.Format, file.Size))
	}

	if len(pc.Errors) > 0 {
		summary.WriteString("\nErrors:\n")
		for _, err := range pc.Errors {
			summary.WriteString(fmt.Sprintf("  • %s\n", err))
		}
	}

	return summary.String()
}

// RefreshIfNeeded refreshes the context if files have been modified
func (pcl *ProjectContextLoader) RefreshIfNeeded() (*ProjectContext, bool, error) {
	// Check if any cached files have been modified
	needsRefresh := false

	for filePath, cached := range pcl.cache {
		if stat, err := os.Stat(filePath); err == nil {
			if !stat.ModTime().Equal(cached.ModTime) {
				needsRefresh = true
				break
			}
		}
	}

	if !needsRefresh {
		return nil, false, nil
	}

	// Clear cache and reload
	pcl.cache = make(map[string]*ContextFile)
	context, err := pcl.LoadProjectContext()
	return context, true, err
}

// WatchContextFiles sets up file watching for context files
func (pcl *ProjectContextLoader) WatchContextFiles(callback func(*ProjectContext)) error {
	if pcl.watcher != nil && pcl.watcher.active {
		return fmt.Errorf("file watching already active")
	}

	pcl.watcher = &ContextFileWatcher{
		callbacks: make(map[string]func(string)),
		active:    true,
	}

	// Start polling for file changes
	go pcl.pollForChanges(callback)

	log.Printf("Context file watching started (polling mode)")
	return nil
}

// StopWatching stops file watching
func (pcl *ProjectContextLoader) StopWatching() {
	pcl.mutex.Lock()
	defer pcl.mutex.Unlock()

	if pcl.watcher != nil {
		pcl.watcher.active = false
		pcl.watcher = nil
		log.Printf("Context file watching stopped")
	}
}

// pollForChanges polls for file changes every 5 seconds
func (pcl *ProjectContextLoader) pollForChanges(callback func(*ProjectContext)) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		pcl.mutex.RLock()
		if pcl.watcher == nil || !pcl.watcher.active {
			pcl.mutex.RUnlock()
			return
		}
		pcl.mutex.RUnlock()

		// Check for changes
		context, changed, err := pcl.RefreshIfNeeded()
		if err != nil {
			log.Printf("Error checking for context file changes: %v", err)
			continue
		}

		if changed && callback != nil {
			log.Printf("Context files changed, triggering callback")
			callback(context)
		}
	}
}

// AddWatchCallback adds a callback for specific file changes
func (pcl *ProjectContextLoader) AddWatchCallback(filePath string, callback func(string)) {
	if pcl.watcher == nil {
		return
	}

	pcl.watcher.mutex.Lock()
	defer pcl.watcher.mutex.Unlock()

	pcl.watcher.callbacks[filePath] = callback
}

// RemoveWatchCallback removes a callback for specific file changes
func (pcl *ProjectContextLoader) RemoveWatchCallback(filePath string) {
	if pcl.watcher == nil {
		return
	}

	pcl.watcher.mutex.Lock()
	defer pcl.watcher.mutex.Unlock()

	delete(pcl.watcher.callbacks, filePath)
}
