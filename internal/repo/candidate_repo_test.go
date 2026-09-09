package repo_test

import (
	"context"
	"testing"

	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/testutil"
)

func newTestUser(t *testing.T, users *repo.UserRepo, email string, role models.Role) *models.User {
	t.Helper()
	u, err := users.Create(context.Background(), email, "hash", role)
	if err != nil {
		t.Fatalf("creating test user: %v", err)
	}
	return u
}

func TestCandidateRepo_CreateGetUpdate(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	candidates := repo.NewCandidateRepo(pool)
	ctx := context.Background()

	u := newTestUser(t, users, "jane@example.com", models.RoleCandidate)

	c := &models.Candidate{
		UserID: u.ID, Slug: "jane-doe", Name: "Jane Doe", Zipcode: "12345",
		Email: "jane@example.com", ResumeHTML: "<p>Resume</p>", ResumeText: "Resume",
	}
	created, err := candidates.Create(ctx, c)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.ID == 0 {
		t.Error("expected a non-zero ID")
	}

	fetched, err := candidates.GetBySlug(ctx, "jane-doe")
	if err != nil {
		t.Fatalf("GetBySlug: %v", err)
	}
	if fetched.Name != "Jane Doe" {
		t.Errorf("got name %q, want Jane Doe", fetched.Name)
	}

	fetched.Name = "Jane Q. Doe"
	updated, err := candidates.Update(ctx, fetched)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "Jane Q. Doe" {
		t.Errorf("got name %q after update, want Jane Q. Doe", updated.Name)
	}

	if _, err := candidates.GetBySlug(ctx, "does-not-exist"); err != repo.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCandidateRepo_SlugExists(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	candidates := repo.NewCandidateRepo(pool)
	ctx := context.Background()

	u := newTestUser(t, users, "jane@example.com", models.RoleCandidate)
	_, err := candidates.Create(ctx, &models.Candidate{
		UserID: u.ID, Slug: "jane-doe", Name: "Jane Doe", Zipcode: "12345",
		Email: "jane@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	exists, err := candidates.SlugExists(ctx, "jane-doe")
	if err != nil || !exists {
		t.Errorf("expected jane-doe to exist, got exists=%v err=%v", exists, err)
	}
	exists, err = candidates.SlugExists(ctx, "jane-doe-2")
	if err != nil || exists {
		t.Errorf("expected jane-doe-2 to not exist, got exists=%v err=%v", exists, err)
	}
}

func TestCandidateRepo_Search(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	candidates := repo.NewCandidateRepo(pool)
	ctx := context.Background()

	u1 := newTestUser(t, users, "gopher@example.com", models.RoleCandidate)
	u2 := newTestUser(t, users, "painter@example.com", models.RoleCandidate)

	if _, err := candidates.Create(ctx, &models.Candidate{
		UserID: u1.ID, Slug: "go-gopher", Name: "Go Gopher", Zipcode: "12345",
		Email: "gopher@example.com", Skills: "Go, PostgreSQL, backend engineering",
		ResumeHTML: "<p>Go backend engineer</p>", ResumeText: "Go backend engineer",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := candidates.Create(ctx, &models.Candidate{
		UserID: u2.ID, Slug: "pat-painter", Name: "Pat Painter", Zipcode: "54321",
		Email: "painter@example.com", Skills: "Oil painting, watercolor",
		ResumeHTML: "<p>Fine artist</p>", ResumeText: "Fine artist",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	results, total, err := candidates.Search(ctx, "Go backend", 10, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 1 || len(results) != 1 {
		t.Fatalf("expected 1 result for 'Go backend', got total=%d len=%d", total, len(results))
	}
	if results[0].Candidate.Slug != "go-gopher" {
		t.Errorf("expected go-gopher to match, got %s", results[0].Candidate.Slug)
	}

	allResults, allTotal, err := candidates.Search(ctx, "", 10, 0)
	if err != nil {
		t.Fatalf("Search empty query: %v", err)
	}
	if allTotal != 2 || len(allResults) != 2 {
		t.Errorf("expected empty query to return all 2 candidates, got total=%d len=%d", allTotal, len(allResults))
	}
}
