package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"resumebank/internal/app"
	"resumebank/internal/auth"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/web"
)

func postWithCSRF(t *testing.T, router http.Handler, path string, form url.Values, sessionCookie *http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	token := ""
	if sessionCookie != nil && sessionCookie.Name == "resumebank_session" {
		token = sessionCookie.Value
	}
	csrf := fetchCSRFCookie(t, router, token)
	if form == nil {
		form = url.Values{}
	}
	form.Set("csrf_token", csrf.Value)
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrf)
	if sessionCookie != nil {
		req.AddCookie(sessionCookie)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func seedCandidate(t *testing.T, a *app.App, email, slug, password string) (*models.User, *models.Candidate) {
	t.Helper()
	ctx := context.Background()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}
	u, err := a.Users.Create(ctx, email, hash, models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}
	c, err := a.Candidates.Create(ctx, &models.Candidate{
		UserID: u.ID, Slug: slug, Name: slug, Zipcode: "12345",
		Email: email, ResumeHTML: "<p>R</p>", ResumeText: "R",
	})
	if err != nil {
		t.Fatalf("creating candidate: %v", err)
	}
	return u, c
}

func TestCandidateDeleteProfile_WrongPasswordKeepsAccount(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	u, c := seedCandidate(t, a, "del1@example.com", "del-one", "correct-password")
	token, _ := a.Sessions.Create(ctx, u.ID)

	rec := postWithCSRF(t, router, "/candidates/"+c.Slug+"/delete",
		url.Values{"password": {"wrong-password"}}, &http.Cookie{Name: "resumebank_session", Value: token})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for wrong password, got %d", rec.Code)
	}
	if _, err := a.Candidates.GetByID(ctx, c.ID); err != nil {
		t.Errorf("candidate should still exist after a wrong password, got: %v", err)
	}
}

func TestCandidateDeleteProfile_DeletesAccountAndFiles(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	u, c := seedCandidate(t, a, "del2@example.com", "del-two", "correct-password")
	token, _ := a.Sessions.Create(ctx, u.ID)

	key := "candidates/" + strconv.FormatInt(c.ID, 10) + "/resume.pdf"
	if err := a.Storage.Save(key, strings.NewReader("%PDF-1.4")); err != nil {
		t.Fatalf("saving file: %v", err)
	}
	if _, err := a.Files.Create(ctx, &models.CandidateFile{
		CandidateID: c.ID, Kind: models.FileKindResumePDF, OriginalFilename: "resume.pdf",
		StoredPath: key, ContentType: "application/pdf", SizeBytes: 8,
	}); err != nil {
		t.Fatalf("creating file record: %v", err)
	}

	rec := postWithCSRF(t, router, "/candidates/"+c.Slug+"/delete",
		url.Values{"password": {"correct-password"}}, &http.Cookie{Name: "resumebank_session", Value: token})
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect after deleting, got %d: %s", rec.Code, rec.Body.String())
	}

	if _, err := a.Candidates.GetByID(ctx, c.ID); err != repo.ErrNotFound {
		t.Errorf("expected candidate to be gone, got: %v", err)
	}
	if _, err := a.Users.GetUserByID(ctx, u.ID); err != repo.ErrNotFound {
		t.Errorf("expected user to be gone, got: %v", err)
	}
	if _, err := a.Sessions.Lookup(ctx, token); err == nil {
		t.Errorf("expected the session to be invalidated")
	}
	if rc, err := a.Storage.Open(key); err == nil {
		rc.Close()
		t.Errorf("expected the stored resume file to be removed")
	}
}

func TestCandidateDeleteProfile_CannotDeleteAnotherCandidate(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	_, victim := seedCandidate(t, a, "victim@example.com", "victim", "victim-password")
	attacker, _ := seedCandidate(t, a, "attacker2@example.com", "attacker2", "attacker-password")
	token, _ := a.Sessions.Create(ctx, attacker.ID)

	rec := postWithCSRF(t, router, "/candidates/"+victim.Slug+"/delete",
		url.Values{"password": {"attacker-password"}}, &http.Cookie{Name: "resumebank_session", Value: token})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if _, err := a.Candidates.GetByID(ctx, victim.ID); err != nil {
		t.Errorf("victim should be untouched, got: %v", err)
	}
}

