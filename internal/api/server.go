package api

import (
    "context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/app"
	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/utils"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
    "strings"
)

// ConnectionManager manages active WebSocket connections
type ConnectionManager struct {
	chatConnections         map[string][]*ChatWebSocketClient
	notificationConnections map[string][]*NotificationWebSocketClient
	mu                      sync.RWMutex
}

// NewConnectionManager creates a new connection manager
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		chatConnections:         make(map[string][]*ChatWebSocketClient),
		notificationConnections: make(map[string][]*NotificationWebSocketClient),
	}
}

// AddChatConnection adds a chat WebSocket connection
func (cm *ConnectionManager) AddChatConnection(sessionID string, client *ChatWebSocketClient) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.chatConnections[sessionID] = append(cm.chatConnections[sessionID], client)
}

// RemoveChatConnection removes a chat WebSocket connection
func (cm *ConnectionManager) RemoveChatConnection(sessionID string, client *ChatWebSocketClient) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	connections := cm.chatConnections[sessionID]
	for i, conn := range connections {
		if conn == client {
			cm.chatConnections[sessionID] = append(connections[:i], connections[i+1:]...)
			break
		}
	}

	if len(cm.chatConnections[sessionID]) == 0 {
		delete(cm.chatConnections, sessionID)
	}
}

// AddNotificationConnection adds a notification WebSocket connection
func (cm *ConnectionManager) AddNotificationConnection(sessionID string, client *NotificationWebSocketClient) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.notificationConnections[sessionID] = append(cm.notificationConnections[sessionID], client)
}

// RemoveNotificationConnection removes a notification WebSocket connection
func (cm *ConnectionManager) RemoveNotificationConnection(sessionID string, client *NotificationWebSocketClient) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	connections := cm.notificationConnections[sessionID]
	for i, conn := range connections {
		if conn == client {
			cm.notificationConnections[sessionID] = append(connections[:i], connections[i+1:]...)
			break
		}
	}

	if len(cm.notificationConnections[sessionID]) == 0 {
		delete(cm.notificationConnections, sessionID)
	}
}

// BroadcastToSession broadcasts a message to all chat connections in a session
func (cm *ConnectionManager) BroadcastToSession(sessionID string, message WebSocketMessage) {
	cm.mu.RLock()
	connections := make([]*ChatWebSocketClient, len(cm.chatConnections[sessionID]))
	copy(connections, cm.chatConnections[sessionID])
	cm.mu.RUnlock()

	for _, conn := range connections {
		select {
		case conn.send <- message:
		default:
			// Connection is blocked, remove it
			cm.RemoveChatConnection(sessionID, conn)
		}
	}
}

// GetConnectionStats returns connection statistics
func (cm *ConnectionManager) GetConnectionStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	chatCount := 0
	notificationCount := 0

	for _, connections := range cm.chatConnections {
		chatCount += len(connections)
	}

	for _, connections := range cm.notificationConnections {
		notificationCount += len(connections)
	}

	return map[string]interface{}{
		"chat_connections":         chatCount,
		"notification_connections": notificationCount,
		"active_sessions":          len(cm.chatConnections),
	}
}

// BroadcastSystemEvent broadcasts a system event to all connected chat clients
func (cm *ConnectionManager) BroadcastSystemEvent(eventType string, payload interface{}) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	message := WebSocketMessage{
		Type: "system_broadcast",
		Data: map[string]interface{}{
			"event_type": eventType,
			"payload":    payload,
			"timestamp":  time.Now().Unix(),
		},
	}

	// Broadcast to all chat connections across all sessions
	for sessionID, connections := range cm.chatConnections {
		for _, conn := range connections {
			select {
			case conn.send <- message:
			default:
				// Connection is blocked, remove it
				log.Printf("Removing blocked connection for session %s", sessionID)
				go cm.RemoveChatConnection(sessionID, conn)
			}
		}
	}
}

