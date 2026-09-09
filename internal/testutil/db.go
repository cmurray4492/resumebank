// Package testutil provides shared test helpers.
package testutil

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"resumebank/internal/db"
)

// OpenTestDB connects to TEST_DATABASE_URL and applies migrations, skipping
// the test if that env var isn't set. Each call truncates all app tables
// first so tests don't see each other's data.
func OpenTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping DB-gated test")
	}

	ctx := context.Background()
	pool, err := db.Open(ctx, url)
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrating test database: %v", err)
	}

	_, err = pool.Exec(ctx, `TRUNCATE candidate_files, jobs, candidates, employers, sessions, users RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncating test database: %v", err)
	}

	return pool
}
