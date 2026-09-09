package web

import (
	"net/http"
	"strconv"
	"time"

	"resumebank/internal/app"
	"resumebank/internal/httpx"
	"resumebank/internal/models"
	"resumebank/internal/repo"
)

type BlogHandlers struct {
	App *app.App
}

func NewBlogHandlers(a *app.App) *BlogHandlers { return &BlogHandlers{App: a} }

const blogPageSize = 10

type blogIndexView struct {
	Posts       []blogListItem
	Page        int
	PrevPage    int
	NextPage    int
	HasPrevPage bool
	HasNextPage bool
}

type blogListItem struct {
	Slug        string
	Title       string
	Excerpt     string
	AuthorName  string
	PublishedAt *time.Time
}

func (h *BlogHandlers) Index(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * blogPageSize

	posts, total, err := h.App.Blog.ListPublished(r.Context(), blogPageSize, offset)
	if err != nil {
		httpServerError(w, err)
		return
	}

	items := make([]blogListItem, 0, len(posts))
	for _, p := range posts {
		items = append(items, blogListItem{
			Slug: p.Slug, Title: p.Title, Excerpt: excerpt(p.BodyText, 200),
			AuthorName: p.AuthorName, PublishedAt: p.PublishedAt,
		})
	}

	view := blogIndexView{
		Posts: items, Page: page,
		PrevPage: page - 1, NextPage: page + 1,
		HasPrevPage: page > 1, HasNextPage: offset+blogPageSize < total,
	}
	pd := newPageData(h.App, w, r, "Blog", "News, tips, and updates from resumebank.biz.", view)
	h.App.Renderer.Render(w, http.StatusOK, "blog_index.html.tmpl", pd)
}

type blogShowView struct {
	Post       *models.BlogPost
	ShareURL   string
	ShareTitle string
}

func (h *BlogHandlers) Show(w http.ResponseWriter, r *http.Request) {
	slugVal := r.PathValue("slug")
	post, err := h.App.Blog.GetPublishedBySlug(r.Context(), slugVal)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
			return
		}
		httpServerError(w, err)
		return
	}

	view := blogShowView{
		Post:       post,
		ShareURL:   h.App.Config.BaseURL + "/blog/" + post.Slug,
		ShareTitle: post.Title,
	}
	desc := excerpt(post.BodyText, 160)
	pd := newPageData(h.App, w, r, post.Title, desc, view)
	h.App.Renderer.Render(w, http.StatusOK, "blog_show.html.tmpl", pd)
}

// excerpt truncates s to at most n runes, breaking on a word boundary and
// appending an ellipsis if it was shortened.
func excerpt(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	cut := runes[:n]
	for i := len(cut) - 1; i >= 0; i-- {
		if cut[i] == ' ' {
			cut = cut[:i]
			break
		}
	}
	return string(cut) + "…"
}
