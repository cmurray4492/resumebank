package web

import (
	"log"
	"net/http"
	"strings"

	"resumebank/internal/app"
	"resumebank/internal/auth"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/sanitize"
	"resumebank/internal/slug"
	"resumebank/internal/validate"
)

type AuthHandlers struct {
	App *app.App
}

func NewAuthHandlers(a *app.App) *AuthHandlers { return &AuthHandlers{App: a} }

func (h *AuthHandlers) SignupCandidateForm(w http.ResponseWriter, r *http.Request) {
	pd := newPageData(h.App, w, r, "Create a Candidate Account", "Sign up to build your candidate profile on resumebank.biz.", nil)
	h.App.Renderer.Render(w, http.StatusOK, "signup_candidate.html.tmpl", pd)
}

func (h *AuthHandlers) SignupCandidate(w http.ResponseWriter, r *http.Request) {
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		httpBadRequest(w, err)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	name := strings.TrimSpace(r.FormValue("name"))
	zipcode := strings.TrimSpace(r.FormValue("zipcode"))
	resumeHTML := sanitize.SanitizeRichText(r.FormValue("resume_html"))

	errs := validate.FieldErrors{}
	validate.Email(email, "email", errs)
	validate.Required(email, "email", errs)
	validate.Required(password, "password", errs)
	validate.MaxLen(password, "password", 200, errs)
	if len(password) > 0 && len(password) < 8 {
		errs.Add("password", "Password must be at least 8 characters.")
	}
	validate.Required(name, "name", errs)
	validate.Zipcode(zipcode, "zipcode", errs)
	validate.Required(zipcode, "zipcode", errs)
	validate.Required(resumeHTML, "resume_html", errs)

	if errs.HasErrors() {
		h.rerenderCandidateSignup(w, r, errs)
		return
	}

	if _, err := h.App.Users.GetByEmail(r.Context(), email); err == nil {
		errs.Add("email", "An account with this email already exists.")
		h.rerenderCandidateSignup(w, r, errs)
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		httpServerError(w, err)
		return
	}

	user, err := h.App.Users.Create(r.Context(), email, passwordHash, models.RoleCandidate)
	if err != nil {
		httpServerError(w, err)
		return
	}

	baseSlug := slug.Make(name)
	uniqueSlugVal, err := uniqueSlug(r.Context(), baseSlug, h.App.Candidates.SlugExists)
	if err != nil {
		httpServerError(w, err)
		return
	}

	candidate := &models.Candidate{
		UserID:      user.ID,
		Slug:        uniqueSlugVal,
		Name:        name,
		Title:       strings.TrimSpace(r.FormValue("title")),
		City:        strings.TrimSpace(r.FormValue("city")),
		State:       strings.TrimSpace(r.FormValue("state")),
		Zipcode:     zipcode,
		Email:       email,
		Phone:       strings.TrimSpace(r.FormValue("phone")),
		LinkedInURL: strings.TrimSpace(r.FormValue("linkedin_url")),
		Skills:      strings.TrimSpace(r.FormValue("skills")),
		Summary:     strings.TrimSpace(r.FormValue("summary")),
		ResumeHTML:  resumeHTML,
		ResumeText:  sanitize.PlainText(resumeHTML),
	}
	created, err := h.App.Candidates.Create(r.Context(), candidate)
	if err != nil {
		// Compensate for the orphaned user row; Phase 1 doesn't wrap this in
		// a DB transaction since repos operate on the shared pool directly.
		log.Printf("candidate creation failed after user creation, user id=%d: %v", user.ID, err)
		httpServerError(w, err)
		return
	}
	h.App.TriggerCandidateEmbedding(created.ID, created.ResumeText)

	h.startSession(w, r, user.ID)
	http.Redirect(w, r, "/candidates/"+candidate.Slug, http.StatusSeeOther)
}

func (h *AuthHandlers) rerenderCandidateSignup(w http.ResponseWriter, r *http.Request, errs validate.FieldErrors) {
	pd := newPageData(h.App, w, r, "Create a Candidate Account", "Sign up to build your candidate profile on resumebank.biz.", r.Form)
	pd.Errors = errs
	h.App.Renderer.Render(w, http.StatusUnprocessableEntity, "signup_candidate.html.tmpl", pd)
}

func (h *AuthHandlers) SignupEmployerForm(w http.ResponseWriter, r *http.Request) {
	pd := newPageData(h.App, w, r, "Create an Employer Account", "Sign up to post jobs and showcase your company on resumebank.biz.", nil)
	h.App.Renderer.Render(w, http.StatusOK, "signup_employer.html.tmpl", pd)
}

