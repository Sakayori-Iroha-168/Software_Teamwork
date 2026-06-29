-- name: InsertModelInvocation :one
INSERT INTO agent_model_invocations (
    response_run_id,
    iteration_no,
    provider,
    profile_id,
    model_name,
    finish_reason,
    status,
    prompt_tokens,
    completion_tokens,
    reasoning_tokens,
    total_tokens,
    latency_ms,
    error_code,
    error_message,
    started_at,
    finished_at
) VALUES (
    sqlc.arg(response_run_id)::uuid,
    sqlc.arg(iteration_no),
    sqlc.arg(provider),
    sqlc.arg(profile_id),
    sqlc.arg(model_name),
    sqlc.narg(finish_reason),
    sqlc.arg(status),
    sqlc.narg(prompt_tokens),
    sqlc.narg(completion_tokens),
    sqlc.narg(reasoning_tokens),
    sqlc.narg(total_tokens),
    sqlc.narg(latency_ms),
    sqlc.narg(error_code),
    sqlc.narg(error_message),
    sqlc.arg(started_at),
    sqlc.narg(finished_at)
)
RETURNING id::text;

-- name: ListModelInvocationsByRun :many
SELECT
    ami.id::text,
    ami.response_run_id::text,
    ami.iteration_no,
    ami.provider,
    ami.profile_id,
    ami.model_name,
    ami.finish_reason,
    ami.status,
    ami.prompt_tokens,
    ami.completion_tokens,
    ami.reasoning_tokens,
    ami.total_tokens,
    ami.latency_ms,
    ami.error_code,
    ami.error_message,
    ami.started_at,
    ami.finished_at
FROM agent_model_invocations ami
JOIN response_runs rr ON rr.id = ami.response_run_id
JOIN conversations c ON c.id = rr.conversation_id
WHERE ami.response_run_id::text = sqlc.arg(response_run_id)::text
    AND c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL
ORDER BY ami.iteration_no ASC, ami.started_at ASC;
