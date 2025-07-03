package search

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

// Result represents a search result
type Result struct {
	Path     string
	Line     int
	Column   int
	Content  string
	Match    string
	Score    float64 // Relevance score for fuzzy matches
}

// Options configures search behavior
type Options struct {
	// Search parameters
	Query         string
	Path          string
	CaseSensitive bool
	WholeWord     bool
	Regex         bool
	
	// File filters
	Include       []string // File patterns to include
	Exclude       []string // File patterns to exclude
	MaxFileSize   int64    // Maximum file size to search (0 = no limit)
	
	// Result limits
	MaxResults    int
	MaxLineLength int
	
	// Fuzzy search options
	UseFuzzy      bool
	FuzzyThreshold int // Minimum score for fuzzy matches (0-100)
}

// Searcher provides search functionality
type Searcher struct {
	useRipgrep bool
	rgPath     string
	cache      *resultCache
}

// NewSearcher creates a new searcher
func NewSearcher() *Searcher {
	s := &Searcher{
		cache: newResultCache(1000),
	}
	
	// Check if ripgrep is available
	if path, err := exec.LookPath("rg"); err == nil {
		s.useRipgrep = true
		s.rgPath = path
	}
	
	return s
}

// Search performs a search with the given options
func (s *Searcher) Search(ctx context.Context, opts Options) ([]Result, error) {
	// Check context first
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	// Check cache
	cacheKey := s.cache.generateKey(opts)
	if cached, found := s.cache.get(cacheKey); found {
		return cached, nil
	}
	
	var results []Result
	var err error
	
	if s.useRipgrep && !opts.UseFuzzy {
		// Use ripgrep for exact/regex searches
		results, err = s.searchWithRipgrep(ctx, opts)
	} else {
		// Use built-in search for fuzzy matching or when ripgrep is not available
		results, err = s.searchBuiltin(ctx, opts)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Cache results
	s.cache.set(cacheKey, results)
	
	return results, nil
}

// SearchFiles searches for files matching the query
func (s *Searcher) SearchFiles(ctx context.Context, query, path string) ([]string, error) {
	var files []string
	
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		if d.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Fuzzy match on filename
		filename := d.Name()
		if fuzzy.MatchFold(query, filename) {
			files = append(files, path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Sort by fuzzy match score
	ranks := fuzzy.RankFindFold(query, files)
	sortedFiles := make([]string, len(ranks))
	for i, rank := range ranks {
		sortedFiles[i] = rank.Target
	}
	files = sortedFiles
	
	return files, nil
}

// searchWithRipgrep uses ripgrep for fast searching
func (s *Searcher) searchWithRipgrep(ctx context.Context, opts Options) ([]Result, error) {
	args := []string{
		"--json",           // JSON output for parsing
		"--max-count", fmt.Sprintf("%d", opts.MaxResults),
		"--max-columns", fmt.Sprintf("%d", opts.MaxLineLength),
	}
	
	// Add search options
	if !opts.CaseSensitive {
		args = append(args, "-i")
	}
	
	if opts.WholeWord {
		args = append(args, "-w")
	}
	
	if opts.Regex {
		args = append(args, "-e")
	} else {
		args = append(args, "-F") // Fixed string
	}
	
	// Add file filters
	for _, pattern := range opts.Include {
		args = append(args, "-g", pattern)
	}
	
	for _, pattern := range opts.Exclude {
		args = append(args, "-g", "!"+pattern)
	}
	
	// Add query and path
	args = append(args, opts.Query)
	if opts.Path != "" {
		args = append(args, opts.Path)
	}
	
	// Execute ripgrep
	cmd := exec.CommandContext(ctx, s.rgPath, args...)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 1 means no matches found
			if exitErr.ExitCode() == 1 {
				return []Result{}, nil
			}
		}
		return nil, fmt.Errorf("ripgrep failed: %w", err)
	}
	
	// Parse JSON output
	return s.parseRipgrepOutput(output)
}

// searchBuiltin uses built-in Go code for searching
func (s *Searcher) searchBuiltin(ctx context.Context, opts Options) ([]Result, error) {
	var results []Result
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	// Create a worker pool
	workerCount := 4
	fileChan := make(chan string, 100)
	
	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.searchWorker(ctx, opts, fileChan, &results, &mu)
		}()
	}
	
	// Walk directory and send files to workers
	err := filepath.WalkDir(opts.Path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		if d.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}
		
		// Check file filters
		if !s.matchesFilters(path, opts) {
			return nil
		}
		
		// Send to worker
		select {
		case fileChan <- path:
		case <-ctx.Done():
			return ctx.Err()
		}
		
		return nil
	})
	
	close(fileChan)
	wg.Wait()
	
	if err != nil {
		return nil, err
	}
	
	// Sort results by relevance if using fuzzy search
	if opts.UseFuzzy {
		s.rankResults(&results, opts.Query)
	}
	
	// Limit results
	if opts.MaxResults > 0 && len(results) > opts.MaxResults {
		results = results[:opts.MaxResults]
	}
	
	return results, nil
}

// searchWorker processes files in parallel
func (s *Searcher) searchWorker(ctx context.Context, opts Options, fileChan <-chan string, results *[]Result, mu *sync.Mutex) {
	for path := range fileChan {
		select {
		case <-ctx.Done():
			return
		default:
		}
		
		fileResults := s.searchFile(ctx, path, opts)
		
		if len(fileResults) > 0 {
			mu.Lock()
			*results = append(*results, fileResults...)
			mu.Unlock()
		}
	}
}

