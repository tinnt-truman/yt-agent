-- OpenCode Zen's free tier rejects API calls made outside the OpenCode client
-- ("OpenCode's free tier can only be used in OpenCode"), so the free-model
-- provider moves to OpenRouter, whose :free variants are callable via API.
-- Reuses the key column added in 0006 and remaps the provider value.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'settings' AND column_name = 'opencode_api_key'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'settings' AND column_name = 'openrouter_api_key'
    ) THEN
        ALTER TABLE settings RENAME COLUMN opencode_api_key TO openrouter_api_key;
    END IF;
END $$;

ALTER TABLE settings ADD COLUMN IF NOT EXISTS openrouter_api_key TEXT NOT NULL DEFAULT '';

UPDATE settings SET ai_provider = 'openrouter' WHERE ai_provider = 'zen';
UPDATE settings SET ai_model = 'nvidia/nemotron-3-ultra-550b-a55b:free'
  WHERE ai_model IN ('big-pickle', 'mimo-v2.5-free', 'ling-3.0-flash-fin-free',
    'nemotron-3-ultra-free', 'nemotron-3.5-lightning-free',
    'muse-spark-1.2-contributor-free', 'muse-spark-1.3-contributor-free');
