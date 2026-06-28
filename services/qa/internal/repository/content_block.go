package repository

import (
	"context"
	"fmt"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ContentBlockRepository struct {
	db *pgxpool.Pool
}

func NewContentBlockRepository(db *pgxpool.Pool) *ContentBlockRepository {
	return &ContentBlockRepository{db: db}
}

func (r *ContentBlockRepository) Create(
	ctx context.Context,
	messageID string,
	blockType string,
	content string,
	visibility domain.ContentBlockVisibility,
	sortOrder int,
) (domain.MessageContentBlock, error) {
	now := nowUTC()
	const query = `
		INSERT INTO message_content_blocks (message_id, block_type, content, visibility, sort_order, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, message_id, block_type, content, visibility, sort_order, created_at
	`
	var block domain.MessageContentBlock
	err := r.db.QueryRow(
		ctx, query,
		messageID, blockType, content, string(visibility), sortOrder, now,
	).Scan(
		&block.ID, &block.MessageID, &block.BlockType, &block.Content,
		&block.Visibility, &block.SortOrder, &block.CreatedAt,
	)
	if err != nil {
		return domain.MessageContentBlock{}, fmt.Errorf("create content block: %w", err)
	}
	return block, nil
}

func (r *ContentBlockRepository) ListByMessageID(
	ctx context.Context,
	messageID string,
) ([]domain.MessageContentBlock, error) {
	const query = `
		SELECT id, message_id, block_type, content, visibility, sort_order, created_at
		FROM message_content_blocks
		WHERE message_id = $1
		ORDER BY sort_order ASC
	`
	rows, err := r.db.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("list content blocks: %w", err)
	}
	defer rows.Close()

	var blocks []domain.MessageContentBlock
	for rows.Next() {
		var block domain.MessageContentBlock
		if err := rows.Scan(
			&block.ID, &block.MessageID, &block.BlockType, &block.Content,
			&block.Visibility, &block.SortOrder, &block.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan content block: %w", err)
		}
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content blocks: %w", err)
	}
	return blocks, nil
}

func (r *ContentBlockRepository) ListPublicByMessageID(
	ctx context.Context,
	messageID string,
) ([]domain.MessageContentBlock, error) {
	const query = `
		SELECT id, message_id, block_type, content, visibility, sort_order, created_at
		FROM message_content_blocks
		WHERE message_id = $1 AND visibility = 'public'
		ORDER BY sort_order ASC
	`
	rows, err := r.db.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("list public content blocks: %w", err)
	}
	defer rows.Close()

	var blocks []domain.MessageContentBlock
	for rows.Next() {
		var block domain.MessageContentBlock
		if err := rows.Scan(
			&block.ID, &block.MessageID, &block.BlockType, &block.Content,
			&block.Visibility, &block.SortOrder, &block.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan content block: %w", err)
		}
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content blocks: %w", err)
	}
	return blocks, nil
}
