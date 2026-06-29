-- name: InsertResponseRun :one
INSERT INTO response_runs (
    conversation_id,
    user_message_id,
    assistant_message_id,
    intent_type,
    route,
    status
) VALUES (
    sqlc.arg(conversation_id)::uuid,
    sqlc.arg(user_message_id)::uuid,
    sqlc.arg(assistant_message_id)::uuid,
    NULLIF(sqlc.arg(intent_type), ''),
    'agent',
    'running'
)
RETURNING
    id::text,
    conversation_id::text,
    user_message_id::text,
    assistant_message_id::text,
    status,
    started_at;

-- name: GetResponseRunForUser :one
SELECT
    rr.id::text,
    rr.conversation_id::text,
    rr.user_message_id::text,
    rr.assistant_message_id::text,
    rr.status,
    COALESCE(rr.current_iteration, 0)::integer,
    COALESCE(rr.max_iterations, 5)::integer,
    rr.termination_reason,
    COALESCE(rr.prompt_tokens, 0) + COALESCE(rr.completion_tokens, 0) + COALESCE(rr.reasoning_tokens, 0),
    COALESCE(rr.latency_ms, 0)::bigint,
    rr.started_at,
    rr.completed_at
FROM response_runs rr
JOIN conversations c ON c.id = rr.conversation_id
WHERE rr.id::text = sqlc.arg(id)::text
    AND c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL;

-- name: GetResponseRunIDByAssistantMessage :one
SELECT rr.id::text
FROM response_runs rr
JOIN conversations c ON c.id = rr.conversation_id
WHERE rr.assistant_message_id = sqlc.arg(assistant_message_id)::uuid
    AND c.external_user_id = sqlc.arg(external_user_id);

-- name: UpdateResponseRunByAssistantMessage :exec
UPDATE response_runs
SET status = sqlc.arg(status),
    termination_reason = CASE
        WHEN sqlc.arg(status) = 'completed' THEN 'completed'
        WHEN sqlc.arg(status) = 'cancelled' THEN 'cancelled'
        ELSE NULL
    END,
    completed_at = CASE
        WHEN sqlc.arg(status) <> 'running' THEN now()
        ELSE NULL
    END,
    latency_ms = CASE
        WHEN sqlc.arg(status) <> 'running' THEN EXTRACT(EPOCH FROM (now() - started_at)) * 1000
        ELSE NULL
    END
WHERE assistant_message_id = sqlc.arg(assistant_message_id)::uuid;

-- name: UpdateResponseRunIteration :exec
UPDATE response_runs
SET current_iteration = GREATEST(current_iteration, sqlc.arg(iteration_no))
WHERE id = sqlc.arg(id)::uuid;

-- name: UpdateResponseRunTermination :exec
UPDATE response_runs rr
SET status = sqlc.arg(status),
    termination_reason = sqlc.arg(termination_reason),
    prompt_tokens = sqlc.arg(prompt_tokens),
    completion_tokens = sqlc.arg(completion_tokens),
    reasoning_tokens = sqlc.arg(reasoning_tokens),
    completed_at = sqlc.arg(completed_at)
FROM conversations c
WHERE rr.id::text = sqlc.arg(id)::text
    AND c.id = rr.conversation_id
    AND c.external_user_id = sqlc.arg(external_user_id);

-- name: CancelResponseRun :one
UPDATE response_runs rr
SET status = 'cancelled',
    termination_reason = 'cancelled',
    completed_at = now(),
    latency_ms = EXTRACT(EPOCH FROM (now() - started_at)) * 1000
FROM conversations c
WHERE rr.id::text = sqlc.arg(id)::text
    AND c.id = rr.conversation_id
    AND c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL
    AND rr.status IN ('running')
RETURNING rr.assistant_message_id::text;

-- name: AuthorizeResponseRunForUser :one
SELECT true
FROM response_runs rr
JOIN conversations c ON c.id = rr.conversation_id
WHERE rr.id::text = sqlc.arg(id)::text
    AND c.external_user_id = sqlc.arg(external_user_id);
