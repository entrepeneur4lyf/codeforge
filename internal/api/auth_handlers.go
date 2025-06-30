package api

import (
	"encoding/json"
	"net/http"
	"time"
)

// LoginRequest represents a login request
type LoginRequest struct {
	DeviceName string `json:"device_name,omitempty"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Success   bool      `json:"success"`
	Token     string    `json:"token"`
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message"`
}

// AuthStatusResponse represents authentication status
type AuthStatusResponse struct {
	Authenticated bool      `json:"authenticated"`
	SessionID     string    `json:"session_id,omitempty"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	IPAddress     string    `json:"ip_address,omitempty"`
	CreatedAt     time.Time `json:"created_at,omitempty"`
}

// handleAuth handles authentication endpoints
func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		s.handleLogin(w, r)
	case "GET":
		s.handleAuthStatus(w, r)
	case "DELETE":
		s.handleLogout(w, r)
	default:
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleLogin creates a new authentication session
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Verify this is localhost
	if !s.auth.isLocalhost(r) {
		s.writeError(w, "Authentication only available for localhost", http.StatusForbidden)
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Allow empty body for simple login
		req = LoginRequest{DeviceName: "Unknown Device"}
	}

	// Create session
	session, err := s.auth.CreateSession(r)
	if err != nil {
		s.writeError(w, "Failed to create session: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := LoginResponse{
		Success:   true,
		Token:     session.Token,
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt,
		Message:   "Authentication successful for localhost",
	}

	s.writeJSON(w, response)
}

// handleAuthStatus returns current authentication status
func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	// Get token from header or query
	token := r.Header.Get("Authorization")
	if token != "" && len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	} else {
		token = r.URL.Query().Get("token")
	}

	if token == "" {
		s.writeJSON(w, AuthStatusResponse{Authenticated: false})
		return
	}

	// Validate token with enhanced security
	ipAddress := s.auth.getRealIP(r)
	userAgent := r.UserAgent()
	session, err := s.auth.ValidateTokenWithContext(token, ipAddress, userAgent)
	if err != nil {
		s.writeJSON(w, AuthStatusResponse{Authenticated: false})
		return
	}

	response := AuthStatusResponse{
		Authenticated: true,
		SessionID:     session.ID,
		ExpiresAt:     session.ExpiresAt,
		IPAddress:     session.IPAddress,
		CreatedAt:     session.CreatedAt,
	}

	s.writeJSON(w, response)
}

// handleLogout invalidates the current session
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Get token from header
	token := r.Header.Get("Authorization")
	if token != "" && len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	}

	if token == "" {
		s.writeError(w, "No token provided", http.StatusBadRequest)
		return
	}

	// Validate and remove session
	ipAddress := s.auth.getRealIP(r)
	userAgent := r.UserAgent()
	session, err := s.auth.ValidateTokenWithContext(token, ipAddress, userAgent)
	if err != nil {
		s.writeError(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Remove session and all associated tokens
	s.auth.mu.Lock()
	delete(s.auth.sessions, session.ID)

	// Find and remove all tokens for this session
	for tokenHash, tokenEntry := range s.auth.tokens {
		if tokenEntry.SessionID == session.ID {
			delete(s.auth.tokens, tokenHash)
		}
	}
	s.auth.mu.Unlock()

	response := map[string]interface{}{
		"success": true,
		"message": "Logged out successfully",
	}

	s.writeJSON(w, response)
}

// handleAuthInfo provides information about the authentication system
func (s *Server) handleAuthInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"type":        "localhost-only",
		"description": "Secure authentication for localhost connections without TLS",
		"features": []string{
			"256-bit cryptographically secure tokens",
			"SHA-256 hashing with session-specific salts",
			"IP address validation",
			"User-Agent validation",
			"Session hijacking prevention",
			"Automatic token expiration",
			"No TLS required for localhost",
		},
		"security": map[string]interface{}{
			"token_length":         64, // 32 bytes = 64 hex chars
			"salt_length":          64, // 32 bytes = 64 hex chars
			"hash_algorithm":       "SHA-256",
			"session_timeout":      "24 hours",
			"ip_validation":        true,
			"user_agent_check":     true,
			"session_binding":      true,
			"localhost_only":       true,
			"hijacking_protection": true,
		},
		"stats": s.auth.GetStats(),
	}

	s.writeJSON(w, info)
}
