-- +goose Up
CREATE TABLE session_attachments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    external_user_id TEXT NOT NULL,
    file_ref TEXT NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    status TEXT NOT NULL CHECK (status IN ('uploaded', 'parsing', 'ready', 'failed', 'deleting', 'deleted')),
    error_code TEXT,
    error_summary TEXT,
    page_count INTEGER NOT NULL DEFAULT 0 CHECK (page_count >= 0),
    chunk_count INTEGER NOT NULL DEFAULT 0 CHECK (chunk_count >= 0),
    expires_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ,
    file_delete_requested_at TIMESTAMPTZ,
    file_delete_error_summary TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE session_attachment_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    attachment_id UUID NOT NULL REFERENCES session_attachments(id) ON DELETE CASCADE,
    chunk_order INTEGER NOT NULL CHECK (chunk_order > 0),
    page_number INTEGER CHECK (page_number IS NULL OR page_number > 0),
    section_path TEXT,
    body TEXT NOT NULL,
    preview TEXT NOT NULL,
    token_count INTEGER NOT NULL DEFAULT 0 CHECK (token_count >= 0),
    char_count INTEGER NOT NULL DEFAULT 0 CHECK (char_count >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (attachment_id, chunk_order)
);

CREATE TABLE message_attachments (
    message_id UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    attachment_id UUID NOT NULL REFERENCES session_attachments(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (message_id, attachment_id)
);

ALTER TABLE citations
    ADD COLUMN IF NOT EXISTS attachment_id UUID REFERENCES session_attachments(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS attachment_chunk_id UUID REFERENCES session_attachment_chunks(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS attachment_filename TEXT,
    ADD COLUMN IF NOT EXISTS attachment_chunk_preview TEXT;

CREATE INDEX idx_session_attachments_conversation
    ON session_attachments(conversation_id, external_user_id, deleted_at, created_at DESC);
CREATE INDEX idx_session_attachments_status
    ON session_attachments(status, updated_at);
CREATE INDEX idx_session_attachments_expires_at
    ON session_attachments(expires_at) WHERE deleted_at IS NULL;
CREATE INDEX idx_session_attachment_chunks_attachment
    ON session_attachment_chunks(attachment_id, chunk_order);
CREATE INDEX idx_session_attachment_chunks_body_fts
    ON session_attachment_chunks USING gin(to_tsvector('simple', body));
CREATE INDEX idx_message_attachments_attachment
    ON message_attachments(attachment_id);
CREATE INDEX idx_citations_attachment_id
    ON citations(attachment_id);

UPDATE qa_config_versions
SET enabled_tool_names = enabled_tool_names || '["search_session_attachments"]'::jsonb
WHERE version_no = 1
  AND created_by_user_id = 'system'
  AND enabled_tool_names ? 'search_knowledge'
  AND NOT (enabled_tool_names ? 'search_session_attachments');

-- +goose Down
UPDATE qa_config_versions
SET enabled_tool_names = enabled_tool_names - 'search_session_attachments'
WHERE version_no = 1
  AND created_by_user_id = 'system'
  AND enabled_tool_names ? 'search_session_attachments';

DROP INDEX IF EXISTS idx_citations_attachment_id;
DROP INDEX IF EXISTS idx_message_attachments_attachment;
DROP INDEX IF EXISTS idx_session_attachment_chunks_body_fts;
DROP INDEX IF EXISTS idx_session_attachment_chunks_attachment;
DROP INDEX IF EXISTS idx_session_attachments_expires_at;
DROP INDEX IF EXISTS idx_session_attachments_status;
DROP INDEX IF EXISTS idx_session_attachments_conversation;

ALTER TABLE citations
    DROP COLUMN IF EXISTS attachment_chunk_preview,
    DROP COLUMN IF EXISTS attachment_filename,
    DROP COLUMN IF EXISTS attachment_chunk_id,
    DROP COLUMN IF EXISTS attachment_id;

DROP TABLE IF EXISTS message_attachments;
DROP TABLE IF EXISTS session_attachment_chunks;
DROP TABLE IF EXISTS session_attachments;
