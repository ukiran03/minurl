CREATE TABLE custom_minurls (
    slug VARCHAR(64) NOT NULL,
    url         TEXT NOT NULL,
    title       VARCHAR(255) NULL,

    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NULL,

    user_id     BIGINT NOT NULL,

    CONSTRAINT pk_custom_minurls PRIMARY KEY (slug)
);

-- Covering index for custom redirect paths (Index-Only Scan)
CREATE INDEX idx_custom_redirect_covering
  ON custom_minurls (slug)
  INCLUDE (url, expires_at);

-- Partial index for background TTL deletion workers on custom links
CREATE INDEX idx_custom_expires_at
  ON custom_minurls (expires_at)
  WHERE expires_at IS NOT NULL;

-- Dashboard history index (Matches the structure of the standard table)
CREATE INDEX idx_custom_user_dashboard
  ON custom_minurls (user_id, created_at DESC);

---- create above / drop below ----

DROP TABLE IF EXISTS custom_minurls;
