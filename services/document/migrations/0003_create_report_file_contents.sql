-- +goose Up
CREATE TABLE report_file_contents (
    report_file_id uuid PRIMARY KEY REFERENCES report_files(id) ON DELETE CASCADE,
    content bytea NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS report_file_contents;
