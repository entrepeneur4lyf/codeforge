package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/models"
)

// GetAgentPrompt returns the appropriate prompt for the given agent and provider
func GetAgentPrompt(agentName config.AgentName, provider models.ModelProvider) string {
	basePrompt := ""
	switch agentName {
	case config.AgentCoder:
		basePrompt = CoderPrompt(provider)
	case config.AgentTitle:
		basePrompt = TitlePrompt(provider)
	case config.AgentTask:
		basePrompt = TaskPrompt(provider)
	case config.AgentSummarizer:
		basePrompt = SummarizerPrompt(provider)
	default:
		basePrompt = "You are a helpful assistant"
	}

	if agentName == config.AgentCoder || agentName == config.AgentTask {
		// Add context from project-specific instruction files if they exist
		contextContent := getContextFromPaths()
		if contextContent != "" {
			return fmt.Sprintf("%s\n\n# Project-Specific Context\nMake sure to follow the instructions in the context below\n%s", basePrompt, contextContent)
		}
	}
	return basePrompt
}

var (
	onceContext    sync.Once
	contextContent string
)

func getContextFromPaths() string {
	onceContext.Do(func() {
		cfg := config.Get()
		if cfg == nil {
			return
		}
		workDir := cfg.WorkingDir
		contextPaths := cfg.ContextPaths

		contextContent = processContextPaths(workDir, contextPaths)
	})

	return contextContent
}

func processContextPaths(workDir string, paths []string) string {
	if len(paths) == 0 {
		return ""
	}

	var (
		wg       sync.WaitGroup
		resultCh = make(chan string, len(paths)*10) // Buffer for multiple files
	)

	// Track processed files to avoid duplicates
	processedFiles := make(map[string]bool)
	var processedMutex sync.Mutex

	for _, path := range paths {
		wg.Add(1)
		go func(p string) {
			defer wg.Done()

			if strings.HasSuffix(p, "/") {
				// Directory path
				err := filepath.WalkDir(filepath.Join(workDir, p), func(path string, d os.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if !d.IsDir() {
						// Check if we've already processed this file (case-insensitive)
						processedMutex.Lock()
						lowerPath := strings.ToLower(path)
						if !processedFiles[lowerPath] {
							processedFiles[lowerPath] = true
							processedMutex.Unlock()

							if result := processFile(path); result != "" {
								select {
								case resultCh <- result:
								default:
									// Channel full, skip this file
								}
							}
						} else {
							processedMutex.Unlock()
						}
					}
					return nil
				})
				if err != nil {
					// Log error but continue processing other paths
					return
				}
			} else {
				// Single file path
				fullPath := filepath.Join(workDir, p)

				// Check if we've already processed this file (case-insensitive)
				processedMutex.Lock()
				lowerPath := strings.ToLower(fullPath)
				if !processedFiles[lowerPath] {
					processedFiles[lowerPath] = true
					processedMutex.Unlock()

					result := processFile(fullPath)
					if result != "" {
						select {
						case resultCh <- result:
						default:
							// Channel full, skip this file
						}
					}
				} else {
					processedMutex.Unlock()
				}
			}
		}(path)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := make([]string, 0)
	for result := range resultCh {
		results = append(results, result)
	}

	return strings.Join(results, "\n")
}

func processFile(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return ""
	}

	// Limit file size to prevent overwhelming the context
	const maxFileSize = 10000 // 10KB limit
	if len(content) > maxFileSize {
		content = content[:maxFileSize]
		return fmt.Sprintf("# From: %s (truncated)\n%s\n... [file truncated]", filePath, string(content))
	}

	return fmt.Sprintf("# From: %s\n%s", filePath, string(content))
}
