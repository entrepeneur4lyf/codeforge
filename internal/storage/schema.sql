-- Chat storage schema for CodeForge
-- Based on libsql/SQLite for persistent conversation storage

-- Sessions table: stores chat session metadata
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT,
    title TEXT NOT NULL DEFAULT 'New Chat',
    model TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    message_count INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    metadata TEXT, -- JSON blob for additional session data
    UNIQUE(id)
);

-- Messages table: stores individual chat messages
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant')),
    content TEXT NOT NULL,
    tokens INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT, -- JSON blob for message metadata (attachments, etc.)
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    UNIQUE(id)
);

-- Context snapshots table: stores processed context for sessions
CREATE TABLE IF NOT EXISTS context_snapshots (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    processed_context TEXT NOT NULL, -- JSON blob of ProcessedContext
    original_tokens INTEGER NOT NULL DEFAULT 0,
    final_tokens INTEGER NOT NULL DEFAULT 0,
    compression_ratio REAL NOT NULL DEFAULT 1.0,
    model_id TEXT NOT NULL,
    processing_steps TEXT, -- JSON array of steps applied
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE,
    UNIQUE(id)
);

-- Attachments table: stores file attachments for messages
CREATE TABLE IF NOT EXISTS attachments (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_name TEXT NOT NULL,
    mime_type TEXT,
    file_size INTEGER NOT NULL DEFAULT 0,
    content_hash TEXT, -- SHA256 hash for deduplication
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (message_id) REFERENCES messages(id) ON DELETE CASCADE,
    UNIQUE(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_updated_at ON sessions(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_session_id ON messages(session_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
CREATE INDEX IF NOT EXISTS idx_context_snapshots_session_id ON context_snapshots(session_id);
CREATE INDEX IF NOT EXISTS idx_context_snapshots_created_at ON context_snapshots(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);
CREATE INDEX IF NOT EXISTS idx_attachments_content_hash ON attachments(content_hash);

-- Triggers to update session metadata
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

CREATE TRIGGER IF NOT EXISTS update_session_on_message_delete
    AFTER DELETE ON messages
    FOR EACH ROW
BEGIN
    UPDATE sessions 
    SET updated_at = CURRENT_TIMESTAMP,
        message_count = message_count - 1,
        total_tokens = total_tokens - OLD.tokens
    WHERE id = OLD.session_id;
END;