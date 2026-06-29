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

-- Remove the old stop_reason column (now replaced by termination_reason)
ALTER TABLE response_runs
    DROP COLUMN IF EXISTS stop_reason;

-- +goose Down
-- Re-add stop_reason column for rollback
ALTER TABLE response_runs
    ADD COLUMN stop_reason TEXT;

-- Migrate termination_reason back to stop_reason
UPDATE response_runs
SET stop_reason = CASE termination_reason
    WHEN 'completed' THEN 'stop'
    WHEN 'max_iterations' THEN 'max_tokens'
    WHEN 'timeout' THEN 'timeout'
    WHEN 'cancelled' THEN 'cancelled'
    WHEN 'model_error' THEN 'error'
    ELSE 'stop'
END
WHERE termination_reason IS NOT NULL AND stop_reason IS NULL;

-- Remove termination_reason column
ALTER TABLE response_runs
    DROP COLUMN IF EXISTS termination_reason;