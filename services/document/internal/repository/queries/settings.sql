-- name: GetReportSettings :one
SELECT id, llm_profile_id, default_templates, default_file_format, default_numbering_mode, updated_at, created_at
FROM report_settings
LIMIT 1;

-- name: UpdateReportSettings :one
UPDATE report_settings
SET
    llm_profile_id         = COALESCE($1, llm_profile_id),
    default_templates      = CASE WHEN $2::jsonb IS NOT NULL
                                  THEN $2::jsonb
                                  ELSE default_templates END,
    default_file_format    = COALESCE($3, default_file_format),
    default_numbering_mode = COALESCE($4, default_numbering_mode),
    updated_at             = NOW()
RETURNING id, llm_profile_id, default_templates, default_file_format, default_numbering_mode, updated_at, created_at;
