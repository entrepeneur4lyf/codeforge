package project

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RepositoryAnalyzer analyzes repository structure
type RepositoryAnalyzer struct {
	root         string
	ignorePaths  map[string]bool
	maxDepth     int
	maxFiles     int
	fileCount    int
}

// NewRepositoryAnalyzer creates a new repository analyzer
func NewRepositoryAnalyzer(root string) *RepositoryAnalyzer {
	return &RepositoryAnalyzer{
		root:     root,
		maxDepth: 5,
		maxFiles: 1000,
		ignorePaths: map[string]bool{
			".git":          true,
			"node_modules":  true,
			"vendor":        true,
			"target":        true,
			"dist":          true,
			"build":         true,
			".next":         true,
			"__pycache__":   true,
			".pytest_cache": true,
			".venv":         true,
			"venv":          true,
			".idea":         true,
			".vscode":       true,
		},
	}
}

// RepoMap represents the repository structure
type RepoMap struct {
	Root        string              `json:"root"`
	Directories []*DirectoryInfo    `json:"directories"`
	Files       []*FileInfo         `json:"files"`
	Summary     *RepoSummary        `json:"summary"`
	GeneratedAt time.Time           `json:"generated_at"`
}

// DirectoryInfo represents directory information
type DirectoryInfo struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
	Depth        int    `json:"depth"`
	FileCount    int    `json:"file_count"`
	Purpose      string `json:"purpose,omitempty"`
}

// FileInfo represents file information
type FileInfo struct {
	Path         string `json:"path"`
	RelativePath string `json:"relative_path"`
	Size         int64  `json:"size"`
	Extension    string `json:"extension"`
	Language     string `json:"language"`
	Purpose      string `json:"purpose,omitempty"`
}

// RepoSummary provides repository statistics
type RepoSummary struct {
	TotalDirectories int            `json:"total_directories"`
	TotalFiles       int            `json:"total_files"`
	TotalSize        int64          `json:"total_size"`
	Languages        map[string]int `json:"languages"`
	FileTypes        map[string]int `json:"file_types"`
}

