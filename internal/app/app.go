// Package app composes the application's dependencies into a single struct
// used by cmd/server and by tests.
package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"resumebank/internal/auth"
	"resumebank/internal/config"
	"resumebank/internal/embeddings"
	"resumebank/internal/mailer"
	"resumebank/internal/render"
	"resumebank/internal/repo"
	"resumebank/internal/sitemap"
	"resumebank/internal/storage"
)

type App struct {
	Config         *config.Config
	Pool           *pgxpool.Pool
	Renderer       *render.Renderer
	Auth           *auth.Middleware
	Sessions       *auth.SessionStore
	PasswordResets *auth.PasswordResetStore
	Mailer         mailer.Mailer
	Storage        storage.Storage
	Users          *repo.UserRepo
	Candidates     *repo.CandidateRepo
	Employers      *repo.EmployerRepo
	Jobs           *repo.JobRepo
	Files          *repo.FileRepo
	Votes          *repo.VoteRepo
	Messages       *repo.MessageRepo
	Blog           *repo.BlogRepo
	Sitemap        *sitemap.Cache
	Embeddings     *embeddings.Client
}

func New(cfg *config.Config, pool *pgxpool.Pool) (*App, error) {
	dev := cfg.Env == "development"

	renderer, err := render.New(dev, "web/templates")
	if err != nil {
		return nil, err
	}

	users := repo.NewUserRepo(pool)
	sessions := auth.NewSessionStore(pool)

	var mail mailer.Mailer
	if cfg.SMTPHost != "" {
		mail = &mailer.SMTPMailer{
			Host: cfg.SMTPHost, Port: cfg.SMTPPort,
			Username: cfg.SMTPUsername, Password: cfg.SMTPPassword, From: cfg.SMTPFrom,
		}
	} else {
		mail = mailer.LogMailer{}
	}

	a := &App{
		Config:         cfg,
		Pool:           pool,
		Renderer:       renderer,
		Sessions:       sessions,
		PasswordResets: auth.NewPasswordResetStore(pool),
		Mailer:         mail,
		Auth:           auth.NewMiddleware(sessions, users, cfg.CookieSecure),
		Storage:        storage.NewLocalStorage(cfg.UploadDir),
		Users:          users,
		Candidates:     repo.NewCandidateRepo(pool),
		Employers:      repo.NewEmployerRepo(pool),
		Jobs:           repo.NewJobRepo(pool),
		Files:          repo.NewFileRepo(pool),
		Votes:          repo.NewVoteRepo(pool),
		Messages:       repo.NewMessageRepo(pool),
		Blog:           repo.NewBlogRepo(pool),
		Sitemap:        sitemap.NewCache(),
		Embeddings:     embeddings.New(cfg.OllamaURL, cfg.EmbedModel),
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
		{Loc: a.Config.BaseURL + "/about"},
		{Loc: a.Config.BaseURL + "/terms"},
		{Loc: a.Config.BaseURL + "/privacy"},
		{Loc: a.Config.BaseURL + "/blog"},
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

	posts, err := a.Blog.ListPublishedSlugs(ctx)
	if err != nil {
		return err
	}
	for _, p := range posts {
		urls = append(urls, sitemap.URL{Loc: a.Config.BaseURL + "/blog/" + p.Slug, LastMod: p.UpdatedAt})
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

// UpdateCandidateEmbedding computes and stores a candidate's resume
// embedding. Call it in its own goroutine with a fresh context (not the
// request's, which is cancelled once the response is sent) after a
// candidate is created or their resume changes; a failure here (most
// likely Ollama not running) is logged and left for the next
// RefreshMissingEmbeddings sweep to retry, never surfaced to the user —
// matching/RAG is an explicitly "in development" supplementary feature,
// not something that should ever block signup or profile edits.
func (a *App) UpdateCandidateEmbedding(ctx context.Context, candidateID int64, resumeText string) {
	vector, err := a.Embeddings.EmbedDocument(ctx, resumeText)
	if err != nil {
		log.Printf("embedding candidate %d failed (will retry): %v", candidateID, err)
		return
	}
	if err := a.Candidates.SetEmbedding(ctx, candidateID, embeddings.FormatVector(vector)); err != nil {
		log.Printf("storing embedding for candidate %d failed: %v", candidateID, err)
	}
}

// UpdateJobEmbedding is UpdateCandidateEmbedding's counterpart for a job's
// description embedding.
func (a *App) UpdateJobEmbedding(ctx context.Context, jobID int64, descriptionText string) {
	vector, err := a.Embeddings.EmbedDocument(ctx, descriptionText)
	if err != nil {
		log.Printf("embedding job %d failed (will retry): %v", jobID, err)
		return
	}
	if err := a.Jobs.SetEmbedding(ctx, jobID, embeddings.FormatVector(vector)); err != nil {
		log.Printf("storing embedding for job %d failed: %v", jobID, err)
	}
}

// TriggerCandidateEmbedding kicks off UpdateCandidateEmbedding in the
// background with its own timeout, so callers (HTTP handlers) can return a
// response immediately without waiting on Ollama.
func (a *App) TriggerCandidateEmbedding(candidateID int64, resumeText string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		a.UpdateCandidateEmbedding(ctx, candidateID, resumeText)
	}()
}

// TriggerJobEmbedding is TriggerCandidateEmbedding's counterpart for jobs.
func (a *App) TriggerJobEmbedding(jobID int64, descriptionText string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		a.UpdateJobEmbedding(ctx, jobID, descriptionText)
	}()
}

const embeddingBackfillBatchSize = 50

// RefreshMissingEmbeddings computes embeddings for any candidates/jobs that
// don't have one yet (new rows created while Ollama was unreachable, or
// rows that existed before this feature was added). Run periodically by a
// background job; safe to call even when Ollama is down, since each
// candidate is independently best-effort.
func (a *App) RefreshMissingEmbeddings(ctx context.Context) error {
	candidates, err := a.Candidates.MissingEmbeddings(ctx, embeddingBackfillBatchSize)
	if err != nil {
		return err
	}
	for _, c := range candidates {
		a.UpdateCandidateEmbedding(ctx, c.ID, c.Text)
	}

	jobs, err := a.Jobs.MissingEmbeddings(ctx, embeddingBackfillBatchSize)
	if err != nil {
		return err
	}
	for _, j := range jobs {
		a.UpdateJobEmbedding(ctx, j.ID, j.Text)
	}

	return nil
}

// SendPasswordResetEmail issues a reset token for userID and emails a reset
// link to email, in the background with its own timeout, so the
// "forgot password" HTTP response never waits on token creation or SMTP
// (matching the fire-and-forget pattern used for embeddings). Callers
// should show the same generic "if that email is registered..." message
// regardless of whether this ever succeeds, to avoid leaking which emails
// have accounts.
func (a *App) SendPasswordResetEmail(userID int64, email string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		token, err := a.PasswordResets.Create(ctx, userID)
		if err != nil {
			log.Printf("creating password reset token for user %d failed: %v", userID, err)
			return
		}

		link := a.Config.BaseURL + "/reset-password/" + token
		body := "Someone (hopefully you) requested a password reset for your resumebank.biz account.\n\n" +
			"Reset your password: " + link + "\n\n" +
			"This link expires in 1 hour. If you didn't request this, you can safely ignore this email."

		if err := a.Mailer.Send(ctx, email, "Reset your resumebank.biz password", body); err != nil {
			log.Printf("sending password reset email to user %d failed: %v", userID, err)
		}
	}()
}
