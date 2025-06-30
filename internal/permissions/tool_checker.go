package permissions

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// ToolPermissionChecker handles tool-specific permission checking
type ToolPermissionChecker struct {
	service *PermissionService
}

// NewToolPermissionChecker creates a new tool permission checker
func NewToolPermissionChecker(service *PermissionService) *ToolPermissionChecker {
	return &ToolPermissionChecker{
		service: service,
	}
}

// ToolPermissionRequest represents a tool-specific permission request
type ToolPermissionRequest struct {
	SessionID   string                 `json:"session_id"`
	ToolName    string                 `json:"tool_name"`
	Operation   string                 `json:"operation"`
	Parameters  map[string]interface{} `json:"parameters"`
	Context     map[string]interface{} `json:"context"`
	Reason      string                 `json:"reason"`
}

// CheckToolPermission checks if a tool operation is permitted
func (tpc *ToolPermissionChecker) CheckToolPermission(ctx context.Context, req *ToolPermissionRequest) (*PermissionCheckResult, error) {
	// Determine the specific permission type based on tool and operation
	permType, resource := tpc.getPermissionTypeAndResource(req)
	
	check := &PermissionCheck{
		SessionID: req.SessionID,
		Type:      permType,
		Resource:  resource,
		Context:   req.Context,
	}

	return tpc.service.CheckPermission(ctx, check)
}

// RequestToolPermission requests permission for a tool operation
func (tpc *ToolPermissionChecker) RequestToolPermission(ctx context.Context, req *ToolPermissionRequest) (*PermissionResponse, error) {
	permType, resource := tpc.getPermissionTypeAndResource(req)
	
	permReq := &PermissionRequest{
		SessionID: req.SessionID,
		Type:      permType,
		Resource:  resource,
		Scope:     ScopeSession, // Default to session scope for tools
		Reason:    req.Reason,
		Context:   req.Context,
	}

	return tpc.service.RequestPermission(ctx, permReq)
}

// getPermissionTypeAndResource determines the permission type and resource for a tool operation
func (tpc *ToolPermissionChecker) getPermissionTypeAndResource(req *ToolPermissionRequest) (PermissionType, string) {
	toolName := strings.ToLower(req.ToolName)
	operation := strings.ToLower(req.Operation)

	switch toolName {
	case "file_operations", "file_manager", "fs":
		return tpc.getFilePermission(operation, req.Parameters)
	
	case "code_execution", "shell", "terminal":
		return tpc.getExecutionPermission(operation, req.Parameters)
	
	case "git", "version_control":
		return tpc.getGitPermission(operation, req.Parameters)
	
	case "network", "http", "api":
		return tpc.getNetworkPermission(operation, req.Parameters)
	
	case "database", "db":
		return tpc.getDatabasePermission(operation, req.Parameters)
	
	case "system", "os":
		return tpc.getSystemPermission(operation, req.Parameters)
	
	default:
		// Generic tool permission
		return PermissionToolUse, fmt.Sprintf("%s:%s", toolName, operation)
	}
}

// getFilePermission determines file operation permissions
func (tpc *ToolPermissionChecker) getFilePermission(operation string, params map[string]interface{}) (PermissionType, string) {
	// Extract file path from parameters
	var filePath string
	if path, ok := params["path"].(string); ok {
		filePath = path
	} else if file, ok := params["file"].(string); ok {
		filePath = file
	} else if filename, ok := params["filename"].(string); ok {
		filePath = filename
	} else {
		filePath = "*" // Wildcard if no specific path
	}

	// Clean and normalize the path
	if filePath != "*" {
		filePath = filepath.Clean(filePath)
	}

	switch operation {
	case "read", "get", "cat", "view", "open":
		return PermissionFileRead, filePath
	
	case "write", "save", "put", "update", "edit":
		return PermissionFileWrite, filePath
	
	case "create", "new", "touch":
		return PermissionFileCreate, filePath
	
	case "delete", "remove", "rm", "unlink":
		return PermissionFileDelete, filePath
	
	case "list", "ls", "dir", "readdir":
		// For directory listing
		dirPath := filePath
		if dirPath == "*" {
			dirPath = "."
		}
		return PermissionDirRead, dirPath
	
	case "mkdir", "create_dir":
		return PermissionDirCreate, filePath
	
	case "rmdir", "remove_dir":
		return PermissionDirDelete, filePath
	
	default:
		// Default to read permission for unknown operations
		return PermissionFileRead, filePath
	}
}

