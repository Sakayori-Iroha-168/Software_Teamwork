-- QA service: conversations, messages, response runs, process steps, content blocks.

CREATE TABLE IF NOT EXISTS conversations (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS messages (
    id               TEXT PRIMARY KEY,
    conversation_id  TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role             TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
    content          TEXT NOT NULL DEFAULT '',
    status           TEXT NOT NULL DEFAULT 'completed'
        CHECK (status IN ('pending', 'streaming', 'completed', 'failed', 'stopped')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);

CREATE TABLE IF NOT EXISTS response_runs (
    id               TEXT PRIMARY KEY,
    message_id       TEXT NOT NULL UNIQUE REFERENCES messages(id) ON DELETE CASCADE,
    conversation_id  TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    status           TEXT NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'completed', 'failed', 'stopped')),
    stop_reason      TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_response_runs_conversation_id ON response_runs(conversation_id);

CREATE TABLE IF NOT EXISTS response_process_steps (
    id               BIGSERIAL PRIMARY KEY,
    response_run_id  TEXT NOT NULL REFERENCES response_runs(id) ON DELETE CASCADE,
    step_order       INT NOT NULL,
    step_type        TEXT NOT NULL
        CHECK (step_type IN ('intent', 'retrieval', 'generation', 'verify')),
    label            TEXT NOT NULL,
    detail           TEXT,
    status           TEXT NOT NULL
        CHECK (status IN ('pending', 'running', 'done', 'failed', 'stopped')),
    started_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at      TIMESTAMPTZ,
    UNIQUE (response_run_id, step_type)
);

CREATE INDEX IF NOT EXISTS idx_response_process_steps_run_order
    ON response_process_steps(response_run_id, step_order);

CREATE TABLE IF NOT EXISTS message_content_blocks (
    id           BIGSERIAL PRIMARY KEY,
    message_id   TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    block_type   TEXT NOT NULL,
    content      TEXT NOT NULL,
    visibility   TEXT NOT NULL DEFAULT 'public'
        CHECK (visibility IN ('public', 'internal')),
    sort_order   INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_message_content_blocks_message_id
    ON message_content_blocks(message_id);
