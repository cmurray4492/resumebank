package web

import (
	"context"
	"net/http"
	"strconv"

	"resumebank/internal/auth"
	"resumebank/internal/httpx"
	"resumebank/internal/models"
	"resumebank/internal/slug"
)

func httpBadRequest(w http.ResponseWriter, err error) {
	httpx.BadRequest(w, err.Error())
}

func httpServerError(w http.ResponseWriter, err error) {
	httpx.ServerError(w, err)
}

func currentUser(r *http.Request) *models.User {
	return auth.UserFromContext(r.Context())
}

// uniqueSlug appends a numeric suffix to base until exists returns false.
func uniqueSlug(ctx context.Context, base string, exists func(context.Context, string) (bool, error)) (string, error) {
	candidate := base
	for i := 2; ; i++ {
		taken, err := exists(ctx, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
		candidate = slug.WithSuffix(base, i)
	}
}

func optionalInt(s string) *int {
	n, err := strconv.Atoi(s)
	if s == "" || err != nil {
		return nil
	}
	return &n
}
