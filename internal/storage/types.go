package storage

import (
	"time"
)

// Session represents a chat session
type Session struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id,omitempty"`
	Title        string            `json:"title"`
	Model        string            `json:"model"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	MessageCount int               `json:"message_count"`
	TotalTokens  int               `json:"total_tokens"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Message represents a chat message
type Message struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Tokens    int                    `json:"tokens"`
	CreatedAt time.Time              `json:"created_at"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ContextSnapshot represents a stored context processing result
type ContextSnapshot struct {
	ID               string                 `json:"id"`
	SessionID        string                 `json:"session_id"`
	ProcessedContext map[string]interface{} `json:"processed_context"` // JSON representation
	OriginalTokens   int                    `json:"original_tokens"`
	FinalTokens      int                    `json:"final_tokens"`
	CompressionRatio float64                `json:"compression_ratio"`
	ModelID          string                 `json:"model_id"`
	ProcessingSteps  []string               `json:"processing_steps"`
	CreatedAt        time.Time              `json:"created_at"`
}

// Attachment represents a file attachment
type Attachment struct {
	ID          string    `json:"id"`
	MessageID   string    `json:"message_id"`
	FilePath    string    `json:"file_path"`
	FileName    string    `json:"file_name"`
	MimeType    string    `json:"mime_type,omitempty"`
	FileSize    int64     `json:"file_size"`
	ContentHash string    `json:"content_hash,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// SessionSummary provides a lightweight view of session data
type SessionSummary struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Model        string    `json:"model"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
	LastMessage  string    `json:"last_message,omitempty"`
}

// MessageBatch represents a batch of messages for efficient loading
type MessageBatch struct {
	Messages   []Message `json:"messages"`
	TotalCount int       `json:"total_count"`
	HasMore    bool      `json:"has_more"`
	NextOffset int       `json:"next_offset,omitempty"`
}