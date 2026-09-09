package web

import (
	"fmt"
	"net/http"
	"strings"

	"resumebank/internal/app"
	"resumebank/internal/httpx"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/validate"
)

type EmployerHandlers struct {
	App *app.App
}

func NewEmployerHandlers(a *app.App) *EmployerHandlers { return &EmployerHandlers{App: a} }

type employerProfileView struct {
	Employer *models.Employer
	Jobs     []models.Job
	IsOwner  bool
}

func (h *EmployerHandlers) Show(w http.ResponseWriter, r *http.Request) {
	slugVal := r.PathValue("slug")
	employer, err := h.App.Employers.GetBySlug(r.Context(), slugVal)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
			return
		}
		httpServerError(w, err)
		return
	}
	jobs, err := h.App.Jobs.ListByEmployer(r.Context(), employer.ID)
	if err != nil {
		httpServerError(w, err)
		return
	}

	u := currentUser(r)
	view := employerProfileView{
		Employer: employer,
		Jobs:     jobs,
		IsOwner:  u != nil && u.ID == employer.UserID,
	}
	desc := employer.Description
	if desc == "" {
		desc = fmt.Sprintf("%s is hiring on resumebank.biz. View open positions and company details.", employer.CompanyName)
	}
	pd := newPageData(h.App, w, r, employer.CompanyName, desc, view)
	h.App.Renderer.Render(w, http.StatusOK, "employer_profile.html.tmpl", pd)
}

func (h *EmployerHandlers) EditForm(w http.ResponseWriter, r *http.Request) {
	employer, ok := h.loadOwnedEmployer(w, r)
	if !ok {
		return
	}
	pd := newPageData(h.App, w, r, "Edit Company Profile", "", employer)
	h.App.Renderer.Render(w, http.StatusOK, "employer_edit.html.tmpl", pd)
}

func (h *EmployerHandlers) Update(w http.ResponseWriter, r *http.Request) {
	employer, ok := h.loadOwnedEmployer(w, r)
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

	companyName := strings.TrimSpace(r.FormValue("company_name"))
	zipcode := strings.TrimSpace(r.FormValue("zipcode"))

	errs := validate.FieldErrors{}
	validate.Required(companyName, "company_name", errs)
	validate.Zipcode(zipcode, "zipcode", errs)
	validate.Required(zipcode, "zipcode", errs)

	if errs.HasErrors() {
		pd := newPageData(h.App, w, r, "Edit Company Profile", "", employer)
		pd.Errors = errs
		h.App.Renderer.Render(w, http.StatusUnprocessableEntity, "employer_edit.html.tmpl", pd)
		return
	}

	employer.CompanyName = companyName
	employer.Industry = strings.TrimSpace(r.FormValue("industry"))
	employer.City = strings.TrimSpace(r.FormValue("city"))
	employer.State = strings.TrimSpace(r.FormValue("state"))
	employer.Zipcode = zipcode
	employer.Phone = strings.TrimSpace(r.FormValue("phone"))
	employer.EmailAddress = strings.TrimSpace(r.FormValue("email_address"))
	employer.Website = strings.TrimSpace(r.FormValue("website"))
	employer.Description = strings.TrimSpace(r.FormValue("description"))
	employer.Locations = strings.TrimSpace(r.FormValue("locations"))

	if _, err := h.App.Employers.Update(r.Context(), employer); err != nil {
		httpServerError(w, err)
		return
	}
	http.Redirect(w, r, "/employers/"+employer.Slug, http.StatusSeeOther)
}

func (h *EmployerHandlers) loadOwnedEmployer(w http.ResponseWriter, r *http.Request) (*models.Employer, bool) {
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
		http.Error(w, "Forbidden: you can only edit your own company profile", http.StatusForbidden)
		return nil, false
	}
	return employer, true
}
