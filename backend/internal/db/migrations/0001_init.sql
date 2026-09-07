CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS analyses (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    input_url      TEXT NOT NULL,
    channel_id     TEXT,
    channel_title  TEXT,
    status         TEXT NOT NULL DEFAULT 'pending',
    stage_message  TEXT,
    error_message  TEXT,
    analysis_json  JSONB,
    ai_output_json JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_analyses_created_at ON analyses (created_at DESC);
