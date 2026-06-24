-- search_path is set to "media" by the connection (and the role default),
-- so unqualified names resolve to the media schema.

CREATE TABLE IF NOT EXISTS media (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id      UUID        NOT NULL,
    original_name TEXT,
    content_type  TEXT,
    sizes         JSONB       NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_media_owner_id ON media (owner_id);
