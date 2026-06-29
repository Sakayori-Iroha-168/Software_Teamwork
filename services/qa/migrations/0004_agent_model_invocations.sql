-- +goose Up
CREATE TABLE agent_model_invocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    response_run_id UUID NOT NULL REFERENCES response_runs(id) ON DELETE CASCADE,
    iteration_no INTEGER NOT NULL CHECK (iteration_no > 0),
    provider TEXT NOT NULL DEFAULT 'ai-gateway',
    profile_id TEXT NOT NULL,
    model_name TEXT NOT NULL,
    finish_reason TEXT,
    status TEXT NOT NULL CHECK (status IN ('running', 'completed', 'failed', 'cancelled')),
    prompt_tokens INTEGER,
    completion_tokens INTEGER,
    reasoning_tokens INTEGER,
    total_tokens INTEGER,
    latency_ms BIGINT,
    error_code TEXT,
    error_message TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    UNIQUE (response_run_id, iteration_no)
);

CREATE INDEX idx_agent_model_invocations_run
    ON agent_model_invocations(response_run_id, iteration_no);

ALTER TABLE response_runs
    ADD COLUMN termination_reason TEXT CHECK (termination_reason IN ('completed', 'max_iterations', 'timeout', 'cancelled', 'model_error'));

ALTER TABLE response_runs
    DROP COLUMN IF EXISTS stop_reason;

-- +goose Down
ALTER TABLE response_runs
    ADD COLUMN stop_reason TEXT;

ALTER TABLE response_runs
    DROP COLUMN IF EXISTS termination_reason;

DROP TABLE IF EXISTS agent_model_invocations;