// searchFile searches within a single file
func (s *Searcher) searchFile(ctx context.Context, path string, opts Options) []Result {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()
	
	// Check file size
	if opts.MaxFileSize > 0 {
		info, err := file.Stat()
		if err != nil || info.Size() > opts.MaxFileSize {
			return nil
		}
	}
	
	var results []Result
	scanner := bufio.NewScanner(file)
	lineNum := 0
	
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		
		// Check context
		select {
		case <-ctx.Done():
			return results
		default:
		}
		
		// Limit line length
		if opts.MaxLineLength > 0 && len(line) > opts.MaxLineLength {
			line = line[:opts.MaxLineLength] + "..."
		}
		
		// Search for matches
		if opts.UseFuzzy {
			// Fuzzy search
			if fuzzy.MatchFold(opts.Query, line) {
				score := fuzzy.RankMatchFold(opts.Query, line)
				if score >= 0 { // Fuzzy search library doesn't use thresholds like this
					results = append(results, Result{
						Path:    path,
						Line:    lineNum,
						Content: line,
						Match:   opts.Query,
						Score:   float64(score) / 100.0,
					})
				}
			}
		} else {
			// Exact search
			var idx int
			if opts.CaseSensitive {
				idx = strings.Index(line, opts.Query)
			} else {
				idx = strings.Index(strings.ToLower(line), strings.ToLower(opts.Query))
			}
			
			if idx != -1 {
				// Extract the actual match from the original line
				match := line[idx : idx+len(opts.Query)]
				results = append(results, Result{
					Path:    path,
					Line:    lineNum,
					Column:  idx + 1,
					Content: line,
					Match:   match,
					Score:   1.0,
				})
			}
		}
	}
	
	return results
}

// matchesFilters checks if a file matches the include/exclude filters
func (s *Searcher) matchesFilters(path string, opts Options) bool {
	// Check exclude patterns first
	for _, pattern := range opts.Exclude {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return false
		}
	}
	
	// If no include patterns, include all
	if len(opts.Include) == 0 {
		return true
	}
	
	// Check include patterns
	for _, pattern := range opts.Include {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
	}
	
	return false
}

// parseRipgrepOutput parses JSON output from ripgrep
func (s *Searcher) parseRipgrepOutput(output []byte) ([]Result, error) {
	var results []Result
	
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		
		// Parse ripgrep JSON output
		var msg map[string]interface{}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue // Skip malformed lines
		}
		
		if msg["type"] == "match" {
			data, ok := msg["data"].(map[string]interface{})
			if !ok {
				continue
			}
			
			// Extract path
			pathData, ok := data["path"].(map[string]interface{})
			if !ok {
				continue
			}
			path, _ := pathData["text"].(string)
			
			// Extract lines
			linesData, ok := data["lines"].(map[string]interface{})
			if !ok {
				continue
			}
			lineText, _ := linesData["text"].(string)
			
			// Extract line number
			lineNumber, _ := data["line_number"].(float64)
			
			// Extract submatches for column info
			var column int
			var match string
			if submatches, ok := data["submatches"].([]interface{}); ok && len(submatches) > 0 {
				if submatch, ok := submatches[0].(map[string]interface{}); ok {
					if matchData, ok := submatch["match"].(map[string]interface{}); ok {
						match, _ = matchData["text"].(string)
					}
					if startCol, ok := submatch["start"].(float64); ok {
						column = int(startCol) + 1
					}
				}
			}
			
			result := Result{
				Path:    path,
				Line:    int(lineNumber),
				Column:  column,
				Content: strings.TrimSpace(lineText),
				Match:   match,
				Score:   1.0, // Ripgrep doesn't provide fuzzy scores
			}
			results = append(results, result)
		}
	}
	
	return results, nil
}

// rankResults sorts results by fuzzy match score
func (s *Searcher) rankResults(results *[]Result, query string) {
	// Calculate scores for each result
	for i := range *results {
		(*results)[i].Score = float64(fuzzy.RankMatchFold(query, (*results)[i].Content))
	}
	
	// Sort by score (descending)
	// Higher scores are better
	for i := 0; i < len(*results); i++ {
		for j := i + 1; j < len(*results); j++ {
			if (*results)[j].Score > (*results)[i].Score {
				(*results)[i], (*results)[j] = (*results)[j], (*results)[i]
			}
		}
	}
}

// resultCache caches search results
type resultCache struct {
	mu       sync.RWMutex
	cache    map[string][]Result
	maxSize  int
	keys     []string
}

func newResultCache(maxSize int) *resultCache {
	return &resultCache{
		cache:   make(map[string][]Result),
		maxSize: maxSize,
		keys:    make([]string, 0, maxSize),
	}
}

func (c *resultCache) generateKey(opts Options) string {
	// Generate a unique key for the search options
	return fmt.Sprintf("%s:%s:%v:%v:%v:%v:%v",
		opts.Query, opts.Path, opts.CaseSensitive, opts.WholeWord,
		opts.Regex, opts.UseFuzzy, opts.FuzzyThreshold)
}

func (c *resultCache) get(key string) ([]Result, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	results, found := c.cache[key]
	return results, found
}

func (c *resultCache) set(key string, results []Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Evict old entries if cache is full
	if len(c.cache) >= c.maxSize && c.maxSize > 0 {
		oldKey := c.keys[0]
		delete(c.cache, oldKey)
		c.keys = c.keys[1:]
	}
	
	c.cache[key] = results
	c.keys = append(c.keys, key)
}