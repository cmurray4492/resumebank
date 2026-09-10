package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"resumebank/internal/models"
)

type JobRepo struct {
	pool *pgxpool.Pool
}

func NewJobRepo(pool *pgxpool.Pool) *JobRepo {
	return &JobRepo{pool: pool}
}

const jobColumns = `id, employer_id, slug, title, location, job_number, salary_min, salary_max,
	description_html, description_text, apply_method, apply_value, date_posted, created_at, updated_at`

func scanJob(row pgx.Row) (*models.Job, error) {
	j := &models.Job{}
	err := row.Scan(
		&j.ID, &j.EmployerID, &j.Slug, &j.Title, &j.Location, &j.JobNumber,
		&j.SalaryMin, &j.SalaryMax, &j.DescriptionHTML, &j.DescriptionText,
		&j.ApplyMethod, &j.ApplyValue, &j.DatePosted, &j.CreatedAt, &j.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return j, nil
}

func (r *JobRepo) Create(ctx context.Context, j *models.Job) (*models.Job, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO jobs (employer_id, slug, title, location, job_number, salary_min, salary_max,
			description_html, description_text, apply_method, apply_value)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING `+jobColumns,
		j.EmployerID, j.Slug, j.Title, j.Location, j.JobNumber, j.SalaryMin, j.SalaryMax,
		j.DescriptionHTML, j.DescriptionText, j.ApplyMethod, j.ApplyValue,
	)
	return scanJob(row)
}

func (r *JobRepo) Update(ctx context.Context, j *models.Job) (*models.Job, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE jobs SET
			title = $2, location = $3, job_number = $4, salary_min = $5, salary_max = $6,
			description_html = $7, description_text = $8, apply_method = $9, apply_value = $10, updated_at = now()
		WHERE id = $1
		RETURNING `+jobColumns,
		j.ID, j.Title, j.Location, j.JobNumber, j.SalaryMin, j.SalaryMax,
		j.DescriptionHTML, j.DescriptionText, j.ApplyMethod, j.ApplyValue,
	)
	return scanJob(row)
}

func (r *JobRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM jobs WHERE id = $1`, id)
	return err
}

func (r *JobRepo) GetByID(ctx context.Context, id int64) (*models.Job, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = $1`, id)
	return scanJob(row)
}

func (r *JobRepo) GetBySlug(ctx context.Context, slug string) (*models.Job, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+jobColumns+` FROM jobs WHERE slug = $1`, slug)
	return scanJob(row)
}

func (r *JobRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM jobs WHERE slug = $1)`, slug).Scan(&exists)
	return exists, err
}

// SetEmbedding stores a pgvector literal (see embeddings.FormatVector) as
// the job's description embedding.
func (r *JobRepo) SetEmbedding(ctx context.Context, id int64, vector string) error {
	_, err := r.pool.Exec(ctx, `UPDATE jobs SET embedding = $1::vector WHERE id = $2`, vector, id)
	return err
}

// MissingEmbeddings returns up to limit jobs whose embedding hasn't been
// computed yet, for the background backfill sweep.
func (r *JobRepo) MissingEmbeddings(ctx context.Context, limit int) ([]EmbeddingTarget, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, description_text FROM jobs WHERE embedding IS NULL LIMIT $1`, limit)
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

type JobMatch struct {
	Slug         string
	Title        string
	CompanyName  string
	EmployerSlug string
	Similarity   float64
}

