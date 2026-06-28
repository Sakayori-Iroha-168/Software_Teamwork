-- Align schema with official qa-system-design database design
-- References: qa-system-design/db/init/02_schema.sql

BEGIN;

-- ---------------------------------------------------------------------------
-- conversations: ensure all fields match official design
-- ---------------------------------------------------------------------------
ALTER TABLE conversations
    ALTER COLUMN id TYPE UUID USING id::uuid,
    ALTER COLUMN id SET DEFAULT gen_random_uuid();

ALTER TABLE conversations
    ADD COLUMN IF NOT EXISTS title VARCHAR(512);

CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at DESC);

-- ---------------------------------------------------------------------------
-- messages: remove content column (content stored in message_content_blocks)
-- ---------------------------------------------------------------------------
ALTER TABLE messages
    ALTER COLUMN id TYPE UUID USING id::uuid,
    ALTER COLUMN id SET DEFAULT gen_random_uuid(),
    ALTER COLUMN conversation_id TYPE UUID USING conversation_id::uuid,
    ALTER COLUMN sequence_no DROP DEFAULT,
    ALTER COLUMN status SET DEFAULT 'completed',
    DROP COLUMN IF EXISTS content;

ALTER TABLE messages
    ADD CONSTRAINT chk_messages_role CHECK (role IN ('user', 'assistant', 'system')),
    ADD CONSTRAINT chk_messages_status CHECK (status IN ('streaming', 'completed', 'stopped', 'failed'));

DROP INDEX IF EXISTS idx_messages_conversation_sequence;
ALTER TABLE messages
    ADD CONSTRAINT uq_messages_conversation_sequence UNIQUE (conversation_id, sequence_no);

CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at DESC);

-- ---------------------------------------------------------------------------
-- response_runs: align with official schema
-- ---------------------------------------------------------------------------
ALTER TABLE response_runs
    ALTER COLUMN id TYPE UUID USING id::uuid,
    ALTER COLUMN id SET DEFAULT gen_random_uuid(),
    ALTER COLUMN conversation_id TYPE UUID USING conversation_id::uuid,
    ALTER COLUMN user_message_id TYPE UUID USING user_message_id::uuid,
    ALTER COLUMN assistant_message_id TYPE UUID USING assistant_message_id::uuid,
    ALTER COLUMN qa_config_version_id TYPE UUID USING qa_config_version_id::uuid,
    ALTER COLUMN llm_config_version_id TYPE UUID USING llm_config_version_id::uuid,
    DROP COLUMN IF EXISTS message_id,
    DROP COLUMN IF EXISTS updated_at,
    ADD COLUMN IF NOT EXISTS started_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE response_runs SET started_at = created_at WHERE started_at = created_at;

ALTER TABLE response_runs
    DROP COLUMN IF EXISTS created_at;

ALTER TABLE response_runs
    ADD CONSTRAINT chk_response_runs_status CHECK (status IN ('running', 'completed', 'stopped', 'failed'));

CREATE INDEX IF NOT EXISTS idx_response_runs_started_at ON response_runs(started_at DESC);

-- ---------------------------------------------------------------------------
-- response_process_steps: align with official schema
-- ---------------------------------------------------------------------------
ALTER TABLE response_process_steps
    ALTER COLUMN response_run_id TYPE UUID USING response_run_id::uuid;

DROP INDEX IF EXISTS idx_response_process_steps_run_order;
CREATE INDEX IF NOT EXISTS idx_response_process_steps_run_id ON response_process_steps(response_run_id);

-- Fix unique constraint: official uses (response_run_id, step_order)
ALTER TABLE response_process_steps
    DROP CONSTRAINT IF EXISTS response_process_steps_response_run_id_step_type_key;
ALTER TABLE response_process_steps
    ADD CONSTRAINT uq_response_process_steps_run_order UNIQUE (response_run_id, step_order);

-- ---------------------------------------------------------------------------
-- message_content_blocks: align with official schema
-- ---------------------------------------------------------------------------
ALTER TABLE message_content_blocks
    ALTER COLUMN message_id TYPE UUID USING message_id::uuid,
    ADD COLUMN IF NOT EXISTS block_order INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS status VARCHAR(32) NOT NULL DEFAULT 'completed'
        CHECK (status IN ('streaming', 'completed', 'stopped')),
    ADD COLUMN IF NOT EXISTS provider_block_id VARCHAR(128),
    ADD COLUMN IF NOT EXISTS provider_metadata JSONB,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    DROP COLUMN IF EXISTS visibility,
    DROP COLUMN IF EXISTS sort_order;

UPDATE message_content_blocks SET block_order = 0 WHERE block_order = 0;

DROP INDEX IF EXISTS idx_message_content_blocks_message_id;
CREATE INDEX IF NOT EXISTS idx_message_content_blocks_message_id ON message_content_blocks(message_id);

ALTER TABLE message_content_blocks
    ADD CONSTRAINT uq_message_content_blocks_message_order UNIQUE (message_id, block_order);

