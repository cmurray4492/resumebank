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

func (r *CandidateRepo) GetByID(ctx context.Context, id int64) (*models.Candidate, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+candidateColumns+` FROM candidates WHERE id = $1`, id)
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

// SetEmbedding stores a pgvector literal (see embeddings.FormatVector) as
// the candidate's resume embedding.
func (r *CandidateRepo) SetEmbedding(ctx context.Context, id int64, vector string) error {
	_, err := r.pool.Exec(ctx, `UPDATE candidates SET embedding = $1::vector WHERE id = $2`, vector, id)
	return err
}

// MissingEmbeddings returns up to limit candidates whose embedding hasn't
// been computed yet, for the background backfill sweep.
func (r *CandidateRepo) MissingEmbeddings(ctx context.Context, limit int) ([]EmbeddingTarget, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, resume_text FROM candidates WHERE embedding IS NULL LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var targets []EmbeddingTarget
	for rows.Next() {
		var t EmbeddingTarget
		if err := rows.Scan(&t.ID, &t.Text); err != nil {
			return nil, err
		}
		targets = append(targets, t)
	}
	return targets, rows.Err()
}

type CandidateMatch struct {
	Slug       string
	Name       string
	Title      string
	Similarity float64
}

// MatchByEmbedding returns the candidates most similar to queryVector
// (see embeddings.FormatVector), most similar first.
func (r *CandidateRepo) MatchByEmbedding(ctx context.Context, queryVector string, limit int) ([]CandidateMatch, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT slug, name, title, 1 - (embedding <=> $1::vector) AS similarity
		FROM candidates
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> $1::vector
		LIMIT $2`, queryVector, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []CandidateMatch
	for rows.Next() {
		var m CandidateMatch
		if err := rows.Scan(&m.Slug, &m.Name, &m.Title, &m.Similarity); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

// ListSlugs returns every candidate's slug and last-updated time, for
// building the sitemap.
func (r *CandidateRepo) ListSlugs(ctx context.Context) ([]SitemapEntry, error) {
	rows, err := r.pool.Query(ctx, `SELECT slug, updated_at FROM candidates ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []SitemapEntry
	for rows.Next() {
		var e SitemapEntry
		if err := rows.Scan(&e.Slug, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
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
