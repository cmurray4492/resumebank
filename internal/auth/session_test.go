package auth_test

import (
	"context"
	"testing"

	"resumebank/internal/auth"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/testutil"
)

func TestSessionStore_CreateLookupDelete(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	sessions := auth.NewSessionStore(pool)
	ctx := context.Background()

	u, err := users.Create(ctx, "jane@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}

	token, err := sessions.Create(ctx, u.ID)
	if err != nil {
		t.Fatalf("Create session: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty raw token")
	}

	gotUserID, err := sessions.Lookup(ctx, token)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if gotUserID != u.ID {
		t.Errorf("got user id %d, want %d", gotUserID, u.ID)
	}

	if err := sessions.Delete(ctx, token); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := sessions.Lookup(ctx, token); err != auth.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestSessionStore_LookupUnknownToken(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	sessions := auth.NewSessionStore(pool)

	if _, err := sessions.Lookup(context.Background(), "does-not-exist"); err != auth.ErrSessionNotFound {
		t.Errorf("expected ErrSessionNotFound, got %v", err)
	}
}
