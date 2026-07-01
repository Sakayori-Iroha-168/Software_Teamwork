package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

const attachmentSelect = `
SELECT
    a.id::text,
    a.conversation_id::text,
    a.external_user_id,
    a.file_ref,
    a.filename,
    a.content_type,
    a.size_bytes,
    a.status,
    COALESCE(a.error_code, ''),
    COALESCE(a.error_summary, ''),
    a.page_count,
    a.chunk_count,
    a.expires_at,
    a.deleted_at,
    a.created_at,
    a.updated_at
FROM session_attachments a`

func (r *Postgres) CreateAttachment(ctx context.Context, attachment service.SessionAttachment) (service.SessionAttachment, error) {
	err := r.pool.QueryRow(ctx, `
INSERT INTO session_attachments (
    id, conversation_id, external_user_id, file_ref, filename, content_type,
    size_bytes, status, expires_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9, $10, $11
) RETURNING id::text, conversation_id::text, external_user_id, file_ref, filename,
    content_type, size_bytes, status, COALESCE(error_code, ''), COALESCE(error_summary, ''),
    page_count, chunk_count, expires_at, deleted_at, created_at, updated_at`,
		attachment.ID, attachment.SessionID, attachment.OwnerUserID, attachment.FileRef,
		attachment.Filename, attachment.ContentType, attachment.SizeBytes,
		attachment.Status, attachment.ExpiresAt, attachment.CreatedAt, attachment.UpdatedAt,
	).Scan(
		&attachment.ID, &attachment.SessionID, &attachment.OwnerUserID, &attachment.FileRef,
		&attachment.Filename, &attachment.ContentType, &attachment.SizeBytes,
		&attachment.Status, &attachment.ErrorCode, &attachment.ErrorSummary,
		&attachment.PageCount, &attachment.ChunkCount, &attachment.ExpiresAt,
		&attachment.DeletedAt, &attachment.CreatedAt, &attachment.UpdatedAt,
	)
	if err != nil {
		return service.SessionAttachment{}, fmt.Errorf("create session attachment: %w", err)
	}
	return attachment, nil
}

func (r *Postgres) CountLiveAttachments(ctx context.Context, userID, sessionID string) (int, error) {
	if _, err := r.GetConversation(ctx, userID, sessionID); err != nil {
		return 0, err
	}
	var count int
	err := r.pool.QueryRow(ctx, `
SELECT count(*)
FROM session_attachments
WHERE conversation_id::text=$1
    AND external_user_id=$2
    AND deleted_at IS NULL
    AND status <> 'deleted'`, sessionID, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count session attachments: %w", err)
	}
	return count, nil
}