// getExecutionPermission determines code execution permissions
func (tpc *ToolPermissionChecker) getExecutionPermission(operation string, params map[string]interface{}) (PermissionType, string) {
	var command string
	if cmd, ok := params["command"].(string); ok {
		command = cmd
	} else if code, ok := params["code"].(string); ok {
		command = code
	} else {
		command = "unknown"
	}

	switch operation {
	case "execute", "run", "exec":
		return PermissionCodeExecute, command
	
	case "shell", "bash", "sh":
		return PermissionShellAccess, command
	
	case "process", "spawn":
		return PermissionProcessRun, command
	
	default:
		return PermissionCodeExecute, command
	}
}

// getGitPermission determines git operation permissions
func (tpc *ToolPermissionChecker) getGitPermission(operation string, params map[string]interface{}) (PermissionType, string) {
	var repo string
	if repository, ok := params["repository"].(string); ok {
		repo = repository
	} else if path, ok := params["path"].(string); ok {
		repo = path
	} else {
		repo = "." // Current directory
	}

	switch operation {
	case "status", "log", "show", "diff", "blame", "ls-files":
		return PermissionGitRead, repo
	
	case "add", "commit", "push", "pull", "merge", "rebase", "checkout", "branch", "tag":
		return PermissionGitWrite, repo
	
	default:
		return PermissionGitRead, repo
	}
}

// getNetworkPermission determines network operation permissions
func (tpc *ToolPermissionChecker) getNetworkPermission(operation string, params map[string]interface{}) (PermissionType, string) {
	var url string
	if u, ok := params["url"].(string); ok {
		url = u
	} else if host, ok := params["host"].(string); ok {
		url = host
	} else {
		url = "*"
	}

	switch operation {
	case "request", "get", "post", "put", "delete", "patch":
		return PermissionHTTPRequest, url
	
	case "connect", "socket":
		return PermissionNetworkAccess, url
	
	default:
		return PermissionNetworkAccess, url
	}
}

// getDatabasePermission determines database operation permissions
func (tpc *ToolPermissionChecker) getDatabasePermission(operation string, params map[string]interface{}) (PermissionType, string) {
	var database string
	if db, ok := params["database"].(string); ok {
		database = db
	} else if table, ok := params["table"].(string); ok {
		database = table
	} else {
		database = "*"
	}

	switch operation {
	case "select", "query", "read", "get":
		return PermissionDBRead, database
	
	case "insert", "update", "delete", "create", "drop", "alter":
		return PermissionDBWrite, database
	
	default:
		return PermissionDBRead, database
	}
}

// getSystemPermission determines system operation permissions
func (tpc *ToolPermissionChecker) getSystemPermission(operation string, params map[string]interface{}) (PermissionType, string) {
	var resource string
	if res, ok := params["resource"].(string); ok {
		resource = res
	} else {
		resource = "system"
	}

	switch operation {
	case "info", "status", "ps", "top", "df", "free":
		return PermissionSystemInfo, resource
	
	case "config", "set", "configure", "install", "uninstall":
		return PermissionSystemConfig, resource
	
	default:
		return PermissionSystemInfo, resource
	}
}

// CheckMCPToolPermission checks permission for MCP tool calls
func (tpc *ToolPermissionChecker) CheckMCPToolPermission(ctx context.Context, sessionID, toolName string, args map[string]interface{}) (*PermissionCheckResult, error) {
	check := &PermissionCheck{
		SessionID: sessionID,
		Type:      PermissionMCPCall,
		Resource:  toolName,
		Context: map[string]interface{}{
			"tool_name": toolName,
			"arguments": args,
		},
	}

	return tpc.service.CheckPermission(ctx, check)
}

// GetRequiredPermissions returns all permissions required for a complex operation
func (tpc *ToolPermissionChecker) GetRequiredPermissions(req *ToolPermissionRequest) []PermissionType {
	var permissions []PermissionType
	
	// Get primary permission
	permType, _ := tpc.getPermissionTypeAndResource(req)
	permissions = append(permissions, permType)
	
	// Add additional permissions based on operation complexity
	toolName := strings.ToLower(req.ToolName)
	operation := strings.ToLower(req.Operation)
	
	switch toolName {
	case "file_operations":
		// Some file operations might need both read and write
		if operation == "copy" || operation == "move" {
			permissions = append(permissions, PermissionFileRead, PermissionFileWrite)
		}
	
	case "git":
		// Git operations often need both read and write
		if operation == "commit" {
			permissions = append(permissions, PermissionGitRead, PermissionGitWrite)
		}
	
	case "code_execution":
		// Code execution might need file access
		if _, hasFile := req.Parameters["file"]; hasFile {
			permissions = append(permissions, PermissionFileRead)
		}
	}
	
	return tpc.deduplicatePermissions(permissions)
}

