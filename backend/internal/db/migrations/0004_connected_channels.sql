CREATE TABLE IF NOT EXISTS connected_channels (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id        TEXT NOT NULL UNIQUE,
    channel_title     TEXT NOT NULL,
    channel_thumbnail TEXT,
    subscriber_count  BIGINT NOT NULL DEFAULT 0,
    google_email      TEXT,
    access_token      TEXT NOT NULL,
    refresh_token     TEXT NOT NULL,
    token_expiry      TIMESTAMPTZ NOT NULL,
    scopes            TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
