-- OpenRouter Smart Two-Table Architecture
-- Efficient model management with automatic cleanup

-- Main models table (lightweight, frequently updated)
CREATE TABLE IF NOT EXISTS openrouter_models (
    model_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_date INTEGER,
    context_length INTEGER,
    provider_name TEXT,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    added_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Detailed metadata table (heavy, cached longer)
CREATE TABLE IF NOT EXISTS openrouter_model_metadata (
    model_id TEXT PRIMARY KEY,
    architecture_json TEXT,     -- Modality, tokenizer, input/output types
    endpoints_json TEXT,        -- All provider endpoints with pricing
    pricing_summary_json TEXT,  -- Aggregated pricing info for quick access
    max_context_length INTEGER, -- Max context across all providers
    supported_modalities TEXT,  -- Comma-separated: text, image, etc.
    provider_count INTEGER,     -- Number of providers supporting this model
    best_price_prompt REAL,     -- Lowest prompt price across providers
    best_price_completion REAL, -- Lowest completion price across providers
    uptime_average REAL,        -- Average uptime across providers
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata_version INTEGER DEFAULT 1,
    FOREIGN KEY (model_id) REFERENCES openrouter_models(model_id) ON DELETE CASCADE
);

-- Automatic cleanup trigger
CREATE TRIGGER IF NOT EXISTS cleanup_openrouter_metadata 
AFTER DELETE ON openrouter_models
FOR EACH ROW
BEGIN
    DELETE FROM openrouter_model_metadata 
    WHERE model_id = OLD.model_id;
END;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_openrouter_models_last_seen ON openrouter_models(last_seen);
CREATE INDEX IF NOT EXISTS idx_openrouter_models_provider ON openrouter_models(provider_name);
CREATE INDEX IF NOT EXISTS idx_openrouter_metadata_updated ON openrouter_model_metadata(last_updated);
CREATE INDEX IF NOT EXISTS idx_openrouter_metadata_price ON openrouter_model_metadata(best_price_prompt, best_price_completion);

-- View for quick model overview with metadata
CREATE VIEW IF NOT EXISTS openrouter_models_overview AS
SELECT 
    m.model_id,
    m.name,
    m.description,
    m.provider_name,
    m.context_length,
    m.last_seen,
    CASE 
        WHEN md.model_id IS NOT NULL THEN 1 
        ELSE 0 
    END as has_metadata,
    md.provider_count,
    md.best_price_prompt,
    md.best_price_completion,
    md.supported_modalities,
    md.uptime_average,
    md.last_updated as metadata_updated
FROM openrouter_models m
LEFT JOIN openrouter_model_metadata md ON m.model_id = md.model_id
ORDER BY m.last_seen DESC;
