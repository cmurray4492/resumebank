package web

import (
	"net/http"
	"strings"

	"resumebank/internal/app"
	"resumebank/internal/auth"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/validate"
)

type AdminAuthHandlers struct {
	App *app.App
}

func NewAdminAuthHandlers(a *app.App) *AdminAuthHandlers { return &AdminAuthHandlers{App: a} }

func (h *AdminAuthHandlers) LoginForm(w http.ResponseWriter, r *http.Request) {
	if auth.AdminUserFromContext(r.Context()) != nil {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}
	pd := newAdminPageData(h.App, w, r, "Admin Login", nil)
	h.App.Renderer.RenderAdmin(w, http.StatusOK, "admin_login.html.tmpl", pd)
}

func (h *AdminAuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		httpBadRequest(w, err)
		return
	}
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	fail := func() {
		errs := validate.FieldErrors{"email": "Invalid email or password."}
		pd := newAdminPageData(h.App, w, r, "Admin Login", nil)
		pd.Errors = errs
		h.App.Renderer.RenderAdmin(w, http.StatusUnprocessableEntity, "admin_login.html.tmpl", pd)
	}

	user, err := h.App.Users.GetByEmail(r.Context(), email)
	if err != nil {
		if err != repo.ErrNotFound {
			httpServerError(w, err)
			return
		}
		fail()
		return
	}
	// Deliberately the same generic failure for "not an admin" as for "wrong
	// password", so this login form can't be used to discover which emails
	// have admin access.
	if user.Role != models.RoleAdmin || !auth.CheckPassword(user.PasswordHash, password) {
		fail()
		return
	}

	_ = h.App.Users.UpdateLastLogin(r.Context(), user.ID)
	token, err := h.App.Sessions.Create(r.Context(), user.ID)
	if err != nil {
		httpServerError(w, err)
		return
	}
	h.App.Auth.SetAdminSessionCookie(w, token)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (h *AdminAuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if cookie, err := r.Cookie(auth.AdminSessionCookieName); err == nil {
		_ = h.App.Sessions.Delete(r.Context(), cookie.Value)
	}
	h.App.Auth.ClearAdminSessionCookie(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}
