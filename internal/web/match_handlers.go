package web

import (
	"net/http"
	"strings"

	"resumebank/internal/app"
	"resumebank/internal/embeddings"
	"resumebank/internal/repo"
	"resumebank/internal/validate"
)

// MatchHandlers implements the two "in development" RAG matching features:
// an employer pastes a job description to find candidates, and a candidate
// pastes a resume to find jobs. Both embed the pasted text with a local
// Ollama model and rank the opposite corpus by cosine similarity.
type MatchHandlers struct {
	App *app.App
}

func NewMatchHandlers(a *app.App) *MatchHandlers { return &MatchHandlers{App: a} }

const matchResultLimit = 15
const matchMaxInputLen = 20000

type candidateMatchView struct {
	JobDescription string
	Location       string
	Searched       bool
	Unavailable    bool
	Results        []repo.CandidateMatch
}

func (h *MatchHandlers) CandidateMatchForm(w http.ResponseWriter, r *http.Request) {
	pd := newPageData(h.App, w, r, "Match Candidates", "Paste a job description to find matching candidates.", candidateMatchView{})
	h.App.Renderer.Render(w, http.StatusOK, "match_candidates.html.tmpl", pd)
}

func (h *MatchHandlers) MatchCandidates(w http.ResponseWriter, r *http.Request) {
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		httpBadRequest(w, err)
		return
	}

	jobDescription := strings.TrimSpace(r.FormValue("job_description"))
	location := strings.TrimSpace(r.FormValue("location"))
	errs := validate.FieldErrors{}
	validate.Required(jobDescription, "job_description", errs)
	validate.MaxLen(jobDescription, "job_description", matchMaxInputLen, errs)

	view := candidateMatchView{JobDescription: jobDescription, Location: location, Searched: true}

	if !errs.HasErrors() {
		vector, err := h.App.Embeddings.EmbedQuery(r.Context(), jobDescription)
		if err != nil {
			view.Unavailable = true
		} else {
			results, err := h.App.Candidates.MatchByEmbedding(r.Context(), embeddings.FormatVector(vector), matchResultLimit, location)
			if err != nil {
				httpServerError(w, err)
				return
			}
			view.Results = results
		}
	}

	pd := newPageData(h.App, w, r, "Match Candidates", "Paste a job description to find matching candidates.", view)
	pd.Errors = errs
	h.App.Renderer.Render(w, http.StatusOK, "match_candidates.html.tmpl", pd)
}

type jobMatchView struct {
	Resume      string
	Location    string
	MinSalary   string
	Searched    bool
	Unavailable bool
	Results     []repo.JobMatch
}

func (h *MatchHandlers) JobMatchForm(w http.ResponseWriter, r *http.Request) {
	pd := newPageData(h.App, w, r, "Match Jobs", "Paste your resume to find matching jobs.", jobMatchView{})
	h.App.Renderer.Render(w, http.StatusOK, "match_jobs.html.tmpl", pd)
}

func (h *MatchHandlers) MatchJobs(w http.ResponseWriter, r *http.Request) {
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		httpBadRequest(w, err)
		return
	}

	resume := strings.TrimSpace(r.FormValue("resume"))
	location := strings.TrimSpace(r.FormValue("location"))
	minSalaryStr := strings.TrimSpace(r.FormValue("min_salary"))
	errs := validate.FieldErrors{}
	validate.Required(resume, "resume", errs)
	validate.MaxLen(resume, "resume", matchMaxInputLen, errs)

	view := jobMatchView{Resume: resume, Location: location, MinSalary: minSalaryStr, Searched: true}

	if !errs.HasErrors() {
		vector, err := h.App.Embeddings.EmbedQuery(r.Context(), resume)
		if err != nil {
			view.Unavailable = true
		} else {
			results, err := h.App.Jobs.MatchByEmbedding(r.Context(), embeddings.FormatVector(vector), matchResultLimit, location, optionalInt(minSalaryStr))
			if err != nil {
				httpServerError(w, err)
				return
			}
			view.Results = results
		}
	}

	pd := newPageData(h.App, w, r, "Match Jobs", "Paste your resume to find matching jobs.", view)
	pd.Errors = errs
	h.App.Renderer.Render(w, http.StatusOK, "match_jobs.html.tmpl", pd)
}
