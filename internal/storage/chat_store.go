package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/tursodatabase/go-libsql"
)

// ChatStore interface defines operations for chat persistence
type ChatStore interface {
	// Sessions
	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	ListSessions(ctx context.Context, userID string, limit, offset int) ([]*SessionSummary, error)
	UpdateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, id string) error
	
	// Messages
	SaveMessage(ctx context.Context, message *Message) error
	GetMessages(ctx context.Context, sessionID string, limit, offset int) (*MessageBatch, error)
	GetLatestMessages(ctx context.Context, sessionID string, limit int) ([]Message, error)
	DeleteMessage(ctx context.Context, id string) error
	
	// Context snapshots
	SaveContextSnapshot(ctx context.Context, snapshot *ContextSnapshot) error
	GetLatestContextSnapshot(ctx context.Context, sessionID string) (*ContextSnapshot, error)
	
	// Attachments
	SaveAttachment(ctx context.Context, attachment *Attachment) error
	GetMessageAttachments(ctx context.Context, messageID string) ([]Attachment, error)
	DeleteAttachment(ctx context.Context, id string) error
	
	// Maintenance
	Close() error
	GetStats(ctx context.Context) (map[string]interface{}, error)
}

// SQLiteChatStore implements ChatStore using SQLite/libsql
type SQLiteChatStore struct {
	db *sql.DB
}

// NewDefaultChatStore creates a new chat store using the default user directory
func NewDefaultChatStore() (ChatStore, error) {
	pathManager := NewPathManager()
	dbPath, err := pathManager.GetChatDatabasePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get default chat database path: %w", err)
	}
	return NewChatStore(dbPath)
}

