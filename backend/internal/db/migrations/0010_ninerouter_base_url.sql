-- 9Router has no fixed public host — it's a process the user runs
-- themselves, which may live on a different machine than this backend.
-- Empty stays "use the default" (same-machine, default port), resolved in
-- ai.NewClientFromSettings rather than baked in here.
ALTER TABLE settings ADD COLUMN IF NOT EXISTS ninerouter_base_url TEXT NOT NULL DEFAULT '';
