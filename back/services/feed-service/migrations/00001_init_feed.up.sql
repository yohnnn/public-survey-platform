CREATE TABLE feed_items (
    id          TEXT NOT NULL PRIMARY KEY,
    creator_id  TEXT NOT NULL,
    question    TEXT NOT NULL,
    total_votes BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE feed_item_options (
    id          TEXT NOT NULL PRIMARY KEY,
    feed_item_id TEXT NOT NULL REFERENCES feed_items(id) ON DELETE CASCADE,
    text        TEXT NOT NULL,
    votes_count BIGINT NOT NULL DEFAULT 0,
    position    INT NOT NULL DEFAULT 0
);

CREATE TABLE feed_item_tags (
    feed_item_id TEXT NOT NULL REFERENCES feed_items(id) ON DELETE CASCADE,
    tag          TEXT NOT NULL,
    PRIMARY KEY (feed_item_id, tag)
);

CREATE INDEX idx_feed_items_created_at_id ON feed_items (created_at DESC, id DESC);
CREATE INDEX idx_feed_items_total_votes_id ON feed_items (total_votes DESC, id DESC);
CREATE INDEX idx_feed_items_creator_created_at_id ON feed_items (creator_id, created_at DESC, id DESC);
CREATE INDEX idx_feed_item_options_feed_item_id ON feed_item_options (feed_item_id);
CREATE INDEX idx_feed_item_tags_feed_item_id ON feed_item_tags (feed_item_id);
