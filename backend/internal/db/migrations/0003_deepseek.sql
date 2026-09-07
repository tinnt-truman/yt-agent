DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'settings' AND column_name = 'anthropic_api_key'
    ) THEN
        ALTER TABLE settings RENAME COLUMN anthropic_api_key TO deepseek_api_key;
    END IF;
END $$;

ALTER TABLE settings ALTER COLUMN ai_model SET DEFAULT 'deepseek-v4-pro';

UPDATE settings SET ai_model = 'deepseek-v4-pro' WHERE ai_model = 'claude-opus-5';
