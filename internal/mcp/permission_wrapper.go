package mcp

import (
	"context"
	"fmt"
	"log"

	"github.com/entrepeneur4lyf/codeforge/internal/permissions"
	"github.com/mark3labs/mcp-go/mcp"
)

// PermissionAwareMCPServer wraps the MCP server with permission checking
type PermissionAwareMCPServer struct {
	*CodeForgeServer
	permissionService *permissions.PermissionService
	toolChecker       *permissions.ToolPermissionChecker
	pathValidator     *permissions.PathValidator
}

// NewPermissionAwareMCPServer creates a new permission-aware MCP server
func NewPermissionAwareMCPServer(cfs *CodeForgeServer, permService *permissions.PermissionService) *PermissionAwareMCPServer {
	toolChecker := permissions.NewToolPermissionChecker(permService)
	pathValidator := permissions.NewPathValidator(cfs.workspaceRoot)
	
	// Configure path validator with safe defaults
	pathValidator.AddAllowedPath(cfs.workspaceRoot + "/*")
	pathValidator.AddDeniedPath("/etc/*")
	pathValidator.AddDeniedPath("/sys/*")
	pathValidator.AddDeniedPath("/proc/*")
	pathValidator.AddRestrictedPath("*.exe", permissions.PermissionCodeExecute)
	pathValidator.AddRestrictedPath("*.sh", permissions.PermissionCodeExecute)

	return &PermissionAwareMCPServer{
		CodeForgeServer:   cfs,
		permissionService: permService,
		toolChecker:       toolChecker,
		pathValidator:     pathValidator,
	}
}

// SessionContext represents the session context for permission checking
type SessionContext struct {
	SessionID string
	UserID    string
	IPAddress string
	UserAgent string
}

// getSessionFromContext extracts session information from context
func (pms *PermissionAwareMCPServer) getSessionFromContext(ctx context.Context) *SessionContext {
	// Try to extract session information from context
	// This would be set by the calling application
	if sessionID, ok := ctx.Value("session_id").(string); ok {
		session := &SessionContext{
			SessionID: sessionID,
		}
		
		if userID, ok := ctx.Value("user_id").(string); ok {
			session.UserID = userID
		}
		
		if ipAddr, ok := ctx.Value("ip_address").(string); ok {
			session.IPAddress = ipAddr
		}
		
		if userAgent, ok := ctx.Value("user_agent").(string); ok {
			session.UserAgent = userAgent
		}
		
		return session
	}
	
	// Default session if no context available
	return &SessionContext{
		SessionID: "default",
		UserID:    "anonymous",
	}
}

// checkToolPermission checks if a tool operation is permitted
func (pms *PermissionAwareMCPServer) checkToolPermission(ctx context.Context, toolName string, args map[string]interface{}) (*permissions.PermissionCheckResult, error) {
	session := pms.getSessionFromContext(ctx)
	
	// Create tool permission request
	toolReq := &permissions.ToolPermissionRequest{
		SessionID:  session.SessionID,
		ToolName:   toolName,
		Operation:  "execute", // Default operation
		Parameters: args,
		Context: map[string]interface{}{
			"user_id":    session.UserID,
			"ip_address": session.IPAddress,
			"user_agent": session.UserAgent,
		},
		Reason: fmt.Sprintf("MCP tool execution: %s", toolName),
	}

	return pms.toolChecker.CheckToolPermission(ctx, toolReq)
}

// Permission-aware tool handlers

// HandleReadFileWithPermissions wraps the read file handler with permission checking
func (pms *PermissionAwareMCPServer) HandleReadFileWithPermissions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("path parameter is required"), nil
	}

	// Check permissions
	args := map[string]interface{}{"path": path}
	permResult, err := pms.checkToolPermission(ctx, "read_file", args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("permission check failed: %v", err)), nil
	}

	if !permResult.Allowed {
		log.Printf("File read permission denied for path: %s, reason: %s", path, permResult.Reason)
		return mcp.NewToolResultError(fmt.Sprintf("Permission denied: %s", permResult.Reason)), nil
	}

	// Additional path validation
	session := pms.getSessionFromContext(ctx)
	pathResult, err := pms.pathValidator.ValidatePath(path, permissions.PermissionFileRead)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	if !pathResult.Allowed {
		log.Printf("Path validation failed for %s: %s", path, pathResult.Reason)
		return mcp.NewToolResultError(fmt.Sprintf("Path access denied: %s", pathResult.Reason)), nil
	}

	// Log the permission usage
	log.Printf("File read permission granted for session %s, path: %s", session.SessionID, path)

	// Call the original handler
	return pms.CodeForgeServer.handleReadFile(ctx, request)
}

