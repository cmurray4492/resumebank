package web

import (
	"net/http"
	"strings"

	"resumebank/internal/app"
	"resumebank/internal/httpx"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/sanitize"
	"resumebank/internal/slug"
	"resumebank/internal/validate"
)

type JobHandlers struct {
	App *app.App
}

func NewJobHandlers(a *app.App) *JobHandlers { return &JobHandlers{App: a} }

type jobView struct {
	Job             *models.Job
	CompanyName     string
	EmployerSlug    string
	EmployerHasLogo bool
	IsOwner         bool
	UpVotes         int
	DownVotes       int
	CanVote         bool
	CurrentVote     int16 // +1, -1, or 0 (no vote); only meaningful when CanVote
	ShareURL        string
	ShareTitle      string
}

func (h *JobHandlers) Show(w http.ResponseWriter, r *http.Request) {
	slugVal := r.PathValue("slug")
	job, err := h.App.Jobs.GetBySlug(r.Context(), slugVal)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
			return
		}
		httpServerError(w, err)
		return
	}
	emp, err := h.employerForJob(r, job)
	if err != nil {
		httpServerError(w, err)
		return
	}

	up, down, err := h.App.Votes.Counts(r.Context(), job.ID)
	if err != nil {
		httpServerError(w, err)
		return
	}

	u := currentUser(r)
	view := jobView{
		Job:             job,
		CompanyName:     emp.CompanyName,
		EmployerSlug:    emp.Slug,
		EmployerHasLogo: emp.LogoPath != "",
		IsOwner:         u != nil && u.ID == emp.UserID,
		UpVotes:         up,
		DownVotes:       down,
		ShareURL:        h.App.Config.BaseURL + "/jobs/" + job.Slug,
		ShareTitle:      job.Title + " at " + emp.CompanyName,
	}
	if u != nil && u.Role == models.RoleCandidate {
		if candidate, err := h.App.Candidates.GetByUserID(r.Context(), u.ID); err == nil {
			view.CanVote = true
			view.CurrentVote, err = h.App.Votes.GetVote(r.Context(), candidate.ID, job.ID)
			if err != nil {
				httpServerError(w, err)
				return
			}
		}
	}
	desc := job.Title + " at " + emp.CompanyName
	if job.Location != "" {
		desc += " (" + job.Location + ")"
	}
	pd := newPageData(h.App, w, r, job.Title+" at "+emp.CompanyName, desc, view)
	h.App.Renderer.Render(w, http.StatusOK, "job_show.html.tmpl", pd)
}

// Vote records or toggles a candidate's thumbs up/down on a job. Voting the
// same direction again clears the vote.
func (h *JobHandlers) Vote(w http.ResponseWriter, r *http.Request) {
	slugVal := r.PathValue("slug")
	job, err := h.App.Jobs.GetBySlug(r.Context(), slugVal)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
			return
		}
		httpServerError(w, err)
		return
	}
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}

	u := currentUser(r)
	candidate, err := h.App.Candidates.GetByUserID(r.Context(), u.ID)
	if err != nil {
		httpServerError(w, err)
		return
	}

	var direction int16
	switch r.FormValue("direction") {
	case "up":
		direction = 1
	case "down":
		direction = -1
	default:
		httpx.BadRequest(w, "direction must be \"up\" or \"down\"")
		return
	}

	current, err := h.App.Votes.GetVote(r.Context(), candidate.ID, job.ID)
	if err != nil {
		httpServerError(w, err)
		return
	}
	if current == direction {
		err = h.App.Votes.Clear(r.Context(), candidate.ID, job.ID)
	} else {
		err = h.App.Votes.Set(r.Context(), candidate.ID, job.ID, direction)
	}
	if err != nil {
		httpServerError(w, err)
		return
	}

	http.Redirect(w, r, "/jobs/"+job.Slug, http.StatusSeeOther)
}

func (h *JobHandlers) NewForm(w http.ResponseWriter, r *http.Request) {
	employer, ok := h.loadOwnedEmployerForJobs(w, r)
	if !ok {
		return
	}
	pd := newPageData(h.App, w, r, "Post a Job", "", employer)
	h.App.Renderer.Render(w, http.StatusOK, "job_new.html.tmpl", pd)
}

func (h *JobHandlers) Create(w http.ResponseWriter, r *http.Request) {
	employer, ok := h.loadOwnedEmployerForJobs(w, r)
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
	applyMethod, applyValue, applyErrs := parseApplyFields(r)

	errs := validate.FieldErrors{}
	validate.Required(title, "title", errs)
	validate.Required(descriptionHTML, "description_html", errs)
	for field, msg := range applyErrs {
		errs.Add(field, msg)
	}

	if errs.HasErrors() {
		pd := newPageData(h.App, w, r, "Post a Job", "", employer)
		pd.Errors = errs
		h.App.Renderer.Render(w, http.StatusUnprocessableEntity, "job_new.html.tmpl", pd)
		return
	}

	baseSlug := slug.Make(employer.CompanyName + " " + title)
	uniqueSlugVal, err := uniqueSlug(r.Context(), baseSlug, h.App.Jobs.SlugExists)
	if err != nil {
		httpServerError(w, err)
		return
	}

	job := &models.Job{
		EmployerID:      employer.ID,
		Slug:            uniqueSlugVal,
		Title:           title,
		Location:        strings.TrimSpace(r.FormValue("location")),
		JobNumber:       strings.TrimSpace(r.FormValue("job_number")),
		SalaryMin:       optionalInt(strings.TrimSpace(r.FormValue("salary_min"))),
		SalaryMax:       optionalInt(strings.TrimSpace(r.FormValue("salary_max"))),
		DescriptionHTML: descriptionHTML,
		DescriptionText: sanitize.PlainText(descriptionHTML),
		ApplyMethod:     applyMethod,
		ApplyValue:      applyValue,
	}
	created, err := h.App.Jobs.Create(r.Context(), job)
	if err != nil {
		httpServerError(w, err)
		return
	}
	h.App.TriggerJobEmbedding(created.ID, created.DescriptionText)
	http.Redirect(w, r, "/jobs/"+job.Slug, http.StatusSeeOther)
}

