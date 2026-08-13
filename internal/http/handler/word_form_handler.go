package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/tomasswaier/RabbitVocab/internal/domain/wordform"
	"github.com/tomasswaier/RabbitVocab/internal/http/middleware"
)

type WordFormHandler struct {
	wordForms *wordform.Service
}

func NewWordFormHandler(wordForms *wordform.Service) *WordFormHandler {
	return &WordFormHandler{wordForms: wordForms}
}

type insertWordFormRequest struct {
	WordID  int64   `json:"wordId"`
	Subject string  `json:"subject"`
	Form    string  `json:"form"`
	Tense   *string `json:"tense,omitempty"`
}

func (h *WordFormHandler) Insert(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.UserIDFromContext(r.Context()); !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req insertWordFormRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.WordID == 0 || req.Subject == "" || req.Form == "" {
		http.Error(w, "wordId, subject, and form are required", http.StatusBadRequest)
		return
	}

	wf, err := h.wordForms.InsertWordForm(r.Context(), req.WordID, req.Subject, req.Form, req.Tense)

	if err != nil {
		http.Error(w, "failed to create word form", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, wf)
}

func (h *WordFormHandler) GetRandom(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	count, err := strconv.Atoi(r.URL.Query().Get("count"))
	if err != nil || count <= 0 {
		http.Error(w, "count query param must be a positive integer", http.StatusBadRequest)
		return
	}

	var languageID *int64
	if v := r.URL.Query().Get("languageId"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			http.Error(w, "languageId must be an integer", http.StatusBadRequest)
			return
		}
		languageID = &id
	}

	forms, err := h.wordForms.GetRandomWordForms(r.Context(), userID, languageID, int32(count))
	if err != nil {
		handleWordFormServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, forms)
}
func (h *WordFormHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid word form id", http.StatusBadRequest)
		return
	}
	if err := h.wordForms.DeleteWordForm(r.Context(), userID, id); err != nil {
		if errors.Is(err, wordform.ErrNotFoundOrForbidden) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func handleWordFormServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, wordform.ErrLanguageIDRequired):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, wordform.ErrNoLanguages):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
func (h *WordFormHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	page, pageSize := parsePagination(r)

	var languageID *int64
	if v := r.URL.Query().Get("languageId"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			http.Error(w, "languageId must be an integer", http.StatusBadRequest)
			return
		}
		languageID = &id
	}

	result, err := h.wordForms.ListWordForms(r.Context(), userID, languageID, page, pageSize)
	if err != nil {
		handleWordFormServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