func (h *AuthHandlers) SignupEmployer(w http.ResponseWriter, r *http.Request) {
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		httpBadRequest(w, err)
		return
	}

	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")
	companyName := strings.TrimSpace(r.FormValue("company_name"))
	zipcode := strings.TrimSpace(r.FormValue("zipcode"))

	errs := validate.FieldErrors{}
	validate.Email(email, "email", errs)
	validate.Required(email, "email", errs)
	validate.Required(password, "password", errs)
	if len(password) > 0 && len(password) < 8 {
		errs.Add("password", "Password must be at least 8 characters.")
	}
	validate.Required(companyName, "company_name", errs)
	validate.Zipcode(zipcode, "zipcode", errs)
	validate.Required(zipcode, "zipcode", errs)

	if errs.HasErrors() {
		h.rerenderEmployerSignup(w, r, errs)
		return
	}

	if _, err := h.App.Users.GetByEmail(r.Context(), email); err == nil {
		errs.Add("email", "An account with this email already exists.")
		h.rerenderEmployerSignup(w, r, errs)
		return
	}

	passwordHash, err := auth.HashPassword(password)
	if err != nil {
		httpServerError(w, err)
		return
	}

	user, err := h.App.Users.Create(r.Context(), email, passwordHash, models.RoleEmployer)
	if err != nil {
		httpServerError(w, err)
		return
	}

	baseSlug := slug.Make(companyName)
	uniqueSlugVal, err := uniqueSlug(r.Context(), baseSlug, h.App.Employers.SlugExists)
	if err != nil {
		httpServerError(w, err)
		return
	}

	employer := &models.Employer{
		UserID:       user.ID,
		Slug:         uniqueSlugVal,
		CompanyName:  companyName,
		Industry:     strings.TrimSpace(r.FormValue("industry")),
		City:         strings.TrimSpace(r.FormValue("city")),
		State:        strings.TrimSpace(r.FormValue("state")),
		Zipcode:      zipcode,
		Phone:        strings.TrimSpace(r.FormValue("phone")),
		EmailAddress: strings.TrimSpace(r.FormValue("email_address")),
		Website:      strings.TrimSpace(r.FormValue("website")),
		Description:  strings.TrimSpace(r.FormValue("description")),
		Locations:    strings.TrimSpace(r.FormValue("locations")),
	}
	if _, err := h.App.Employers.Create(r.Context(), employer); err != nil {
		log.Printf("employer creation failed after user creation, user id=%d: %v", user.ID, err)
		httpServerError(w, err)
		return
	}

	h.startSession(w, r, user.ID)
	http.Redirect(w, r, "/employers/"+employer.Slug, http.StatusSeeOther)
}

func (h *AuthHandlers) rerenderEmployerSignup(w http.ResponseWriter, r *http.Request, errs validate.FieldErrors) {
	pd := newPageData(h.App, w, r, "Create an Employer Account", "Sign up to post jobs and showcase your company on resumebank.biz.", r.Form)
	pd.Errors = errs
	h.App.Renderer.Render(w, http.StatusUnprocessableEntity, "signup_employer.html.tmpl", pd)
}

type loginView struct {
	PasswordWasReset bool
}

func (h *AuthHandlers) LoginForm(w http.ResponseWriter, r *http.Request) {
	view := loginView{PasswordWasReset: r.URL.Query().Get("reset") == "success"}
	pd := newPageData(h.App, w, r, "Log In", "Log in to your resumebank.biz account.", view)
	h.App.Renderer.Render(w, http.StatusOK, "login.html.tmpl", pd)
}

func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		httpBadRequest(w, err)
		return
	}
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	password := r.FormValue("password")

	fail := func() {
		errs := validate.FieldErrors{"email": "Invalid email or password."}
		pd := newPageData(h.App, w, r, "Log In", "Log in to your resumebank.biz account.", loginView{})
		pd.Errors = errs
		h.App.Renderer.Render(w, http.StatusUnprocessableEntity, "login.html.tmpl", pd)
	}

	user, err := h.App.Users.GetByEmail(r.Context(), email)
	if err != nil {
		if err != repo.ErrNotFound {
			httpServerError(w, err)
			return
		}
		fail()
		return
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		fail()
		return
	}

	_ = h.App.Users.UpdateLastLogin(r.Context(), user.ID)
	h.startSession(w, r, user.ID)

	switch user.Role {
	case models.RoleCandidate:
		c, err := h.App.Candidates.GetByUserID(r.Context(), user.ID)
		if err == nil {
			http.Redirect(w, r, "/candidates/"+c.Slug, http.StatusSeeOther)
			return
		}
	case models.RoleEmployer:
		e, err := h.App.Employers.GetByUserID(r.Context(), user.ID)
		if err == nil {
			http.Redirect(w, r, "/employers/"+e.Slug, http.StatusSeeOther)
			return
		}
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if cookie, err := r.Cookie(auth.SessionCookieName); err == nil {
		_ = h.App.Sessions.Delete(r.Context(), cookie.Value)
	}
	h.App.Auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandlers) startSession(w http.ResponseWriter, r *http.Request, userID int64) {
	token, err := h.App.Sessions.Create(r.Context(), userID)
	if err != nil {
		log.Printf("failed to create session for user %d: %v", userID, err)
		return
	}
	h.App.Auth.SetSessionCookie(w, token)
}
