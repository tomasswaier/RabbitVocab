package word

import (
	"context"
	"errors"

	"github.com/tomasswaier/RabbitVocab/internal/domain/language"
)

var (
	ErrLanguageIDRequired = errors.New("languageId is required: user has multiple languages")
	ErrNoLanguages        = errors.New("user has no languages configured")
)

type Service struct {
	words     Repository
	languages language.Repository
}

func NewService(words Repository, languages language.Repository) *Service {
	return &Service{words: words, languages: languages}
}

type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
	Total    int64 `json:"total"`
}

// resolveLanguageID returns languageID if provided, otherwise resolves it
// from the user's languages, requiring exactly one to exist.
func (s *Service) resolveLanguageID(ctx context.Context, userID int64, languageID *int64) (int64, error) {
	if languageID != nil {
		return *languageID, nil
	}

	langs, err := s.languages.ListByUser(ctx, userID)
	if err != nil {
		return 0, err
	}

	switch len(langs) {
	case 0:
		return 0, ErrNoLanguages
	case 1:
		return langs[0].ID, nil
	default:
		return 0, ErrLanguageIDRequired
	}
}

func (s *Service) InsertWord(ctx context.Context, userID int64, languageID *int64, nativeWord, learningWord string, article *string) (*Word, error) {
	resolvedID, err := s.resolveLanguageID(ctx, userID, languageID)
	if err != nil {
		return nil, err
	}
	return s.words.Insert(ctx, resolvedID, nativeWord, learningWord, article)
}

func (s *Service) GetRandomWords(ctx context.Context, userID int64, languageID *int64, count int32) ([]*Word, error) {
	resolvedID, err := s.resolveLanguageID(ctx, userID, languageID)
	if err != nil {
		return nil, err
	}
	return s.words.GetRandom(ctx, resolvedID, count)
}
func (s *Service) SearchWords(ctx context.Context, userID int64, languageID *int64, query string) ([]*Word, error) {
	resolvedID, err := s.resolveLanguageID(ctx, userID, languageID)
	if err != nil {
		return nil, err
	}
	return s.words.Search(ctx, resolvedID, query)
}

func (s *Service) UpdateWordState(ctx context.Context, id int64, state State) (*Word, error) {
	return s.words.UpdateState(ctx, id, state)
}

var ErrNotFoundOrForbidden = errors.New("word not found or not owned by user")

func (s *Service) DeleteWord(ctx context.Context, userID, wordID int64) error {
	deleted, err := s.words.Delete(ctx, wordID, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFoundOrForbidden
	}
	return nil
}

func (s *Service) ListWords(ctx context.Context, userID int64, languageID *int64, page, pageSize int) (*Page[*Word], error) {
	resolvedID, err := s.resolveLanguageID(ctx, userID, languageID)
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * pageSize
	items, err := s.words.List(ctx, resolvedID, int32(pageSize), int32(offset))
	if err != nil {
		return nil, err
	}
	total, err := s.words.Count(ctx, resolvedID)
	if err != nil {
		return nil, err
	}

	return &Page[*Word]{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}
