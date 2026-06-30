-- name: DeleteCitationsByMessage :exec
DELETE FROM citations
WHERE message_id = sqlc.arg(message_id)::uuid;

-- name: InsertCitation :exec
INSERT INTO citations (
    id,
    message_id,
    citation_no,
    char_start,
    char_end,
    external_kb_id,
    external_doc_id,
    external_chunk_id,
    doc_name,
    section_path,
    quote_text,
    context,
    page_number,
    score,
    rerank_score,
    chunk_type,
    metadata
) VALUES (
    sqlc.arg(id)::uuid,
    sqlc.arg(message_id)::uuid,
    sqlc.arg(citation_no),
    NULLIF(sqlc.arg(char_start), 0),
    NULLIF(sqlc.arg(char_end), 0),
    NULLIF(sqlc.arg(external_kb_id), ''),
    NULLIF(sqlc.arg(external_doc_id), ''),
    NULLIF(sqlc.arg(external_chunk_id), ''),
    sqlc.arg(doc_name),
    NULLIF(sqlc.arg(section_path), ''),
    NULLIF(sqlc.arg(quote_text), ''),
    NULLIF(sqlc.arg(context), ''),
    NULLIF(sqlc.arg(page_number), 0),
    NULLIF(sqlc.arg(score), 0),
    NULLIF(sqlc.arg(rerank_score), 0),
    NULLIF(sqlc.arg(chunk_type), ''),
    sqlc.arg(metadata)
);
