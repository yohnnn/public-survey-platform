CREATE TABLE processed_events (
    event_id   TEXT PRIMARY KEY,
    topic      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_processed_events_created_at ON processed_events (created_at DESC);