// GenerateRepoMap generates a repository map
func (ra *RepositoryAnalyzer) GenerateRepoMap() (*RepoMap, error) {
	repoMap := &RepoMap{
		Root:        ra.root,
		Directories: make([]*DirectoryInfo, 0),
		Files:       make([]*FileInfo, 0),
		Summary: &RepoSummary{
			Languages: make(map[string]int),
			FileTypes: make(map[string]int),
		},
		GeneratedAt: time.Now(),
	}

	// Walk the directory tree
	err := filepath.WalkDir(ra.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Get relative path
		relPath, err := filepath.Rel(ra.root, path)
		if err != nil {
			return nil
		}

		// Calculate depth
		depth := strings.Count(relPath, string(os.PathSeparator))
		if depth > ra.maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if should ignore
		if ra.shouldIgnore(relPath, d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() {
			// Process directory
			dirInfo := &DirectoryInfo{
				Path:         path,
				RelativePath: relPath,
				Depth:        depth,
				Purpose:      ra.inferDirectoryPurpose(relPath, d.Name()),
			}
			repoMap.Directories = append(repoMap.Directories, dirInfo)
			repoMap.Summary.TotalDirectories++
		} else {
			// Process file
			if ra.fileCount >= ra.maxFiles {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}

			ext := filepath.Ext(d.Name())
			lang := ra.detectLanguage(ext, d.Name())

			fileInfo := &FileInfo{
				Path:         path,
				RelativePath: relPath,
				Size:         info.Size(),
				Extension:    ext,
				Language:     lang,
				Purpose:      ra.inferFilePurpose(relPath, d.Name()),
			}

			repoMap.Files = append(repoMap.Files, fileInfo)
			repoMap.Summary.TotalFiles++
			repoMap.Summary.TotalSize += info.Size()
			ra.fileCount++

			// Update language statistics
			if lang != "" {
				repoMap.Summary.Languages[lang]++
			}

			// Update file type statistics
			if ext != "" {
				repoMap.Summary.FileTypes[ext]++
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	// Update directory file counts
	ra.updateDirectoryFileCounts(repoMap)

	return repoMap, nil
}

// shouldIgnore checks if a path should be ignored
func (ra *RepositoryAnalyzer) shouldIgnore(relPath, name string) bool {
	// Check exact name matches
	if ra.ignorePaths[name] {
		return true
	}

	// Check patterns
	if strings.HasPrefix(name, ".") && name != "." {
		// Allow some dotfiles
		allowedDotfiles := map[string]bool{
			".github":     true,
			".gitignore":  true,
			".env":        true,
			".codeforge":  true,
		}
		if !allowedDotfiles[name] {
			return true
		}
	}

	// Check if it's a test file we should skip
	if strings.Contains(relPath, "test_data") || strings.Contains(relPath, "testdata") {
		return true
	}

	return false
}

// detectLanguage detects the programming language from file extension
func (ra *RepositoryAnalyzer) detectLanguage(ext, filename string) string {
	// Handle special filenames
	specialFiles := map[string]string{
		"Dockerfile":     "docker",
		"Makefile":       "make",
		"Rakefile":       "ruby",
		"Gemfile":        "ruby",
		"requirements.txt": "python",
		"package.json":   "javascript",
		"go.mod":         "go",
		"Cargo.toml":     "rust",
		"pom.xml":        "java",
		"build.gradle":   "java",
	}

	if lang, ok := specialFiles[filename]; ok {
		return lang
	}

	// Map extensions to languages
	extMap := map[string]string{
		".go":    "go",
		".py":    "python",
		".js":    "javascript",
		".ts":    "typescript",
		".jsx":   "javascript",
		".tsx":   "typescript",
		".java":  "java",
		".c":     "c",
		".cpp":   "cpp",
		".cc":    "cpp",
		".cxx":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
		".rb":    "ruby",
		".rs":    "rust",
		".php":   "php",
		".swift": "swift",
		".kt":    "kotlin",
		".scala": "scala",
		".r":     "r",
		".R":     "r",
		".sh":    "shell",
		".bash":  "shell",
		".zsh":   "shell",
		".fish":  "shell",
		".ps1":   "powershell",
		".lua":   "lua",
		".vim":   "vim",
		".el":    "elisp",
		".clj":   "clojure",
		".dart":  "dart",
		".ml":    "ocaml",
		".hs":    "haskell",
		".ex":    "elixir",
		".exs":   "elixir",
		".erl":   "erlang",
		".hrl":   "erlang",
		".vue":   "vue",
		".svelte": "svelte",
		".md":    "markdown",
		".rst":   "restructuredtext",
		".tex":   "latex",
		".sql":   "sql",
		".html":  "html",
		".htm":   "html",
		".xml":   "xml",
		".css":   "css",
		".scss":  "scss",
		".sass":  "sass",
		".less":  "less",
		".json":  "json",
		".yaml":  "yaml",
		".yml":   "yaml",
		".toml":  "toml",
		".ini":   "ini",
		".cfg":   "ini",
		".conf":  "conf",
	}

	if lang, ok := extMap[strings.ToLower(ext)]; ok {
		return lang
	}

	return ""
}

// inferDirectoryPurpose infers the purpose of a directory
func (ra *RepositoryAnalyzer) inferDirectoryPurpose(relPath, name string) string {
	purposes := map[string]string{
		"cmd":           "Command-line interfaces",
		"internal":      "Internal packages",
		"pkg":           "Public packages",
		"api":           "API definitions",
		"web":           "Web interface",
		"docs":          "Documentation",
		"test":          "Test files",
		"tests":         "Test files",
		"scripts":       "Build and utility scripts",
		"config":        "Configuration files",
		"configs":       "Configuration files",
		"migrations":    "Database migrations",
		"models":        "Data models",
		"controllers":   "Request handlers",
		"views":         "View templates",
		"templates":     "Template files",
		"static":        "Static assets",
		"assets":        "Static assets",
		"public":        "Public files",
		"src":           "Source code",
		"lib":           "Library code",
		"bin":           "Binary files",
		"dist":          "Distribution files",
		"build":         "Build output",
		"vendor":        "Third-party dependencies",
		"node_modules":  "Node.js dependencies",
		".github":       "GitHub configuration",
		"docker":        "Docker configuration",
		"k8s":           "Kubernetes configuration",
		"kubernetes":    "Kubernetes configuration",
		"terraform":     "Infrastructure as code",
		"ansible":       "Configuration management",
	}

	// Check exact matches
	if purpose, ok := purposes[strings.ToLower(name)]; ok {
		return purpose
	}

	// Check path components
	parts := strings.Split(relPath, string(os.PathSeparator))
	for _, part := range parts {
		if purpose, ok := purposes[strings.ToLower(part)]; ok {
			return purpose
		}
	}

	return ""
}

// inferFilePurpose infers the purpose of a file
func (ra *RepositoryAnalyzer) inferFilePurpose(relPath, name string) string {
	purposes := map[string]string{
		"main.go":         "Application entry point",
		"main.py":         "Application entry point",
		"index.js":        "Module entry point",
		"index.ts":        "Module entry point",
		"app.js":          "Application file",
		"app.py":          "Application file",
		"server.js":       "Server implementation",
		"server.py":       "Server implementation",
		"router.go":       "Request routing",
		"routes.js":       "Request routing",
		"models.py":       "Data models",
		"schema.sql":      "Database schema",
		"Dockerfile":      "Container definition",
		"docker-compose.yml": "Container orchestration",
		"Makefile":        "Build configuration",
		"package.json":    "Node.js project configuration",
		"go.mod":          "Go module definition",
		"requirements.txt": "Python dependencies",
		"Cargo.toml":      "Rust project configuration",
		"README.md":       "Project documentation",
		"LICENSE":         "License information",
		".gitignore":      "Git ignore rules",
		".env":            "Environment variables",
		"config.json":     "Application configuration",
		"settings.py":     "Application settings",
	}

	// Check exact matches
	if purpose, ok := purposes[name]; ok {
		return purpose
	}

	// Check patterns
	lowerName := strings.ToLower(name)
	if strings.HasSuffix(lowerName, "_test.go") {
		return "Test file"
	}
	if strings.HasSuffix(lowerName, ".test.js") || strings.HasSuffix(lowerName, ".spec.js") {
		return "Test file"
	}
	if strings.HasSuffix(lowerName, "_test.py") || strings.HasPrefix(lowerName, "test_") {
		return "Test file"
	}
	if strings.Contains(lowerName, "config") {
		return "Configuration file"
	}
	if strings.Contains(lowerName, "util") || strings.Contains(lowerName, "helper") {
		return "Utility functions"
	}

	return ""
}

// updateDirectoryFileCounts updates file counts for directories
func (ra *RepositoryAnalyzer) updateDirectoryFileCounts(repoMap *RepoMap) {
	dirCounts := make(map[string]int)

	// Count files per directory
	for _, file := range repoMap.Files {
		dir := filepath.Dir(file.RelativePath)
		for dir != "." && dir != "" {
			dirCounts[dir]++
			dir = filepath.Dir(dir)
		}
	}

	// Update directory info
	for _, dirInfo := range repoMap.Directories {
		if count, ok := dirCounts[dirInfo.RelativePath]; ok {
			dirInfo.FileCount = count
		}
	}
}

// GenerateMarkdown generates a markdown representation of the repository map
func (rm *RepoMap) GenerateMarkdown() string {
	var sb strings.Builder

	sb.WriteString("# Repository Structure\n\n")
	sb.WriteString(fmt.Sprintf("**Root**: %s\n", rm.Root))
	sb.WriteString(fmt.Sprintf("**Generated**: %s\n\n", rm.GeneratedAt.Format("2006-01-02 15:04:05")))

	// Summary
	sb.WriteString("## Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Directories**: %d\n", rm.Summary.TotalDirectories))
	sb.WriteString(fmt.Sprintf("- **Files**: %d\n", rm.Summary.TotalFiles))
	sb.WriteString(fmt.Sprintf("- **Total Size**: %s\n\n", formatBytes(rm.Summary.TotalSize)))

	// Languages
	if len(rm.Summary.Languages) > 0 {
		sb.WriteString("### Languages\n\n")
		for lang, count := range rm.Summary.Languages {
			sb.WriteString(fmt.Sprintf("- %s: %d files\n", lang, count))
		}
		sb.WriteString("\n")
	}

	// Key directories
	sb.WriteString("## Key Directories\n\n")
	for _, dir := range rm.Directories {
		if dir.Purpose != "" && dir.Depth <= 2 {
			sb.WriteString(fmt.Sprintf("- **%s**: %s", dir.RelativePath, dir.Purpose))
			if dir.FileCount > 0 {
				sb.WriteString(fmt.Sprintf(" (%d files)", dir.FileCount))
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("\n")

	// Key files
	sb.WriteString("## Key Files\n\n")
	for _, file := range rm.Files {
		if file.Purpose != "" {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", file.RelativePath, file.Purpose))
		}
	}

	return sb.String()
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}