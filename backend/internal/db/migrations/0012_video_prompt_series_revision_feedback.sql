-- Same purpose as scripts.revision_feedback (see 0011) but for video prompt
-- series jobs.
ALTER TABLE video_prompt_series ADD COLUMN IF NOT EXISTS revision_feedback TEXT;
