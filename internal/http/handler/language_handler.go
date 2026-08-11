package handler

import (
	"encoding/json"
	"net/http"

	"github.com/tomasswaier/RabbitVocab/internal/domain/language"
	"github.com/tomasswaier/RabbitVocab/internal/http/middleware"
)

type LanguageHandler struct {
	languages language.Repository
}

func NewLanguageHandler(languages language.Repository) *LanguageHandler {
	return &LanguageHandler{languages: languages}
}

type createLanguageRequest struct {
	Name string `json:"name"`
}

func (h *LanguageHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req createLanguageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	lang, err := h.languages.Create(r.Context(), userID, req.Name)
	if err != nil {
		http.Error(w, "failed to create language", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, lang)
}

func (h *LanguageHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	langs, err := h.languages.ListByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to list languages", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, langs)
}
