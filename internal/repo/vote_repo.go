package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VoteRepo struct {
	pool *pgxpool.Pool
}

func NewVoteRepo(pool *pgxpool.Pool) *VoteRepo {
	return &VoteRepo{pool: pool}
}

// Set upserts a candidate's vote on a job to the given direction (+1 or -1).
func (r *VoteRepo) Set(ctx context.Context, candidateID, jobID int64, vote int16) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO job_votes (candidate_id, job_id, vote)
		VALUES ($1, $2, $3)
		ON CONFLICT (candidate_id, job_id) DO UPDATE SET vote = EXCLUDED.vote, updated_at = now()`,
		candidateID, jobID, vote,
	)
	return err
}

// Clear removes a candidate's vote on a job, if any.
func (r *VoteRepo) Clear(ctx context.Context, candidateID, jobID int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM job_votes WHERE candidate_id = $1 AND job_id = $2`, candidateID, jobID)
	return err
}

// GetVote returns the candidate's current vote on a job (+1, -1), or 0 if
// they haven't voted.
func (r *VoteRepo) GetVote(ctx context.Context, candidateID, jobID int64) (int16, error) {
	var vote int16
	err := r.pool.QueryRow(ctx,
		`SELECT vote FROM job_votes WHERE candidate_id = $1 AND job_id = $2`, candidateID, jobID,
	).Scan(&vote)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return vote, nil
}

// Counts returns the number of up-votes and down-votes on a job.
func (r *VoteRepo) Counts(ctx context.Context, jobID int64) (up, down int, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE vote = 1),
			count(*) FILTER (WHERE vote = -1)
		FROM job_votes WHERE job_id = $1`, jobID,
	).Scan(&up, &down)
	return up, down, err
}
