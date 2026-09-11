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
		&c.Email, &c.Phone, &c.LinkedInURL, &c.Skills, &c.Summary, &c.ResumeHTML, &c.ResumeText,
		&c.PhotoPath, &c.PhotoContentType, &c.CreatedAt, &c.UpdatedAt,
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
	email, phone, linkedin_url, skills, summary, resume_html, resume_text,
	photo_path, photo_content_type, created_at, updated_at`

func (r *CandidateRepo) Create(ctx context.Context, c *models.Candidate) (*models.Candidate, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO candidates (user_id, slug, name, title, city, state, zipcode,
			email, phone, linkedin_url, skills, summary, resume_html, resume_text,
			photo_path, photo_content_type)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		RETURNING `+candidateColumns,
		c.UserID, c.Slug, c.Name, c.Title, c.City, c.State, c.Zipcode,
		c.Email, c.Phone, c.LinkedInURL, c.Skills, c.Summary, c.ResumeHTML, c.ResumeText,
		c.PhotoPath, c.PhotoContentType,
	)
	return scanCandidate(row)
}

func (r *CandidateRepo) Update(ctx context.Context, c *models.Candidate) (*models.Candidate, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE candidates SET
			name = $2, title = $3, city = $4, state = $5, zipcode = $6,
			email = $7, phone = $8, linkedin_url = $9, skills = $10, summary = $11,
			resume_html = $12, resume_text = $13, photo_path = $14, photo_content_type = $15,
			updated_at = now()
		WHERE id = $1
		RETURNING `+candidateColumns,
		c.ID, c.Name, c.Title, c.City, c.State, c.Zipcode,
		c.Email, c.Phone, c.LinkedInURL, c.Skills, c.Summary, c.ResumeHTML, c.ResumeText,
		c.PhotoPath, c.PhotoContentType,
	)
	return scanCandidate(row)
}

// UpdatePhoto sets just the candidate's profile photo, independent of the
// full profile edit form (see CandidateHandlers.UploadPhoto/DeletePhoto).
func (r *CandidateRepo) UpdatePhoto(ctx context.Context, id int64, path, contentType string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE candidates SET photo_path = $1, photo_content_type = $2, updated_at = now() WHERE id = $3`,
		path, contentType, id)
	return err
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
	City       string
	State      string
	Skills     string
	Similarity float64
}

// candidateRankingExpr blends raw cosine similarity with profile recency
// into a single ranking score (higher is better), used only in ORDER BY -
// the Similarity field returned to callers stays the pure cosine similarity
// so the displayed match percentage remains an honest, undiluted number.
// Weights: 85% similarity, 15% recency (profiles updated in the last ~30
// days get a meaningful boost, decaying smoothly for older ones). alias
// must be the table alias (or "" for an unaliased query) prefixing
// "embedding"/"updated_at".
func candidateRankingExpr(alias, vectorParam string) string {
	col := alias
	if col != "" {
		col += "."
	}
	return `(0.85 * (1 - (` + col + `embedding <=> ` + vectorParam + `))
		+ 0.15 * exp(-extract(epoch from (now() - ` + col + `updated_at)) / 86400.0 / 30.0))`
}

// MatchByEmbedding returns the candidates most similar to queryVector (see
// embeddings.FormatVector), ranked by a blend of similarity and profile
// recency (see candidateRankingExpr). location, if non-empty, is matched as
// a case-insensitive substring against the candidate's city or state.
func (r *CandidateRepo) MatchByEmbedding(ctx context.Context, queryVector string, limit int, location string) ([]CandidateMatch, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT slug, name, title, city, state, skills, 1 - (embedding <=> $1::vector) AS similarity
		FROM candidates
		WHERE embedding IS NOT NULL
			AND ($3 = '' OR city ILIKE '%' || $3 || '%' OR state ILIKE '%' || $3 || '%')
		ORDER BY `+candidateRankingExpr("", "$1::vector")+` DESC
		LIMIT $2`, queryVector, limit, location)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []CandidateMatch
	for rows.Next() {
		var m CandidateMatch
		if err := rows.Scan(&m.Slug, &m.Name, &m.Title, &m.City, &m.State, &m.Skills, &m.Similarity); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

// RecommendedForJob returns the candidates most similar to jobID's own
// description embedding, ranked by a blend of similarity and profile
// recency (see candidateRankingExpr), for the automatic "Recommended
// Candidates" panel shown to a job's owner - no pasted text required.
// Returns an empty slice (not an error) if either embedding isn't computed
// yet.
func (r *CandidateRepo) RecommendedForJob(ctx context.Context, jobID int64, limit int) ([]CandidateMatch, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT c.slug, c.name, c.title, c.city, c.state, c.skills, 1 - (c.embedding <=> j.embedding) AS similarity
		FROM candidates c, jobs j
		WHERE j.id = $1 AND c.embedding IS NOT NULL AND j.embedding IS NOT NULL
		ORDER BY `+candidateRankingExpr("c", "j.embedding")+` DESC
		LIMIT $2`, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []CandidateMatch
	for rows.Next() {
		var m CandidateMatch
		if err := rows.Scan(&m.Slug, &m.Name, &m.Title, &m.City, &m.State, &m.Skills, &m.Similarity); err != nil {
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
			&c.Email, &c.Phone, &c.LinkedInURL, &c.Skills, &c.Summary, &c.ResumeHTML, &c.ResumeText,
			&c.PhotoPath, &c.PhotoContentType, &c.CreatedAt, &c.UpdatedAt, &rank,
		); err != nil {
			return nil, 0, err
		}
		results = append(results, CandidateSearchResult{Candidate: c, Rank: rank})
	}
	return results, total, rows.Err()
}
