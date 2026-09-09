package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"resumebank/internal/models"
	"resumebank/internal/web"
)

func TestMessaging_CandidateCannotMessageAnotherCandidate(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	userA, err := a.Users.Create(ctx, "a@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating user A: %v", err)
	}
	userB, err := a.Users.Create(ctx, "b@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating user B: %v", err)
	}
	token, err := a.Sessions.Create(ctx, userA.ID)
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}

	csrfCookie := fetchCSRFCookie(t, router, token)

	form := url.Values{"body": {"hi"}, "csrf_token": {csrfCookie.Value}}
	req := httptest.NewRequest(http.MethodPost, "/messages/"+strconv.FormatInt(userB.ID, 10), nil)
	req.PostForm = form
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "resumebank_session", Value: token})
	req.AddCookie(csrfCookie)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 (candidate-to-candidate messaging rejected), got %d: %s", rec.Code, rec.Body.String())
	}
}

// fetchCSRFCookie loads a GET page with the given session to pick up the
// CSRF cookie a real browser would send back on the subsequent POST.
func fetchCSRFCookie(t *testing.T, router http.Handler, sessionToken string) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "resumebank_session", Value: sessionToken})
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	for _, c := range rec.Result().Cookies() {
		if c.Name == "resumebank_csrf" {
			return c
		}
	}
	t.Fatal("expected a CSRF cookie to be set")
	return nil
}

func TestMessaging_EmployerCanMessageCandidateEndToEnd(t *testing.T) {
	a := newTestApp(t)
	router := web.NewRouter(a)
	ctx := context.Background()

	candidateUser, err := a.Users.Create(ctx, "candidate@example.com", "hash", models.RoleCandidate)
	if err != nil {
		t.Fatalf("creating candidate user: %v", err)
	}
	if _, err := a.Candidates.Create(ctx, &models.Candidate{
		UserID: candidateUser.ID, Slug: "candidate", Name: "Candidate", Zipcode: "12345",
		Email: "candidate@example.com", ResumeHTML: "<p>R</p>", ResumeText: "R",
	}); err != nil {
		t.Fatalf("creating candidate profile: %v", err)
	}

	employerUser, err := a.Users.Create(ctx, "employer@example.com", "hash", models.RoleEmployer)
	if err != nil {
		t.Fatalf("creating employer user: %v", err)
	}
	token, err := a.Sessions.Create(ctx, employerUser.ID)
	if err != nil {
		t.Fatalf("creating session: %v", err)
	}

	csrfCookie := fetchCSRFCookie(t, router, token)

	form := url.Values{"body": {"Interested in your profile!"}, "csrf_token": {csrfCookie.Value}}
	postReq := httptest.NewRequest(http.MethodPost, "/messages/"+strconv.FormatInt(candidateUser.ID, 10), nil)
	postReq.PostForm = form
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: "resumebank_session", Value: token})
	postReq.AddCookie(csrfCookie)
	postRec := httptest.NewRecorder()
	router.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 after sending message, got %d: %s", postRec.Code, postRec.Body.String())
	}

	unread, err := a.Messages.UnreadCount(ctx, candidateUser.ID)
	if err != nil {
		t.Fatalf("UnreadCount: %v", err)
	}
	if unread != 1 {
		t.Errorf("expected 1 unread message for the candidate, got %d", unread)
	}
}
