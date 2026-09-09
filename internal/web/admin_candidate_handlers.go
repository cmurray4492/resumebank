package web

import (
	"net/http"
	"strconv"
	"strings"

	"resumebank/internal/app"
	"resumebank/internal/httpx"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/sanitize"
	"resumebank/internal/validate"
)

type AdminCandidateHandlers struct {
	App *app.App
}

func NewAdminCandidateHandlers(a *app.App) *AdminCandidateHandlers {
	return &AdminCandidateHandlers{App: a}
}

const adminListPageSize = 20

type adminCandidateListView struct {
	Candidates  []repo.CandidateSearchResult
	Page        int
	PrevPage    int
	NextPage    int
	HasPrevPage bool
	HasNextPage bool
}

func (h *AdminCandidateHandlers) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * adminListPageSize

	candidates, total, err := h.App.Candidates.Search(r.Context(), "", adminListPageSize, offset)
	if err != nil {
		httpServerError(w, err)
		return
	}

	view := adminCandidateListView{
		Candidates: candidates, Page: page,
		PrevPage: page - 1, NextPage: page + 1,
		HasPrevPage: page > 1, HasNextPage: offset+adminListPageSize < total,
	}
	pd := newAdminPageData(h.App, w, r, "Candidates", view)
	h.App.Renderer.RenderAdmin(w, http.StatusOK, "admin_candidates_list.html.tmpl", pd)
}

func (h *AdminCandidateHandlers) EditForm(w http.ResponseWriter, r *http.Request) {
	candidate, ok := h.loadCandidate(w, r)
	if !ok {
		return
	}
	pd := newAdminPageData(h.App, w, r, "Edit Candidate", candidate)
	h.App.Renderer.RenderAdmin(w, http.StatusOK, "admin_candidate_edit.html.tmpl", pd)
}

func (h *AdminCandidateHandlers) Update(w http.ResponseWriter, r *http.Request) {
	candidate, ok := h.loadCandidate(w, r)
	if !ok {
		return
	}
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		httpBadRequest(w, err)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	zipcode := strings.TrimSpace(r.FormValue("zipcode"))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	resumeHTML := sanitize.SanitizeRichText(r.FormValue("resume_html"))

	errs := validate.FieldErrors{}
	validate.Required(name, "name", errs)
	validate.Zipcode(zipcode, "zipcode", errs)
	validate.Required(zipcode, "zipcode", errs)
	validate.Email(email, "email", errs)
	validate.Required(email, "email", errs)
	validate.Required(resumeHTML, "resume_html", errs)

	if errs.HasErrors() {
		pd := newAdminPageData(h.App, w, r, "Edit Candidate", candidate)
		pd.Errors = errs
		h.App.Renderer.RenderAdmin(w, http.StatusUnprocessableEntity, "admin_candidate_edit.html.tmpl", pd)
		return
	}

	newResumeText := sanitize.PlainText(resumeHTML)
	resumeChanged := newResumeText != candidate.ResumeText

	candidate.Name = name
	candidate.Title = strings.TrimSpace(r.FormValue("title"))
	candidate.City = strings.TrimSpace(r.FormValue("city"))
	candidate.State = strings.TrimSpace(r.FormValue("state"))
	candidate.Zipcode = zipcode
	candidate.Email = email
	candidate.LinkedInURL = strings.TrimSpace(r.FormValue("linkedin_url"))
	candidate.Skills = strings.TrimSpace(r.FormValue("skills"))
	candidate.Summary = strings.TrimSpace(r.FormValue("summary"))
	candidate.ResumeHTML = resumeHTML
	candidate.ResumeText = newResumeText

	updated, err := h.App.Candidates.Update(r.Context(), candidate)
	if err != nil {
		httpServerError(w, err)
		return
	}
	if resumeChanged {
		h.App.TriggerCandidateEmbedding(updated.ID, updated.ResumeText)
	}
	http.Redirect(w, r, "/admin/candidates", http.StatusSeeOther)
}

func (h *AdminCandidateHandlers) loadCandidate(w http.ResponseWriter, r *http.Request) (*models.Candidate, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpBadRequest(w, err)
		return nil, false
	}
	candidate, err := h.App.Candidates.GetByID(r.Context(), id)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
		} else {
			httpServerError(w, err)
		}
		return nil, false
	}
	return candidate, true
}
