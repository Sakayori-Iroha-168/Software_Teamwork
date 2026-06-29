-- +goose Up
CREATE TABLE report_settings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    llm_profile_id text,
    default_template_id uuid,
    default_file_format text NOT NULL DEFAULT 'docx',
    default_numbering_mode text NOT NULL DEFAULT 'global',
    updated_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO report_settings (default_file_format, default_numbering_mode)
VALUES ('docx', 'global')
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS report_settings;
