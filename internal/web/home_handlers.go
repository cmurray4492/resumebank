package web

import (
	"net/http"

	"resumebank/internal/app"
	"resumebank/internal/models"
	"resumebank/internal/repo"
)

type HomeHandlers struct {
	App *app.App
}

func NewHomeHandlers(a *app.App) *HomeHandlers { return &HomeHandlers{App: a} }

type homeView struct {
	RecentCandidates []repo.CandidateSearchResult
	RecentJobs       []repo.JobSearchResult
}

func (h *HomeHandlers) Show(w http.ResponseWriter, r *http.Request) {
	candidates, _, err := h.App.Candidates.Search(r.Context(), "", 5, 0)
	if err != nil {
		httpServerError(w, err)
		return
	}
	jobs, _, err := h.App.Jobs.Search(r.Context(), "", 5, 0)
	if err != nil {
		httpServerError(w, err)
		return
	}

	view := homeView{RecentCandidates: candidates, RecentJobs: jobs}
	pd := newPageData(h.App, w, r, "resumebank.biz - Connect Candidates and Employers",
		"Browse candidate profiles and job postings, or create your own on resumebank.biz.", view)
	h.App.Renderer.Render(w, http.StatusOK, "home.html.tmpl", pd)
}

func (h *HomeHandlers) Healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// Me redirects a logged-in user to their own profile page.
func (h *HomeHandlers) Me(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	switch u.Role {
	case models.RoleCandidate:
		c, err := h.App.Candidates.GetByUserID(r.Context(), u.ID)
		if err != nil {
			httpServerError(w, err)
			return
		}
		http.Redirect(w, r, "/candidates/"+c.Slug, http.StatusSeeOther)
	case models.RoleEmployer:
		e, err := h.App.Employers.GetByUserID(r.Context(), u.ID)
		if err != nil {
			httpServerError(w, err)
			return
		}
		http.Redirect(w, r, "/employers/"+e.Slug, http.StatusSeeOther)
	default:
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
