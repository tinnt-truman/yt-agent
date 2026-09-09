ALTER TABLE video_prompt_series ADD COLUMN IF NOT EXISTS scenes_per_episode INTEGER NOT NULL DEFAULT 3;
