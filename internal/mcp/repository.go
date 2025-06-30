package mcp

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

const (
	// Official MCP servers repository README URL
	MCPServersReadmeURL = "https://raw.githubusercontent.com/modelcontextprotocol/servers/main/README.md"

	// Cache duration for repository data
	CacheDuration = 1 * time.Hour
)

// RepositoryFetcher handles fetching and parsing MCP server data from the official repository
type RepositoryFetcher struct {
	httpClient    *http.Client
	cachedData    []MCPServerInfo
	lastFetched   time.Time
	cacheDuration time.Duration
}

// MCPServerInfo represents parsed server information from the repository
type MCPServerInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Category    string   `json:"category"`
	InstallCmd  string   `json:"install_cmd"`
	GitHubURL   string   `json:"github_url"`
	Tags        []string `json:"tags"`
	Language    string   `json:"language"`
	PackageType string   `json:"package_type"` // npm, python, docker, binary
}

// NewRepositoryFetcher creates a new repository fetcher
func NewRepositoryFetcher() *RepositoryFetcher {
	return &RepositoryFetcher{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheDuration: CacheDuration,
	}
}

// FetchServers fetches and parses MCP servers from the official repository
func (rf *RepositoryFetcher) FetchServers() ([]MCPServerInfo, error) {
	// Check cache first
	if rf.isCacheValid() {
		log.Debug("Using cached MCP server data")
		return rf.cachedData, nil
	}

	log.Info("Fetching MCP servers from repository", "url", MCPServersReadmeURL)

	// Fetch README.md content
	readmeContent, err := rf.fetchReadme()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch README: %w", err)
	}

	// Parse servers from README
	servers, err := rf.parseReadme(readmeContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse README: %w", err)
	}

	// Update cache
	rf.cachedData = servers
	rf.lastFetched = time.Now()

	log.Info("Successfully fetched MCP servers", "count", len(servers))
	return servers, nil
}

