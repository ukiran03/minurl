CREATE TABLE minurls (
    slug        BIGINT NOT NULL,
    url         TEXT NOT NULL,
    title       VARCHAR(255) NULL,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NULL,

    user_id     BIGINT NULL,

    CONSTRAINT pk_minurls PRIMARY KEY (slug)
);

-- Covering index for standard redirect paths (Index-Only Scan)
CREATE INDEX idx_minurls_redirect_covering
  ON minurls (slug)
  INCLUDE (url, expires_at);

-- Partial index for background TTL deletion workers
CREATE INDEX idx_minurls_expires_at
  ON minurls (expires_at)
  WHERE expires_at IS NOT NULL;

-- Dashboard history index (Pre-sorted for pagination)
CREATE INDEX idx_minurls_user_dashboard
  ON minurls (user_id, created_at DESC)
  WHERE user_id IS NOT NULL;

---- create above / drop below ----

DROP TABLE IF EXISTS minurls;