func (h *JobHandlers) EditForm(w http.ResponseWriter, r *http.Request) {
	job, _, ok := h.loadOwnedJob(w, r)
	if !ok {
		return
	}
	pd := newPageData(h.App, w, r, "Edit Job Posting", "", job)
	h.App.Renderer.Render(w, http.StatusOK, "job_edit.html.tmpl", pd)
}

func (h *JobHandlers) Update(w http.ResponseWriter, r *http.Request) {
	job, _, ok := h.loadOwnedJob(w, r)
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
	applyMethod, applyValue, applyErrs := parseApplyFields(r)

	errs := validate.FieldErrors{}
	validate.Required(title, "title", errs)
	validate.Required(descriptionHTML, "description_html", errs)
	for field, msg := range applyErrs {
		errs.Add(field, msg)
	}

	if errs.HasErrors() {
		pd := newPageData(h.App, w, r, "Edit Job Posting", "", job)
		pd.Errors = errs
		h.App.Renderer.Render(w, http.StatusUnprocessableEntity, "job_edit.html.tmpl", pd)
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
	job.ApplyMethod = applyMethod
	job.ApplyValue = applyValue

	updated, err := h.App.Jobs.Update(r.Context(), job)
	if err != nil {
		httpServerError(w, err)
		return
	}
	if descriptionChanged {
		h.App.TriggerJobEmbedding(updated.ID, updated.DescriptionText)
	}
	http.Redirect(w, r, "/jobs/"+job.Slug, http.StatusSeeOther)
}

func (h *JobHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	job, employer, ok := h.loadOwnedJob(w, r)
	if !ok {
		return
	}
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := h.App.Jobs.Delete(r.Context(), job.ID); err != nil {
		httpServerError(w, err)
		return
	}
	http.Redirect(w, r, "/employers/"+employer.Slug, http.StatusSeeOther)
}

// loadOwnedEmployerForJobs loads the employer by slug (from the URL) and
// verifies the logged-in user owns it, for job-creation routes nested under
// /employers/{slug}/jobs.
func (h *JobHandlers) loadOwnedEmployerForJobs(w http.ResponseWriter, r *http.Request) (*models.Employer, bool) {
	slugVal := r.PathValue("slug")
	employer, err := h.App.Employers.GetBySlug(r.Context(), slugVal)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
		} else {
			httpServerError(w, err)
		}
		return nil, false
	}
	u := currentUser(r)
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil, false
	}
	if u.ID != employer.UserID {
		http.Error(w, "Forbidden: you can only post jobs for your own company", http.StatusForbidden)
		return nil, false
	}
	return employer, true
}

// loadOwnedJob loads the job by slug and verifies the logged-in user owns
// the employer that posted it.
func (h *JobHandlers) loadOwnedJob(w http.ResponseWriter, r *http.Request) (*models.Job, *models.Employer, bool) {
	slugVal := r.PathValue("slug")
	job, err := h.App.Jobs.GetBySlug(r.Context(), slugVal)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
		} else {
			httpServerError(w, err)
		}
		return nil, nil, false
	}
	employer, err := h.employerForJob(r, job)
	if err != nil {
		httpServerError(w, err)
		return nil, nil, false
	}
	u := currentUser(r)
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return nil, nil, false
	}
	if u.ID != employer.UserID {
		http.Error(w, "Forbidden: you can only edit your own company's jobs", http.StatusForbidden)
		return nil, nil, false
	}
	return job, employer, true
}

func (h *JobHandlers) employerForJob(r *http.Request, job *models.Job) (*models.Employer, error) {
	return h.App.Employers.GetByID(r.Context(), job.EmployerID)
}

// parseApplyFields reads and validates the job posting's apply method
// (a URL or an email address) shared by the employer and admin job forms.
func parseApplyFields(r *http.Request) (method, value string, errs validate.FieldErrors) {
	errs = validate.FieldErrors{}
	method = r.FormValue("apply_method")
	value = strings.TrimSpace(r.FormValue("apply_value"))

	if method != "url" && method != "email" {
		errs.Add("apply_method", "Choose how candidates should apply.")
		return method, value, errs
	}
	validate.Required(value, "apply_value", errs)
	if method == "url" {
		validate.URL(value, "apply_value", errs)
	} else {
		validate.Email(value, "apply_value", errs)
	}
	return method, value, errs
}
