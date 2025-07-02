package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/chat"
	"github.com/entrepeneur4lyf/codeforge/internal/config"
	contextmgmt "github.com/entrepeneur4lyf/codeforge/internal/context"
	"github.com/entrepeneur4lyf/codeforge/internal/events"
	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/providers"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/tools"
	"github.com/entrepeneur4lyf/codeforge/internal/lsp"
	"github.com/entrepeneur4lyf/codeforge/internal/mcp"
	"github.com/entrepeneur4lyf/codeforge/internal/models"
	"github.com/entrepeneur4lyf/codeforge/internal/notifications"
	"github.com/entrepeneur4lyf/codeforge/internal/permissions"
	"github.com/entrepeneur4lyf/codeforge/internal/storage"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
	"github.com/google/uuid"
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
	ToolRegistry         *tools.ToolRegistry
	ChatStore            storage.ChatStore
	PathManager          *storage.PathManager
	ModelManager         *models.ModelManager
	ModelRegistry        *models.ModelRegistry
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

	// Initialize path manager for standardized storage locations
	pathManager := storage.NewPathManager()
	
	app := &App{
		Config:        cfg,
		PathManager:   pathManager,
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

	// Initialize tool registry
	if err := app.initializeToolRegistry(); err != nil {
		return nil, fmt.Errorf("failed to initialize tool registry: %w", err)
	}

	// Initialize models system
	if err := app.initializeModelsSystem(); err != nil {
		return nil, fmt.Errorf("failed to initialize models system: %w", err)
	}

	// Initialize chat store
	if err := app.initializeChatStore(); err != nil {
		return nil, fmt.Errorf("failed to initialize chat store: %w", err)
	}

	// Link managers to context manager for tool context integration
	if app.ContextManager != nil {
		if app.MCPManager != nil {
			app.ContextManager.SetMCPManager(app.MCPManager)
			log.Printf("MCP manager linked to context manager for tool context integration")
		}
		if app.ToolRegistry != nil {
			app.ContextManager.SetToolRegistry(app.ToolRegistry)
			log.Printf("Tool registry linked to context manager for built-in tool context integration")
		}
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

	// Set database path if not provided - use standardized user directory
	if dbPath == "" {
		var err error
		dbPath, err = app.PathManager.GetVectorDatabasePath()
		if err != nil {
			return fmt.Errorf("failed to get vector database path: %w", err)
		}
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

	// Set permission database path if not provided - use standardized user directory
	if permDBPath == "" {
		var err error
		permDBPath, err = app.PathManager.GetPermissionDatabasePath()
		if err != nil {
			return fmt.Errorf("failed to get permission database path: %w", err)
		}
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

	// Create config directory path - use standardized user directory
	configDir, err := app.PathManager.GetMCPConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get MCP config directory: %w", err)
	}

	// Initialize MCP manager
	app.MCPManager = mcp.NewMCPManager(configDir)

	// Initialize the manager (this starts enabled servers)
	if err := app.MCPManager.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize MCP manager: %w", err)
	}

	log.Printf("MCP manager initialized")
	return nil
}

// initializeToolRegistry initializes the built-in tool registry
func (app *App) initializeToolRegistry() error {
	log.Printf("Initializing tool registry...")

	// Get LSP manager for tool registry initialization
	var lspManager *lsp.Manager
	if mgr := lsp.GetManager(); mgr != nil {
		lspManager = mgr
	}

	// Initialize tool registry with available services
	app.ToolRegistry = tools.NewToolRegistry(lspManager, app.PermissionService)

	log.Printf("Tool registry initialized with built-in tools")
	return nil
}

// initializeChatStore initializes the chat storage system
func (app *App) initializeChatStore() error {
	log.Printf("Initializing chat store...")

	// Use default chat store with standardized user directory
	chatStore, err := storage.NewDefaultChatStore()
	if err != nil {
		return fmt.Errorf("failed to create chat store: %w", err)
	}

	app.ChatStore = chatStore

	// Validate storage paths
	if err := app.PathManager.ValidatePaths(); err != nil {
		return fmt.Errorf("failed to validate storage paths: %w", err)
	}

	// Log storage location
	dbPath, _ := app.PathManager.GetChatDatabasePath()
	codeforgeDir, _ := app.PathManager.GetCodeForgeDir()
	
	log.Printf("Chat store initialized: %s", dbPath)
	log.Printf("CodeForge user directory: %s", codeforgeDir)
	
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

	// Close chat store
	if app.ChatStore != nil {
		if err := app.ChatStore.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close chat store: %w", err))
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

// initializeModelsSystem initializes the advanced models management system
func (app *App) initializeModelsSystem() error {
	log.Printf("Initializing advanced models system...")

	// Initialize model registry with canonical models
	app.ModelRegistry = models.NewModelRegistry()

	// Initialize model manager with intelligent selection
	app.ModelManager = models.NewModelManager(app.ModelRegistry)

	log.Printf("Models system initialized with canonical registry and intelligent management")
	return nil
}

// GetAvailableModels returns all available LLM models
func (app *App) GetAvailableModels() []llm.ModelResponse {
	return llm.GetAvailableModels()
}

// GetCanonicalModels returns all canonical models with advanced metadata
func (app *App) GetCanonicalModels() []*models.CanonicalModel {
	if app.ModelRegistry == nil {
		return []*models.CanonicalModel{}
	}
	return app.ModelRegistry.ListModels()
}

// GetModelRecommendation returns intelligent model recommendations
func (app *App) GetModelRecommendation(ctx context.Context, criteria models.ModelSelectionCriteria) (*models.ModelRecommendation, error) {
	if app.ModelManager == nil {
		return nil, fmt.Errorf("model manager not initialized")
	}
	return app.ModelManager.GetRecommendation(ctx, criteria)
}

// GetFavoriteModels returns user's favorite models
func (app *App) GetFavoriteModels() []*models.CanonicalModel {
	if app.ModelManager == nil {
		return []*models.CanonicalModel{}
	}
	return app.ModelManager.GetFavoriteModels()
}

// AddFavoriteModel adds a model to user's favorites
func (app *App) AddFavoriteModel(modelID models.CanonicalModelID) error {
	if app.ModelManager == nil {
		return fmt.Errorf("model manager not initialized")
	}
	return app.ModelManager.AddFavorite(modelID)
}

// RemoveFavoriteModel removes a model from user's favorites
func (app *App) RemoveFavoriteModel(modelID models.CanonicalModelID) {
	if app.ModelManager != nil {
		app.ModelManager.RemoveFavorite(modelID)
	}
}

// UpdateModelPreferences updates user model preferences
func (app *App) UpdateModelPreferences(prefs models.UserPreferences) {
	if app.ModelManager != nil {
		app.ModelManager.UpdatePreferences(prefs)
	}
}

// GetModelPreferences returns current user model preferences
func (app *App) GetModelPreferences() models.UserPreferences {
	if app.ModelManager == nil {
		return models.UserPreferences{}
	}
	return app.ModelManager.GetPreferences()
}

// SetCurrentModel sets the current model for the app
func (app *App) SetCurrentModel(provider, model string) error {
	// Validate the model exists
	available := llm.GetAvailableModels()
	found := false
	for _, m := range available {
		if m.Provider == provider && m.Name == model {
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("model %s/%s not found", provider, model)
	}
	
	// In a real implementation, we might store this preference
	// For now, the model is specified per request
	return nil
}

// GetCurrentModel returns the currently selected model
func (app *App) GetCurrentModel() (provider, model string) {
	// Return default model for now
	defaultModel := llm.GetDefaultModel()
	return defaultModel.Provider, defaultModel.Name
}

// ProcessChatMessageWithStream processes a chat message with context management and returns a stream channel
func (app *App) ProcessChatMessageWithStream(ctx context.Context, sessionID, message, modelID string) (*contextmgmt.ProcessedContext, <-chan string, error) {
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
			return nil, nil, fmt.Errorf("permission check failed: %w", err)
		}

		if !result.Allowed {
			return nil, nil, fmt.Errorf("permission denied: %s", result.Reason)
		}
	}

	// Ensure session exists and save user message to database
	if app.ChatStore != nil {
		// Check if session exists, create if not
		existingSession, err := app.ChatStore.GetSession(ctx, sessionID)
		if err != nil {
			// Session doesn't exist, create it
			now := time.Now()
			newSession := &storage.Session{
				ID:           sessionID,
				Title:        "New Chat",
				Model:        modelID,
				CreatedAt:    now,
				UpdatedAt:    now,
				MessageCount: 0,
				TotalTokens:  0,
				Metadata:     map[string]interface{}{"created_by": "app"},
			}
			
			if err := app.ChatStore.CreateSession(ctx, newSession); err != nil {
				log.Printf("Warning: Failed to create session: %v", err)
			} else {
				log.Printf("Created new chat session: %s", sessionID)
			}
		} else {
			log.Printf("Using existing session: %s (title: %s)", sessionID, existingSession.Title)
		}

		// Save user message to database
		userMsgID := uuid.New().String()
		userMsg := &storage.Message{
			ID:        userMsgID,
			SessionID: sessionID,
			Role:      "user",
			Content:   message,
			Tokens:    0, // Will be calculated after processing
			CreatedAt: time.Now(),
			Metadata:  map[string]interface{}{"model": modelID},
		}

		if err := app.ChatStore.SaveMessage(ctx, userMsg); err != nil {
			log.Printf("Warning: Failed to save user message: %v", err)
		} else {
			log.Printf("Saved user message to database: %s", userMsgID)
		}
	}

	// Build context using context management
	var processedCtx *contextmgmt.ProcessedContext
	var fullContext []contextmgmt.ConversationMessage
	
	if app.ContextManager != nil {
		log.Printf("Building context for streaming response (session %s)", sessionID)

		// Create conversation message for context processing
		conversationMsg := contextmgmt.ConversationMessage{
			Role:      "user",
			Content:   message,
			Timestamp: time.Now().Unix(),
			Metadata:  map[string]interface{}{"session_id": sessionID},
		}

		// Process with relevance filtering for better context
		var err error
		processedCtx, err = app.ContextManager.ProcessWithRelevanceFiltering(
			ctx,
			[]contextmgmt.ConversationMessage{conversationMsg},
			modelID,
			message,
			0.3, // relevance threshold
		)
		if err != nil {
			log.Printf("Warning: Failed to process context: %v", err)
			// Fall back to simple message
			fullContext = []contextmgmt.ConversationMessage{conversationMsg}
		} else {
			fullContext = processedCtx.Messages
			log.Printf("Context processed: %d messages, %d tokens", len(fullContext), processedCtx.FinalTokens)
			// Log first few messages for debugging
			for i, msg := range fullContext {
				if i < 3 {
					preview := msg.Content
					if len(preview) > 100 {
						preview = preview[:100] + "..."
					}
					log.Printf("  Message %d [%s]: %s", i, msg.Role, preview)
				}
			}
		}
	} else {
		// No context manager, use simple message
		fullContext = []contextmgmt.ConversationMessage{
			{
				Role:      "user",
				Content:   message,
				Timestamp: time.Now().Unix(),
			},
		}
	}

	// Create stream channel for response
	streamChan := make(chan string, 100)
	
	// Process with LLM in background
	go func() {
		defer close(streamChan)
		
		// Get LLM handler
		handler := app.GetLLMHandler(modelID)
		if handler == nil {
			streamChan <- fmt.Sprintf("Error: Failed to get handler for model %s", modelID)
			return
		}

		// Convert context messages to LLM messages
		messages := []llm.Message{}
		
		// Build system prompt from first system messages in context
		systemPrompt := ""
		for _, msg := range fullContext {
			if msg.Role == "system" {
				if systemPrompt != "" {
					systemPrompt += "\n\n"
				}
				systemPrompt += msg.Content
			} else {
				// Add non-system messages
				messages = append(messages, llm.Message{
					Role: msg.Role,
					Content: []llm.ContentBlock{
						llm.TextBlock{Text: msg.Content},
					},
				})
			}
		}
		
		// If no system prompt found, use default
		if systemPrompt == "" {
			systemPrompt = "You are CodeForge, an AI coding assistant."
		}

		// Stream response from LLM
		stream, err := handler.CreateMessage(ctx, systemPrompt, messages)
		if err != nil {
			streamChan <- fmt.Sprintf("Error: %v", err)
			return
		}

		// Forward chunks to stream channel
		fullResponse := ""
		for chunk := range stream {
			if textChunk, ok := chunk.(llm.ApiStreamTextChunk); ok {
				if textChunk.Text != "" {
					streamChan <- textChunk.Text
					fullResponse += textChunk.Text
				}
			}
		}

		// Save assistant message to database
		if app.ChatStore != nil && fullResponse != "" {
			assistantMsgID := uuid.New().String()
			assistantMsg := &storage.Message{
				ID:        assistantMsgID,
				SessionID: sessionID,
				Role:      "assistant",
				Content:   fullResponse,
				Tokens:    0, // Will be calculated if needed
				CreatedAt: time.Now(),
				Metadata:  map[string]interface{}{"model": modelID},
			}

			if err := app.ChatStore.SaveMessage(ctx, assistantMsg); err != nil {
				log.Printf("Warning: Failed to save assistant message: %v", err)
			} else {
				log.Printf("Saved assistant message to database: %s", assistantMsgID)
			}

			// Save context snapshot if available
			if processedCtx != nil {
				snapshotID := uuid.New().String()
				snapshot := &storage.ContextSnapshot{
					ID:               snapshotID,
					SessionID:        sessionID,
					ProcessedContext: map[string]interface{}{
						"original_count":    processedCtx.OriginalCount,
						"final_count":       processedCtx.FinalCount,
						"original_tokens":   processedCtx.OriginalTokens,
						"final_tokens":      processedCtx.FinalTokens,
						"compression_ratio": processedCtx.CompressionRatio,
						"model_id":          processedCtx.ModelID,
						"cache_key":         processedCtx.CacheKey,
					},
					OriginalTokens:   processedCtx.OriginalTokens,
					FinalTokens:      processedCtx.FinalTokens,
					CompressionRatio: processedCtx.CompressionRatio,
					ModelID:          processedCtx.ModelID,
					ProcessingSteps:  processedCtx.ProcessingSteps,
					CreatedAt:        time.Now(),
				}

				if err := app.ChatStore.SaveContextSnapshot(ctx, snapshot); err != nil {
					log.Printf("Warning: Failed to save context snapshot: %v", err)
				} else {
					log.Printf("Saved context snapshot to database: %s", snapshotID)
				}
			}
		}

		// Publish chat message sent event
		if app.EventManager != nil {
			app.EventManager.PublishChat(events.ChatMessageSent, events.ChatEventPayload{
				SessionID: sessionID,
				Role:      "assistant",
				Content:   fullResponse,
				Model:     modelID,
			}, events.WithSessionID(sessionID))
		}

		// Show success notification
		if app.NotificationManager != nil {
			app.NotificationManager.Success("Response Ready",
				"Your message has been processed successfully",
				notifications.WithSessionID(sessionID))
		}
	}()

	return processedCtx, streamChan, nil
}

// GetLLMHandler returns an LLM handler for the specified model
func (app *App) GetLLMHandler(modelID string) llm.ApiHandler {
	// Parse provider from model ID
	provider := ""
	if strings.Contains(modelID, "/") {
		parts := strings.Split(modelID, "/")
		if len(parts) >= 2 {
			provider = parts[0]
			modelID = strings.Join(parts[1:], "/")
		}
	} else {
		// Detect provider from model ID
		if strings.HasPrefix(modelID, "claude-") {
			provider = "anthropic"
		} else if strings.HasPrefix(modelID, "gpt-") || strings.HasPrefix(modelID, "o1-") {
			provider = "openai"
		} else if strings.HasPrefix(modelID, "gemini-") {
			provider = "gemini"
		}
	}
	
	// Get API key for provider
	apiKey := ""
	switch provider {
	case "anthropic":
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	case "openai":
		apiKey = os.Getenv("OPENAI_API_KEY")
	case "gemini":
		apiKey = os.Getenv("GEMINI_API_KEY")
	case "openrouter":
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	default:
		// Try to find any available key
		if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
			provider = "openrouter"
			apiKey = key
		}
	}
	
	if apiKey == "" {
		return nil
	}
	
	// Create handler options
	options := llm.ApiHandlerOptions{
		APIKey:  apiKey,
		ModelID: modelID,
	}
	
	// Build the handler
	handler, err := providers.BuildApiHandler(options)
	if err != nil {
		log.Printf("Failed to create LLM handler: %v", err)
		return nil
	}
	
	return handler
}

// Session Management Methods

// GetChatSessions returns a list of chat sessions for the user
func (app *App) GetChatSessions(ctx context.Context, userID string, limit, offset int) ([]*storage.SessionSummary, error) {
	if app.ChatStore == nil {
		return nil, fmt.Errorf("chat store not initialized")
	}
	return app.ChatStore.ListSessions(ctx, userID, limit, offset)
}

// GetChatSession returns a specific chat session
func (app *App) GetChatSession(ctx context.Context, sessionID string) (*storage.Session, error) {
	if app.ChatStore == nil {
		return nil, fmt.Errorf("chat store not initialized")
	}
	return app.ChatStore.GetSession(ctx, sessionID)
}

// GetChatMessages returns messages for a session with pagination
func (app *App) GetChatMessages(ctx context.Context, sessionID string, limit, offset int) (*storage.MessageBatch, error) {
	if app.ChatStore == nil {
		return nil, fmt.Errorf("chat store not initialized")
	}
	return app.ChatStore.GetMessages(ctx, sessionID, limit, offset)
}

// GetLatestChatMessages returns the most recent messages for a session
func (app *App) GetLatestChatMessages(ctx context.Context, sessionID string, limit int) ([]storage.Message, error) {
	if app.ChatStore == nil {
		return nil, fmt.Errorf("chat store not initialized")
	}
	return app.ChatStore.GetLatestMessages(ctx, sessionID, limit)
}

// CreateChatSession creates a new chat session
func (app *App) CreateChatSession(ctx context.Context, title, modelID string) (*storage.Session, error) {
	if app.ChatStore == nil {
		return nil, fmt.Errorf("chat store not initialized")
	}

	now := time.Now()
	session := &storage.Session{
		ID:           uuid.New().String(),
		Title:        title,
		Model:        modelID,
		CreatedAt:    now,
		UpdatedAt:    now,
		MessageCount: 0,
		TotalTokens:  0,
		Metadata:     map[string]interface{}{"created_by": "user"},
	}

	if err := app.ChatStore.CreateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	log.Printf("Created new chat session: %s (title: %s)", session.ID, session.Title)
	return session, nil
}

// UpdateChatSession updates an existing chat session
func (app *App) UpdateChatSession(ctx context.Context, sessionID, title string) error {
	if app.ChatStore == nil {
		return fmt.Errorf("chat store not initialized")
	}

	session, err := app.ChatStore.GetSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	session.Title = title
	session.UpdatedAt = time.Now()

	if err := app.ChatStore.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	log.Printf("Updated chat session: %s (new title: %s)", sessionID, title)
	return nil
}

// DeleteChatSession deletes a chat session and all its messages
func (app *App) DeleteChatSession(ctx context.Context, sessionID string) error {
	if app.ChatStore == nil {
		return fmt.Errorf("chat store not initialized")
	}

	if err := app.ChatStore.DeleteSession(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	log.Printf("Deleted chat session: %s", sessionID)
	return nil
}

// GetChatStorageStats returns statistics about chat storage
func (app *App) GetChatStorageStats(ctx context.Context) (map[string]interface{}, error) {
	if app.ChatStore == nil {
		return nil, fmt.Errorf("chat store not initialized")
	}
	return app.ChatStore.GetStats(ctx)
}
