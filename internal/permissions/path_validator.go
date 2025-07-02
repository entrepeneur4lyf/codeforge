package permissions

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PathValidator handles path-based permission validation
type PathValidator struct {
	allowedPaths    []string
	deniedPaths     []string
	restrictedPaths map[string]PermissionType // path -> required permission
	sandboxRoot     string
}

// NewPathValidator creates a new path validator
func NewPathValidator(sandboxRoot string) *PathValidator {
	return &PathValidator{
		allowedPaths:    []string{},
		deniedPaths:     []string{},
		restrictedPaths: make(map[string]PermissionType),
		sandboxRoot:     sandboxRoot,
	}
}

// PathValidationResult represents the result of path validation
type PathValidationResult struct {
	Allowed         bool           `json:"allowed"`
	Path            string         `json:"path"`
	NormalizedPath  string         `json:"normalized_path"`
	RequiredPermission PermissionType `json:"required_permission,omitempty"`
	Reason          string         `json:"reason"`
	RiskLevel       int            `json:"risk_level"`
	IsSystemPath    bool           `json:"is_system_path"`
	IsHidden        bool           `json:"is_hidden"`
	IsExecutable    bool           `json:"is_executable"`
	Warnings        []string       `json:"warnings,omitempty"`
}

// ValidatePath validates a file path for access
func (pv *PathValidator) ValidatePath(path string, permType PermissionType) (*PathValidationResult, error) {
	// Normalize the path
	normalizedPath, err := pv.normalizePath(path)
	if err != nil {
		return &PathValidationResult{
			Allowed: false,
			Path:    path,
			Reason:  fmt.Sprintf("Invalid path: %v", err),
		}, nil
	}

	result := &PathValidationResult{
		Path:           path,
		NormalizedPath: normalizedPath,
		RequiredPermission: permType,
	}

	// Check for path traversal attacks
	if pv.hasPathTraversal(path) {
		result.Allowed = false
		result.Reason = "Path traversal detected"
		result.RiskLevel = 10
		return result, nil
	}

	// Check if path is within sandbox (if configured)
	if pv.sandboxRoot != "" && !pv.isWithinSandbox(normalizedPath) {
		result.Allowed = false
		result.Reason = "Path outside sandbox"
		result.RiskLevel = 9
		return result, nil
	}

	// Check explicitly denied paths
	if pv.isDeniedPath(normalizedPath) {
		result.Allowed = false
		result.Reason = "Path explicitly denied"
		result.RiskLevel = 8
		return result, nil
	}

	// Check system paths
	result.IsSystemPath = pv.isSystemPath(normalizedPath)
	if result.IsSystemPath && !pv.isSystemAccessAllowed(normalizedPath, permType) {
		result.Allowed = false
		result.Reason = "System path access denied"
		result.RiskLevel = 9
		return result, nil
	}

	// Check hidden files/directories
	result.IsHidden = pv.isHiddenPath(normalizedPath)
	if result.IsHidden && !pv.isHiddenAccessAllowed(normalizedPath, permType) {
		result.Allowed = false
		result.Reason = "Hidden file access denied"
		result.RiskLevel = 6
		return result, nil
	}

	// Check executable files
	result.IsExecutable = pv.isExecutablePath(normalizedPath)
	if result.IsExecutable && permType == PermissionFileWrite {
		result.Warnings = append(result.Warnings, "Writing to executable file")
		result.RiskLevel = 7
	}

	// Check restricted paths
	if requiredPerm, isRestricted := pv.isRestrictedPath(normalizedPath); isRestricted {
		if requiredPerm != permType && !pv.isPermissionSufficient(permType, requiredPerm) {
			result.Allowed = false
			result.Reason = fmt.Sprintf("Insufficient permission for restricted path (requires %s)", requiredPerm)
			result.RiskLevel = 7
			return result, nil
		}
	}

	// Check explicitly allowed paths
	if pv.isAllowedPath(normalizedPath) {
		result.Allowed = true
		result.Reason = "Path explicitly allowed"
		result.RiskLevel = pv.calculateRiskLevel(normalizedPath, permType)
		return result, nil
	}

	// Default validation based on permission type and path characteristics
	result.Allowed = pv.isDefaultAllowed(normalizedPath, permType)
	if result.Allowed {
		result.Reason = "Path allowed by default rules"
	} else {
		result.Reason = "Path denied by default rules"
	}
	result.RiskLevel = pv.calculateRiskLevel(normalizedPath, permType)

	return result, nil
}

// normalizePath normalizes and cleans a file path
func (pv *PathValidator) normalizePath(path string) (string, error) {
	// Clean the path
	cleaned := filepath.Clean(path)
	
	// Convert to absolute path if relative
	if !filepath.IsAbs(cleaned) {
		abs, err := filepath.Abs(cleaned)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		cleaned = abs
	}

	// Resolve symlinks
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		// If symlink resolution fails, use the cleaned path
		// This handles cases where the target doesn't exist yet
		resolved = cleaned
	}

	return resolved, nil
}

