-- +goose Up
CREATE TABLE IF NOT EXISTS provider_credentials (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL REFERENCES model_profiles(id),
    storage_mode TEXT NOT NULL CHECK (storage_mode IN ('secret_ref', 'encrypted_column')),
    secret_ref TEXT,
    ciphertext BYTEA,
    encryption_key_version TEXT,
    fingerprint_sha256 TEXT NOT NULL,
    key_last4 TEXT,
    status TEXT NOT NULL CHECK (status IN ('active', 'rotated', 'disabled', 'deleted')),
    created_by_user_id TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    rotated_at TIMESTAMPTZ,
    disabled_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,
    CHECK (
        (storage_mode = 'secret_ref' AND secret_ref IS NOT NULL AND ciphertext IS NULL)
        OR
        (storage_mode = 'encrypted_column' AND ciphertext IS NOT NULL AND secret_ref IS NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uniq_provider_credentials_active_profile
    ON provider_credentials (profile_id)
    WHERE status = 'active' AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_provider_credentials_profile_status
    ON provider_credentials (profile_id, status, created_at DESC);

ALTER TABLE model_profiles
    ADD CONSTRAINT fk_model_profiles_credential
    FOREIGN KEY (credential_id) REFERENCES provider_credentials(id);

-- +goose Down
ALTER TABLE model_profiles DROP CONSTRAINT IF EXISTS fk_model_profiles_credential;
DROP TABLE IF EXISTS provider_credentials;
