package repository

import (
	"context"
	"fmt"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
	"github.com/google/uuid"
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
	blockOrder int,
	blockType string,
	content string,
	status domain.MessageContentBlockStatus,
) (domain.MessageContentBlock, error) {
	id := uuid.NewString()
	now := nowUTC()
	const query = `
		INSERT INTO message_content_blocks (
			id, message_id, block_order, block_type, content, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)
		RETURNING id, message_id, block_order, block_type, content, status,
			provider_block_id, provider_metadata, created_at, updated_at
	`
	var block domain.MessageContentBlock
	var providerBlockID *string
	var providerMetadata map[string]any
	err := r.db.QueryRow(
		ctx, query,
		id, messageID, blockOrder, blockType, content, string(status), now,
	).Scan(
		&block.ID, &block.MessageID, &block.BlockOrder, &block.BlockType,
		&block.Content, &block.Status, &providerBlockID, &providerMetadata,
		&block.CreatedAt, &block.UpdatedAt,
	)
	if err != nil {
		return domain.MessageContentBlock{}, fmt.Errorf("create content block: %w", err)
	}
	if providerBlockID != nil {
		block.ProviderBlockID = *providerBlockID
	}
	block.ProviderMetadata = providerMetadata
	return block, nil
}

func (r *ContentBlockRepository) UpdateContent(
	ctx context.Context,
	id string,
	content string,
) error {
	now := nowUTC()
	const query = `
		UPDATE message_content_blocks SET content = $2, updated_at = $3 WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, content, now)
	if err != nil {
		return fmt.Errorf("update content block: %w", err)
	}
	return nil
}

func (r *ContentBlockRepository) UpdateStatus(
	ctx context.Context,
	id string,
	status domain.MessageContentBlockStatus,
) error {
	now := nowUTC()
	const query = `
		UPDATE message_content_blocks SET status = $2, updated_at = $3 WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id, string(status), now)
	if err != nil {
		return fmt.Errorf("update content block status: %w", err)
	}
	return nil
}

func (r *ContentBlockRepository) ListByMessageID(
	ctx context.Context,
	messageID string,
) ([]domain.MessageContentBlock, error) {
	const query = `
		SELECT id, message_id, block_order, block_type, content, status,
			provider_block_id, provider_metadata, created_at, updated_at
		FROM message_content_blocks
		WHERE message_id = $1
		ORDER BY block_order ASC
	`
	rows, err := r.db.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("list content blocks: %w", err)
	}
	defer rows.Close()

	var blocks []domain.MessageContentBlock
	for rows.Next() {
		var block domain.MessageContentBlock
		var providerBlockID *string
		var providerMetadata map[string]any
		if err := rows.Scan(
			&block.ID, &block.MessageID, &block.BlockOrder, &block.BlockType,
			&block.Content, &block.Status, &providerBlockID, &providerMetadata,
			&block.CreatedAt, &block.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan content block: %w", err)
		}
		if providerBlockID != nil {
			block.ProviderBlockID = *providerBlockID
		}
		block.ProviderMetadata = providerMetadata
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content blocks: %w", err)
	}
	return blocks, nil
}

func (r *ContentBlockRepository) ListCompletedTextByMessageID(
	ctx context.Context,
	messageID string,
) ([]domain.MessageContentBlock, error) {
	const query = `
		SELECT id, message_id, block_order, block_type, content, status,
			provider_block_id, provider_metadata, created_at, updated_at
		FROM message_content_blocks
		WHERE message_id = $1 AND block_type = 'text' AND status = 'completed'
		ORDER BY block_order ASC
	`
	rows, err := r.db.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("list completed text blocks: %w", err)
	}
	defer rows.Close()

	var blocks []domain.MessageContentBlock
	for rows.Next() {
		var block domain.MessageContentBlock
		var providerBlockID *string
		var providerMetadata map[string]any
		if err := rows.Scan(
			&block.ID, &block.MessageID, &block.BlockOrder, &block.BlockType,
			&block.Content, &block.Status, &providerBlockID, &providerMetadata,
			&block.CreatedAt, &block.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan content block: %w", err)
		}
		if providerBlockID != nil {
			block.ProviderBlockID = *providerBlockID
		}
		block.ProviderMetadata = providerMetadata
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate content blocks: %w", err)
	}
	return blocks, nil
}
