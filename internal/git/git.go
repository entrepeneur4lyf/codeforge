package git

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitStatus represents the status of a git repository
type GitStatus struct {
	Branch       string            `json:"branch"`
	Status       string            `json:"status"`
	Modified     []string          `json:"modified"`
	Untracked    []string          `json:"untracked"`
	Staged       []string          `json:"staged"`
	Deleted      []string          `json:"deleted"`
	Renamed      []string          `json:"renamed"`
	Copied       []string          `json:"copied"`
	Ahead        int               `json:"ahead"`
	Behind       int               `json:"behind"`
	LastCommit   *GitCommit        `json:"last_commit,omitempty"`
	FileStatuses map[string]string `json:"file_statuses"`
}

// GitCommit represents a git commit
type GitCommit struct {
	Hash      string    `json:"hash"`
	ShortHash string    `json:"short_hash"`
	Subject   string    `json:"subject"`
	Author    string    `json:"author"`
	Date      time.Time `json:"date"`
	Body      string    `json:"body,omitempty"`
}

// GitDiff represents a git diff
type GitDiff struct {
	FilePath    string `json:"file_path"`
	OldPath     string `json:"old_path,omitempty"`
	Status      string `json:"status"` // A, M, D, R, C
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
	Content     string `json:"content"`
	IsBinary    bool   `json:"is_binary"`
	IsRenamed   bool   `json:"is_renamed"`
	IsNewFile   bool   `json:"is_new_file"`
	IsDeleted   bool   `json:"is_deleted"`
	Similarity  int    `json:"similarity,omitempty"` // For renames/copies
}

// Repository represents a git repository
type Repository struct {
	workingDir string
}

// NewRepository creates a new git repository instance
func NewRepository(workingDir string) *Repository {
	return &Repository{
		workingDir: workingDir,
	}
}

// IsGitRepository checks if the directory is a git repository
func (r *Repository) IsGitRepository() bool {
	gitDir := filepath.Join(r.workingDir, ".git")
	if stat, err := os.Stat(gitDir); err == nil {
		return stat.IsDir()
	}
	
	// Check if it's a git file (for worktrees)
	if _, err := os.Stat(gitDir); err == nil {
		return true
	}
	
	return false
}

// IsGitInstalled checks if git is installed and available
func IsGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// GetStatus gets the current git status
func (r *Repository) GetStatus(ctx context.Context) (*GitStatus, error) {
	if !r.IsGitRepository() {
		return nil, fmt.Errorf("not a git repository")
	}

	status := &GitStatus{
		FileStatuses: make(map[string]string),
	}

	// Get current branch
	branch, err := r.getCurrentBranch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current branch: %w", err)
	}
	status.Branch = branch

	// Get porcelain status
	if err := r.getPortcelainStatus(ctx, status); err != nil {
		return nil, fmt.Errorf("failed to get porcelain status: %w", err)
	}

	// Get ahead/behind information
	if err := r.getAheadBehind(ctx, status); err != nil {
		// Non-fatal error, continue without ahead/behind info
	}

	// Get last commit
	lastCommit, err := r.getLastCommit(ctx)
	if err == nil {
		status.LastCommit = lastCommit
	}

	// Determine overall status
	status.Status = r.determineOverallStatus(status)

	return status, nil
}

// getCurrentBranch gets the current branch name
func (r *Repository) getCurrentBranch(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "--show-current")
	cmd.Dir = r.workingDir
	
	output, err := cmd.Output()
	if err != nil {
		// Fallback to symbolic-ref for older git versions
		cmd = exec.CommandContext(ctx, "git", "symbolic-ref", "--short", "HEAD")
		cmd.Dir = r.workingDir
		output, err = cmd.Output()
		if err != nil {
			return "HEAD", nil // Detached HEAD state
		}
	}
	
	return strings.TrimSpace(string(output)), nil
}

// getPortcelainStatus parses git status --porcelain output
func (r *Repository) getPortcelainStatus(ctx context.Context, status *GitStatus) error {
	cmd := exec.CommandContext(ctx, "git", "status", "--porcelain")
	cmd.Dir = r.workingDir
	
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 3 {
			continue
		}

		statusCode := line[:2]
		filePath := line[3:]

		// Handle renames (format: "R  old -> new")
		if strings.HasPrefix(statusCode, "R") {
			parts := strings.Split(filePath, " -> ")
			if len(parts) == 2 {
				status.Renamed = append(status.Renamed, fmt.Sprintf("%s -> %s", parts[0], parts[1]))
				status.FileStatuses[parts[1]] = "R"
				continue
			}
		}

		// Handle copies (format: "C  old -> new")
		if strings.HasPrefix(statusCode, "C") {
			parts := strings.Split(filePath, " -> ")
			if len(parts) == 2 {
				status.Copied = append(status.Copied, fmt.Sprintf("%s -> %s", parts[0], parts[1]))
				status.FileStatuses[parts[1]] = "C"
				continue
			}
		}

		// Parse status codes
		indexStatus := statusCode[0]
		workTreeStatus := statusCode[1]

		status.FileStatuses[filePath] = statusCode

		// Categorize files based on status
		switch {
		case indexStatus == 'A' || workTreeStatus == 'A':
			status.Staged = append(status.Staged, filePath)
		case indexStatus == 'M' || workTreeStatus == 'M':
			if indexStatus == 'M' {
				status.Staged = append(status.Staged, filePath)
			} else {
				status.Modified = append(status.Modified, filePath)
			}
		case indexStatus == 'D' || workTreeStatus == 'D':
			status.Deleted = append(status.Deleted, filePath)
		case statusCode == "??":
			status.Untracked = append(status.Untracked, filePath)
		}
	}

	return scanner.Err()
}

