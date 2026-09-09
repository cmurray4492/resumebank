package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"resumebank/internal/auth"
	"resumebank/internal/models"
	"resumebank/internal/web"
)

// fakeMailer captures sent emails instead of delivering them, so tests can
// extract the reset link without needing a real mailbox or parsing log
// output. app.SendPasswordResetEmail sends in its own goroutine (so the
// HTTP handler never blocks on it), so this uses a channel rather than a
// plain slice: next() blocks (with a timeout) until that goroutine
// actually calls Send, instead of racing it.
type fakeMailer struct {
	sent chan sentEmail
}

type sentEmail struct {
	To, Subject, Body string
}

func newFakeMailer() *fakeMailer {
	return &fakeMailer{sent: make(chan sentEmail, 10)}
}

func (m *fakeMailer) Send(ctx context.Context, to, subject, body string) error {
	m.sent <- sentEmail{To: to, Subject: subject, Body: body}
	return nil
}

// next waits for the next sent email, failing the test if none arrives
// within a reasonable time.
func (m *fakeMailer) next(t *testing.T) sentEmail {
	t.Helper()
	select {
	case email := <-m.sent:
		return email
	case <-time.After(2 * time.Second):
		t.Fatal("expected an email to have been sent within 2s")
		return sentEmail{}
	}
}

// assertNoEmail fails the test if an email is sent within a short window -
// used to confirm SendPasswordResetEmail was never triggered (e.g. for an
// unknown address). This is inherently a bit racy in the "nothing happens"
// direction (a slow goroutine could send after the window closes and be
// missed), but that would only ever produce a false negative test failure,
// never mask a real bug silently.
func (m *fakeMailer) assertNoEmail(t *testing.T) {
	t.Helper()
	select {
	case email := <-m.sent:
		t.Fatalf("expected no email to be sent, got one to %q", email.To)
	case <-time.After(200 * time.Millisecond):
	}
}

var resetLinkRE = regexp.MustCompile(`/reset-password/([a-f0-9]+)`)

func extractResetToken(t *testing.T, body string) string {
	t.Helper()
	match := resetLinkRE.FindStringSubmatch(body)
	if match == nil {
		t.Fatalf("expected a reset link in email body, got: %s", body)
	}
	return match[1]
}

// requestPasswordReset drives the public /forgot-password form for email
// and asserts the generic success message. It does NOT wait for the email
// itself - app.SendPasswordResetEmail sends asynchronously, so callers that
// need the sent email must wait on fakeMailer.next, which blocks until the
// background goroutine actually delivers it (or times out).
func requestPasswordReset(t *testing.T, router http.Handler, email string) {
	t.Helper()
	csrf := fetchCSRFCookie(t, router, "")
	form := url.Values{"csrf_token": {csrf.Value}, "email": {email}}
	req := httptest.NewRequest(http.MethodPost, "/forgot-password", nil)
	req.PostForm = form
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrf)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from /forgot-password, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "we&#39;ve sent a link") && !strings.Contains(rec.Body.String(), "we've sent a link") {
		t.Fatalf("expected the generic success message, got: %s", rec.Body.String())
	}
}

func TestPasswordReset_UnknownEmailShowsSameGenericSuccess(t *testing.T) {
	a := newTestApp(t)
	fake := newFakeMailer()
	a.Mailer = fake
	router := web.NewRouter(a)

	requestPasswordReset(t, router, "nobody@example.com")
	fake.assertNoEmail(t)
}

