package web

import (
	"context"
	"log"
	"net/http"

	"resumebank/internal/app"
	"resumebank/internal/auth"
	"resumebank/internal/models"
	"resumebank/internal/validate"
)

// Deleting the owning user row is what removes a profile: foreign keys
// cascade to the candidate/employer, its jobs, files, votes and messages, and
// to sessions. Stored uploads aren't covered by the DB, so their keys are
// collected first and removed afterwards on a best-effort basis (a failure
// only leaves an orphaned file behind, never a half-deleted account).

func deleteCandidateAccount(ctx context.Context, a *app.App, c *models.Candidate) error {
	files, err := a.Files.ListByCandidate(ctx, c.ID)
	if err != nil {
		return err
	}
	var keys []string
	if c.PhotoPath != "" {
		keys = append(keys, c.PhotoPath)
	}
	for _, f := range files {
		keys = append(keys, f.StoredPath)
	}
	if err := a.Users.Delete(ctx, c.UserID); err != nil {
		return err
	}
	removeStoredFiles(a, keys)
	return nil
}

// deleteEmployerAccount also removes all of the employer's jobs (cascade).
func deleteEmployerAccount(ctx context.Context, a *app.App, e *models.Employer) error {
	if err := a.Users.Delete(ctx, e.UserID); err != nil {
		return err
	}
	if e.LogoPath != "" {
		removeStoredFiles(a, []string{e.LogoPath})
	}
	return nil
}

func removeStoredFiles(a *app.App, keys []string) {
	for _, k := range keys {
		if err := a.Storage.Delete(k); err != nil {
			log.Printf("deleted account but could not remove stored file %q: %v", k, err)
		}
	}
}

// DeleteProfile lets a candidate permanently delete their own profile and
// account. It re-checks their password, since it's irreversible and a
// stolen session shouldn't be enough to destroy an account.
func (h *CandidateHandlers) DeleteProfile(w http.ResponseWriter, r *http.Request) {
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

	user := currentUser(r)
	if !auth.CheckPassword(user.PasswordHash, r.FormValue("password")) {
		view, err := h.editView(r, candidate)
		if err != nil {
			httpServerError(w, err)
			return
		}
		pd := newPageData(h.App, w, r, "Edit Your Profile", "", view)
		pd.Errors = validate.FieldErrors{}
		pd.Errors.Add("delete_password", "Incorrect password.")
		h.App.Renderer.Render(w, http.StatusUnprocessableEntity, "candidate_edit.html.tmpl", pd)
		return
	}

	if err := deleteCandidateAccount(r.Context(), h.App, candidate); err != nil {
		httpServerError(w, err)
		return
	}
	h.App.Auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
