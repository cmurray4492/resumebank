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

// identicalEmbeddingVector returns a vector literal usable directly in SQL,
// so two rows given the same one are equally similar to any query vector -
// isolating ranking-test assertions to the recency/vote tie-breaker alone.
func identicalEmbeddingVector() string {
	vector := make([]string, 768)
	for i := range vector {
		vector[i] = "0.001"
	}
	return "[" + strings.Join(vector, ",") + "]"
}

func TestJobRepo_RankingFavorsRecencyOnTiedSimilarity(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	employers := repo.NewEmployerRepo(pool)
	jobs := repo.NewJobRepo(pool)
	ctx := context.Background()

	employerUser := newTestUser(t, users, "acme@example.com", models.RoleEmployer)
	employer := newTestEmployer(t, employers, employerUser.ID, "acme", "Acme")

	older, err := jobs.Create(ctx, &models.Job{
		EmployerID: employer.ID, Slug: "older-job", Title: "Older Job",
		DescriptionHTML: "<p>D</p>", DescriptionText: "Go backend role",
		ApplyMethod: "url", ApplyValue: "https://acme.example.com/apply",
	})
	if err != nil {
		t.Fatalf("Create older job: %v", err)
	}
	newer, err := jobs.Create(ctx, &models.Job{
		EmployerID: employer.ID, Slug: "newer-job", Title: "Newer Job",
		DescriptionHTML: "<p>D</p>", DescriptionText: "Go backend role",
		ApplyMethod: "url", ApplyValue: "https://acme.example.com/apply",
	})
	if err != nil {
		t.Fatalf("Create newer job: %v", err)
	}

	vectorLiteral := identicalEmbeddingVector()
	if err := jobs.SetEmbedding(ctx, older.ID, vectorLiteral); err != nil {
		t.Fatalf("SetEmbedding older: %v", err)
	}
	if err := jobs.SetEmbedding(ctx, newer.ID, vectorLiteral); err != nil {
		t.Fatalf("SetEmbedding newer: %v", err)
	}

	// Backdate the older job well past the ~30-day recency decay window so
	// the newer job's recency component clearly outweighs it despite tied
	// similarity.
	if _, err := pool.Exec(ctx, `UPDATE jobs SET date_posted = now() - interval '90 days' WHERE id = $1`, older.ID); err != nil {
		t.Fatalf("backdating older job: %v", err)
	}

	results, err := jobs.MatchByEmbedding(ctx, vectorLiteral, 5, "", nil)
	if err != nil {
		t.Fatalf("MatchByEmbedding: %v", err)
	}
	if len(results) != 2 || results[0].Slug != "newer-job" {
		t.Fatalf("expected newer-job to rank first on recency with tied similarity, got %+v", results)
	}
	if results[0].Similarity < 0.99 || results[1].Similarity < 0.99 {
		t.Fatalf("expected both jobs to have near-1.0 similarity (tied), got %+v", results)
	}
}

