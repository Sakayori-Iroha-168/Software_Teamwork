-- Add missing fields to existing tables and create new tables

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS external_user_id TEXT,
    ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'archived')),
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_conversations_external_user_id ON conversations(external_user_id);
CREATE INDEX IF NOT EXISTS idx_conversations_created_at ON conversations(created_at);

ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS sequence_no INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS model_name TEXT,
    ADD COLUMN IF NOT EXISTS error_code TEXT,
    ADD COLUMN IF NOT EXISTS error_message TEXT,
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_conversation_sequence ON messages(conversation_id, sequence_no);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);

ALTER TABLE response_runs
    ADD COLUMN IF NOT EXISTS user_message_id TEXT,
    ADD COLUMN IF NOT EXISTS assistant_message_id TEXT,
    ADD COLUMN IF NOT EXISTS qa_config_version_id TEXT,
    ADD COLUMN IF NOT EXISTS llm_config_version_id TEXT,
    ADD COLUMN IF NOT EXISTS request_id TEXT,
    ADD COLUMN IF NOT EXISTS intent_type TEXT
        CHECK (intent_type IN ('knowledge_qa', 'general_chat', 'document_query', 'system_command')),
    ADD COLUMN IF NOT EXISTS route TEXT,
    ADD COLUMN IF NOT EXISTS confidence DECIMAL(5,4),
    ADD COLUMN IF NOT EXISTS retry_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS prompt_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS completion_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS reasoning_tokens INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS latency_ms INT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_response_runs_user_message_id ON response_runs(user_message_id);
CREATE INDEX IF NOT EXISTS idx_response_runs_request_id ON response_runs(request_id);
CREATE INDEX IF NOT EXISTS idx_response_runs_started_at ON response_runs(created_at);

CREATE TABLE IF NOT EXISTS response_stream_events (
    id               BIGSERIAL PRIMARY KEY,
    response_run_id  TEXT NOT NULL REFERENCES response_runs(id) ON DELETE CASCADE,
    event_seq        INT NOT NULL,
    event_type       TEXT NOT NULL
        CHECK (event_type IN ('intent', 'step', 'token', 'citation', 'done', 'error')),
    payload          JSONB NOT NULL,
    expires_at       TIMESTAMPTZ NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (response_run_id, event_seq)
);

CREATE INDEX IF NOT EXISTS idx_response_stream_events_run_id ON response_stream_events(response_run_id);
CREATE INDEX IF NOT EXISTS idx_response_stream_events_expires_at ON response_stream_events(expires_at);

CREATE TABLE IF NOT EXISTS citations (
    id               BIGSERIAL PRIMARY KEY,
    message_id       TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    citation_no      INT NOT NULL,
    char_start       INT,
    char_end         INT,
    external_kb_id   TEXT,
    external_doc_id  TEXT,
    external_chunk_id TEXT,
    doc_name         TEXT,
    quote_text       TEXT,
    context          TEXT,
    page_number      INT,
    score            DECIMAL(5,4),
    metadata         JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (message_id, citation_no)
);

CREATE INDEX IF NOT EXISTS idx_citations_message_id ON citations(message_id);
