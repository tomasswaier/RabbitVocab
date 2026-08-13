package http

import (
	"net/http"

	"github.com/tomasswaier/RabbitVocab/internal/domain/session"
	"github.com/tomasswaier/RabbitVocab/internal/domain/user"
	"github.com/tomasswaier/RabbitVocab/internal/http/handler"
	"github.com/tomasswaier/RabbitVocab/internal/http/middleware"
)

type Handlers struct {
	Auth     *handler.AuthHandler
	Language *handler.LanguageHandler
	Word     *handler.WordHandler
	WordForm *handler.WordFormHandler
}

func RegisterRoutes(mux *http.ServeMux, h Handlers, users user.Repository, sessions *session.Service) {
	auth := middleware.Authenticate(users, sessions)

	mux.HandleFunc("POST /auth/register", h.Auth.Register)
	mux.HandleFunc("POST /auth/login", h.Auth.Login)
	mux.Handle("POST /auth/logout", auth(http.HandlerFunc(h.Auth.Logout)))
	mux.Handle("POST /auth/regenerate-key", auth(http.HandlerFunc(h.Auth.RegenerateKey)))
	mux.Handle("POST /languages", auth(http.HandlerFunc(h.Language.Create)))
	mux.Handle("GET /languages", auth(http.HandlerFunc(h.Language.List)))
	mux.Handle("POST /words", auth(http.HandlerFunc(h.Word.Insert)))
	mux.Handle("GET /words/random", auth(http.HandlerFunc(h.Word.GetRandom)))
	mux.Handle("GET /words/search", auth(http.HandlerFunc(h.Word.Search)))
	mux.Handle("PATCH /words/{id}/state", auth(http.HandlerFunc(h.Word.UpdateState)))
	mux.Handle("POST /word-forms", auth(http.HandlerFunc(h.WordForm.Insert)))
	mux.Handle("GET /word-forms/random", auth(http.HandlerFunc(h.WordForm.GetRandom)))
	mux.Handle("DELETE /words/{id}", auth(http.HandlerFunc(h.Word.Delete)))
	mux.Handle("DELETE /word-forms/{id}", auth(http.HandlerFunc(h.WordForm.Delete)))
	mux.Handle("GET /words", auth(http.HandlerFunc(h.Word.List)))
	mux.Handle("GET /word-forms", auth(http.HandlerFunc(h.WordForm.List)))
}
