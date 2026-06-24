-- search_path is set to "noti" by the connection (and the role default),
-- so unqualified names resolve to the noti schema.

CREATE TABLE IF NOT EXISTS device_tokens (
    user_id    UUID        NOT NULL,
    token      TEXT        PRIMARY KEY,
    platform   TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_user_id ON device_tokens (user_id);

CREATE TABLE IF NOT EXISTS notifications (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID,
    type       TEXT,
    payload    JSONB,
    read       BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications (user_id);
