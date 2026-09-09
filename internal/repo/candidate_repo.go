package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"resumebank/internal/models"
)

type CandidateRepo struct {
	pool *pgxpool.Pool
}

func NewCandidateRepo(pool *pgxpool.Pool) *CandidateRepo {
	return &CandidateRepo{pool: pool}
}

func scanCandidate(row pgx.Row) (*models.Candidate, error) {
	c := &models.Candidate{}
	err := row.Scan(
		&c.ID, &c.UserID, &c.Slug, &c.Name, &c.Title, &c.City, &c.State, &c.Zipcode,
		&c.Email, &c.LinkedInURL, &c.Skills, &c.Summary, &c.ResumeHTML, &c.ResumeText,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

const candidateColumns = `id, user_id, slug, name, title, city, state, zipcode,
	email, linkedin_url, skills, summary, resume_html, resume_text, created_at, updated_at`

func (r *CandidateRepo) Create(ctx context.Context, c *models.Candidate) (*models.Candidate, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO candidates (user_id, slug, name, title, city, state, zipcode,
			email, linkedin_url, skills, summary, resume_html, resume_text)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		RETURNING `+candidateColumns,
		c.UserID, c.Slug, c.Name, c.Title, c.City, c.State, c.Zipcode,
		c.Email, c.LinkedInURL, c.Skills, c.Summary, c.ResumeHTML, c.ResumeText,
	)
	return scanCandidate(row)
}

func (r *CandidateRepo) Update(ctx context.Context, c *models.Candidate) (*models.Candidate, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE candidates SET
			name = $2, title = $3, city = $4, state = $5, zipcode = $6,
			email = $7, linkedin_url = $8, skills = $9, summary = $10,
			resume_html = $11, resume_text = $12, updated_at = now()
		WHERE id = $1
		RETURNING `+candidateColumns,
		c.ID, c.Name, c.Title, c.City, c.State, c.Zipcode,
		c.Email, c.LinkedInURL, c.Skills, c.Summary, c.ResumeHTML, c.ResumeText,
	)
	return scanCandidate(row)
}

func (r *CandidateRepo) GetBySlug(ctx context.Context, slug string) (*models.Candidate, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+candidateColumns+` FROM candidates WHERE slug = $1`, slug)
	return scanCandidate(row)
}

func (r *CandidateRepo) GetByUserID(ctx context.Context, userID int64) (*models.Candidate, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+candidateColumns+` FROM candidates WHERE user_id = $1`, userID)
	return scanCandidate(row)
}

// SlugExists is used to find a unique slug before insert.
func (r *CandidateRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM candidates WHERE slug = $1)`, slug).Scan(&exists)
	return exists, err
}

type CandidateSearchResult struct {
	Candidate models.Candidate
	Rank      float32
}

func (r *CandidateRepo) Search(ctx context.Context, query string, limit, offset int) ([]CandidateSearchResult, int, error) {
	var rows pgx.Rows
	var err error
	var total int

	if query == "" {
		err = r.pool.QueryRow(ctx, `SELECT count(*) FROM candidates`).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		rows, err = r.pool.Query(ctx, `
			SELECT `+candidateColumns+`, 0 AS rank FROM candidates
			ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	} else {
		err = r.pool.QueryRow(ctx,
			`SELECT count(*) FROM candidates WHERE search_vector @@ websearch_to_tsquery('english', $1)`, query,
		).Scan(&total)
		if err != nil {
			return nil, 0, err
		}
		rows, err = r.pool.Query(ctx, `
			SELECT `+candidateColumns+`,
				ts_rank(search_vector, websearch_to_tsquery('english', $1)) AS rank
			FROM candidates
			WHERE search_vector @@ websearch_to_tsquery('english', $1)
			ORDER BY rank DESC, created_at DESC
			LIMIT $2 OFFSET $3`, query, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []CandidateSearchResult
	for rows.Next() {
		var c models.Candidate
		var rank float32
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.Slug, &c.Name, &c.Title, &c.City, &c.State, &c.Zipcode,
			&c.Email, &c.LinkedInURL, &c.Skills, &c.Summary, &c.ResumeHTML, &c.ResumeText,
			&c.CreatedAt, &c.UpdatedAt, &rank,
		); err != nil {
			return nil, 0, err
		}
		results = append(results, CandidateSearchResult{Candidate: c, Rank: rank})
	}
	return results, total, rows.Err()
}
