-- Candidate-only thumbs up/down voting on jobs. One vote per candidate per
-- job; voting the same direction again clears the vote (toggle), enforced
-- in the application layer.
CREATE TABLE job_votes (
    candidate_id BIGINT NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    job_id       BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    vote         SMALLINT NOT NULL CHECK (vote IN (-1, 1)),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (candidate_id, job_id)
);

CREATE INDEX idx_job_votes_job_id ON job_votes(job_id);
