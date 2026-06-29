-- name: GetReportSettings :one
SELECT id, llm_profile_id, default_template_id, default_file_format, default_numbering_mode, updated_at, created_at
FROM report_settings
LIMIT 1;

-- name: UpdateReportSettings :one
UPDATE report_settings
SET
    llm_profile_id        = COALESCE($1, llm_profile_id),
    default_template_id   = COALESCE($2, default_template_id),
    default_file_format   = COALESCE($3, default_file_format),
    default_numbering_mode = COALESCE($4, default_numbering_mode),
    updated_at            = NOW()
RETURNING id, llm_profile_id, default_template_id, default_file_format, default_numbering_mode, updated_at, created_at;
