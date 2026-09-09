package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sessionTTL = 30 * 24 * time.Hour

var ErrSessionNotFound = errors.New("session not found or expired")

type SessionStore struct {
	pool *pgxpool.Pool
}

func NewSessionStore(pool *pgxpool.Pool) *SessionStore {
	return &SessionStore{pool: pool}
}

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create issues a new session for userID and returns the raw token to be
// stored in the client's cookie. Only its hash is persisted.
func (s *SessionStore) Create(ctx context.Context, userID int64) (rawToken string, err error) {
	rawToken, err = generateToken()
	if err != nil {
		return "", err
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashToken(rawToken), userID, time.Now().Add(sessionTTL),
	)
	if err != nil {
		return "", err
	}
	return rawToken, nil
}

// Lookup resolves a raw token from a cookie to a user ID, if the session
// exists and has not expired.
func (s *SessionStore) Lookup(ctx context.Context, rawToken string) (int64, error) {
	var userID int64
	err := s.pool.QueryRow(ctx,
		`SELECT user_id FROM sessions WHERE token_hash = $1 AND expires_at > now()`,
		hashToken(rawToken),
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrSessionNotFound
	}
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (s *SessionStore) Delete(ctx context.Context, rawToken string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, hashToken(rawToken))
	return err
}
