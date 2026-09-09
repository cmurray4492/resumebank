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
	"resumebank/internal/web"
)

func createAdminUser(t *testing.T, ctx context.Context, a *app.App, email, password string) *models.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}
	u, err := a.Users.Create(ctx, email, hash, models.RoleAdmin)
	if err != nil {
		t.Fatalf("creating admin user: %v", err)
	}
	return u
}

func TestAdminAuth_LoginRejectsNonAdminEvenWithCorrectPassword(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	hash, _ := auth.HashPassword("supersecret123")
	if _, err := a.Users.Create(ctx, "candidate@example.com", hash, models.RoleCandidate); err != nil {
		t.Fatalf("creating candidate user: %v", err)
	}

	csrf := fetchCSRFCookie(t, router, "")
	form := url.Values{"csrf_token": {csrf.Value}, "email": {"candidate@example.com"}, "password": {"supersecret123"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/login", nil)
	req.PostForm = form
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrf)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 (generic invalid credentials) for a non-admin account, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAdminAuth_LoginSucceedsForAdmin(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()
	createAdminUser(t, ctx, a, "admin@example.com", "supersecret123")

	csrf := fetchCSRFCookie(t, router, "")
	form := url.Values{"csrf_token": {csrf.Value}, "email": {"admin@example.com"}, "password": {"supersecret123"}}
	req := httptest.NewRequest(http.MethodPost, "/admin/login", nil)
	req.PostForm = form
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrf)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 redirect on successful admin login, got %d: %s", rec.Code, rec.Body.String())
	}
	var adminCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.AdminSessionCookieName {
			adminCookie = c
		}
	}
	if adminCookie == nil {
		t.Fatal("expected an admin session cookie to be set")
	}

	dashReq := httptest.NewRequest(http.MethodGet, "/admin", nil)
	dashReq.AddCookie(adminCookie)
	dashRec := httptest.NewRecorder()
	router.ServeHTTP(dashRec, dashReq)
	if dashRec.Code != http.StatusOK {
		t.Errorf("expected 200 loading the admin dashboard, got %d: %s", dashRec.Code, dashRec.Body.String())
	}
}

