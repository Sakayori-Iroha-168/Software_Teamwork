package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CitationRepository struct {
	db *pgxpool.Pool
}

func NewCitationRepository(db *pgxpool.Pool) *CitationRepository {
	return &CitationRepository{db: db}
}

func (r *CitationRepository) Create(
	ctx context.Context,
	messageID string,
	citationNo int,
	charStart *int,
	charEnd *int,
	externalKBID string,
	externalDocID string,
	externalChunkID string,
	docName string,
	quoteText string,
	contextStr string,
	pageNumber *int,
	score *float64,
	metadata map[string]any,
) (domain.Citation, error) {
	id := uuid.NewString()
	var metadataJSON []byte
	if metadata != nil {
		var err error
		metadataJSON, err = json.Marshal(metadata)
		if err != nil {
			return domain.Citation{}, fmt.Errorf("marshal metadata: %w", err)
		}
	}

	const query = `
		INSERT INTO citations (
			id, message_id, citation_no, char_start, char_end,
			external_kb_id, external_doc_id, external_chunk_id,
			doc_name, quote_text, context, page_number, score, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id, message_id, citation_no, char_start, char_end,
			external_kb_id, external_doc_id, external_chunk_id,
			doc_name, quote_text, context, page_number, score, metadata
	`

	var citation domain.Citation
	var metadataRaw []byte
	err := r.db.QueryRow(ctx, query,
		id, messageID, citationNo, charStart, charEnd,
		externalKBID, externalDocID, externalChunkID,
		docName, quoteText, contextStr, pageNumber, score, metadataJSON,
	).Scan(
		&citation.ID, &citation.MessageID, &citation.CitationNo,
		&citation.CharStart, &citation.CharEnd,
		&citation.ExternalKBID, &citation.ExternalDocID, &citation.ExternalChunkID,
		&citation.DocName, &citation.QuoteText, &citation.Context,
		&citation.PageNumber, &citation.Score, &metadataRaw,
	)
	if err != nil {
		return domain.Citation{}, fmt.Errorf("create citation: %w", err)
	}

	if metadataRaw != nil {
		if err := json.Unmarshal(metadataRaw, &citation.Metadata); err != nil {
			return domain.Citation{}, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	return citation, nil
}

func (r *CitationRepository) ListByMessageID(ctx context.Context, messageID string) ([]domain.Citation, error) {
	const query = `
		SELECT id, message_id, citation_no, char_start, char_end,
			external_kb_id, external_doc_id, external_chunk_id,
			doc_name, quote_text, context, page_number, score, metadata
		FROM citations
		WHERE message_id = $1
		ORDER BY citation_no ASC
	`

	rows, err := r.db.Query(ctx, query, messageID)
	if err != nil {
		return nil, fmt.Errorf("list citations: %w", err)
	}
	defer rows.Close()

	var citations []domain.Citation
	for rows.Next() {
		var citation domain.Citation
		var metadataRaw []byte
		if err := rows.Scan(
			&citation.ID, &citation.MessageID, &citation.CitationNo,
			&citation.CharStart, &citation.CharEnd,
			&citation.ExternalKBID, &citation.ExternalDocID, &citation.ExternalChunkID,
			&citation.DocName, &citation.QuoteText, &citation.Context,
			&citation.PageNumber, &citation.Score, &metadataRaw,
		); err != nil {
			return nil, fmt.Errorf("scan citation: %w", err)
		}
		if metadataRaw != nil {
			if err := json.Unmarshal(metadataRaw, &citation.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshal metadata: %w", err)
			}
		}
		citations = append(citations, citation)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate citations: %w", err)
	}
	return citations, nil
}

func (r *CitationRepository) GetByID(ctx context.Context, id string) (domain.Citation, error) {
	const query = `
		SELECT id, message_id, citation_no, char_start, char_end,
			external_kb_id, external_doc_id, external_chunk_id,
			doc_name, quote_text, context, page_number, score, metadata
		FROM citations
		WHERE id = $1
	`

	var citation domain.Citation
	var metadataRaw []byte
	err := r.db.QueryRow(ctx, query, id).Scan(
		&citation.ID, &citation.MessageID, &citation.CitationNo,
		&citation.CharStart, &citation.CharEnd,
		&citation.ExternalKBID, &citation.ExternalDocID, &citation.ExternalChunkID,
		&citation.DocName, &citation.QuoteText, &citation.Context,
		&citation.PageNumber, &citation.Score, &metadataRaw,
	)
	if err != nil {
		return domain.Citation{}, fmt.Errorf("get citation: %w", err)
	}

	if metadataRaw != nil {
		if err := json.Unmarshal(metadataRaw, &citation.Metadata); err != nil {
			return domain.Citation{}, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	return citation, nil
}
