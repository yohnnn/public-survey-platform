CREATE TABLE IF NOT EXISTS processed_events (
    event_id TEXT PRIMARY KEY,
    topic TEXT NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_processed_events_processed_at
    ON processed_events(processed_at DESC);
