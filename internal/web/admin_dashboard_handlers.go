package web

import (
	"net/http"

	"resumebank/internal/app"
)

type AdminDashboardHandlers struct {
	App *app.App
}

func NewAdminDashboardHandlers(a *app.App) *AdminDashboardHandlers {
	return &AdminDashboardHandlers{App: a}
}

type adminDashboardView struct {
	CandidateCount int
	EmployerCount  int
	JobCount       int
	BlogPostCount  int
}

func (h *AdminDashboardHandlers) Show(w http.ResponseWriter, r *http.Request) {
	_, candidateCount, err := h.App.Candidates.Search(r.Context(), "", 1, 0)
	if err != nil {
		httpServerError(w, err)
		return
	}
	_, employerCount, err := h.App.Employers.ListAll(r.Context(), 1, 0)
	if err != nil {
		httpServerError(w, err)
		return
	}
	_, jobCount, err := h.App.Jobs.Search(r.Context(), "", 1, 0)
	if err != nil {
		httpServerError(w, err)
		return
	}
	_, blogCount, err := h.App.Blog.ListAll(r.Context(), 1, 0)
	if err != nil {
		httpServerError(w, err)
		return
	}

	view := adminDashboardView{
		CandidateCount: candidateCount,
		EmployerCount:  employerCount,
		JobCount:       jobCount,
		BlogPostCount:  blogCount,
	}
	pd := newAdminPageData(h.App, w, r, "Dashboard", view)
	h.App.Renderer.RenderAdmin(w, http.StatusOK, "admin_dashboard.html.tmpl", pd)
}