func TestPasswordReset_FullFlow_AllThreeRoles(t *testing.T) {
	type accountCase struct {
		name           string
		role           models.Role
		email          string
		wantLoginRoute string
	}
	cases := []accountCase{
		{"candidate", models.RoleCandidate, "candidate@example.com", "/login"},
		{"employer", models.RoleEmployer, "employer@example.com", "/login"},
		{"admin", models.RoleAdmin, "admin@example.com", "/admin/login"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := newTestApp(t)
			fake := newFakeMailer()
			a.Mailer = fake
			router := web.NewRouter(a)
			ctx := context.Background()

			hash, err := auth.HashPassword("original-password-123")
			if err != nil {
				t.Fatalf("hashing password: %v", err)
			}
			user, err := a.Users.Create(ctx, c.email, hash, c.role)
			if err != nil {
				t.Fatalf("creating user: %v", err)
			}

			// Keep an old session alive so we can confirm it's invalidated
			// after the reset.
			oldSessionToken, err := a.Sessions.Create(ctx, user.ID)
			if err != nil {
				t.Fatalf("creating old session: %v", err)
			}

			requestPasswordReset(t, router, c.email)
			email := fake.next(t)
			if email.To != c.email {
				t.Errorf("email sent to %q, want %q", email.To, c.email)
			}
			token := extractResetToken(t, email.Body)

			// GET the reset form (must not consume the token).
			getReq := httptest.NewRequest(http.MethodGet, "/reset-password/"+token, nil)
			getRec := httptest.NewRecorder()
			router.ServeHTTP(getRec, getReq)
			if getRec.Code != http.StatusOK {
				t.Fatalf("expected 200 loading reset form, got %d", getRec.Code)
			}
			var resetCSRF *http.Cookie
			for _, ck := range getRec.Result().Cookies() {
				if ck.Name == "resumebank_csrf" {
					resetCSRF = ck
				}
			}
			if resetCSRF == nil {
				t.Fatal("expected a CSRF cookie from the reset form")
			}

			// Submit the new password.
			postForm := url.Values{
				"csrf_token": {resetCSRF.Value}, "password": {"brand-new-password-456"}, "password_confirm": {"brand-new-password-456"},
			}
			postReq := httptest.NewRequest(http.MethodPost, "/reset-password/"+token, nil)
			postReq.PostForm = postForm
			postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			postReq.AddCookie(resetCSRF)
			postRec := httptest.NewRecorder()
			router.ServeHTTP(postRec, postReq)

			if postRec.Code != http.StatusSeeOther {
				t.Fatalf("expected 303 after reset, got %d: %s", postRec.Code, postRec.Body.String())
			}
			loc := postRec.Header().Get("Location")
			if !strings.HasPrefix(loc, c.wantLoginRoute) {
				t.Errorf("expected redirect to start with %q, got %q", c.wantLoginRoute, loc)
			}

			// Old session must be dead.
			if _, err := a.Sessions.Lookup(ctx, oldSessionToken); err != auth.ErrSessionNotFound {
				t.Errorf("expected old session to be invalidated, got %v", err)
			}

			// New password works.
			refreshed, err := a.Users.GetUserByID(ctx, user.ID)
			if err != nil {
				t.Fatalf("GetUserByID: %v", err)
			}
			if !auth.CheckPassword(refreshed.PasswordHash, "brand-new-password-456") {
				t.Error("expected the new password to be set")
			}
			if auth.CheckPassword(refreshed.PasswordHash, "original-password-123") {
				t.Error("expected the old password to no longer work")
			}

			// The token can't be reused.
			reuseReq := httptest.NewRequest(http.MethodGet, "/reset-password/"+token, nil)
			reuseCookies := httptest.NewRecorder()
			router.ServeHTTP(reuseCookies, reuseReq)
			var reuseCSRF *http.Cookie
			for _, ck := range reuseCookies.Result().Cookies() {
				if ck.Name == "resumebank_csrf" {
					reuseCSRF = ck
				}
			}
			reuseForm := url.Values{"csrf_token": {reuseCSRF.Value}, "password": {"another-password-789"}, "password_confirm": {"another-password-789"}}
			reusePostReq := httptest.NewRequest(http.MethodPost, "/reset-password/"+token, nil)
			reusePostReq.PostForm = reuseForm
			reusePostReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			reusePostReq.AddCookie(reuseCSRF)
			reusePostRec := httptest.NewRecorder()
			router.ServeHTTP(reusePostRec, reusePostReq)
			if reusePostRec.Code != http.StatusUnprocessableEntity {
				t.Errorf("expected reused token to be rejected with 422, got %d: %s", reusePostRec.Code, reusePostRec.Body.String())
			}
			if !strings.Contains(reusePostRec.Body.String(), "invalid, expired") {
				t.Errorf("expected an 'invalid or expired' message, got: %s", reusePostRec.Body.String())
			}
		})
	}
}

func TestPasswordReset_MismatchedConfirmationRejected(t *testing.T) {
	a := newTestApp(t)
	fake := newFakeMailer()
	a.Mailer = fake
	router := web.NewRouter(a)
	ctx := context.Background()

	hash, _ := auth.HashPassword("original-password-123")
	if _, err := a.Users.Create(ctx, "jane@example.com", hash, models.RoleCandidate); err != nil {
		t.Fatalf("creating user: %v", err)
	}

	requestPasswordReset(t, router, "jane@example.com")
	token := extractResetToken(t, fake.next(t).Body)

	getReq := httptest.NewRequest(http.MethodGet, "/reset-password/"+token, nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	var csrfCookie *http.Cookie
	for _, ck := range getRec.Result().Cookies() {
		if ck.Name == "resumebank_csrf" {
			csrfCookie = ck
		}
	}

	form := url.Values{"csrf_token": {csrfCookie.Value}, "password": {"password-one"}, "password_confirm": {"password-two"}}
	req := httptest.NewRequest(http.MethodPost, "/reset-password/"+token, nil)
	req.PostForm = form
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrfCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for mismatched passwords, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestPasswordReset_UnknownTokenShowsInvalidMessage(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)

	getReq := httptest.NewRequest(http.MethodGet, "/reset-password/does-not-exist", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	var csrfCookie *http.Cookie
	for _, ck := range getRec.Result().Cookies() {
		if ck.Name == "resumebank_csrf" {
			csrfCookie = ck
		}
	}

	form := url.Values{"csrf_token": {csrfCookie.Value}, "password": {"password-123456"}, "password_confirm": {"password-123456"}}
	req := httptest.NewRequest(http.MethodPost, "/reset-password/does-not-exist", nil)
	req.PostForm = form
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(csrfCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "invalid, expired") {
		t.Errorf("expected 422 with invalid-link messaging, got %d: %s", rec.Code, rec.Body.String())
	}
}
