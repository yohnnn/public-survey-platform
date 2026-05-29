DROP INDEX IF EXISTS idx_feed_item_impressions_seen_at;
DROP INDEX IF EXISTS idx_feed_items_discovery;
DROP TABLE IF EXISTS feed_item_impressions;
ALTER TABLE feed_items
    DROP COLUMN IF EXISTS impression_count,
    DROP COLUMN IF EXISTS exposure_target;