// hasPathTraversal checks for path traversal attempts
func (pv *PathValidator) hasPathTraversal(path string) bool {
	// Check for obvious path traversal patterns
	dangerous := []string{
		"..",
		"../",
		"..\\",
		"%2e%2e",
		"%2e%2e%2f",
		"%2e%2e%5c",
		"..%2f",
		"..%5c",
	}

	pathLower := strings.ToLower(path)
	for _, pattern := range dangerous {
		if strings.Contains(pathLower, pattern) {
			return true
		}
	}

	// Check for encoded path traversal
	if strings.Contains(path, "%") {
		// Simple URL decode check
		decoded := strings.ReplaceAll(path, "%2e", ".")
		decoded = strings.ReplaceAll(decoded, "%2f", "/")
		decoded = strings.ReplaceAll(decoded, "%5c", "\\")
		if strings.Contains(decoded, "..") {
			return true
		}
	}

	return false
}

// isWithinSandbox checks if path is within the sandbox root
func (pv *PathValidator) isWithinSandbox(path string) bool {
	if pv.sandboxRoot == "" {
		return true // No sandbox configured
	}

	sandboxAbs, err := filepath.Abs(pv.sandboxRoot)
	if err != nil {
		return false
	}

	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Check if path is within or equal to sandbox root
	rel, err := filepath.Rel(sandboxAbs, pathAbs)
	if err != nil {
		return false
	}

	// If relative path starts with "..", it's outside the sandbox
	return !strings.HasPrefix(rel, "..")
}

// isSystemPath checks if path is a system path
func (pv *PathValidator) isSystemPath(path string) bool {
	systemPaths := []string{
		"/etc",
		"/sys",
		"/proc",
		"/dev",
		"/boot",
		"/root",
		"/var/log",
		"/usr/bin",
		"/usr/sbin",
		"/sbin",
		"/bin",
		"C:\\Windows",
		"C:\\System32",
		"C:\\Program Files",
	}

	pathLower := strings.ToLower(path)
	for _, sysPath := range systemPaths {
		if strings.HasPrefix(pathLower, strings.ToLower(sysPath)) {
			return true
		}
	}

	return false
}

// isHiddenPath checks if path refers to a hidden file or directory
func (pv *PathValidator) isHiddenPath(path string) bool {
	base := filepath.Base(path)
	
	// Unix-style hidden files (start with .)
	if strings.HasPrefix(base, ".") && base != "." && base != ".." {
		return true
	}

	// Check for common hidden directories
	hiddenDirs := []string{".git", ".svn", ".hg", ".bzr", "node_modules", ".vscode", ".idea"}
	for _, hidden := range hiddenDirs {
		if strings.Contains(path, hidden) {
			return true
		}
	}

	return false
}

// isExecutablePath checks if path refers to an executable file
func (pv *PathValidator) isExecutablePath(path string) bool {
	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	executableExts := []string{".exe", ".bat", ".cmd", ".sh", ".bash", ".zsh", ".fish", ".ps1", ".py", ".pl", ".rb"}
	
	for _, execExt := range executableExts {
		if ext == execExt {
			return true
		}
	}

	// Check if file exists and has executable permissions
	if info, err := os.Stat(path); err == nil {
		mode := info.Mode()
		return mode&0111 != 0 // Check if any execute bit is set
	}

	return false
}

// isDeniedPath checks if path is explicitly denied
func (pv *PathValidator) isDeniedPath(path string) bool {
	for _, denied := range pv.deniedPaths {
		if ResourceMatches(denied, path) {
			return true
		}
	}
	return false
}

// isAllowedPath checks if path is explicitly allowed
func (pv *PathValidator) isAllowedPath(path string) bool {
	for _, allowed := range pv.allowedPaths {
		if ResourceMatches(allowed, path) {
			return true
		}
	}
	return false
}

// isRestrictedPath checks if path requires special permissions
func (pv *PathValidator) isRestrictedPath(path string) (PermissionType, bool) {
	for restrictedPath, requiredPerm := range pv.restrictedPaths {
		if ResourceMatches(restrictedPath, path) {
			return requiredPerm, true
		}
	}
	return "", false
}

// isSystemAccessAllowed checks if system path access is allowed
func (pv *PathValidator) isSystemAccessAllowed(path string, permType PermissionType) bool {
	// Only allow read access to some system paths
	if permType == PermissionFileRead {
		readOnlySystemPaths := []string{
			"/proc",
			"/sys",
			"/etc/passwd",
			"/etc/group",
		}
		
		pathLower := strings.ToLower(path)
		for _, readOnlyPath := range readOnlySystemPaths {
			if strings.HasPrefix(pathLower, strings.ToLower(readOnlyPath)) {
				return true
			}
		}
	}

	return false
}

