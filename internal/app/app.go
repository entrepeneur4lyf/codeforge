package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	contextmgmt "github.com/entrepeneur4lyf/codeforge/internal/context"
	"github.com/entrepeneur4lyf/codeforge/internal/mcp"
	"github.com/entrepeneur4lyf/codeforge/internal/permissions"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

// App represents the main CodeForge application with all integrated systems
type App struct {
	Config         *config.Config
	VectorDB       *vectordb.VectorDB
	ContextManager *contextmgmt.ContextManager

	PermissionService    *permissions.PermissionService
	PermissionStorage    *permissions.PermissionStorage
	FileOperationManager *permissions.FileOperationManager
	MCPServer            *mcp.PermissionAwareMCPServer
	WorkspaceRoot        string
}

// AppConfig represents configuration for app initialization
type AppConfig struct {
	ConfigPath        string
	WorkspaceRoot     string
	DatabasePath      string
	PermissionDBPath  string
	EnablePermissions bool
	EnableContextMgmt bool
	Debug             bool
}

// NewApp creates a new CodeForge application with all systems initialized
func NewApp(ctx context.Context, appConfig *AppConfig) (*App, error) {
	log.Printf("Initializing CodeForge application...")

	// Load configuration
	cfg, err := config.Load(appConfig.ConfigPath, appConfig.Debug)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Resolve workspace root
	workspaceRoot := appConfig.WorkspaceRoot
	if workspaceRoot == "" {
		workspaceRoot = "."
	}
	absWorkspace, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve workspace path: %w", err)
	}

	app := &App{
		Config:        cfg,
		WorkspaceRoot: absWorkspace,
	}

	// Initialize vector database
	if err := app.initializeVectorDB(appConfig.DatabasePath); err != nil {
		return nil, fmt.Errorf("failed to initialize vector database: %w", err)
	}

	// Initialize permission system
	if appConfig.EnablePermissions {
		if err := app.initializePermissionSystem(appConfig.PermissionDBPath); err != nil {
			return nil, fmt.Errorf("failed to initialize permission system: %w", err)
		}
	}

	// Initialize context management
	if appConfig.EnableContextMgmt {
		if err := app.initializeContextManagement(); err != nil {
			return nil, fmt.Errorf("failed to initialize context management: %w", err)
		}
	}

	// Initialize MCP server
	if err := app.initializeMCPServer(); err != nil {
		return nil, fmt.Errorf("failed to initialize MCP server: %w", err)
	}

	log.Printf("CodeForge application initialized successfully")
	log.Printf("Workspace: %s", absWorkspace)
	log.Printf("Permissions: %v", appConfig.EnablePermissions)
	log.Printf("Context Management: %v", appConfig.EnableContextMgmt)

	return app, nil
}

// initializeVectorDB initializes the vector database
func (app *App) initializeVectorDB(dbPath string) error {
	log.Printf("Initializing vector database...")

	// Set database path if not provided
	if dbPath == "" {
		dbPath = filepath.Join(app.WorkspaceRoot, ".codeforge", "vector.db")
	}

	// Initialize vector database
	if err := vectordb.Initialize(app.Config); err != nil {
		return fmt.Errorf("failed to initialize vector database: %w", err)
	}

	app.VectorDB = vectordb.Get()
	if app.VectorDB == nil {
		return fmt.Errorf("vector database not available after initialization")
	}

	log.Printf("Vector database initialized: %s", dbPath)
	return nil
}

// initializePermissionSystem initializes the permission system
func (app *App) initializePermissionSystem(permDBPath string) error {
	log.Printf("Initializing permission system...")

	// Set permission database path if not provided
	if permDBPath == "" {
		permDBPath = filepath.Join(app.WorkspaceRoot, ".codeforge", "permissions.db")
	}

	// Ensure .codeforge directory exists
	codeforgeDir := filepath.Dir(permDBPath)
	if err := os.MkdirAll(codeforgeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .codeforge directory: %w", err)
	}

	// Create permission configuration with defaults
	permConfig := &permissions.PermissionConfig{
		DefaultScope:             permissions.ScopeSession,
		DefaultExpiration:        24 * time.Hour,
		RequireApproval:          true,
		AutoApproveThreshold:     80,
		MaxPermissionsPerSession: 100,
		AuditEnabled:             true,
		CleanupInterval:          1 * time.Hour,
	}

	// Initialize permission service
	app.PermissionService = permissions.NewPermissionService(permConfig)

	// Initialize permission storage
	storage, err := permissions.NewPermissionStorage(permDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize permission storage: %w", err)
	}
	app.PermissionStorage = storage

	// Initialize file operation manager
	app.FileOperationManager = permissions.NewFileOperationManager(app.PermissionService, app.WorkspaceRoot)

	log.Printf("Permission system initialized: %s", permDBPath)
	return nil
}

