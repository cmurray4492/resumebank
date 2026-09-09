// Package app composes the application's dependencies into a single struct
// used by cmd/server and by tests.
package app

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"resumebank/internal/auth"
	"resumebank/internal/config"
	"resumebank/internal/render"
	"resumebank/internal/repo"
	"resumebank/internal/sitemap"
	"resumebank/internal/storage"
)

type App struct {
	Config     *config.Config
	Pool       *pgxpool.Pool
	Renderer   *render.Renderer
	Auth       *auth.Middleware
	Sessions   *auth.SessionStore
	Storage    storage.Storage
	Users      *repo.UserRepo
	Candidates *repo.CandidateRepo
	Employers  *repo.EmployerRepo
	Jobs       *repo.JobRepo
	Files      *repo.FileRepo
	Votes      *repo.VoteRepo
	Messages   *repo.MessageRepo
	Sitemap    *sitemap.Cache
}

func New(cfg *config.Config, pool *pgxpool.Pool) (*App, error) {
	dev := cfg.Env == "development"

	renderer, err := render.New(dev, "web/templates")
	if err != nil {
		return nil, err
	}

	users := repo.NewUserRepo(pool)
	sessions := auth.NewSessionStore(pool)

	a := &App{
		Config:     cfg,
		Pool:       pool,
		Renderer:   renderer,
		Sessions:   sessions,
		Auth:       auth.NewMiddleware(sessions, users, cfg.CookieSecure),
		Storage:    storage.NewLocalStorage(cfg.UploadDir),
		Users:      users,
		Candidates: repo.NewCandidateRepo(pool),
		Employers:  repo.NewEmployerRepo(pool),
		Jobs:       repo.NewJobRepo(pool),
		Files:      repo.NewFileRepo(pool),
		Votes:      repo.NewVoteRepo(pool),
		Messages:   repo.NewMessageRepo(pool),
		Sitemap:    sitemap.NewCache(),
	}
	return a, nil
}

func (a *App) IsDev() bool {
	return a.Config.Env == "development"
}

func (a *App) StaticFileServer() http.Handler {
	return http.FileServer(render.StaticFileSystem(a.IsDev(), "web/static"))
}

// RefreshSitemap rebuilds sitemap.xml from the current candidates,
// employers, and jobs and swaps it into the in-memory cache served by
// GET /sitemap.xml.
func (a *App) RefreshSitemap(ctx context.Context) error {
	urls := []sitemap.URL{
		{Loc: a.Config.BaseURL + "/"},
		{Loc: a.Config.BaseURL + "/search"},
	}

	candidates, err := a.Candidates.ListSlugs(ctx)
	if err != nil {
		return err
	}
	for _, c := range candidates {
		urls = append(urls, sitemap.URL{Loc: a.Config.BaseURL + "/candidates/" + c.Slug, LastMod: c.UpdatedAt})
	}

	employers, err := a.Employers.ListSlugs(ctx)
	if err != nil {
		return err
	}
	for _, e := range employers {
		urls = append(urls, sitemap.URL{Loc: a.Config.BaseURL + "/employers/" + e.Slug, LastMod: e.UpdatedAt})
	}

	jobs, err := a.Jobs.ListSlugs(ctx)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		urls = append(urls, sitemap.URL{Loc: a.Config.BaseURL + "/jobs/" + j.Slug, LastMod: j.UpdatedAt})
	}

	a.Sitemap.Set(sitemap.BuildXML(urls))
	return nil
}

// RefreshSearchIndex rebuilds the unified candidates+jobs search_index
// materialized view. This is the literal "search index regenerated hourly"
// artifact from the spec; live search pages query the base tables directly
// instead (see internal/db/migrations/0010_search_index.sql).
func (a *App) RefreshSearchIndex(ctx context.Context) error {
	_, err := a.Pool.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY search_index`)
	return err
}
