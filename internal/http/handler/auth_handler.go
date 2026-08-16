package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/tomasswaier/RabbitVocab/internal/apikey"
	domainapikey "github.com/tomasswaier/RabbitVocab/internal/domain/apikey"
	"github.com/tomasswaier/RabbitVocab/internal/domain/language"
	"github.com/tomasswaier/RabbitVocab/internal/domain/session"
	"github.com/tomasswaier/RabbitVocab/internal/domain/user"
	"github.com/tomasswaier/RabbitVocab/internal/http/middleware"
)

type AuthHandler struct {
	users     user.Repository
	languages language.Repository
	apiKeys   domainapikey.Repository
	sessions  *session.Service
}

func NewAuthHandler(users user.Repository, languages language.Repository, apiKeys domainapikey.Repository, sessions *session.Service) *AuthHandler {
	return &AuthHandler{users: users, languages: languages, apiKeys: apiKeys, sessions: sessions}
}

type registerRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	LanguageName string `json:"languageName,omitempty"`
}

type registerResponse struct {
	UserID int64  `json:"userId"`
	APIKey string `json:"apiKey"`
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Password == "" {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	u, err := h.users.Create(r.Context(), req.Username, string(passwordHash))
	if err != nil {
		http.Error(w, "failed to create user (username may already exist)", http.StatusConflict)
		return
	}

	if req.LanguageName != "" {
		if _, err := h.languages.Create(r.Context(), u.ID, req.LanguageName); err != nil {
			http.Error(w, "user created but failed to add language", http.StatusInternalServerError)
			return
		}
	}

	rawKey, err := apikey.Generate()
	if err != nil {
		http.Error(w, "failed to generate api key", http.StatusInternalServerError)
		return
	}

	label := "default"
	if _, err := h.apiKeys.Create(r.Context(), u.ID, apikey.Hash(rawKey), &label, nil, nil); err != nil {
		http.Error(w, "user created but failed to create api key", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, registerResponse{UserID: u.ID, APIKey: rawKey})
}

type createAPIKeyRequest struct {
	Label string `json:"label,omitempty"`
}

type createAPIKeyResponse struct {
	APIKey string `json:"apiKey"`
}

// CreateAPIKey issues a new, additional API key for the authenticated user.
// Unlike the old design, this does NOT invalidate any existing keys.
func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createAPIKeyRequest
	_ = json.NewDecoder(r.Body).Decode(&req) // body is optional

	rawKey, err := apikey.Generate()
	if err != nil {
		http.Error(w, "failed to generate api key", http.StatusInternalServerError)
		return
	}

	var label *string
	if req.Label != "" {
		label = &req.Label
	}

	if _, err := h.apiKeys.Create(r.Context(), userID, apikey.Hash(rawKey), label, nil, nil); err != nil {
		http.Error(w, "failed to create api key", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, createAPIKeyResponse{APIKey: rawKey})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	u, err := h.users.GetByUsername(r.Context(), req.Username)
	if err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, "invalid username or password", http.StatusUnauthorized)
		return
	}

	rawToken, expiresAt, err := h.sessions.Create(r.Context(), u.ID)
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    rawToken,
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(middleware.SessionCookieName); err == nil {
		_ = h.sessions.Delete(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     middleware.SessionCookieName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
