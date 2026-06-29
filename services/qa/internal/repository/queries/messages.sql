-- name: CountMessagesForConversation :one
SELECT count(*)
FROM messages m
JOIN conversations c ON c.id = m.conversation_id
WHERE m.conversation_id::text = sqlc.arg(conversation_id)::text
    AND c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL;

-- name: ListMessagesForConversation :many
SELECT
    m.id::text,
    m.conversation_id::text,
    m.sequence_no,
    m.role,
    COALESCE(b.content, ''),
    COALESCE(m.intent, ''),
    m.status,
    m.created_at,
    m.completed_at,
    (
        SELECT count(*)
        FROM citations ci
        WHERE ci.message_id = m.id
    )::integer AS citation_count
FROM messages m
JOIN conversations c ON c.id = m.conversation_id
LEFT JOIN message_content_blocks b
    ON b.message_id = m.id
    AND b.block_order = 0
WHERE m.conversation_id::text = sqlc.arg(conversation_id)::text
    AND c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL
ORDER BY m.sequence_no
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: GetMaxMessageSequence :one
SELECT COALESCE(max(sequence_no), 0)::integer
FROM messages
WHERE conversation_id::text = sqlc.arg(conversation_id)::text;

-- name: InsertMessage :exec
INSERT INTO messages (
    id,
    conversation_id,
    role,
    sequence_no,
    intent,
    status,
    created_at,
    completed_at
) VALUES (
    sqlc.arg(id)::uuid,
    sqlc.arg(conversation_id)::uuid,
    sqlc.arg(role),
    sqlc.arg(sequence_no),
    NULLIF(sqlc.arg(intent), ''),
    sqlc.arg(status),
    sqlc.arg(created_at),
    CASE
        WHEN sqlc.arg(status) = 'completed' THEN sqlc.arg(created_at)::timestamptz
        ELSE NULL
    END
);

-- name: InsertMessageContentBlock :exec
INSERT INTO message_content_blocks (
    message_id,
    block_order,
    content,
    status,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(message_id)::uuid,
    0,
    sqlc.arg(content),
    sqlc.arg(status),
    sqlc.arg(created_at),
    sqlc.arg(created_at)
);

-- name: UpdateMessageStatus :execrows
UPDATE messages m
SET status = sqlc.arg(status),
    intent = NULLIF(sqlc.arg(intent), ''),
    completed_at = CASE
        WHEN sqlc.arg(status) IN ('completed', 'failed', 'cancelled') THEN now()
        ELSE NULL
    END
FROM conversations c
WHERE m.id = sqlc.arg(id)::uuid
    AND c.id = m.conversation_id
    AND c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL;

-- name: UpdateMessageContentBlock :exec
UPDATE message_content_blocks
SET content = sqlc.arg(content),
    status = sqlc.arg(status),
    updated_at = now()
WHERE message_id = sqlc.arg(message_id)::uuid
    AND block_order = 0;

-- name: CancelAssistantMessage :exec
UPDATE messages
SET status = 'cancelled',
    completed_at = now()
WHERE id = sqlc.arg(id)::uuid;

-- name: CancelAssistantMessageContent :exec
UPDATE message_content_blocks
SET status = 'cancelled',
    updated_at = now()
WHERE message_id = sqlc.arg(message_id)::uuid
    AND block_order = 0;
