-- Pg Setup
-- 1. Create a dedicated database
CREATE DATABASE minurldb;

-- 2. Create the Application User
-- CREATE USER web WITH ENCRYPTED PASSWORD 'qwe';

-- 3. Revoke default public permissions (Safety first)
REVOKE ALL ON SCHEMA public FROM PUBLIC;
GRANT CONNECT ON DATABASE minurldb TO ukiran;
GRANT USAGE ON SCHEMA public TO ukiran;


-- name: users
CREATE TABLE IF NOT EXISTS users (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email               VARCHAR(255) UNIQUE NOT NULL,
    username            VARCHAR(100) UNIQUE,
    password_hash       TEXT NOT NULL,
    full_name           VARCHAR(255),
    avatar_url          TEXT,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    is_verified         BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at       TIMESTAMPTZ,
    monthly_link_limit  BIGINT NOT NULL DEFAULT 100,
    current_month_links BIGINT NOT NULL DEFAULT 0,
    plan_tier           VARCHAR(20) NOT NULL DEFAULT 'free'
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created ON users(created_at DESC);

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();


-- name: urls
CREATE TABLE minurls (
    slug        VARCHAR(255) COLLATE "C" NOT NULL,   -- still use "C" collation for speed
    name        VARCHAR(255),                        -- Added: Human-readable title
    url         TEXT NOT NULL,
    owner_id    UUID,                                 -- nullable for anonymous links
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ,
    is_custom   BOOLEAN NOT NULL DEFAULT false,

    PRIMARY KEY (slug)
);

-- Important indexes for current version
CREATE INDEX idx_minurls_owner_created
    ON minurls (owner_id, created_at DESC)
    WHERE owner_id IS NOT NULL;

CREATE INDEX idx_minurls_created
    ON minurls (created_at DESC);

CREATE INDEX idx_minurls_expires
    ON minurls (expires_at)
    WHERE expires_at IS NOT NULL;

-- Index for searching by name
CREATE INDEX idx_minurls_name
    ON minurls (name)
    WHERE name IS NOT NULL;


-- name: click_events
CREATE TABLE click_events (
    id            BIGSERIAL PRIMARY KEY,
    slug          VARCHAR(255) COLLATE "C" NOT NULL,
    clicked_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip_address    INET,
    user_agent    TEXT,
    referrer      TEXT,
    country_code  VARCHAR(2),
    device_type   VARCHAR(20)
);

-- Key indexes for analytics
CREATE INDEX idx_click_events_slug_time
    ON click_events (slug, clicked_at DESC);

CREATE INDEX idx_click_events_time
    ON click_events (clicked_at DESC);
