package api

import (
    "context"
    "fmt"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	sessionContextKey contextKey = "session"
)

// WithSession adds a session to the context
func WithSession(ctx context.Context, session *Session) context.Context {
	return context.WithValue(ctx, sessionContextKey, session)
}

// GetSession retrieves a session from the context
func GetSession(ctx context.Context) (*Session, bool) {
	session, ok := ctx.Value(sessionContextKey).(*Session)
	return session, ok
}

// RequireSession gets a session from context or returns an error instead of panicking
func RequireSession(ctx context.Context) (*Session, error) {
    session, ok := GetSession(ctx)
    if !ok || session == nil {
        return nil, fmt.Errorf("session not found in context")
    }
    return session, nil
}
