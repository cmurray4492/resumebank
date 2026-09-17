package web

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"resumebank/internal/models"
	"resumebank/internal/resumeextract"
	"resumebank/internal/sanitize"
	"resumebank/internal/storage"
)

type extractResumeResponse struct {
	HTML           string `json:"html"`
	ResumePDFSaved bool   `json:"resume_pdf_saved"`
}

// ExtractResume reads a PDF or DOCX resume upload and returns Quill-ready
// HTML built from its text. It's used both anonymously (candidate signup,
// before an account exists) and on the candidate edit page (which passes
// candidate_slug so a PDF can also be saved as the candidate's required
// resume file - see maybeSaveResumePDF). It never touches resume_html /
// resume_text itself: the caller's existing save flow does that, exactly as
// it does for pasted/typed content.
func (h *CandidateHandlers) ExtractResume(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, h.App.Config.MaxUploadBytes)
	if err := r.ParseMultipartForm(h.App.Config.MaxUploadBytes); err != nil {
		writeExtractError(w, http.StatusBadRequest, "That file is too large, or the upload didn't come through correctly.")
		return
	}
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeExtractError(w, http.StatusBadRequest, "Please choose a PDF or DOCX file to upload.")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeExtractError(w, http.StatusBadRequest, "Could not read the uploaded file.")
		return
	}

	format, err := resumeextract.DetectFormat(header.Filename, data)
	if err != nil {
		writeExtractError(w, http.StatusBadRequest, err.Error())
		return
	}

	rawHTML, err := resumeextract.ExtractHTML(format, data)
	if err != nil {
		writeExtractError(w, http.StatusUnprocessableEntity, userFacingExtractError(err))
		return
	}

	resp := extractResumeResponse{HTML: sanitize.SanitizeRichText(rawHTML)}

	if slugVal := strings.TrimSpace(r.FormValue("candidate_slug")); slugVal != "" && format == resumeextract.FormatPDF {
		if u := currentUser(r); u != nil {
			if candidate, err := h.App.Candidates.GetBySlug(r.Context(), slugVal); err == nil && candidate.UserID == u.ID {
				resp.ResumePDFSaved = h.maybeSaveResumePDF(r.Context(), candidate, header, data)
			}
			// Candidate not found, or not owned by this user: skip the
			// file-save side effect silently rather than fail the whole
			// extraction over a forged/stale candidate_slug.
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// maybeSaveResumePDF saves the uploaded PDF as the candidate's resume file,
// but only if they don't already have one - it never overwrites an
// existing resume_pdf.
func (h *CandidateHandlers) maybeSaveResumePDF(ctx context.Context, candidate *models.Candidate, header *multipart.FileHeader, data []byte) bool {
	count, err := h.App.Files.CountByKind(ctx, candidate.ID, models.FileKindResumePDF)
	if err != nil || count > 0 {
		return false
	}

	safeName := storage.SafeFilename(header.Filename)
	key := storage.CandidateFileKey(candidate.ID, safeName)
	if err := h.App.Storage.Save(key, bytes.NewReader(data)); err != nil {
		return false
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/pdf"
	}
	record := &models.CandidateFile{
		CandidateID:      candidate.ID,
		Kind:             models.FileKindResumePDF,
		OriginalFilename: header.Filename,
		StoredPath:       key,
		ContentType:      contentType,
		SizeBytes:        int64(len(data)),
	}
	if _, err := h.App.Files.Create(ctx, record); err != nil {
		// Most likely a concurrent request already created one (the DB's
		// partial unique index enforces at most one resume_pdf per
		// candidate) - clean up the orphaned upload and report "not saved".
		_ = h.App.Storage.Delete(key)
		return false
	}
	return true
}

func writeExtractError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func userFacingExtractError(err error) string {
	switch {
	case errors.Is(err, resumeextract.ErrNoTextFound):
		return "We couldn't find any text in that file - if it's a scanned or image-only PDF, please paste your resume text manually."
	case errors.Is(err, resumeextract.ErrCorruptFile):
		return "That file looks corrupted or isn't a valid PDF/DOCX. Please try re-saving and uploading it again."
	default:
		return "Something went wrong reading that file. Please try again or paste your resume manually."
	}
}
