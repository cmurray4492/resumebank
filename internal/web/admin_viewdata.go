package web

import (
	"net/http"

	"resumebank/internal/app"
	"resumebank/internal/auth"
	"resumebank/internal/models"
	"resumebank/internal/validate"
)

// AdminPageData is the admin-panel counterpart to PageData. It's a
// deliberately separate type (not reused/extended from PageData) so admin
// identity is never confused with the public site's CurrentUser, which the
// public navbar and templates key off of.
type AdminPageData struct {
	Title     string
	AdminUser *models.User
	CSRFToken string
	Errors    validate.FieldErrors
	Data      any
}

func newAdminPageData(a *app.App, w http.ResponseWriter, r *http.Request, title string, data any) AdminPageData {
	return AdminPageData{
		Title:     title,
		AdminUser: auth.AdminUserFromContext(r.Context()),
		CSRFToken: a.Auth.EnsureCSRFCookie(w, r),
		Errors:    validate.FieldErrors{},
		Data:      data,
	}
}
