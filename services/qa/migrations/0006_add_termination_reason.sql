-- +goose Up
-- Add termination_reason column to response_runs for agent run tracking
ALTER TABLE response_runs
    ADD COLUMN termination_reason TEXT CHECK (termination_reason IN ('completed', 'max_iterations', 'timeout', 'cancelled', 'tool_error', 'model_error', 'policy_denied'));

-- Migrate existing stop_reason values to termination_reason if any exist
UPDATE response_runs
SET termination_reason = CASE stop_reason
    WHEN 'stop' THEN 'completed'
    WHEN 'max_tokens' THEN 'max_iterations'
    WHEN 'timeout' THEN 'timeout'
    WHEN 'cancelled' THEN 'cancelled'
    WHEN 'error' THEN 'model_error'
    ELSE 'completed'
END
WHERE stop_reason IS NOT NULL AND termination_reason IS NULL;

-- +goose Down
-- Clear termination_reason values (keep column for compatibility)
UPDATE response_runs
SET termination_reason = NULL
WHERE termination_reason IS NOT NULL;
