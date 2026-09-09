-- Unified candidates+jobs search index, refreshed hourly by a background
-- job (internal/app RefreshSearchIndex). This exists as the literal
-- "search index regenerated hourly" artifact from the spec; the live
-- candidate/job search pages intentionally query the always-fresh base
-- tables instead of this view, so new signups/postings show up in search
-- immediately rather than waiting up to an hour. A unique index on
-- (entity_type, entity_id) is required for REFRESH ... CONCURRENTLY, which
-- lets the refresh run without blocking concurrent reads of the view.
CREATE MATERIALIZED VIEW search_index AS
SELECT
    'candidate'::text AS entity_type,
    c.id AS entity_id,
    c.slug,
    c.name AS title,
    coalesce(c.title, '') AS subtitle,
    c.search_vector,
    c.updated_at
FROM candidates c
UNION ALL
SELECT
    'job'::text AS entity_type,
    j.id AS entity_id,
    j.slug,
    j.title,
    e.company_name AS subtitle,
    j.search_vector,
    j.updated_at
FROM jobs j
JOIN employers e ON e.id = j.employer_id;

CREATE UNIQUE INDEX idx_search_index_entity ON search_index(entity_type, entity_id);
CREATE INDEX idx_search_index_vector ON search_index USING GIN (search_vector);
