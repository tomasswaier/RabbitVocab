package http

import (
	"net/http"

	domainapikey "github.com/tomasswaier/RabbitVocab/internal/domain/apikey"
	"github.com/tomasswaier/RabbitVocab/internal/domain/session"
	"github.com/tomasswaier/RabbitVocab/internal/http/handler"
	"github.com/tomasswaier/RabbitVocab/internal/http/middleware"
)

type Handlers struct {
	Auth     *handler.AuthHandler
	Language *handler.LanguageHandler
	Word     *handler.WordHandler
	WordForm *handler.WordFormHandler
	OAuth    *handler.OAuthHandler
}

func RegisterRoutes(mux *http.ServeMux, h Handlers, apiKeys domainapikey.Repository, sessions *session.Service) {
	auth := middleware.Authenticate(apiKeys, sessions)
	mux.HandleFunc("POST /auth/register", h.Auth.Register)
	mux.HandleFunc("POST /auth/login", h.Auth.Login)
	mux.Handle("POST /auth/logout", auth(http.HandlerFunc(h.Auth.Logout)))
	mux.HandleFunc("GET /.well-known/oauth-authorization-server", h.OAuth.Metadata)
	mux.HandleFunc("POST /oauth/register", h.OAuth.Register)
	mux.HandleFunc("GET /oauth/authorize", h.OAuth.AuthorizeForm)
	mux.HandleFunc("POST /oauth/authorize", h.OAuth.AuthorizeSubmit)
	mux.HandleFunc("POST /oauth/token", h.OAuth.Token)

	mux.Handle("POST /auth/api-keys", auth(http.HandlerFunc(h.Auth.CreateAPIKey)))
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