func TestJobRepo_RankingFavorsVotesOnTiedSimilarityAndRecency(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	employers := repo.NewEmployerRepo(pool)
	jobs := repo.NewJobRepo(pool)
	candidates := repo.NewCandidateRepo(pool)
	votes := repo.NewVoteRepo(pool)
	ctx := context.Background()

	employerUser := newTestUser(t, users, "acme@example.com", models.RoleEmployer)
	employer := newTestEmployer(t, employers, employerUser.ID, "acme", "Acme")

	// Both jobs are created back-to-back, so their date_posted values are
	// (for ranking purposes) tied - isolating this test to the vote signal
	// alone, unlike the recency test above.
	upvoted, err := jobs.Create(ctx, &models.Job{
		EmployerID: employer.ID, Slug: "upvoted-job", Title: "Upvoted Job",
		DescriptionHTML: "<p>D</p>", DescriptionText: "Go backend role",
		ApplyMethod: "url", ApplyValue: "https://acme.example.com/apply",
	})
	if err != nil {
		t.Fatalf("Create upvoted job: %v", err)
	}
	unvoted, err := jobs.Create(ctx, &models.Job{
		EmployerID: employer.ID, Slug: "unvoted-job", Title: "Unvoted Job",
		DescriptionHTML: "<p>D</p>", DescriptionText: "Go backend role",
		ApplyMethod: "url", ApplyValue: "https://acme.example.com/apply",
	})
	if err != nil {
		t.Fatalf("Create unvoted job: %v", err)
	}

	vectorLiteral := identicalEmbeddingVector()
	if err := jobs.SetEmbedding(ctx, upvoted.ID, vectorLiteral); err != nil {
		t.Fatalf("SetEmbedding upvoted: %v", err)
	}
	if err := jobs.SetEmbedding(ctx, unvoted.ID, vectorLiteral); err != nil {
		t.Fatalf("SetEmbedding unvoted: %v", err)
	}

	voterUser := newTestUser(t, users, "voter@example.com", models.RoleCandidate)
	voter, err := candidates.Create(ctx, &models.Candidate{
		UserID: voterUser.ID, Slug: "voter", Name: "Voter", Zipcode: "12345",
		Email: "voter@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	})
	if err != nil {
		t.Fatalf("Create voter candidate: %v", err)
	}
	if err := votes.Set(ctx, voter.ID, upvoted.ID, 1); err != nil {
		t.Fatalf("Set vote: %v", err)
	}

	results, err := jobs.MatchByEmbedding(ctx, vectorLiteral, 5, "", nil)
	if err != nil {
		t.Fatalf("MatchByEmbedding: %v", err)
	}
	if len(results) != 2 || results[0].Slug != "upvoted-job" {
		t.Fatalf("expected upvoted-job to rank first with tied similarity and recency, got %+v", results)
	}
}

func TestCandidateRepo_RankingFavorsRecencyOnTiedSimilarity(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	candidates := repo.NewCandidateRepo(pool)
	ctx := context.Background()

	olderUser := newTestUser(t, users, "older@example.com", models.RoleCandidate)
	older, err := candidates.Create(ctx, &models.Candidate{
		UserID: olderUser.ID, Slug: "older-candidate", Name: "Older Candidate", Zipcode: "12345",
		Email: "older@example.com", ResumeHTML: "<p>R</p>", ResumeText: "Go backend engineer",
	})
	if err != nil {
		t.Fatalf("Create older candidate: %v", err)
	}
	newerUser := newTestUser(t, users, "newer@example.com", models.RoleCandidate)
	newer, err := candidates.Create(ctx, &models.Candidate{
		UserID: newerUser.ID, Slug: "newer-candidate", Name: "Newer Candidate", Zipcode: "12345",
		Email: "newer@example.com", ResumeHTML: "<p>R</p>", ResumeText: "Go backend engineer",
	})
	if err != nil {
		t.Fatalf("Create newer candidate: %v", err)
	}

	vectorLiteral := identicalEmbeddingVector()
	if err := candidates.SetEmbedding(ctx, older.ID, vectorLiteral); err != nil {
		t.Fatalf("SetEmbedding older: %v", err)
	}
	if err := candidates.SetEmbedding(ctx, newer.ID, vectorLiteral); err != nil {
		t.Fatalf("SetEmbedding newer: %v", err)
	}

	if _, err := pool.Exec(ctx, `UPDATE candidates SET updated_at = now() - interval '90 days' WHERE id = $1`, older.ID); err != nil {
		t.Fatalf("backdating older candidate: %v", err)
	}

	results, err := candidates.MatchByEmbedding(ctx, vectorLiteral, 5, "")
	if err != nil {
		t.Fatalf("MatchByEmbedding: %v", err)
	}
	if len(results) != 2 || results[0].Slug != "newer-candidate" {
		t.Fatalf("expected newer-candidate to rank first on recency with tied similarity, got %+v", results)
	}
}
