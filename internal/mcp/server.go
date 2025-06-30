package mcp

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/permissions"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CodeForgeServer represents the MCP server for CodeForge
type CodeForgeServer struct {
	server        *server.MCPServer
	vectorDB      *vectordb.VectorDB
	config        *config.Config
	workspaceRoot string
}

// NewCodeForgeServer creates a new CodeForge MCP server
func NewCodeForgeServer(cfg *config.Config, vdb *vectordb.VectorDB, workspaceRoot string) *CodeForgeServer {
	// Create MCP server with capabilities
	s := server.NewMCPServer(
		"CodeForge",
		"0.1.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true), // subscribe, listChanged
		server.WithPromptCapabilities(true),
		server.WithRecovery(),
	)

	cfs := &CodeForgeServer{
		server:        s,
		vectorDB:      vdb,
		config:        cfg,
		workspaceRoot: workspaceRoot,
	}

	// Register tools, resources, and prompts
	cfs.registerTools()
	cfs.registerResources()
	cfs.registerPrompts()

	return cfs
}

// NewPermissionAwareCodeForgeServer creates a new CodeForge MCP server with permission checking
func NewPermissionAwareCodeForgeServer(cfg *config.Config, vdb *vectordb.VectorDB, workspaceRoot string, permService *permissions.PermissionService) *PermissionAwareMCPServer {
	// Create the base server
	baseServer := NewCodeForgeServer(cfg, vdb, workspaceRoot)

	// Wrap with permission checking
	permAwareServer := NewPermissionAwareMCPServer(baseServer, permService)

	// Register permission-aware tools (this will replace the base tools)
	permAwareServer.RegisterPermissionAwareTools()

	log.Printf("Created permission-aware CodeForge MCP server")
	return permAwareServer
}

