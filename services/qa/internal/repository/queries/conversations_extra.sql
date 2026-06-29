-- name: LockConversationForUser :one
SELECT true
FROM conversations
WHERE id::text = sqlc.arg(id)::text
    AND external_user_id = sqlc.arg(external_user_id)
    AND deleted_at IS NULL
FOR UPDATE;

-- name: TouchConversationActivity :exec
UPDATE conversations
SET updated_at = sqlc.arg(updated_at),
    last_message_at = sqlc.arg(last_message_at)
WHERE id::text = sqlc.arg(id)::text;
