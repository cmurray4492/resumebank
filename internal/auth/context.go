package auth

import (
	"context"

	"resumebank/internal/models"
)

type contextKey int

const userContextKey contextKey = iota

func ContextWithUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, u)
}

// UserFromContext returns the logged-in user, or nil if the request is
// unauthenticated.
func UserFromContext(ctx context.Context) *models.User {
	u, _ := ctx.Value(userContextKey).(*models.User)
	return u
}
