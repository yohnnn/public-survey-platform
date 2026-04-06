DROP INDEX IF EXISTS idx_polls_created_at;
DROP INDEX IF EXISTS idx_poll_tags_tag;
DROP INDEX IF EXISTS idx_poll_tags_poll;
DROP INDEX IF EXISTS idx_poll_options_id_unique;
DROP TABLE IF EXISTS poll_tags;
DROP TABLE IF EXISTS poll_options;
DROP TABLE IF EXISTS polls;
DROP TABLE IF EXISTS tags;
