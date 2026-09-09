// Package repo implements SQL access for each domain aggregate.
package repo

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"resumebank/internal/models"
)

var ErrNotFound = errors.New("not found")

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, email, passwordHash string, role models.Role) (*models.User, error) {
	u := &models.User{Email: email, PasswordHash: passwordHash, Role: role}
	err := r.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role) VALUES ($1, $2, $3)
		 RETURNING id, created_at, updated_at`,
		email, passwordHash, role,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	u := &models.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, role, created_at, updated_at, last_login_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) GetUserByID(ctx context.Context, id int64) (*models.User, error) {
	u := &models.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, role, created_at, updated_at, last_login_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt, &u.LastLoginAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, id)
	return err
}

func (r *UserRepo) SetPassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $1, updated_at = now() WHERE id = $2`, passwordHash, id)
	return err
}