func TestAdminAuth_UnauthenticatedRedirectsToAdminLogin(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "/admin/login") {
		t.Errorf("expected redirect to /admin/login, got %d Location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestAdminAuth_RegularUserSessionDoesNotGrantAdminAccess(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	candidateUser, err := a.Users.Create(ctx, "candidate@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating candidate: %v", err)
	}
	token, err := a.Sessions.Create(ctx, candidateUser.ID)
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "resumebank_session", Value: token})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther || !strings.Contains(rec.Header().Get("Location"), "/admin/login") {
		t.Errorf("expected a regular user's public session to be rejected for /admin, got %d Location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

// adminSession logs in as an admin and returns the admin session + CSRF cookies.
func adminSession(t *testing.T, router http.Handler, userID int64, a *app.App) (*http.Cookie, *http.Cookie) {
	t.Helper()
	token, err := a.Sessions.Create(context.Background(), userID)
	if err != nil {
		t.Fatalf("creating admin session: %v", err)
	}
	adminCookie := &http.Cookie{Name: auth.AdminSessionCookieName, Value: token}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(adminCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	var csrfCookie *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == "resumebank_csrf" {
			csrfCookie = c
		}
	}
	if csrfCookie == nil {
		t.Fatal("expected a CSRF cookie from loading the admin dashboard")
	}
	return adminCookie, csrfCookie
}

func TestAdminBlog_FullLifecycle(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()
	admin := createAdminUser(t, ctx, a, "admin@example.com", "supersecret123")
	adminCookie, csrfCookie := adminSession(t, router, admin.ID, a)

	// Create a published post.
	createForm := url.Values{
		"csrf_token": {csrfCookie.Value}, "title": {"My First Post"},
		"body_html": {"<p>Hello readers</p>"}, "author_name": {"Admin"}, "published": {"on"},
	}
	createReq := httptest.NewRequest(http.MethodPost, "/admin/blog", nil)
	createReq.PostForm = createForm
	createReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createReq.AddCookie(adminCookie)
	createReq.AddCookie(csrfCookie)
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 after creating post, got %d: %s", createRec.Code, createRec.Body.String())
	}

	post, err := a.Blog.GetBySlug(ctx, "my-first-post")
	if err != nil {
		t.Fatalf("expected post to exist with slug my-first-post: %v", err)
	}

	// Public blog shows it.
	publicReq := httptest.NewRequest(http.MethodGet, "/blog/my-first-post", nil)
	publicRec := httptest.NewRecorder()
	router.ServeHTTP(publicRec, publicReq)
	if publicRec.Code != http.StatusOK || !strings.Contains(publicRec.Body.String(), "Hello readers") {
		t.Fatalf("expected the published post to be publicly visible, got %d", publicRec.Code)
	}

	// Non-admin can't reach the admin edit route.
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/admin/blog/"+strconv.FormatInt(post.ID, 10)+"/edit", nil)
	forbiddenRec := httptest.NewRecorder()
	router.ServeHTTP(forbiddenRec, forbiddenReq)
	if forbiddenRec.Code != http.StatusSeeOther {
		t.Errorf("expected unauthenticated edit access to redirect to login, got %d", forbiddenRec.Code)
	}

	// Admin unpublishes it.
	updateForm := url.Values{
		"csrf_token": {csrfCookie.Value}, "title": {"My First Post"},
		"body_html": {"<p>Hello readers</p>"}, "author_name": {"Admin"}, // no "published" field = unchecked
	}
	updateReq := httptest.NewRequest(http.MethodPost, "/admin/blog/"+strconv.FormatInt(post.ID, 10)+"/edit", nil)
	updateReq.PostForm = updateForm
	updateReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	updateReq.AddCookie(adminCookie)
	updateReq.AddCookie(csrfCookie)
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 after updating post, got %d: %s", updateRec.Code, updateRec.Body.String())
	}

	afterUnpublish := httptest.NewRequest(http.MethodGet, "/blog/my-first-post", nil)
	afterUnpublishRec := httptest.NewRecorder()
	router.ServeHTTP(afterUnpublishRec, afterUnpublish)
	if afterUnpublishRec.Code != http.StatusNotFound {
		t.Errorf("expected unpublished post to 404 publicly, got %d", afterUnpublishRec.Code)
	}

	// Admin deletes it.
	deleteReq := httptest.NewRequest(http.MethodPost, "/admin/blog/"+strconv.FormatInt(post.ID, 10)+"/delete", nil)
	deleteReq.PostForm = url.Values{"csrf_token": {csrfCookie.Value}}
	deleteReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	deleteReq.AddCookie(adminCookie)
	deleteReq.AddCookie(csrfCookie)
	deleteRec := httptest.NewRecorder()
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 after deleting post, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
	if _, err := a.Blog.GetByID(ctx, post.ID); err == nil {
		t.Error("expected post to be gone after admin delete")
	}
}

func TestAdminCandidate_EditBypassesOwnership(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()
	admin := createAdminUser(t, ctx, a, "admin@example.com", "supersecret123")
	adminCookie, csrfCookie := adminSession(t, router, admin.ID, a)

	candidateUser, err := a.Users.Create(ctx, "jane@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating candidate user: %v", err)
	}
	candidate, err := a.Candidates.Create(ctx, &models.Candidate{
		UserID: candidateUser.ID, Slug: "jane-doe", Name: "Jane Doe", Zipcode: "12345",
		Email: "jane@example.com", ResumeHTML: "<p>Old resume</p>", ResumeText: "Old resume",
	})
	if err != nil {
		t.Fatalf("creating candidate: %v", err)
	}

	form := url.Values{
		"csrf_token": {csrfCookie.Value}, "name": {"Jane A. Doe"}, "zipcode": {"54321"},
		"email": {"jane@example.com"}, "resume_html": {"<p>New resume text</p>"},
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/candidates/"+strconv.FormatInt(candidate.ID, 10)+"/edit", nil)
	req.PostForm = form
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(adminCookie)
	req.AddCookie(csrfCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 after admin edits candidate, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, err := a.Candidates.GetByID(ctx, candidate.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if updated.Name != "Jane A. Doe" || updated.Zipcode != "54321" {
		t.Errorf("expected admin edit to persist, got name=%q zipcode=%q", updated.Name, updated.Zipcode)
	}
}
