CREATE TABLE candidate_files (
    id                BIGSERIAL PRIMARY KEY,
    candidate_id      BIGINT NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    kind              TEXT NOT NULL CHECK (kind IN ('resume_pdf', 'additional')),
    original_filename TEXT NOT NULL,
    stored_path       TEXT NOT NULL,
    content_type      TEXT NOT NULL,
    size_bytes        BIGINT NOT NULL,
    position          SMALLINT NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Enforce exactly one PDF resume per candidate at the DB level; the cap of
-- three "additional" files is enforced in the application layer.
CREATE UNIQUE INDEX idx_candidate_files_one_resume
    ON candidate_files(candidate_id)
    WHERE kind = 'resume_pdf';

CREATE INDEX idx_candidate_files_candidate_id ON candidate_files(candidate_id);