func (r *Postgres) ListAttachments(ctx context.Context, userID, sessionID string) ([]service.SessionAttachment, error) {
	rows, err := r.pool.Query(ctx, attachmentSelect+`
JOIN conversations c ON c.id=a.conversation_id
WHERE a.conversation_id::text=$1
    AND a.external_user_id=$2
    AND c.external_user_id=$2
    AND c.deleted_at IS NULL
    AND a.deleted_at IS NULL
    AND a.status <> 'deleted'
ORDER BY a.created_at DESC`, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("list session attachments: %w", err)
	}
	defer rows.Close()
	items, err := scanAttachments(rows)
	if err != nil {
		return nil, err
	}
	rows.Close()
	if len(items) == 0 {
		if _, err := r.GetConversation(ctx, userID, sessionID); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (r *Postgres) GetAttachment(ctx context.Context, userID, sessionID, attachmentID string) (service.SessionAttachment, error) {
	item, err := scanAttachment(r.pool.QueryRow(ctx, attachmentSelect+`
JOIN conversations c ON c.id=a.conversation_id
WHERE a.id::text=$1
    AND a.conversation_id::text=$2
    AND a.external_user_id=$3
    AND c.external_user_id=$3
    AND c.deleted_at IS NULL
    AND a.deleted_at IS NULL
    AND a.status <> 'deleted'`, attachmentID, sessionID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		if _, accessErr := r.GetConversation(ctx, userID, sessionID); accessErr != nil {
			return service.SessionAttachment{}, accessErr
		}
		return service.SessionAttachment{}, service.NewError(service.CodeNotFound, "attachment not found", err)
	}
	return item, err
}

func (r *Postgres) MarkAttachmentParsing(ctx context.Context, attachmentID string) (service.SessionAttachment, error) {
	item, err := scanAttachment(r.pool.QueryRow(ctx, `
UPDATE session_attachments
SET status='parsing', error_code=NULL, error_summary=NULL, updated_at=now()
WHERE id::text=$1
    AND deleted_at IS NULL
    AND status IN ('uploaded', 'failed')
RETURNING id::text, conversation_id::text, external_user_id, file_ref, filename,
    content_type, size_bytes, status, COALESCE(error_code, ''), COALESCE(error_summary, ''),
    page_count, chunk_count, expires_at, deleted_at, created_at, updated_at`, attachmentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return service.SessionAttachment{}, service.NewError(service.CodeNotFound, "attachment not found", err)
	}
	return item, err
}

func (r *Postgres) MarkAttachmentFailed(ctx context.Context, attachmentID, code, summary string) error {
	result, err := r.pool.Exec(ctx, `
UPDATE session_attachments
SET status='failed',
    error_code=NULLIF($2, ''),
    error_summary=NULLIF($3, ''),
    updated_at=now()
WHERE id::text=$1
    AND deleted_at IS NULL`, attachmentID, safeErrorCode(code), safeErrorSummary(summary))
	if err != nil {
		return fmt.Errorf("mark attachment failed: %w", err)
	}
	if result.RowsAffected() == 0 {
		return service.NewError(service.CodeNotFound, "attachment not found", nil)
	}
	return nil
}

func (r *Postgres) ReplaceAttachmentChunks(ctx context.Context, attachmentID string, chunks []service.AttachmentChunk, pageCount int) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace attachment chunks: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM session_attachment_chunks WHERE attachment_id::text=$1`, attachmentID); err != nil {
		return fmt.Errorf("delete attachment chunks: %w", err)
	}
	for _, chunk := range chunks {
		_, err := tx.Exec(ctx, `
INSERT INTO session_attachment_chunks (
    id, attachment_id, chunk_order, page_number, section_path, body,
    preview, token_count, char_count, created_at
) VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, $9, $10)`,
			chunk.ID, attachmentID, chunk.ChunkOrder, nullableChunkPage(chunk.PageNumber),
			chunk.SectionPath, chunk.Body, chunk.Preview, chunk.TokenCount, chunk.CharCount, chunk.CreatedAt)
		if err != nil {
			return fmt.Errorf("insert attachment chunk: %w", err)
		}
	}
	result, err := tx.Exec(ctx, `
UPDATE session_attachments
SET status='ready',
    error_code=NULL,
    error_summary=NULL,
    page_count=$2,
    chunk_count=$3,
    updated_at=now()
WHERE id::text=$1
    AND deleted_at IS NULL`, attachmentID, pageCount, len(chunks))
	if err != nil {
		return fmt.Errorf("mark attachment ready: %w", err)
	}
	if result.RowsAffected() == 0 {
		return service.NewError(service.CodeNotFound, "attachment not found", nil)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit replace attachment chunks: %w", err)
	}
	return nil
}

func (r *Postgres) SoftDeleteAttachment(ctx context.Context, userID, sessionID, attachmentID string) (service.SessionAttachment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return service.SessionAttachment{}, fmt.Errorf("begin delete attachment: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	item, err := scanAttachment(tx.QueryRow(ctx, `
UPDATE session_attachments a
SET status='deleted', deleted_at=COALESCE(a.deleted_at, now()), updated_at=now()
FROM conversations c
WHERE c.id=a.conversation_id
    AND a.id::text=$1
    AND a.conversation_id::text=$2
    AND a.external_user_id=$3
    AND c.external_user_id=$3
    AND c.deleted_at IS NULL
    AND a.deleted_at IS NULL
RETURNING a.id::text, a.conversation_id::text, a.external_user_id, a.file_ref, a.filename,
    a.content_type, a.size_bytes, a.status, COALESCE(a.error_code, ''), COALESCE(a.error_summary, ''),
    a.page_count, a.chunk_count, a.expires_at, a.deleted_at, a.created_at, a.updated_at`,
		attachmentID, sessionID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		if _, accessErr := r.GetConversation(ctx, userID, sessionID); accessErr != nil {
			return service.SessionAttachment{}, accessErr
		}
		return service.SessionAttachment{}, service.NewError(service.CodeNotFound, "attachment not found", err)
	}
	if err != nil {
		return service.SessionAttachment{}, err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM session_attachment_chunks WHERE attachment_id::text=$1`, attachmentID); err != nil {
		return service.SessionAttachment{}, fmt.Errorf("delete attachment chunks: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return service.SessionAttachment{}, fmt.Errorf("commit delete attachment: %w", err)
	}
	return item, nil
}

func (r *Postgres) ValidateReadyAttachments(ctx context.Context, userID, sessionID string, ids []string) ([]service.SessionAttachment, error) {
	ids = normalizeRepoIDs(ids)
	if len(ids) == 0 {
		return nil, nil
	}
	if _, err := r.GetConversation(ctx, userID, sessionID); err != nil {
		return nil, err
	}
	rows, err := r.pool.Query(ctx, attachmentSelect+`
JOIN conversations c ON c.id=a.conversation_id
WHERE a.id::text=ANY($1::text[])
    AND a.conversation_id::text=$2
    AND a.external_user_id=$3
    AND c.external_user_id=$3
    AND c.deleted_at IS NULL
    AND a.deleted_at IS NULL
    AND a.status='ready'
ORDER BY array_position($1::text[], a.id::text)`, ids, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("validate ready attachments: %w", err)
	}
	defer rows.Close()
	items, err := scanAttachments(rows)
	if err != nil {
		return nil, err
	}
	if len(items) != len(ids) {
		return nil, service.NewError(service.CodeNotFound, "attachment not found", nil)
	}
	return items, nil
}

func (r *Postgres) SearchSessionAttachmentChunks(ctx context.Context, userID, sessionID string, attachmentIDs []string, query string, limit int) ([]service.AttachmentChunk, error) {
	attachmentIDs = normalizeRepoIDs(attachmentIDs)
	if len(attachmentIDs) == 0 {
		return nil, service.ValidationError(map[string]string{"attachmentIds": "at least one attachment id is required"})
	}
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	query = strings.TrimSpace(query)
	rows, err := r.pool.Query(ctx, `
SELECT
    ch.id::text,
    ch.attachment_id::text,
    ch.chunk_order,
    ch.page_number,
    COALESCE(ch.section_path, ''),
    ch.body,
    ch.preview,
    ch.token_count,
    ch.char_count,
    a.filename,
    ch.created_at
FROM session_attachment_chunks ch
JOIN session_attachments a ON a.id=ch.attachment_id
JOIN conversations c ON c.id=a.conversation_id
WHERE a.id::text=ANY($1::text[])
    AND a.conversation_id::text=$2
    AND a.external_user_id=$3
    AND c.external_user_id=$3
    AND c.deleted_at IS NULL
    AND a.deleted_at IS NULL
    AND a.status='ready'
    AND (
        NULLIF($4, '') IS NULL
        OR ch.body ILIKE '%' || $4 || '%'
        OR ch.preview ILIKE '%' || $4 || '%'
        OR to_tsvector('simple', ch.body) @@ plainto_tsquery('simple', $4)
    )
ORDER BY
    CASE WHEN NULLIF($4, '') IS NULL THEN 1 ELSE ts_rank_cd(to_tsvector('simple', ch.body), plainto_tsquery('simple', $4)) END DESC,
    ch.chunk_order
LIMIT $5`, attachmentIDs, sessionID, userID, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search attachment chunks: %w", err)
	}
	defer rows.Close()
	items := make([]service.AttachmentChunk, 0)
	for rows.Next() {
		var item service.AttachmentChunk
		var page *int
		if err := rows.Scan(&item.ID, &item.AttachmentID, &item.ChunkOrder, &page, &item.SectionPath, &item.Body, &item.Preview, &item.TokenCount, &item.CharCount, &item.Filename, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan attachment chunk: %w", err)
		}
		item.PageNumber = page
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Postgres) CleanupExpiredAttachments(ctx context.Context, now time.Time, limit int) ([]service.SessionAttachment, error) {
	if limit <= 0 {
		limit = 100
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin cleanup expired attachments: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
WITH expired AS (
    SELECT id
    FROM session_attachments
    WHERE deleted_at IS NULL
        AND expires_at <= $1
    ORDER BY expires_at
    LIMIT $2
    FOR UPDATE SKIP LOCKED
), updated AS (
    UPDATE session_attachments a
    SET status='deleted', deleted_at=now(), updated_at=now()
    FROM expired
    WHERE a.id=expired.id
    RETURNING a.id::text, a.conversation_id::text, a.external_user_id, a.file_ref, a.filename,
        a.content_type, a.size_bytes, a.status, COALESCE(a.error_code, ''), COALESCE(a.error_summary, ''),
        a.page_count, a.chunk_count, a.expires_at, a.deleted_at, a.created_at, a.updated_at
)
SELECT * FROM updated`, now, limit)
	if err != nil {
		return nil, fmt.Errorf("cleanup expired attachments: %w", err)
	}
	items, err := scanAttachments(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if _, err := tx.Exec(ctx, `DELETE FROM session_attachment_chunks WHERE attachment_id::text=$1`, item.ID); err != nil {
			return nil, fmt.Errorf("delete expired attachment chunks: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit cleanup expired attachments: %w", err)
	}
	return items, nil
}

func (r *Postgres) CleanupSessionAttachments(ctx context.Context, userID, sessionID string) ([]service.SessionAttachment, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin cleanup session attachments: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
WITH updated AS (
    UPDATE session_attachments a
    SET status='deleted', deleted_at=COALESCE(a.deleted_at, now()), updated_at=now()
    FROM conversations c
    WHERE c.id=a.conversation_id
        AND a.conversation_id::text=$1
        AND a.external_user_id=$2
        AND c.external_user_id=$2
        AND a.deleted_at IS NULL
    RETURNING a.id::text, a.conversation_id::text, a.external_user_id, a.file_ref, a.filename,
        a.content_type, a.size_bytes, a.status, COALESCE(a.error_code, ''), COALESCE(a.error_summary, ''),
        a.page_count, a.chunk_count, a.expires_at, a.deleted_at, a.created_at, a.updated_at
)
SELECT * FROM updated`, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("cleanup session attachments: %w", err)
	}
	items, err := scanAttachments(rows)
	rows.Close()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if _, err := tx.Exec(ctx, `DELETE FROM session_attachment_chunks WHERE attachment_id::text=$1`, item.ID); err != nil {
			return nil, fmt.Errorf("delete session attachment chunks: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit cleanup session attachments: %w", err)
	}
	return items, nil
}

func (r *Postgres) ListAttachmentsPendingFileDeleteRetry(ctx context.Context, limit int) ([]service.SessionAttachment, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.pool.Query(ctx, attachmentSelect+`
WHERE a.deleted_at IS NOT NULL
    AND a.file_delete_error_summary IS NOT NULL
ORDER BY a.updated_at
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list attachments pending file delete retry: %w", err)
	}
	defer rows.Close()
	return scanAttachments(rows)
}

func (r *Postgres) MarkAttachmentFileDeleteRequested(ctx context.Context, attachmentID, summary string) error {
	_, err := r.pool.Exec(ctx, `
UPDATE session_attachments
SET file_delete_requested_at=COALESCE(file_delete_requested_at, now()),
    file_delete_error_summary=NULLIF($2, ''),
    updated_at=now()
WHERE id::text=$1`, attachmentID, safeErrorSummary(summary))
	if err != nil {
		return fmt.Errorf("mark attachment file delete requested: %w", err)
	}
	return nil
}

func (r *Postgres) bindMessageAttachments(ctx context.Context, tx pgx.Tx, userID, conversationID, messageID string, attachmentIDs []string) error {
	attachmentIDs = normalizeRepoIDs(attachmentIDs)
	if len(attachmentIDs) == 0 {
		return nil
	}
	rows, err := tx.Query(ctx, `
SELECT a.id::text
FROM session_attachments a
JOIN conversations c ON c.id=a.conversation_id
WHERE a.id::text=ANY($1::text[])
    AND a.conversation_id::text=$2
    AND a.external_user_id=$3
    AND c.external_user_id=$3
    AND c.deleted_at IS NULL
    AND a.deleted_at IS NULL
    AND a.status='ready'`, attachmentIDs, conversationID, userID)
	if err != nil {
		return fmt.Errorf("validate message attachments: %w", err)
	}
	found := map[string]struct{}{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		found[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if len(found) != len(attachmentIDs) {
		return service.NewError(service.CodeNotFound, "attachment not found", nil)
	}
	for _, id := range attachmentIDs {
		if _, err := tx.Exec(ctx, `INSERT INTO message_attachments(message_id, attachment_id) VALUES($1, $2) ON CONFLICT DO NOTHING`, messageID, id); err != nil {
			return fmt.Errorf("bind message attachment: %w", err)
		}
	}
	return nil
}

func scanAttachments(rows pgx.Rows) ([]service.SessionAttachment, error) {
	items := make([]service.SessionAttachment, 0)
	for rows.Next() {
		item, err := scanAttachment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanAttachment(row rowScanner) (service.SessionAttachment, error) {
	var item service.SessionAttachment
	err := row.Scan(
		&item.ID, &item.SessionID, &item.OwnerUserID, &item.FileRef,
		&item.Filename, &item.ContentType, &item.SizeBytes, &item.Status,
		&item.ErrorCode, &item.ErrorSummary, &item.PageCount, &item.ChunkCount,
		&item.ExpiresAt, &item.DeletedAt, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return service.SessionAttachment{}, err
	}
	return item, nil
}

func normalizeRepoIDs(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func nullableChunkPage(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func safeErrorCode(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, value)
	if value == "" {
		return "dependency_error"
	}
	if len(value) > 64 {
		return value[:64]
	}
	return value
}

func safeErrorSummary(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lowered := strings.ToLower(value)
	for _, marker := range []string{"token", "apikey", "api_key", "secret", "bucket", "object key", "internal url", "http://", "https://"} {
		if strings.Contains(lowered, marker) {
			return "attachment processing failed"
		}
	}
	runes := []rune(value)
	if len(runes) > 240 {
		return string(runes[:240])
	}
	return value
}