// registerTools registers all CodeForge tools
func (cfs *CodeForgeServer) registerTools() {
	// Semantic search tool
	semanticSearchTool := mcp.NewTool("semantic_search",
		mcp.WithDescription("Search for code using semantic similarity"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query to find similar code"),
		),
		mcp.WithNumber("max_results",
			mcp.Description("Maximum number of results to return (default: 10)"),
		),
		mcp.WithString("language",
			mcp.Description("Filter by programming language"),
		),
		mcp.WithString("chunk_type",
			mcp.Description("Filter by chunk type (function, class, module, etc.)"),
		),
	)

	cfs.server.AddTool(semanticSearchTool, cfs.handleSemanticSearch)

	// File reading tool
	fileReadTool := mcp.NewTool("read_file",
		mcp.WithDescription("Read the contents of a file"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the file to read (relative to workspace root)"),
		),
	)

	cfs.server.AddTool(fileReadTool, cfs.handleReadFile)

	// File writing tool
	fileWriteTool := mcp.NewTool("write_file",
		mcp.WithDescription("Write content to a file"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the file to write (relative to workspace root)"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Content to write to the file"),
		),
	)

	cfs.server.AddTool(fileWriteTool, cfs.handleWriteFile)

	// Code analysis tool
	codeAnalysisTool := mcp.NewTool("analyze_code",
		mcp.WithDescription("Analyze code structure and extract symbols"),
		mcp.WithString("path",
			mcp.Required(),
			mcp.Description("Path to the file to analyze"),
		),
	)

	cfs.server.AddTool(codeAnalysisTool, cfs.handleCodeAnalysis)

	// Project structure tool
	projectStructureTool := mcp.NewTool("get_project_structure",
		mcp.WithDescription("Get the project directory structure"),
		mcp.WithString("path",
			mcp.Description("Path to analyze (default: workspace root)"),
		),
		mcp.WithNumber("max_depth",
			mcp.Description("Maximum depth to traverse (default: 3)"),
		),
	)

	cfs.server.AddTool(projectStructureTool, cfs.handleProjectStructure)

	// Symbol search tool
	symbolSearchTool := mcp.NewTool("symbol_search",
		mcp.WithDescription("Search for symbols across the workspace"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Symbol name or pattern to search for"),
		),
		mcp.WithString("kind",
			mcp.Description("Symbol kind filter (function, class, variable, etc.)"),
		),
	)

	cfs.server.AddTool(symbolSearchTool, cfs.handleSymbolSearch)

	// Get definition tool
	getDefinitionTool := mcp.NewTool("get_definition",
		mcp.WithDescription("Get the definition of a symbol at a specific location"),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the file"),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("Line number (1-based)"),
		),
		mcp.WithNumber("character",
			mcp.Required(),
			mcp.Description("Character position (1-based)"),
		),
	)

	cfs.server.AddTool(getDefinitionTool, cfs.handleGetDefinition)

	// Get references tool
	getReferencesTool := mcp.NewTool("get_references",
		mcp.WithDescription("Get all references to a symbol at a specific location"),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("Path to the file"),
		),
		mcp.WithNumber("line",
			mcp.Required(),
			mcp.Description("Line number (1-based)"),
		),
		mcp.WithNumber("character",
			mcp.Required(),
			mcp.Description("Character position (1-based)"),
		),
		mcp.WithBoolean("include_declaration",
			mcp.Description("Include the declaration in results (default: true)"),
		),
	)

	cfs.server.AddTool(getReferencesTool, cfs.handleGetReferences)

	// Git diff tool
	gitDiffTool := mcp.NewTool("git_diff",
		mcp.WithDescription("Get git diff for the repository"),
		mcp.WithBoolean("staged",
			mcp.Description("Get staged changes instead of working directory changes"),
		),
	)

	cfs.server.AddTool(gitDiffTool, cfs.handleGitDiff)

	// AI-powered git commit tool
	gitCommitAITool := mcp.NewTool("git_commit_ai",
		mcp.WithDescription("Generate an AI-powered commit message and commit changes"),
		mcp.WithBoolean("staged",
			mcp.Description("Commit only staged changes (default: false - commits all changes)"),
		),
	)

	cfs.server.AddTool(gitCommitAITool, cfs.handleGitCommitAI)

	// AI commit message generation tool
	gitGenerateCommitMessageTool := mcp.NewTool("git_generate_commit_message",
		mcp.WithDescription("Generate an AI-powered commit message without committing"),
		mcp.WithBoolean("staged",
			mcp.Description("Generate message for staged changes (default: false - analyzes all changes)"),
		),
	)

	cfs.server.AddTool(gitGenerateCommitMessageTool, cfs.handleGitGenerateCommitMessage)

	// Git conflict detection tool
	gitDetectConflictsTool := mcp.NewTool("git_detect_conflicts",
		mcp.WithDescription("Detect merge conflicts in the repository"),
	)

	cfs.server.AddTool(gitDetectConflictsTool, cfs.handleGitDetectConflicts)

	// Git conflict resolution tool
	gitResolveConflictsTool := mcp.NewTool("git_resolve_conflicts",
		mcp.WithDescription("Get AI-powered suggestions for resolving merge conflicts"),
		mcp.WithBoolean("auto_apply",
			mcp.Description("Automatically apply the suggested resolutions (default: false)"),
		),
	)

	cfs.server.AddTool(gitResolveConflictsTool, cfs.handleGitResolveConflicts)
}

// registerResources registers all CodeForge resources
func (cfs *CodeForgeServer) registerResources() {
	// Project metadata resource
	projectMetadata := mcp.NewResource(
		"codeforge://project/metadata",
		"Project Metadata",
		mcp.WithResourceDescription("Information about the current project"),
		mcp.WithMIMEType("application/json"),
	)

	cfs.server.AddResource(projectMetadata, cfs.handleProjectMetadata)

	// File content resource template
	fileTemplate := mcp.NewResourceTemplate(
		"codeforge://files/{path}",
		"File Content",
		mcp.WithTemplateDescription("Access to file contents in the workspace"),
		mcp.WithTemplateMIMEType("text/plain"),
	)

	cfs.server.AddResourceTemplate(fileTemplate, cfs.handleFileResource)

	// Git information resource
	gitInfo := mcp.NewResource(
		"codeforge://git/status",
		"Git Status",
		mcp.WithResourceDescription("Current git repository status"),
		mcp.WithMIMEType("application/json"),
	)

	cfs.server.AddResource(gitInfo, cfs.handleGitStatus)
}

