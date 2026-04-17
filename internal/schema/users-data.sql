-- name: users data: Insert statements (create users)
-- User 1: Free tier user (Alice)
INSERT INTO users
       (email, username, password_hash, full_name, monthly_link_limit, plan_tier)
VALUES ('alice@example.com',
       'alice',
       '$2a$12$examplehashforalice1234567890',
       'Alice Sharma', 100, 'free')
RETURNING id, created_at;

-- User 2: Pro user (Bob)
INSERT INTO users
       (email, username, password_hash, full_name, monthly_link_limit, plan_tier)
VALUES ('bob@techcorp.in',
       'bobcodes',
       '$2a$12$examplehashforbob9876543210',
       'Bob Patel', 5000, 'pro')
RETURNING id, created_at;

-- User 3: Anonymous-friendly user (no username)
INSERT INTO users
       (email, username, password_hash, full_name, monthly_link_limit, plan_tier)
VALUES ('guest@short.ly',
       NULL,
       '$2a$12$examplehashforguest0000000000',
       'Guest User', 50, 'free')
RETURNING id, created_at;


-- name: minurls
-- Auto-generated slug by Alice
INSERT INTO minurls (slug, url, owner_id, is_custom, expires_at)
VALUES ('gK9pQ2', 'https://www.youtube.com/watch?v=dQw4w9wgxcq',
        '9aa128c0-3e8f-4a95-b0df-3f6ee589c653', FALSE, NULL);

-- Custom slug by Bob
INSERT INTO minurls (slug, url, owner_id, is_custom, expires_at)
VALUES ('my-portfolio', 'https://bobpatel.in',
        '3dd9a426-7839-4ba7-8cb2-2166aa39f529', TRUE, '2027-12-31 23:59:59');

-- Temporary link (expires soon)
INSERT INTO minurls (slug, url, owner_id, is_custom, expires_at)
VALUES ('temp-promo', 'https://example.com/big-sale-2026',
        '9aa128c0-3e8f-4a95-b0df-3f6ee589c653', TRUE, now() + interval '7 days');

-- Anonymous link (no owner)
INSERT INTO minurls (slug, url, owner_id, is_custom, expires_at)
VALUES ('funny-cat', 'https://catmemes.com/funny-video',
        NULL, FALSE, NULL);
