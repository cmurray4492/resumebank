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

// PasswordResetHandlers implements the "forgot password" flow shared by
// candidates, employers, and admins alike - a reset token proves control of
// an email address regardless of the account's role, so there's one flow
// rather than three. See internal/auth/password_reset.go.
type PasswordResetHandlers struct {
	App *app.App
}

func NewPasswordResetHandlers(a *app.App) *PasswordResetHandlers {
	return &PasswordResetHandlers{App: a}
}

type forgotPasswordView struct {
	Email     string
	Submitted bool
}

func (h *PasswordResetHandlers) ForgotPasswordForm(w http.ResponseWriter, r *http.Request) {
	pd := newPageData(h.App, w, r, "Forgot Password", "Reset your resumebank.biz password.", forgotPasswordView{})
	h.App.Renderer.Render(w, http.StatusOK, "forgot_password.html.tmpl", pd)
}

func (h *PasswordResetHandlers) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		httpBadRequest(w, err)
		return
	}
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))

	errs := validate.FieldErrors{}
	validate.Required(email, "email", errs)
	validate.Email(email, "email", errs)

	if errs.HasErrors() {
		pd := newPageData(h.App, w, r, "Forgot Password", "Reset your resumebank.biz password.", forgotPasswordView{Email: email})
		pd.Errors = errs
		h.App.Renderer.Render(w, http.StatusUnprocessableEntity, "forgot_password.html.tmpl", pd)
		return
	}

	// Deliberately do the same thing (show a generic "check your email"
	// message) whether or not the address has an account, so this form
	// can't be used to discover which emails are registered.
	if user, err := h.App.Users.GetByEmail(r.Context(), email); err == nil {
		h.App.SendPasswordResetEmail(user.ID, user.Email)
	} else if err != repo.ErrNotFound {
		httpServerError(w, err)
		return
	}

	pd := newPageData(h.App, w, r, "Forgot Password", "Reset your resumebank.biz password.", forgotPasswordView{Email: email, Submitted: true})
	h.App.Renderer.Render(w, http.StatusOK, "forgot_password.html.tmpl", pd)
}

type resetPasswordView struct {
	Token   string
	Invalid bool
}

func (h *PasswordResetHandlers) ResetPasswordForm(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	pd := newPageData(h.App, w, r, "Reset Password", "Choose a new password.", resetPasswordView{Token: token})
	h.App.Renderer.Render(w, http.StatusOK, "reset_password.html.tmpl", pd)
}

func (h *PasswordResetHandlers) ResetPassword(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		httpBadRequest(w, err)
		return
	}

	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")

	errs := validate.FieldErrors{}
	validate.Required(password, "password", errs)
	if len(password) > 0 && len(password) < 8 {
		errs.Add("password", "Password must be at least 8 characters.")
	}
	if password != confirm {
		errs.Add("password_confirm", "Passwords do not match.")
	}

	if errs.HasErrors() {
		pd := newPageData(h.App, w, r, "Reset Password", "Choose a new password.", resetPasswordView{Token: token})
		pd.Errors = errs
		h.App.Renderer.Render(w, http.StatusUnprocessableEntity, "reset_password.html.tmpl", pd)
		return
	}

	userID, err := h.App.PasswordResets.Consume(r.Context(), token)
	if err != nil {
		if err != auth.ErrPasswordResetTokenInvalid {
			httpServerError(w, err)
			return
		}
		pd := newPageData(h.App, w, r, "Reset Password", "Choose a new password.", resetPasswordView{Token: token, Invalid: true})
		h.App.Renderer.Render(w, http.StatusUnprocessableEntity, "reset_password.html.tmpl", pd)
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		httpServerError(w, err)
		return
	}
	if err := h.App.Users.SetPassword(r.Context(), userID, passwordHash); err != nil {
		httpServerError(w, err)
		return
	}
	// Log the account out everywhere - both the public site and the admin
	// panel share the sessions table - so a compromised old session can't
	// outlive the password change.
	if err := h.App.Sessions.DeleteAllForUser(r.Context(), userID); err != nil {
		httpServerError(w, err)
		return
	}

	redirectTo := "/login?reset=success"
	if user, err := h.App.Users.GetUserByID(r.Context(), userID); err == nil && user.Role == models.RoleAdmin {
		redirectTo = "/admin/login?reset=success"
	}
	http.Redirect(w, r, redirectTo, http.StatusSeeOther)
}
