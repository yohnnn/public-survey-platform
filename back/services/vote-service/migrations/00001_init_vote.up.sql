CREATE TABLE IF NOT EXISTS votes (
    user_id TEXT NOT NULL,
    poll_id TEXT NOT NULL,
    option_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, poll_id, option_id)
);

CREATE INDEX IF NOT EXISTS idx_votes_poll_id ON votes (poll_id);
CREATE INDEX IF NOT EXISTS idx_votes_user_poll ON votes (user_id, poll_id);
