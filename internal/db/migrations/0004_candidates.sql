CREATE TABLE candidates (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    slug         TEXT NOT NULL UNIQUE,
    name         TEXT NOT NULL,
    title        TEXT NOT NULL DEFAULT '',
    city         TEXT NOT NULL DEFAULT '',
    state        TEXT NOT NULL DEFAULT '',
    zipcode      TEXT NOT NULL,
    email        TEXT NOT NULL,
    linkedin_url TEXT NOT NULL DEFAULT '',
    skills       TEXT NOT NULL DEFAULT '',
    summary      TEXT NOT NULL DEFAULT '',
    resume_html  TEXT NOT NULL,
    resume_text  TEXT NOT NULL,
    search_vector TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('english',
            coalesce(name, '') || ' ' ||
            coalesce(title, '') || ' ' ||
            coalesce(skills, '') || ' ' ||
            coalesce(summary, '') || ' ' ||
            coalesce(resume_text, '')
        )
    ) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_candidates_search_vector ON candidates USING GIN (search_vector);

-- Later phase: ALTER TABLE candidates ADD COLUMN embedding vector(768);
