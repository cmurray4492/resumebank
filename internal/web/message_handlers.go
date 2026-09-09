package web

import (
	"net/http"
	"strconv"
	"strings"

	"resumebank/internal/app"
	"resumebank/internal/httpx"
	"resumebank/internal/models"
	"resumebank/internal/repo"
	"resumebank/internal/validate"
)

type MessageHandlers struct {
	App *app.App
}

func NewMessageHandlers(a *app.App) *MessageHandlers { return &MessageHandlers{App: a} }

// conversationView adds a display name/link to a raw Conversation for the
// inbox template, resolved from the other party's candidate/employer profile.
type conversationView struct {
	repo.Conversation
	OtherName string
	OtherLink string
}

func (h *MessageHandlers) profileFor(r *http.Request, userID int64) (name, link string, err error) {
	u, err := h.App.Users.GetUserByID(r.Context(), userID)
	if err != nil {
		return "", "", err
	}
	switch u.Role {
	case models.RoleCandidate:
		c, err := h.App.Candidates.GetByUserID(r.Context(), userID)
		if err != nil {
			return "", "", err
		}
		return c.Name, "/candidates/" + c.Slug, nil
	case models.RoleEmployer:
		e, err := h.App.Employers.GetByUserID(r.Context(), userID)
		if err != nil {
			return "", "", err
		}
		return e.CompanyName, "/employers/" + e.Slug, nil
	default:
		return u.Email, "", nil
	}
}

func (h *MessageHandlers) Inbox(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	conversations, err := h.App.Messages.ListConversations(r.Context(), u.ID)
	if err != nil {
		httpServerError(w, err)
		return
	}

	views := make([]conversationView, 0, len(conversations))
	for _, c := range conversations {
		name, link, err := h.profileFor(r, c.OtherUserID)
		if err != nil {
			httpServerError(w, err)
			return
		}
		views = append(views, conversationView{Conversation: c, OtherName: name, OtherLink: link})
	}

	pd := newPageData(h.App, w, r, "Messages", "", views)
	h.App.Renderer.Render(w, http.StatusOK, "messages_inbox.html.tmpl", pd)
}

type threadView struct {
	OtherUserID int64
	OtherName   string
	OtherLink   string
	Messages    []models.Message
	CurrentUser *models.User
}

func (h *MessageHandlers) Thread(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	otherUserID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil {
		httpBadRequest(w, err)
		return
	}
	if otherUserID == u.ID {
		httpx.BadRequest(w, "cannot message yourself")
		return
	}

	otherUser, err := h.App.Users.GetUserByID(r.Context(), otherUserID)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
			return
		}
		httpServerError(w, err)
		return
	}
	if otherUser.Role == u.Role {
		httpx.BadRequest(w, "messaging is only between a candidate and an employer")
		return
	}

	if err := h.App.Messages.MarkThreadRead(r.Context(), u.ID, otherUserID); err != nil {
		httpServerError(w, err)
		return
	}

	messages, err := h.App.Messages.ListThread(r.Context(), u.ID, otherUserID)
	if err != nil {
		httpServerError(w, err)
		return
	}

	name, link, err := h.profileFor(r, otherUserID)
	if err != nil {
		httpServerError(w, err)
		return
	}

	view := threadView{OtherUserID: otherUserID, OtherName: name, OtherLink: link, Messages: messages, CurrentUser: u}
	pd := newPageData(h.App, w, r, "Conversation with "+name, "", view)
	h.App.Renderer.Render(w, http.StatusOK, "messages_thread.html.tmpl", pd)
}

func (h *MessageHandlers) Send(w http.ResponseWriter, r *http.Request) {
	u := currentUser(r)
	if !h.App.Auth.VerifyCSRF(r) {
		http.Error(w, "Invalid or missing CSRF token", http.StatusForbidden)
		return
	}

	otherUserID, err := strconv.ParseInt(r.PathValue("userID"), 10, 64)
	if err != nil {
		httpBadRequest(w, err)
		return
	}
	if otherUserID == u.ID {
		httpx.BadRequest(w, "cannot message yourself")
		return
	}

	otherUser, err := h.App.Users.GetUserByID(r.Context(), otherUserID)
	if err != nil {
		if err == repo.ErrNotFound {
			httpx.NotFound(w, r)
			return
		}
		httpServerError(w, err)
		return
	}
	if otherUser.Role == u.Role {
		httpx.BadRequest(w, "messaging is only between a candidate and an employer")
		return
	}

	if err := r.ParseForm(); err != nil {
		httpBadRequest(w, err)
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	errs := validate.FieldErrors{}
	validate.Required(body, "body", errs)
	validate.MaxLen(body, "body", 5000, errs)
	if errs.HasErrors() {
		httpx.BadRequest(w, errs["body"])
		return
	}

	if _, err := h.App.Messages.Create(r.Context(), u.ID, otherUserID, body); err != nil {
		httpServerError(w, err)
		return
	}

	http.Redirect(w, r, "/messages/"+strconv.FormatInt(otherUserID, 10), http.StatusSeeOther)
}
