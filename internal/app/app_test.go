package app_test

import (
	"context"
	"strings"
	"testing"

	"resumebank/internal/app"
	"resumebank/internal/config"
	"resumebank/internal/models"
	"resumebank/internal/testutil"
)

func newTestApp(t *testing.T) *app.App {
	t.Helper()
	pool := testutil.OpenTestDB(t)
	cfg := &config.Config{
		Env:            "test",
		SessionSecret:  "test-secret",
		UploadDir:      t.TempDir(),
		MaxUploadBytes: 10 << 20,
		BaseURL:        "https://resumebank.example",
	}
	a, err := app.New(cfg, pool)
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	return a
}

func TestRefreshSitemap_IncludesAllEntities(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()

	candidateUser, err := a.Users.Create(ctx, "candidate@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating candidate user: %v", err)
	}
	if _, err := a.Candidates.Create(ctx, &models.Candidate{
		UserID: candidateUser.ID, Slug: "jane-doe", Name: "Jane Doe", Zipcode: "12345",
		Email: "candidate@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	}); err != nil {
		t.Fatalf("creating candidate: %v", err)
	}

	employerUser, err := a.Users.Create(ctx, "acme@example.com", "hash", models.RoleEmployer)
	if err != nil {
		t.Fatalf("creating employer user: %v", err)
	}
	employer, err := a.Employers.Create(ctx, &models.Employer{
		UserID: employerUser.ID, Slug: "acme", CompanyName: "Acme", Zipcode: "12345",
	})
	if err != nil {
		t.Fatalf("creating employer: %v", err)
	}
	if _, err := a.Jobs.Create(ctx, &models.Job{
		EmployerID: employer.ID, Slug: "acme-engineer", Title: "Engineer",
		DescriptionHTML: "<p>D</p>", DescriptionText: "D",
	}); err != nil {
		t.Fatalf("creating job: %v", err)
	}

	if err := a.RefreshSitemap(ctx); err != nil {
		t.Fatalf("RefreshSitemap: %v", err)
	}

	data, generatedAt := a.Sitemap.Get()
	if generatedAt.IsZero() {
		t.Error("expected generatedAt to be set after RefreshSitemap")
	}
	xml := string(data)
	for _, want := range []string{
		"https://resumebank.example/",
		"https://resumebank.example/search",
		"https://resumebank.example/candidates/jane-doe",
		"https://resumebank.example/employers/acme",
		"https://resumebank.example/jobs/acme-engineer",
	} {
		if !strings.Contains(xml, "<loc>"+want+"</loc>") {
			t.Errorf("expected sitemap to contain %q, got:\n%s", want, xml)
		}
	}
}

func TestRefreshSearchIndex_Succeeds(t *testing.T) {
	a := newTestApp(t)
	ctx := context.Background()

	if err := a.RefreshSearchIndex(ctx); err != nil {
		t.Fatalf("RefreshSearchIndex: %v", err)
	}

	var count int
	if err := a.Pool.QueryRow(ctx, `SELECT count(*) FROM search_index`).Scan(&count); err != nil {
		t.Fatalf("querying search_index: %v", err)
	}
	if count != 0 {
		t.Errorf("expected an empty search_index on a fresh test database, got %d rows", count)
	}
}