// initializeContextManagement initializes the context management system
func (app *App) initializeContextManagement() error {
	log.Printf("Initializing context management...")

	// Initialize context manager with the app config
	app.ContextManager = contextmgmt.NewContextManager(app.Config)

	log.Printf("Context management initialized")
	return nil
}

// initializeMCPServer initializes the MCP server with all systems
func (app *App) initializeMCPServer() error {
	log.Printf("Initializing MCP server...")

	if app.PermissionService != nil {
		// Create permission-aware MCP server
		app.MCPServer = mcp.NewPermissionAwareCodeForgeServer(
			app.Config,
			app.VectorDB,
			app.WorkspaceRoot,
			app.PermissionService,
		)
		log.Printf("Permission-aware MCP server initialized")
	} else {
		// Create basic MCP server (fallback)
		baseServer := mcp.NewCodeForgeServer(app.Config, app.VectorDB, app.WorkspaceRoot)
		log.Printf("Basic MCP server initialized (no permissions)")

		// We need to wrap this in a compatible interface
		// For now, we'll create a minimal wrapper
		app.MCPServer = &mcp.PermissionAwareMCPServer{
			CodeForgeServer: baseServer,
		}
	}

	return nil
}

// GetContextManager returns the context manager (creates one if needed)
func (app *App) GetContextManager() *contextmgmt.ContextManager {
	if app.ContextManager == nil {
		// Create a default context manager if not initialized
		app.ContextManager = contextmgmt.NewContextManager(app.Config)
		log.Printf("Created default context manager")
	}
	return app.ContextManager
}

// GetPermissionService returns the permission service
func (app *App) GetPermissionService() *permissions.PermissionService {
	return app.PermissionService
}

// GetFileOperationManager returns the file operation manager
func (app *App) GetFileOperationManager() *permissions.FileOperationManager {
	return app.FileOperationManager
}

// ProcessChatMessage processes a chat message with full context management and permissions
func (app *App) ProcessChatMessage(ctx context.Context, sessionID, message, modelID string) (string, error) {
	// Check if we have permission to process messages
	if app.PermissionService != nil {
		check := &permissions.PermissionCheck{
			SessionID: sessionID,
			Type:      permissions.PermissionMCPCall,
			Resource:  "chat_processing",
			Context: map[string]interface{}{
				"message": message,
				"model":   modelID,
			},
		}

		result, err := app.PermissionService.CheckPermission(ctx, check)
		if err != nil {
			return "", fmt.Errorf("permission check failed: %w", err)
		}

		if !result.Allowed {
			return "", fmt.Errorf("permission denied: %s", result.Reason)
		}
	}

	// Use context management if available
	if app.ContextManager != nil {
		log.Printf("Processing message with context management for session %s", sessionID)
		// TODO: Integrate context management with chat processing
	}

	// Integrate with actual LLM processing using chat module
	return app.processWithLLM(ctx, message, modelID)
}

// processWithLLM processes a message using the LLM chat system
func (app *App) processWithLLM(ctx context.Context, message, modelID string) (string, error) {
	// Import the chat module
	chat := &chatModule{}

	// Use default model if none specified
	if modelID == "" {
		modelID = chat.GetDefaultModel()
	}

	// Get API key for the model
	apiKey := chat.GetAPIKeyForModel(modelID)
	if apiKey == "" {
		return "", fmt.Errorf("no API key found for model: %s", modelID)
	}

	// Create chat session
	session, err := chat.NewChatSession(modelID, apiKey, "", true, "text")
	if err != nil {
		return "", fmt.Errorf("failed to create chat session: %w", err)
	}

	// Process the message
	response, err := session.ProcessMessage(message)
	if err != nil {
		return "", fmt.Errorf("failed to process message: %w", err)
	}

	return response, nil
}

