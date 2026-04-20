-- alter the minurls table by adding name
ALTER TABLE minurls
ADD COLUMN name VARCHAR(255);

-- Index for searching by name
CREATE INDEX idx_minurls_name
    ON minurls (name)
    WHERE name IS NOT NULL;

-- Adding names to the data already present in the db
-- Give Alice's Rickroll a proper name
UPDATE minurls
SET name = 'Never Gonna Give You Up'
WHERE slug = 'gK9pQ2';
-- Name Bob's portfolio
UPDATE minurls
SET name = 'Personal Portfolio'
WHERE slug = 'my-portfolio';
-- Name the promo link
UPDATE minurls
SET name = 'Flash Sale - April 2026'
WHERE slug = 'temp-promo';
-- Name the anonymous link
UPDATE minurls
SET name = 'Funny Cat Video'
WHERE slug = 'funny-cat';

-- This ensures your UI always has something to display
SELECT
    slug,
    COALESCE(name, slug) AS display_name,
    url
FROM minurls;
