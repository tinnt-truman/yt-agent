CREATE TABLE IF NOT EXISTS music_prompts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_title       TEXT NOT NULL,
    video_description TEXT,
    video_tags        JSONB,
    status            TEXT NOT NULL DEFAULT 'pending',
    error_message     TEXT,
    music_prompt_json JSONB,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_music_prompts_created_at ON music_prompts (created_at DESC);
