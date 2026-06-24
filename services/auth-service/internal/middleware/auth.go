package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/marketplace/auth-service/internal/token"
)

type ctxKey string

const (
	ctxUserID ctxKey = "userID"
	ctxRole   ctxKey = "role"
)

// Auth verifies the Bearer access token and injects userID + role into the context.
func Auth(tm *token.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				unauthorized(w, "missing or invalid Authorization header")
				return
			}
			claims, err := tm.Verify(parts[1])
			if err != nil {
				unauthorized(w, "invalid or expired token")
				return
			}
			ctx := context.WithValue(r.Context(), ctxUserID, claims.Subject)
			ctx = context.WithValue(ctx, ctxRole, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}

// UserID returns the authenticated user id from the context ("" if absent).
func UserID(ctx context.Context) string {
	v, _ := ctx.Value(ctxUserID).(string)
	return v
}

// Role returns the authenticated user's role from the context ("" if absent).
func Role(ctx context.Context) string {
	v, _ := ctx.Value(ctxRole).(string)
	return v
}
