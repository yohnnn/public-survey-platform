ALTER TABLE users
    ADD COLUMN IF NOT EXISTS nickname TEXT;

UPDATE users
SET nickname = 'user_' || SUBSTRING(id FROM 1 FOR 8)
WHERE nickname IS NULL OR BTRIM(nickname) = '';

ALTER TABLE users
    ALTER COLUMN nickname SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'users_nickname_key'
    ) THEN
        ALTER TABLE users
            ADD CONSTRAINT users_nickname_key UNIQUE (nickname);
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS user_follows (
    follower_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followee_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, followee_id),
    CONSTRAINT user_follows_no_self_follow CHECK (follower_id <> followee_id)
);

CREATE INDEX IF NOT EXISTS idx_user_follows_followee_id
    ON user_follows (followee_id);

CREATE INDEX IF NOT EXISTS idx_user_follows_follower_id
    ON user_follows (follower_id);
