package http

import (
	"net/http"

	"github.com/tomasswaier/RabbitVocab/internal/domain/user"
	"github.com/tomasswaier/RabbitVocab/internal/http/handler"
	"github.com/tomasswaier/RabbitVocab/internal/http/middleware"
)

type Handlers struct {
	Auth     *handler.AuthHandler
	Language *handler.LanguageHandler
	Word     *handler.WordHandler
}

func NewRouter(h Handlers, users user.Repository) *http.ServeMux {
	mux := http.NewServeMux()

	auth := middleware.APIKeyAuth(users)

	// Public
	mux.HandleFunc("POST /auth/register", h.Auth.Register)

	// Authenticated
	mux.Handle("POST /auth/regenerate-key", auth(http.HandlerFunc(h.Auth.RegenerateKey)))
	mux.Handle("POST /languages", auth(http.HandlerFunc(h.Language.Create)))
	mux.Handle("GET /languages", auth(http.HandlerFunc(h.Language.List)))
	mux.Handle("POST /words", auth(http.HandlerFunc(h.Word.Insert)))
	mux.Handle("GET /words/random", auth(http.HandlerFunc(h.Word.GetRandom)))
	mux.Handle("PATCH /words/{id}/state", auth(http.HandlerFunc(h.Word.UpdateState)))

	return mux
}
