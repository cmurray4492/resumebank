package web

import (
	"net/http"
	"strconv"

	"resumebank/internal/app"
	"resumebank/internal/repo"
)

type SearchHandlers struct {
	App *app.App
}

func NewSearchHandlers(a *app.App) *SearchHandlers { return &SearchHandlers{App: a} }

const searchPageSize = 20

type searchView struct {
	Query         string
	Type          string
	CandidateHits []repo.CandidateSearchResult
	JobHits       []repo.JobSearchResult
	Total         int
	Page          int
	PrevPage      int
	NextPage      int
	HasNextPage   bool
	HasPrevPage   bool
}

func (h *SearchHandlers) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	searchType := r.URL.Query().Get("type")
	if searchType != "candidates" && searchType != "jobs" {
		searchType = "candidates"
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * searchPageSize

	view := searchView{Query: query, Type: searchType, Page: page}

	if searchType == "jobs" {
		hits, total, err := h.App.Jobs.Search(r.Context(), query, searchPageSize, offset)
		if err != nil {
			httpServerError(w, err)
			return
		}
		view.JobHits = hits
		view.Total = total
	} else {
		hits, total, err := h.App.Candidates.Search(r.Context(), query, searchPageSize, offset)
		if err != nil {
			httpServerError(w, err)
			return
		}
		view.CandidateHits = hits
		view.Total = total
	}
	view.HasPrevPage = page > 1
	view.HasNextPage = offset+searchPageSize < view.Total
	view.PrevPage = page - 1
	view.NextPage = page + 1

	pd := newPageData(h.App, w, r, "Search", "Search candidates and job postings on resumebank.biz.", view)
	h.App.Renderer.Render(w, http.StatusOK, "search.html.tmpl", pd)
}
