-- +goose Up
-- Add response_run_id, content_preview, is_source_available, and
-- source_unavailable_reason columns to citations to align with the
-- documented data model (docs/services/qa/docs/data-models.md §10.1).
ALTER TABLE citations
    ADD COLUMN response_run_id UUID REFERENCES response_runs(id),
    ADD COLUMN content_preview TEXT,
    ADD COLUMN is_source_available BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN source_unavailable_reason TEXT;

CREATE INDEX idx_citations_response_run_id ON citations(response_run_id);

-- +goose Down
ALTER TABLE citations
    DROP COLUMN IF EXISTS source_unavailable_reason,
    DROP COLUMN IF EXISTS is_source_available,
    DROP COLUMN IF EXISTS content_preview,
    DROP COLUMN IF EXISTS response_run_id;
