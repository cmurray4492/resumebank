package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"resumebank/internal/app"
	"resumebank/internal/config"
	"resumebank/internal/models"
	"resumebank/internal/testutil"
	"resumebank/internal/web"
)

func newTestApp(t *testing.T) *app.App {
	t.Helper()
	pool := testutil.OpenTestDB(t)
	cfg := &config.Config{
		Env:            "test", // anything other than "development" so templates load from the embedded FS, not a cwd-relative disk path
		SessionSecret:  "test-secret",
		UploadDir:      t.TempDir(),
		MaxUploadBytes: 10 << 20,
	}
	a, err := app.New(cfg, pool)
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	return a
}

func TestOwnership_CandidateCannotEditAnotherCandidatesProfile(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	ownerUser, err := a.Users.Create(ctx, "owner@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating owner user: %v", err)
	}
	owner, err := a.Candidates.Create(ctx, &models.Candidate{
		UserID: ownerUser.ID, Slug: "owner-candidate", Name: "Owner Candidate", Zipcode: "12345",
		Email: "owner@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	})
	if err != nil {
		t.Fatalf("creating owner candidate: %v", err)
	}

	attackerUser, err := a.Users.Create(ctx, "attacker@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating attacker user: %v", err)
	}
	if _, err := a.Candidates.Create(ctx, &models.Candidate{
		UserID: attackerUser.ID, Slug: "attacker-candidate", Name: "Attacker Candidate", Zipcode: "54321",
		Email: "attacker@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	}); err != nil {
		t.Fatalf("creating attacker candidate: %v", err)
	}

	token, err := a.Sessions.Create(ctx, attackerUser.ID)
	if err != nil {
		t.Fatalf("creating attacker session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/candidates/"+owner.Slug+"/edit", nil)
	req.AddCookie(&http.Cookie{Name: "resumebank_session", Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden when editing another candidate's profile, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestOwnership_AnonymousEditRedirectsToLogin(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	ownerUser, err := a.Users.Create(ctx, "owner@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating owner user: %v", err)
	}
	owner, err := a.Candidates.Create(ctx, &models.Candidate{
		UserID: ownerUser.ID, Slug: "owner-candidate", Name: "Owner Candidate", Zipcode: "12345",
		Email: "owner@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	})
	if err != nil {
		t.Fatalf("creating owner candidate: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/candidates/"+owner.Slug+"/edit", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "/login") {
		t.Errorf("expected redirect to /login, got %d Location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestOwnership_CandidateOwnerCanLoadEditForm(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	ownerUser, err := a.Users.Create(ctx, "owner@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating owner user: %v", err)
	}
	owner, err := a.Candidates.Create(ctx, &models.Candidate{
		UserID: ownerUser.ID, Slug: "owner-candidate", Name: "Owner Candidate", Zipcode: "12345",
		Email: "owner@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	})
	if err != nil {
		t.Fatalf("creating owner candidate: %v", err)
	}

	token, err := a.Sessions.Create(ctx, ownerUser.ID)
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/candidates/"+owner.Slug+"/edit", nil)
	req.AddCookie(&http.Cookie{Name: "resumebank_session", Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200 OK for the owner loading their own edit form, got %d: %s", rec.Code, rec.Body.String())
	}
}