// isHiddenAccessAllowed checks if hidden file access is allowed
func (pv *PathValidator) isHiddenAccessAllowed(path string, permType PermissionType) bool {
	// Allow access to common development hidden files
	allowedHidden := []string{
		".gitignore",
		".env",
		".config",
		".bashrc",
		".profile",
		".vimrc",
		".editorconfig",
	}

	base := filepath.Base(path)
	for _, allowed := range allowedHidden {
		if base == allowed {
			return true
		}
	}

	// Allow read access to .git directory for git operations
	if permType == PermissionFileRead && strings.Contains(path, ".git") {
		return true
	}

	return false
}

// isPermissionSufficient checks if one permission is sufficient for another
func (pv *PathValidator) isPermissionSufficient(have, need PermissionType) bool {
	// Write permission includes read permission
	if have == PermissionFileWrite && need == PermissionFileRead {
		return true
	}
	
	// Delete permission includes read permission
	if have == PermissionFileDelete && need == PermissionFileRead {
		return true
	}

	return have == need
}

// isDefaultAllowed determines if path is allowed by default rules
func (pv *PathValidator) isDefaultAllowed(path string, _ PermissionType) bool {
	// Default deny for system paths
	if pv.isSystemPath(path) {
		return false
	}

	// Default allow for user directories
	userDirs := []string{
		os.Getenv("HOME"),
		"/home",
		"/Users",
		"C:\\Users",
	}

	for _, userDir := range userDirs {
		if userDir != "" && strings.HasPrefix(path, userDir) {
			return true
		}
	}

	// Default allow for current working directory and subdirectories
	if cwd, err := os.Getwd(); err == nil {
		if strings.HasPrefix(path, cwd) {
			return true
		}
	}

	// Default deny for everything else
	return false
}

// calculateRiskLevel calculates risk level for a path operation
func (pv *PathValidator) calculateRiskLevel(path string, permType PermissionType) int {
	risk := GetRiskLevel(permType)

	// Increase risk for system paths
	if pv.isSystemPath(path) {
		risk += 3
	}

	// Increase risk for executable files
	if pv.isExecutablePath(path) {
		risk += 2
	}

	// Increase risk for hidden files
	if pv.isHiddenPath(path) {
		risk += 1
	}

	// Decrease risk for user directories
	if userHome := os.Getenv("HOME"); userHome != "" && strings.HasPrefix(path, userHome) {
		risk -= 1
	}

	// Clamp to 1-10 range
	if risk > 10 {
		risk = 10
	}
	if risk < 1 {
		risk = 1
	}

	return risk
}

// Configuration methods

// AddAllowedPath adds a path to the allowed list
func (pv *PathValidator) AddAllowedPath(path string) {
	pv.allowedPaths = append(pv.allowedPaths, path)
}

// AddDeniedPath adds a path to the denied list
func (pv *PathValidator) AddDeniedPath(path string) {
	pv.deniedPaths = append(pv.deniedPaths, path)
}

// AddRestrictedPath adds a path that requires specific permissions
func (pv *PathValidator) AddRestrictedPath(path string, requiredPermission PermissionType) {
	pv.restrictedPaths[path] = requiredPermission
}

// SetSandboxRoot sets the sandbox root directory
func (pv *PathValidator) SetSandboxRoot(root string) {
	pv.sandboxRoot = root
}

// GetPathInfo returns detailed information about a path
func (pv *PathValidator) GetPathInfo(path string) map[string]interface{} {
	normalized, _ := pv.normalizePath(path)
	
	info := map[string]interface{}{
		"original_path":    path,
		"normalized_path":  normalized,
		"is_system_path":   pv.isSystemPath(normalized),
		"is_hidden":        pv.isHiddenPath(normalized),
		"is_executable":    pv.isExecutablePath(normalized),
		"has_traversal":    pv.hasPathTraversal(path),
		"within_sandbox":   pv.isWithinSandbox(normalized),
		"explicitly_allowed": pv.isAllowedPath(normalized),
		"explicitly_denied":  pv.isDeniedPath(normalized),
	}

	// Add file system information if path exists
	if stat, err := os.Stat(normalized); err == nil {
		info["exists"] = true
		info["is_dir"] = stat.IsDir()
		info["size"] = stat.Size()
		info["mode"] = stat.Mode().String()
		info["mod_time"] = stat.ModTime()
	} else {
		info["exists"] = false
	}

	return info
}

// ValidatePathPattern validates a path pattern (with wildcards)
func (pv *PathValidator) ValidatePathPattern(pattern string) error {
	// Check for valid wildcard usage
	if strings.Count(pattern, "*") > 3 {
		return fmt.Errorf("too many wildcards in pattern")
	}

	// Check for dangerous patterns
	if strings.Contains(pattern, "**") {
		return fmt.Errorf("recursive wildcards not allowed")
	}

	// Validate regex if pattern contains special characters
	if strings.ContainsAny(pattern, "[]{}()^$+?|\\") {
		_, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
	}

	return nil
}
