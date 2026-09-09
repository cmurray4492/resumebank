package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"resumebank/internal/models"
)

type BlogRepo struct {
	pool *pgxpool.Pool
}

func NewBlogRepo(pool *pgxpool.Pool) *BlogRepo {
	return &BlogRepo{pool: pool}
}

const blogColumns = `id, slug, title, body_html, body_text, author_name,
	published, published_at, created_at, updated_at`

func scanBlogPost(row pgx.Row) (*models.BlogPost, error) {
	p := &models.BlogPost{}
	err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.BodyHTML, &p.BodyText, &p.AuthorName,
		&p.Published, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *BlogRepo) Create(ctx context.Context, p *models.BlogPost) (*models.BlogPost, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO blog_posts (slug, title, body_html, body_text, author_name, published, published_at)
		VALUES ($1,$2,$3,$4,$5,$6, CASE WHEN $6 THEN now() ELSE NULL END)
		RETURNING `+blogColumns,
		p.Slug, p.Title, p.BodyHTML, p.BodyText, p.AuthorName, p.Published,
	)
	return scanBlogPost(row)
}

func (r *BlogRepo) Update(ctx context.Context, p *models.BlogPost, wasPublished bool) (*models.BlogPost, error) {
	// Set published_at the first time a post transitions to published;
	// leave it alone on subsequent edits so it reflects original publish date.
	setPublishedAt := p.Published && !wasPublished
	row := r.pool.QueryRow(ctx, `
		UPDATE blog_posts SET
			title = $2, body_html = $3, body_text = $4, author_name = $5, published = $6,
			published_at = CASE WHEN $7 THEN now() ELSE published_at END,
			updated_at = now()
		WHERE id = $1
		RETURNING `+blogColumns,
		p.ID, p.Title, p.BodyHTML, p.BodyText, p.AuthorName, p.Published, setPublishedAt,
	)
	return scanBlogPost(row)
}

func (r *BlogRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM blog_posts WHERE id = $1`, id)
	return err
}

func (r *BlogRepo) GetByID(ctx context.Context, id int64) (*models.BlogPost, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+blogColumns+` FROM blog_posts WHERE id = $1`, id)
	return scanBlogPost(row)
}

func (r *BlogRepo) GetBySlug(ctx context.Context, slug string) (*models.BlogPost, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+blogColumns+` FROM blog_posts WHERE slug = $1`, slug)
	return scanBlogPost(row)
}

// GetPublishedBySlug is like GetBySlug but only returns a post that's
// currently published, for the public-facing blog.
func (r *BlogRepo) GetPublishedBySlug(ctx context.Context, slug string) (*models.BlogPost, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+blogColumns+` FROM blog_posts WHERE slug = $1 AND published`, slug)
	return scanBlogPost(row)
}

func (r *BlogRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM blog_posts WHERE slug = $1)`, slug).Scan(&exists)
	return exists, err
}

// ListPublished returns published posts, most recently published first, for
// the public /blog listing.
func (r *BlogRepo) ListPublished(ctx context.Context, limit, offset int) ([]models.BlogPost, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM blog_posts WHERE published`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+blogColumns+` FROM blog_posts
		WHERE published ORDER BY published_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	posts, err := scanBlogPosts(rows)
	return posts, total, err
}

// ListAll returns every post regardless of published status, most recently
// updated first, for the admin panel.
func (r *BlogRepo) ListAll(ctx context.Context, limit, offset int) ([]models.BlogPost, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM blog_posts`).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx, `
		SELECT `+blogColumns+` FROM blog_posts
		ORDER BY updated_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	posts, err := scanBlogPosts(rows)
	return posts, total, err
}

// ListPublishedSlugs returns every published post's slug and publish time,
// for building the sitemap.
func (r *BlogRepo) ListPublishedSlugs(ctx context.Context) ([]SitemapEntry, error) {
	rows, err := r.pool.Query(ctx, `SELECT slug, updated_at FROM blog_posts WHERE published ORDER BY id`)
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

func scanBlogPosts(rows pgx.Rows) ([]models.BlogPost, error) {
	var posts []models.BlogPost
	for rows.Next() {
		var p models.BlogPost
		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.BodyHTML, &p.BodyText, &p.AuthorName,
			&p.Published, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		posts = append(posts, p)
	}
	return posts, rows.Err()
}
