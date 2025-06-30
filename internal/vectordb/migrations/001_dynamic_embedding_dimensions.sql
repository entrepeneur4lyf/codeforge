-- Migration: Dynamic Embedding Dimensions Support
-- Handles variable embedding dimensions (384, 768, 1536+) based on provider

-- Drop existing vector index (will be recreated with correct dimensions)
DROP INDEX IF EXISTS idx_chunks_embedding;

-- Create new chunks table with truly dynamic embedding storage
-- No default dimensions - they're set based on actual embedding provider
CREATE TABLE IF NOT EXISTS chunks_new (
    id TEXT PRIMARY KEY,
    file_path TEXT NOT NULL,
    content TEXT NOT NULL,
    chunk_type TEXT NOT NULL,
    language TEXT NOT NULL,
    start_line INTEGER NOT NULL,
    end_line INTEGER NOT NULL,
    start_column INTEGER NOT NULL,
    end_column INTEGER NOT NULL,
    metadata TEXT, -- JSON
    hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    embedding BLOB, -- Variable length based on provider (384/768/1536+ floats)
    embedding_dimensions INTEGER, -- Actual dimensions (no default)
    embedding_provider TEXT -- Actual provider (no default)
);

-- Copy existing data (embeddings will need to be regenerated)
INSERT INTO chunks_new (
    id, file_path, content, chunk_type, language,
    start_line, end_line, start_column, end_column,
    metadata, hash, created_at, updated_at
)
SELECT 
    id, file_path, content, chunk_type, language,
    start_line, end_line, start_column, end_column,
    metadata, hash, created_at, updated_at
FROM chunks
WHERE EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name='chunks');

-- Drop old table and rename new one
DROP TABLE IF EXISTS chunks;
ALTER TABLE chunks_new RENAME TO chunks;

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_chunks_file_path ON chunks(file_path);
CREATE INDEX IF NOT EXISTS idx_chunks_type ON chunks(chunk_type);
CREATE INDEX IF NOT EXISTS idx_chunks_language ON chunks(language);
CREATE INDEX IF NOT EXISTS idx_chunks_hash ON chunks(hash);
CREATE INDEX IF NOT EXISTS idx_chunks_provider ON chunks(embedding_provider);
CREATE INDEX IF NOT EXISTS idx_chunks_dimensions ON chunks(embedding_dimensions);

-- Create embedding configuration table (truly dynamic)
CREATE TABLE IF NOT EXISTS embedding_config (
    id INTEGER PRIMARY KEY CHECK (id = 1), -- Singleton table
    provider TEXT, -- Current provider (determined at runtime)
    dimensions INTEGER, -- Current dimensions (determined at runtime)
    model_name TEXT, -- Current model name
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    migration_version INTEGER DEFAULT 1
);

-- No default configuration - will be set when first embedding provider is detected

-- Note: Vector index will be created dynamically based on detected embedding dimensions
-- This happens in the application code after determining the actual embedding provider
