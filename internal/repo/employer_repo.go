package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"resumebank/internal/models"
)

type EmployerRepo struct {
	pool *pgxpool.Pool
}

func NewEmployerRepo(pool *pgxpool.Pool) *EmployerRepo {
	return &EmployerRepo{pool: pool}
}

const employerColumns = `id, user_id, slug, company_name, industry, city, state, zipcode,
	phone, email_address, website, description, locations, created_at, updated_at`

func scanEmployer(row pgx.Row) (*models.Employer, error) {
	e := &models.Employer{}
	err := row.Scan(
		&e.ID, &e.UserID, &e.Slug, &e.CompanyName, &e.Industry, &e.City, &e.State, &e.Zipcode,
		&e.Phone, &e.EmailAddress, &e.Website, &e.Description, &e.Locations,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (r *EmployerRepo) Create(ctx context.Context, e *models.Employer) (*models.Employer, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO employers (user_id, slug, company_name, industry, city, state, zipcode,
			phone, email_address, website, description, locations)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		RETURNING `+employerColumns,
		e.UserID, e.Slug, e.CompanyName, e.Industry, e.City, e.State, e.Zipcode,
		e.Phone, e.EmailAddress, e.Website, e.Description, e.Locations,
	)
	return scanEmployer(row)
}

func (r *EmployerRepo) Update(ctx context.Context, e *models.Employer) (*models.Employer, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE employers SET
			company_name = $2, industry = $3, city = $4, state = $5, zipcode = $6,
			phone = $7, email_address = $8, website = $9, description = $10, locations = $11,
			updated_at = now()
		WHERE id = $1
		RETURNING `+employerColumns,
		e.ID, e.CompanyName, e.Industry, e.City, e.State, e.Zipcode,
		e.Phone, e.EmailAddress, e.Website, e.Description, e.Locations,
	)
	return scanEmployer(row)
}

func (r *EmployerRepo) GetBySlug(ctx context.Context, slug string) (*models.Employer, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+employerColumns+` FROM employers WHERE slug = $1`, slug)
	return scanEmployer(row)
}

func (r *EmployerRepo) GetByID(ctx context.Context, id int64) (*models.Employer, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+employerColumns+` FROM employers WHERE id = $1`, id)
	return scanEmployer(row)
}

func (r *EmployerRepo) GetByUserID(ctx context.Context, userID int64) (*models.Employer, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+employerColumns+` FROM employers WHERE user_id = $1`, userID)
	return scanEmployer(row)
}

func (r *EmployerRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM employers WHERE slug = $1)`, slug).Scan(&exists)
	return exists, err
}

// ListAll returns employers most-recently-created first, for the admin panel.
func (r *EmployerRepo) ListAll(ctx context.Context, limit, offset int) ([]models.Employer, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM employers`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `SELECT `+employerColumns+` FROM employers ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var employers []models.Employer
	for rows.Next() {
		var e models.Employer
		if err := rows.Scan(&e.ID, &e.UserID, &e.Slug, &e.CompanyName, &e.Industry, &e.City, &e.State, &e.Zipcode,
			&e.Phone, &e.EmailAddress, &e.Website, &e.Description, &e.Locations, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, 0, err
		}
		employers = append(employers, e)
	}
	return employers, total, rows.Err()
}

// ListSlugs returns every employer's slug and last-updated time, for
// building the sitemap.
func (r *EmployerRepo) ListSlugs(ctx context.Context) ([]SitemapEntry, error) {
	rows, err := r.pool.Query(ctx, `SELECT slug, updated_at FROM employers ORDER BY id`)
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
