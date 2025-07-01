package utils

import (
	"os"
	"path/filepath"

	gitignore "github.com/sabhiram/go-gitignore"
)

// GitIgnoreFilter provides gitignore-aware file filtering
type GitIgnoreFilter struct {
	ignore      *gitignore.GitIgnore
	projectRoot string
}

// NewGitIgnoreFilter creates a new gitignore filter for the given project root
func NewGitIgnoreFilter(projectRoot string) *GitIgnoreFilter {
	filter := &GitIgnoreFilter{
		projectRoot: projectRoot,
	}

	// Load .gitignore patterns
	filter.loadGitIgnorePatterns()

	return filter
}

// loadGitIgnorePatterns loads patterns from .gitignore files
func (g *GitIgnoreFilter) loadGitIgnorePatterns() {
	var patterns []string

	// Always ignore .git directory
	patterns = append(patterns, ".git/")

	// Load from .gitignore file if it exists
	gitignorePath := filepath.Join(g.projectRoot, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		if gitIgnore, err := gitignore.CompileIgnoreFile(gitignorePath); err == nil {
			g.ignore = gitIgnore
			return
		}
	}

	// Load from .git/info/exclude if it exists
	excludePath := filepath.Join(g.projectRoot, ".git", "info", "exclude")
	if _, err := os.Stat(excludePath); err == nil {
		if gitIgnore, err := gitignore.CompileIgnoreFile(excludePath); err == nil {
			g.ignore = gitIgnore
			return
		}
	}

	// Fallback to common ignore patterns if no .gitignore files found
	patterns = append(patterns, getDefaultIgnorePatterns()...)

	g.ignore = gitignore.CompileIgnoreLines(patterns...)
}

// IsIgnored checks if a file path should be ignored
func (g *GitIgnoreFilter) IsIgnored(path string) bool {
	if g.ignore == nil {
		return false
	}

	// Convert absolute path to relative path from project root
	relPath, err := filepath.Rel(g.projectRoot, path)
	if err != nil {
		// If we can't get relative path, use the path as-is
		relPath = path
	}

	// Normalize path separators for cross-platform compatibility
	relPath = filepath.ToSlash(relPath)

	return g.ignore.MatchesPath(relPath)
}

// ShouldIgnoreFile checks if a file should be ignored based on its name and path
func (g *GitIgnoreFilter) ShouldIgnoreFile(filePath string) bool {
	return g.IsIgnored(filePath)
}

// ShouldIgnoreDirectory checks if a directory should be ignored
func (g *GitIgnoreFilter) ShouldIgnoreDirectory(dirPath string) bool {
	return g.IsIgnored(dirPath)
}

// getDefaultIgnorePatterns returns common ignore patterns when no .gitignore is found
func getDefaultIgnorePatterns() []string {
	return []string{
		// Version control
		".git/",
		".svn/",
		".hg/",

		// Dependencies
		"node_modules/",
		"vendor/",
		"target/",

		// IDE files
		".vscode/",
		".idea/",
		".vs/",
		"*.swp",
		"*.swo",
		"*~",

		// Build outputs
		"build/",
		"dist/",
		"out/",
		"bin/",

		// Language-specific
		"__pycache__/",
		".pytest_cache/",
		"*.pyc",
		"*.pyo",
		"*.pyd",
		"*.class",
		"*.jar",
		"*.war",
		"*.exe",
		"*.dll",
		"*.so",
		"*.dylib",

		// Logs and temporary files
		"*.log",
		"*.tmp",
		"*.temp",
		".DS_Store",
		"Thumbs.db",

		// Environment files
		".env",
		".env.local",
		".env.*.local",

		// CodeForge specific
		".codeforge/",
	}
}

// WalkWithGitIgnore walks a directory tree while respecting .gitignore patterns
func (g *GitIgnoreFilter) WalkWithGitIgnore(root string, walkFn filepath.WalkFunc) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Check if this path should be ignored
		if g.IsIgnored(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		return walkFn(path, info, err)
	})
}
