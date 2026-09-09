package web

import (
	"net/http"
	"strconv"
	"strings"

	"resumebank/internal/app"
	"resumebank/internal/httpx"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/validate"
)

type AdminEmployerHandlers struct {
	App *app.App
}

func NewAdminEmployerHandlers(a *app.App) *AdminEmployerHandlers {
	return &AdminEmployerHandlers{App: a}
}

type adminEmployerListView struct {
	Employers   []models.Employer
	Page        int
	PrevPage    int
	NextPage    int
	HasPrevPage bool
	HasNextPage bool
}

func (h *AdminEmployerHandlers) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * adminListPageSize

	employers, total, err := h.App.Employers.ListAll(r.Context(), adminListPageSize, offset)
	if err != nil {
		httpServerError(w, err)
		return
	}

	view := adminEmployerListView{
		Employers: employers, Page: page,
		PrevPage: page - 1, NextPage: page + 1,
		HasPrevPage: page > 1, HasNextPage: offset+adminListPageSize < total,
	}
	pd := newAdminPageData(h.App, w, r, "Companies", view)
	h.App.Renderer.RenderAdmin(w, http.StatusOK, "admin_employers_list.html.tmpl", pd)
}

func (h *AdminEmployerHandlers) EditForm(w http.ResponseWriter, r *http.Request) {
	employer, ok := h.loadEmployer(w, r)
	if !ok {
		return
	}
	pd := newAdminPageData(h.App, w, r, "Edit Company", employer)
	h.App.Renderer.RenderAdmin(w, http.StatusOK, "admin_employer_edit.html.tmpl", pd)
}

func (h *AdminEmployerHandlers) Update(w http.ResponseWriter, r *http.Request) {
	employer, ok := h.loadEmployer(w, r)
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
		pd := newAdminPageData(h.App, w, r, "Edit Company", employer)
		pd.Errors = errs
		h.App.Renderer.RenderAdmin(w, http.StatusUnprocessableEntity, "admin_employer_edit.html.tmpl", pd)
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
	http.Redirect(w, r, "/admin/employers", http.StatusSeeOther)
}

func (h *AdminEmployerHandlers) loadEmployer(w http.ResponseWriter, r *http.Request) (*models.Employer, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpBadRequest(w, err)
		return nil, false
	}
	employer, err := h.App.Employers.GetByID(r.Context(), id)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
		} else {
			httpServerError(w, err)
		}
		return nil, false
	}
	return employer, true
}
