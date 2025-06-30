package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/app"
	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Server represents the API server
type Server struct {
	config      *config.Config
	vectorDB    *vectordb.VectorDB
	auth        *LocalhostAuth
	upgrader    websocket.Upgrader
	chatStorage *ChatStorage
	app         *app.App // Integrated CodeForge application
}

// NewServer creates a new API server
func NewServer(cfg *config.Config) *Server {
	return &Server{
		config:      cfg,
		auth:        NewLocalhostAuth(),
		chatStorage: NewChatStorage(),
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
	return &Server{
		config:      cfg,
		vectorDB:    codeforgeApp.VectorDB,
		auth:        NewLocalhostAuth(),
		chatStorage: NewChatStorage(),
		app:         codeforgeApp,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Only allow localhost connections for security
				return isLocalhostOrigin(r)
			},
		},
	}
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

	return http.ListenAndServe(addr, router)
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

	// WebSocket for real-time chat (protected via token in URL)
	protected.HandleFunc("/chat/ws/{sessionId}", s.handleChatWebSocket)

	// SSE for metrics and status (protected)
	protected.HandleFunc("/events/metrics", s.handleMetricsSSE)
	protected.HandleFunc("/events/status", s.handleStatusSSE)

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
		w.Header().Set("Access-Control-Allow-Origin", "*")
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
