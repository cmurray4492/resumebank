package web

import (
	"net/http"
	"strings"

	"resumebank/internal/app"
	"resumebank/internal/embeddings"
	"resumebank/internal/matching"
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

// candidateMatchResult pairs a candidate match with the subset of their
// declared Skills found in the job description text, so results show why
// they were suggested and not just a bare similarity score.
type candidateMatchResult struct {
	repo.CandidateMatch
	MatchedSkills []string
}

func explainCandidateMatches(results []repo.CandidateMatch, jobDescription string) []candidateMatchResult {
	explained := make([]candidateMatchResult, len(results))
	for i, r := range results {
		explained[i] = candidateMatchResult{
			CandidateMatch: r,
			MatchedSkills:  matching.OverlappingSkills(r.Skills, jobDescription),
		}
	}
	return explained
}

type candidateMatchView struct {
	JobDescription string
	Location       string
	Searched       bool
	Unavailable    bool
	Results        []candidateMatchResult
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
			view.Results = explainCandidateMatches(results, jobDescription)
		}
	}

	pd := newPageData(h.App, w, r, "Match Candidates", "Paste a job description to find matching candidates.", view)
	pd.Errors = errs
	h.App.Renderer.Render(w, http.StatusOK, "match_candidates.html.tmpl", pd)
}

// jobMatchResult pairs a job match with the subset of the searching
// candidate's declared Skills found in that job's description text, so
// results show why they were suggested and not just a bare similarity
// score.
type jobMatchResult struct {
	repo.JobMatch
	MatchedSkills []string
}

func explainJobMatches(results []repo.JobMatch, candidateSkills string) []jobMatchResult {
	explained := make([]jobMatchResult, len(results))
	for i, r := range results {
		explained[i] = jobMatchResult{
			JobMatch:      r,
			MatchedSkills: matching.OverlappingSkills(candidateSkills, r.DescriptionText),
		}
	}
	return explained
}

type jobMatchView struct {
	Resume      string
	Location    string
	MinSalary   string
	Searched    bool
	Unavailable bool
	Results     []jobMatchResult
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
			var candidateSkills string
			if u := currentUser(r); u != nil {
				if candidate, err := h.App.Candidates.GetByUserID(r.Context(), u.ID); err == nil {
					candidateSkills = candidate.Skills
				}
			}
			view.Results = explainJobMatches(results, candidateSkills)
		}
	}

	pd := newPageData(h.App, w, r, "Match Jobs", "Paste your resume to find matching jobs.", view)
	pd.Errors = errs
	h.App.Renderer.Render(w, http.StatusOK, "match_jobs.html.tmpl", pd)
}