// HandleWriteFileWithPermissions wraps the write file handler with permission checking
func (pms *PermissionAwareMCPServer) HandleWriteFileWithPermissions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("path parameter is required"), nil
	}

	content, err := request.RequireString("content")
	if err != nil {
		return mcp.NewToolResultError("content parameter is required"), nil
	}

	// Check permissions
	args := map[string]interface{}{
		"path":    path,
		"content": content,
	}
	permResult, err := pms.checkToolPermission(ctx, "write_file", args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("permission check failed: %v", err)), nil
	}

	if !permResult.Allowed {
		log.Printf("File write permission denied for path: %s, reason: %s", path, permResult.Reason)
		return mcp.NewToolResultError(fmt.Sprintf("Permission denied: %s", permResult.Reason)), nil
	}

	// Additional path validation
	session := pms.getSessionFromContext(ctx)
	pathResult, err := pms.pathValidator.ValidatePath(path, permissions.PermissionFileWrite)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	if !pathResult.Allowed {
		log.Printf("Path validation failed for %s: %s", path, pathResult.Reason)
		return mcp.NewToolResultError(fmt.Sprintf("Path access denied: %s", pathResult.Reason)), nil
	}

	// Check for high-risk operations
	if pathResult.RiskLevel >= 8 {
		log.Printf("High-risk file write operation for session %s, path: %s, risk: %d", 
			session.SessionID, path, pathResult.RiskLevel)
	}

	// Log the permission usage
	log.Printf("File write permission granted for session %s, path: %s", session.SessionID, path)

	// Call the original handler
	return pms.CodeForgeServer.handleWriteFile(ctx, request)
}

// HandleCodeAnalysisWithPermissions wraps the code analysis handler with permission checking
func (pms *PermissionAwareMCPServer) HandleCodeAnalysisWithPermissions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	path, err := request.RequireString("path")
	if err != nil {
		return mcp.NewToolResultError("path parameter is required"), nil
	}

	// Check permissions
	args := map[string]interface{}{"path": path}
	permResult, err := pms.checkToolPermission(ctx, "analyze_code", args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("permission check failed: %v", err)), nil
	}

	if !permResult.Allowed {
		log.Printf("Code analysis permission denied for path: %s, reason: %s", path, permResult.Reason)
		return mcp.NewToolResultError(fmt.Sprintf("Permission denied: %s", permResult.Reason)), nil
	}

	// Path validation (code analysis requires read permission)
	session := pms.getSessionFromContext(ctx)
	pathResult, err := pms.pathValidator.ValidatePath(path, permissions.PermissionFileRead)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	if !pathResult.Allowed {
		log.Printf("Path validation failed for %s: %s", path, pathResult.Reason)
		return mcp.NewToolResultError(fmt.Sprintf("Path access denied: %s", pathResult.Reason)), nil
	}

	// Log the permission usage
	log.Printf("Code analysis permission granted for session %s, path: %s", session.SessionID, path)

	// Call the original handler
	return pms.CodeForgeServer.handleCodeAnalysis(ctx, request)
}

// HandleProjectStructureWithPermissions wraps the project structure handler with permission checking
func (pms *PermissionAwareMCPServer) HandleProjectStructureWithPermissions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	path := request.GetString("path", ".")
	maxDepth := int(request.GetFloat("max_depth", 3))

	// Check permissions
	args := map[string]interface{}{
		"path":      path,
		"max_depth": maxDepth,
	}
	permResult, err := pms.checkToolPermission(ctx, "get_project_structure", args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("permission check failed: %v", err)), nil
	}

	if !permResult.Allowed {
		log.Printf("Project structure permission denied for path: %s, reason: %s", path, permResult.Reason)
		return mcp.NewToolResultError(fmt.Sprintf("Permission denied: %s", permResult.Reason)), nil
	}

	// Path validation (directory listing requires read permission)
	session := pms.getSessionFromContext(ctx)
	pathResult, err := pms.pathValidator.ValidatePath(path, permissions.PermissionDirRead)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("path validation failed: %v", err)), nil
	}

	if !pathResult.Allowed {
		log.Printf("Path validation failed for %s: %s", path, pathResult.Reason)
		return mcp.NewToolResultError(fmt.Sprintf("Path access denied: %s", pathResult.Reason)), nil
	}

	// Log the permission usage
	log.Printf("Project structure permission granted for session %s, path: %s", session.SessionID, path)

	// Call the original handler
	return pms.CodeForgeServer.handleProjectStructure(ctx, request)
}

