package app_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"resumebank/internal/app"
	"resumebank/internal/config"
	"resumebank/internal/embeddings"
	"resumebank/internal/models"
	"resumebank/internal/testutil"
)

const testOllamaURL = "http://localhost:11434"

func newTestApp(t *testing.T) *app.App {
	t.Helper()
	pool := testutil.OpenTestDB(t)
	cfg := &config.Config{
		Env:            "test",
		SessionSecret:  "test-secret",
		UploadDir:      t.TempDir(),
		MaxUploadBytes: 10 << 20,
		BaseURL:        "https://resumebank.example",
		OllamaURL:      testOllamaURL,
		EmbedModel:     "nomic-embed-text",
	}
	a, err := app.New(cfg, pool)
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	return a
}

// requireOllama skips the test if no Ollama server is reachable at
// testOllamaURL, so the embeddings tests stay portable to environments
// (like CI) that don't have one running.
func requireOllama(t *testing.T) {
	t.Helper()
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(testOllamaURL + "/api/version")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Skipf("Ollama not reachable at %s; skipping embeddings test", testOllamaURL)
	}
	resp.Body.Close()
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

func TestEmbeddings_EndToEndMatchQuality(t *testing.T) {
	requireOllama(t)
	a := newTestApp(t)
	ctx := context.Background()

	backendUser, err := a.Users.Create(ctx, "backend@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating backend candidate user: %v", err)
	}
	backendCandidate, err := a.Candidates.Create(ctx, &models.Candidate{
		UserID: backendUser.ID, Slug: "backend-candidate", Name: "Backend Candidate", Zipcode: "12345",
		Email: "backend@example.com", ResumeHTML: "<p>R</p>",
		ResumeText: "Senior Go backend engineer with 8 years building distributed systems, PostgreSQL, and Kubernetes.",
	})
	if err != nil {
		t.Fatalf("creating backend candidate: %v", err)
	}

	designerUser, err := a.Users.Create(ctx, "designer@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating designer candidate user: %v", err)
	}
	designerCandidate, err := a.Candidates.Create(ctx, &models.Candidate{
		UserID: designerUser.ID, Slug: "designer-candidate", Name: "Designer Candidate", Zipcode: "12345",
		Email: "designer@example.com", ResumeHTML: "<p>R</p>",
		ResumeText: "Creative graphic designer specializing in brand identity, typography, and Adobe Illustrator.",
	})
	if err != nil {
		t.Fatalf("creating designer candidate: %v", err)
	}

	// Compute embeddings synchronously (not via the fire-and-forget
	// Trigger* methods) so the test can assert on the result deterministically.
	a.UpdateCandidateEmbedding(ctx, backendCandidate.ID, backendCandidate.ResumeText)
	a.UpdateCandidateEmbedding(ctx, designerCandidate.ID, designerCandidate.ResumeText)

	missing, err := a.Candidates.MissingEmbeddings(ctx, 10)
	if err != nil {
		t.Fatalf("MissingEmbeddings: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected both candidates to have embeddings computed, still missing: %+v", missing)
	}

	jobDescription := "Looking for a backend engineer experienced in Go, PostgreSQL, and building scalable distributed systems."
	queryVector, err := a.Embeddings.EmbedQuery(ctx, jobDescription)
	if err != nil {
		t.Fatalf("EmbedQuery: %v", err)
	}

	matches, err := a.Candidates.MatchByEmbedding(ctx, embeddings.FormatVector(queryVector), 5)
	if err != nil {
		t.Fatalf("MatchByEmbedding: %v", err)
	}
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d: %+v", len(matches), matches)
	}
	if matches[0].Slug != "backend-candidate" {
		t.Errorf("expected the backend candidate to rank first for a backend job description, got top match %q (results: %+v)",
			matches[0].Slug, matches)
	}
}