// NewChatStore creates a new chat store using SQLite/libsql
func NewChatStore(dbPath string) (ChatStore, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database using libsql (same as vectordb and permissions)
	db, err := sql.Open("libsql", "file:"+dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &SQLiteChatStore{db: db}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	log.Printf("Chat store initialized: %s", dbPath)
	return store, nil
}

// initSchema loads and executes the schema.sql file
func (s *SQLiteChatStore) initSchema() error {
	// Read schema from embedded file or filesystem
	schemaPath := filepath.Join("internal", "storage", "schema.sql")
	
	// Try to read from filesystem
	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		// Fallback to inline schema if file not found
		schemaBytes = []byte(fallbackSchema)
	}

	// Execute schema
	if _, err := s.db.Exec(string(schemaBytes)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

// CreateSession creates a new chat session
func (s *SQLiteChatStore) CreateSession(ctx context.Context, session *Session) error {
	var metadataJSON string
	if session.Metadata != nil {
		metadataBytes, err := json.Marshal(session.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	query := `INSERT INTO sessions (id, user_id, title, model, created_at, updated_at, message_count, total_tokens, metadata)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := s.db.ExecContext(ctx, query,
		session.ID, session.UserID, session.Title, session.Model,
		session.CreatedAt, session.UpdatedAt, session.MessageCount, session.TotalTokens, metadataJSON)
	
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// GetSession retrieves a session by ID
func (s *SQLiteChatStore) GetSession(ctx context.Context, id string) (*Session, error) {
	query := `SELECT id, user_id, title, model, created_at, updated_at, message_count, total_tokens, metadata
	          FROM sessions WHERE id = ?`
	
	row := s.db.QueryRowContext(ctx, query, id)
	
	var session Session
	var userID, metadataJSON sql.NullString
	
	err := row.Scan(&session.ID, &userID, &session.Title, &session.Model,
		&session.CreatedAt, &session.UpdatedAt, &session.MessageCount, &session.TotalTokens, &metadataJSON)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	session.UserID = userID.String
	
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &session.Metadata); err != nil {
			log.Printf("Warning: failed to unmarshal session metadata: %v", err)
		}
	}

	return &session, nil
}

// ListSessions returns a list of session summaries
func (s *SQLiteChatStore) ListSessions(ctx context.Context, userID string, limit, offset int) ([]*SessionSummary, error) {
	query := `
		SELECT s.id, s.title, s.model, s.updated_at, s.message_count,
		       COALESCE(m.content, '') as last_message
		FROM sessions s
		LEFT JOIN (
			SELECT DISTINCT session_id, 
			       FIRST_VALUE(content) OVER (PARTITION BY session_id ORDER BY created_at DESC) as content
			FROM messages 
			WHERE role = 'user'
		) m ON s.id = m.session_id
		WHERE (? = '' OR s.user_id = ?)
		ORDER BY s.updated_at DESC
		LIMIT ? OFFSET ?`
	
	rows, err := s.db.QueryContext(ctx, query, userID, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*SessionSummary
	for rows.Next() {
		var session SessionSummary
		var lastMessage sql.NullString
		
		err := rows.Scan(&session.ID, &session.Title, &session.Model,
			&session.UpdatedAt, &session.MessageCount, &lastMessage)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}

		session.LastMessage = lastMessage.String
		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// UpdateSession updates an existing session
func (s *SQLiteChatStore) UpdateSession(ctx context.Context, session *Session) error {
	var metadataJSON string
	if session.Metadata != nil {
		metadataBytes, err := json.Marshal(session.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	query := `UPDATE sessions SET title = ?, model = ?, updated_at = ?, metadata = ? WHERE id = ?`
	
	_, err := s.db.ExecContext(ctx, query, session.Title, session.Model, session.UpdatedAt, metadataJSON, session.ID)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

// DeleteSession deletes a session and all its messages
func (s *SQLiteChatStore) DeleteSession(ctx context.Context, id string) error {
	// CASCADE DELETE will handle messages, context_snapshots, and attachments
	query := `DELETE FROM sessions WHERE id = ?`
	
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

// SaveMessage saves a message to the database
func (s *SQLiteChatStore) SaveMessage(ctx context.Context, message *Message) error {
	var metadataJSON string
	if message.Metadata != nil {
		metadataBytes, err := json.Marshal(message.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	query := `INSERT INTO messages (id, session_id, role, content, tokens, created_at, metadata)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`
	
	_, err := s.db.ExecContext(ctx, query,
		message.ID, message.SessionID, message.Role, message.Content,
		message.Tokens, message.CreatedAt, metadataJSON)
	
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// GetMessages retrieves messages for a session with pagination
func (s *SQLiteChatStore) GetMessages(ctx context.Context, sessionID string, limit, offset int) (*MessageBatch, error) {
	// Get total count
	countQuery := `SELECT COUNT(*) FROM messages WHERE session_id = ?`
	var totalCount int
	err := s.db.QueryRowContext(ctx, countQuery, sessionID).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get message count: %w", err)
	}

	// Get messages
	query := `SELECT id, session_id, role, content, tokens, created_at, metadata
	          FROM messages WHERE session_id = ?
	          ORDER BY created_at ASC
	          LIMIT ? OFFSET ?`
	
	rows, err := s.db.QueryContext(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var message Message
		var metadataJSON sql.NullString
		
		err := rows.Scan(&message.ID, &message.SessionID, &message.Role,
			&message.Content, &message.Tokens, &message.CreatedAt, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &message.Metadata); err != nil {
				log.Printf("Warning: failed to unmarshal message metadata: %v", err)
			}
		}

		messages = append(messages, message)
	}

	batch := &MessageBatch{
		Messages:   messages,
		TotalCount: totalCount,
		HasMore:    offset+len(messages) < totalCount,
	}
	
	if batch.HasMore {
		batch.NextOffset = offset + len(messages)
	}

	return batch, nil
}

// GetLatestMessages retrieves the most recent messages for a session
func (s *SQLiteChatStore) GetLatestMessages(ctx context.Context, sessionID string, limit int) ([]Message, error) {
	query := `SELECT id, session_id, role, content, tokens, created_at, metadata
	          FROM messages WHERE session_id = ?
	          ORDER BY created_at DESC
	          LIMIT ?`
	
	rows, err := s.db.QueryContext(ctx, query, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest messages: %w", err)
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var message Message
		var metadataJSON sql.NullString
		
		err := rows.Scan(&message.ID, &message.SessionID, &message.Role,
			&message.Content, &message.Tokens, &message.CreatedAt, &metadataJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &message.Metadata); err != nil {
				log.Printf("Warning: failed to unmarshal message metadata: %v", err)
			}
		}

		messages = append(messages, message)
	}

	// Reverse to get chronological order
	for i := 0; i < len(messages)/2; i++ {
		j := len(messages) - 1 - i
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// DeleteMessage deletes a message
func (s *SQLiteChatStore) DeleteMessage(ctx context.Context, id string) error {
	query := `DELETE FROM messages WHERE id = ?`
	
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}

	return nil
}

// SaveContextSnapshot saves a context processing snapshot
func (s *SQLiteChatStore) SaveContextSnapshot(ctx context.Context, snapshot *ContextSnapshot) error {
	contextBytes, err := json.Marshal(snapshot.ProcessedContext)
	if err != nil {
		return fmt.Errorf("failed to marshal processed context: %w", err)
	}

	stepsBytes, err := json.Marshal(snapshot.ProcessingSteps)
	if err != nil {
		return fmt.Errorf("failed to marshal processing steps: %w", err)
	}

	query := `INSERT INTO context_snapshots 
	          (id, session_id, processed_context, original_tokens, final_tokens, compression_ratio, model_id, processing_steps, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err = s.db.ExecContext(ctx, query,
		snapshot.ID, snapshot.SessionID, string(contextBytes),
		snapshot.OriginalTokens, snapshot.FinalTokens, snapshot.CompressionRatio,
		snapshot.ModelID, string(stepsBytes), snapshot.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to save context snapshot: %w", err)
	}

	return nil
}

// GetLatestContextSnapshot retrieves the most recent context snapshot for a session
func (s *SQLiteChatStore) GetLatestContextSnapshot(ctx context.Context, sessionID string) (*ContextSnapshot, error) {
	query := `SELECT id, session_id, processed_context, original_tokens, final_tokens, compression_ratio, model_id, processing_steps, created_at
	          FROM context_snapshots WHERE session_id = ?
	          ORDER BY created_at DESC LIMIT 1`
	
	row := s.db.QueryRowContext(ctx, query, sessionID)
	
	var snapshot ContextSnapshot
	var contextJSON, stepsJSON string
	
	err := row.Scan(&snapshot.ID, &snapshot.SessionID, &contextJSON,
		&snapshot.OriginalTokens, &snapshot.FinalTokens, &snapshot.CompressionRatio,
		&snapshot.ModelID, &stepsJSON, &snapshot.CreatedAt)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No snapshots found
		}
		return nil, fmt.Errorf("failed to get context snapshot: %w", err)
	}

	// Unmarshal processed context
	if err := json.Unmarshal([]byte(contextJSON), &snapshot.ProcessedContext); err != nil {
		return nil, fmt.Errorf("failed to unmarshal processed context: %w", err)
	}

	// Unmarshal processing steps
	if err := json.Unmarshal([]byte(stepsJSON), &snapshot.ProcessingSteps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal processing steps: %w", err)
	}

	return &snapshot, nil
}

// SaveAttachment saves a file attachment
func (s *SQLiteChatStore) SaveAttachment(ctx context.Context, attachment *Attachment) error {
	query := `INSERT INTO attachments (id, message_id, file_path, file_name, mime_type, file_size, content_hash, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	
	_, err := s.db.ExecContext(ctx, query,
		attachment.ID, attachment.MessageID, attachment.FilePath, attachment.FileName,
		attachment.MimeType, attachment.FileSize, attachment.ContentHash, attachment.CreatedAt)
	
	if err != nil {
		return fmt.Errorf("failed to save attachment: %w", err)
	}

	return nil
}

// GetMessageAttachments retrieves attachments for a message
func (s *SQLiteChatStore) GetMessageAttachments(ctx context.Context, messageID string) ([]Attachment, error) {
	query := `SELECT id, message_id, file_path, file_name, mime_type, file_size, content_hash, created_at
	          FROM attachments WHERE message_id = ?
	          ORDER BY created_at ASC`
	
	rows, err := s.db.QueryContext(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachments: %w", err)
	}
	defer rows.Close()

	var attachments []Attachment
	for rows.Next() {
		var attachment Attachment
		var mimeType, contentHash sql.NullString
		
		err := rows.Scan(&attachment.ID, &attachment.MessageID, &attachment.FilePath,
			&attachment.FileName, &mimeType, &attachment.FileSize, &contentHash, &attachment.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attachment: %w", err)
		}

		attachment.MimeType = mimeType.String
		attachment.ContentHash = contentHash.String
		attachments = append(attachments, attachment)
	}

	return attachments, nil
}

// DeleteAttachment deletes an attachment
func (s *SQLiteChatStore) DeleteAttachment(ctx context.Context, id string) error {
	query := `DELETE FROM attachments WHERE id = ?`
	
	_, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete attachment: %w", err)
	}

	return nil
}

// GetStats returns storage statistics
func (s *SQLiteChatStore) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get session count
	var sessionCount int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sessions").Scan(&sessionCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get session count: %w", err)
	}
	stats["sessions"] = sessionCount

	// Get message count
	var messageCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages").Scan(&messageCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get message count: %w", err)
	}
	stats["messages"] = messageCount

	// Get context snapshot count
	var snapshotCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM context_snapshots").Scan(&snapshotCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot count: %w", err)
	}
	stats["context_snapshots"] = snapshotCount

	// Get attachment count
	var attachmentCount int
	err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM attachments").Scan(&attachmentCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get attachment count: %w", err)
	}
	stats["attachments"] = attachmentCount

	// Get total tokens
	var totalTokens sql.NullInt64
	err = s.db.QueryRowContext(ctx, "SELECT SUM(total_tokens) FROM sessions").Scan(&totalTokens)
	if err != nil {
		return nil, fmt.Errorf("failed to get total tokens: %w", err)
	}
	stats["total_tokens"] = totalTokens.Int64

	return stats, nil
}

// Close closes the database connection
func (s *SQLiteChatStore) Close() error {
	return s.db.Close()
}

// fallbackSchema is used if schema.sql file is not found
const fallbackSchema = `
-- Fallback schema embedded in code
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    title TEXT NOT NULL DEFAULT 'New Chat',
    model TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    message_count INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    metadata TEXT,
    UNIQUE(id)
);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant')),
    content TEXT NOT NULL,
    tokens INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    UNIQUE(id)
);

CREATE TABLE IF NOT EXISTS context_snapshots (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    processed_context TEXT NOT NULL,
    original_tokens INTEGER NOT NULL DEFAULT 0,
    final_tokens INTEGER NOT NULL DEFAULT 0,
    compression_ratio REAL NOT NULL DEFAULT 1.0,
    model_id TEXT NOT NULL,
    processing_steps TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    UNIQUE(id)
);

CREATE TABLE IF NOT EXISTS attachments (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    mime_type TEXT,
    file_size INTEGER NOT NULL DEFAULT 0,
    content_hash TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    UNIQUE(id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_updated_at ON sessions(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_context_snapshots_session_id ON context_snapshots(session_id);
CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);

CREATE TRIGGER IF NOT EXISTS update_session_modified
    AFTER INSERT ON messages
    FOR EACH ROW
BEGIN
    UPDATE sessions 
    SET updated_at = CURRENT_TIMESTAMP,
        message_count = message_count + 1,
        total_tokens = total_tokens + NEW.tokens
    WHERE id = NEW.session_id;
END;
`