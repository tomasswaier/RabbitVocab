package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/tomasswaier/RabbitVocab/internal/domain/word"
	"github.com/tomasswaier/RabbitVocab/internal/http/middleware"
)

type WordHandler struct {
	words *word.Service
}

func NewWordHandler(words *word.Service) *WordHandler {
	return &WordHandler{words: words}
}

type insertWordRequest struct {
	LanguageID   *int64  `json:"languageId,omitempty"`
	NativeWord   string  `json:"nativeWord"`
	LearningWord string  `json:"learningWord"`
	Article      *string `json:"article,omitempty"`
}

func (h *WordHandler) Insert(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req insertWordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.NativeWord == "" || req.LearningWord == "" {
		http.Error(w, "nativeWord and learningWord are required", http.StatusBadRequest)
		return
	}

	wordEntity, err := h.words.InsertWord(r.Context(), userID, req.LanguageID, req.NativeWord, req.LearningWord, req.Article)
	if err != nil {
		handleWordServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, wordEntity)
}

func (h *WordHandler) GetRandom(w http.ResponseWriter, r *http.Request) {
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

	words, err := h.words.GetRandomWords(r.Context(), userID, languageID, int32(count))
	if err != nil {
		handleWordServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, words)
}

type updateWordStateRequest struct {
	State string `json:"state"`
}

func (h *WordHandler) UpdateState(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid word id", http.StatusBadRequest)
		return
	}

	var req updateWordStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.State == "" {
		http.Error(w, "state is required", http.StatusBadRequest)
		return
	}

	wordEntity, err := h.words.UpdateWordState(r.Context(), id, word.State(req.State))
	if err != nil {
		http.Error(w, "failed to update word state", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, wordEntity)
}
func (h *WordHandler) Search(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	query := r.URL.Query().Get("query")
	if query == "" {
		http.Error(w, "query param is required", http.StatusBadRequest)
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

	words, err := h.words.SearchWords(r.Context(), userID, languageID, query)
	if err != nil {
		handleWordServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, words)
}

func handleWordServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, word.ErrLanguageIDRequired):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, word.ErrNoLanguages):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
func (h *WordHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid word id", http.StatusBadRequest)
		return
	}
	if err := h.words.DeleteWord(r.Context(), userID, id); err != nil {
		if errors.Is(err, word.ErrNotFoundOrForbidden) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
