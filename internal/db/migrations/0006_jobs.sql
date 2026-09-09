CREATE TABLE jobs (
    id               BIGSERIAL PRIMARY KEY,
    employer_id      BIGINT NOT NULL REFERENCES employers(id) ON DELETE CASCADE,
    slug             TEXT NOT NULL UNIQUE,
    title            TEXT NOT NULL,
    location         TEXT NOT NULL DEFAULT '',
    job_number       TEXT NOT NULL DEFAULT '',
    salary_min       INTEGER,
    salary_max       INTEGER,
    description_html TEXT NOT NULL,
    description_text TEXT NOT NULL,
    date_posted      TIMESTAMPTZ NOT NULL DEFAULT now(),
    search_vector TSVECTOR GENERATED ALWAYS AS (
        to_tsvector('english',
            coalesce(title, '') || ' ' ||
            coalesce(location, '') || ' ' ||
            coalesce(description_text, '')
        )
    ) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_jobs_search_vector ON jobs USING GIN (search_vector);
CREATE INDEX idx_jobs_employer_id ON jobs(employer_id);

-- Later phase: embedding vector(768) column; job_votes(candidate_id, job_id, vote) table;
-- messages(sender_id, recipient_id, body, ...) table.
