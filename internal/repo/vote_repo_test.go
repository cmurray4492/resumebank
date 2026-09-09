package repo_test

import (
	"context"
	"testing"

	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/testutil"
)

func TestVoteRepo_SetGetClearToggle(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	candidates := repo.NewCandidateRepo(pool)
	employers := repo.NewEmployerRepo(pool)
	jobs := repo.NewJobRepo(pool)
	votes := repo.NewVoteRepo(pool)
	ctx := context.Background()

	candidateUser := newTestUser(t, users, "voter@example.com", models.RoleCandidate)
	candidate, err := candidates.Create(ctx, &models.Candidate{
		UserID: candidateUser.ID, Slug: "voter", Name: "Voter", Zipcode: "12345",
		Email: "voter@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	})
	if err != nil {
		t.Fatalf("creating candidate: %v", err)
	}

	employerUser := newTestUser(t, users, "acme@example.com", models.RoleEmployer)
	employer := newTestEmployer(t, employers, employerUser.ID, "acme", "Acme")
	job, err := jobs.Create(ctx, &models.Job{
		EmployerID: employer.ID, Slug: "acme-role", Title: "Role",
		DescriptionHTML: "<p>D</p>", DescriptionText: "D",
	})
	if err != nil {
		t.Fatalf("creating job: %v", err)
	}

	if v, err := votes.GetVote(ctx, candidate.ID, job.ID); err != nil || v != 0 {
		t.Fatalf("expected no vote initially, got %d err=%v", v, err)
	}

	if err := votes.Set(ctx, candidate.ID, job.ID, 1); err != nil {
		t.Fatalf("Set up: %v", err)
	}
	if v, err := votes.GetVote(ctx, candidate.ID, job.ID); err != nil || v != 1 {
		t.Fatalf("expected vote 1, got %d err=%v", v, err)
	}
	up, down, err := votes.Counts(ctx, job.ID)
	if err != nil || up != 1 || down != 0 {
		t.Fatalf("expected up=1 down=0, got up=%d down=%d err=%v", up, down, err)
	}

	// Switching to down should overwrite, not add a second row.
	if err := votes.Set(ctx, candidate.ID, job.ID, -1); err != nil {
		t.Fatalf("Set down: %v", err)
	}
	up, down, err = votes.Counts(ctx, job.ID)
	if err != nil || up != 0 || down != 1 {
		t.Fatalf("expected up=0 down=1 after switching, got up=%d down=%d err=%v", up, down, err)
	}

	if err := votes.Clear(ctx, candidate.ID, job.ID); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if v, err := votes.GetVote(ctx, candidate.ID, job.ID); err != nil || v != 0 {
		t.Fatalf("expected no vote after clear, got %d err=%v", v, err)
	}
}
