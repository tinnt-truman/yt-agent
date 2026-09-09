CREATE TABLE IF NOT EXISTS video_prompt_series (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_title       TEXT NOT NULL,
    video_description TEXT,
    video_tags        JSONB,
    episode_count     INTEGER NOT NULL DEFAULT 5,
    status            TEXT NOT NULL DEFAULT 'pending',
    error_message     TEXT,
    series_json       JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_video_prompt_series_created_at ON video_prompt_series (created_at DESC);
