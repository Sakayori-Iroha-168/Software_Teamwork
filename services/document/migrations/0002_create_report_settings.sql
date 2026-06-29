-- +goose Up
CREATE TABLE report_settings (
    id text PRIMARY KEY,
    settings_json jsonb NOT NULL DEFAULT '{}'::jsonb,
    updated_by text,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS report_settings;