// registerPrompts registers all CodeForge prompts
func (cfs *CodeForgeServer) registerPrompts() {
	// Code review prompt
	codeReviewPrompt := mcp.NewPrompt("code_review",
		mcp.WithPromptDescription("Assistance with code review"),
		mcp.WithArgument("file_path",
			mcp.ArgumentDescription("Path to the file to review"),
			mcp.RequiredArgument(),
		),
	)

	cfs.server.AddPrompt(codeReviewPrompt, cfs.handleCodeReviewPrompt)

	// Debugging help prompt
	debuggingPrompt := mcp.NewPrompt("debug_help",
		mcp.WithPromptDescription("Assistance with debugging code"),
		mcp.WithArgument("error_message",
			mcp.ArgumentDescription("The error message or description"),
		),
		mcp.WithArgument("file_path",
			mcp.ArgumentDescription("Path to the file with the issue"),
		),
	)

	cfs.server.AddPrompt(debuggingPrompt, cfs.handleDebuggingPrompt)

	// Refactoring guidance prompt
	refactoringPrompt := mcp.NewPrompt("refactoring_guide",
		mcp.WithPromptDescription("Guidance for code refactoring"),
		mcp.WithArgument("target",
			mcp.ArgumentDescription("What to refactor (function, class, module)"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("goal",
			mcp.ArgumentDescription("Refactoring goal (performance, readability, etc.)"),
		),
	)

	cfs.server.AddPrompt(refactoringPrompt, cfs.handleRefactoringPrompt)

	// Documentation generation prompt
	documentationPrompt := mcp.NewPrompt("documentation_help",
		mcp.WithPromptDescription("Assistance with generating documentation"),
		mcp.WithArgument("file_path",
			mcp.ArgumentDescription("Path to the file to document"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("type",
			mcp.ArgumentDescription("Type of documentation (api, user, technical)"),
		),
	)

	cfs.server.AddPrompt(documentationPrompt, cfs.handleDocumentationPrompt)

	// Testing assistance prompt
	testingPrompt := mcp.NewPrompt("testing_help",
		mcp.WithPromptDescription("Assistance with creating tests"),
		mcp.WithArgument("file_path",
			mcp.ArgumentDescription("Path to the file to test"),
			mcp.RequiredArgument(),
		),
		mcp.WithArgument("test_type",
			mcp.ArgumentDescription("Type of tests (unit, integration, e2e)"),
		),
	)

	cfs.server.AddPrompt(testingPrompt, cfs.handleTestingPrompt)
}

// Start starts the MCP server using stdio transport
func (cfs *CodeForgeServer) Start() error {
	log.Printf("Starting CodeForge MCP server...")
	return server.ServeStdio(cfs.server)
}

// StartSSE starts the MCP server using Server-Sent Events transport
func (cfs *CodeForgeServer) StartSSE(addr string) error {
	log.Printf("Starting CodeForge MCP server with SSE on %s...", addr)
	sseServer := server.NewSSEServer(cfs.server)
	return sseServer.Start(addr)
}

// StartStreamableHTTP starts the MCP server using Streamable HTTP transport
func (cfs *CodeForgeServer) StartStreamableHTTP(addr string) error {
	log.Printf("Starting CodeForge MCP server with Streamable HTTP on %s...", addr)
	httpServer := server.NewStreamableHTTPServer(cfs.server)
	return httpServer.Start(addr)
}

// GetServer returns the underlying MCP server
func (cfs *CodeForgeServer) GetServer() *server.MCPServer {
	return cfs.server
}

// validatePath validates and resolves a file path relative to workspace root
func (cfs *CodeForgeServer) validatePath(path string) (string, error) {
	// Clean the path
	cleanPath := filepath.Clean(path)

	// Reject absolute paths that don't start with workspace
	if filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("absolute paths are not allowed: %s", path)
	}

	// Resolve relative to workspace root
	fullPath := filepath.Join(cfs.workspaceRoot, cleanPath)

	// Ensure the path is within the workspace
	absWorkspace, err := filepath.Abs(cfs.workspaceRoot)
	if err != nil {
		return "", fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Ensure the resolved path is still within workspace (prevents path traversal)
	if !strings.HasPrefix(absPath+string(filepath.Separator), absWorkspace+string(filepath.Separator)) {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	return absPath, nil
}

// fileExists checks if a file exists
func (cfs *CodeForgeServer) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
