-- name: CreateOperationLog :one
INSERT INTO report_operation_logs (
    operator_id, operator_name, operation_type, target_type, target_id,
    request_id, request_source, tool_name, parameter_summary_json,
    operation_result, error_message, metadata_json
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9,
    $10, $11, $12
)
RETURNING id, created_at;

-- name: ListOperationLogs :many
SELECT id, operator_id, operator_name, operation_type, target_type, target_id,
       request_id, request_source, tool_name, parameter_summary_json,
       operation_result, error_message, metadata_json, created_at
FROM report_operation_logs
WHERE
    ($1::text IS NULL OR operation_type = $1)
    AND ($2::text IS NULL OR target_id = $2)
    AND ($3::text IS NULL OR request_source = $3)
ORDER BY created_at DESC
LIMIT $4 OFFSET $5;

-- name: CountOperationLogs :one
SELECT COUNT(*) FROM report_operation_logs
WHERE
    ($1::text IS NULL OR operation_type = $1)
    AND ($2::text IS NULL OR target_id = $2)
    AND ($3::text IS NULL OR request_source = $3);
