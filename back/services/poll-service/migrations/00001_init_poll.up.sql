CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS polls (
    id TEXT PRIMARY KEY,
    creator_id TEXT NOT NULL,
    question TEXT NOT NULL,
    type SMALLINT NOT NULL CHECK (type IN (1, 2)),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    total_votes BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS poll_options (
    id TEXT NOT NULL,
    poll_id TEXT NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    votes_count INTEGER NOT NULL DEFAULT 0,
    position INTEGER NOT NULL,
    PRIMARY KEY (poll_id, position)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_poll_options_id_unique ON poll_options(id);

CREATE TABLE IF NOT EXISTS poll_tags (
    poll_id TEXT NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (poll_id, tag_id)
);

CREATE INDEX idx_poll_tags_poll ON poll_tags(poll_id);
CREATE INDEX idx_poll_tags_tag ON poll_tags(tag_id);
CREATE INDEX idx_polls_created_at ON polls(created_at DESC);
