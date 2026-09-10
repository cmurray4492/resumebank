package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"resumebank/internal/app"
	"resumebank/internal/httpx"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/sanitize"
	"resumebank/internal/storage"
	"resumebank/internal/validate"
)

type CandidateHandlers struct {
	App *app.App
}

func NewCandidateHandlers(a *app.App) *CandidateHandlers { return &CandidateHandlers{App: a} }

type candidateProfileView struct {
	Candidate *models.Candidate
	Files     []models.CandidateFile
	IsOwner   bool
}

func (h *CandidateHandlers) Show(w http.ResponseWriter, r *http.Request) {
	slugVal := r.PathValue("slug")
	candidate, err := h.App.Candidates.GetBySlug(r.Context(), slugVal)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
			return
		}
		httpServerError(w, err)
		return
	}
	files, err := h.App.Files.ListByCandidate(r.Context(), candidate.ID)
	if err != nil {
		httpServerError(w, err)
		return
	}

	u := currentUser(r)
	view := candidateProfileView{
		Candidate: candidate,
		Files:     files,
		IsOwner:   u != nil && u.ID == candidate.UserID,
	}
	desc := candidate.Summary
	if desc == "" {
		desc = fmt.Sprintf("%s - %s. View resume and contact details on resumebank.biz.", candidate.Name, candidate.Title)
	}
	pd := newPageData(h.App, w, r, candidate.Name+" - "+candidate.Title, desc, view)
	h.App.Renderer.Render(w, http.StatusOK, "candidate_profile.html.tmpl", pd)
}

func (h *CandidateHandlers) EditForm(w http.ResponseWriter, r *http.Request) {
	candidate, ok := h.loadOwnedCandidate(w, r)
	if !ok {
		return
	}
	view, err := h.editView(r, candidate)
	if err != nil {
		httpServerError(w, err)
		return
	}
	pd := newPageData(h.App, w, r, "Edit Your Profile", "", view)
	h.App.Renderer.Render(w, http.StatusOK, "candidate_edit.html.tmpl", pd)
}

func (h *CandidateHandlers) editView(r *http.Request, candidate *models.Candidate) (candidateProfileView, error) {
	files, err := h.App.Files.ListByCandidate(r.Context(), candidate.ID)
	if err != nil {
		return candidateProfileView{}, err
	}
	return candidateProfileView{Candidate: candidate, Files: files, IsOwner: true}, nil
}

func (h *CandidateHandlers) Update(w http.ResponseWriter, r *http.Request) {
	candidate, ok := h.loadOwnedCandidate(w, r)
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
		view, err := h.editView(r, candidate)
		if err != nil {
			httpServerError(w, err)
			return
		}
		pd := newPageData(h.App, w, r, "Edit Your Profile", "", view)
		pd.Errors = errs
		h.App.Renderer.Render(w, http.StatusUnprocessableEntity, "candidate_edit.html.tmpl", pd)
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
	candidate.Phone = strings.TrimSpace(r.FormValue("phone"))
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
	http.Redirect(w, r, "/candidates/"+candidate.Slug, http.StatusSeeOther)
}

const maxAdditionalFiles = 3

func (h *CandidateHandlers) UploadFile(w http.ResponseWriter, r *http.Request) {
	candidate, ok := h.loadOwnedCandidate(w, r)
	if !ok {
		return
	}
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.App.Config.MaxUploadBytes)
	if err := r.ParseMultipartForm(h.App.Config.MaxUploadBytes); err != nil {
		httpBadRequest(w, fmt.Errorf("file too large or invalid form: %w", err))
		return
	}

	kindParam := r.FormValue("kind")
	isResume := kindParam == "resume_pdf"
	kind := models.FileKindAdditional
	if isResume {
		kind = models.FileKindResumePDF
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpBadRequest(w, err)
		return
	}
	defer file.Close()

	if err := storage.ValidateExtension(header.Filename, isResume); err != nil {
		httpBadRequest(w, err)
		return
	}

	if !isResume {
		count, err := h.App.Files.CountByKind(r.Context(), candidate.ID, models.FileKindAdditional)
		if err != nil {
			httpServerError(w, err)
			return
		}
		if count >= maxAdditionalFiles {
			httpBadRequest(w, fmt.Errorf("you can only upload up to %d additional files", maxAdditionalFiles))
			return
		}
	}

	safeName := storage.SafeFilename(header.Filename)
	key := storage.CandidateFileKey(candidate.ID, safeName)
	if err := h.App.Storage.Save(key, file); err != nil {
		httpServerError(w, err)
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	record := &models.CandidateFile{
		CandidateID:      candidate.ID,
		Kind:             kind,
		OriginalFilename: header.Filename,
		StoredPath:       key,
		ContentType:      contentType,
		SizeBytes:        header.Size,
	}
	if _, err := h.App.Files.Create(r.Context(), record); err != nil {
		_ = h.App.Storage.Delete(key)
		httpBadRequest(w, fmt.Errorf("could not save file record (a resume PDF may already exist): %w", err))
		return
	}

	http.Redirect(w, r, "/candidates/"+candidate.Slug+"/edit", http.StatusSeeOther)
}

func (h *CandidateHandlers) DeleteFile(w http.ResponseWriter, r *http.Request) {
	candidate, ok := h.loadOwnedCandidate(w, r)
	if !ok {
		return
	}
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}

	fileID, err := strconv.ParseInt(r.PathValue("fileID"), 10, 64)
	if err != nil {
		httpBadRequest(w, err)
		return
	}
	file, err := h.App.Files.GetByID(r.Context(), fileID)
	if err != nil || file.CandidateID != candidate.ID {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	if err := h.App.Files.Delete(r.Context(), fileID); err != nil {
		httpServerError(w, err)
		return
	}
	_ = h.App.Storage.Delete(file.StoredPath)

	http.Redirect(w, r, "/candidates/"+candidate.Slug+"/edit", http.StatusSeeOther)
}

// loadOwnedCandidate loads the candidate by slug and verifies the logged-in
// user owns it, writing an error response and returning ok=false otherwise.
func (h *CandidateHandlers) loadOwnedCandidate(w http.ResponseWriter, r *http.Request) (*models.Candidate, bool) {
	slugVal := r.PathValue("slug")
	candidate, err := h.App.Candidates.GetBySlug(r.Context(), slugVal)
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
	if u.ID != candidate.UserID {
		http.Error(w, "Forbidden: you can only edit your own profile", http.StatusForbidden)
		return nil, false
	}
	return candidate, true
}
