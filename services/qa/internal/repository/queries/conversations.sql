-- name: InsertConversation :exec
INSERT INTO conversations (
    id,
    external_user_id,
    title,
    status,
    created_at,
    updated_at
) VALUES (
    sqlc.arg(id)::uuid,
    sqlc.arg(external_user_id),
    sqlc.arg(title),
    sqlc.arg(status),
    sqlc.arg(created_at),
    sqlc.arg(updated_at)
);

-- name: GetConversationAccess :one
SELECT
    external_user_id,
    deleted_at
FROM conversations
WHERE id::text = sqlc.arg(id)::text;

-- name: GetConversationForUser :one
SELECT
    c.id::text,
    c.title,
    c.external_user_id,
    c.status,
    c.created_at,
    c.updated_at,
    c.last_message_at,
    (
        SELECT count(*)
        FROM messages m
        WHERE m.conversation_id = c.id
    )::bigint AS message_count,
    COALESCE((
        SELECT left(b.content, 200)
        FROM messages m
        JOIN message_content_blocks b
            ON b.message_id = m.id
            AND b.block_order = 0
        WHERE m.conversation_id = c.id
        ORDER BY m.sequence_no DESC
        LIMIT 1
    ), '') AS last_message_preview
FROM conversations c
WHERE c.id::text = sqlc.arg(id)::text
    AND c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL;

-- name: CountConversationsByStatus :one
SELECT count(*)
FROM conversations
WHERE external_user_id = sqlc.arg(external_user_id)
    AND deleted_at IS NULL
    AND status = sqlc.arg(status);

-- name: ListConversationsUpdatedDesc :many
SELECT
    c.id::text,
    c.title,
    c.external_user_id,
    c.status,
    c.created_at,
    c.updated_at,
    c.last_message_at,
    (
        SELECT count(*)
        FROM messages m
        WHERE m.conversation_id = c.id
    )::bigint AS message_count,
    COALESCE((
        SELECT left(b.content, 200)
        FROM messages m
        JOIN message_content_blocks b
            ON b.message_id = m.id
            AND b.block_order = 0
        WHERE m.conversation_id = c.id
        ORDER BY m.sequence_no DESC
        LIMIT 1
    ), '') AS last_message_preview
FROM conversations c
WHERE c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL
    AND c.status = sqlc.arg(status)
ORDER BY c.updated_at DESC, c.id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListConversationsUpdatedAsc :many
SELECT
    c.id::text,
    c.title,
    c.external_user_id,
    c.status,
    c.created_at,
    c.updated_at,
    c.last_message_at,
    (
        SELECT count(*)
        FROM messages m
        WHERE m.conversation_id = c.id
    )::bigint AS message_count,
    COALESCE((
        SELECT left(b.content, 200)
        FROM messages m
        JOIN message_content_blocks b
            ON b.message_id = m.id
            AND b.block_order = 0
        WHERE m.conversation_id = c.id
        ORDER BY m.sequence_no DESC
        LIMIT 1
    ), '') AS last_message_preview
FROM conversations c
WHERE c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL
    AND c.status = sqlc.arg(status)
ORDER BY c.updated_at ASC, c.id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListConversationsCreatedDesc :many
SELECT
    c.id::text,
    c.title,
    c.external_user_id,
    c.status,
    c.created_at,
    c.updated_at,
    c.last_message_at,
    (
        SELECT count(*)
        FROM messages m
        WHERE m.conversation_id = c.id
    )::bigint AS message_count,
    COALESCE((
        SELECT left(b.content, 200)
        FROM messages m
        JOIN message_content_blocks b
            ON b.message_id = m.id
            AND b.block_order = 0
        WHERE m.conversation_id = c.id
        ORDER BY m.sequence_no DESC
        LIMIT 1
    ), '') AS last_message_preview
FROM conversations c
WHERE c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL
    AND c.status = sqlc.arg(status)
ORDER BY c.created_at DESC, c.id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: ListConversationsCreatedAsc :many
SELECT
    c.id::text,
    c.title,
    c.external_user_id,
    c.status,
    c.created_at,
    c.updated_at,
    c.last_message_at,
    (
        SELECT count(*)
        FROM messages m
        WHERE m.conversation_id = c.id
    )::bigint AS message_count,
    COALESCE((
        SELECT left(b.content, 200)
        FROM messages m
        JOIN message_content_blocks b
            ON b.message_id = m.id
            AND b.block_order = 0
        WHERE m.conversation_id = c.id
        ORDER BY m.sequence_no DESC
        LIMIT 1
    ), '') AS last_message_preview
FROM conversations c
WHERE c.external_user_id = sqlc.arg(external_user_id)
    AND c.deleted_at IS NULL
    AND c.status = sqlc.arg(status)
ORDER BY c.created_at ASC, c.id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);

-- name: UpdateConversation :execrows
UPDATE conversations
SET title = sqlc.arg(title),
    status = sqlc.arg(status),
    updated_at = sqlc.arg(updated_at)
WHERE id::text = sqlc.arg(id)::text
    AND external_user_id = sqlc.arg(external_user_id)
    AND deleted_at IS NULL;

-- name: SoftDeleteConversation :execrows
UPDATE conversations
SET deleted_at = now(),
    updated_at = now()
WHERE id::text = sqlc.arg(id)::text
    AND external_user_id = sqlc.arg(external_user_id)
    AND deleted_at IS NULL;
