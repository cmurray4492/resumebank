package web

import (
	"fmt"
	"net/http"

	"resumebank/internal/app"
)

type SitemapHandlers struct {
	App *app.App
}

func NewSitemapHandlers(a *app.App) *SitemapHandlers { return &SitemapHandlers{App: a} }

func (h *SitemapHandlers) Sitemap(w http.ResponseWriter, r *http.Request) {
	data, _ := h.App.Sitemap.Get()
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write(data)
}

func (h *SitemapHandlers) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "User-agent: *\nAllow: /\nSitemap: %s/sitemap.xml\n", h.App.Config.BaseURL)
}
