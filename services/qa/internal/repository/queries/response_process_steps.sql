-- name: DeleteProcessStepsByRun :exec
DELETE FROM response_process_steps
WHERE response_run_id = sqlc.arg(response_run_id)::uuid;

-- name: InsertProcessStep :exec
INSERT INTO response_process_steps (
    id,
    response_run_id,
    step_order,
    step_type,
    label,
    detail,
    status,
    created_at
) VALUES (
    sqlc.arg(id)::uuid,
    sqlc.arg(response_run_id)::uuid,
    sqlc.arg(step_order),
    sqlc.arg(step_type),
    sqlc.arg(label),
    sqlc.arg(detail),
    sqlc.arg(status),
    sqlc.arg(created_at)
);
