package web

import (
	"net/http"

	"resumebank/internal/app"
	"resumebank/internal/httpx"
	"resumebank/internal/models"
)

// NewRouter builds the full route table and wraps it with shared middleware.
func NewRouter(a *app.App) http.Handler {
	mux := http.NewServeMux()

	home := NewHomeHandlers(a)
	authH := NewAuthHandlers(a)
	candidateH := NewCandidateHandlers(a)
	employerH := NewEmployerHandlers(a)
	jobH := NewJobHandlers(a)
	searchH := NewSearchHandlers(a)
	fileH := NewFileHandlers(a)
	messageH := NewMessageHandlers(a)
	sitemapH := NewSitemapHandlers(a)
	matchH := NewMatchHandlers(a)

	mux.HandleFunc("GET /{$}", home.Show)
	mux.HandleFunc("GET /healthz", home.Healthz)
	mux.HandleFunc("GET /me", home.Me)

	mux.HandleFunc("GET /signup/candidate", authH.SignupCandidateForm)
	mux.HandleFunc("POST /signup/candidate", authH.SignupCandidate)
	mux.HandleFunc("GET /signup/employer", authH.SignupEmployerForm)
	mux.HandleFunc("POST /signup/employer", authH.SignupEmployer)
	mux.HandleFunc("GET /login", authH.LoginForm)
	mux.HandleFunc("POST /login", authH.Login)
	mux.HandleFunc("POST /logout", authH.Logout)

	mux.HandleFunc("GET /candidates/{slug}", candidateH.Show)
	mux.HandleFunc("GET /candidates/{slug}/edit", a.Auth.RequireRole(models.RoleCandidate, candidateH.EditForm))
	mux.HandleFunc("POST /candidates/{slug}/edit", a.Auth.RequireRole(models.RoleCandidate, candidateH.Update))
	mux.HandleFunc("POST /candidates/{slug}/files", a.Auth.RequireRole(models.RoleCandidate, candidateH.UploadFile))
	mux.HandleFunc("POST /candidates/{slug}/files/{fileID}/delete", a.Auth.RequireRole(models.RoleCandidate, candidateH.DeleteFile))

	mux.HandleFunc("GET /files/{fileID}/download", fileH.Download)

	mux.HandleFunc("GET /employers/{slug}", employerH.Show)
	mux.HandleFunc("GET /employers/{slug}/edit", a.Auth.RequireRole(models.RoleEmployer, employerH.EditForm))
	mux.HandleFunc("POST /employers/{slug}/edit", a.Auth.RequireRole(models.RoleEmployer, employerH.Update))
	mux.HandleFunc("GET /employers/{slug}/jobs/new", a.Auth.RequireRole(models.RoleEmployer, jobH.NewForm))
	mux.HandleFunc("POST /employers/{slug}/jobs", a.Auth.RequireRole(models.RoleEmployer, jobH.Create))

	mux.HandleFunc("GET /jobs/{slug}", jobH.Show)
	mux.HandleFunc("GET /jobs/{slug}/edit", a.Auth.RequireRole(models.RoleEmployer, jobH.EditForm))
	mux.HandleFunc("POST /jobs/{slug}/edit", a.Auth.RequireRole(models.RoleEmployer, jobH.Update))
	mux.HandleFunc("POST /jobs/{slug}/delete", a.Auth.RequireRole(models.RoleEmployer, jobH.Delete))
	mux.HandleFunc("POST /jobs/{slug}/vote", a.Auth.RequireRole(models.RoleCandidate, jobH.Vote))

	mux.HandleFunc("GET /search", searchH.Search)

	mux.HandleFunc("GET /messages", a.Auth.RequireAuth(messageH.Inbox))
	mux.HandleFunc("GET /messages/{userID}", a.Auth.RequireAuth(messageH.Thread))
	mux.HandleFunc("POST /messages/{userID}", a.Auth.RequireAuth(messageH.Send))

	mux.Handle("GET /static/", http.StripPrefix("/static/", a.StaticFileServer()))

	mux.HandleFunc("GET /sitemap.xml", sitemapH.Sitemap)
	mux.HandleFunc("GET /robots.txt", sitemapH.Robots)

	mux.HandleFunc("GET /match/candidates", a.Auth.RequireRole(models.RoleEmployer, matchH.CandidateMatchForm))
	mux.HandleFunc("POST /match/candidates", a.Auth.RequireRole(models.RoleEmployer, matchH.MatchCandidates))
	mux.HandleFunc("GET /match/jobs", a.Auth.RequireRole(models.RoleCandidate, matchH.JobMatchForm))
	mux.HandleFunc("POST /match/jobs", a.Auth.RequireRole(models.RoleCandidate, matchH.MatchJobs))

	var handler http.Handler = mux
	handler = a.Auth.LoadSession(handler)
	handler = httpx.SecurityHeaders(handler)
	handler = httpx.Recover(handler)
	handler = httpx.Logging(handler)
	return handler
}
