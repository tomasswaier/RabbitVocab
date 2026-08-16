package middleware

import (
	"context"
	"net/http"

	"github.com/tomasswaier/RabbitVocab/internal/apikey"
	domainapikey "github.com/tomasswaier/RabbitVocab/internal/domain/apikey"
	"github.com/tomasswaier/RabbitVocab/internal/domain/session"
)

type contextKey string

const userIDKey contextKey = "userID"

const SessionCookieName = "session_token"

func UserIDFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}

// Authenticate accepts either an X-API-Key/Bearer token (matched against the
// api_keys table, covering both manually-issued keys and OAuth access
// tokens) or a session cookie. Both resolve to the same userID context value.
func Authenticate(apiKeys domainapikey.Repository, sessions *session.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rawKey := r.Header.Get("X-API-Key")
			if rawKey == "" {
				if auth := r.Header.Get("Authorization"); len(auth) > 7 && auth[:7] == "Bearer " {
					rawKey = auth[7:]
				}
			}

			if rawKey != "" {
				u, err := apiKeys.GetUserByHash(r.Context(), apikey.Hash(rawKey))
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
