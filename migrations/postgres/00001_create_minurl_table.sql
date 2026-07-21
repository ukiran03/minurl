-- +goose Up
CREATE TABLE minurls (
    slug        BIGINT NOT NULL,
    url         TEXT NOT NULL,
    url_hash    CHAR(32) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NULL,

    CONSTRAINT pk_minurls PRIMARY KEY (slug),
    CONSTRAINT uq_minurls_url_hash UNIQUE (url_hash)
);

-- Partial index for background TTL workers
CREATE INDEX idx_minurls_expires_at
  ON minurls (expires_at)
  WHERE expires_at IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS minurls;
