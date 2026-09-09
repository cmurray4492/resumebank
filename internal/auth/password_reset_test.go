package auth_test

import (
	"context"
	"testing"

	"resumebank/internal/auth"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/testutil"
)

func TestPasswordResetStore_CreateConsume(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	resets := auth.NewPasswordResetStore(pool)
	ctx := context.Background()

	u, err := users.Create(ctx, "jane@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}

	token, err := resets.Create(ctx, u.ID)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty raw token")
	}

	userID, err := resets.Consume(ctx, token)
	if err != nil {
		t.Fatalf("Consume: %v", err)
	}
	if userID != u.ID {
		t.Errorf("got user id %d, want %d", userID, u.ID)
	}

	// A token can only be consumed once.
	if _, err := resets.Consume(ctx, token); err != auth.ErrPasswordResetTokenInvalid {
		t.Errorf("expected ErrPasswordResetTokenInvalid on reuse, got %v", err)
	}
}

func TestPasswordResetStore_ConsumeUnknownToken(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	resets := auth.NewPasswordResetStore(pool)

	if _, err := resets.Consume(context.Background(), "does-not-exist"); err != auth.ErrPasswordResetTokenInvalid {
		t.Errorf("expected ErrPasswordResetTokenInvalid, got %v", err)
	}
}
