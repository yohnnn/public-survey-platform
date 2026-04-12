CREATE TABLE poll_option_stats (
    poll_id TEXT NOT NULL,
    option_id TEXT NOT NULL,
    votes_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (poll_id, option_id)
);

CREATE TABLE poll_country_stats (
    poll_id TEXT NOT NULL,
    country TEXT NOT NULL,
    votes_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (poll_id, country)
);

CREATE TABLE poll_gender_stats (
    poll_id TEXT NOT NULL,
    gender TEXT NOT NULL,
    votes_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (poll_id, gender)
);

CREATE TABLE poll_age_stats (
    poll_id TEXT NOT NULL,
    age_range TEXT NOT NULL,
    votes_count BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (poll_id, age_range)
);

CREATE INDEX idx_poll_option_stats_poll ON poll_option_stats (poll_id);
CREATE INDEX idx_poll_country_stats_poll ON poll_country_stats (poll_id);
CREATE INDEX idx_poll_gender_stats_poll ON poll_gender_stats (poll_id);
CREATE INDEX idx_poll_age_stats_poll ON poll_age_stats (poll_id);
