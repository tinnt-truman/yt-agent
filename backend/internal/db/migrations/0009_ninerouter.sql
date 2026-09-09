-- 9Router (https://github.com/decolua/9router) is a self-hosted local AI
-- router fronting many upstream providers behind one OpenAI-compatible
-- endpoint, added as a third selectable ai_provider alongside deepseek and
-- openrouter. Its API key is generated per-install from its own dashboard,
-- so it gets its own column rather than reusing openrouter_api_key.
ALTER TABLE settings ADD COLUMN IF NOT EXISTS ninerouter_api_key TEXT NOT NULL DEFAULT '';
