-- +goose Up
CREATE TABLE IF NOT EXISTS model_profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    purpose TEXT NOT NULL CHECK (purpose IN ('chat', 'embedding', 'rerank')),
    provider TEXT NOT NULL CHECK (provider IN ('openai_compatible', 'siliconflow', 'local_compatible')),
    base_url TEXT NOT NULL,
    model TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT true,
    is_default BOOLEAN NOT NULL DEFAULT false,
    timeout_ms INTEGER NOT NULL CHECK (timeout_ms >= 1000),
    api_key_configured BOOLEAN NOT NULL DEFAULT false,
    supports_streaming BOOLEAN NOT NULL DEFAULT false,
    dimensions INTEGER CHECK (dimensions IS NULL OR dimensions > 0),
    top_n INTEGER CHECK (top_n IS NULL OR top_n > 0),
    default_parameters_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    credential_id TEXT,
    created_by_user_id TEXT,
    updated_by_user_id TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ,
    CHECK (purpose = 'chat' OR supports_streaming = false)
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_model_profiles_purpose_name_active
    ON model_profiles (purpose, name)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uniq_model_profiles_enabled_default_purpose
    ON model_profiles (purpose)
    WHERE enabled = true AND is_default = true AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_model_profiles_purpose_enabled
    ON model_profiles (purpose, enabled, updated_at DESC)
    WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS model_profiles;
