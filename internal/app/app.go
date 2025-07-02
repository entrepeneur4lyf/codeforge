package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/chat"
	"github.com/entrepeneur4lyf/codeforge/internal/config"
	contextmgmt "github.com/entrepeneur4lyf/codeforge/internal/context"
	"github.com/entrepeneur4lyf/codeforge/internal/events"
	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/mcp"
	"github.com/entrepeneur4lyf/codeforge/internal/notifications"
	"github.com/entrepeneur4lyf/codeforge/internal/permissions"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

// App represents the main CodeForge application with all integrated systems
type App struct {
	Config              *config.Config
	VectorDB            *vectordb.VectorDB
	ContextManager      *contextmgmt.ContextManager
	EventManager        *events.Manager
	NotificationManager *notifications.Manager

	PermissionService    *permissions.PermissionService
	PermissionStorage    *permissions.PermissionStorage
	FileOperationManager *permissions.FileOperationManager
	MCPServer            *mcp.PermissionAwareMCPServer
	MCPManager           *mcp.MCPManager
	WorkspaceRoot        string

	// Server reference for broadcasting events (set externally)
	server interface {
		BroadcastProgressUpdate(operationID string, progress float64, message string, sessionID string)
		BroadcastSystemEvent(eventType string, payload interface{})
	}
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

	// Initialize event system
	if err := app.initializeEventSystem(); err != nil {
		return nil, fmt.Errorf("failed to initialize event system: %w", err)
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

	// Initialize MCP manager
	if err := app.initializeMCPManager(); err != nil {
		return nil, fmt.Errorf("failed to initialize MCP manager: %w", err)
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
		// Create basic MCP server (fallback when permissions are disabled)
		baseServer := mcp.NewCodeForgeServer(app.Config, app.VectorDB, app.WorkspaceRoot)
		log.Printf("Basic MCP server initialized (no permissions)")

		// Wrap in permission-aware interface for consistency
		app.MCPServer = &mcp.PermissionAwareMCPServer{
			CodeForgeServer: baseServer,
		}
	}

	return nil
}

// initializeMCPManager initializes the MCP manager for external MCP servers
func (app *App) initializeMCPManager() error {
	log.Printf("Initializing MCP manager...")

	// Create config directory path
	configDir := filepath.Join(app.WorkspaceRoot, ".codeforge")

	// Initialize MCP manager
	app.MCPManager = mcp.NewMCPManager(configDir)

	// Initialize the manager (this starts enabled servers)
	if err := app.MCPManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize MCP manager: %w", err)
	}

	log.Printf("MCP manager initialized")
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

// SetServer sets the server reference for event broadcasting
func (app *App) SetServer(server interface {
	BroadcastProgressUpdate(operationID string, progress float64, message string, sessionID string)
	BroadcastSystemEvent(eventType string, payload interface{})
}) {
	app.server = server
}

// ProcessChatMessage processes a chat message with full context management and permissions
func (app *App) ProcessChatMessage(ctx context.Context, sessionID, message, modelID string) (string, error) {
	// Publish chat message received event
	if app.EventManager != nil {
		app.EventManager.PublishChat(events.ChatMessageReceived, events.ChatEventPayload{
			SessionID: sessionID,
			Role:      "user",
			Content:   message,
			Model:     modelID,
		}, events.WithSessionID(sessionID))
	}

	// Show notification for message processing
	if app.NotificationManager != nil {
		app.NotificationManager.Info("Processing Message",
			fmt.Sprintf("Processing your message with %s...", modelID),
			notifications.WithSessionID(sessionID),
			notifications.WithDuration(3*time.Second))
	}

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
	var contextualMessage string
	if app.ContextManager != nil {
		log.Printf("Processing message with context management for session %s", sessionID)

		// Create conversation message for context processing
		conversationMsg := contextmgmt.ConversationMessage{
			Role:      "user",
			Content:   message,
			Timestamp: time.Now().Unix(),
			Metadata:  map[string]interface{}{"session_id": sessionID},
		}

		// Process with relevance filtering for better context
		processedCtx, err := app.ContextManager.ProcessWithRelevanceFiltering(
			ctx,
			[]contextmgmt.ConversationMessage{conversationMsg},
			modelID,
			message,
			0.3, // relevance threshold
		)
		if err != nil {
			log.Printf("Warning: Failed to process context: %v", err)
			contextualMessage = message
		} else {
			// Use the processed context
			if len(processedCtx.Messages) > 0 {
				contextualMessage = processedCtx.Messages[0].Content
			} else {
				contextualMessage = message
			}
		}
	} else {
		contextualMessage = message
	}

	// Integrate with actual LLM processing using chat module
	response, err := app.processWithLLM(ctx, contextualMessage, modelID, sessionID)
	if err != nil {
		return "", err
	}

	// Publish chat message sent event
	if app.EventManager != nil {
		app.EventManager.PublishChat(events.ChatMessageSent, events.ChatEventPayload{
			SessionID: sessionID,
			Role:      "assistant",
			Content:   response,
			Model:     modelID,
		}, events.WithSessionID(sessionID))
	}

	// Show success notification
	if app.NotificationManager != nil {
		app.NotificationManager.Success("Response Ready",
			"Your message has been processed successfully",
			notifications.WithSessionID(sessionID))
	}

	return response, nil
}

// processWithLLM processes a message using the LLM chat system
func (app *App) processWithLLM(ctx context.Context, message, modelID, sessionID string) (string, error) {
	// Generate operation ID for progress tracking
	operationID := fmt.Sprintf("llm_%s_%d", sessionID, time.Now().Unix())

	// Simulate processing with progress updates
	steps := []struct {
		progress float64
		message  string
	}{
		{0.1, "Initializing LLM request..."},
		{0.3, "Analyzing message content..."},
		{0.6, "Generating response..."},
		{0.9, "Finalizing output..."},
		{1.0, "Complete"},
	}

	for _, step := range steps {
		// Broadcast progress update if server is available
		if app.server != nil {
			app.server.BroadcastProgressUpdate(operationID, step.progress, step.message, sessionID)
		}

		// Simulate processing time
		time.Sleep(200 * time.Millisecond)
	}
	// Use the actual chat and llm modules directly
	chatModule := &realChatModule{}
	llmModule := &realLLMModule{}

	// Use default model if none specified
	if modelID == "" {
		modelID = chatModule.GetDefaultModel()
	}

	// Get API key for the model
	apiKey := chatModule.GetAPIKeyForModel(modelID)
	if apiKey == "" {
		return "", fmt.Errorf("no API key found for model: %s", modelID)
	}

	// Create chat session using the actual chat module
	session, err := chatModule.NewChatSession(modelID, apiKey, "", true, "text")
	if err != nil {
		return "", fmt.Errorf("failed to create chat session: %w", err)
	}

	// Process the message using the actual LLM
	response, err := session.ProcessMessage(message)
	if err != nil {
		// Fallback to direct LLM completion if chat session fails
		return app.processWithDirectLLM(ctx, message, modelID, llmModule)
	}

	return response, nil
}

// processWithDirectLLM processes a message using direct LLM completion
func (app *App) processWithDirectLLM(ctx context.Context, message, modelID string, llmModule *realLLMModule) (string, error) {
	// Get the default model info
	defaultModel := llmModule.GetDefaultModel()
	if modelID == "" {
		modelID = defaultModel.ID
	}

	// Create completion request
	temp := 0.7
	completionReq := llmModule.CreateCompletionRequest(modelID, message, &temp, defaultModel.Info.MaxTokens)

	// Get completion
	resp, err := llmModule.GetCompletion(ctx, completionReq)
	if err != nil {
		return "", fmt.Errorf("LLM completion failed: %w", err)
	}

	return resp.Content, nil
}

// realChatModule implements the actual chat module integration
type realChatModule struct{}

func (c *realChatModule) GetDefaultModel() string {
	return chat.GetDefaultModel()
}

func (c *realChatModule) GetAPIKeyForModel(model string) string {
	return chat.GetAPIKeyForModel(model)
}

func (c *realChatModule) NewChatSession(model, apiKey, provider string, quiet bool, format string) (*realChatSession, error) {
	session, err := chat.NewChatSession(model, apiKey, provider, quiet, format)
	if err != nil {
		return nil, err
	}

	return &realChatSession{session: session}, nil
}

// realChatSession wraps the actual chat session
type realChatSession struct {
	session *chat.ChatSession
}

func (cs *realChatSession) ProcessMessage(message string) (string, error) {
	return cs.session.ProcessMessage(message)
}

// realLLMModule implements the actual LLM module integration
type realLLMModule struct{}

func (l *realLLMModule) GetDefaultModel() llm.ModelResponse {
	return llm.GetDefaultModel()
}

func (l *realLLMModule) CreateCompletionRequest(modelID, message string, temperature *float64, maxTokens int) llm.CompletionRequest {
	return llm.CompletionRequest{
		Model: modelID,
		Messages: []llm.Message{
			{
				Role: "user",
				Content: []llm.ContentBlock{
					llm.TextBlock{Text: message},
				},
			},
		},
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}
}

func (l *realLLMModule) GetCompletion(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	return llm.GetCompletion(ctx, req)
}

// initializeEventSystem initializes the event management system
func (app *App) initializeEventSystem() error {
	log.Printf("Initializing event system...")

	// Create event manager
	app.EventManager = events.NewManager()

	// Set up in-memory persistence for now
	persistence := events.NewMemoryPersistenceStore(10000)
	app.EventManager.SetPersistence(persistence)

	// Start the event manager
	if err := app.EventManager.Start(); err != nil {
		return fmt.Errorf("failed to start event manager: %w", err)
	}

	// Create notification manager
	app.NotificationManager = notifications.NewManager(app.EventManager)

	// Start the notification manager
	if err := app.NotificationManager.Start(); err != nil {
		return fmt.Errorf("failed to start notification manager: %w", err)
	}

	log.Printf("Event system and notification manager initialized")
	return nil
}

// Close closes all app resources
func (app *App) Close() error {
	log.Printf("Closing CodeForge application...")

	var errors []error

	// Close notification manager
	if app.NotificationManager != nil {
		app.NotificationManager.Stop()
	}

	// Close event manager
	if app.EventManager != nil {
		app.EventManager.Shutdown()
	}

	// Shutdown MCP manager
	if app.MCPManager != nil {
		if err := app.MCPManager.Shutdown(); err != nil {
			errors = append(errors, fmt.Errorf("failed to shutdown MCP manager: %w", err))
		}
	}

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