func adminCookie(t *testing.T, a *app.App) *http.Cookie {
	t.Helper()
	admin := createAdminUser(t, context.Background(), a, "admin-del@example.com", "supersecret123")
	token, err := a.Sessions.Create(context.Background(), admin.ID)
	if err != nil {
		t.Fatalf("creating admin session: %v", err)
	}
	return &http.Cookie{Name: auth.AdminSessionCookieName, Value: token}
}

func TestAdminDelete_CandidateEmployerAndJob(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()
	admin := adminCookie(t, a)

	// Candidate
	cu, c := seedCandidate(t, a, "admdel@example.com", "adm-del", "pw-pw-pw-pw")
	rec := postWithCSRF(t, router, "/admin/candidates/"+strconv.FormatInt(c.ID, 10)+"/delete", nil, admin)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("candidate delete: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := a.Users.GetUserByID(ctx, cu.ID); err != repo.ErrNotFound {
		t.Errorf("expected candidate's user to be gone, got: %v", err)
	}

	// Employer (cascades to its jobs) and a separate job delete
	eu, err := a.Users.Create(ctx, "emp-del@example.com", "hash", models.RoleEmployer)
	if err != nil {
		t.Fatalf("creating employer user: %v", err)
	}
	e, err := a.Employers.Create(ctx, &models.Employer{UserID: eu.ID, Slug: "del-co", CompanyName: "Del Co", Zipcode: "12345"})
	if err != nil {
		t.Fatalf("creating employer: %v", err)
	}
	mkJob := func(slug string) *models.Job {
		j, err := a.Jobs.Create(ctx, &models.Job{
			EmployerID: e.ID, Slug: slug, Title: slug, DescriptionHTML: "<p>D</p>", DescriptionText: "D",
			ApplyMethod: "url", ApplyValue: "https://example.com/apply",
		})
		if err != nil {
			t.Fatalf("creating job: %v", err)
		}
		return j
	}
	solo, other := mkJob("del-job-solo"), mkJob("del-job-other")

	rec = postWithCSRF(t, router, "/admin/jobs/"+strconv.FormatInt(solo.ID, 10)+"/delete", nil, admin)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("job delete: expected 303, got %d", rec.Code)
	}
	if _, err := a.Jobs.GetByID(ctx, solo.ID); err != repo.ErrNotFound {
		t.Errorf("expected job to be gone, got: %v", err)
	}
	if _, err := a.Employers.GetByID(ctx, e.ID); err != nil {
		t.Errorf("deleting one job must not delete the employer, got: %v", err)
	}

	rec = postWithCSRF(t, router, "/admin/employers/"+strconv.FormatInt(e.ID, 10)+"/delete", nil, admin)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("employer delete: expected 303, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := a.Employers.GetByID(ctx, e.ID); err != repo.ErrNotFound {
		t.Errorf("expected employer to be gone, got: %v", err)
	}
	if _, err := a.Jobs.GetByID(ctx, other.ID); err != repo.ErrNotFound {
		t.Errorf("expected the employer's remaining job to be deleted with it, got: %v", err)
	}
}

func TestAdminDelete_RequiresAdmin(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	u, c := seedCandidate(t, a, "nonadmin@example.com", "non-admin", "pw-pw-pw-pw")
	token, _ := a.Sessions.Create(ctx, u.ID)

	rec := postWithCSRF(t, router, "/admin/candidates/"+strconv.FormatInt(c.ID, 10)+"/delete", nil,
		&http.Cookie{Name: "resumebank_session", Value: token})
	if rec.Code == http.StatusSeeOther && !strings.HasPrefix(rec.Header().Get("Location"), "/admin/login") {
		t.Fatalf("non-admin was not sent to the admin login, got redirect to %q", rec.Header().Get("Location"))
	}
	if _, err := a.Candidates.GetByID(ctx, c.ID); err != nil {
		t.Errorf("candidate must survive a non-admin delete attempt, got: %v", err)
	}
}

func TestUserRepoDelete_NeverDeletesAdmins(t *testing.T) {
	a := newTestApp(t)
	admin := createAdminUser(t, context.Background(), a, "keep-admin@example.com", "supersecret123")
	if err := a.Users.Delete(context.Background(), admin.ID); err != repo.ErrNotFound {
		t.Errorf("expected ErrNotFound when deleting an admin, got: %v", err)
	}
	if _, err := a.Users.GetUserByID(context.Background(), admin.ID); err != nil {
		t.Errorf("admin user must still exist, got: %v", err)
	}
}