// chatModule wraps the actual chat module functions
type chatModule struct{}

func (c *chatModule) GetDefaultModel() string {
	// Import and use the actual chat module
	chat := getChatPackage()
	return chat.GetDefaultModel()
}

func (c *chatModule) GetAPIKeyForModel(model string) string {
	// Import and use the actual chat module
	chat := getChatPackage()
	return chat.GetAPIKeyForModel(model)
}

func (c *chatModule) NewChatSession(model, apiKey, provider string, quiet bool, format string) (*chatSessionWrapper, error) {
	// Import and use the actual chat module
	chat := getChatPackage()
	session, err := chat.NewChatSession(model, apiKey, provider, quiet, format)
	if err != nil {
		return nil, err
	}

	return &chatSessionWrapper{session: session}, nil
}

// chatSessionWrapper wraps a chat session
type chatSessionWrapper struct {
	session chatSessionInterface
}

type chatSessionInterface interface {
	ProcessMessage(message string) (string, error)
}

func (cs *chatSessionWrapper) ProcessMessage(message string) (string, error) {
	return cs.session.ProcessMessage(message)
}

// getChatPackage returns the chat package interface
func getChatPackage() chatPackageInterface {
	return &actualChatPackage{}
}

type chatPackageInterface interface {
	GetDefaultModel() string
	GetAPIKeyForModel(model string) string
	NewChatSession(model, apiKey, provider string, quiet bool, format string) (chatSessionInterface, error)
}

type actualChatPackage struct{}

func (a *actualChatPackage) GetDefaultModel() string {
	// For now, return a default model
	// This will be replaced with actual chat.GetDefaultModel() when import is working
	return "claude-3-5-sonnet-20241022"
}

func (a *actualChatPackage) GetAPIKeyForModel(model string) string {
	// For now, return empty - will be replaced with actual implementation
	// This will be replaced with actual chat.GetAPIKeyForModel() when import is working
	return ""
}

func (a *actualChatPackage) NewChatSession(model, apiKey, provider string, quiet bool, format string) (chatSessionInterface, error) {
	// For now, return a mock session
	// This will be replaced with actual chat.NewChatSession() when import is working
	return &mockChatSession{model: model}, nil
}

type mockChatSession struct {
	model string
}

func (m *mockChatSession) ProcessMessage(message string) (string, error) {
	// Return a simple response for now
	return fmt.Sprintf("Echo from %s: %s", m.model, message), nil
}

// Close closes all app resources
func (app *App) Close() error {
	log.Printf("Closing CodeForge application...")

	var errors []error

	// Close vector database
	if app.VectorDB != nil {
		if err := app.VectorDB.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close vector database: %w", err))
		}
	}

	// Close permission storage
	if app.PermissionStorage != nil {
		if err := app.PermissionStorage.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close permission storage: %w", err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("errors during close: %v", errors)
	}

	log.Printf("CodeForge application closed successfully")
	return nil
}

// DefaultAppConfig returns default application configuration
func DefaultAppConfig() *AppConfig {
	return &AppConfig{
		ConfigPath:        "",
		WorkspaceRoot:     ".",
		DatabasePath:      "",
		PermissionDBPath:  "",
		EnablePermissions: true,
		EnableContextMgmt: true,
		Debug:             false,
	}
}

// NewAppFromWorkspace creates an app instance from a workspace directory
func NewAppFromWorkspace(ctx context.Context, workspaceRoot string) (*App, error) {
	appConfig := DefaultAppConfig()
	appConfig.WorkspaceRoot = workspaceRoot
	return NewApp(ctx, appConfig)
}

// NewAppWithConfig creates an app instance with custom configuration
func NewAppWithConfig(ctx context.Context, configPath, workspaceRoot string, enablePermissions, enableContextMgmt bool) (*App, error) {
	appConfig := &AppConfig{
		ConfigPath:        configPath,
		WorkspaceRoot:     workspaceRoot,
		EnablePermissions: enablePermissions,
		EnableContextMgmt: enableContextMgmt,
		Debug:             false,
	}
	return NewApp(ctx, appConfig)
}
