package web

import (
	"net/http"

	"resumebank/internal/app"
)

// StaticPageHandlers serves fixed-content pages (About, Terms) that have no
// dynamic data of their own.
type StaticPageHandlers struct {
	App *app.App
}

func NewStaticPageHandlers(a *app.App) *StaticPageHandlers { return &StaticPageHandlers{App: a} }

func (h *StaticPageHandlers) About(w http.ResponseWriter, r *http.Request) {
	pd := newPageData(h.App, w, r, "About resumebank.biz",
		"Learn about resumebank.biz, a site connecting job seekers and employers.", nil)
	h.App.Renderer.Render(w, http.StatusOK, "about.html.tmpl", pd)
}

func (h *StaticPageHandlers) Terms(w http.ResponseWriter, r *http.Request) {
	pd := newPageData(h.App, w, r, "Terms and Conditions",
		"The terms and conditions for using resumebank.biz.", nil)
	h.App.Renderer.Render(w, http.StatusOK, "terms.html.tmpl", pd)
}
