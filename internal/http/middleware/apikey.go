package middleware

import (
	"context"
	"net/http"

	"github.com/tomasswaier/RabbitVocab/internal/apikey"
	"github.com/tomasswaier/RabbitVocab/internal/domain/user"
)

type contextKey string

const userIDKey contextKey = "userID"

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}

// APIKeyAuth resolves the X-API-Key header to a userID and injects it into
// the request context. Requests without a valid key are rejected with 401.
func APIKeyAuth(users user.Repository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawKey := r.Header.Get("X-API-Key")
			if rawKey == "" {
				http.Error(w, "missing X-API-Key header", http.StatusUnauthorized)
				return
			}

			hash := apikey.Hash(rawKey)
			u, err := users.GetByAPIKeyHash(r.Context(), hash)
			if err != nil {
				http.Error(w, "invalid api key", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, u.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
