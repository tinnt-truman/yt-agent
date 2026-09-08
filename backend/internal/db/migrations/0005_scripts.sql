CREATE TABLE IF NOT EXISTS scripts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idea_title        TEXT NOT NULL,
    idea_description  TEXT,
    idea_hook         TEXT,
    duration_format   TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'pending',
    error_message     TEXT,
    script_json       JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_scripts_created_at ON scripts (created_at DESC);
