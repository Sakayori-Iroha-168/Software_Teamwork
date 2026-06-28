CREATE TABLE conversations (
    id TEXT PRIMARY KEY,
    external_user_id TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at TIMESTAMPTZ,
    CONSTRAINT chk_conversations_status CHECK (status IN ('active', 'archived'))
);

CREATE INDEX idx_conversations_external_user_status_updated
    ON conversations (external_user_id, status, updated_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX idx_conversations_created_at
    ON conversations (created_at DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    sequence_no INTEGER NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    error_code TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    CONSTRAINT chk_messages_role CHECK (role IN ('user', 'assistant', 'system')),
    CONSTRAINT chk_messages_status CHECK (status IN ('streaming', 'completed', 'stopped', 'failed')),
    CONSTRAINT uniq_messages_conversation_sequence UNIQUE (conversation_id, sequence_no)
);

CREATE INDEX idx_messages_conversation_sequence
    ON messages (conversation_id, sequence_no);

CREATE INDEX idx_messages_created_at
    ON messages (created_at DESC);

CREATE TABLE message_content_blocks (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL,
    block_order INTEGER NOT NULL,
    block_type TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'completed',
    provider_block_id TEXT,
    provider_metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_message_content_blocks_status CHECK (status IN ('streaming', 'completed', 'stopped')),
    CONSTRAINT uniq_message_content_blocks_message_order UNIQUE (message_id, block_order)
);

CREATE INDEX idx_message_content_blocks_message_order
    ON message_content_blocks (message_id, block_order);