// HandleSemanticSearchWithPermissions wraps the semantic search handler with permission checking
func (pms *PermissionAwareMCPServer) HandleSemanticSearchWithPermissions(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Extract parameters
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	limit := int(request.GetFloat("limit", 10))

	// Check permissions
	args := map[string]interface{}{
		"query": query,
		"limit": limit,
	}
	permResult, err := pms.checkToolPermission(ctx, "semantic_search", args)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("permission check failed: %v", err)), nil
	}

	if !permResult.Allowed {
		log.Printf("Semantic search permission denied for query: %s, reason: %s", query, permResult.Reason)
		return mcp.NewToolResultError(fmt.Sprintf("Permission denied: %s", permResult.Reason)), nil
	}

	// Log the permission usage
	session := pms.getSessionFromContext(ctx)
	log.Printf("Semantic search permission granted for session %s, query: %s", session.SessionID, query)

	// Call the original handler
	return pms.CodeForgeServer.handleSemanticSearch(ctx, request)
}

// RegisterPermissionAwareTools registers all tools with permission checking
func (pms *PermissionAwareMCPServer) RegisterPermissionAwareTools() {
	// Replace the original tool handlers with permission-aware versions
	
	// Semantic search tool
	semanticSearchTool := mcp.NewTool("semantic_search",
		mcp.WithDescription("Search for code using semantic similarity (requires permission)"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Maximum number of results (default: 10)"),
		),
	)
	pms.server.AddTool(semanticSearchTool, pms.HandleSemanticSearchWithPermissions)

	// File read tool
	fileReadTool := mcp.NewTool("read_file",
		mcp.WithDescription("Read file contents from the workspace (requires permission)"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the file to read"),
		),
	)
	pms.server.AddTool(fileReadTool, pms.HandleReadFileWithPermissions)

	// File write tool
	fileWriteTool := mcp.NewTool("write_file",
		mcp.WithDescription("Write content to files in the workspace (requires permission)"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the file to write"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Content to write to the file"),
		),
	)
	pms.server.AddTool(fileWriteTool, pms.HandleWriteFileWithPermissions)

	// Code analysis tool
	codeAnalysisTool := mcp.NewTool("analyze_code",
		mcp.WithDescription("Analyze code structure and extract symbols (requires permission)"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the file to analyze"),
		),
	)
	pms.server.AddTool(codeAnalysisTool, pms.HandleCodeAnalysisWithPermissions)

	// Project structure tool
	projectStructureTool := mcp.NewTool("get_project_structure",
		mcp.WithDescription("Get the project directory structure (requires permission)"),
		mcp.WithString("path",
			mcp.Description("Path to analyze (default: workspace root)"),
		),
		mcp.WithNumber("max_depth",
			mcp.Description("Maximum depth to traverse (default: 3)"),
		),
	)
	pms.server.AddTool(projectStructureTool, pms.HandleProjectStructureWithPermissions)

	log.Printf("Registered permission-aware MCP tools")
}

// GetPermissionService returns the permission service
func (pms *PermissionAwareMCPServer) GetPermissionService() *permissions.PermissionService {
	return pms.permissionService
}

// GetToolChecker returns the tool permission checker
func (pms *PermissionAwareMCPServer) GetToolChecker() *permissions.ToolPermissionChecker {
	return pms.toolChecker
}

// GetPathValidator returns the path validator
func (pms *PermissionAwareMCPServer) GetPathValidator() *permissions.PathValidator {
	return pms.pathValidator
}

// UpdateSessionContext updates the session context in the request context
func UpdateSessionContext(ctx context.Context, sessionID, userID, ipAddress, userAgent string) context.Context {
	ctx = context.WithValue(ctx, "session_id", sessionID)
	ctx = context.WithValue(ctx, "user_id", userID)
	ctx = context.WithValue(ctx, "ip_address", ipAddress)
	ctx = context.WithValue(ctx, "user_agent", userAgent)
	return ctx
}
