-- OpenCode Zen (https://opencode.ai/zen) as a second AI provider next to DeepSeek.
-- ai_provider: 'deepseek' (default, back-compat) or 'zen'.
-- opencode_api_key: Zen API key from https://opencode.ai/auth (only needed when ai_provider = 'zen').
ALTER TABLE settings ADD COLUMN IF NOT EXISTS ai_provider TEXT NOT NULL DEFAULT 'deepseek';
ALTER TABLE settings ADD COLUMN IF NOT EXISTS opencode_api_key TEXT NOT NULL DEFAULT '';
