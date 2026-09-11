package repo_test

import (
	"context"
	"strings"
	"testing"

	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/testutil"
)

func TestCandidateRepo_EmbeddingLifecycle(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	candidates := repo.NewCandidateRepo(pool)
	ctx := context.Background()

	u := newTestUser(t, users, "jane@example.com", models.RoleCandidate)
	c, err := candidates.Create(ctx, &models.Candidate{
		UserID: u.ID, Slug: "jane-doe", Name: "Jane Doe", Zipcode: "12345",
		Email: "jane@example.com", ResumeHTML: "<p>R</p>", ResumeText: "Go backend engineer",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	missing, err := candidates.MissingEmbeddings(ctx, 10)
	if err != nil {
		t.Fatalf("MissingEmbeddings: %v", err)
	}
	if len(missing) != 1 || missing[0].ID != c.ID || missing[0].Text != "Go backend engineer" {
		t.Fatalf("expected the new candidate to be missing an embedding, got %+v", missing)
	}

	vector := make([]string, 768)
	for i := range vector {
		vector[i] = "0.001"
	}
	vectorLiteral := "[" + strings.Join(vector, ",") + "]"
	if err := candidates.SetEmbedding(ctx, c.ID, vectorLiteral); err != nil {
		t.Fatalf("SetEmbedding: %v", err)
	}

	missing, err = candidates.MissingEmbeddings(ctx, 10)
	if err != nil {
		t.Fatalf("MissingEmbeddings after set: %v", err)
	}
	if len(missing) != 0 {
		t.Fatalf("expected no candidates missing an embedding after SetEmbedding, got %+v", missing)
	}

	matches, err := candidates.MatchByEmbedding(ctx, vectorLiteral, 5, "")
	if err != nil {
		t.Fatalf("MatchByEmbedding: %v", err)
	}
	if len(matches) != 1 || matches[0].Slug != "jane-doe" {
		t.Fatalf("expected jane-doe to match its own embedding exactly, got %+v", matches)
	}
	if matches[0].Similarity < 0.99 {
		t.Errorf("expected near-1.0 similarity for an identical vector, got %f", matches[0].Similarity)
	}
}

func TestJobRepo_EmbeddingLifecycle(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	employers := repo.NewEmployerRepo(pool)
	jobs := repo.NewJobRepo(pool)
	ctx := context.Background()

	u := newTestUser(t, users, "acme@example.com", models.RoleEmployer)
	employer := newTestEmployer(t, employers, u.ID, "acme", "Acme")
	job, err := jobs.Create(ctx, &models.Job{
		EmployerID: employer.ID, Slug: "acme-engineer", Title: "Engineer",
		DescriptionHTML: "<p>D</p>", DescriptionText: "Go backend role",
		ApplyMethod: "url", ApplyValue: "https://acme.example.com/apply",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	missing, err := jobs.MissingEmbeddings(ctx, 10)
	if err != nil {
		t.Fatalf("MissingEmbeddings: %v", err)
	}
	if len(missing) != 1 || missing[0].ID != job.ID {
		t.Fatalf("expected the new job to be missing an embedding, got %+v", missing)
	}

	vector := make([]string, 768)
	for i := range vector {
		vector[i] = "0.001"
	}
	vectorLiteral := "[" + strings.Join(vector, ",") + "]"
	if err := jobs.SetEmbedding(ctx, job.ID, vectorLiteral); err != nil {
		t.Fatalf("SetEmbedding: %v", err)
	}

	matches, err := jobs.MatchByEmbedding(ctx, vectorLiteral, 5, "", nil)
	if err != nil {
		t.Fatalf("MatchByEmbedding: %v", err)
	}
	if len(matches) != 1 || matches[0].Slug != "acme-engineer" || matches[0].CompanyName != "Acme" {
		t.Fatalf("expected acme-engineer to match, got %+v", matches)
	}
}

func TestRecommendations_MatchAcrossCandidateAndJobEmbeddings(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	candidates := repo.NewCandidateRepo(pool)
	employers := repo.NewEmployerRepo(pool)
	jobs := repo.NewJobRepo(pool)
	ctx := context.Background()

	candidateUser := newTestUser(t, users, "jane@example.com", models.RoleCandidate)
	c, err := candidates.Create(ctx, &models.Candidate{
		UserID: candidateUser.ID, Slug: "jane-doe", Name: "Jane Doe", Zipcode: "12345",
		Email: "jane@example.com", ResumeHTML: "<p>R</p>", ResumeText: "Go backend engineer",
	})
	if err != nil {
		t.Fatalf("Create candidate: %v", err)
	}

	employerUser := newTestUser(t, users, "acme@example.com", models.RoleEmployer)
	employer := newTestEmployer(t, employers, employerUser.ID, "acme", "Acme")
	job, err := jobs.Create(ctx, &models.Job{
		EmployerID: employer.ID, Slug: "acme-engineer", Title: "Engineer",
		DescriptionHTML: "<p>D</p>", DescriptionText: "Go backend role",
		ApplyMethod: "url", ApplyValue: "https://acme.example.com/apply",
	})
	if err != nil {
		t.Fatalf("Create job: %v", err)
	}

	vector := make([]string, 768)
	for i := range vector {
		vector[i] = "0.001"
	}
	vectorLiteral := "[" + strings.Join(vector, ",") + "]"
	if err := candidates.SetEmbedding(ctx, c.ID, vectorLiteral); err != nil {
		t.Fatalf("SetEmbedding candidate: %v", err)
	}
	if err := jobs.SetEmbedding(ctx, job.ID, vectorLiteral); err != nil {
		t.Fatalf("SetEmbedding job: %v", err)
	}

	candidateResults, err := candidates.RecommendedForJob(ctx, job.ID, 5)
	if err != nil {
		t.Fatalf("RecommendedForJob: %v", err)
	}
	if len(candidateResults) != 1 || candidateResults[0].Slug != "jane-doe" {
		t.Fatalf("expected jane-doe recommended for the job, got %+v", candidateResults)
	}
	if candidateResults[0].Similarity < 0.99 {
		t.Errorf("expected near-1.0 similarity for identical embeddings, got %f", candidateResults[0].Similarity)
	}

	jobResults, err := jobs.RecommendedForCandidate(ctx, c.ID, 5)
	if err != nil {
		t.Fatalf("RecommendedForCandidate: %v", err)
	}
	if len(jobResults) != 1 || jobResults[0].Slug != "acme-engineer" {
		t.Fatalf("expected acme-engineer recommended for the candidate, got %+v", jobResults)
	}
	if jobResults[0].Similarity < 0.99 {
		t.Errorf("expected near-1.0 similarity for identical embeddings, got %f", jobResults[0].Similarity)
	}
}
