package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"resumebank/internal/models"
	"resumebank/internal/web"
)

// TestRenderSmoke_PublicPagesRenderWithoutError guards against a template
// silently failing to render: html/template's escaping pass fails a page's
// *entire* render (writing nothing but an HTML comment, from
// render.Renderer.Render's fallback) if any {{template "x" .}} it
// references - most easily, "scripts", which every page must define, even
// as {{define "scripts"}}{{end}} - isn't defined, and that failure doesn't
// change the HTTP status code, so a bare status check can't catch it.
func TestRenderSmoke_PublicPagesRenderWithoutError(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)

	paths := []string{
		"/", "/about", "/terms", "/search", "/login",
		"/signup/candidate", "/signup/employer",
		"/blog", "/admin/login",
		"/forgot-password", "/reset-password/some-token",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			assertRendersCleanly(t, router, path, 500)
		})
	}
}

// TestRenderSmoke_AdminPagesRenderWithoutError is
// TestRenderSmoke_PublicPagesRenderWithoutError's counterpart for pages
// behind RequireAdmin, using admin_base.html.tmpl instead of base.html.tmpl.
func TestRenderSmoke_AdminPagesRenderWithoutError(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	admin := createAdminUser(t, ctx, a, "admin@example.com", "supersecret123")
	adminCookie, _ := adminSession(t, router, admin.ID, a)

	candidateUser, err := a.Users.Create(ctx, "jane@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating candidate user: %v", err)
	}
	candidate, err := a.Candidates.Create(ctx, &models.Candidate{
		UserID: candidateUser.ID, Slug: "jane-doe", Name: "Jane Doe", Zipcode: "12345",
		Email: "jane@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	})
	if err != nil {
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
	job, err := a.Jobs.Create(ctx, &models.Job{
		EmployerID: employer.ID, Slug: "acme-engineer", Title: "Engineer",
		DescriptionHTML: "<p>D</p>", DescriptionText: "D",
	})
	if err != nil {
		t.Fatalf("creating job: %v", err)
	}
	post, err := a.Blog.Create(ctx, &models.BlogPost{
		Slug: "hello", Title: "Hello", BodyHTML: "<p>Hi</p>", BodyText: "Hi",
	})
	if err != nil {
		t.Fatalf("creating blog post: %v", err)
	}

	paths := []string{
		"/admin",
		"/admin/blog", "/admin/blog/new", "/admin/blog/" + strconv.FormatInt(post.ID, 10) + "/edit",
		"/admin/candidates", "/admin/candidates/" + strconv.FormatInt(candidate.ID, 10) + "/edit",
		"/admin/employers", "/admin/employers/" + strconv.FormatInt(employer.ID, 10) + "/edit",
		"/admin/jobs", "/admin/jobs/" + strconv.FormatInt(job.ID, 10) + "/edit",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.AddCookie(adminCookie)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			body := rec.Body.String()
			if strings.Contains(body, "render error") {
				t.Errorf("page rendered with an error instead of content: %s", body)
			}
			if len(body) < 300 {
				t.Errorf("page body suspiciously short (%d bytes), likely a broken render: %s", len(body), body)
			}
		})
	}
}

func assertRendersCleanly(t *testing.T, router http.Handler, path string, minBytes int) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	respBody := rec.Body.String()
	if strings.Contains(respBody, "render error") {
		t.Errorf("page rendered with an error instead of content: %s", respBody)
	}
	if len(respBody) < minBytes {
		t.Errorf("page body suspiciously short (%d bytes), likely a broken render: %s", len(respBody), respBody)
	}
}
