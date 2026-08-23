-- +goose Up
CREATE TABLE IF NOT EXISTS url_clicks (
    slug LowCardinality(String),
    timestamp DateTime,
    remote_addr String,
    user_agent String,
    referrer String
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(timestamp)
ORDER BY (slug, timestamp);

-- +goose Down
DROP TABLE IF EXISTS url_clicks;
