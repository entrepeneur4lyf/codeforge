package api

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// LocalhostAuth provides secure authentication for localhost-only connections
type LocalhostAuth struct {
	sessions map[string]*Session
	tokens   map[string]*Token
	mu       sync.RWMutex
}

// Session represents an authenticated session
type Session struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	Salt      string    `json:"-"` // Never expose salt in JSON
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}

// Token represents an API token
type Token struct {
	Hash      string    `json:"hash"`
	SessionID string    `json:"session_id"`
	Salt      string    `json:"-"` // Never expose salt in JSON
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewLocalhostAuth creates a new localhost authentication system
func NewLocalhostAuth() *LocalhostAuth {
	auth := &LocalhostAuth{
		sessions: make(map[string]*Session),
		tokens:   make(map[string]*Token),
	}

	// Start cleanup goroutine
	go auth.cleanupExpiredSessions()

	return auth
}

// isLocalhost checks if the request is from localhost
func (auth *LocalhostAuth) isLocalhost(r *http.Request) bool {
	// Get the real IP address
	ip := auth.getRealIP(r)

	// Parse the IP
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	// Check if it's localhost/loopback
	return parsedIP.IsLoopback() || ip == "127.0.0.1" || ip == "::1"
}

// getRealIP gets the real IP address from the request
func (auth *LocalhostAuth) getRealIP(r *http.Request) string {
	// Check X-Forwarded-For header first
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

// generateSecureToken generates a cryptographically secure token
func (auth *LocalhostAuth) generateSecureToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// hashToken creates a SHA-256 hash of the token with salt
func (auth *LocalhostAuth) hashToken(token, salt string) string {
	// Combine token + salt for stronger security
	combined := token + salt
	hash := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(hash[:])
}

// hashTokenWithSessionSalt creates a hash using session-specific salt
func (auth *LocalhostAuth) hashTokenWithSessionSalt(token, sessionSalt, ipAddress, userAgent string) string {
	// Multi-factor salt: session salt + IP + User-Agent
	// This prevents session hijacking even if token is compromised
	combinedSalt := sessionSalt + ipAddress + userAgent
	return auth.hashToken(token, combinedSalt)
}

// CreateSession creates a new authenticated session for localhost
func (auth *LocalhostAuth) CreateSession(r *http.Request) (*Session, error) {
	// Verify this is a localhost request
	if !auth.isLocalhost(r) {
		return nil, fmt.Errorf("authentication only available for localhost connections")
	}

	// Generate session ID, token, and salt
	sessionID, err := auth.generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session ID: %w", err)
	}

	token, err := auth.generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	salt, err := auth.generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Get client info for salt
	ipAddress := auth.getRealIP(r)
	userAgent := r.UserAgent()

	// Create session
	session := &Session{
		ID:        sessionID,
		Token:     token,
		Salt:      salt,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour sessions
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	// Create token hash with multi-factor salt
	tokenHash := auth.hashTokenWithSessionSalt(token, salt, ipAddress, userAgent)
	tokenEntry := &Token{
		Hash:      tokenHash,
		SessionID: sessionID,
		Salt:      salt,
		CreatedAt: time.Now(),
		ExpiresAt: session.ExpiresAt,
	}

	// Store session and token
	auth.mu.Lock()
	auth.sessions[sessionID] = session
	auth.tokens[tokenHash] = tokenEntry
	auth.mu.Unlock()

	return session, nil
}

// ValidateToken validates an API token and returns the session
func (auth *LocalhostAuth) ValidateToken(tokenString string) (*Session, error) {
	return auth.ValidateTokenWithContext(tokenString, "", "")
}

// ValidateTokenWithContext validates a token with IP and User-Agent validation
func (auth *LocalhostAuth) ValidateTokenWithContext(tokenString, ipAddress, userAgent string) (*Session, error) {
	if tokenString == "" {
		return nil, fmt.Errorf("token required")
	}

	// We need to find the token by trying all stored tokens
	// since we need the salt to recreate the hash
	auth.mu.RLock()
	var foundToken *Token
	var foundSession *Session

	for _, token := range auth.tokens {
		// Get the session to access salt and client info
		if session, exists := auth.sessions[token.SessionID]; exists {
			// If IP/UserAgent provided, validate they match session
			if ipAddress != "" && session.IPAddress != ipAddress {
				continue // Skip this token, IP doesn't match
			}
			if userAgent != "" && session.UserAgent != userAgent {
				continue // Skip this token, User-Agent doesn't match
			}

			// Recreate hash with session salt and client info
			expectedHash := auth.hashTokenWithSessionSalt(
				tokenString,
				token.Salt,
				session.IPAddress,
				session.UserAgent,
			)

			if token.Hash == expectedHash {
				foundToken = token
				foundSession = session
				break
			}
		}
	}
	auth.mu.RUnlock()

	if foundToken == nil {
		return nil, fmt.Errorf("invalid token")
	}

	// Check if token is expired
	if time.Now().After(foundToken.ExpiresAt) {
		// Clean up expired token
		auth.mu.Lock()
		delete(auth.tokens, foundToken.Hash)
		delete(auth.sessions, foundToken.SessionID)
		auth.mu.Unlock()
		return nil, fmt.Errorf("token expired")
	}

	// Return the found session
	return foundSession, nil
}

// hijackerResponseWriter wraps http.ResponseWriter and preserves http.Hijacker interface
type hijackerResponseWriter struct {
	http.ResponseWriter
	hijacker http.Hijacker
}

// Hijack implements http.Hijacker interface
func (hrw *hijackerResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return hrw.hijacker.Hijack()
}

// newHijackerResponseWriter creates a new hijacker-aware response writer
func newHijackerResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	if hijacker, ok := w.(http.Hijacker); ok {
		return &hijackerResponseWriter{
			ResponseWriter: w,
			hijacker:       hijacker,
		}
	}
	return w
}

// AuthMiddleware provides authentication middleware for localhost
func (auth *LocalhostAuth) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check and auth endpoints
		if strings.HasSuffix(r.URL.Path, "/health") ||
			strings.HasSuffix(r.URL.Path, "/auth") ||
			strings.HasSuffix(r.URL.Path, "/login") {
			next.ServeHTTP(w, r)
			return
		}

		// Verify localhost
		if !auth.isLocalhost(r) {
			http.Error(w, "Access denied: localhost only", http.StatusForbidden)
			return
		}

		// Get token from header or query parameter
		token := r.Header.Get("Authorization")
		if token != "" && strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		} else {
			// Fallback to query parameter for WebSocket connections
			token = r.URL.Query().Get("token")
		}

		// Validate token with IP and User-Agent checking
		ipAddress := auth.getRealIP(r)
		userAgent := r.UserAgent()
		session, err := auth.ValidateTokenWithContext(token, ipAddress, userAgent)
		if err != nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		// Add session to request context
		r = r.WithContext(WithSession(r.Context(), session))

		// Use hijacker-aware response writer for WebSocket and SSE endpoints
		hijackerAwareWriter := newHijackerResponseWriter(w)
		next.ServeHTTP(hijackerAwareWriter, r)
	})
}

// cleanupExpiredSessions periodically removes expired sessions
func (auth *LocalhostAuth) cleanupExpiredSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		auth.mu.Lock()
		// Clean up expired sessions
		for sessionID, session := range auth.sessions {
			if now.After(session.ExpiresAt) {
				delete(auth.sessions, sessionID)
			}
		}

		// Clean up expired tokens
		for tokenHash, token := range auth.tokens {
			if now.After(token.ExpiresAt) {
				delete(auth.tokens, tokenHash)
			}
		}
		auth.mu.Unlock()
	}
}

// GetStats returns authentication statistics
func (auth *LocalhostAuth) GetStats() map[string]interface{} {
	auth.mu.RLock()
	defer auth.mu.RUnlock()

	return map[string]interface{}{
		"active_sessions": len(auth.sessions),
		"active_tokens":   len(auth.tokens),
		"security_features": []string{
			"256-bit random tokens",
			"SHA-256 hashing with salt",
			"IP address validation",
			"User-Agent validation",
			"Session-specific salts",
			"24-hour expiration",
			"Localhost-only access",
		},
	}
}
