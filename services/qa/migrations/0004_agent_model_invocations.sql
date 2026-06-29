ALTER TABLE response_runs
    ADD COLUMN termination_reason TEXT CHECK (termination_reason IN ('completed', 'max_iterations', 'timeout', 'cancelled', 'tool_error', 'model_error', 'policy_denied'));

UPDATE response_runs
SET termination_reason = CASE
    WHEN status = 'completed' THEN 'completed'
    WHEN status = 'cancelled' THEN 'cancelled'
    WHEN status = 'failed' THEN 'model_error'
    ELSE NULL
END
WHERE termination_reason IS NULL;

ALTER TABLE response_runs
    DROP COLUMN IF EXISTS stop_reason;

CREATE TABLE agent_model_invocations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    response_run_id UUID NOT NULL REFERENCES response_runs(id) ON DELETE CASCADE,
    iteration_no INTEGER NOT NULL CHECK (iteration_no > 0),
    provider TEXT NOT NULL DEFAULT 'ai-gateway',
    profile_id TEXT,
    model_name TEXT NOT NULL,
    finish_reason TEXT CHECK (finish_reason IN ('stop', 'length', 'content_filter', 'tool_calls', 'error')),
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

CREATE INDEX idx_agent_model_invocations_started_at
    ON agent_model_invocations(started_at DESC);