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
	staticH := NewStaticPageHandlers(a)
	blogH := NewBlogHandlers(a)
	adminAuthH := NewAdminAuthHandlers(a)
	adminDashboardH := NewAdminDashboardHandlers(a)
	adminBlogH := NewAdminBlogHandlers(a)
	adminCandidateH := NewAdminCandidateHandlers(a)
	adminEmployerH := NewAdminEmployerHandlers(a)
	adminJobH := NewAdminJobHandlers(a)
	resetH := NewPasswordResetHandlers(a)

	mux.HandleFunc("GET /{$}", home.Show)
	mux.HandleFunc("GET /healthz", home.Healthz)
	mux.HandleFunc("GET /me", home.Me)
	mux.HandleFunc("GET /about", staticH.About)
	mux.HandleFunc("GET /terms", staticH.Terms)
	mux.HandleFunc("GET /privacy", staticH.Privacy)

	mux.HandleFunc("GET /blog", blogH.Index)
	mux.HandleFunc("GET /blog/{slug}", blogH.Show)

	mux.HandleFunc("GET /signup/candidate", authH.SignupCandidateForm)
	mux.HandleFunc("POST /signup/candidate", authH.SignupCandidate)
	mux.HandleFunc("GET /signup/employer", authH.SignupEmployerForm)
	mux.HandleFunc("POST /signup/employer", authH.SignupEmployer)
	mux.HandleFunc("GET /login", authH.LoginForm)
	mux.HandleFunc("POST /login", authH.Login)
	mux.HandleFunc("POST /logout", authH.Logout)

	// Shared by candidates, employers, and admins - see password_reset_handlers.go.
	mux.HandleFunc("GET /forgot-password", resetH.ForgotPasswordForm)
	mux.HandleFunc("POST /forgot-password", resetH.ForgotPassword)
	mux.HandleFunc("GET /reset-password/{token}", resetH.ResetPasswordForm)
	mux.HandleFunc("POST /reset-password/{token}", resetH.ResetPassword)

	mux.HandleFunc("GET /candidates/{slug}", candidateH.Show)
	mux.HandleFunc("GET /candidates/{slug}/edit", a.Auth.RequireRole(models.RoleCandidate, candidateH.EditForm))
	mux.HandleFunc("POST /candidates/{slug}/edit", a.Auth.RequireRole(models.RoleCandidate, candidateH.Update))
	mux.HandleFunc("GET /candidates/{slug}/photo", candidateH.Photo)
	mux.HandleFunc("POST /candidates/{slug}/photo", a.Auth.RequireRole(models.RoleCandidate, candidateH.UploadPhoto))
	mux.HandleFunc("POST /candidates/{slug}/photo/delete", a.Auth.RequireRole(models.RoleCandidate, candidateH.DeletePhoto))
	mux.HandleFunc("POST /candidates/{slug}/files", a.Auth.RequireRole(models.RoleCandidate, candidateH.UploadFile))
	mux.HandleFunc("POST /candidates/{slug}/files/{fileID}/delete", a.Auth.RequireRole(models.RoleCandidate, candidateH.DeleteFile))

	mux.HandleFunc("GET /files/{fileID}/download", fileH.Download)

	mux.HandleFunc("GET /employers/{slug}", employerH.Show)
	mux.HandleFunc("GET /employers/{slug}/edit", a.Auth.RequireRole(models.RoleEmployer, employerH.EditForm))
	mux.HandleFunc("POST /employers/{slug}/edit", a.Auth.RequireRole(models.RoleEmployer, employerH.Update))
	mux.HandleFunc("GET /employers/{slug}/logo", employerH.Logo)
	mux.HandleFunc("POST /employers/{slug}/logo", a.Auth.RequireRole(models.RoleEmployer, employerH.UploadLogo))
	mux.HandleFunc("POST /employers/{slug}/logo/delete", a.Auth.RequireRole(models.RoleEmployer, employerH.DeleteLogo))
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

	// Admin panel. /admin/login is the only unauthenticated admin route;
	// everything else under /admin is gated by RequireAdmin. There is no
	// public admin signup - see cmd/createadmin.
	mux.HandleFunc("GET /admin/login", adminAuthH.LoginForm)
	mux.HandleFunc("POST /admin/login", adminAuthH.Login)
	mux.HandleFunc("POST /admin/logout", a.Auth.RequireAdmin(adminAuthH.Logout))

	mux.HandleFunc("GET /admin", a.Auth.RequireAdmin(adminDashboardH.Show))

	mux.HandleFunc("GET /admin/blog", a.Auth.RequireAdmin(adminBlogH.List))
	mux.HandleFunc("GET /admin/blog/new", a.Auth.RequireAdmin(adminBlogH.NewForm))
	mux.HandleFunc("POST /admin/blog", a.Auth.RequireAdmin(adminBlogH.Create))
	mux.HandleFunc("GET /admin/blog/{id}/edit", a.Auth.RequireAdmin(adminBlogH.EditForm))
	mux.HandleFunc("POST /admin/blog/{id}/edit", a.Auth.RequireAdmin(adminBlogH.Update))
	mux.HandleFunc("POST /admin/blog/{id}/delete", a.Auth.RequireAdmin(adminBlogH.Delete))

	mux.HandleFunc("GET /admin/candidates", a.Auth.RequireAdmin(adminCandidateH.List))
	mux.HandleFunc("GET /admin/candidates/{id}/edit", a.Auth.RequireAdmin(adminCandidateH.EditForm))
	mux.HandleFunc("POST /admin/candidates/{id}/edit", a.Auth.RequireAdmin(adminCandidateH.Update))

	mux.HandleFunc("GET /admin/employers", a.Auth.RequireAdmin(adminEmployerH.List))
	mux.HandleFunc("GET /admin/employers/{id}/edit", a.Auth.RequireAdmin(adminEmployerH.EditForm))
	mux.HandleFunc("POST /admin/employers/{id}/edit", a.Auth.RequireAdmin(adminEmployerH.Update))

	mux.HandleFunc("GET /admin/jobs", a.Auth.RequireAdmin(adminJobH.List))
	mux.HandleFunc("GET /admin/jobs/{id}/edit", a.Auth.RequireAdmin(adminJobH.EditForm))
	mux.HandleFunc("POST /admin/jobs/{id}/edit", a.Auth.RequireAdmin(adminJobH.Update))

	var handler http.Handler = mux
	handler = a.Auth.LoadAdminSession(handler)
	handler = a.Auth.LoadSession(handler)
	handler = httpx.SecurityHeaders(handler)
	handler = httpx.Recover(handler)
	handler = httpx.Logging(handler)
	return handler
}
