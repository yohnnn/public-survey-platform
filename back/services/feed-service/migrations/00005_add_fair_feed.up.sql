ALTER TABLE feed_items
    ADD COLUMN exposure_target INT NOT NULL DEFAULT 100,
    ADD COLUMN impression_count INT NOT NULL DEFAULT 0;

CREATE TABLE feed_item_impressions (
    feed_item_id TEXT NOT NULL REFERENCES feed_items(id) ON DELETE CASCADE,
    viewer_key   TEXT NOT NULL,
    seen_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (feed_item_id, viewer_key)
);

CREATE INDEX idx_feed_items_discovery ON feed_items (impression_count ASC, created_at DESC, id DESC);
CREATE INDEX idx_feed_item_impressions_seen_at ON feed_item_impressions (seen_at DESC);
