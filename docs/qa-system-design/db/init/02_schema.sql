-- 智能问答系统逻辑模型 → PostgreSQL DDL
-- 约定: UUID 主键、timestamptz、jsonb；外键仅逻辑关联，不建物理 FK 约束

BEGIN;

-- ---------------------------------------------------------------------------
-- 运行配置与管理
-- ---------------------------------------------------------------------------

CREATE TABLE qa_config_versions (
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

CREATE TABLE qa_config_knowledge_bases (
    config_id           UUID NOT NULL,
    external_kb_id      VARCHAR(128) NOT NULL,
    kb_type             VARCHAR(64) NOT NULL,
    display_name_snapshot VARCHAR(256),
    sort_order          INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (config_id, external_kb_id)
);

CREATE TABLE llm_config_versions (
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

CREATE TABLE admin_audit_logs (
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

-- ---------------------------------------------------------------------------
-- 对话与流式问答
-- ---------------------------------------------------------------------------

CREATE TABLE conversations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    external_user_id    VARCHAR(128) NOT NULL,
    title               VARCHAR(512),
    status              VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    CONSTRAINT chk_conversations_status CHECK (status IN ('active', 'archived'))
);

CREATE TABLE messages (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id     UUID NOT NULL,
    role                VARCHAR(16) NOT NULL,
    sequence_no         INTEGER NOT NULL,
    status              VARCHAR(32) NOT NULL DEFAULT 'completed',
    model_name          VARCHAR(128),
    error_code          VARCHAR(64),
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ,
    CONSTRAINT chk_messages_role CHECK (role IN ('user', 'assistant', 'system')),
    CONSTRAINT chk_messages_status CHECK (status IN ('streaming', 'completed', 'stopped', 'failed')),
    CONSTRAINT uq_messages_conversation_sequence UNIQUE (conversation_id, sequence_no)
);

CREATE TABLE response_runs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id         UUID NOT NULL,
    user_message_id         UUID NOT NULL,
    assistant_message_id    UUID,
    qa_config_version_id    UUID,
    llm_config_version_id   UUID,
    request_id              VARCHAR(128),
    intent_type             VARCHAR(64),
    route                   VARCHAR(64),
    confidence              NUMERIC(5, 4),
    status                  VARCHAR(32) NOT NULL DEFAULT 'running',
    stop_reason             VARCHAR(64),
    retry_count             INTEGER NOT NULL DEFAULT 0,
    prompt_tokens           INTEGER,
    completion_tokens       INTEGER,
    reasoning_tokens        INTEGER,
    latency_ms              INTEGER,
    started_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at             TIMESTAMPTZ,
    CONSTRAINT chk_response_runs_status CHECK (status IN ('running', 'completed', 'stopped', 'failed'))
);

CREATE TABLE message_content_blocks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id          UUID NOT NULL,
    block_order         INTEGER NOT NULL,
    block_type          VARCHAR(64) NOT NULL,
    content             TEXT NOT NULL DEFAULT '',
    status              VARCHAR(32) NOT NULL DEFAULT 'completed',
    provider_block_id   VARCHAR(128),
    provider_metadata   JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_message_content_blocks_status CHECK (status IN ('streaming', 'completed', 'stopped')),
    CONSTRAINT uq_message_content_blocks_message_order UNIQUE (message_id, block_order)
);

CREATE TABLE response_process_steps (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    response_run_id     UUID NOT NULL,
    step_order          INTEGER NOT NULL,
    step_type           VARCHAR(64) NOT NULL,
    label               VARCHAR(256) NOT NULL,
    detail              TEXT,
    status              VARCHAR(32) NOT NULL DEFAULT 'running',
    started_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at         TIMESTAMPTZ,
    CONSTRAINT uq_response_process_steps_run_order UNIQUE (response_run_id, step_order)
);

CREATE TABLE response_stream_events (
    id                  BIGSERIAL PRIMARY KEY,
    response_run_id     UUID NOT NULL,
    event_seq           INTEGER NOT NULL,
    event_type          VARCHAR(32) NOT NULL,
    payload             JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at          TIMESTAMPTZ,
    CONSTRAINT uq_response_stream_events_run_seq UNIQUE (response_run_id, event_seq),
    CONSTRAINT chk_response_stream_events_type CHECK (
        event_type IN ('intent', 'step', 'token', 'citation', 'done', 'error')
    )
);

CREATE TABLE citations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id          UUID NOT NULL,
    citation_no         INTEGER NOT NULL,
    char_start          INTEGER,
    char_end            INTEGER,
    external_kb_id      VARCHAR(128) NOT NULL,
    external_doc_id     VARCHAR(128) NOT NULL,
    external_chunk_id   VARCHAR(128) NOT NULL,
    doc_name            VARCHAR(512),
    quote_text          TEXT,
    context             TEXT,
    page_number         INTEGER,
    score               NUMERIC(8, 6),
    metadata            JSONB,
    CONSTRAINT uq_citations_message_no UNIQUE (message_id, citation_no)
);

-- ---------------------------------------------------------------------------
-- 检索体验测试
-- ---------------------------------------------------------------------------

CREATE TABLE retrieval_test_runs (
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

CREATE TABLE retrieval_test_results (
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

-- ---------------------------------------------------------------------------
-- 索引（高频过滤字段）
-- ---------------------------------------------------------------------------

CREATE INDEX idx_conversations_external_user_id ON conversations (external_user_id);
CREATE INDEX idx_conversations_created_at ON conversations (created_at DESC);

CREATE INDEX idx_messages_conversation_id ON messages (conversation_id);
CREATE INDEX idx_messages_created_at ON messages (created_at DESC);

CREATE INDEX idx_response_runs_conversation_id ON response_runs (conversation_id);
CREATE INDEX idx_response_runs_user_message_id ON response_runs (user_message_id);
CREATE INDEX idx_response_runs_started_at ON response_runs (started_at DESC);
CREATE INDEX idx_response_runs_request_id ON response_runs (request_id);

CREATE INDEX idx_message_content_blocks_message_id ON message_content_blocks (message_id);
CREATE INDEX idx_response_process_steps_run_id ON response_process_steps (response_run_id);
CREATE INDEX idx_response_stream_events_run_id ON response_stream_events (response_run_id);
CREATE INDEX idx_response_stream_events_expires_at ON response_stream_events (expires_at);

CREATE INDEX idx_citations_message_id ON citations (message_id);

CREATE INDEX idx_retrieval_test_runs_created_at ON retrieval_test_runs (created_at DESC);
CREATE INDEX idx_retrieval_test_results_run_id ON retrieval_test_results (test_run_id);

CREATE INDEX idx_admin_audit_logs_created_at ON admin_audit_logs (created_at DESC);
CREATE INDEX idx_admin_audit_logs_external_user_id ON admin_audit_logs (external_user_id);

COMMIT;
