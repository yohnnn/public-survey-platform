CREATE TABLE IF NOT EXISTS pending_feed_item_votes (
    feed_item_id TEXT PRIMARY KEY,
    votes_delta BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS pending_feed_item_option_votes (
    feed_item_id TEXT NOT NULL,
    option_id TEXT NOT NULL,
    votes_delta BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (feed_item_id, option_id)
);
