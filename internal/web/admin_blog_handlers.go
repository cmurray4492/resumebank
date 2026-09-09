package web

import (
	"net/http"
	"strconv"
	"strings"

	"resumebank/internal/app"
	"resumebank/internal/httpx"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/sanitize"
	"resumebank/internal/slug"
	"resumebank/internal/validate"
)

type AdminBlogHandlers struct {
	App *app.App
}

func NewAdminBlogHandlers(a *app.App) *AdminBlogHandlers { return &AdminBlogHandlers{App: a} }

const adminBlogPageSize = 20

type adminBlogListView struct {
	Posts       []models.BlogPost
	Page        int
	PrevPage    int
	NextPage    int
	HasPrevPage bool
	HasNextPage bool
}

func (h *AdminBlogHandlers) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * adminBlogPageSize

	posts, total, err := h.App.Blog.ListAll(r.Context(), adminBlogPageSize, offset)
	if err != nil {
		httpServerError(w, err)
		return
	}

	view := adminBlogListView{
		Posts: posts, Page: page,
		PrevPage: page - 1, NextPage: page + 1,
		HasPrevPage: page > 1, HasNextPage: offset+adminBlogPageSize < total,
	}
	pd := newAdminPageData(h.App, w, r, "Blog Posts", view)
	h.App.Renderer.RenderAdmin(w, http.StatusOK, "admin_blog_list.html.tmpl", pd)
}

type adminBlogFormView struct {
	Post *models.BlogPost // nil when creating a new post
}

func (h *AdminBlogHandlers) NewForm(w http.ResponseWriter, r *http.Request) {
	pd := newAdminPageData(h.App, w, r, "New Blog Post", adminBlogFormView{})
	h.App.Renderer.RenderAdmin(w, http.StatusOK, "admin_blog_form.html.tmpl", pd)
}

func (h *AdminBlogHandlers) Create(w http.ResponseWriter, r *http.Request) {
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		httpBadRequest(w, err)
		return
	}

	title := strings.TrimSpace(r.FormValue("title"))
	bodyHTML := sanitize.SanitizeRichText(r.FormValue("body_html"))
	authorName := strings.TrimSpace(r.FormValue("author_name"))
	published := r.FormValue("published") == "on"

	errs := validate.FieldErrors{}
	validate.Required(title, "title", errs)
	validate.Required(bodyHTML, "body_html", errs)

	if errs.HasErrors() {
		pd := newAdminPageData(h.App, w, r, "New Blog Post", adminBlogFormView{})
		pd.Errors = errs
		h.App.Renderer.RenderAdmin(w, http.StatusUnprocessableEntity, "admin_blog_form.html.tmpl", pd)
		return
	}

	baseSlug := slug.Make(title)
	uniqueSlugVal, err := uniqueSlug(r.Context(), baseSlug, h.App.Blog.SlugExists)
	if err != nil {
		httpServerError(w, err)
		return
	}

	post := &models.BlogPost{
		Slug: uniqueSlugVal, Title: title, BodyHTML: bodyHTML,
		BodyText: sanitize.PlainText(bodyHTML), AuthorName: authorName, Published: published,
	}
	if _, err := h.App.Blog.Create(r.Context(), post); err != nil {
		httpServerError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/blog", http.StatusSeeOther)
}

func (h *AdminBlogHandlers) EditForm(w http.ResponseWriter, r *http.Request) {
	post, ok := h.loadPost(w, r)
	if !ok {
		return
	}
	pd := newAdminPageData(h.App, w, r, "Edit Blog Post", adminBlogFormView{Post: post})
	h.App.Renderer.RenderAdmin(w, http.StatusOK, "admin_blog_form.html.tmpl", pd)
}

func (h *AdminBlogHandlers) Update(w http.ResponseWriter, r *http.Request) {
	post, ok := h.loadPost(w, r)
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

	title := strings.TrimSpace(r.FormValue("title"))
	bodyHTML := sanitize.SanitizeRichText(r.FormValue("body_html"))
	authorName := strings.TrimSpace(r.FormValue("author_name"))
	published := r.FormValue("published") == "on"

	errs := validate.FieldErrors{}
	validate.Required(title, "title", errs)
	validate.Required(bodyHTML, "body_html", errs)

	if errs.HasErrors() {
		pd := newAdminPageData(h.App, w, r, "Edit Blog Post", adminBlogFormView{Post: post})
		pd.Errors = errs
		h.App.Renderer.RenderAdmin(w, http.StatusUnprocessableEntity, "admin_blog_form.html.tmpl", pd)
		return
	}

	wasPublished := post.Published
	post.Title = title
	post.BodyHTML = bodyHTML
	post.BodyText = sanitize.PlainText(bodyHTML)
	post.AuthorName = authorName
	post.Published = published

	if _, err := h.App.Blog.Update(r.Context(), post, wasPublished); err != nil {
		httpServerError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/blog", http.StatusSeeOther)
}

func (h *AdminBlogHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	post, ok := h.loadPost(w, r)
	if !ok {
		return
	}
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}
	if err := h.App.Blog.Delete(r.Context(), post.ID); err != nil {
		httpServerError(w, err)
		return
	}
	http.Redirect(w, r, "/admin/blog", http.StatusSeeOther)
}

func (h *AdminBlogHandlers) loadPost(w http.ResponseWriter, r *http.Request) (*models.BlogPost, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		httpBadRequest(w, err)
		return nil, false
	}
	post, err := h.App.Blog.GetByID(r.Context(), id)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
		} else {
			httpServerError(w, err)
		}
		return nil, false
	}
	return post, true
}