// fetchReadme fetches the raw README.md content from GitHub
func (rf *RepositoryFetcher) fetchReadme() (string, error) {
	req, err := http.NewRequest("GET", MCPServersReadmeURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to avoid rate limiting
	req.Header.Set("User-Agent", "CodeForge-MCP-Client/1.0")
	req.Header.Set("Accept", "text/plain")

	resp, err := rf.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch README: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// parseReadme parses the README.md content to extract MCP server information
func (rf *RepositoryFetcher) parseReadme(content string) ([]MCPServerInfo, error) {
	var servers []MCPServerInfo

	// Split content into sections
	sections := rf.extractSections(content)

	for category, sectionContent := range sections {
		categoryServers := rf.parseSection(category, sectionContent)
		servers = append(servers, categoryServers...)
	}

	return servers, nil
}

// extractSections extracts different categories of servers from the README
func (rf *RepositoryFetcher) extractSections(content string) map[string]string {
	sections := make(map[string]string)

	// Common section patterns in MCP servers README
	sectionPatterns := map[string]*regexp.Regexp{
		"Official":    regexp.MustCompile(`(?s)## Official.*?(?=##|$)`),
		"Reference":   regexp.MustCompile(`(?s)## Reference.*?(?=##|$)`),
		"Third-party": regexp.MustCompile(`(?s)## Third-party.*?(?=##|$)`),
		"Community":   regexp.MustCompile(`(?s)## Community.*?(?=##|$)`),
		"Frameworks":  regexp.MustCompile(`(?s)## Frameworks.*?(?=##|$)`),
	}

	for category, pattern := range sectionPatterns {
		matches := pattern.FindString(content)
		if matches != "" {
			sections[category] = matches
		}
	}

	// If no sections found, treat entire content as one section
	if len(sections) == 0 {
		sections["General"] = content
	}

	return sections
}

// parseSection parses a specific section to extract server information
func (rf *RepositoryFetcher) parseSection(category, content string) []MCPServerInfo {
	var servers []MCPServerInfo

	// Regex patterns to match server entries
	// Pattern for entries like: ### [server-name](github-url) - Description
	serverPattern := regexp.MustCompile(`### \[([^\]]+)\]\(([^)]+)\)\s*-?\s*(.*)`)

	// Pattern for installation commands
	installPattern := regexp.MustCompile(`(?i)(?:npm install|pip install|docker run|npx|uvx)\s+([^\s\n]+)`)

	lines := strings.Split(content, "\n")

	for i, line := range lines {
		matches := serverPattern.FindStringSubmatch(line)
		if len(matches) >= 4 {
			server := MCPServerInfo{
				Name:        strings.TrimSpace(matches[1]),
				GitHubURL:   strings.TrimSpace(matches[2]),
				Description: strings.TrimSpace(matches[3]),
				Category:    category,
				Author:      rf.extractAuthorFromURL(matches[2]),
				Language:    rf.DetectLanguage(matches[2]),
				PackageType: rf.detectPackageType(matches[2]),
			}

			// Look for installation command in the next few lines
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				if installMatches := installPattern.FindStringSubmatch(lines[j]); len(installMatches) > 1 {
					server.InstallCmd = strings.TrimSpace(lines[j])
					break
				}
			}

			// Extract tags from description
			server.Tags = rf.extractTags(server.Description)

			servers = append(servers, server)
		}
	}

	return servers
}

// extractAuthorFromURL extracts the author/organization from a GitHub URL
func (rf *RepositoryFetcher) extractAuthorFromURL(url string) string {
	// Pattern to match GitHub URLs: https://github.com/author/repo
	pattern := regexp.MustCompile(`github\.com/([^/]+)`)
	matches := pattern.FindStringSubmatch(url)
	if len(matches) >= 2 {
		return matches[1]
	}
	return "Unknown"
}

// DetectLanguage detects the programming language from GitHub URL and repository content
func (rf *RepositoryFetcher) DetectLanguage(githubURL string) string {
	// Enhanced language detection using multiple heuristics

	// URL-based detection patterns
	urlPatterns := map[string][]string{
		"Python":     {"python", "py", "django", "flask", "fastapi", "pytorch", "tensorflow"},
		"JavaScript": {"node", "js", "javascript", "typescript", "react", "vue", "angular", "npm"},
		"TypeScript": {"typescript", "ts", "angular", "nest"},
		"Go":         {"go", "golang", "gin", "echo", "fiber"},
		"Rust":       {"rust", "rs", "cargo", "actix", "tokio", "serde"},
		"Java":       {"java", "spring", "maven", "gradle", "android"},
		"C++":        {"cpp", "c++", "cmake", "opencv", "boost"},
		"C":          {"/c/", "c-", "gcc", "clang"},
		"PHP":        {"php", "laravel", "symfony", "composer", "wordpress"},
		"Ruby":       {"ruby", "rails", "gem", "bundler"},
		"Swift":      {"swift", "ios", "xcode", "cocoapods"},
		"Kotlin":     {"kotlin", "android", "ktor"},
		"C#":         {"csharp", "dotnet", "asp.net", "unity"},
		"Scala":      {"scala", "akka", "play"},
		"Clojure":    {"clojure", "clj", "leiningen"},
		"Haskell":    {"haskell", "hs", "cabal", "stack"},
		"Erlang":     {"erlang", "elixir", "phoenix", "otp"},
		"Lua":        {"lua", "love2d", "openresty"},
		"R":          {"/r/", "r-", "rstudio", "cran"},
		"Julia":      {"julia", "jl"},
		"Dart":       {"dart", "flutter"},
		"Shell":      {"bash", "shell", "zsh", "fish"},
	}

	// Convert URL to lowercase for case-insensitive matching
	lowerURL := strings.ToLower(githubURL)

	// Check each language pattern
	for language, patterns := range urlPatterns {
		for _, pattern := range patterns {
			if strings.Contains(lowerURL, pattern) {
				return language
			}
		}
	}

	// Repository name-based detection
	repoName := rf.extractRepoName(githubURL)
	if repoName != "" {
		lowerRepoName := strings.ToLower(repoName)

		// Check for language-specific naming conventions
		if strings.HasSuffix(lowerRepoName, ".py") || strings.HasSuffix(lowerRepoName, "-python") {
			return "Python"
		}
		if strings.HasSuffix(lowerRepoName, ".js") || strings.HasSuffix(lowerRepoName, "-js") || strings.HasSuffix(lowerRepoName, "-node") {
			return "JavaScript"
		}
		if strings.HasSuffix(lowerRepoName, ".go") || strings.HasSuffix(lowerRepoName, "-go") {
			return "Go"
		}
		if strings.HasSuffix(lowerRepoName, ".rs") || strings.HasSuffix(lowerRepoName, "-rust") {
			return "Rust"
		}
	}

	return "Unknown"
}

// extractRepoName extracts the repository name from a GitHub URL
func (rf *RepositoryFetcher) extractRepoName(githubURL string) string {
	// Extract repo name from URLs like: https://github.com/user/repo
	parts := strings.Split(githubURL, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}
	return ""
}

// detectPackageType detects the package type from GitHub URL or installation command
func (rf *RepositoryFetcher) detectPackageType(githubURL string) string {
	if strings.Contains(githubURL, "npm") {
		return "npm"
	}
	if strings.Contains(githubURL, "python") || strings.Contains(githubURL, "pypi") {
		return "python"
	}
	if strings.Contains(githubURL, "docker") {
		return "docker"
	}
	return "binary"
}

// extractTags extracts relevant tags from the description
func (rf *RepositoryFetcher) extractTags(description string) []string {
	var tags []string
	desc := strings.ToLower(description)

	// Common tag patterns
	tagPatterns := map[string][]string{
		"database":     {"database", "db", "sql", "postgres", "mysql", "sqlite"},
		"api":          {"api", "rest", "graphql", "http"},
		"cloud":        {"aws", "azure", "gcp", "cloud"},
		"ai":           {"ai", "ml", "machine learning", "openai", "anthropic"},
		"development":  {"dev", "development", "coding", "programming"},
		"productivity": {"productivity", "automation", "workflow"},
		"monitoring":   {"monitoring", "logging", "metrics", "observability"},
		"security":     {"security", "auth", "authentication", "encryption"},
	}

	for tag, patterns := range tagPatterns {
		for _, pattern := range patterns {
			if strings.Contains(desc, pattern) {
				tags = append(tags, tag)
				break
			}
		}
	}

	return tags
}

// isCacheValid checks if the cached data is still valid
func (rf *RepositoryFetcher) isCacheValid() bool {
	if rf.cachedData == nil || len(rf.cachedData) == 0 {
		return false
	}
	return time.Since(rf.lastFetched) < rf.cacheDuration
}

// ClearCache clears the cached repository data
func (rf *RepositoryFetcher) ClearCache() {
	rf.cachedData = nil
	rf.lastFetched = time.Time{}
}

// SetCacheDuration sets the cache duration
func (rf *RepositoryFetcher) SetCacheDuration(duration time.Duration) {
	rf.cacheDuration = duration
}

// GetCachedData returns the cached data if available
func (rf *RepositoryFetcher) GetCachedData() []MCPServerInfo {
	if rf.isCacheValid() {
		return rf.cachedData
	}
	return nil
}
