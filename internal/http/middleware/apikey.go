package middleware

import (
	"context"
	"net/http"

	"github.com/tomasswaier/RabbitVocab/internal/apikey"
	"github.com/tomasswaier/RabbitVocab/internal/domain/session"
	"github.com/tomasswaier/RabbitVocab/internal/domain/user"
)

type contextKey string

const userIDKey contextKey = "userID"

const SessionCookieName = "session_token"

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}

// Authenticate accepts either an X-API-Key header or a session cookie,
// resolving both to the same userID context value. API keys and sessions
// are fully independent — logging in never touches the API key.
func Authenticate(users user.Repository, sessions *session.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rawKey := r.Header.Get("X-API-Key"); rawKey != "" {
				u, err := users.GetByAPIKeyHash(r.Context(), apikey.Hash(rawKey))
				if err == nil {
					ctx := context.WithValue(r.Context(), userIDKey, u.ID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			if cookie, err := r.Cookie(SessionCookieName); err == nil {
				userID, err := sessions.Resolve(r.Context(), cookie.Value)
				if err == nil {
					ctx := context.WithValue(r.Context(), userIDKey, userID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			http.Error(w, "unauthorized", http.StatusUnauthorized)
		})
	}
}