// MatchByEmbedding returns the jobs most similar to queryVector (see
// embeddings.FormatVector), most similar first.
func (r *JobRepo) MatchByEmbedding(ctx context.Context, queryVector string, limit int) ([]JobMatch, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT j.slug, j.title, e.company_name, e.slug, 1 - (j.embedding <=> $1::vector) AS similarity
		FROM jobs j JOIN employers e ON e.id = j.employer_id
		WHERE j.embedding IS NOT NULL
		ORDER BY j.embedding <=> $1::vector
		LIMIT $2`, queryVector, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []JobMatch
	for rows.Next() {
		var m JobMatch
		if err := rows.Scan(&m.Slug, &m.Title, &m.CompanyName, &m.EmployerSlug, &m.Similarity); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

// ListSlugs returns every job's slug and last-updated time, for building
// the sitemap.
func (r *JobRepo) ListSlugs(ctx context.Context) ([]SitemapEntry, error) {
	rows, err := r.pool.Query(ctx, `SELECT slug, updated_at FROM jobs ORDER BY id`)
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

func (r *JobRepo) ListByEmployer(ctx context.Context, employerID int64) ([]models.Job, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+jobColumns+` FROM jobs WHERE employer_id = $1 ORDER BY date_posted DESC`, employerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var jobs []models.Job
	for rows.Next() {
		j, err := scanJobRows(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *j)
	}
	return jobs, rows.Err()
}

func scanJobRows(rows pgx.Rows) (*models.Job, error) {
	j := &models.Job{}
	err := rows.Scan(
		&j.ID, &j.EmployerID, &j.Slug, &j.Title, &j.Location, &j.JobNumber,
		&j.SalaryMin, &j.SalaryMax, &j.DescriptionHTML, &j.DescriptionText,
		&j.ApplyMethod, &j.ApplyValue, &j.DatePosted, &j.CreatedAt, &j.UpdatedAt,
	)
	return j, err
}

type JobSearchResult struct {
	Job          models.Job
	CompanyName  string
	EmployerSlug string
	Rank         float32
}

func (r *JobRepo) Search(ctx context.Context, query string, limit, offset int) ([]JobSearchResult, int, error) {
	var rows pgx.Rows
	var err error
	var total int

	if query == "" {
		if err = r.pool.QueryRow(ctx, `SELECT count(*) FROM jobs`).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err = r.pool.Query(ctx, `
			SELECT `+jobColumnsAliased()+`, e.company_name, e.slug, 0 AS rank
			FROM jobs j JOIN employers e ON e.id = j.employer_id
			ORDER BY j.date_posted DESC LIMIT $1 OFFSET $2`, limit, offset)
	} else {
		if err = r.pool.QueryRow(ctx,
			`SELECT count(*) FROM jobs WHERE search_vector @@ websearch_to_tsquery('english', $1)`, query,
		).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err = r.pool.Query(ctx, `
			SELECT `+jobColumnsAliased()+`, e.company_name, e.slug,
				ts_rank(j.search_vector, websearch_to_tsquery('english', $1)) AS rank
			FROM jobs j JOIN employers e ON e.id = j.employer_id
			WHERE j.search_vector @@ websearch_to_tsquery('english', $1)
			ORDER BY rank DESC, j.date_posted DESC
			LIMIT $2 OFFSET $3`, query, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var results []JobSearchResult
	for rows.Next() {
		var j models.Job
		var rank float32
		var companyName, employerSlug string
		if err := rows.Scan(
			&j.ID, &j.EmployerID, &j.Slug, &j.Title, &j.Location, &j.JobNumber,
			&j.SalaryMin, &j.SalaryMax, &j.DescriptionHTML, &j.DescriptionText,
			&j.ApplyMethod, &j.ApplyValue, &j.DatePosted, &j.CreatedAt, &j.UpdatedAt,
			&companyName, &employerSlug, &rank,
		); err != nil {
			return nil, 0, err
		}
		results = append(results, JobSearchResult{Job: j, CompanyName: companyName, EmployerSlug: employerSlug, Rank: rank})
	}
	return results, total, rows.Err()
}

// jobColumnsAliased mirrors jobColumns but with every column individually
// prefixed "j.", since Search joins against employers (which also has
// slug/created_at/updated_at columns that would otherwise be ambiguous).
func jobColumnsAliased() string {
	return `j.id, j.employer_id, j.slug, j.title, j.location, j.job_number, j.salary_min, j.salary_max,
		j.description_html, j.description_text, j.apply_method, j.apply_value, j.date_posted, j.created_at, j.updated_at`
}