// BroadcastProgressUpdate broadcasts a progress update for long-running operations
func (cm *ConnectionManager) BroadcastProgressUpdate(operationID string, progress float64, message string, sessionID string) {
	progressMessage := WebSocketMessage{
		Type: "progress_update",
		Data: map[string]interface{}{
			"operation_id": operationID,
			"progress":     progress,
			"message":      message,
			"timestamp":    time.Now().Unix(),
		},
	}

	if sessionID != "" {
		// Broadcast to specific session
		cm.BroadcastToSession(sessionID, progressMessage)
	} else {
		// Broadcast to all sessions
		cm.mu.RLock()
		defer cm.mu.RUnlock()

		for sid := range cm.chatConnections {
			cm.BroadcastToSession(sid, progressMessage)
		}
	}
}

// Server represents the API server
type Server struct {
	config            *config.Config
	vectorDB          *vectordb.VectorDB
	auth              *LocalhostAuth
	upgrader          websocket.Upgrader
	chatStorage       *ChatStorage
	app               *app.App // Integrated CodeForge application
	connectionManager *ConnectionManager
	gitignoreFilter   *utils.GitIgnoreFilter
    httpServer        *http.Server
}

// NewServer creates a new API server
func NewServer(cfg *config.Config) *Server {
	return &Server{
		config:            cfg,
		auth:              NewLocalhostAuth(),
		chatStorage:       NewChatStorage(),
		connectionManager: NewConnectionManager(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Only allow localhost connections for security
				return isLocalhostOrigin(r)
			},
		},
	}
}

// NewServerWithApp creates a new API server with integrated CodeForge app
func NewServerWithApp(cfg *config.Config, codeforgeApp *app.App) *Server {
	server := &Server{
		config:            cfg,
		vectorDB:          codeforgeApp.VectorDB,
		auth:              NewLocalhostAuth(),
		chatStorage:       NewChatStorage(),
		app:               codeforgeApp,
		connectionManager: NewConnectionManager(),
		gitignoreFilter:   utils.NewGitIgnoreFilter(cfg.WorkingDir),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Only allow localhost connections for security
				return isLocalhostOrigin(r)
			},
		},
	}

	// Set server reference in app for event broadcasting
	codeforgeApp.SetServer(server)

	return server
}

// isLocalhostOrigin checks if the WebSocket origin is localhost
func isLocalhostOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // Allow connections without origin (like from Postman)
	}

	// Allow localhost origins
	return origin == "http://localhost:47000" ||
		origin == "http://127.0.0.1:47000" ||
		origin == "http://[::1]:47000"
}