-- ---------------------------------------------------------------------------
-- response_stream_events: align with official schema
-- ---------------------------------------------------------------------------
ALTER TABLE response_stream_events
    ALTER COLUMN response_run_id TYPE UUID USING response_run_id::uuid,
    ALTER COLUMN expires_at DROP NOT NULL,
    ALTER COLUMN payload SET DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_response_stream_events_run_id ON response_stream_events(response_run_id);
CREATE INDEX IF NOT EXISTS idx_response_stream_events_expires_at ON response_stream_events(expires_at);

ALTER TABLE response_stream_events
    ADD CONSTRAINT uq_response_stream_events_run_seq UNIQUE (response_run_id, event_seq),
    ADD CONSTRAINT chk_response_stream_events_type CHECK (
        event_type IN ('intent', 'step', 'token', 'citation', 'done', 'error')
    );

-- ---------------------------------------------------------------------------
-- citations: align with official schema
-- ---------------------------------------------------------------------------
ALTER TABLE citations
    ADD COLUMN IF NOT EXISTS id_uuid UUID,
    ALTER COLUMN message_id TYPE UUID USING message_id::uuid,
    ALTER COLUMN score TYPE NUMERIC(8,6);

-- Migrate to UUID primary key if needed
-- (Keeping BIGSERIAL id for now, adding UUID column as alternative)

CREATE INDEX IF NOT EXISTS idx_citations_message_id ON citations(message_id);

ALTER TABLE citations
    ADD CONSTRAINT uq_citations_message_no UNIQUE (message_id, citation_no);

-- ---------------------------------------------------------------------------
-- Missing tables: config, audit, retrieval test
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS qa_config_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_no          BIGINT NOT NULL,
    top_k               INTEGER NOT NULL DEFAULT 5,
    similarity_threshold NUMERIC(5, 4) NOT NULL DEFAULT 0.7000,
    use_rerank          BOOLEAN NOT NULL DEFAULT FALSE,
    rerank_threshold    NUMERIC(5, 4),
    rerank_top_n        INTEGER,
    is_active           BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by_user_id  VARCHAR(128),
    CONSTRAINT uq_qa_config_versions_version_no UNIQUE (version_no)
);

CREATE TABLE IF NOT EXISTS qa_config_knowledge_bases (
    config_id           UUID NOT NULL,
    external_kb_id      VARCHAR(128) NOT NULL,
    kb_type             VARCHAR(64) NOT NULL,
    display_name_snapshot VARCHAR(256),
    sort_order          INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (config_id, external_kb_id)
);

CREATE TABLE IF NOT EXISTS llm_config_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version_no          BIGINT NOT NULL,
    provider            VARCHAR(64) NOT NULL,
    api_url             VARCHAR(512) NOT NULL,
    model_name          VARCHAR(128) NOT NULL,
    api_key_secret_ref  VARCHAR(256),
    api_key_last4       VARCHAR(4),
    timeout_seconds     INTEGER NOT NULL DEFAULT 60,
    temperature         NUMERIC(4, 2) NOT NULL DEFAULT 0.70,
    max_tokens          INTEGER NOT NULL DEFAULT 4096,
    is_active           BOOLEAN NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_llm_config_versions_version_no UNIQUE (version_no)
);

CREATE TABLE IF NOT EXISTS admin_audit_logs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_user_id    VARCHAR(128) NOT NULL,
    action              VARCHAR(64) NOT NULL,
    target_type         VARCHAR(64) NOT NULL,
    target_id           VARCHAR(128),
    before_data         JSONB,
    after_data          JSONB,
    request_id          VARCHAR(128),
    ip_address          VARCHAR(64),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_created_at ON admin_audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_admin_audit_logs_external_user_id ON admin_audit_logs(external_user_id);

CREATE TABLE IF NOT EXISTS retrieval_test_runs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    qa_config_version_id    UUID,
    external_user_id        VARCHAR(128) NOT NULL,
    query                   TEXT NOT NULL,
    overrides               JSONB,
    status                  VARCHAR(32) NOT NULL DEFAULT 'running',
    result_count            INTEGER,
    latency_ms              INTEGER,
    error_message           TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at             TIMESTAMPTZ,
    CONSTRAINT chk_retrieval_test_runs_status CHECK (status IN ('running', 'completed', 'failed'))
);

CREATE TABLE IF NOT EXISTS retrieval_test_results (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_run_id         UUID NOT NULL,
    rank_no             INTEGER NOT NULL,
    external_kb_id      VARCHAR(128) NOT NULL,
    external_doc_id     VARCHAR(128) NOT NULL,
    external_chunk_id   VARCHAR(128) NOT NULL,
    doc_name            VARCHAR(512),
    text_snapshot       TEXT,
    vector_score        NUMERIC(8, 6),
    rerank_score        NUMERIC(8, 6),
    metadata            JSONB,
    CONSTRAINT uq_retrieval_test_results_run_rank UNIQUE (test_run_id, rank_no)
);

CREATE INDEX IF NOT EXISTS idx_retrieval_test_runs_created_at ON retrieval_test_runs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_retrieval_test_results_run_id ON retrieval_test_results(test_run_id);

COMMIT;
