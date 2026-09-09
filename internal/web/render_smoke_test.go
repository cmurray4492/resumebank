package web_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}
			body := rec.Body.String()
			if strings.Contains(body, "render error") {
				t.Errorf("page rendered with an error instead of content: %s", body)
			}
			if len(body) < 500 {
				t.Errorf("page body suspiciously short (%d bytes), likely a broken render: %s", len(body), body)
			}
		})
	}
}