// Start starts the API server
func (s *Server) Start(port int) error {
	// Initialize dependencies
	if err := s.initializeDependencies(); err != nil {
		return fmt.Errorf("failed to initialize dependencies: %w", err)
	}

    router := s.setupRoutes()

    addr := fmt.Sprintf(":%d", port)
    log.Printf("🌐 Starting API server on %s", addr)

    s.httpServer = &http.Server{
        Addr:    addr,
        Handler: router,
    }
    return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the HTTP server
func (s *Server) Stop(ctx context.Context) error {
    if s.httpServer == nil {
        return nil
    }
    return s.httpServer.Shutdown(ctx)
}

// initializeDependencies initializes required services
func (s *Server) initializeDependencies() error {
	// Initialize vector database
	s.vectorDB = vectordb.Get()
	if s.vectorDB == nil {
		return fmt.Errorf("vector database not initialized")
	}

	// Initialize LLM system
	if err := s.initializeLLMSystem(); err != nil {
		log.Printf("Warning: Failed to initialize LLM system: %v", err)
	}

	return nil
}

// initializeLLMSystem initializes the LLM system for the API server
func (s *Server) initializeLLMSystem() error {
	// The LLM system is initialized on-demand when creating chat sessions
	// This ensures we always use the latest API keys from environment variables
	return nil
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Enable CORS
	router.Use(s.corsMiddleware)

	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Authentication endpoints (no auth required)
	api.HandleFunc("/auth", s.handleAuth).Methods("GET", "POST", "DELETE")
	api.HandleFunc("/auth/info", s.handleAuthInfo).Methods("GET")

	// Apply authentication middleware to protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(s.auth.AuthMiddleware)

	// Chat endpoints (protected)
	protected.HandleFunc("/chat/sessions", s.handleChatSessions).Methods("GET", "POST")
	protected.HandleFunc("/chat/sessions/{id}", s.handleChatSession).Methods("GET", "DELETE")
	protected.HandleFunc("/chat/sessions/{id}/messages", s.handleChatMessages).Methods("GET", "POST")
	protected.HandleFunc("/chat/sessions/{sessionID}/messages/enhanced", s.sendChatMessageEnhanced).Methods("POST")

	// WebSocket for real-time chat (protected via token in URL)
	protected.HandleFunc("/chat/ws/{sessionId}", s.handleChatWebSocket)

	// WebSocket for real-time notifications (protected via token in URL)
	protected.HandleFunc("/notifications/ws/{sessionId}", s.handleNotificationWebSocket)

	// SSE for metrics and status (protected)
	protected.HandleFunc("/events/metrics", s.handleMetricsSSE)
	protected.HandleFunc("/events/status", s.handleStatusSSE)

	// WebSocket connection management (protected)
	protected.HandleFunc("/websocket/stats", s.handleWebSocketStats).Methods("GET")

	// Project and file management (protected)
	protected.HandleFunc("/project/structure", s.handleProjectStructure).Methods("GET")
	protected.HandleFunc("/project/files", s.handleProjectFiles).Methods("GET")
	protected.HandleFunc("/project/search", s.handleProjectSearch).Methods("POST")

	// Code analysis (protected)
	protected.HandleFunc("/code/analyze", s.handleCodeAnalysis).Methods("POST")
	protected.HandleFunc("/code/symbols", s.handleCodeSymbols).Methods("POST")

	// LLM providers and models (protected)
	protected.HandleFunc("/llm/providers", s.handleLLMProviders).Methods("GET")
	protected.HandleFunc("/llm/models", s.handleLLMModels).Methods("GET")
	protected.HandleFunc("/llm/models/{provider}", s.handleProviderModels).Methods("GET")

	// Provider management (protected)
	protected.HandleFunc("/providers", s.handleProviders).Methods("GET")
	protected.HandleFunc("/providers/{id}", s.handleProvider).Methods("GET", "PUT", "DELETE")
	protected.HandleFunc("/providers/embedding", s.handleEmbeddingProvider).Methods("GET", "PUT")

	// Configuration (protected)
	protected.HandleFunc("/config", s.handleConfig).Methods("GET", "PUT")

	// Environment variables (protected)
	protected.HandleFunc("/environment", s.handleEnvironment).Methods("GET", "PUT")
	protected.PathPrefix("/environment/").HandlerFunc(s.handleEnvironmentVariable).Methods("GET", "PUT", "DELETE")

	// Health check (public)
	api.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Static file serving for web UI
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./web/dist/")))

	return router
}

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Restrict default CORS to localhost; can be expanded via config later
        origin := r.Header.Get("Origin")
        allow := ""
        if origin == "" || strings.HasPrefix(origin, "http://localhost:") || strings.HasPrefix(origin, "http://127.0.0.1:") || strings.HasPrefix(origin, "http://[::1]:") {
            allow = origin
        }
        if allow == "" {
            allow = "http://localhost:47000"
        }
        w.Header().Set("Access-Control-Allow-Origin", allow)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Response helpers
func (s *Server) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func (s *Server) writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// Health check endpoint
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"services": map[string]bool{
			"vectordb": s.vectorDB != nil,
			"api":      true,
		},
	}
	s.writeJSON(w, health)
}

// handleWebSocketStats returns WebSocket connection statistics
func (s *Server) handleWebSocketStats(w http.ResponseWriter, r *http.Request) {
	stats := s.connectionManager.GetConnectionStats()
	stats["timestamp"] = time.Now().Unix()
	s.writeJSON(w, stats)
}

// BroadcastSystemEvent broadcasts a system event to all connected clients
func (s *Server) BroadcastSystemEvent(eventType string, payload interface{}) {
	if s.connectionManager != nil {
		s.connectionManager.BroadcastSystemEvent(eventType, payload)
	}
}

// BroadcastProgressUpdate broadcasts a progress update for long-running operations
func (s *Server) BroadcastProgressUpdate(operationID string, progress float64, message string, sessionID string) {
	if s.connectionManager != nil {
		s.connectionManager.BroadcastProgressUpdate(operationID, progress, message, sessionID)
	}
}
