// Package web implements the HTTP handlers and route table.
package web

import (
	"net/http"

	"resumebank/internal/app"
	"resumebank/internal/models"
	"resumebank/internal/validate"
)

// PageData is passed to every page template. Page-specific fields live in
// Data; shared chrome (nav, CSRF token, current user) is available directly.
type PageData struct {
	Title          string
	Description    string
	CurrentUser    *models.User
	CSRFToken      string
	UnreadMessages int
	Errors         validate.FieldErrors
	Data           any
}

func newPageData(a *app.App, w http.ResponseWriter, r *http.Request, title, description string, data any) PageData {
	u := currentUser(r)
	var unread int
	if u != nil {
		unread, _ = a.Messages.UnreadCount(r.Context(), u.ID)
	}
	return PageData{
		Title:          title,
		Description:    description,
		CurrentUser:    u,
		CSRFToken:      a.Auth.EnsureCSRFCookie(w, r),
		UnreadMessages: unread,
		Errors:         validate.FieldErrors{},
		Data:           data,
	}
}
