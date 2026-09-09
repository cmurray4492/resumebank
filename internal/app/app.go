// Package app composes the application's dependencies into a single struct
// used by cmd/server and by tests.
package app

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"resumebank/internal/auth"
	"resumebank/internal/config"
	"resumebank/internal/render"
	"resumebank/internal/repo"
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
	}
	return a, nil
}

func (a *App) IsDev() bool {
	return a.Config.Env == "development"
}

func (a *App) StaticFileServer() http.Handler {
	return http.FileServer(render.StaticFileSystem(a.IsDev(), "web/static"))
}
