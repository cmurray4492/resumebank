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

type AdminJobHandlers struct {
	App *app.App
}

func NewAdminJobHandlers(a *app.App) *AdminJobHandlers { return &AdminJobHandlers{App: a} }

type adminJobListView struct {
	Jobs        []repo.JobSearchResult
	Page        int
	PrevPage    int
	NextPage    int
	HasPrevPage bool
	HasNextPage bool
}

func (h *AdminJobHandlers) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * adminListPageSize

	jobs, total, err := h.App.Jobs.Search(r.Context(), "", adminListPageSize, offset)
	if err != nil {
		httpServerError(w, err)
		return
	}

	view := adminJobListView{
		Jobs: jobs, Page: page,
		PrevPage: page - 1, NextPage: page + 1,
		HasPrevPage: page > 1, HasNextPage: offset+adminListPageSize < total,
	}
	pd := newAdminPageData(h.App, w, r, "Jobs", view)
	h.App.Renderer.RenderAdmin(w, http.StatusOK, "admin_jobs_list.html.tmpl", pd)
}

func (h *AdminJobHandlers) EditForm(w http.ResponseWriter, r *http.Request) {
	job, ok := h.loadJob(w, r)
	if !ok {
		return
	}
	pd := newAdminPageData(h.App, w, r, "Edit Job", job)
	h.App.Renderer.RenderAdmin(w, http.StatusOK, "admin_job_edit.html.tmpl", pd)
}

func (h *AdminJobHandlers) Update(w http.ResponseWriter, r *http.Request) {
	job, ok := h.loadJob(w, r)
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

	title := strings.TrimSpace(r.FormValue("title"))
	descriptionHTML := sanitize.SanitizeRichText(r.FormValue("description_html"))

	errs := validate.FieldErrors{}
	validate.Required(title, "title", errs)
	validate.Required(descriptionHTML, "description_html", errs)

	if errs.HasErrors() {
		pd := newAdminPageData(h.App, w, r, "Edit Job", job)
		pd.Errors = errs
		h.App.Renderer.RenderAdmin(w, http.StatusUnprocessableEntity, "admin_job_edit.html.tmpl", pd)
		return
	}

	newDescriptionText := sanitize.PlainText(descriptionHTML)
	descriptionChanged := newDescriptionText != job.DescriptionText

	job.Title = title
	job.Location = strings.TrimSpace(r.FormValue("location"))
	job.JobNumber = strings.TrimSpace(r.FormValue("job_number"))
	job.SalaryMin = optionalInt(strings.TrimSpace(r.FormValue("salary_min")))
	job.SalaryMax = optionalInt(strings.TrimSpace(r.FormValue("salary_max")))
	job.DescriptionHTML = descriptionHTML
	job.DescriptionText = newDescriptionText

	updated, err := h.App.Jobs.Update(r.Context(), job)
	if err != nil {
		httpServerError(w, err)
		return
	}
	if descriptionChanged {
		h.App.TriggerJobEmbedding(updated.ID, updated.DescriptionText)
	}
	http.Redirect(w, r, "/admin/jobs", http.StatusSeeOther)
}

func (h *AdminJobHandlers) loadJob(w http.ResponseWriter, r *http.Request) (*models.Job, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpBadRequest(w, err)
		return nil, false
	}
	job, err := h.App.Jobs.GetByID(r.Context(), id)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
		} else {
			httpServerError(w, err)
		}
		return nil, false
	}
	return job, true
}
