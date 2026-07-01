-- +goose Up
ALTER TABLE messages
    ADD COLUMN IF NOT EXISTS content_preview TEXT;

CREATE INDEX idx_response_runs_completed_at ON response_runs(completed_at DESC);
CREATE INDEX idx_messages_created_at_intent ON messages(created_at DESC, intent);

-- +goose Down
DROP INDEX IF EXISTS idx_response_runs_completed_at;
DROP INDEX IF EXISTS idx_messages_created_at_intent;

ALTER TABLE messages
    DROP COLUMN IF EXISTS content_preview;
