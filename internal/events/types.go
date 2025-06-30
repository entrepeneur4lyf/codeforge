package events

import (
	"context"
	"time"
)

// EventType identifies the type of event
type EventType string

// Core event types
const (
	// Chat events
	ChatMessageReceived EventType = "chat.message.received"
	ChatMessageSent     EventType = "chat.message.sent"
	ChatTypingStart     EventType = "chat.typing.start"
	ChatTypingStop      EventType = "chat.typing.stop"
	ChatSessionCreated  EventType = "chat.session.created"
	ChatSessionUpdated  EventType = "chat.session.updated"

	// Context events
	ContextUpdated     EventType = "context.updated"
	ContextFileChanged EventType = "context.file.changed"
	ContextOptimized   EventType = "context.optimized"
	ContextSummarized  EventType = "context.summarized"

	// Permission events
	PermissionRequested EventType = "permission.requested"
	PermissionGranted   EventType = "permission.granted"
	PermissionDenied    EventType = "permission.denied"
	PermissionRevoked   EventType = "permission.revoked"

	// System events
	SystemStarted     EventType = "system.started"
	SystemShutdown    EventType = "system.shutdown"
	SystemError       EventType = "system.error"
	SystemHealthCheck EventType = "system.health.check"

	// Notification events
	NotificationInfo    EventType = "notification.info"
	NotificationSuccess EventType = "notification.success"
	NotificationWarning EventType = "notification.warning"
	NotificationError   EventType = "notification.error"

	// File events
	FileCreated  EventType = "file.created"
	FileUpdated  EventType = "file.updated"
	FileDeleted  EventType = "file.deleted"
	FileWatched  EventType = "file.watched"

	// Vector database events
	VectorIndexUpdated EventType = "vector.index.updated"
	VectorSearched     EventType = "vector.searched"
	VectorCacheHit     EventType = "vector.cache.hit"
	VectorCacheMiss    EventType = "vector.cache.miss"

	// MCP events
	MCPToolCalled    EventType = "mcp.tool.called"
	MCPToolCompleted EventType = "mcp.tool.completed"
	MCPToolFailed    EventType = "mcp.tool.failed"
)

// Event represents a generic event in the system
type Event[T any] struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Payload   T                      `json:"payload"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	SessionID string                 `json:"session_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
}

// Publisher defines the interface for publishing events
type Publisher[T any] interface {
	Publish(eventType EventType, payload T, opts ...PublishOption)
}

// Subscriber defines the interface for subscribing to events
type Subscriber[T any] interface {
	Subscribe(ctx context.Context, filter ...EventFilter) <-chan Event[T]
}

// EventFilter defines a filter function for events
type EventFilter func(Event[any]) bool

// PublishOption defines options for publishing events
type PublishOption func(*PublishOptions)

// PublishOptions contains options for publishing events
type PublishOptions struct {
	SessionID string
	UserID    string
	Metadata  map[string]interface{}
	Persist   bool
	TTL       time.Duration
}

// WithSessionID sets the session ID for the event
func WithSessionID(sessionID string) PublishOption {
	return func(opts *PublishOptions) {
		opts.SessionID = sessionID
	}
}

// WithUserID sets the user ID for the event
func WithUserID(userID string) PublishOption {
	return func(opts *PublishOptions) {
		opts.UserID = userID
	}
}

// WithMetadata sets metadata for the event
func WithMetadata(metadata map[string]interface{}) PublishOption {
	return func(opts *PublishOptions) {
		opts.Metadata = metadata
	}
}

// WithPersistence enables event persistence
func WithPersistence(ttl time.Duration) PublishOption {
	return func(opts *PublishOptions) {
		opts.Persist = true
		opts.TTL = ttl
	}
}

// Common event payload types

// ChatEventPayload represents chat-related event data
type ChatEventPayload struct {
	MessageID string                 `json:"message_id"`
	SessionID string                 `json:"session_id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Model     string                 `json:"model,omitempty"`
	Provider  string                 `json:"provider,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ContextEventPayload represents context-related event data
type ContextEventPayload struct {
	SessionID     string                 `json:"session_id"`
	ContextType   string                 `json:"context_type"`
	FilePath      string                 `json:"file_path,omitempty"`
	TokenCount    int                    `json:"token_count,omitempty"`
	ChangeType    string                 `json:"change_type,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// PermissionEventPayload represents permission-related event data
type PermissionEventPayload struct {
	SessionID    string                 `json:"session_id"`
	ToolName     string                 `json:"tool_name"`
	Action       string                 `json:"action"`
	Resource     string                 `json:"resource,omitempty"`
	Granted      bool                   `json:"granted"`
	Reason       string                 `json:"reason,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationEventPayload represents notification event data
type NotificationEventPayload struct {
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Level       string                 `json:"level"` // info, success, warning, error
	Duration    time.Duration          `json:"duration,omitempty"`
	Dismissible bool                   `json:"dismissible"`
	Actions     []NotificationAction   `json:"actions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationAction represents an action that can be taken on a notification
type NotificationAction struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Style string `json:"style"` // primary, secondary, danger
}

// SystemEventPayload represents system-related event data
type SystemEventPayload struct {
	Component string                 `json:"component"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// FileEventPayload represents file-related event data
type FileEventPayload struct {
	Path      string                 `json:"path"`
	Operation string                 `json:"operation"` // create, update, delete, watch
	Size      int64                  `json:"size,omitempty"`
	MimeType  string                 `json:"mime_type,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// VectorEventPayload represents vector database event data
type VectorEventPayload struct {
	Operation    string                 `json:"operation"`
	DocumentID   string                 `json:"document_id,omitempty"`
	Query        string                 `json:"query,omitempty"`
	ResultCount  int                    `json:"result_count,omitempty"`
	Duration     time.Duration          `json:"duration,omitempty"`
	CacheHit     bool                   `json:"cache_hit,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// MCPEventPayload represents MCP tool event data
type MCPEventPayload struct {
	ToolName    string                 `json:"tool_name"`
	SessionID   string                 `json:"session_id"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Event filter helpers

// FilterByType creates a filter for specific event types
func FilterByType(eventTypes ...EventType) EventFilter {
	typeMap := make(map[EventType]bool)
	for _, t := range eventTypes {
		typeMap[t] = true
	}
	return func(event Event[any]) bool {
		return typeMap[event.Type]
	}
}

// FilterBySessionID creates a filter for specific session ID
func FilterBySessionID(sessionID string) EventFilter {
	return func(event Event[any]) bool {
		return event.SessionID == sessionID
	}
}

// FilterByUserID creates a filter for specific user ID
func FilterByUserID(userID string) EventFilter {
	return func(event Event[any]) bool {
		return event.UserID == userID
	}
}

// CombineFilters combines multiple filters with AND logic
func CombineFilters(filters ...EventFilter) EventFilter {
	return func(event Event[any]) bool {
		for _, filter := range filters {
			if !filter(event) {
				return false
			}
		}
		return true
	}
}
