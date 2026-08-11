package handler

import (
	"encoding/json"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"github.com/tomasswaier/RabbitVocab/internal/apikey"
	"github.com/tomasswaier/RabbitVocab/internal/domain/language"
	"github.com/tomasswaier/RabbitVocab/internal/domain/user"
	"github.com/tomasswaier/RabbitVocab/internal/http/middleware"
)

type AuthHandler struct {
	users     user.Repository
	languages language.Repository
}

func NewAuthHandler(users user.Repository, languages language.Repository) *AuthHandler {
	return &AuthHandler{users: users, languages: languages}
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

	rawKey, err := apikey.Generate()
	if err != nil {
		http.Error(w, "failed to generate api key", http.StatusInternalServerError)
		return
	}

	u, err := h.users.Create(r.Context(), req.Username, string(passwordHash), apikey.Hash(rawKey))
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

	writeJSON(w, http.StatusCreated, registerResponse{UserID: u.ID, APIKey: rawKey})
}

type regenerateKeyResponse struct {
	APIKey string `json:"apiKey"`
}

func (h *AuthHandler) RegenerateKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	rawKey, err := apikey.Generate()
	if err != nil {
		http.Error(w, "failed to generate api key", http.StatusInternalServerError)
		return
	}

	if _, err := h.users.UpdateAPIKeyHash(r.Context(), userID, apikey.Hash(rawKey)); err != nil {
		http.Error(w, "failed to update api key", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, regenerateKeyResponse{APIKey: rawKey})
}
