package repo_test

import (
	"context"
	"testing"

	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/testutil"
)

func TestFileRepo_OneResumePDFPerCandidate(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	candidates := repo.NewCandidateRepo(pool)
	files := repo.NewFileRepo(pool)
	ctx := context.Background()

	u := newTestUser(t, users, "jane@example.com", models.RoleCandidate)
	c, err := candidates.Create(ctx, &models.Candidate{
		UserID: u.ID, Slug: "jane-doe", Name: "Jane Doe", Zipcode: "12345",
		Email: "jane@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	})
	if err != nil {
		t.Fatalf("Create candidate: %v", err)
	}

	first := &models.CandidateFile{
		CandidateID: c.ID, Kind: models.FileKindResumePDF,
		OriginalFilename: "resume.pdf", StoredPath: "candidates/1/a.pdf",
		ContentType: "application/pdf", SizeBytes: 100,
	}
	if _, err := files.Create(ctx, first); err != nil {
		t.Fatalf("expected first resume_pdf to succeed, got: %v", err)
	}

	second := &models.CandidateFile{
		CandidateID: c.ID, Kind: models.FileKindResumePDF,
		OriginalFilename: "resume2.pdf", StoredPath: "candidates/1/b.pdf",
		ContentType: "application/pdf", SizeBytes: 100,
	}
	if _, err := files.Create(ctx, second); err == nil {
		t.Error("expected second resume_pdf for the same candidate to be rejected")
	}
}

func TestFileRepo_CountByKind(t *testing.T) {
	pool := testutil.OpenTestDB(t)
	users := repo.NewUserRepo(pool)
	candidates := repo.NewCandidateRepo(pool)
	files := repo.NewFileRepo(pool)
	ctx := context.Background()

	u := newTestUser(t, users, "jane@example.com", models.RoleCandidate)
	c, err := candidates.Create(ctx, &models.Candidate{
		UserID: u.ID, Slug: "jane-doe", Name: "Jane Doe", Zipcode: "12345",
		Email: "jane@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	})
	if err != nil {
		t.Fatalf("Create candidate: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := files.Create(ctx, &models.CandidateFile{
			CandidateID: c.ID, Kind: models.FileKindAdditional,
			OriginalFilename: "file.txt", StoredPath: "candidates/1/file.txt",
			ContentType: "text/plain", SizeBytes: 10, Position: int16(i),
		})
		if err != nil {
			t.Fatalf("Create additional file %d: %v", i, err)
		}
	}

	count, err := files.CountByKind(ctx, c.ID, models.FileKindAdditional)
	if err != nil {
		t.Fatalf("CountByKind: %v", err)
	}
	if count != 3 {
		t.Errorf("got count %d, want 3", count)
	}
}
