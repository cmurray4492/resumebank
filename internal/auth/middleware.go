package auth

import (
	"context"
	"net/http"

	"resumebank/internal/models"
)

const SessionCookieName = "resumebank_session"

// UserGetter loads a user by ID. Satisfied by *repo.UserRepo without auth
// needing to import the concrete repo package.
type UserGetter interface {
	GetUserByID(ctx context.Context, id int64) (*models.User, error)
}

type Middleware struct {
	Sessions     *SessionStore
	Users        UserGetter
	CookieSecure bool
}

func NewMiddleware(sessions *SessionStore, users UserGetter, cookieSecure bool) *Middleware {
	return &Middleware{Sessions: sessions, Users: users, CookieSecure: cookieSecure}
}

// LoadSession attaches the logged-in user (if any) to the request context.
// It never blocks the request; downstream handlers decide what to require.
func (m *Middleware) LoadSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || cookie.Value == "" {
			next.ServeHTTP(w, r)
			return
		}
		userID, err := m.Sessions.Lookup(r.Context(), cookie.Value)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		user, err := m.Users.GetUserByID(r.Context(), userID)
		if err != nil || user == nil {
			next.ServeHTTP(w, r)
			return
		}
		ctx := ContextWithUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAuth rejects the request with a redirect to /login unless a user is
// present in the request context (set by LoadSession).
func (m *Middleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// RequireRole rejects the request with 403 unless the logged-in user has the
// given role. Call after RequireAuth.
func (m *Middleware) RequireRole(role models.Role, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u == nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if u.Role != role {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (m *Middleware) SetSessionCookie(w http.ResponseWriter, rawToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    rawToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((30 * 24 * 60 * 60)),
	})
}

func (m *Middleware) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   m.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
