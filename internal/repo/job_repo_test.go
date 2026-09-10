package repo_test

import (
	"context"
	"testing"

	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/testutil"
)

func newTestEmployer(t *testing.T, employers *repo.EmployerRepo, userID int64, slug, name string) *models.Employer {
	t.Helper()
	e, err := employers.Create(context.Background(), &models.Employer{
		UserID: userID, Slug: slug, CompanyName: name, Zipcode: "12345",
	})
	if err != nil {
		t.Fatalf("creating test employer: %v", err)
	}
	return e
}

func TestJobRepo_CreateSearchDelete(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	employers := repo.NewEmployerRepo(pool)
	jobs := repo.NewJobRepo(pool)
	ctx := context.Background()

	u := newTestUser(t, users, "acme@example.com", models.RoleEmployer)
	e := newTestEmployer(t, employers, u.ID, "acme-inc", "Acme Inc")

	job, err := jobs.Create(ctx, &models.Job{
		EmployerID: e.ID, Slug: "acme-inc-backend-engineer", Title: "Backend Engineer",
		DescriptionHTML: "<p>Build Go services</p>", DescriptionText: "Build Go services",
		ApplyMethod: "url", ApplyValue: "https://acme.example.com/apply",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	results, total, err := jobs.Search(ctx, "Go services", 10, 0)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 1 || len(results) != 1 || results[0].CompanyName != "Acme Inc" {
		t.Fatalf("expected 1 result with company Acme Inc, got total=%d results=%+v", total, results)
	}

	listed, err := jobs.ListByEmployer(ctx, e.ID)
	if err != nil {
		t.Fatalf("ListByEmployer: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 job for employer, got %d", len(listed))
	}

	if err := jobs.Delete(ctx, job.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := jobs.GetBySlug(ctx, job.Slug); err != repo.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}
