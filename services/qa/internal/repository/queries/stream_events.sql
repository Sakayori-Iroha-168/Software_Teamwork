-- name: DeleteStreamEventsByRun :exec
DELETE FROM response_stream_events
WHERE response_run_id = sqlc.arg(response_run_id)::uuid;

-- name: DeleteToolCallsByRun :exec
DELETE FROM agent_tool_calls
WHERE response_run_id = sqlc.arg(response_run_id)::uuid;

-- name: InsertStreamEvent :exec
INSERT INTO response_stream_events (
    response_run_id,
    event_seq,
    event_type,
    payload,
    created_at
) VALUES (
    sqlc.arg(response_run_id)::uuid,
    sqlc.arg(event_seq),
    sqlc.arg(event_type),
    sqlc.arg(payload),
    sqlc.arg(created_at)
);

-- name: UpsertAgentToolCall :exec
INSERT INTO agent_tool_calls (
    response_run_id,
    iteration_no,
    tool_call_id,
    tool_name,
    status,
    started_at,
    finished_at
) VALUES (
    sqlc.arg(response_run_id)::uuid,
    GREATEST(sqlc.arg(iteration_no), 1),
    sqlc.arg(tool_call_id),
    sqlc.arg(tool_name),
    sqlc.arg(status),
    sqlc.arg(started_at),
    CASE
        WHEN sqlc.arg(status) = 'running' THEN NULL
        ELSE sqlc.arg(started_at)
    END
)
ON CONFLICT (response_run_id, tool_call_id) DO UPDATE SET
    status = EXCLUDED.status,
    finished_at = CASE
        WHEN EXCLUDED.status = 'running' THEN agent_tool_calls.finished_at
        ELSE EXCLUDED.finished_at
    END,
    latency_ms = CASE
        WHEN EXCLUDED.status = 'running' THEN agent_tool_calls.latency_ms
        ELSE EXTRACT(EPOCH FROM (EXCLUDED.finished_at - agent_tool_calls.started_at)) * 1000
    END;

-- name: ListStreamEventsForRun :many
SELECT
    e.event_seq,
    e.event_type,
    e.payload,
    e.created_at
FROM response_stream_events e
JOIN response_runs rr ON rr.id = e.response_run_id
JOIN conversations c ON c.id = rr.conversation_id
WHERE rr.id::text = sqlc.arg(response_run_id)::text
    AND rr.conversation_id::text = sqlc.arg(conversation_id)::text
    AND c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL
    AND e.event_seq > sqlc.arg(after_seq)
    AND e.expires_at > now()
ORDER BY e.event_seq;
