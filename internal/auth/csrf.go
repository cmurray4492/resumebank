package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

// CSRF uses the double-submit-cookie pattern: a random token is set in a
// non-HttpOnly cookie (so templates can read it into a hidden form field)
// and every state-changing request must echo it back in the form body.
// This works even for pre-login forms (signup/login) since it doesn't
// depend on an authenticated session.
const csrfCookieName = "resumebank_csrf"
const CSRFFormField = "csrf_token"

func newCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// EnsureCSRFCookie returns the current request's CSRF token, setting a new
// cookie if one isn't already present.
func (m *Middleware) EnsureCSRFCookie(w http.ResponseWriter, r *http.Request) string {
	if cookie, err := r.Cookie(csrfCookieName); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	token, err := newCSRFToken()
	if err != nil {
		return ""
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   m.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
	return token
}

// VerifyCSRF checks the csrf_token form field against the CSRF cookie.
func (m *Middleware) VerifyCSRF(r *http.Request) bool {
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	submitted := r.FormValue(CSRFFormField)
	if submitted == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(submitted)) == 1
}

// RequireCSRF wraps a POST handler, rejecting requests whose CSRF token
// doesn't match.
func (m *Middleware) RequireCSRF(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.VerifyCSRF(r) {
			http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
