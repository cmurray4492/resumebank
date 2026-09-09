package auth

import (
	"context"
	"net/http"

	"resumebank/internal/models"
)

// AdminSessionCookieName is deliberately separate from SessionCookieName
// (and scoped to Path=/admin below) so an admin's session never collides
// with, or is sent alongside, a regular candidate/employer session in the
// same browser, and so admin identity is never confused with the public
// site's "current user" (used by the public navbar/templates).
const AdminSessionCookieName = "resumebank_admin_session"

type adminContextKey int

const adminUserContextKey adminContextKey = iota

func ContextWithAdminUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, adminUserContextKey, u)
}

func AdminUserFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(adminUserContextKey).(*models.User)
	return u
}

// LoadAdminSession attaches the logged-in admin (if any) to the request
// context under a separate key from LoadSession's regular user. It never
// blocks the request; RequireAdmin decides what to require.
func (m *Middleware) LoadAdminSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(AdminSessionCookieName)
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
		if err != nil || user == nil || user.Role != models.RoleAdmin {
			next.ServeHTTP(w, r)
			return
		}
		ctx := ContextWithAdminUser(r.Context(), user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin rejects the request with a redirect to /admin/login unless an
// admin is present in the request context (set by LoadAdminSession).
func (m *Middleware) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if AdminUserFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (m *Middleware) SetAdminSessionCookie(w http.ResponseWriter, rawToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     AdminSessionCookieName,
		Value:    rawToken,
		Path:     "/admin",
		HttpOnly: true,
		Secure:   m.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(12 * 60 * 60), // 12 hours - shorter-lived than the public 30-day session
	})
}

func (m *Middleware) ClearAdminSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     AdminSessionCookieName,
		Value:    "",
		Path:     "/admin",
		HttpOnly: true,
		Secure:   m.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
