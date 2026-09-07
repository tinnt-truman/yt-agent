CREATE TABLE IF NOT EXISTS settings (
    id                INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    youtube_api_key   TEXT NOT NULL DEFAULT '',
    anthropic_api_key TEXT NOT NULL DEFAULT '',
    ai_model          TEXT NOT NULL DEFAULT 'claude-opus-5',
    max_videos        INTEGER NOT NULL DEFAULT 50,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO settings (id) VALUES (1) ON CONFLICT (id) DO NOTHING;
