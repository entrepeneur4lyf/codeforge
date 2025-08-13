package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// AuthServer is a minimal localhost-only server exposing just auth endpoints
// It is intended to be started temporarily for login and shut down immediately after.
type AuthServer struct {
	auth       *LocalhostAuth
	httpServer *http.Server
}

// NewAuthServer creates a new minimal auth server
func NewAuthServer() *AuthServer {
	return &AuthServer{auth: NewLocalhostAuth()}
}

// Start runs the auth server on the given port
func (s *AuthServer) Start(port int) error {
	router := mux.NewRouter()

	// Health
	router.HandleFunc("/api/v1/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "healthy", "timestamp": time.Now().Unix()})
	}).Methods("GET")

	// Auth endpoints (only localhost allowed inside handlers)
	router.HandleFunc("/api/v1/auth", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Create session
			session, err := s.auth.CreateSession(r)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to create session: %v", err), http.StatusForbidden)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success":    true,
				"token":      session.Token,
				"session_id": session.ID,
				"expires_at": session.ExpiresAt,
				"message":    "Authentication successful for localhost",
			})
			return
		}
		if r.Method == http.MethodGet {
			// Status
			token := r.Header.Get("Authorization")
			if token != "" && len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			} else {
				token = r.URL.Query().Get("token")
			}
			if token == "" {
				_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false})
				return
			}
			sess, err := s.auth.ValidateTokenWithContext(token, s.auth.getRealIP(r), r.UserAgent())
			if err != nil {
				_ = json.NewEncoder(w).Encode(map[string]any{"authenticated": false})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"authenticated": true,
				"session_id":   sess.ID,
				"expires_at":   sess.ExpiresAt,
			})
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}).Methods("GET", "POST")

	s.httpServer = &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: router}
	return s.httpServer.ListenAndServe()
}

// Stop gracefully shuts down the auth server
func (s *AuthServer) Stop(ctx context.Context) error {
	if s.httpServer == nil {
		return nil
	}
	return s.httpServer.Shutdown(ctx)
}
