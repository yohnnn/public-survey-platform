DROP INDEX IF EXISTS idx_user_follows_follower_id;
DROP INDEX IF EXISTS idx_user_follows_followee_id;
DROP TABLE IF EXISTS user_follows;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_nickname_key;

ALTER TABLE users
    DROP COLUMN IF EXISTS nickname;
