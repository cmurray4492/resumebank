package repo_test

import (
	"context"
	"testing"

	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/testutil"
)

func TestListSlugs_CandidatesEmployersJobs(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	candidates := repo.NewCandidateRepo(pool)
	employers := repo.NewEmployerRepo(pool)
	jobs := repo.NewJobRepo(pool)
	ctx := context.Background()

	candidateUser := newTestUser(t, users, "candidate@example.com", models.RoleCandidate)
	if _, err := candidates.Create(ctx, &models.Candidate{
		UserID: candidateUser.ID, Slug: "jane-doe", Name: "Jane Doe", Zipcode: "12345",
		Email: "candidate@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	}); err != nil {
		t.Fatalf("creating candidate: %v", err)
	}

	employerUser := newTestUser(t, users, "acme@example.com", models.RoleEmployer)
	employer := newTestEmployer(t, employers, employerUser.ID, "acme", "Acme")
	if _, err := jobs.Create(ctx, &models.Job{
		EmployerID: employer.ID, Slug: "acme-engineer", Title: "Engineer",
		DescriptionHTML: "<p>D</p>", DescriptionText: "D",
	}); err != nil {
		t.Fatalf("creating job: %v", err)
	}

	candidateSlugs, err := candidates.ListSlugs(ctx)
	if err != nil {
		t.Fatalf("CandidateRepo.ListSlugs: %v", err)
	}
	if len(candidateSlugs) != 1 || candidateSlugs[0].Slug != "jane-doe" || candidateSlugs[0].UpdatedAt.IsZero() {
		t.Errorf("unexpected candidate slugs: %+v", candidateSlugs)
	}

	employerSlugs, err := employers.ListSlugs(ctx)
	if err != nil {
		t.Fatalf("EmployerRepo.ListSlugs: %v", err)
	}
	if len(employerSlugs) != 1 || employerSlugs[0].Slug != "acme" || employerSlugs[0].UpdatedAt.IsZero() {
		t.Errorf("unexpected employer slugs: %+v", employerSlugs)
	}

	jobSlugs, err := jobs.ListSlugs(ctx)
	if err != nil {
		t.Fatalf("JobRepo.ListSlugs: %v", err)
	}
	if len(jobSlugs) != 1 || jobSlugs[0].Slug != "acme-engineer" || jobSlugs[0].UpdatedAt.IsZero() {
		t.Errorf("unexpected job slugs: %+v", jobSlugs)
	}
}
