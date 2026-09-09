package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const passwordResetTTL = time.Hour

var ErrPasswordResetTokenInvalid = errors.New("password reset token not found, expired, or already used")

type PasswordResetStore struct {
	pool *pgxpool.Pool
}

func NewPasswordResetStore(pool *pgxpool.Pool) *PasswordResetStore {
	return &PasswordResetStore{pool: pool}
}

// Create issues a new password reset token for userID and returns the raw
// token to be embedded in the reset-link email. Only its hash is persisted.
func (s *PasswordResetStore) Create(ctx context.Context, userID int64) (rawToken string, err error) {
	rawToken, err = generateToken()
	if err != nil {
		return "", err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO password_reset_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashToken(rawToken), userID, time.Now().Add(passwordResetTTL),
	)
	if err != nil {
		return "", err
	}
	return rawToken, nil
}

// Consume validates rawToken (unexpired, not already used) and atomically
// marks it used, returning the user it belongs to. Safe to call
// concurrently: only one caller can ever successfully consume a given
// token, since the UPDATE's WHERE clause excludes already-used rows.
func (s *PasswordResetStore) Consume(ctx context.Context, rawToken string) (int64, error) {
	var userID int64
	err := s.pool.QueryRow(ctx, `
		UPDATE password_reset_tokens SET used_at = now()
		WHERE token_hash = $1 AND expires_at > now() AND used_at IS NULL
		RETURNING user_id`,
		hashToken(rawToken),
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrPasswordResetTokenInvalid
	}
	if err != nil {
		return 0, err
	}
	return userID, nil
}