// getAheadBehind gets ahead/behind commit count
func (r *Repository) getAheadBehind(ctx context.Context, status *GitStatus) error {
	cmd := exec.CommandContext(ctx, "git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	cmd.Dir = r.workingDir
	
	output, err := cmd.Output()
	if err != nil {
		return err // No upstream or other error
	}

	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) == 2 {
		if ahead, err := parseIntSafe(parts[0]); err == nil {
			status.Ahead = ahead
		}
		if behind, err := parseIntSafe(parts[1]); err == nil {
			status.Behind = behind
		}
	}

	return nil
}

// getLastCommit gets the last commit information
func (r *Repository) getLastCommit(ctx context.Context) (*GitCommit, error) {
	cmd := exec.CommandContext(ctx, "git", "log", "-1", "--format=%H%n%h%n%s%n%an%n%ad%n%b", "--date=iso")
	cmd.Dir = r.workingDir
	
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 5 {
		return nil, fmt.Errorf("invalid git log output")
	}

	date, err := time.Parse("2006-01-02 15:04:05 -0700", lines[4])
	if err != nil {
		date = time.Now() // Fallback
	}

	commit := &GitCommit{
		Hash:      lines[0],
		ShortHash: lines[1],
		Subject:   lines[2],
		Author:    lines[3],
		Date:      date,
	}

	// Add body if present
	if len(lines) > 5 {
		commit.Body = strings.Join(lines[5:], "\n")
	}

	return commit, nil
}

// determineOverallStatus determines the overall repository status
func (r *Repository) determineOverallStatus(status *GitStatus) string {
	if len(status.Staged) > 0 {
		return "staged"
	}
	if len(status.Modified) > 0 || len(status.Deleted) > 0 {
		return "modified"
	}
	if len(status.Untracked) > 0 {
		return "untracked"
	}
	if status.Ahead > 0 {
		return "ahead"
	}
	if status.Behind > 0 {
		return "behind"
	}
	return "clean"
}

// GetDiff gets the diff for the repository
func (r *Repository) GetDiff(ctx context.Context, staged bool) ([]GitDiff, error) {
	if !r.IsGitRepository() {
		return nil, fmt.Errorf("not a git repository")
	}

	var cmd *exec.Cmd
	if staged {
		cmd = exec.CommandContext(ctx, "git", "diff", "--cached", "--name-status")
	} else {
		cmd = exec.CommandContext(ctx, "git", "diff", "--name-status")
	}
	cmd.Dir = r.workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var diffs []GitDiff
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 2 {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		status := parts[0]
		filePath := parts[1]

		diff := GitDiff{
			FilePath:  filePath,
			Status:    status,
			IsNewFile: status == "A",
			IsDeleted: status == "D",
			IsRenamed: strings.HasPrefix(status, "R"),
		}

		// Get detailed diff content
		content, err := r.getFileDiff(ctx, filePath, staged)
		if err == nil {
			diff.Content = content
		}

		diffs = append(diffs, diff)
	}

	return diffs, scanner.Err()
}

// getFileDiff gets the diff content for a specific file
func (r *Repository) getFileDiff(ctx context.Context, filePath string, staged bool) (string, error) {
	var cmd *exec.Cmd
	if staged {
		cmd = exec.CommandContext(ctx, "git", "diff", "--cached", filePath)
	} else {
		cmd = exec.CommandContext(ctx, "git", "diff", filePath)
	}
	cmd.Dir = r.workingDir

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

// GetCommitHistory gets the commit history
func (r *Repository) GetCommitHistory(ctx context.Context, limit int) ([]GitCommit, error) {
	if !r.IsGitRepository() {
		return nil, fmt.Errorf("not a git repository")
	}

	if limit <= 0 {
		limit = 10
	}

	cmd := exec.CommandContext(ctx, "git", "log", fmt.Sprintf("-%d", limit), "--format=%H%n%h%n%s%n%an%n%ad", "--date=iso")
	cmd.Dir = r.workingDir

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var commits []GitCommit
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for i := 0; i < len(lines); i += 5 {
		if i+4 >= len(lines) {
			break
		}

		date, err := time.Parse("2006-01-02 15:04:05 -0700", lines[i+4])
		if err != nil {
			date = time.Now() // Fallback
		}

		commit := GitCommit{
			Hash:      lines[i],
			ShortHash: lines[i+1],
			Subject:   lines[i+2],
			Author:    lines[i+3],
			Date:      date,
		}

		commits = append(commits, commit)
	}

	return commits, nil
}

// parseIntSafe safely parses an integer, returning 0 on error
func parseIntSafe(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	
	var result int
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
		result = result*10 + int(r-'0')
	}
	
	return result, nil
}