// ValidateToolParameters validates that tool parameters are safe
func (tpc *ToolPermissionChecker) ValidateToolParameters(req *ToolPermissionRequest) error {
	toolName := strings.ToLower(req.ToolName)
	
	switch toolName {
	case "file_operations":
		return tpc.validateFileParameters(req.Parameters)
	
	case "code_execution":
		return tpc.validateExecutionParameters(req.Parameters)
	
	case "network":
		return tpc.validateNetworkParameters(req.Parameters)
	
	default:
		return nil // No specific validation
	}
}

// validateFileParameters validates file operation parameters
func (tpc *ToolPermissionChecker) validateFileParameters(params map[string]interface{}) error {
	if path, ok := params["path"].(string); ok {
		// Check for dangerous paths
		if strings.Contains(path, "..") {
			return &PermissionError{
				Code:    "DANGEROUS_PATH",
				Message: "Path traversal not allowed",
			}
		}
		
		// Check for system directories
		dangerousPaths := []string{"/etc", "/sys", "/proc", "/dev", "/boot"}
		for _, dangerous := range dangerousPaths {
			if strings.HasPrefix(path, dangerous) {
				return &PermissionError{
					Code:    "SYSTEM_PATH",
					Message: "Access to system directories not allowed",
				}
			}
		}
	}
	
	return nil
}

// validateExecutionParameters validates code execution parameters
func (tpc *ToolPermissionChecker) validateExecutionParameters(params map[string]interface{}) error {
	if command, ok := params["command"].(string); ok {
		// Check for dangerous commands
		dangerousCommands := []string{"rm -rf", "format", "fdisk", "dd", "mkfs"}
		commandLower := strings.ToLower(command)
		
		for _, dangerous := range dangerousCommands {
			if strings.Contains(commandLower, dangerous) {
				return &PermissionError{
					Code:    "DANGEROUS_COMMAND",
					Message: "Dangerous command not allowed",
				}
			}
		}
	}
	
	return nil
}

// validateNetworkParameters validates network operation parameters
func (tpc *ToolPermissionChecker) validateNetworkParameters(params map[string]interface{}) error {
	if url, ok := params["url"].(string); ok {
		// Check for localhost/internal network access
		if strings.Contains(url, "localhost") || strings.Contains(url, "127.0.0.1") || strings.Contains(url, "192.168.") {
			return &PermissionError{
				Code:    "INTERNAL_NETWORK",
				Message: "Access to internal network requires explicit permission",
			}
		}
	}
	
	return nil
}

// deduplicatePermissions removes duplicate permissions
func (tpc *ToolPermissionChecker) deduplicatePermissions(permissions []PermissionType) []PermissionType {
	seen := make(map[PermissionType]bool)
	var result []PermissionType
	
	for _, perm := range permissions {
		if !seen[perm] {
			seen[perm] = true
			result = append(result, perm)
		}
	}
	
	return result
}

// GetToolRiskAssessment provides a risk assessment for a tool operation
func (tpc *ToolPermissionChecker) GetToolRiskAssessment(req *ToolPermissionRequest) map[string]interface{} {
	permType, resource := tpc.getPermissionTypeAndResource(req)
	riskLevel := GetRiskLevel(permType)
	
	assessment := map[string]interface{}{
		"tool_name":    req.ToolName,
		"operation":    req.Operation,
		"permission":   permType,
		"resource":     resource,
		"risk_level":   riskLevel,
		"risk_category": tpc.getRiskCategory(riskLevel),
		"requires_approval": riskLevel >= 7,
	}
	
	// Add specific warnings
	var warnings []string
	if riskLevel >= 8 {
		warnings = append(warnings, "High risk operation - destructive potential")
	}
	if strings.Contains(strings.ToLower(req.ToolName), "execute") {
		warnings = append(warnings, "Code execution - security risk")
	}
	if strings.Contains(resource, "*") {
		warnings = append(warnings, "Wildcard resource - broad access")
	}
	
	if len(warnings) > 0 {
		assessment["warnings"] = warnings
	}
	
	return assessment
}

// getRiskCategory returns a human-readable risk category
func (tpc *ToolPermissionChecker) getRiskCategory(riskLevel int) string {
	switch {
	case riskLevel <= 3:
		return "Low"
	case riskLevel <= 6:
		return "Medium"
	case riskLevel <= 8:
		return "High"
	default:
		return "Critical"
	}
}
